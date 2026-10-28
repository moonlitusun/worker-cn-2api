package trae

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Operation struct {
	ModelCount *int   `json:"model_count,omitempty"`
	At         string `json:"at,omitempty"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type Diagnostics struct {
	Configured         bool      `json:"configured"`
	RefreshAvailable   bool      `json:"refresh_available"`
	ExpiresAt          string    `json:"expires_at,omitempty"`
	ExpiresInSeconds   *int64    `json:"expires_in_seconds,omitempty"`
	PersistencePending bool      `json:"persistence_pending"`
	Active             int       `json:"active_requests"`
	Queued             int64     `json:"queued_requests"`
	LastCatalog        Operation `json:"last_catalog"`
	LastChat           Operation `json:"last_chat"`
	LastRefresh        Operation `json:"last_refresh"`
	LastTest           Operation `json:"last_test"`
}

type observation struct {
	Generation uint64
	Config     Config
	Start      time.Time
}

func (b *Bridge) observe() observation {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return observation{b.generation, b.cfg, time.Now()}
}
func (b *Bridge) record(kind string, o observation, err error, modelCount ...int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.generation != o.Generation {
		return
	}
	op := Operation{At: time.Now().UTC().Format(time.RFC3339), Success: err == nil, DurationMS: time.Since(o.Start).Milliseconds()}
	if len(modelCount) > 0 {
		n := modelCount[0]
		op.ModelCount = &n
	}
	if err != nil {
		op.Error = scrub(scrub(err.Error(), o.Config), b.cfg)
		if len(op.Error) > 2000 {
			op.Error = op.Error[:2000]
		}
	}
	switch kind {
	case "catalog":
		b.diagnostics.LastCatalog = op
	case "chat":
		b.diagnostics.LastChat = op
	case "refresh":
		b.diagnostics.LastRefresh = op
	case "test":
		b.diagnostics.LastTest = op
	}
}

func (b *Bridge) Status() Diagnostics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	s := b.diagnostics
	s.Configured = b.cfg.Token != ""
	s.RefreshAvailable = b.cfg.RefreshToken != "" || strings.HasPrefix(b.cfg.Token, "trae-lt-")
	s.PersistencePending = b.unsavedSession
	s.Active = len(b.slots)
	s.Queued = b.waiters.Load()
	effective := b.cfg
	if b.enterpriseSession != nil && b.enterpriseSource == b.cfg {
		effective = *b.enterpriseSession
	}
	if expiry := tokenExpiry(effective); !expiry.IsZero() {
		s.ExpiresAt = expiry.UTC().Format(time.RFC3339)
		seconds := int64(time.Until(expiry).Seconds())
		s.ExpiresInSeconds = &seconds
	}
	return s
}

type ProbeResult struct {
	ModelCount int     `json:"model_count"`
	Models     []Model `json:"models"`
	Model      string  `json:"model,omitempty"`
	ChatTested bool    `json:"chat_tested"`
	DurationMS int64   `json:"duration_ms"`
}

// Probe always obtains a fresh catalog. Text generation occurs only when a
// concrete model was explicitly selected by the administrator.
func (b *Bridge) Probe(ctx context.Context, model string) (result ProbeResult, err error) {
	o := b.observe()
	defer func() { b.record("test", o, err, result.ModelCount) }()
	models, err := b.models(ctx, true)
	if err != nil {
		return result, err
	}
	result.ModelCount = len(models)
	result.Models = models
	if model != "" {
		found := false
		for _, m := range models {
			if m.ID == model {
				found = true
				break
			}
		}
		if !found {
			return result, bad("test model is not present in the fresh catalog")
		}
		err = b.HandleChat(ctx, &probeWriter{}, map[string]any{"model": "trae/" + model, "messages": []any{map[string]any{"role": "user", "content": "Reply with OK only."}}, "max_tokens": 64})
		if err != nil {
			return result, err
		}
		result.Model = model
		result.ChatTested = true
	}
	result.DurationMS = time.Since(o.Start).Milliseconds()
	b.mu.RLock()
	changed := b.generation != o.Generation
	b.mu.RUnlock()
	if changed {
		return ProbeResult{}, bad("credentials changed during the test; retry")
	}
	return result, nil
}

// Keep counters independent of credential generation so in-flight requests
// remain visible while an administrator updates the session.
type requestCounter = atomic.Int64

type probeWriter struct{ header http.Header }

func (w *probeWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}
func (*probeWriter) WriteHeader(int)             {}
func (*probeWriter) Write(p []byte) (int, error) { return len(p), nil }

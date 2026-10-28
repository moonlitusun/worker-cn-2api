package qoder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	sharedauth "worker-cn-2api/auth"
	"worker-cn-2api/qoder/auth"
	"worker-cn-2api/qoder/bridge"
	"worker-cn-2api/qoder/models"
	"worker-cn-2api/qoder/transform"
	"worker-cn-2api/stats"
)

type Provider struct {
	mu     sync.RWMutex
	pat    string
	bridge *bridge.OpenAiBridge
}

func NewProvider(pat string) *Provider { p := &Provider{}; p.UpdatePAT(pat); return p }
func (p *Provider) Configured() bool   { p.mu.RLock(); defer p.mu.RUnlock(); return p.pat != "" }
func (p *Provider) UpdatePAT(pat string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pat = strings.TrimSpace(pat)
	p.bridge = nil
	if p.pat != "" {
		p.bridge = bridge.NewOpenAiBridge(p.pat, auth.CN)
	}
}
func (p *Provider) snapshot() (*bridge.OpenAiBridge, string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.bridge, p.pat
}
func timeouts() {
	t := sharedauth.CurrentStreamTimeouts()
	old := auth.CurrentStreamTimeouts()
	if t.Header != old.Header || t.Idle != old.Idle {
		auth.SetStreamTimeouts(t.Header, t.Idle)
	}
}

type Error struct {
	Status   int
	Message  string
	Reported bool
}

func (e *Error) Error() string { return e.Message }
func classify(err error, pat string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if pat != "" {
		message = strings.ReplaceAll(message, pat, "[redacted]")
	}
	e := &Error{Status: 502, Message: message}
	var reported *bridge.StreamReportedError
	var queue *bridge.QueueTimeoutError
	var busy *auth.BusyError
	var invalid *models.UnsupportedModelError
	if errors.As(err, &reported) {
		e.Reported = true
	}
	if errors.As(err, &queue) || errors.As(err, &busy) {
		e.Status = 429
	}
	if errors.As(err, &invalid) {
		e.Status = 400
	}
	return e
}
func (p *Provider) Models(ctx context.Context) ([]string, error) {
	timeouts()
	b, pat := p.snapshot()
	if b == nil {
		return nil, &Error{Status: 503, Message: "Qoder CN PAT is not configured"}
	}
	catalog, err := b.VerifiedCatalog(ctx)
	if err != nil {
		return nil, classify(err, pat)
	}
	return catalog.Keys(), nil
}

// TestModels verifies the saved PAT with a new session and uncached catalog.
func (p *Provider) TestModels(ctx context.Context) ([]string, error) {
	_, pat := p.snapshot()
	return NewProvider(pat).Models(ctx)
}
func (p *Provider) HandleChat(ctx context.Context, w http.ResponseWriter, body map[string]any, recorders ...*stats.Recorder) error {
	timeouts()
	b, pat := p.snapshot()
	if b == nil {
		return &Error{Status: 503, Message: "Qoder CN PAT is not configured"}
	}
	copy := make(map[string]any, len(body))
	for k, v := range body {
		copy[k] = v
	}
	name, _ := body["model"].(string)
	name = strings.TrimPrefix(strings.TrimSpace(name), "qoder/")
	if name == "" {
		return &Error{Status: 400, Message: "Qoder model is required"}
	}
	copy["model"] = name
	copy["_gateway_response_model"] = body["model"]
	if _, err := b.VerifiedCatalog(ctx); err != nil {
		return classify(err, pat)
	}
	var sink bridge.UsageSink
	if len(recorders) > 0 && recorders[0] != nil {
		sink = func(u *transform.Usage) {
			recorders[0].RecordUsageFor("qoder", &stats.Usage{PromptTokens: u.PromptTokens, CompletionTokens: u.CompletionTokens, CachedTokens: u.CachedPromptTokens(), Credits: u.Credits})
		}
	}
	return classify(b.HandleChat(ctx, w, copy, sink), pat)
}
func (p *Provider) String() string { return fmt.Sprintf("Qoder CN configured=%t", p.Configured()) }

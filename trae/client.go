package trae

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"worker-cn-2api/auth"
)

type Bridge struct {
	mu              sync.RWMutex
	refreshGate     chan struct{}
	credentialSaver func(Config) error
	unsavedSession  bool
	generation      uint64
	diagnostics     Diagnostics
	waiters         requestCounter
	cfg             Config
	client          *http.Client
	slots           chan struct{}
	catalogGate     chan struct{}
	catalog         []Model
	catalogAt       time.Time
	catalogConfig   Config
}

func NewBridge(cfg Config) *Bridge {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	return &Bridge{cfg: defaults(cfg), slots: make(chan struct{}, 1), refreshGate: make(chan struct{}, 1), catalogGate: make(chan struct{}, 1), client: &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (b *Bridge) Config() Config { b.mu.RLock(); defer b.mu.RUnlock(); return b.cfg }
func (b *Bridge) UpdateCredentials(token, userID string) {
	b.UpdateSession(token, userID, "", "", "")
}
func (b *Bridge) UpdateSession(token, userID, refreshToken, expiresAt, clientID string) {
	_ = b.ReplaceSession(token, userID, refreshToken, expiresAt, clientID, nil)
}

// ReplaceSession serializes admin persistence with refresh persistence. The save
// callback must not call back into the Bridge.
func (b *Bridge) ReplaceSession(token, userID, refreshToken, expiresAt, clientID string, save func() error) error {
	return b.ChangeSession(func(current Config) (Config, error) {
		if save != nil {
			if err := save(); err != nil {
				return current, err
			}
		}
		current.Token = token
		current.UserID = userID
		current.RefreshToken = refreshToken
		current.ExpiresAt = expiresAt
		if clientID != "" {
			current.ClientID = clientID
		}
		return current, nil
	})
}

// ChangeSession keeps account selection, persistence, and refresh serialized.
// The callback receives the latest snapshot and must not reenter the Bridge.
func (b *Bridge) ChangeSession(change func(Config) (Config, error)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.unsavedSession && b.credentialSaver != nil {
		if err := b.credentialSaver(b.cfg); err != nil {
			return fmt.Errorf("Trae session could not be saved")
		}
		b.unsavedSession = false
	}
	next, err := change(b.cfg)
	if err != nil {
		return err
	}
	next.Token = strings.TrimSpace(next.Token)
	next.UserID = strings.TrimSpace(next.UserID)
	next.RefreshToken = strings.TrimSpace(next.RefreshToken)
	if next == b.cfg {
		return nil
	}
	b.cfg = next
	b.unsavedSession = false
	b.generation++
	b.diagnostics = Diagnostics{}
	return nil
}

// Named-account operations wait for refresh completion so a rotated session is
// saved to its original profile before the active profile changes.
func (b *Bridge) ChangeAccount(ctx context.Context, change func(Config) (Config, error)) error {
	select {
	case b.refreshGate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-b.refreshGate }()
	return b.ChangeSession(change)
}

func (b *Bridge) SetCredentialSaver(save func(Config) error) {
	b.mu.Lock()
	b.credentialSaver = save
	b.mu.Unlock()
}

func scrub(s string, c Config) string {
	for _, secret := range []string{c.Token, c.RefreshToken, c.UserID, c.DeviceID, c.MachineID} {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "[redacted]")
		}
	}
	return s
}

type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelBody) Close() error { c.cancel(); return c.ReadCloser.Close() }

func (b *Bridge) request(ctx context.Context, c Config, path string, body any, stream bool) (*http.Response, error) {
	if c.Token == "" {
		return nil, &Error{Status: 503, Message: "auth token is not configured"}
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid Trae base URL")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("invalid Trae request URL")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	req.Header.Set("Authorization", "Cloud-IDE-JWT "+c.Token)
	req.Header.Set("X-Cloudide-Token", c.Token)
	req.Header.Set("X-Ide-Token", c.Token)
	if c.UserID != "" {
		req.Header.Set("X-Uid", c.UserID)
	}
	req.Header.Set("X-App-Id", "6eefa01c-1036-4c7e-9ca5-d891f63bfcd8")
	req.Header.Set("X-App-Version", "default")
	req.Header.Set("X-App-Version-Code", c.VersionCode)
	req.Header.Set("X-Ide-Version-Code", c.VersionCode)
	req.Header.Set("X-Ide-Version", c.Version)
	req.Header.Set("X-Ide-Version-Type", "stable")
	req.Header.Set("User-Agent", "Trae/"+c.Version)
	req.Header.Set("X-Device-Type", runtime.GOOS)
	req.Header.Set("X-OS-Version", runtime.GOOS)
	req.Header.Set("Request-Traffic-Type", "prod")
	req.Header.Set("X-Request-Id", uuid.NewString())
	if c.DeviceID != "" {
		req.Header.Set("X-Device-Id", c.DeviceID)
	}
	if c.MachineID != "" {
		req.Header.Set("X-Machine-Id", c.MachineID)
	}
	timeout := auth.CurrentStreamTimeouts().Header
	timer := time.AfterFunc(timeout, cancel)
	resp, err := b.client.Do(req)
	timer.Stop()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("Trae connection failed: %s", scrub(err.Error(), c))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Error bodies are bounded and use the same timeout as headers.
		timer = time.AfterFunc(timeout, cancel)
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		timer.Stop()
		resp.Body.Close()
		cancel()
		status := resp.StatusCode
		if status < 400 {
			status = 502
		}
		return nil, &Error{Status: status, Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, scrub(string(data), c))}
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
	if stream && !strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		resp.Body.Close()
		return nil, fmt.Errorf("Trae returned a non-SSE response")
	}
	return resp, nil
}

type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextWindow int64  `json:"context_window,omitempty"`
}

func (b *Bridge) Models(ctx context.Context) ([]Model, error) {
	return b.models(ctx, false)
}
func (b *Bridge) models(ctx context.Context, fresh bool) (result []Model, retErr error) {
	o := b.observe()
	defer func() { b.record("catalog", o, retErr) }()
	select {
	case b.catalogGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-b.catalogGate }()
	if fresh {
		b.catalogAt = time.Time{}
		b.catalog = nil
	}
	cfg, err := b.ensureCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Token == "" {
		return nil, &Error{Status: 503, Message: "auth token is not configured"}
	}
	if !fresh && cfg == b.catalogConfig && time.Since(b.catalogAt) < time.Hour {
		return append([]Model{}, b.catalog...), nil
	}
	ctx, cancel := context.WithTimeout(ctx, auth.CurrentStreamTimeouts().Header)
	defer cancel()
	resp, err := b.request(ctx, cfg, "/api/ide/v1/get_detail_param", map[string]any{"function": Function, "config_names": nil, "need_prompt": false, "current_config_info": nil, "poly_prompt": true, "mode_type": nil, "agent_type": nil}, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, (10<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 10<<20 {
		return nil, fmt.Errorf("Trae catalog exceeds 10 MiB")
	}
	var obj map[string]any
	if json.Unmarshal(data, &obj) != nil || obj == nil {
		return nil, fmt.Errorf("invalid Trae catalog JSON")
	}
	if code, ok := obj["code"]; ok && code != float64(0) {
		return nil, &Error{Status: 502, Code: code, Message: scrub(fmt.Sprint(obj["message"]), cfg)}
	}
	if nested, ok := obj["data"].(map[string]any); ok {
		obj = nested
	}
	list, ok := obj["config_info_list"].([]any)
	if !ok {
		return nil, fmt.Errorf("Trae catalog has no config_info_list")
	}
	models := []Model{}
	seen := map[string]bool{}
	for _, raw := range list {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid Trae catalog entry")
		}
		id, _ := m["config_name"].(string)
		if id == "" || seen[id] || m["config_switch"] == false || m["is_invisible_to_user"] == true {
			continue
		}
		display, _ := m["display_config"].(map[string]any)
		name, _ := display["display_name"].(string)
		if name == "" {
			continue
		}
		if f, _ := m["raw_chat_function"].(string); f != "" && f != Function {
			continue
		}
		// These exact configurations are raw-agent-only in the reference lane's
		// captured tests; do not hide other similarly named configs (e.g. -Official).
		switch strings.ToLower(id) {
		case "agnes-2.0-flash", "agnes-agent-x", "deepseek-v4-flash", "deepseek-v4-pro":
			continue
		}
		var window int64
		if v, ok := m["context_window_tokens"].(map[string]any); ok {
			if n, ok := v["dev"].(float64); ok && n > 0 && n < 1e9 {
				window = int64(n)
			}
		}
		models = append(models, Model{ID: id, Name: name, ContextWindow: window})
		seen[id] = true
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("Trae returned no visible models for %s", Function)
	}
	b.catalog = models
	b.catalogConfig = cfg
	b.catalogAt = time.Now()
	return append([]Model{}, models...), nil
}

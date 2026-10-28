package workbuddy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"worker-cn-2api/auth"
)

type StreamCallback func(line string) error

type UpstreamError struct {
	Status  int
	Message string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("WorkBuddy CN HTTP %d: %s", e.Status, e.Message)
}
func readUpstreamError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &UpstreamError{Status: resp.StatusCode, Message: strings.TrimSpace(string(body))}
}

type Client struct {
	mu         sync.RWMutex
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg *Config) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return &Client{cfg: *cfg, httpClient: &http.Client{Transport: transport, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) config() *Config { c.mu.RLock(); defer c.mu.RUnlock(); cfg := c.cfg; return &cfg }
func (c *Client) updateToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg.Token = strings.TrimSpace(token)
}

func setHeaders(req *http.Request, cfg *Config) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Accept", "text/event-stream, application/json")
	req.Header.Set("X-Client-Version", cfg.ClientVer)
	req.Header.Set("X-Client-ID", "workbuddy")
	req.Header.Set("X-Product", "WorkBuddy")
	req.Header.Set("X-Domain", "www.codebuddy.cn")
	req.Header.Set("X-No-Enterprise-Id", "1")
	req.Header.Set("X-Request-ID", uuid.NewString())
	req.Header.Set("User-Agent", fmt.Sprintf("WorkBuddy/%s WorkBuddy AI/%s", cfg.ClientVer, cfg.ClientVer))
	if cfg.UserID != "" {
		req.Header.Set("X-User-Id", cfg.UserID)
	}
	if cfg.DeviceID != "" {
		req.Header.Set("X-Device-Id", cfg.DeviceID)
	}
}

// OpenStream validates upstream status before any downstream response is committed.
func (c *Client) OpenStream(ctx context.Context, req *UpstreamChatRequest) (*http.Response, error) {
	cfg := c.config()
	if cfg.Token == "" {
		return nil, fmt.Errorf("WorkBuddy CN auth token is not configured")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, err
	}
	setHeaders(httpReq, cfg)
	var headerTimedOut atomic.Bool
	timer := time.AfterFunc(auth.CurrentStreamTimeouts().Header, func() { headerTimedOut.Store(true); cancel() })
	resp, err := c.httpClient.Do(httpReq)
	timer.Stop()
	if err != nil {
		cancel()
		if headerTimedOut.Load() {
			return nil, fmt.Errorf("WorkBuddy response header timeout after %s", auth.CurrentStreamTimeouts().Header)
		}
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		defer cancel()
		return nil, readUpstreamError(resp)
	}
	if !strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		defer resp.Body.Close()
		defer cancel()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("WorkBuddy CN expected SSE, received %s: %s", resp.Header.Get("Content-Type"), body)
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

type cancelBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelBody) Close() error { b.cancel(); return b.ReadCloser.Close() }

// ReadStream handles SSE event boundaries, including multiline data fields.
func ReadStream(resp *http.Response, callback StreamCallback) error {
	defer resp.Body.Close()
	var idleTimedOut atomic.Bool
	timer := time.AfterFunc(auth.CurrentStreamTimeouts().Idle, func() { idleTimedOut.Store(true); resp.Body.Close() })
	defer timer.Stop()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	data := []string{}
	dispatch := func() error {
		if len(data) == 0 {
			return nil
		}
		payload := strings.Join(data, "\n")
		data = nil
		return callback("data: " + payload)
	}
	for scanner.Scan() {
		timer.Reset(auth.CurrentStreamTimeouts().Idle)
		line := scanner.Text()
		if line == "" {
			if err := dispatch(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if idleTimedOut.Load() {
		return fmt.Errorf("WorkBuddy stream idle timeout after %s", auth.CurrentStreamTimeouts().Idle)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return dispatch()
}
func (c *Client) PostStream(ctx context.Context, req *UpstreamChatRequest, callback StreamCallback) error {
	resp, err := c.OpenStream(ctx, req)
	if err != nil {
		return err
	}
	return ReadStream(resp, callback)
}

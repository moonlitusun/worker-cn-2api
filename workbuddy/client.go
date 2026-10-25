package workbuddy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StreamCallback is invoked for each SSE line received from upstream.
type StreamCallback func(line string) error

// Client manages upstream HTTP communication with Tencent Copilot.
type Client struct {
	cfg        *Config
	httpClient *http.Client
}

// NewClient creates a new WorkBuddy upstream client.
func NewClient(cfg *Config) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 120 * time.Second
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

// PostStream sends the chat request to upstream and streams lines to callback.
func (c *Client) PostStream(ctx context.Context, req *UpstreamChatRequest, callback StreamCallback) error {
	if c.cfg.Token == "" {
		return fmt.Errorf("WorkBuddy auth token not configured (set WORKER_WORKBUDDY_TOKEN or login to desktop client)")
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	httpReq.Header.Set("X-Device-Id", c.cfg.DeviceID)
	httpReq.Header.Set("X-Client-Version", c.cfg.ClientVer)
	httpReq.Header.Set("X-Platform", c.cfg.Platform)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("User-Agent", fmt.Sprintf("WorkBuddy-Desktop/%s (%s; x64)", c.cfg.ClientVer, c.cfg.Platform))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("upstream returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if err := callback(line); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return scanner.Err()
}

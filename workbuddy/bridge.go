package workbuddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Bridge struct {
	client       *Client
	chatSlots    chan struct{}
	catalogMu    sync.Mutex
	catalog      []ModelInfo
	catalogAt    time.Time
	catalogToken string
}

func NewBridge(cfg *Config) *Bridge {
	n := cfg.Concurrency
	if n <= 0 {
		n = 2
	}
	return &Bridge{client: NewClient(cfg), chatSlots: make(chan struct{}, n)}
}
func (b *Bridge) Config() *Config          { return b.client.config() }
func (b *Bridge) UpdateToken(token string) { b.client.updateToken(token) }

// ReportedError indicates an error already written inside an SSE response.
type ReportedError struct{ Err error }

func (e *ReportedError) Error() string { return e.Err.Error() }
func (e *ReportedError) Unwrap() error { return e.Err }

type QueueTimeoutError struct{}

func (e *QueueTimeoutError) Error() string { return "WorkBuddy CN upstream queue timeout" }

func (b *Bridge) HandleChat(ctx context.Context, w http.ResponseWriter, body map[string]interface{}) error {
	model, _ := body["model"].(string)
	model, err := normalizeModel(model)
	if err != nil {
		return err
	}
	// Exact IDs are passed through; the upstream validates availability. The catalog
	// is discovery metadata and must not block chat when its endpoint is unavailable.
	req, err := BuildUpstreamRequest(body, model, b.Config().DeviceID)
	if err != nil {
		return err
	}
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case b.chatSlots <- struct{}{}:
	case <-timer.C:
		return &QueueTimeoutError{}
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-b.chatSlots }()
	resp, err := b.client.OpenStream(ctx, req)
	if err != nil {
		return err
	}
	stream, _ := body["stream"].(bool)
	if stream {
		return b.handleStream(w, resp)
	}
	return b.handleSync(w, resp, model)
}

var errDone = errors.New("SSE done")

func parseEvent(line string) (map[string]interface{}, bool, error) {
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "[DONE]" {
		return nil, true, nil
	}
	var chunk map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		return nil, false, fmt.Errorf("invalid WorkBuddy SSE JSON: %w", err)
	}
	if chunk == nil {
		return nil, false, fmt.Errorf("empty WorkBuddy SSE event")
	}
	if code, ok := chunk["code"].(float64); ok && code != 0 {
		return nil, false, fmt.Errorf("WorkBuddy CN code %.0f: %v", code, chunk["msg"])
	}
	if raw, ok := chunk["error"]; ok {
		msg, _ := json.Marshal(raw)
		return nil, false, fmt.Errorf("WorkBuddy CN stream error: %s", msg)
	}
	return chunk, false, nil
}

func (b *Bridge) handleStream(w http.ResponseWriter, resp *http.Response) error {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	var mu sync.Mutex
	write := func(s string) error {
		mu.Lock()
		defer mu.Unlock()
		_, err := io.WriteString(w, s)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return err
	}
	if err := write(": worker-cn-2api connected\n\n"); err != nil {
		resp.Body.Close()
		return &ReportedError{err}
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_ = write(": ping\n\n")
			}
		}
	}()
	defer func() { close(stop); <-done }()
	completed := false
	finished := false
	err := ReadStream(resp, func(line string) error {
		chunk, isDone, err := parseEvent(line)
		if err != nil {
			return err
		}
		if isDone {
			if !finished {
				return fmt.Errorf("WorkBuddy CN stream has no terminal choice")
			}
			completed = true
			if err := write("data: [DONE]\n\n"); err != nil {
				return err
			}
			return errDone
		}
		choices, _ := chunk["choices"].([]interface{})
		for _, raw := range choices {
			choice, _ := raw.(map[string]interface{})
			if reason, ok := choice["finish_reason"].(string); ok && reason != "" {
				if reason == "error" {
					return fmt.Errorf("WorkBuddy CN reported finish_reason: error")
				}
				finished = true
			}
		}
		return write(line + "\n\n")
	})
	if errors.Is(err, errDone) {
		return nil
	}
	if err == nil && !completed {
		err = fmt.Errorf("WorkBuddy CN stream ended before [DONE]")
	}
	if err != nil {
		payload, _ := json.Marshal(map[string]interface{}{"error": map[string]interface{}{"type": "upstream_error", "message": err.Error()}})
		_ = write("data: " + string(payload) + "\n\ndata: [DONE]\n\n")
		return &ReportedError{err}
	}
	return nil
}

type aggregatedChoice struct {
	Index        int                    `json:"index"`
	Message      map[string]interface{} `json:"message"`
	FinishReason interface{}            `json:"finish_reason"`
	tools        map[int]map[string]interface{}
}

func (b *Bridge) handleSync(w http.ResponseWriter, resp *http.Response, model string) error {
	result := map[string]interface{}{"id": NewRequestID(), "object": "chat.completion", "created": time.Now().Unix(), "model": model}
	choices := map[int]*aggregatedChoice{}
	completed := false
	err := ReadStream(resp, func(line string) error {
		chunk, done, err := parseEvent(line)
		if err != nil {
			return err
		}
		if done {
			completed = true
			return errDone
		}
		for _, k := range []string{"id", "created", "model", "usage", "system_fingerprint"} {
			if v, ok := chunk[k]; ok && v != nil {
				result[k] = v
			}
		}
		raw, _ := chunk["choices"].([]interface{})
		for _, v := range raw {
			c, ok := v.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid WorkBuddy choice")
			}
			index := number(c["index"])
			out := choices[index]
			if out == nil {
				out = &aggregatedChoice{Index: index, Message: map[string]interface{}{"role": "assistant", "content": nil}, tools: map[int]map[string]interface{}{}}
				choices[index] = out
			}
			if fr, ok := c["finish_reason"].(string); ok && fr != "" {
				if fr == "error" {
					return fmt.Errorf("WorkBuddy CN reported finish_reason: error")
				}
				out.FinishReason = fr
			}
			delta, _ := c["delta"].(map[string]interface{})
			for _, key := range []string{"content", "reasoning_content", "refusal"} {
				if part, ok := delta[key].(string); ok {
					old, _ := out.Message[key].(string)
					out.Message[key] = old + part
				}
			}
			calls, _ := delta["tool_calls"].([]interface{})
			for _, rawCall := range calls {
				call, ok := rawCall.(map[string]interface{})
				if !ok {
					return fmt.Errorf("invalid WorkBuddy tool call")
				}
				i := number(call["index"])
				tool := out.tools[i]
				if tool == nil {
					tool = map[string]interface{}{"function": map[string]interface{}{"name": "", "arguments": ""}}
					out.tools[i] = tool
				}
				for _, key := range []string{"id", "type"} {
					if value, ok := call[key]; ok {
						tool[key] = value
					}
				}
				f, _ := call["function"].(map[string]interface{})
				target := tool["function"].(map[string]interface{})
				for _, key := range []string{"name", "arguments"} {
					if value, ok := f[key].(string); ok {
						old, _ := target[key].(string)
						target[key] = old + value
					}
				}
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDone) {
		return err
	}
	if !completed {
		return fmt.Errorf("WorkBuddy CN stream ended before [DONE]")
	}
	if len(choices) == 0 {
		return fmt.Errorf("WorkBuddy CN returned no choices")
	}
	indices := []int{}
	for i := range choices {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	list := []*aggregatedChoice{}
	for _, i := range indices {
		c := choices[i]
		if c.FinishReason == nil {
			return fmt.Errorf("WorkBuddy CN choice has no finish_reason")
		}
		if len(c.tools) > 0 {
			keys := []int{}
			for k := range c.tools {
				keys = append(keys, k)
			}
			sort.Ints(keys)
			calls := []map[string]interface{}{}
			for _, k := range keys {
				calls = append(calls, c.tools[k])
			}
			c.Message["tool_calls"] = calls
		}
		list = append(list, c)
	}
	result["choices"] = list
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(result)
}
func number(v interface{}) int { n, _ := v.(float64); return int(n) }

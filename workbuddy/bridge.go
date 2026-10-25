package workbuddy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Bridge handles WorkBuddy API requests with admission control and streaming.
type Bridge struct {
	cfg       *Config
	client    *Client
	chatSlots chan struct{}
}

// NewBridge creates a new WorkBuddy Bridge instance.
func NewBridge(cfg *Config) *Bridge {
	concurrency := 2
	if cfg.Concurrency > 0 {
		concurrency = cfg.Concurrency
	}
	return &Bridge{
		cfg:       cfg,
		client:    NewClient(cfg),
		chatSlots: make(chan struct{}, concurrency),
	}
}

// Config returns the active configuration.
func (b *Bridge) Config() *Config {
	return b.cfg
}

// HandleChat processes an incoming chat completion request for WorkBuddy models.
func (b *Bridge) HandleChat(ctx context.Context, w http.ResponseWriter, reqBody map[string]interface{}) error {
	stream, _ := reqBody["stream"].(bool)
	modelParam, _ := reqBody["model"].(string)

	modelInfo, err := ResolveModel(modelParam)
	if err != nil {
		return err
	}

	upstreamReq, err := BuildUpstreamRequest(reqBody, modelInfo.UpstreamKey, b.cfg.DeviceID)
	if err != nil {
		return err
	}

	reqID := NewRequestID()
	created := time.Now().Unix()

	// Admission slot queueing
	select {
	case b.chatSlots <- struct{}{}:
	default:
		log.Printf("[workbuddy] all %d slot(s) busy; queuing request", cap(b.chatSlots))
		select {
		case b.chatSlots <- struct{}{}:
		case <-time.After(15 * time.Second):
			return fmt.Errorf("workbuddy upstream queue timeout")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	defer func() { <-b.chatSlots }()

	if stream {
		return b.handleStream(ctx, w, upstreamReq, reqID, created, modelInfo.ID)
	}
	return b.handleSync(ctx, w, upstreamReq, reqID, created, modelInfo.ID)
}

func (b *Bridge) handleStream(ctx context.Context, w http.ResponseWriter, upstreamReq *UpstreamChatRequest, reqID string, created int64, model string) error {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, _ := w.(http.Flusher)

	// Thread-safe write helper with idle timestamp tracking
	var mu sync.Mutex
	var lastWriteNs atomic.Int64
	lastWriteNs.Store(time.Now().UnixNano())

	writeData := func(data []byte) {
		mu.Lock()
		defer mu.Unlock()
		lastWriteNs.Store(time.Now().UnixNano())
		w.Write(data)
		if flusher != nil {
			flusher.Flush()
		}
	}

	// Immediate connected packet to prevent proxy 504 timeouts
	writeData([]byte(": worker-cn-2api connected\n\n"))

	// Keep-alive heartbeat watchdog
	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-ticker.C:
				if time.Since(time.Unix(0, lastWriteNs.Load())) >= 3*time.Second {
					mu.Lock()
					if flusher != nil {
						w.Write([]byte(": ping\n\n"))
						flusher.Flush()
					}
					lastWriteNs.Store(time.Now().UnixNano())
					mu.Unlock()
				}
			}
		}
	}()

	err := b.client.PostStream(ctx, upstreamReq, func(line string) error {
		delta := ParseSSELine(line)
		if delta == nil {
			return nil
		}
		if delta.IsDone {
			return nil
		}

		if delta.Content != "" || delta.ReasoningContent != "" || delta.FinishReason != "" {
			chunk := MakeOpenAIChunk(reqID, created, model, delta.Content, delta.ReasoningContent, delta.FinishReason)
			writeData(append(chunk, []byte("\n\n")...))
		}
		return nil
	})

	if err != nil {
		log.Printf("[workbuddy] stream error: %v", err)
		errChunk := MakeOpenAIChunk(reqID, created, model, "", "", "error")
		writeData(append(errChunk, []byte("\n\n")...))
		writeData([]byte("data: [DONE]\n\n"))
		return err
	}

	// Terminal chunks
	doneChunk := MakeOpenAIChunk(reqID, created, model, "", "", "stop")
	writeData(append(doneChunk, []byte("\n\n")...))
	writeData([]byte("data: [DONE]\n\n"))
	return nil
}

func (b *Bridge) handleSync(ctx context.Context, w http.ResponseWriter, upstreamReq *UpstreamChatRequest, reqID string, created int64, model string) error {
	var fullContent strings.Builder
	var fullReasoning strings.Builder
	var promptTokens, completionTokens, totalTokens int

	err := b.client.PostStream(ctx, upstreamReq, func(line string) error {
		delta := ParseSSELine(line)
		if delta == nil || delta.IsDone {
			return nil
		}
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
		}
		if delta.ReasoningContent != "" {
			fullReasoning.WriteString(delta.ReasoningContent)
		}
		if delta.TotalTokens > 0 {
			promptTokens = delta.PromptTokens
			completionTokens = delta.CompletionTokens
			totalTokens = delta.TotalTokens
		}
		return nil
	})

	if err != nil {
		return err
	}

	resp := map[string]interface{}{
		"id":      reqID,
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []interface{}{
			map[string]interface{}{
				"index": 0,
				"message": map[string]interface{}{
					"role":              "assistant",
					"content":           fullContent.String(),
					"reasoning_content": fullReasoning.String(),
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      totalTokens,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

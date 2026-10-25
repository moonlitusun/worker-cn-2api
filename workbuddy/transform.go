package workbuddy

import (
	"bytes"
	"encoding/json"
	"qoder2api/requestopts"
	"strings"

	"github.com/google/uuid"
)

// UpstreamChatRequest defines the JSON payload sent to copilot.tencent.com.
type UpstreamChatRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream"`
	Options  map[string]interface{}   `json:"options,omitempty"`
	DeriveID string                   `json:"derive_id"`
}

// BuildUpstreamRequest converts incoming OpenAI request body into the WorkBuddy payload.
func BuildUpstreamRequest(reqBody map[string]interface{}, upstreamModel string, deriveID string) (*UpstreamChatRequest, error) {
	stream, _ := reqBody["stream"].(bool)

	rawMessages, _ := reqBody["messages"].([]interface{})
	messages := make([]map[string]interface{}, 0, len(rawMessages))
	for _, m := range rawMessages {
		if mMap, ok := m.(map[string]interface{}); ok {
			messages = append(messages, mMap)
		}
	}

	options := make(map[string]interface{})
	if temp, ok := reqBody["temperature"]; ok {
		options["temperature"] = temp
	}
	if maxTokens, ok := reqBody["max_tokens"]; ok {
		options["max_tokens"] = maxTokens
	}
	reasoning, err := requestopts.ReasoningEnabled(reqBody)
	if err != nil {
		return nil, err
	}
	options["enable_reasoning"] = reasoning

	return &UpstreamChatRequest{
		Model:    upstreamModel,
		Messages: messages,
		Stream:   stream,
		Options:  options,
		DeriveID: deriveID,
	}, nil
}

// DeltaExtract represents the extracted content or reasoning delta from an upstream chunk.
type DeltaExtract struct {
	Content          string
	ReasoningContent string
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	IsDone           bool
}

// ParseSSELine extracts text, reasoning, and usage from a raw SSE line received from WorkBuddy.
func ParseSSELine(line string) *DeltaExtract {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "data:") {
		return nil
	}
	payload := strings.TrimSpace(s[5:])
	if payload == "[DONE]" {
		return &DeltaExtract{IsDone: true}
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return nil
	}

	out := &DeltaExtract{}

	// Choices
	if choices, ok := raw["choices"].([]interface{}); ok && len(choices) > 0 {
		if first, ok := choices[0].(map[string]interface{}); ok {
			if fr, ok := first["finish_reason"].(string); ok && fr != "" {
				out.FinishReason = fr
			}
			if delta, ok := first["delta"].(map[string]interface{}); ok {
				if c, ok := delta["content"].(string); ok {
					out.Content = c
				}
				if r, ok := delta["reasoning_content"].(string); ok {
					out.ReasoningContent = r
				}
			}
		}
	}

	// Usage
	if usage, ok := raw["usage"].(map[string]interface{}); ok {
		if pt, ok := usage["prompt_tokens"].(float64); ok {
			out.PromptTokens = int(pt)
		}
		if ct, ok := usage["completion_tokens"].(float64); ok {
			out.CompletionTokens = int(ct)
		}
		if tt, ok := usage["total_tokens"].(float64); ok {
			out.TotalTokens = int(tt)
		}
	}

	return out
}

// MakeOpenAIChunk creates a standard OpenAI-compatible SSE completion chunk.
func MakeOpenAIChunk(reqID string, created int64, model string, content, reasoning, finishReason string) []byte {
	delta := make(map[string]interface{})
	if content != "" {
		delta["content"] = content
	}
	if reasoning != "" {
		delta["reasoning_content"] = reasoning
	}

	choice := map[string]interface{}{
		"index": 0,
		"delta": delta,
	}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	} else {
		choice["finish_reason"] = nil
	}

	chunk := map[string]interface{}{
		"id":      reqID,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []interface{}{choice},
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(chunk)
	return append([]byte("data: "), bytes.TrimRight(buf.Bytes(), "\n")...)
}

// NewRequestID generates a standard OpenAI-style chat completion ID.
func NewRequestID() string {
	return "chatcmpl-wb-" + uuid.New().String()[:20]
}

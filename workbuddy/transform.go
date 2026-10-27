package workbuddy

import (
	"encoding/json"
	"fmt"
	"worker-cn-2api/requestopts"

	"github.com/google/uuid"
)

type RequestError struct{ Message string }

func (e *RequestError) Error() string { return e.Message }

// UpstreamChatRequest defines the JSON payload sent to WorkBuddy CN.
type UpstreamChatRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Stream   bool                     `json:"stream"`
	Options  map[string]interface{}   `json:"options,omitempty"`
	DeriveID string                   `json:"derive_id"`
	Extra    map[string]interface{}   `json:"-"`
}

// BuildUpstreamRequest converts incoming OpenAI request body into the WorkBuddy payload.
func BuildUpstreamRequest(reqBody map[string]interface{}, upstreamModel string, deriveID string) (*UpstreamChatRequest, error) {
	extra := make(map[string]interface{}, len(reqBody))
	for k, v := range reqBody {
		extra[k] = v
	}
	if choice, ok := reqBody["tool_choice"].(map[string]interface{}); ok {
		function, _ := choice["function"].(map[string]interface{})
		name, _ := function["name"].(string)
		if choice["type"] != "function" || name == "" {
			return nil, &RequestError{"tool_choice must name a function"}
		}
		tools, _ := reqBody["tools"].([]interface{})
		selected := []interface{}{}
		for _, raw := range tools {
			tool, _ := raw.(map[string]interface{})
			f, _ := tool["function"].(map[string]interface{})
			if f["name"] == name {
				selected = append(selected, raw)
			}
		}
		if len(selected) != 1 {
			return nil, &RequestError{fmt.Sprintf("tool_choice function %q must match one tool", name)}
		}
		extra["tools"], extra["tool_choice"] = selected, "required"
	}
	stream := true

	rawMessages, _ := reqBody["messages"].([]interface{})
	messages := make([]map[string]interface{}, 0, len(rawMessages))
	for _, m := range rawMessages {
		if mMap, ok := m.(map[string]interface{}); ok {
			copyMsg := make(map[string]interface{}, len(mMap))
			for k, v := range mMap {
				copyMsg[k] = v
			}
			if copyMsg["role"] == "developer" {
				copyMsg["role"] = "system"
			}
			messages = append(messages, copyMsg)
		}
	}

	options := make(map[string]interface{})
	if supplied, ok := reqBody["options"].(map[string]interface{}); ok {
		for k, v := range supplied {
			options[k] = v
		}
	}
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
		Extra:    extra,
	}, nil
}

// MarshalJSON preserves OpenAI fields, including tool definitions and sampling controls.
func (r UpstreamChatRequest) MarshalJSON() ([]byte, error) {
	body := make(map[string]interface{}, len(r.Extra)+5)
	for k, v := range r.Extra {
		body[k] = v
	}
	body["model"], body["messages"], body["stream"] = r.Model, r.Messages, true
	body["options"], body["derive_id"] = r.Options, r.DeriveID
	return json.Marshal(body)
}

// NewRequestID generates a standard OpenAI-style chat completion ID.
func NewRequestID() string {
	return "chatcmpl-wb-" + uuid.New().String()[:20]
}

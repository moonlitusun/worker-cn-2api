package trae

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Error struct {
	Status  int
	Code    any
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("Trae CN: %s (code=%v)", e.Message, e.Code) }
func bad(message string) error { return &Error{Status: 400, Message: message} }

// BuildRequest clones the caller's data and emits only fields supported by this lane.
func BuildRequest(body map[string]any) (map[string]any, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, bad("invalid request")
	}
	var src map[string]any
	if err = json.Unmarshal(b, &src); err != nil {
		return nil, bad("invalid request")
	}
	model, _ := src["model"].(string)
	model = strings.TrimPrefix(strings.TrimSpace(model), "trae/")
	if model == "" {
		return nil, bad("model is required")
	}
	if v, present := src["stream"]; present {
		if _, ok := v.(bool); !ok {
			return nil, bad("stream must be a boolean")
		}
	}
	if v, present := src["stream_options"]; present {
		options, ok := v.(map[string]any)
		if !ok {
			return nil, bad("stream_options must be an object")
		}
		if v, present := options["include_usage"]; present {
			if _, ok := v.(bool); !ok {
				return nil, bad("include_usage must be a boolean")
			}
		}
	}
	msgs, ok := src["messages"].([]any)
	if !ok || len(msgs) == 0 {
		return nil, bad("messages must be a non-empty array")
	}
	known := map[string]bool{}
	for _, raw := range msgs {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, bad("each message must be an object")
		}
		role, _ := m["role"].(string)
		if role == "developer" {
			role = "system"
			m["role"] = role
		}
		switch role {
		case "system", "user", "assistant", "tool":
		default:
			return nil, bad("unsupported message role")
		}
		switch c := m["content"].(type) {
		case string:
			m["content"] = []any{map[string]any{"type": "text", "text": c}}
		case []any:
			for _, rawPart := range c {
				part, ok := rawPart.(map[string]any)
				if !ok || part["type"] != "text" {
					return nil, bad("Trae compatibility lane currently supports text content only")
				}
				if _, ok := part["text"].(string); !ok {
					return nil, bad("text content must be a string")
				}
			}
		case nil:
			if role != "assistant" {
				return nil, bad("message content is required")
			}
		default:
			return nil, bad("invalid message content")
		}
		if role == "tool" {
			id, _ := m["tool_call_id"].(string)
			if !known[id] {
				return nil, bad("tool result has no preceding assistant call")
			}
		}
		if calls, exists := m["tool_calls"]; exists {
			list, ok := calls.([]any)
			if !ok || role != "assistant" {
				return nil, bad("invalid assistant tool_calls")
			}
			for _, rawCall := range list {
				call, ok := rawCall.(map[string]any)
				if !ok {
					return nil, bad("invalid tool call")
				}
				fn, ok := call["function"].(map[string]any)
				if !ok {
					return nil, bad("tool call function is required")
				}
				name, _ := fn["name"].(string)
				id, _ := call["id"].(string)
				if name == "" || id == "" {
					return nil, bad("tool call name and id are required")
				}
				if _, ok := fn["arguments"].(string); !ok {
					return nil, bad("tool call arguments must be a string")
				}
				known[id] = true
				call["function_call"] = fn
				delete(call, "function")
			}
		}
	}
	out := map[string]any{"messages": msgs, "function": Function, "stream": true, "config_name": model, "model": model}
	for _, k := range []string{"temperature", "top_p", "max_tokens", "presence_penalty", "frequency_penalty", "seed", "stop"} {
		if v, ok := src[k]; ok {
			out[k] = v
		}
	}
	if n, ok := src["n"]; ok && n != float64(1) {
		return nil, bad("Trae supports n=1 only")
	}
	if v, ok := src["max_completion_tokens"]; ok {
		if _, set := out["max_tokens"]; set {
			return nil, bad("use only one token limit")
		}
		out["max_tokens"] = v
	}
	for _, k := range []string{"response_format", "thinking", "enable_thinking", "chat_template_kwargs", "parallel_tool_calls"} {
		if _, ok := src[k]; ok {
			return nil, bad(k + " is not supported by the Trae compatibility lane")
		}
	}
	if _, ok := src["reasoning_effort"]; ok {
		return nil, bad("reasoning_effort is not verified for this lane; omit it")
	}
	tools := []any{}
	names := map[string]bool{}
	if raw, exists := src["tools"]; exists {
		list, ok := raw.([]any)
		if !ok {
			return nil, bad("tools must be an array")
		}
		for _, item := range list {
			t, ok := item.(map[string]any)
			if !ok || t["type"] != "function" {
				return nil, bad("only function tools are supported")
			}
			fn, ok := t["function"].(map[string]any)
			if !ok {
				return nil, bad("tool function is required")
			}
			name, _ := fn["name"].(string)
			if name == "" || names[name] {
				return nil, bad("tool names must be non-empty and unique")
			}
			names[name] = true
			params := fn["parameters"]
			if params == nil {
				params = map[string]any{"type": "object"}
			}
			if _, ok := params.(map[string]any); !ok {
				return nil, bad("tool parameters must be an object")
			}
			encoded, _ := json.Marshal(params)
			fn["parameters"] = string(encoded)
			tools = append(tools, t)
		}
	}
	choice := "auto"
	if raw, exists := src["tool_choice"]; exists {
		switch v := raw.(type) {
		case string:
			if v != "auto" && v != "none" && v != "required" {
				return nil, bad("invalid tool_choice")
			}
			choice = v
		case map[string]any:
			fn, _ := v["function"].(map[string]any)
			name, _ := fn["name"].(string)
			if v["type"] != "function" || !names[name] {
				return nil, bad("tool_choice must name a declared function")
			}
			choice = name
		default:
			return nil, bad("invalid tool_choice")
		}
	}
	if choice != "none" && len(tools) > 0 {
		out["tools"] = tools
		out["tool_choice"] = choice
	}
	if choice == "required" && len(tools) == 0 {
		return nil, bad("required tool_choice needs tools")
	}
	return out, nil
}

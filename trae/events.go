package trae

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Event struct {
	Name string
	Data map[string]any
}

// ReadEvents preserves SSE event boundaries and refuses truncated/malformed frames.
func ReadEvents(r io.Reader, emit func(Event) error) error {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 10<<20)
	name := ""
	data := []string{}
	size := 0
	dispatch := func() error {
		if len(data) == 0 {
			name = ""
			size = 0
			return nil
		}
		// Current IDE models send non-JSON progress notices while queued.
		// They carry no completion data and must not fail a valid text stream.
		if name == "progress_notice" {
			name = ""
			data = nil
			size = 0
			return nil
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(strings.Join(data, "\n")), &obj); err != nil || obj == nil {
			return fmt.Errorf("invalid Trae SSE JSON for event %q", name)
		}
		e := Event{Name: name, Data: obj}
		name = ""
		data = nil
		size = 0
		return emit(e)
	}
	for s.Scan() {
		line := s.Text()
		if line == "" {
			if err := dispatch(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			name = value
		case "data":
			data = append(data, value)
			size += len(value) + 1
		}
		if size > 10<<20 {
			return fmt.Errorf("Trae SSE frame exceeds 10 MiB")
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	if len(data) > 0 {
		return io.ErrUnexpectedEOF
	}
	return nil
}

type tool struct{ ID, Name, Arguments string }
type completion struct {
	Content, Reasoning strings.Builder
	Tools              map[int]*tool
	Usage              map[string]any
	Finish             string
	Done               bool
	bytes              int
}

func (c *completion) apply(e Event) (map[string]any, error) {
	if c.Tools == nil {
		c.Tools = map[int]*tool{}
	}
	switch e.Name {
	case "metadata", "timing_cost", "extra_info":
		return nil, nil // extra_info repeats full reasoning
	case "token_usage":
		c.Usage = e.Data
		return nil, nil
	case "error":
		msg, _ := e.Data["message"].(string)
		if msg == "" {
			msg, _ = e.Data["msg"].(string)
		}
		if msg == "" {
			msg = "upstream stream error"
		}
		code := e.Data["code"]
		if code == nil {
			code = e.Data["biz_code"]
		}
		status := 502
		switch code {
		case float64(4001), float64(4023):
			status = 400
		case float64(1005), float64(4008):
			status = 429
		}
		return nil, &Error{Status: status, Code: code, Message: msg}
	case "done":
		fr, _ := e.Data["finish_reason"].(string)
		switch fr {
		case "stop", "length", "tool_calls", "content_filter":
		default:
			return nil, fmt.Errorf("Trae done event has invalid finish_reason")
		}
		if fr == "tool_calls" && len(c.Tools) == 0 {
			return nil, fmt.Errorf("Trae finished tool_calls without calls")
		}
		for _, t := range c.Tools {
			if t.ID == "" || t.Name == "" {
				return nil, fmt.Errorf("incomplete Trae tool call")
			}
			if !json.Valid([]byte(t.Arguments)) {
				return nil, fmt.Errorf("invalid Trae tool arguments")
			}
		}
		c.Finish = fr
		c.Done = true
		return nil, nil
	case "output":
		delta := map[string]any{}
		for _, field := range []string{"response", "reasoning_content"} {
			v, exists := e.Data[field]
			if !exists || v == nil {
				continue
			}
			text, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("invalid Trae output %s", field)
			}
			if text == "" {
				continue
			}
			c.bytes += len(text)
			if field == "response" {
				c.Content.WriteString(text)
				delta["content"] = text
			} else {
				c.Reasoning.WriteString(text)
				delta["reasoning_content"] = text
			}
		}
		if raw := e.Data["tool_calls"]; raw != nil {
			list, ok := raw.([]any)
			if !ok {
				if one, yes := raw.(map[string]any); yes {
					list = []any{one}
				} else {
					return nil, fmt.Errorf("invalid Trae tool_calls")
				}
			}
			calls := []any{}
			for pos, rawCall := range list {
				call, ok := rawCall.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid Trae tool call")
				}
				idx := pos
				if rawIndex, exists := call["index"]; exists {
					n, ok := rawIndex.(float64)
					if !ok || n < 0 || n > 1024 || n != float64(int(n)) {
						return nil, fmt.Errorf("invalid Trae tool index")
					}
					idx = int(n)
				} else if id, _ := call["id"].(string); id != "" {
					found := false
					for i, t := range c.Tools {
						if t.ID == id {
							idx = i
							found = true
							break
						}
					}
					if !found {
						for c.Tools[idx] != nil {
							idx++
						}
					}
				} else if len(c.Tools) > 1 && len(list) == 1 {
					return nil, fmt.Errorf("ambiguous Trae tool fragment without index or id")
				}
				if idx > 1024 {
					return nil, fmt.Errorf("too many Trae tool calls")
				}
				t := c.Tools[idx]
				if t == nil {
					t = &tool{}
					c.Tools[idx] = t
				}
				out := map[string]any{"index": idx, "type": "function"}
				if id, ok := call["id"].(string); ok && id != "" {
					if t.ID != "" && id != t.ID {
						return nil, fmt.Errorf("Trae tool id changed at one index")
					}
					if t.ID == "" {
						out["id"] = id
					}
					t.ID = id
				}
				fn, _ := call["function"].(map[string]any)
				if fn == nil {
					fn, _ = call["function_call"].(map[string]any)
				}
				if fn == nil {
					return nil, fmt.Errorf("Trae tool function is missing")
				}
				f := map[string]any{}
				if name, ok := fn["name"].(string); ok && name != "" {
					if t.Name != "" && name != t.Name {
						return nil, fmt.Errorf("Trae tool name changed")
					}
					if t.Name == "" {
						f["name"] = name
					}
					t.Name = name
				}
				if v, exists := fn["arguments"]; exists && v != nil {
					args, ok := v.(string)
					if !ok {
						return nil, fmt.Errorf("Trae tool arguments must be strings")
					}
					t.Arguments += args
					c.bytes += len(args)
					f["arguments"] = args
				}
				out["function"] = f
				calls = append(calls, out)
			}
			if len(calls) > 0 {
				delta["tool_calls"] = calls
			}
		}
		if c.bytes > 32<<20 {
			return nil, fmt.Errorf("Trae response exceeds 32 MiB")
		}
		for _, t := range c.Tools {
			if len(t.Arguments) > 10<<20 {
				return nil, fmt.Errorf("Trae tool arguments exceed 10 MiB")
			}
		}
		return delta, nil
	default:
		return nil, fmt.Errorf("unsupported Trae SSE event %q", e.Name)
	}
}

func (c *completion) message() map[string]any {
	m := map[string]any{"role": "assistant", "content": c.Content.String()}
	if c.Reasoning.Len() > 0 {
		m["reasoning_content"] = c.Reasoning.String()
	}
	if len(c.Tools) > 0 {
		indices := []int{}
		for i := range c.Tools {
			indices = append(indices, i)
		}
		sort.Ints(indices)
		calls := []any{}
		for _, i := range indices {
			t := c.Tools[i]
			calls = append(calls, map[string]any{"id": t.ID, "type": "function", "function": map[string]any{"name": t.Name, "arguments": t.Arguments}})
		}
		m["tool_calls"] = calls
		if c.Content.Len() == 0 {
			m["content"] = nil
		}
	}
	return m
}

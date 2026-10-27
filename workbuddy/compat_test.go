package workbuddy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"worker-cn-2api/auth"
)

func TestCNCompatibility(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v2/enterprises/personal/models" {
					fmt.Fprint(w, `{"models":[{"id":"glm-5.3","name":"GLM-5.3","supportsToolCall":true}]}`)
					return
				}
				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)
				if body["stream"] != true || body["model"] != "glm-5.3" || body["tools"] == nil {
					t.Errorf("lost upstream controls: %v", body)
				}
				msgs := body["messages"].([]interface{})
				if msgs[0].(map[string]interface{})["role"] != "system" {
					t.Error("developer role not normalized")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"id\":\"chatcmpl-test\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read\",\"arguments\":\"{\\\"path\\\":\"}}]}}]}\n\n")
				fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"a\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n")
				fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":8,\"total_tokens\":13}}\n\ndata: [DONE]\n\n")
			}))
			defer srv.Close()
			b := NewBridge(&Config{Token: "test", Endpoint: srv.URL + "/v2/chat/completions"})
			req := map[string]interface{}{"model": "workbuddy/glm-5.3", "stream": stream, "messages": []interface{}{map[string]interface{}{"role": "developer", "content": "help"}}, "tools": []interface{}{map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "read"}}}}
			w := httptest.NewRecorder()
			if err := b.HandleChat(context.Background(), w, req); err != nil {
				t.Fatal(err)
			}
			body := w.Body.String()
			if !strings.Contains(body, "call_1") || !strings.Contains(body, `"finish_reason":"tool_calls"`) || !strings.Contains(body, `"total_tokens":13`) {
				t.Fatalf("lost tools, finish reason or usage: %s", body)
			}
			if !stream {
				var out struct {
					Choices []struct {
						Message struct {
							ToolCalls []struct{ Function struct{ Arguments string } } `json:"tool_calls"`
						}
					}
				}
				if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
					t.Fatal(err)
				}
				if out.Choices[0].Message.ToolCalls[0].Function.Arguments != `{"path":"a"}` {
					t.Fatalf("arguments not aggregated: %s", body)
				}
			}
		})
	}
}

func TestUpstreamFailureBeforeStreamDoesNotCommitSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprint(w, `{"code":11102,"msg":"model unavailable"}`)
	}))
	defer srv.Close()
	b := NewBridge(&Config{Token: "test", Endpoint: srv.URL + "/v2/chat/completions"})
	for _, stream := range []bool{false, true} {
		w := httptest.NewRecorder()
		err := b.HandleChat(context.Background(), w, map[string]interface{}{"model": "workbuddy/glm-5.3", "stream": stream, "messages": []interface{}{map[string]interface{}{"role": "user", "content": "hi"}}})
		if err == nil || !strings.Contains(err.Error(), "11102") {
			t.Fatalf("missing original error: %v", err)
		}
		if w.Body.Len() != 0 {
			t.Fatalf("committed response before upstream accepted: %s", w.Body.String())
		}
	}
}

func TestCatalogCacheAndTokenChange(t *testing.T) {
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if r.Header.Get("Authorization") == "Bearer second" {
			fmt.Fprint(w, `{"models":[{"id":"new-model"}]}`)
			return
		}
		fmt.Fprint(w, `{"models":[{"id":"glm-5.3"},{"id":"glm-5.3"},{"id":"image","tags":["text-to-image"]}]}`)
	}))
	defer srv.Close()
	b := NewBridge(&Config{Token: "first", Endpoint: srv.URL + "/v2/chat/completions"})
	for i := 0; i < 2; i++ {
		m, err := b.Models(context.Background())
		if err != nil || len(m) != 1 || m[0].ID != "glm-5.3" {
			t.Fatalf("catalog=%v err=%v", m, err)
		}
	}
	if count != 1 {
		t.Fatalf("catalog not cached: %d", count)
	}
	b.UpdateToken("second")
	m, err := b.Models(context.Background())
	if err != nil || m[0].ID != "new-model" || count != 2 {
		t.Fatalf("stale catalog after credential update: %v %v", m, err)
	}
}

func TestStreamFailureIsExplicit(t *testing.T) {
	for _, payload := range []string{`data: {"error":{"message":"quota exceeded"}}

`, `data: {"choices":[{"index":0,"delta":{"content":"partial"}}]}

`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, payload)
		}))
		b := NewBridge(&Config{Token: "test", Endpoint: srv.URL + "/v2/chat/completions"})
		for _, stream := range []bool{false, true} {
			w := httptest.NewRecorder()
			err := b.HandleChat(context.Background(), w, map[string]interface{}{"model": "glm-5.3", "stream": stream})
			if err == nil {
				t.Fatal("failed stream reported success")
			}
			if stream && !strings.Contains(w.Body.String(), `"error"`) {
				t.Fatalf("missing SSE error: %s", w.Body.String())
			}
			if !stream && w.Body.Len() != 0 {
				t.Fatal("partial non-streaming success")
			}
		}
		srv.Close()
	}
}

func TestNamedToolChoiceIsNormalized(t *testing.T) {
	input := map[string]interface{}{"messages": []interface{}{}, "tool_choice": map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "read"}}, "tools": []interface{}{map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "read"}}, map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "write"}}}}
	req, err := BuildUpstreamRequest(input, "glm-5.3", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(req)
	var out map[string]interface{}
	json.Unmarshal(raw, &out)
	if out["tool_choice"] != "required" || len(out["tools"].([]interface{})) != 1 {
		t.Fatalf("named tool choice not preserved: %s", raw)
	}
	if len(input["tools"].([]interface{})) != 2 {
		t.Fatal("mutated caller input")
	}
}

func TestUpstreamTimeouts(t *testing.T) {
	previous := auth.CurrentStreamTimeouts()
	defer auth.SetStreamTimeouts(previous.Header, previous.Idle)
	for _, headers := range []bool{false, true} {
		t.Run(fmt.Sprint(headers), func(t *testing.T) {
			auth.SetStreamTimeouts(50*time.Millisecond, 50*time.Millisecond)
			release := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				if headers {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, ": heartbeat\n\n")
					w.(http.Flusher).Flush()
				}
				select {
				case <-r.Context().Done():
				case <-release:
				}
			}))
			defer srv.Close()
			defer close(release)
			b := NewBridge(&Config{Token: "test", Endpoint: srv.URL + "/v2/chat/completions"})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := b.HandleChat(ctx, httptest.NewRecorder(), map[string]interface{}{"model": "test", "stream": true})
			if err == nil || ctx.Err() != nil {
				t.Fatalf("configured timeout not applied: %v ctx=%v", err, ctx.Err())
			}
		})
	}
}

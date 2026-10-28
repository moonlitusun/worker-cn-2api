package trae

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"worker-cn-2api/stats"
)

// Synthetic data with the public captured event schema; no account data.
const fixture = ": upstream heartbeat\r\n\r\nevent: metadata\r\ndata: {}\r\n\r\n" +
	"event: output\ndata: {\"response\":\"hello\",\n" + "data: \"reasoning_content\":\"think\"}\n\n" +
	"event: output\ndata: {\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function_call\":{\"name\":\"weather\",\"arguments\":\"{\\\"city\\\":\"}}]}\n\n" +
	"event: output\ndata: {\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function_call\":{\"name\":\"weather\",\"arguments\":\"\\\"Taipei\\\"}\"}}]}\n\n" +
	"event: extra_info\ndata: {\"reasoning_content\":\"think\"}\n\n" +
	"event: token_usage\ndata: {\"prompt_tokens\":3,\"completion_tokens\":5,\"total_tokens\":8}\n\n" +
	"event: done\ndata: {\"finish_reason\":\"tool_calls\"}\n\n"

func body(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRequestToolsAndContinuation(t *testing.T) {
	in := body(t, `{"model":"trae/vendor/model","messages":[{"role":"developer","content":"rules"},{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"weather","arguments":"{}"}}]},{"role":"tool","tool_call_id":"c1","content":"sunny"}],"tools":[{"type":"function","function":{"name":"weather","parameters":{"type":"object"}}}],"tool_choice":{"type":"function","function":{"name":"weather"}},"stream":false}`)
	out, err := BuildRequest(in)
	if err != nil {
		t.Fatal(err)
	}
	if out["stream"] != true || out["function"] != Function || out["model"] != "vendor/model" || out["tool_choice"] != "weather" {
		t.Fatal(out)
	}
	msgs := out["messages"].([]any)
	if msgs[0].(map[string]any)["role"] != "system" {
		t.Fatal(msgs)
	}
	call := msgs[1].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)
	if call["function_call"] == nil || call["function"] != nil {
		t.Fatal(call)
	}
	fn := out["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if fn["parameters"] != `{"type":"object"}` {
		t.Fatal(fn)
	}
	if in["messages"].([]any)[0].(map[string]any)["role"] != "developer" {
		t.Fatal("input was mutated")
	}
}

func TestInvalidRequestsFailBeforeUpstream(t *testing.T) {
	for _, s := range []string{
		`{"model":"trae/","messages":[{"role":"user","content":"hi"}]}`,
		`{"model":"trae/x","messages":[{"role":"tool","tool_call_id":"orphan","content":"x"}]}`,
		`{"model":"trae/x","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.com/image"}}]}]}`,
		`{"model":"trae/x","messages":[{"role":"user","content":"hi"}],"tool_choice":"required"}`,
		`{"model":"trae/x","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"high"}`,
		`{"model":"trae/x","messages":[{"role":"user","content":"hi"}],"tools":[{}]}`,
	} {
		if _, err := BuildRequest(body(t, s)); err == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}

func TestBridgeStreamingAndSync(t *testing.T) {
	var mu sync.Mutex
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		if r.URL.Path != "/api/agent/v3/llm_utils_chat" || r.Header.Get("Authorization") != "Cloud-IDE-JWT secret-token" {
			t.Error("wrong upstream route or auth")
		}
		var m map[string]any
		json.NewDecoder(r.Body).Decode(&m)
		if m["stream"] != true || m["function"] != Function || m["model"] != "model" {
			t.Errorf("bad upstream body: %v", m)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: token_usage\ndata: {\"prompt_tokens\":1,\"completion_tokens\":1}\n\n"+fixture)
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "secret-token", BaseURL: srv.URL})
	for _, stream := range []bool{false, true} {
		in := body(t, `{"model":"trae/model","messages":[{"role":"user","content":"hi"}],"stream_options":{"include_usage":true}}`)
		in["stream"] = stream
		w := httptest.NewRecorder()
		recorder := stats.NewRecorder(nil)
		if err := b.HandleChat(context.Background(), w, in, recorder); err != nil {
			t.Fatal(err)
		}
		for _, provider := range []string{"", "trae"} {
			report := recorder.Report(provider)
			if report.PromptTokens != 3 || report.CompletionTokens != 5 {
				t.Fatalf("usage not recorded once for %q: %+v", provider, report)
			}
		}
		if recorder.Report("workbuddy").PromptTokens != 0 {
			t.Fatal("usage leaked to other channel")
		}
		if stream {
			if !strings.Contains(w.Body.String(), "data: [DONE]") || !strings.Contains(w.Body.String(), `"total_tokens":8`) {
				t.Fatal(w.Body.String())
			}
			if strings.Count(w.Body.String(), `"name":"weather"`) != 1 {
				t.Fatal("repeated tool name leaked as deltas")
			}
		} else {
			var result map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			choice := result["choices"].([]any)[0].(map[string]any)
			m := choice["message"].(map[string]any)
			if m["content"] != "hello" || m["reasoning_content"] != "think" {
				t.Fatal(m)
			}
			fn := m["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)
			if fn["arguments"] != `{"city":"Taipei"}` {
				t.Fatal(fn)
			}
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if requests != 2 {
		t.Fatal(requests)
	}
}

func TestNoFalseSuccessOnBrokenStream(t *testing.T) {
	for _, broken := range []string{
		"event: output\ndata: {\"response\":\"partial\"}\n\n",
		"event: output\ndata: not-json\n\n",
		"event: error\ndata: {\"code\":4001,\"message\":\"bad model\"}\n\n",
		"event: done\ndata: {}\n\n",
		"event: done\ndata: {\"finish_reason\":\"stop\"}",
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, broken)
		}))
		b := NewBridge(Config{Token: "token", BaseURL: srv.URL})
		for _, stream := range []bool{true, false} {
			in := body(t, `{"model":"trae/m","messages":[{"role":"user","content":"hi"}]}`)
			in["stream"] = stream
			w := httptest.NewRecorder()
			if err := b.HandleChat(context.Background(), w, in); err == nil {
				t.Fatalf("accepted truncated/error stream %q", broken)
			}
			if strings.Contains(w.Body.String(), "[DONE]") || strings.Contains(w.Body.String(), `"finish_reason":"stop"`) {
				t.Fatal("false success", w.Body.String())
			}
			if !stream && w.Body.Len() != 0 {
				t.Fatal("sync output committed before validation")
			}
		}
		srv.Close()
	}
}

func TestCatalogIsolationFilteringAndNoFallback(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		if r.Header.Get("X-Cloudide-Token") == "bad" {
			fmt.Fprint(w, `{"code":123,"message":"not allowed"}`)
			return
		}
		fmt.Fprint(w, `{"config_info_list":[{"config_name":"visible","display_config":{"display_name":"Visible"},"context_window_tokens":{"dev":8192}},{"config_name":"hidden","is_invisible_to_user":true,"display_config":{"display_name":"Hidden"}},{"config_name":"disabled","config_switch":false,"display_config":{"display_name":"Disabled"}},{"config_name":"raw","raw_chat_function":"solo_agent","display_config":{"display_name":"Raw"}}]}`)
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "first", BaseURL: srv.URL})
	for i := 0; i < 2; i++ {
		models, err := b.Models(context.Background())
		if err != nil || len(models) != 1 || models[0].ID != "visible" || models[0].ContextWindow != 8192 {
			t.Fatalf("%v %v", models, err)
		}
		models[0].ID = "mutated"
	}
	b.UpdateCredentials("second", "")
	if _, err := b.Models(context.Background()); err != nil {
		t.Fatal(err)
	}
	b.UpdateCredentials("bad", "")
	if _, err := b.Models(context.Background()); err == nil {
		t.Fatal("catalog error masked")
	}
	b.UpdateCredentials("", "")
	if _, err := b.Models(context.Background()); err == nil {
		t.Fatal("cleared token still serving catalog")
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 3 {
		t.Fatal(calls)
	}
}

func TestCancellationReleasesSlot(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
		close(canceled)
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "token", BaseURL: srv.URL})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- b.HandleChat(ctx, httptest.NewRecorder(), map[string]any{"model": "trae/m", "messages": []any{map[string]any{"role": "user", "content": "hi"}}})
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancel succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler leaked")
	}
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream not canceled")
	}
	if len(b.slots) != 0 {
		t.Fatal("slot leaked")
	}
}

func TestInterleavedToolIDsWithoutIndices(t *testing.T) {
	c := &completion{}
	for _, raw := range []string{
		`{"tool_calls":[{"id":"a","function_call":{"name":"one","arguments":"{"}}]}`,
		`{"tool_calls":[{"id":"b","function_call":{"name":"two","arguments":"{"}}]}`,
		`{"tool_calls":[{"id":"a","function_call":{"arguments":"}"}}]}`,
		`{"tool_calls":[{"id":"b","function_call":{"arguments":"}"}}]}`,
	} {
		if _, err := c.apply(Event{Name: "output", Data: body(t, raw)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.apply(Event{Name: "done", Data: map[string]any{"finish_reason": "tool_calls"}}); err != nil {
		t.Fatal(err)
	}
	if len(c.Tools) != 2 || c.Tools[0].Arguments != "{}" || c.Tools[1].Arguments != "{}" {
		t.Fatal("tool fragments crossed calls")
	}
}

func TestIncompleteUsageIsNotRecorded(t *testing.T) {
	for _, stream := range []bool{false, true} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: token_usage\ndata: {\"prompt_tokens\":5,\"completion_tokens\":8}\n\n")
		}))
		b := NewBridge(Config{Token: "test", BaseURL: srv.URL})
		rec := stats.NewRecorder(nil)
		err := b.HandleChat(context.Background(), httptest.NewRecorder(), map[string]any{"model": "trae/glm", "stream": stream, "messages": []any{map[string]any{"role": "user", "content": "hi"}}}, rec)
		srv.Close()
		if err == nil || rec.Report().PromptTokens != 0 || rec.Report("trae").PromptTokens != 0 {
			t.Fatal("incomplete stream was billed", err)
		}
	}
}

func TestProgressNoticeAllowsPlainTextWithoutWeakeningOutputValidation(t *testing.T) {
	input := "event: progress_notice\ndata: Processing_model\n\nevent: progress_notice\ndata: \"Queued\"\n\nevent: output\ndata: {\"response\":\"OK\"}\n\n"
	count := 0
	if err := ReadEvents(strings.NewReader(input), func(e Event) error {
		count++
		if e.Name != "output" {
			t.Fatal(e.Name)
		}
		return nil
	}); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if err := ReadEvents(strings.NewReader("event: output\ndata: malformed\n\n"), func(Event) error { return nil }); err == nil {
		t.Fatal("malformed completion event accepted")
	}
}

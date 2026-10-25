package workbuddy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeriveDeviceID(t *testing.T) {
	id1 := DeriveDeviceID("user-123456")
	id2 := DeriveDeviceID("user-123456")
	if id1 != id2 {
		t.Errorf("DeriveDeviceID must be deterministic, got %s vs %s", id1, id2)
	}

	if !strings.HasPrefix(id1, "wb-") {
		t.Errorf("expected derive_id to start with wb-, got %s", id1)
	}

	idOther := DeriveDeviceID("user-999999")
	if id1 == idOther {
		t.Errorf("different users must have distinct derive_ids")
	}
}

func TestModelsResolution(t *testing.T) {
	// Default
	m, err := ResolveModel("")
	if err != nil || m.ID != "deepseek-v3" {
		t.Errorf("expected default deepseek-v3, got %v, err=%v", m, err)
	}

	// Case-insensitive
	m, err = ResolveModel("DeepSeek-R1")
	if err != nil || m.ID != "deepseek-r1" || !m.IsReasoning {
		t.Errorf("expected deepseek-r1 reasoning model, got %v, err=%v", m, err)
	}

	// Prefix workbuddy/
	m, err = ResolveModel("workbuddy/hunyuan-code")
	if err != nil || m.UpstreamKey != "hunyuan-standard-code" {
		t.Errorf("expected hunyuan-standard-code, got %v, err=%v", m, err)
	}

	// Unsupported
	_, err = ResolveModel("non-existent-model")
	if err == nil {
		t.Error("expected error for unknown model, got nil")
	}
}

func TestBuildUpstreamRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"model": "deepseek-v3",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "hello"},
		},
		"stream":      true,
		"temperature": 0.5,
	}

	req, err := BuildUpstreamRequest(reqBody, "deepseek-v3", "wb-device-123")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest failed: %v", err)
	}

	if req.DeriveID != "wb-device-123" {
		t.Errorf("derive_id mismatch: %s", req.DeriveID)
	}
	if len(req.Messages) != 1 {
		t.Errorf("messages count mismatch: %d", len(req.Messages))
	}
	if req.Options["enable_reasoning"] != true {
		t.Errorf("expected enable_reasoning to be true")
	}
}

func TestParseSSELine(t *testing.T) {
	// Standard content
	line1 := `data: {"id":"1","choices":[{"index":0,"delta":{"content":"Hi"}}]}`
	d1 := ParseSSELine(line1)
	if d1 == nil || d1.Content != "Hi" {
		t.Errorf("failed to parse content: %+v", d1)
	}

	// Reasoning content
	line2 := `data: {"id":"2","choices":[{"index":0,"delta":{"reasoning_content":"Thinking..."}}]}`
	d2 := ParseSSELine(line2)
	if d2 == nil || d2.ReasoningContent != "Thinking..." {
		t.Errorf("failed to parse reasoning: %+v", d2)
	}

	// Done
	dDone := ParseSSELine("data: [DONE]")
	if dDone == nil || !dDone.IsDone {
		t.Errorf("expected isDone=true")
	}
}

func TestBridgeHandleStreamEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-wb-token" {
			http.Error(w, "unauthorized", 401)
			return
		}
		if !strings.HasPrefix(r.Header.Get("X-Device-Id"), "wb-") {
			http.Error(w, "missing device id", 400)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"I think...\"}}]}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello from WorkBuddy!\"}}]}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := &Config{
		Token:    "test-wb-token",
		UserID:   "uid-123",
		Endpoint: srv.URL + "/v2/chat/completions",
		DeviceID: "wb-test-device",
	}
	b := NewBridge(cfg)

	w := httptest.NewRecorder()
	reqBody := map[string]interface{}{
		"model": "deepseek-r1",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "hi"},
		},
		"stream": true,
	}

	err := b.HandleChat(context.Background(), w, reqBody)
	if err != nil {
		t.Fatalf("HandleChat failed: %v", err)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, ": worker-cn-2api connected\n\n") {
		t.Errorf("expected immediate connected packet, got:\n%s", body)
	}
	if !strings.Contains(body, "I think...") {
		t.Errorf("missing reasoning_content in response:\n%s", body)
	}
	if !strings.Contains(body, "Hello from WorkBuddy!") {
		t.Errorf("missing content in response:\n%s", body)
	}
	if !strings.HasSuffix(strings.TrimSpace(body), "data: [DONE]") {
		t.Errorf("stream must terminate with [DONE]")
	}
}

func TestBuildUpstreamRequestHonorsDisabledThinking(t *testing.T) {
	req, err := BuildUpstreamRequest(map[string]interface{}{
		"thinking":    map[string]interface{}{"type": "disabled"},
		"temperature": 0.2, "max_tokens": float64(256),
	}, "deepseek-r1", "test-device")
	if err != nil {
		t.Fatal(err)
	}
	if req.Options["enable_reasoning"] != false {
		t.Fatalf("reasoning was forced on: %v", req.Options)
	}
	if req.Options["temperature"] != 0.2 || req.Options["max_tokens"] != float64(256) {
		t.Fatal("existing controls lost")
	}
}

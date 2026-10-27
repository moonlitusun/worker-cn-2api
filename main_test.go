package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"worker-cn-2api/store"
	"worker-cn-2api/workbuddy"
)

func TestHealthHandler(t *testing.T) {
	s, err := store.New(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	wbCfg := &workbuddy.Config{Token: "test-token"}
	wbBridge := workbuddy.NewBridge(wbCfg)
	h := healthHandler(s, wbBridge)

	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
	if _, ok := resp["has_pat"]; ok {
		t.Errorf("has_pat must be absent, got %v", resp["has_pat"])
	}
	if resp["has_workbuddy"] != true {
		t.Errorf("expected has_workbuddy=true, got %v", resp["has_workbuddy"])
	}
	if int(resp["chat_timeout_seconds"].(float64)) != store.DefaultChatTimeoutSeconds {
		t.Errorf("expected default chat timeout %d, got %v", store.DefaultChatTimeoutSeconds, resp["chat_timeout_seconds"])
	}

}

func TestHealthHandlerMethodNotAllowed(t *testing.T) {
	s, err := store.New(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	w := httptest.NewRecorder()
	healthHandler(s, nil)(w, httptest.NewRequest("POST", "/health", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestUnifiedChatRejectsConflictingReasoningBeforeRouting(t *testing.T) {
	s, err := store.New(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddKey("test-key", "parameters"); err != nil {
		t.Fatal(err)
	}
	handler := makeChatHandler(s, nil, nil)
	for _, model := range []string{"DeepSeek-Flash", "workbuddy/deepseek-r1"} {
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"`+model+`","thinking":{"type":"disabled"},"reasoning_effort":"high"}`))
		req.Header.Set("Authorization", "Bearer test-key")
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "conflicting reasoning controls") {
			t.Fatalf("model=%s status=%d body=%s", model, w.Code, w.Body.String())
		}
	}
}

func TestWorkBuddyStartupRestoresSavedToken(t *testing.T) {
	t.Setenv("WORKER_WORKBUDDY_TOKEN", "")
	t.Setenv("CODEBUDDY_AUTH_TOKEN", "")
	file := t.TempDir() + "/data.json"
	original, err := store.New(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := original.SetWorkBuddyToken("saved-workbuddy-token"); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.New(file)
	if err != nil {
		t.Fatal(err)
	}
	b := newWorkBuddyBridge(reopened)
	if b.Config().Token != "saved-workbuddy-token" {
		t.Fatal("saved token lost on startup")
	}
}

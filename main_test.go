package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qoder2api/store"
	"qoder2api/workbuddy"
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
	if resp["has_pat"] != false {
		t.Errorf("expected has_pat=false without PAT, got %v", resp["has_pat"])
	}
	if resp["has_workbuddy"] != true {
		t.Errorf("expected has_workbuddy=true, got %v", resp["has_workbuddy"])
	}
	if int(resp["chat_timeout_seconds"].(float64)) != store.DefaultChatTimeoutSeconds {
		t.Errorf("expected default chat timeout %d, got %v", store.DefaultChatTimeoutSeconds, resp["chat_timeout_seconds"])
	}

	s.SetPAT("pat-x")
	w = httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/health", nil))
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["has_pat"] != true {
		t.Errorf("expected has_pat=true with PAT, got %v", resp["has_pat"])
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

func TestUnifiedModelsHandler(t *testing.T) {
	s, err := store.New(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	provider := newBridgeProvider(s)
	wbCfg := &workbuddy.Config{Token: "test-wb"}
	wbBridge := workbuddy.NewBridge(wbCfg)

	h := makeUnifiedModelsHandler(s, provider, wbBridge)
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/v1/models", nil))

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Must contain Qoder default models
	if !strings.Contains(body, "Qwen3.7-Max") {
		t.Error("models list must contain Qoder model Qwen3.7-Max")
	}
	// Must contain WorkBuddy models
	if !strings.Contains(body, "deepseek-v3") {
		t.Error("models list must contain WorkBuddy model deepseek-v3")
	}
	if !strings.Contains(body, "hunyuan-code") {
		t.Error("models list must contain WorkBuddy model hunyuan-code")
	}
}

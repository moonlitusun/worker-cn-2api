package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"worker-cn-2api/store"
	"worker-cn-2api/trae"
	"worker-cn-2api/workbuddy"
)

func TestTraeRoutesAndMixedCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/enterprises/personal/models":
			fmt.Fprint(w, `{"models":[{"id":"wb-model"}]}`)
		case "/api/ide/v1/get_detail_param":
			fmt.Fprint(w, `{"config_info_list":[{"config_name":"tr-model","display_config":{"display_name":"Trae Model"}}]}`)
		case "/api/agent/v3/llm_utils_chat":
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: output\ndata: {\"response\":\"Trae reply\"}\n\nevent: done\ndata: {\"finish_reason\":\"stop\"}\n\n")
		default:
			t.Error("unexpected route", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	st, err := store.New(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.AddKey("client-key", ""); err != nil {
		t.Fatal(err)
	}
	wb := workbuddy.NewBridge(&workbuddy.Config{Token: "wb-token", Endpoint: srv.URL + "/v2/chat/completions"})
	tb := trae.NewBridge(trae.Config{Token: "tr-token", BaseURL: srv.URL})
	r := httptest.NewRequest("GET", "/v1/models", nil)
	r.Header.Set("Authorization", "Bearer client-key")
	w := httptest.NewRecorder()
	makeModelsHandler(st, wb, tb)(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"trae/tr-model"`) || !strings.Contains(w.Body.String(), `"id":"workbuddy/wb-model"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	for _, stream := range []string{"false", "true"} {
		r = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"trae/tr-model","messages":[{"role":"user","content":"hi"}],"stream":`+stream+`}`))
		r.Header.Set("Authorization", "Bearer client-key")
		w = httptest.NewRecorder()
		makeChatHandler(st, wb, nil, tb)(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Trae reply") {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
	r = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	healthHandler(st, wb, tb)(w, r)
	var health map[string]any
	json.Unmarshal(w.Body.Bytes(), &health)
	if health["has_trae"] != true || strings.Contains(w.Body.String(), "tr-token") {
		t.Fatal(w.Body.String())
	}
	// A Trae-only setup must work without any WorkBuddy credentials.
	r = httptest.NewRequest("GET", "/v1/models", nil)
	r.Header.Set("Authorization", "Bearer client-key")
	w = httptest.NewRecorder()
	makeModelsHandler(st, nil, tb)(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), "wb-model") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestTraeErrorsAndAuthentication(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401); fmt.Fprint(w, "expired token") }))
	defer srv.Close()
	st, err := store.New(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	st.AddKey("key", "")
	tb := trae.NewBridge(trae.Config{Token: "upstream-token", BaseURL: srv.URL})
	h := makeChatHandler(st, nil, nil, tb)
	for _, key := range []string{"", "key"} {
		r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"trae/m","messages":[{"role":"user","content":"hi"}],"stream":true}`))
		if key != "" {
			r.Header.Set("Authorization", "Bearer "+key)
		}
		w := httptest.NewRecorder()
		h(w, r)
		if w.Code != 401 || strings.Contains(w.Body.String(), "connected") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"trae/","messages":[{"role":"user","content":"hi"}]}`))
	r.Header.Set("Authorization", "Bearer key")
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
}

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"worker-cn-2api/store"
	"worker-cn-2api/workbuddy"
)

func TestRoutesExposeUpstreamErrorsAndCNModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/enterprises/personal/models" {
			fmt.Fprint(w, `{"models":[{"id":"glm-5.3"}]}`)
			return
		}
		w.WriteHeader(400)
		fmt.Fprint(w, `{"code":11102,"msg":"model unavailable"}`)
	}))
	defer srv.Close()
	st, err := store.New(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	st.AddKey("test-key", "")
	b := workbuddy.NewBridge(&workbuddy.Config{Token: "upstream-token", Endpoint: srv.URL + "/v2/chat/completions"})
	for _, stream := range []string{"true", "false"} {
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"workbuddy/hunyuan-code","stream":`+stream+`,"messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("Authorization", "Bearer test-key")
		w := httptest.NewRecorder()
		makeChatHandler(st, b, nil)(w, req)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "11102") || strings.Contains(w.Body.String(), "connected") {
			t.Fatalf("error hidden: %d %s", w.Code, w.Body.String())
		}
	}
	req := httptest.NewRequest("GET", "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	makeModelsHandler(st, b)(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "workbuddy/glm-5.3") || strings.Contains(w.Body.String(), "Qwen") {
		t.Fatalf("bad model list: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"id":"glm-5.3"`) {
		t.Fatal("catalog must not advertise duplicate bare aliases")
	}
	req = httptest.NewRequest("GET", "/v1/models", nil)
	w = httptest.NewRecorder()
	makeModelsHandler(st, b)(w, req)
	if w.Code != 401 {
		t.Fatal("catalog must require API key")
	}
}

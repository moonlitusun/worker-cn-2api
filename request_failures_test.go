package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"worker-cn-2api/stats"
	"worker-cn-2api/store"
	"worker-cn-2api/workbuddy"
)

func TestRequestFailurePersistsAndRedacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		fmt.Fprint(w, `{"message":"unavailable upstream-token test-key confidential-message","token":"unknown-value"}`)
	}))
	defer srv.Close()
	path := t.TempDir() + "/data.json"
	st, err := store.New(path)
	if err != nil {
		t.Fatal(err)
	}
	st.AddKey("test-key", "")
	rec := stats.NewRecorder(st)
	b := workbuddy.NewBridge(&workbuddy.Config{Token: "upstream-token", Endpoint: srv.URL})
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"workbuddy/model","messages":[{"role":"user","content":"confidential-message"}]}`))
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	makeChatHandler(st, b, rec)(w, req)
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
	loaded, err := store.New(path)
	if err != nil {
		t.Fatal(err)
	}
	report := stats.NewRecorder(loaded).Report("workbuddy")
	if report.Failed != 1 || len(report.Failures) != 1 {
		t.Fatal("failure was not immediately persisted", report)
	}
	f := report.Failures[0]
	if f.Status != 503 || f.Kind != "upstream_error" || f.Model != "workbuddy/model" {
		t.Fatal(f)
	}
	for _, secret := range []string{"upstream-token", "test-key", "confidential-message", "unknown-value"} {
		if strings.Contains(f.Reason, secret) {
			t.Fatal("secret leaked", secret)
		}
	}
	if len(stats.NewRecorder(loaded).Report("trae").Failures) != 0 {
		t.Fatal("failure leaked across filters")
	}
}

func TestFailureClassification(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
		kind   string
	}{
		{context.DeadlineExceeded, 504, "timeout"}, {context.Canceled, 499, "client_cancelled"},
		{&workbuddy.QueueTimeoutError{}, 429, "queue_timeout"},
		{&workbuddy.UpstreamError{Status: 401}, 401, "authentication"},
		{&workbuddy.UpstreamError{Status: 429}, 429, "rate_limit"},
		{errors.New("stream ended before [DONE]"), 502, "stream_error"},
	} {
		status, kind := classifyRequestFailure(tt.err)
		if status != tt.status || kind != tt.kind {
			t.Fatal(status, kind, tt)
		}
	}
}

func TestHeaderTimeoutIsNotClientCancellation(t *testing.T) {
	status, kind := classifyRequestFailure(context.Canceled, context.Background())
	if status != 504 || kind != "timeout" {
		t.Fatal(status, kind)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	status, kind = classifyRequestFailure(context.Canceled, ctx)
	if status != 499 || kind != "client_cancelled" {
		t.Fatal(status, kind)
	}
}

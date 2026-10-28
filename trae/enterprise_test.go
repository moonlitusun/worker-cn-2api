package trae

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEnterprisePATExchangeAndRenewal(t *testing.T) {
	var exchanges atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloudide/api/v3/trae/oauth/ExchangeToken" {
			t.Error("wrong path", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("PAT sent as an access token")
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["RefreshToken"] != "trae-lt-test" || body["ClientID"] != EnterpriseClientID {
			t.Error("wrong exchange body")
		}
		exchanges.Add(1)
		fmt.Fprintf(w, `{"code":0,"Data":{"Token":"enterprise-access","RefreshToken":"not-persisted","TokenExpireAt":%d}}`, time.Now().Add(time.Hour).UnixMilli())
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "trae-lt-test", EnterpriseURL: srv.URL})
	b.SetCredentialSaver(func(Config) error { t.Error("original PAT must remain saved"); return nil })
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := b.ensureCredentials(context.Background())
			if err != nil || c.Token != "enterprise-access" || c.BaseURL != srv.URL || c.ClientID != EnterpriseClientID {
				t.Errorf("enterprise exchange failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if exchanges.Load() != 1 || b.Config().Token != "trae-lt-test" {
		t.Fatal("exchange was repeated or PAT was overwritten")
	}
	if s := b.Status(); !s.RefreshAvailable || s.ExpiresAt == "" {
		t.Fatal("missing effective expiry/renewal status")
	}
	b.mu.Lock()
	b.enterpriseSession.ExpiresAt = time.Now().Add(-time.Hour).Format(time.RFC3339)
	b.mu.Unlock()
	if _, err := b.ensureCredentials(context.Background()); err != nil || exchanges.Load() != 2 {
		t.Fatal("expired session not renewed", err)
	}
	b.UpdateCredentials("plain-jwt", "")
	c, err := b.ensureCredentials(context.Background())
	if err != nil || c.Token != "plain-jwt" || c.BaseURL == srv.URL {
		t.Fatal("cached enterprise account leaked into replacement", err)
	}
}

func TestEnterprisePATFailureRedactsResponse(t *testing.T) {
	for _, status := range []int{200, 401, 302} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			fmt.Fprint(w, `{"code":1001,"message":"trae-lt-secret enterprise-access","Data":{"Token":"enterprise-access"}}`)
		}))
		b := NewBridge(Config{Token: "trae-lt-secret", EnterpriseURL: srv.URL})
		_, err := b.ensureCredentials(context.Background())
		srv.Close()
		if err == nil || strings.Contains(err.Error(), "trae-lt-secret") || strings.Contains(err.Error(), "enterprise-access") {
			t.Fatal("unsafe enterprise error", err)
		}
	}
}

func TestEnterpriseCatalogAndRawChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		switch r.URL.Path {
		case "/cloudide/api/v3/trae/oauth/ExchangeToken":
			fmt.Fprintf(w, `{"code":0,"Data":{"Token":"access","TokenExpireAt":%d}}`, time.Now().Add(time.Hour).Unix())
		case "/api/ide/v1/batch_get_detail_param":
			if r.Header.Get("X-Ide-Version") != "99.99.99" {
				t.Error("wrong enterprise compatibility profile")
			}
			functions, _ := body["functions"].([]any)
			if len(functions) != 2 || functions[0] != "chat_v3" || functions[1] != "chat" {
				t.Error("missing current IDE/legacy catalog union")
			}
			if body["access_type"] != float64(4) || body["agent_type"] != "chat" {
				t.Error("wrong enterprise catalog body")
			}
			fmt.Fprint(w, `{"function_configs":[{"function":"chat","config_info_list":[{"config_name":"model","display_config":{"display_name":"Old model"},"model_detail_list":[{"model_name":"old-backend"}]},{"config_name":"legacy","display_config":{"display_name":"Legacy"},"model_detail_list":[{"model_name":"legacy"}]}]},{"function":"chat_v3","config_info_list":[{"config_name":"model","display_config":{"display_name":"Model"},"model_detail_list":[{"model_name":"model__max"},{"model_name":"model__dev"}]},{"config_name":"custom","display_config":{"display_name":"Custom","is_custom_model":true},"model_detail_list":[{"model_name":"custom"}]}]}]}`)
		case "/api/ide/v2/llm_raw_chat":
			if r.Header.Get("Authorization") != "Cloud-IDE-JWT access" || body["function"] != "chat_v3" || body["model_name"] != "model__dev" || body["config_name"] != "model" || body["session_id"] == "" || body["access_type"] != float64(4) {
				t.Error("wrong enterprise chat body")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, fixture)
		default:
			t.Error("wrong endpoint", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	b := NewBridge(Config{Token: "trae-lt-test", EnterpriseURL: srv.URL})
	models, err := b.Models(context.Background())
	if err != nil || len(models) != 2 || models[0].BackendModel != "model__dev" {
		t.Fatal(models, err)
	}
	for _, stream := range []bool{false, true} {
		in := map[string]any{"model": "trae/model", "stream": stream, "messages": []any{map[string]any{"role": "user", "content": "hi"}}}
		if err := b.HandleChat(context.Background(), httptest.NewRecorder(), in); err != nil {
			t.Fatal(err)
		}
	}
}

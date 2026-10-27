package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"worker-cn-2api/trae"
)

func TestTraeConnectionProbeAndStatus(t *testing.T) {
	a, _ := newAdmin(t)
	var reject atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reject.Load() {
			w.WriteHeader(401)
			fmt.Fprint(w, "expired secret-token")
			return
		}
		fmt.Fprint(w, `{"config_info_list":[{"config_name":"model","display_config":{"display_name":"Model"},"context_window_tokens":{"dev":8192}}]}`)
	}))
	defer srv.Close()
	a.SetTraeBridge(trae.NewBridge(trae.Config{Token: "secret-token", RefreshToken: "secret-refresh", BaseURL: srv.URL}))
	mux := http.NewServeMux()
	a.RegisterRoutes(mux)
	token, err := a.createSession()
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		r.AddCookie(&http.Cookie{Name: "admin_session", Value: token})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	w := request("POST", "/admin/api/trae/test")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"model_count":1`) {
		t.Fatalf("probe unavailable: %d %s", w.Code, w.Body.String())
	}
	w = request("GET", "/admin/api/trae")
	var response struct {
		Status struct {
			RefreshAvailable bool `json:"refresh_available"`
			LastTest         struct {
				Success bool `json:"success"`
			} `json:"last_test"`
		} `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || !response.Status.RefreshAvailable || !response.Status.LastTest.Success {
		t.Fatal("missing diagnostics", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "secret-token") || strings.Contains(w.Body.String(), "secret-refresh") {
		t.Fatal("diagnostics exposed credentials")
	}
	reject.Store(true)
	w = request("POST", "/admin/api/trae/test")
	if w.Code != 502 {
		t.Fatal("probe used cached catalog", w.Code, w.Body.String())
	}
	w = request("GET", "/admin/api/trae")
	if strings.Contains(w.Body.String(), "secret-token") || !strings.Contains(w.Body.String(), `"success":false`) {
		t.Fatal(w.Body.String())
	}
	w = request("GET", "/admin/api/trae/test")
	if w.Code != 405 {
		t.Fatal("probe must require POST")
	}
	r := httptest.NewRequest("POST", "/admin/api/trae/test", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("unprotected probe")
	}
}

func TestTraeAdminCredentialsAndModels(t *testing.T) {
	a, st := newAdmin(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"config_info_list":[{"config_name":"model","display_config":{"display_name":"Model"}}]}`)
	}))
	defer srv.Close()
	tb := trae.NewBridge(trae.Config{BaseURL: srv.URL})
	a.SetTraeBridge(tb)
	mux := http.NewServeMux()
	a.RegisterRoutes(mux)
	r := httptest.NewRequest("POST", "/admin/api/trae", strings.NewReader(`{"token":"stolen"}`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("unprotected credential endpoint")
	}
	w = doRequest(t, a.handleTrae, "POST", "/admin/api/trae", map[string]string{"token": "secret-trae-token", "user_id": "user"})
	if w.Code != 200 || tb.Config().Token != "secret-trae-token" {
		t.Fatal(w.Code, w.Body.String())
	}
	if token, uid := st.GetTraeCredentials(); token != "secret-trae-token" || uid != "user" {
		t.Fatal("credentials not saved")
	}
	w = doRequest(t, a.handleTrae, "GET", "/admin/api/trae", nil)
	if strings.Contains(w.Body.String(), "secret-trae-token") || !strings.Contains(w.Body.String(), `"configured":true`) {
		t.Fatal(w.Body.String())
	}
	w = doRequest(t, a.handleModels, "GET", "/admin/api/models", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "trae/model") || !strings.Contains(w.Body.String(), "trae-cn") {
		t.Fatal(w.Code, w.Body.String())
	}
	w = doRequest(t, a.handleTrae, "POST", "/admin/api/trae", map[string]string{"token": "bad\nvalue"})
	if w.Code != 400 || tb.Config().Token != "secret-trae-token" {
		t.Fatal("invalid credential changed configuration")
	}
	w = doRequest(t, a.handleTrae, "DELETE", "/admin/api/trae", nil)
	if w.Code != 200 || tb.Config().Token != "" {
		t.Fatal("delete did not hot-apply")
	}
	if token, uid := st.GetTraeCredentials(); token != "" || uid != "" {
		t.Fatal("delete not persisted")
	}
}

func TestTraeNamedAccountsNeverExposeCredentials(t *testing.T) {
	a, st := newAdmin(t)
	b := trae.NewBridge(trae.Config{Token: "current-secret", RefreshToken: "current-refresh", ClientID: "en1oxy7wnw8j9n"})
	a.SetTraeBridge(b)
	post := func(v map[string]string) *httptest.ResponseRecorder {
		return doRequest(t, a.handleTraeAccounts, "POST", "/admin/api/trae/accounts", v)
	}
	w := post(map[string]string{"action": "save", "name": "主账号"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	_, first := st.ListTraeAccounts()
	w = post(map[string]string{"action": "save", "name": "备用账号", "token": "backup-secret", "refresh_token": "backup-refresh", "client_id": "ono9krqynydwx5"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	_, backup := st.ListTraeAccounts()
	if b.Config().Token != "backup-secret" {
		t.Fatal("save did not activate account")
	}
	w = doRequest(t, a.handleTraeAccounts, "GET", "/admin/api/trae/accounts", nil)
	for _, secret := range []string{"current-secret", "current-refresh", "backup-secret", "backup-refresh"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("profile API exposed credentials")
		}
	}
	// Renaming an inactive profile without a new token must preserve that profile.
	w = post(map[string]string{"action": "save", "id": first, "name": "重命名主账号"})
	if w.Code != 200 || b.Config().Token != "current-secret" {
		t.Fatal("inactive profile was overwritten", w.Body.String())
	}
	w = post(map[string]string{"action": "activate", "id": backup})
	if w.Code != 200 || b.Config().Token != "backup-secret" {
		t.Fatal(w.Body.String())
	}
	w = doRequest(t, a.handleTraeAccounts, "DELETE", "/admin/api/trae/accounts?id="+backup, nil)
	if w.Code != 200 || b.Config().Token != "" {
		t.Fatal("deleting active profile did not clear runtime")
	}
	w = post(map[string]string{"action": "activate", "id": "missing"})
	if w.Code != 404 {
		t.Fatal("missing profile accepted", w.Code)
	}
	mux := http.NewServeMux()
	a.RegisterRoutes(mux)
	r := httptest.NewRequest("GET", "/admin/api/trae/accounts", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("profile API missing authentication")
	}
}

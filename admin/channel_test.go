package admin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"worker-cn-2api/workbuddy"
)

func TestChannelConnectionFreshAndRedacted(t *testing.T) {
	a, _ := newAdmin(t)
	var reject atomic.Bool
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if reject.Load() {
			w.WriteHeader(401)
			fmt.Fprint(w, "rejected secret-wb")
			return
		}
		fmt.Fprint(w, `{"models":[{"id":"glm","name":"GLM"}]}`)
	}))
	defer upstream.Close()
	a.SetWorkBuddyBridge(workbuddy.NewBridge(&workbuddy.Config{Token: "secret-wb", Endpoint: upstream.URL}))
	for i := 0; i < 2; i++ {
		w := doRequest(t, a.handleChannelTest, "POST", "/admin/api/channel/test", map[string]string{"provider": "workbuddy"})
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if calls.Load() != 2 {
		t.Fatal("explicit test reused cached catalog")
	}
	reject.Store(true)
	w := doRequest(t, a.handleChannelTest, "POST", "/admin/api/channel/test", map[string]string{"provider": "workbuddy"})
	if w.Code != 502 || strings.Contains(w.Body.String(), "secret-wb") {
		t.Fatal("failed or exposed credential", w.Code, w.Body.String())
	}
	mux := http.NewServeMux()
	a.RegisterRoutes(mux)
	unauth := httptest.NewRecorder()
	mux.ServeHTTP(unauth, httptest.NewRequest("POST", "/admin/api/channel/test", strings.NewReader(`{"provider":"workbuddy"}`)))
	if unauth.Code != 401 {
		t.Fatal("unprotected test endpoint")
	}
	w = doRequest(t, a.handleChannelTest, "GET", "/admin/api/channel/test", nil)
	if w.Code != 405 {
		t.Fatal("test must require POST")
	}
	w = doRequest(t, a.handleChannelTest, "POST", "/admin/api/channel/test", map[string]string{"provider": "other"})
	if w.Code != 400 {
		t.Fatal("unknown provider accepted")
	}
}

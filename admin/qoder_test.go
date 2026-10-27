package admin

import (
	"strings"
	"testing"
	"worker-cn-2api/qoder"
)

func TestQoderCredentialsSaveMaskAndClear(t *testing.T) {
	a, s := newAdmin(t)
	p := qoder.NewProvider("")
	a.SetQoderProvider(p)
	w := doRequest(t, a.handleQoder, "POST", "/admin/api/qoder", map[string]string{"pat": "qoder-private-pat"})
	if w.Code != 200 || !p.Configured() || s.GetQoderPAT() != "qoder-private-pat" {
		t.Fatal("Qoder not activated")
	}
	w = doRequest(t, a.handleQoder, "GET", "/admin/api/qoder", nil)
	if strings.Contains(w.Body.String(), "qoder-private-pat") || !strings.Contains(w.Body.String(), `"configured":true`) {
		t.Fatal("Qoder status exposed credentials")
	}
	w = doRequest(t, a.handleQoder, "DELETE", "/admin/api/qoder", nil)
	if w.Code != 200 || p.Configured() || s.GetQoderPAT() != "" {
		t.Fatal("Qoder not cleared")
	}
}

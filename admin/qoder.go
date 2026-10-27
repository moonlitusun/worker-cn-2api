package admin

import (
	"net/http"
	"strings"
)

func (a *Admin) handleQoder(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		configured := a.store.GetQoderPAT() != ""
		if a.qoderProvider != nil {
			configured = a.qoderProvider.Configured()
		}
		masked := ""
		if configured {
			masked = "********"
		}
		writeJSON(w, 200, map[string]any{"configured": configured, "token_masked": masked, "pat_masked": masked})
	case http.MethodPost:
		var body struct {
			PAT   string `json:"pat"`
			Token string `json:"token"`
		}
		if err := readJSON(w, r, &body); err != nil {
			writeJSONError(w, 400, "请求格式错误")
			return
		}
		pat := strings.TrimSpace(body.PAT)
		if pat == "" {
			pat = strings.TrimSpace(body.Token)
		}
		if pat == "" || strings.ContainsAny(pat, " \t\r\n") {
			writeJSONError(w, 400, "请输入有效 PAT")
			return
		}
		if err := a.store.SetQoderPAT(pat); err != nil {
			writeJSONError(w, 500, "保存失败")
			return
		}
		if a.qoderProvider != nil {
			a.qoderProvider.UpdatePAT(pat)
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	case http.MethodDelete:
		if err := a.store.SetQoderPAT(""); err != nil {
			writeJSONError(w, 500, "保存失败")
			return
		}
		if a.qoderProvider != nil {
			a.qoderProvider.UpdatePAT("")
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		writeJSONError(w, 405, "方法不允许")
	}
}

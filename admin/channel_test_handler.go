package admin

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// Explicit connection tests verify only the requested channel, never generate chat.
func (a *Admin) handleChannelTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	var body struct {
		Provider string `json:"provider"`
	}
	if readJSON(w, r, &body) != nil {
		writeJSONError(w, 400, "请求格式错误")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	count := 0
	var err error
	secrets := []string{}
	switch body.Provider {
	case "qoder":
		if a.qoderProvider == nil || !a.qoderProvider.Configured() {
			writeJSONError(w, 503, "请先配置 Qoder 凭据")
			return
		}
		models, e := a.qoderProvider.TestModels(ctx)
		count, err = len(models), e
	case "workbuddy":
		if a.wbBridge == nil || a.wbBridge.Config().Token == "" {
			writeJSONError(w, 503, "请先配置 WorkBuddy 凭据")
			return
		}
		secrets = append(secrets, a.wbBridge.Config().Token)
		models, e := a.wbBridge.TestModels(ctx)
		count, err = len(models), e
		secrets = append(secrets, a.wbBridge.Config().Token)
	default:
		writeJSONError(w, 400, "未知渠道")
		return
	}
	if err != nil {
		message := err.Error()
		for _, secret := range secrets {
			if secret != "" {
				message = strings.ReplaceAll(message, secret, "[redacted]")
			}
		}
		writeJSONError(w, 502, message)
		return
	}
	if count == 0 {
		writeJSONError(w, 502, "未返回可用模型，请检查账号权限")
		return
	}
	writeJSON(w, 200, map[string]any{"model_count": count})
}

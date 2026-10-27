package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"worker-cn-2api/store"
	"worker-cn-2api/trae"
)

func (a *Admin) handleTraeAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		accounts, active := a.store.ListTraeAccounts()
		entries := []map[string]any{}
		for _, account := range accounts {
			s := account.Session
			entries = append(entries, map[string]any{"id": account.ID, "name": account.Name, "active": account.ID == active, "configured": s.Token != "", "refresh_available": s.RefreshToken != "", "expires_at": s.ExpiresAt, "client_id": s.ClientID})
		}
		writeJSON(w, 200, map[string]any{"accounts": entries, "active_id": active})
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	if a.traeBridge == nil {
		writeJSONError(w, 503, "Trae 渠道未初始化")
		return
	}
	var body struct {
		Action       string `json:"action"`
		ID           string `json:"id"`
		Name         string `json:"name"`
		Token        string `json:"token"`
		UserID       string `json:"user_id"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    string `json:"expires_at"`
		ClientID     string `json:"client_id"`
	}
	if r.Method == http.MethodDelete {
		body.Action = "delete"
		body.ID = r.URL.Query().Get("id")
	} else if err := readJSON(w, r, &body); err != nil {
		writeJSONError(w, 400, "请求格式错误")
		return
	}
	if body.Action != "save" && body.Action != "activate" && body.Action != "delete" {
		writeJSONError(w, 400, "不支持的账号操作")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Action == "save" && (body.Name == "" || len([]rune(body.Name)) > 80) {
		writeJSONError(w, 400, "账号名称需为 1–80 个字符")
		return
	}
	if body.Action != "save" && body.ID == "" {
		writeJSONError(w, 400, "请选择账号")
		return
	}
	status := 500
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	err := a.traeBridge.ChangeAccount(ctx, func(c trae.Config) (trae.Config, error) {
		s := store.TraeSession{Token: c.Token, UserID: c.UserID, RefreshToken: c.RefreshToken, ExpiresAt: c.ExpiresAt, ClientID: c.ClientID}
		if body.Action == "save" {
			status = 400
			if body.ID != "" && body.Token == "" {
				accounts, active := a.store.ListTraeAccounts()
				if body.ID != active {
					for _, profile := range accounts {
						if profile.ID == body.ID {
							s = profile.Session
							break
						}
					}
				}
			}
			if body.Token != "" {
				s = store.TraeSession{Token: strings.TrimSpace(body.Token), UserID: strings.TrimSpace(body.UserID), RefreshToken: strings.TrimSpace(body.RefreshToken), ExpiresAt: strings.TrimSpace(body.ExpiresAt), ClientID: body.ClientID}
			}
			if s.Token == "" || strings.ContainsAny(s.Token, " \t\r\n") || strings.ContainsAny(s.UserID, "\r\n") || strings.ContainsAny(s.RefreshToken, " \t\r\n") {
				return c, fmt.Errorf("请配置有效账号凭据")
			}
			if s.ClientID == "" {
				s.ClientID = "ono9krqynydwx5"
			}
			if s.ClientID != "ono9krqynydwx5" && s.ClientID != "en1oxy7wnw8j9n" {
				return c, fmt.Errorf("请选择有效的中国区产品类型")
			}
			if s.ExpiresAt != "" {
				if _, err := time.Parse(time.RFC3339Nano, s.ExpiresAt); err != nil {
					return c, fmt.Errorf("过期时间需为 ISO 8601 格式")
				}
			}
		}
		status = 500
		next, err := a.store.ChangeTraeAccount(body.Action, body.ID, body.Name, s)
		if err != nil {
			return c, err
		}
		c.Token = next.Token
		c.UserID = next.UserID
		c.RefreshToken = next.RefreshToken
		c.ExpiresAt = next.ExpiresAt
		if next.ClientID != "" {
			c.ClientID = next.ClientID
		}
		return c, nil
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeJSONError(w, 504, "账号操作超时，请稍后重试")
			return
		}
		if errors.Is(err, store.ErrTraeAccountNotFound) {
			status = 404
		}
		writeJSONError(w, status, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

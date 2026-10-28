package trae

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// enterpriseCredentials runs under refreshGate. Persist the original CLI PAT,
// never replace it with the short-lived JWT; restart and renewal re-exchange it.
func (b *Bridge) enterpriseCredentials(ctx context.Context, source Config) (result Config, retErr error) {
	b.mu.RLock()
	if b.enterpriseSession != nil && b.enterpriseSource == source && time.Until(tokenExpiry(*b.enterpriseSession)) > time.Minute {
		c := *b.enterpriseSession
		b.mu.RUnlock()
		return c, nil
	}
	b.mu.RUnlock()
	o := b.observe()
	defer func() { b.record("refresh", o, retErr) }()
	u, err := url.Parse(source.EnterpriseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return source, bad("invalid Trae enterprise URL")
	}
	body, _ := json.Marshal(map[string]string{"RefreshToken": source.Token, "ClientID": EnterpriseClientID})
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, source.EnterpriseURL+"/cloudide/api/v3/trae/oauth/ExchangeToken", bytes.NewReader(body))
	if err != nil {
		return source, bad("invalid Trae enterprise URL")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "traecli/0.211.1")
	resp, err := b.client.Do(req)
	if err != nil {
		return source, fmt.Errorf("Trae enterprise login failed: %s", scrub(err.Error(), source))
	}
	defer resp.Body.Close()
	// Do not include the upstream response body in errors: it may contain credentials.
	if resp.StatusCode != http.StatusOK {
		return source, &Error{Status: 502, Message: fmt.Sprintf("Trae 企业版登录失败（HTTP %d），请检查 CLI 登录令牌是否有效", resp.StatusCode)}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return source, fmt.Errorf("Trae enterprise login response failed")
	}
	var response struct {
		Code int `json:"code"`
		Data struct {
			Token               string `json:"Token"`
			TokenExpireAt       int64  `json:"TokenExpireAt"`
			TokenExpireDuration int64  `json:"TokenExpireDuration"`
		} `json:"Data"`
	}
	if json.Unmarshal(raw, &response) != nil || response.Code != 0 || strings.TrimSpace(response.Data.Token) == "" {
		return source, &Error{Status: 401, Message: "Trae 企业版登录令牌无效或已过期，请重新生成 CLI 登录令牌"}
	}
	next := source
	next.Token = response.Data.Token
	next.RefreshToken = ""
	next.ExpiresAt = ""
	next.BaseURL = source.EnterpriseURL
	next.AuthURL = source.EnterpriseURL
	next.ClientID = EnterpriseClientID
	n := response.Data.TokenExpireAt
	if n > 1e12 {
		n /= 1000
	}
	expiry := time.Time{}
	if n > 0 {
		expiry = time.Unix(n, 0)
	} else if duration := response.Data.TokenExpireDuration; duration > 0 && duration < 365*86400 {
		expiry = time.Now().Add(time.Duration(duration) * time.Second)
	} else {
		expiry = tokenExpiry(next)
	}
	if expiry.IsZero() || !expiry.After(time.Now()) {
		return source, fmt.Errorf("Trae enterprise login returned no valid expiry")
	}
	next.ExpiresAt = expiry.UTC().Format(time.RFC3339)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cfg != source {
		return source, bad("Trae 凭据已发生变化，请重试")
	}
	b.enterpriseSource = source
	b.enterpriseSession = &next
	return next, nil
}

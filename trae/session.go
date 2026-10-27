package trae

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

func tokenExpiry(c Config) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, c.ExpiresAt); err == nil {
		return t
	}
	parts := strings.Split(c.Token, ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(data, &claims) == nil && claims.Exp > 0 {
		return time.Unix(claims.Exp, 0)
	}
	return time.Time{}
}

// Refresh once per account snapshot. Admin updates win over in-flight refreshes.
func (b *Bridge) ensureCredentials(ctx context.Context) (result Config, retErr error) {
	select {
	case b.refreshGate <- struct{}{}:
	case <-ctx.Done():
		return Config{}, ctx.Err()
	}
	defer func() { <-b.refreshGate }()
	b.mu.Lock()
	if b.unsavedSession && b.credentialSaver != nil {
		if err := b.credentialSaver(b.cfg); err != nil {
			c := b.cfg
			b.mu.Unlock()
			return c, fmt.Errorf("Trae session could not be saved")
		}
		b.unsavedSession = false
	}
	c := b.cfg
	b.mu.Unlock()
	expiry := tokenExpiry(c)
	if c.RefreshToken == "" || expiry.IsZero() || time.Until(expiry) > time.Minute {
		return c, nil
	}
	o := b.observe()
	defer func() { b.record("refresh", o, retErr) }()
	if c.ClientID != "ono9krqynydwx5" && c.ClientID != "en1oxy7wnw8j9n" {
		return c, bad("unsupported Trae CN ClientID")
	}
	authCfg := c
	authCfg.BaseURL = strings.TrimRight(c.AuthURL, "/")
	resp, err := b.request(ctx, authCfg, "/cloudide/api/v3/trae/oauth/ExchangeToken", map[string]any{"ClientID": c.ClientID, "RefreshToken": c.RefreshToken, "ClientSecret": "-", "UserID": ""}, false)
	if err != nil {
		return c, err
	}
	defer resp.Body.Close()
	// The session response must complete within the same bounded header budget.
	timer := time.AfterFunc(30*time.Second, func() { resp.Body.Close() })
	defer timer.Stop()
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return c, fmt.Errorf("Trae token refresh response failed")
	}
	if len(data) > 1<<20 {
		return c, fmt.Errorf("Trae token refresh response too large")
	}
	var response struct {
		Result struct {
			Token               string `json:"Token"`
			RefreshToken        string `json:"RefreshToken"`
			TokenExpireAt       int64  `json:"TokenExpireAt"`
			TokenExpireDuration int64  `json:"TokenExpireDuration"`
		} `json:"Result"`
	}
	if json.Unmarshal(data, &response) != nil || response.Result.Token == "" {
		return c, &Error{Status: 401, Message: "token refresh failed; sign in to Trae CN again"}
	}
	next := c
	next.Token = response.Result.Token
	if response.Result.RefreshToken != "" {
		next.RefreshToken = response.Result.RefreshToken
	}
	n := response.Result.TokenExpireAt
	if n > 1e12 {
		n /= 1000
	}
	expires := time.Unix(n, 0)
	if n <= 0 && response.Result.TokenExpireDuration > 0 && response.Result.TokenExpireDuration < 365*86400 {
		expires = time.Now().Add(time.Duration(response.Result.TokenExpireDuration) * time.Second)
	}
	if n <= 0 && response.Result.TokenExpireDuration <= 0 {
		next.ExpiresAt = ""
		expires = tokenExpiry(next)
	}
	if expires.IsZero() || !expires.After(time.Now()) {
		return c, fmt.Errorf("Trae refresh returned no valid expiry")
	}
	next.ExpiresAt = expires.UTC().Format(time.RFC3339)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cfg != c {
		return b.cfg, nil
	}
	// Retain the rotated session in memory even if disk fails: retrying the old
	// refresh token could lose the newly issued session.
	b.cfg = next
	if b.credentialSaver != nil {
		if err := b.credentialSaver(next); err != nil {
			b.unsavedSession = true
			return next, fmt.Errorf("Trae session renewed but could not be saved")
		}
	}
	return next, nil
}

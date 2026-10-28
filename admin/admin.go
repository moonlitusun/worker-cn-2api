// Package admin provides WebUI management for the API bridge:
// WorkBuddy CN token configuration, API key CRUD, model listing, and password-protected access.
package admin

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"worker-cn-2api/auth"
	"worker-cn-2api/qoder"
	"worker-cn-2api/stats"
	"worker-cn-2api/store"
	"worker-cn-2api/trae"
	"worker-cn-2api/workbuddy"
)

const sessionDuration = 24 * time.Hour

// adminMaxBody caps JSON request bodies handled by the admin API.
const adminMaxBody = 1 << 16 // 64 KiB

// Admin manages the web UI and admin API endpoints.
type Admin struct {
	store         *store.Store
	stats         *stats.Recorder
	wbBridge      *workbuddy.Bridge
	traeBridge    *trae.Bridge
	qoderProvider *qoder.Provider
	sessions      map[string]time.Time
	sessionMu     sync.Mutex
	limiter       *loginLimiter
}

// SetWorkBuddyBridge attaches the WorkBuddy bridge for hot updates.
func (a *Admin) SetWorkBuddyBridge(wb *workbuddy.Bridge) {
	a.wbBridge = wb
}

func (a *Admin) SetTraeBridge(tb *trae.Bridge)       { a.traeBridge = tb }
func (a *Admin) SetQoderProvider(qb *qoder.Provider) { a.qoderProvider = qb }

// New creates an Admin instance. rec may be nil.
func New(s *store.Store, rec *stats.Recorder) *Admin {
	return &Admin{
		store:    s,
		stats:    rec,
		sessions: make(map[string]time.Time),
		limiter:  newLoginLimiter(),
	}
}

// RegisterRoutes registers all admin routes on the given mux.
func (a *Admin) RegisterRoutes(mux *http.ServeMux) {
	// WebUI (no auth — the page handles auth client-side)
	mux.HandleFunc("/admin", a.serveIndex)
	mux.HandleFunc("/admin/", a.serveIndex)

	// Auth endpoints (no auth required)
	mux.HandleFunc("/admin/api/login", a.handleLogin)
	mux.HandleFunc("/admin/api/logout", a.handleLogout)
	mux.HandleFunc("/admin/api/auth", a.handleAuth)

	// Protected API endpoints (require auth)
	mux.HandleFunc("/admin/api/keys", a.requireAuth(a.handleKeys))
	mux.HandleFunc("/admin/api/workbuddy", a.requireAuth(a.handleWorkBuddy))
	mux.HandleFunc("/admin/api/trae", a.requireAuth(a.handleTrae))
	mux.HandleFunc("/admin/api/qoder", a.requireAuth(a.handleQoder))
	mux.HandleFunc("/admin/api/pat", a.requireAuth(a.handleQoder))
	mux.HandleFunc("/admin/api/channel/test", a.requireAuth(a.handleChannelTest))
	mux.HandleFunc("/admin/api/trae/test", a.requireAuth(a.handleTraeTest))
	mux.HandleFunc("/admin/api/trae/accounts", a.requireAuth(a.handleTraeAccounts))
	mux.HandleFunc("/admin/api/models", a.requireAuth(a.handleModels))
	mux.HandleFunc("/admin/api/stats", a.requireAuth(a.handleStats))
	mux.HandleFunc("/admin/api/config", a.requireAuth(a.handleConfig))
	mux.HandleFunc("/admin/api/password", a.requireAuth(a.handlePassword))
}

// --- Session & Auth ---

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (a *Admin) createSession() (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}
	a.sessionMu.Lock()
	a.sessions[token] = time.Now().Add(sessionDuration)
	// Opportunistic cleanup: remove expired sessions
	now := time.Now()
	for k, exp := range a.sessions {
		if now.After(exp) {
			delete(a.sessions, k)
		}
	}
	a.sessionMu.Unlock()
	return token, nil
}

func (a *Admin) isValidSession(token string) bool {
	if token == "" {
		return false
	}
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()
	expiry, ok := a.sessions[token]
	if !ok || time.Now().After(expiry) {
		delete(a.sessions, token)
		return false
	}
	return true
}

func (a *Admin) removeSession(token string) {
	a.sessionMu.Lock()
	delete(a.sessions, token)
	a.sessionMu.Unlock()
}

func (a *Admin) getPassword() string {
	if env := os.Getenv("WORKER_ADMIN_PASSWORD"); env != "" {
		return env
	}
	return a.store.GetPassword()
}

// --- Login rate limiting (per client IP, exponential backoff) ---

type loginAttempt struct {
	failures    int
	lockedUntil time.Time
	lastFail    time.Time
}

type loginLimiter struct {
	mu    sync.Mutex
	state map[string]*loginAttempt
}

const (
	loginMaxBackoff = 30 * time.Second
	loginPruneAge   = time.Hour
)

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{state: make(map[string]*loginAttempt)}
}

// check reports whether the IP is currently locked out and, if so, how long
// until it may retry.
func (l *loginLimiter) check(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked()
	st := l.state[ip]
	if st == nil {
		return false, 0
	}
	now := time.Now()
	if now.Before(st.lockedUntil) {
		return true, st.lockedUntil.Sub(now)
	}
	return false, 0
}

func (l *loginLimiter) fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	st := l.state[ip]
	if st == nil {
		st = &loginAttempt{}
		l.state[ip] = st
	}
	st.failures++
	st.lastFail = time.Now()
	// Compute the backoff in plain integers and clamp before converting to
	// time.Duration: 1<<63 seconds overflows int64 nanoseconds and yields a
	// negative duration, which would silently disable the lockout after
	// enough consecutive failures.
	shift := st.failures - 1
	if shift > 5 { // 1<<5 s == 32s > loginMaxBackoff
		shift = 5
	}
	backoff := time.Second << uint(shift)
	if backoff > loginMaxBackoff {
		backoff = loginMaxBackoff
	}
	st.lockedUntil = st.lastFail.Add(backoff)
}

func (l *loginLimiter) success(ip string) {
	l.mu.Lock()
	delete(l.state, ip)
	l.mu.Unlock()
}

func (l *loginLimiter) pruneLocked() {
	cutoff := time.Now().Add(-loginPruneAge)
	for ip, st := range l.state {
		if st.lastFail.Before(cutoff) {
			delete(l.state, ip)
		}
	}
}

// --- Request helpers ---

// readJSON decodes a JSON request body capped at adminMaxBody. The size cap
// bounds memory use against oversized payloads.
func readJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, adminMaxBody)
	return json.NewDecoder(r.Body).Decode(v)
}

// passwordsEqual compares two secrets in constant time to avoid timing leaks.
func passwordsEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	// Trim the port, handling both "1.2.3.4:5678" and "[::1]:5678".
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return host
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp == "https" {
		return true
	}
	return false
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		MaxAge:   int(sessionDuration.Seconds()),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		MaxAge:   -1,
	})
}

// requireAuth wraps a handler with session authentication.
func (a *Admin) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("admin_session")
		if err != nil || !a.isValidSession(cookie.Value) {
			writeJSONError(w, 401, "未登录或会话已过期")
			return
		}
		next(w, r)
	}
}

func (a *Admin) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	ip := clientIP(r)
	if locked, retry := a.limiter.check(ip); locked {
		w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
		writeJSONError(w, 429, "登录尝试过于频繁，请稍后再试")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeJSONError(w, 400, "请求格式错误")
		return
	}
	if !passwordsEqual(body.Password, a.getPassword()) {
		a.limiter.fail(ip)
		writeJSONError(w, 401, "密码错误")
		return
	}
	a.limiter.success(ip)
	token, err := a.createSession()
	if err != nil {
		writeJSONError(w, 500, "无法创建会话")
		return
	}
	setSessionCookie(w, r, token)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *Admin) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	if cookie, err := r.Cookie("admin_session"); err == nil {
		a.removeSession(cookie.Value)
	}
	clearSessionCookie(w, r)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (a *Admin) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	cookie, err := r.Cookie("admin_session")
	authed := err == nil && a.isValidSession(cookie.Value)
	writeJSON(w, 200, map[string]interface{}{"authenticated": authed})
}

func (a *Admin) handlePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeJSONError(w, 400, "请求格式错误")
		return
	}
	if os.Getenv("WORKER_ADMIN_PASSWORD") != "" {
		writeJSONError(w, 400, "密码由环境变量 WORKER_ADMIN_PASSWORD 管理，无法在面板中修改")
		return
	}
	if !passwordsEqual(body.CurrentPassword, a.getPassword()) {
		writeJSONError(w, 401, "当前密码错误")
		return
	}
	if len(body.NewPassword) < 1 {
		writeJSONError(w, 400, "新密码不能为空")
		return
	}
	if err := a.store.SetPassword(body.NewPassword); err != nil {
		writeJSONError(w, 500, err.Error())
		return
	}
	// Invalidate every existing session so a (possibly stolen) cookie from
	// before the password change cannot be replayed.
	a.sessionMu.Lock()
	a.sessions = make(map[string]time.Time)
	a.sessionMu.Unlock()
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// --- HTTP Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// --- API Key endpoints ---

func (a *Admin) handleKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys := a.store.ListKeys()
		writeJSON(w, 200, map[string]interface{}{"keys": keys})
	case http.MethodPost:
		var body struct {
			Key  string `json:"key"`
			Note string `json:"note"`
		}
		if err := readJSON(w, r, &body); err != nil {
			writeJSONError(w, 400, "请求格式错误")
			return
		}
		entry, err := a.store.AddKey(body.Key, body.Note)
		if err != nil {
			if errors.Is(err, store.ErrDuplicateKey) {
				writeJSONError(w, 409, err.Error())
			} else {
				writeJSONError(w, 500, err.Error())
			}
			return
		}
		writeJSON(w, 201, entry)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSONError(w, 400, "缺少 id 参数")
			return
		}
		if err := a.store.DeleteKey(id); err != nil {
			writeJSONError(w, 404, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"status": "deleted"})
	default:
		writeJSONError(w, 405, "方法不允许")
	}
}

// --- Config endpoint ---

// timeoutEnvManaged reports whether stream timeouts are pinned by env vars,
// in which case the panel must not override them (same policy as
// WORKER_ADMIN_PASSWORD for password changes).
func timeoutEnvManaged() bool {
	return os.Getenv("WORKER_CHAT_TIMEOUT_SECONDS") != "" ||
		os.Getenv("WORKER_IDLE_TIMEOUT_SECONDS") != ""
}

func (a *Admin) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Report the effective stream timeouts (env startup overrides are
		// already folded into them), not just the persisted file values.
		timeouts := auth.CurrentStreamTimeouts()
		writeJSON(w, 200, map[string]interface{}{
			"host":                 a.store.GetHost(),
			"port":                 a.store.GetPort(),
			"chat_timeout_seconds": int(timeouts.Header.Seconds()),
			"idle_timeout_seconds": int(timeouts.Idle.Seconds()),
		})
	case http.MethodPost:
		var body struct {
			Host               string `json:"host"`
			Port               int    `json:"port"`
			ChatTimeoutSeconds int    `json:"chat_timeout_seconds"`
			IdleTimeoutSeconds int    `json:"idle_timeout_seconds"`
		}
		if err := readJSON(w, r, &body); err != nil {
			writeJSONError(w, 400, "请求格式错误")
			return
		}
		if body.Host == "" {
			body.Host = store.DefaultHost
		}
		if body.Port == 0 {
			body.Port = store.DefaultPort
		}
		if body.Port < 1 || body.Port > 65535 {
			writeJSONError(w, 400, "端口必须在 1-65535 范围内")
			return
		}
		// Timeout fields are optional (0 = keep current) so legacy callers
		// that only send host/port keep working. Validate everything before
		// saving anything so a bad value cannot leave a partial update.
		wantsTimeouts := body.ChatTimeoutSeconds != 0 || body.IdleTimeoutSeconds != 0
		if wantsTimeouts && timeoutEnvManaged() {
			writeJSONError(w, 400, "超时由环境变量 WORKER_CHAT_TIMEOUT_SECONDS / WORKER_IDLE_TIMEOUT_SECONDS 管理，无法在面板中修改")
			return
		}
		if body.ChatTimeoutSeconds != 0 {
			if err := store.ValidateTimeoutSeconds(body.ChatTimeoutSeconds, store.MaxChatTimeoutSeconds); err != nil {
				writeJSONError(w, 400, "Chat 响应超时："+err.Error())
				return
			}
		}
		if body.IdleTimeoutSeconds != 0 {
			if err := store.ValidateTimeoutSeconds(body.IdleTimeoutSeconds, store.MaxIdleTimeoutSeconds); err != nil {
				writeJSONError(w, 400, "流空闲超时："+err.Error())
				return
			}
		}
		if err := a.store.SetHostPort(body.Host, body.Port); err != nil {
			writeJSONError(w, 500, err.Error())
			return
		}
		if body.ChatTimeoutSeconds != 0 {
			if err := a.store.SetChatTimeoutSeconds(body.ChatTimeoutSeconds); err != nil {
				writeJSONError(w, 500, err.Error())
				return
			}
		}
		if body.IdleTimeoutSeconds != 0 {
			if err := a.store.SetIdleTimeoutSeconds(body.IdleTimeoutSeconds); err != nil {
				writeJSONError(w, 500, err.Error())
				return
			}
		}
		// Hot-apply only when the request actually carried timeout fields:
		// re-applying store values unconditionally would clobber env-pinned
		// timeouts on a legacy host/port-only save.
		if wantsTimeouts {
			auth.SetStreamTimeouts(
				time.Duration(a.store.GetChatTimeoutSeconds())*time.Second,
				time.Duration(a.store.GetIdleTimeoutSeconds())*time.Second,
			)
		}
		timeouts := auth.CurrentStreamTimeouts()
		writeJSON(w, 200, map[string]interface{}{
			"host":                 body.Host,
			"port":                 body.Port,
			"chat_timeout_seconds": int(timeouts.Header.Seconds()),
			"idle_timeout_seconds": int(timeouts.Idle.Seconds()),
			"restart":              "修改主机或端口后需要重启服务才能生效（超时配置已即时生效）",
		})
	default:
		writeJSONError(w, 405, "方法不允许")
	}
}

// --- WorkBuddy endpoint ---

func (a *Admin) handleWorkBuddy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		token := a.store.GetWorkBuddyToken()
		if token == "" && a.wbBridge != nil {
			token = a.wbBridge.Config().Token
		}
		configured := token != ""
		masked := ""
		if configured {
			if len(token) <= 8 {
				masked = "********"
			} else {
				masked = token[:4] + "...." + token[len(token)-4:]
			}
		}
		writeJSON(w, 200, map[string]interface{}{
			"configured":   configured,
			"token_masked": masked,
			"provider":     "workbuddy-cn",
		})
	case http.MethodPost:
		var body struct {
			Token string `json:"token"`
		}
		if err := readJSON(w, r, &body); err != nil {
			writeJSONError(w, 400, "请求格式错误")
			return
		}
		token := body.Token
		if err := a.store.SetWorkBuddyToken(token); err != nil {
			writeJSONError(w, 500, err.Error())
			return
		}
		if a.wbBridge != nil {
			a.wbBridge.UpdateToken(token)
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	case http.MethodDelete:
		if err := a.store.SetWorkBuddyToken(""); err != nil {
			writeJSONError(w, 500, err.Error())
			return
		}
		if a.wbBridge != nil {
			a.wbBridge.UpdateToken("")
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		writeJSONError(w, 405, "方法不允许")
	}
}

// --- Models endpoint ---

func (a *Admin) handleTrae(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		token, userID := a.store.GetTraeCredentials()
		expiresAt, clientID := "", ""
		if a.traeBridge != nil {
			cfg := a.traeBridge.Config()
			token, userID = cfg.Token, cfg.UserID
			expiresAt, clientID = cfg.ExpiresAt, cfg.ClientID
		}
		masked := ""
		if token != "" {
			masked = "********"
		}
		status := trae.Diagnostics{Configured: token != ""}
		if a.traeBridge != nil {
			status = a.traeBridge.Status()
		}
		writeJSON(w, 200, map[string]any{"configured": token != "", "token_masked": masked, "user_id": userID, "provider": "trae-cn", "expires_at": expiresAt, "client_id": clientID, "status": status})
	case http.MethodPost:
		var body struct {
			Token        string `json:"token"`
			UserID       string `json:"user_id"`
			RefreshToken string `json:"refresh_token"`
			ExpiresAt    string `json:"expires_at"`
			ClientID     string `json:"client_id"`
		}
		if err := readJSON(w, r, &body); err != nil {
			writeJSONError(w, 400, "请求格式错误")
			return
		}
		body.Token = strings.TrimSpace(body.Token)
		for _, prefix := range []string{"Cloud-IDE-JWT ", "Bearer "} {
			if len(body.Token) >= len(prefix) && strings.EqualFold(body.Token[:len(prefix)], prefix) {
				body.Token = strings.TrimSpace(body.Token[len(prefix):])
				break
			}
		}
		body.UserID = strings.TrimSpace(body.UserID)
		if body.Token == "" || strings.ContainsAny(body.Token, " \t\r\n") || strings.ContainsAny(body.UserID, "\r\n") {
			writeJSONError(w, 400, "请输入有效的 Trae 登录令牌（JWT 或 trae-lt-）")
			return
		}
		if body.ClientID != "" && body.ClientID != "ono9krqynydwx5" && body.ClientID != "en1oxy7wnw8j9n" {
			writeJSONError(w, 400, "请选择有效的中国区产品类型")
			return
		}
		if body.ExpiresAt != "" {
			if _, err := time.Parse(time.RFC3339Nano, body.ExpiresAt); err != nil {
				writeJSONError(w, 400, "令牌过期时间需为 ISO 8601 格式")
				return
			}
		}
		if strings.ContainsAny(body.RefreshToken, " \t\r\n") {
			writeJSONError(w, 400, "刷新令牌格式错误")
			return
		}
		if body.ClientID == "" {
			body.ClientID = "ono9krqynydwx5"
		}
		save := func() error {
			return a.store.SetTraeSession(store.TraeSession{Token: body.Token, UserID: body.UserID, RefreshToken: body.RefreshToken, ExpiresAt: body.ExpiresAt, ClientID: body.ClientID})
		}
		var err error
		if a.traeBridge != nil {
			err = a.traeBridge.ReplaceSession(body.Token, body.UserID, body.RefreshToken, body.ExpiresAt, body.ClientID, save)
		} else {
			err = save()
		}
		if err != nil {
			writeJSONError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	case http.MethodDelete:
		save := func() error { return a.store.SetTraeCredentials("", "") }
		var err error
		if a.traeBridge != nil {
			err = a.traeBridge.ReplaceSession("", "", "", "", "", save)
		} else {
			err = save()
		}
		if err != nil {
			writeJSONError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		writeJSONError(w, 405, "方法不允许")
	}
}

func (a *Admin) handleTraeTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	if a.traeBridge == nil {
		writeJSONError(w, 503, "Trae 渠道未初始化")
		return
	}
	var body struct {
		Model string `json:"model"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeJSONError(w, 400, "请求格式错误")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	result, err := a.traeBridge.Probe(ctx, strings.TrimPrefix(strings.TrimSpace(body.Model), "trae/"))
	if err != nil {
		status := a.traeBridge.Status()
		message := status.LastTest.Error
		if message == "" {
			message = "测试期间账号已发生变化，请重试"
		}
		writeJSON(w, 502, map[string]any{"error": message, "status": status})
		return
	}
	writeJSON(w, 200, result)
}

func (a *Admin) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	type modelEntry struct {
		Name              string `json:"name"`
		Provider          string `json:"provider"`
		DisplayName       string `json:"display_name,omitempty"`
		ContextWindow     int64  `json:"context_window,omitempty"`
		MaxOutputTokens   int64  `json:"max_output_tokens,omitempty"`
		SupportsTools     *bool  `json:"supports_tools,omitempty"`
		SupportsReasoning *bool  `json:"supports_reasoning,omitempty"`
		Channel           string `json:"channel,omitempty"`
	}

	entries := []modelEntry{}
	ids := []string{}
	providerErrors := map[string]string{}
	if a.qoderProvider != nil && a.qoderProvider.Configured() {
		catalog, err := a.qoderProvider.Models(r.Context())
		if err != nil {
			providerErrors["qoder-cn"] = err.Error()
		} else {
			for _, id := range catalog {
				entries = append(entries, modelEntry{Name: "qoder/" + id, DisplayName: id, Provider: "qoder-cn"})
				ids = append(ids, id)
			}
		}
	}
	if a.wbBridge != nil && a.wbBridge.Config().Token != "" {
		initialToken := a.wbBridge.Config().Token
		catalog, err := a.wbBridge.Models(r.Context())
		if err != nil {
			message := err.Error()
			for _, token := range []string{initialToken, a.wbBridge.Config().Token} {
				if token != "" {
					message = strings.ReplaceAll(message, token, "[redacted]")
				}
			}
			providerErrors["workbuddy-cn"] = message
		} else {
			for _, m := range catalog {
				entries = append(entries, modelEntry{Name: "workbuddy/" + m.ID, Provider: "workbuddy-cn", DisplayName: m.Name, ContextWindow: int64(m.MaxInputTokens), MaxOutputTokens: int64(m.MaxOutputTokens), SupportsTools: m.SupportsToolCall})
				ids = append(ids, m.ID)
			}
		}
	}
	if a.traeBridge != nil && a.traeBridge.Config().Token != "" {
		catalog, err := a.traeBridge.Models(r.Context())
		if err != nil {
			providerErrors["trae-cn"] = a.traeBridge.Status().LastCatalog.Error
		} else {
			for _, m := range catalog {
				id := "trae/" + m.ID
				entries = append(entries, modelEntry{Name: id, Provider: "trae-cn", DisplayName: m.Name, ContextWindow: m.ContextWindow, Channel: m.Channel})
				ids = append(ids, id)
			}
		}
	}
	writeJSON(w, 200, map[string]interface{}{"models": entries, "raw_ids": ids, "provider_errors": providerErrors})
}

// --- Stats endpoint ---

func (a *Admin) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, 405, "方法不允许")
		return
	}
	if a.stats == nil {
		writeJSON(w, 200, stats.Report{})
		return
	}
	provider := r.URL.Query().Get("provider")
	if provider != "" && provider != "qoder" && provider != "workbuddy" && provider != "trae" {
		writeJSONError(w, 400, "未知渠道")
		return
	}
	writeJSON(w, 200, a.stats.Report(provider))
}

// --- WebUI ---

func (a *Admin) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin" && r.URL.Path != "/admin/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(200)
	w.Write([]byte(indexHTML))
}

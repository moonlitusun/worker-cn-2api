// worker-cn-2api exposes WorkBuddy CN and Trae CN through an OpenAI-compatible API.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"worker-cn-2api/admin"
	"worker-cn-2api/auth"
	"worker-cn-2api/qoder"
	"worker-cn-2api/stats"
	"worker-cn-2api/store"
	"worker-cn-2api/trae"
	"worker-cn-2api/workbuddy"

	"time"
)

func newWorkBuddyBridge(st *store.Store) *workbuddy.Bridge {
	return workbuddy.NewBridge(workbuddy.ResolveConfig(st.GetWorkBuddyToken()))
}

// healthHandler serves an unauthenticated liveness/readiness probe. It is
// deliberately cheap (no upstream calls, no session bootstrap): the body
// carries readiness details (WorkBuddy CN token configured, effective stream timeouts)
// while the status stays 200 as long as the process is serving.
func healthHandler(st *store.Store, wbBridge *workbuddy.Bridge, providers ...any) http.HandlerFunc {
	tb, qb := channelProviders(providers)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		timeouts := auth.CurrentStreamTimeouts()
		hasTrae := tb != nil && tb.Config().Token != ""
		w.Header().Set("Content-Type", "application/json")
		hasWB := false
		if wbBridge != nil && wbBridge.Config().Token != "" {
			hasWB = true
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":               "ok",
			"has_workbuddy":        hasWB,
			"has_trae":             hasTrae,
			"has_qoder":            qb != nil && qb.Configured(),
			"chat_timeout_seconds": int(timeouts.Header.Seconds()),
			"idle_timeout_seconds": int(timeouts.Idle.Seconds()),
		})
	}
}

func getEnv(key string) string { return os.Getenv(key) }

func main() {
	// Initialize store
	dataPath := getEnv("WORKER_DATA_PATH")
	if dataPath == "" {
		dataPath = "data.json"
	}
	st, err := store.New(dataPath)
	if err != nil {
		log.Fatalf("[store] failed to init: %v", err)
	}

	// Host/port: env vars override config file; config file overrides defaults
	host := getEnv("WORKER_HOST")
	if host == "" {
		host = st.GetHost()
	}
	port := st.GetPort()
	if envPort := getEnv("WORKER_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		} else {
			log.Printf("[bridge] WARN invalid port=%q; using %d", envPort, port)
		}
	}

	// Chat stream timeouts: env vars override the persisted config, the
	// config file overrides the built-in defaults (same precedence as
	// host/port). Applied to new requests immediately.
	chatTimeout := st.GetChatTimeoutSeconds()
	if v := getEnv("WORKER_CHAT_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= store.MaxChatTimeoutSeconds {
			chatTimeout = n
		} else {
			log.Printf("[bridge] WARN invalid CHAT_TIMEOUT_SECONDS=%q; using %d", v, chatTimeout)
		}
	}
	idleTimeout := st.GetIdleTimeoutSeconds()
	if v := getEnv("WORKER_IDLE_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= store.MaxIdleTimeoutSeconds {
			idleTimeout = n
		} else {
			log.Printf("[bridge] WARN invalid IDLE_TIMEOUT_SECONDS=%q; using %d", v, idleTimeout)
		}
	}
	auth.SetStreamTimeouts(time.Duration(chatTimeout)*time.Second, time.Duration(idleTimeout)*time.Second)

	// Request statistics recorder with periodic persistence (30s cadence).
	// The loop also drains a stop channel so graceful shutdown can trigger a
	// final flush, avoiding loss of the last <=30s of stats.
	rec := stats.NewRecorder(st)
	statsStop := make(chan struct{})
	statsDone := make(chan struct{})
	go func() {
		defer close(statsDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := rec.Flush(); err != nil {
					log.Printf("[stats] WARN flush failed: %v", err)
				}
			case <-statsStop:
				if err := rec.Flush(); err != nil {
					log.Printf("[stats] WARN final flush failed: %v", err)
				}
				return
			}
		}
	}()

	// Initialize WorkBuddy bridge
	wbBridge := newWorkBuddyBridge(st)
	wbConfig := wbBridge.Config()
	if wbConfig.Token != "" {
		log.Printf("[workbuddy] CN backend ready")
	} else {
		log.Printf("[workbuddy] backend standby (no auth token configured)")
	}

	// Initialize admin
	traeToken, traeUserID := st.GetTraeCredentials()
	traeCfg := trae.ResolveConfig(traeToken, traeUserID)
	if saved := st.GetTraeSession(); saved.Token != "" {
		traeCfg.RefreshToken = saved.RefreshToken
		traeCfg.ExpiresAt = saved.ExpiresAt
		if saved.ClientID != "" {
			traeCfg.ClientID = saved.ClientID
		}
	}
	traeBridge := trae.NewBridge(traeCfg)
	traeBridge.SetCredentialSaver(func(c trae.Config) error {
		return st.SetTraeSession(store.TraeSession{Token: c.Token, UserID: c.UserID, RefreshToken: c.RefreshToken, ExpiresAt: c.ExpiresAt, ClientID: c.ClientID})
	})
	adminInst := admin.New(st, rec)
	adminInst.SetWorkBuddyBridge(wbBridge)
	adminInst.SetTraeBridge(traeBridge)
	pat := st.GetQoderPAT()
	if pat == "" {
		pat = os.Getenv("WORKER_QODER_PAT")
	}
	if pat == "" {
		pat = os.Getenv("QODER_PAT")
	}
	qoderProvider := qoder.NewProvider(pat)
	adminInst.SetQoderProvider(qoderProvider)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", makeChatHandler(st, wbBridge, rec, traeBridge, qoderProvider))
	mux.HandleFunc("/v1/models", makeModelsHandler(st, wbBridge, traeBridge, qoderProvider))
	mux.HandleFunc("/health", healthHandler(st, wbBridge, traeBridge, qoderProvider))

	// Root redirect to admin UI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/admin", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// Register admin routes
	adminInst.RegisterRoutes(mux)

	addr := host + ":" + strconv.Itoa(port)
	for _, displayAddr := range startupAddresses(host, port) {
		log.Printf("[bridge] listening http://%s/v1/chat/completions", displayAddr)
		log.Printf("[admin]  http://%s/admin", displayAddr)
	}

	// Timeouts: ReadHeaderTimeout/IdleTimeout harden against slowloris-style
	// resource exhaustion. WriteTimeout is intentionally unset so that
	// long-lived SSE streaming responses are not prematurely cancelled.
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM: stop the stats ticker (final
	// flush) and drain in-flight requests with a bounded deadline.
	shutdownErr := make(chan error, 1)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("[server] received %s, shutting down...", sig)
		close(statsStop)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		shutdownErr <- srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[server] failed to listen on %s: %v\n[server] If port %d is in use, change it in the admin UI or data.json", addr, err, port)
	}
	if err := <-shutdownErr; err != nil {
		log.Printf("[server] shutdown error: %v", err)
	}
	// Wait for the stats goroutine to finish its final flush before exiting.
	<-statsDone
	log.Printf("[server] stopped")
}

package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"qoder2api/auth"
	"qoder2api/bridge"
	"qoder2api/models"
	"qoder2api/requestopts"
	"qoder2api/stats"
	"qoder2api/store"
	"qoder2api/transform"
	"qoder2api/workbuddy"
)

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(authHeader[7:])
		if token != "" {
			return token
		}
	}
	return ""
}

func writeJSONError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    errType,
		},
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(resp)
}

// makeUnifiedChatHandler creates the chat completions handler that dynamically routes
// requests to WorkBuddy CN or Qoder CN based on the requested model name.
func makeUnifiedChatHandler(st *store.Store, qoderProvider *bridgeProvider, wbBridge *workbuddy.Bridge, rec *stats.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeJSONError(w, 401, "invalid_request_error", "Missing Authorization: Bearer <key>")
			return
		}
		if !st.ValidateKey(token) {
			writeJSONError(w, 401, "invalid_request_error", "Invalid API key")
			return
		}

		// Cap body size (10 MiB)
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			writeJSONError(w, 400, "invalid_request_error", "Invalid JSON body: "+err.Error())
			return
		}

		if _, err := requestopts.ReasoningEnabled(reqBody); err != nil {
			writeJSONError(w, 400, "invalid_request_error", err.Error())
			return
		}

		modelParam, _ := reqBody["model"].(string)
		modelNorm := strings.TrimSpace(modelParam)

		// Dispatcher: Route to WorkBuddy if model belongs to WorkBuddy catalog
		if workbuddy.IsWorkbuddyModel(modelNorm) {
			if wbBridge == nil || wbBridge.Config().Token == "" {
				writeJSONError(w, 503, "workbuddy_error", "WorkBuddy backend is not configured (missing auth token)")
				return
			}
			err := wbBridge.HandleChat(r.Context(), w, reqBody)
			if err != nil {
				if rec != nil {
					rec.Record("workbuddy:"+modelNorm, false)
				}
				log.Printf("[router] WorkBuddy chat error: %v", err)
			} else {
				if rec != nil {
					rec.Record("workbuddy:"+modelNorm, true)
				}
			}
			return
		}

		// Otherwise dispatch to Qoder
		qb := qoderProvider.currentBridge()
		if qb == nil {
			writeJSONError(w, 503, "qoder_error", "No Qoder PAT configured")
			return
		}

		var usageSink bridge.UsageSink
		if rec != nil {
			usageSink = func(u *transform.Usage) {
				rec.RecordUsage(&stats.Usage{
					PromptTokens:     u.PromptTokens,
					CompletionTokens: u.CompletionTokens,
					CachedTokens:     u.CachedPromptTokens(),
					Credits:          u.Credits,
				})
			}
		}

		err := qb.HandleChat(r.Context(), w, reqBody, usageSink)
		if err != nil {
			clientGone := errors.Is(err, r.Context().Err())
			var repErr *bridge.StreamReportedError
			if errors.As(err, &repErr) {
				if !clientGone && rec != nil {
					rec.Record(modelNorm, false)
				}
				return
			}
			var qErr *bridge.QueueTimeoutError
			if errors.As(err, &qErr) {
				w.Header().Set("Retry-After", "3")
				writeJSONError(w, 429, "rate_limit_error", "upstream busy: queue timeout")
				if rec != nil {
					rec.Record(modelNorm, false)
				}
				return
			}
			var busyErr *auth.BusyError
			if errors.As(err, &busyErr) {
				if busyErr.RetryAfter > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(int(busyErr.RetryAfter.Seconds()+1)))
				}
				writeJSONError(w, 429, "rate_limit_error", "upstream busy: "+busyErr.Message)
				if rec != nil {
					rec.Record(modelNorm, false)
				}
				return
			}
			var valErr *models.UnsupportedModelError
			if errors.As(err, &valErr) {
				writeJSONError(w, 400, "invalid_request_error", valErr.Error())
				if rec != nil {
					rec.Record(modelNorm, false)
				}
				return
			}
			writeJSONError(w, 500, "upstream_error", err.Error())
			if !clientGone && rec != nil {
				rec.Record(modelNorm, false)
			}
			return
		}
		if rec != nil {
			rec.Record(modelNorm, true)
		}
	}
}

// makeUnifiedModelsHandler returns an aggregated OpenAI-compatible model list
// containing models from both Qoder CN and WorkBuddy CN.
func makeUnifiedModelsHandler(st *store.Store, qoderProvider *bridgeProvider, wbBridge *workbuddy.Bridge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token != "" && !st.ValidateKey(token) {
			writeJSONError(w, 401, "invalid_request_error", "Invalid API key")
			return
		}

		type openAIModel struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}

		modelMap := make(map[string]openAIModel)
		now := time.Now().Unix()

		// 1. Collect Qoder models
		qb := qoderProvider.currentBridge()
		var qoderCatalog *models.ModelCatalog
		if qb != nil {
			qoderCatalog = qb.GetCatalog(r.Context())
		} else {
			qoderCatalog = models.DefaultCatalog()
		}
		if qoderCatalog != nil {
			for _, k := range qoderCatalog.Keys() {
				modelMap[k] = openAIModel{
					ID:      k,
					Object:  "model",
					Created: now,
					OwnedBy: "qoder",
				}
			}
		}

		// 2. Collect WorkBuddy models
		for _, mID := range workbuddy.SupportedModelIDs() {
			modelMap[mID] = openAIModel{
				ID:      mID,
				Object:  "model",
				Created: now,
				OwnedBy: "workbuddy",
			}
			// Also add prefixed version for disambiguation
			prefixed := "workbuddy/" + mID
			modelMap[prefixed] = openAIModel{
				ID:      prefixed,
				Object:  "model",
				Created: now,
				OwnedBy: "workbuddy",
			}
		}

		modelList := make([]openAIModel, 0, len(modelMap))
		for _, m := range modelMap {
			modelList = append(modelList, m)
		}
		sort.Slice(modelList, func(i, j int) bool {
			return modelList[i].ID < modelList[j].ID
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"data":   modelList,
		})
	}
}

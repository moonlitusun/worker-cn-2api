package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"worker-cn-2api/qoder"
	"worker-cn-2api/requestopts"
	"worker-cn-2api/stats"
	"worker-cn-2api/store"
	"worker-cn-2api/trae"
	"worker-cn-2api/workbuddy"
)

func extractBearerToken(r *http.Request) string {
	return strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
}
func writeJSONError(w http.ResponseWriter, status int, kind, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": map[string]interface{}{"type": kind, "message": message}})
}
func writeWorkBuddyError(w http.ResponseWriter, err error) {
	var qe *qoder.Error
	if errors.As(err, &qe) {
		if qe.Reported {
			return
		}
		if qe.Status == 429 {
			w.Header().Set("Retry-After", "3")
		}
		status := qe.Status
		if normalized, kind := classifyRequestFailure(err); kind == "timeout" {
			status = normalized
		}
		writeJSONError(w, status, "qoder_error", qe.Message)
		return
	}
	var traeReported *trae.ReportedError
	if errors.As(err, &traeReported) {
		return
	}
	var traeErr *trae.Error
	if errors.As(err, &traeErr) {
		if traeErr.Status == 429 {
			w.Header().Set("Retry-After", "3")
		}
		status := traeErr.Status
		if normalized, kind := classifyRequestFailure(err); kind == "timeout" {
			status = normalized
		}
		writeJSONError(w, status, "trae_error", traeErr.Error())
		return
	}
	var reported *workbuddy.ReportedError
	if errors.As(err, &reported) {
		return
	}
	status, _ := classifyRequestFailure(err)
	var request *workbuddy.RequestError
	if errors.As(err, &request) {
		writeJSONError(w, 400, "invalid_request_error", err.Error())
		return
	}
	var upstream *workbuddy.UpstreamError
	if errors.As(err, &upstream) {
		status = upstream.Status
		if status < 400 || status > 599 {
			status = 502
		}
	}
	var queue *workbuddy.QueueTimeoutError
	if errors.As(err, &queue) {
		status = 429
		w.Header().Set("Retry-After", "3")
	}
	writeJSONError(w, status, "upstream_error", err.Error())
}
func makeChatHandler(st *store.Store, b *workbuddy.Bridge, rec *stats.Recorder, providers ...any) http.HandlerFunc {
	tb, qb := channelProviders(providers)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeJSONError(w, 405, "invalid_request_error", "method not allowed")
			return
		}
		token := extractBearerToken(r)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || token == "" || !st.ValidateKey(token) {
			writeJSONError(w, 401, "invalid_request_error", "Invalid API key")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body == nil {
			writeJSONError(w, 400, "invalid_request_error", "Invalid JSON body")
			return
		}
		model, _ := body["model"].(string)
		isTrae := strings.HasPrefix(strings.TrimSpace(model), "trae/")
		isQoder := strings.HasPrefix(strings.TrimSpace(model), "qoder/")
		if !isQoder && !isTrae && !strings.Contains(model, "/") && qb != nil && qb.Configured() {
			if names, err := qb.Models(r.Context()); err == nil {
				for _, name := range names {
					if strings.TrimSpace(model) == name {
						isQoder = true
						break
					}
				}
			}
		}
		if !isTrae {
			if _, err := requestopts.ReasoningEnabled(body); err != nil {
				writeJSONError(w, 400, "invalid_request_error", err.Error())
				return
			}
		}
		if strings.TrimSpace(model) == "" || (!isTrae && !isQoder && strings.Contains(strings.TrimPrefix(strings.TrimSpace(model), "workbuddy/"), "/")) {
			writeJSONError(w, 400, "invalid_request_error", "model must be a WorkBuddy CN ID from /v1/models")
			return
		}
		messages, ok := body["messages"].([]interface{})
		if !ok || len(messages) == 0 {
			writeJSONError(w, 400, "invalid_request_error", "messages must be a non-empty array")
			return
		}
		if _, ok := body["stream"]; ok {
			if _, ok := body["stream"].(bool); !ok {
				writeJSONError(w, 400, "invalid_request_error", "stream must be a boolean")
				return
			}
		}
		started := time.Now()
		secrets := requestTextSecrets(body["messages"])
		secrets = append(secrets, token, st.GetQoderPAT(), st.GetWorkBuddyToken())
		if b != nil {
			secrets = append(secrets, b.Config().Token)
		}
		if tb != nil {
			c := tb.Config()
			secrets = append(secrets, c.Token, c.RefreshToken)
		}
		if isTrae {
			if tb == nil {
				recordChatResult(rec, "trae", model, started, &workbuddy.UpstreamError{Status: 503, Message: "Trae CN is not configured"}, secrets, r.Context())
				writeJSONError(w, 503, "trae_error", "Trae CN is not configured")
				return
			}
			err := tb.HandleChat(r.Context(), w, body, rec)
			recordChatResult(rec, "trae", model, started, err, secrets, r.Context())
			if err != nil {
				writeWorkBuddyError(w, err)
			}
			return
		}
		if isQoder {
			if qb == nil {
				recordChatResult(rec, "qoder", model, started, &workbuddy.UpstreamError{Status: 503, Message: "Qoder CN is not configured"}, secrets, r.Context())
				writeJSONError(w, 503, "qoder_error", "Qoder CN is not configured")
				return
			}
			err := qb.HandleChat(r.Context(), w, body, rec)
			recordChatResult(rec, "qoder", model, started, err, secrets, r.Context())
			if err != nil {
				writeWorkBuddyError(w, err)
			}
			return
		}
		if b == nil || b.Config().Token == "" {
			recordChatResult(rec, "workbuddy", model, started, &workbuddy.UpstreamError{Status: 503, Message: "WorkBuddy CN auth token is not configured"}, secrets, r.Context())
			writeJSONError(w, 503, "workbuddy_error", "WorkBuddy CN auth token is not configured")
			return
		}
		err := b.HandleChat(r.Context(), w, body, rec)
		recordChatResult(rec, "workbuddy", model, started, err, secrets, r.Context())
		if err != nil {
			writeWorkBuddyError(w, err)
		}
	}
}
func makeModelsHandler(st *store.Store, b *workbuddy.Bridge, providers ...any) http.HandlerFunc {
	tb, qb := channelProviders(providers)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeJSONError(w, 405, "invalid_request_error", "method not allowed")
			return
		}
		if !st.ValidateKey(extractBearerToken(r)) || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			writeJSONError(w, 401, "invalid_request_error", "Invalid API key")
			return
		}
		hasWB := b != nil && b.Config().Token != ""
		hasTrae := tb != nil && tb.Config().Token != ""
		hasQoder := qb != nil && qb.Configured()
		if !hasWB && !hasTrae && !hasQoder {
			writeJSONError(w, 503, "workbuddy_error", "WorkBuddy CN auth token is not configured")
			return
		}
		list := []map[string]interface{}{}
		if hasQoder {
			models, err := qb.Models(r.Context())
			if err != nil {
				writeWorkBuddyError(w, err)
				return
			}
			for _, id := range models {
				list = append(list, map[string]any{"id": "qoder/" + id, "object": "model", "created": time.Now().Unix(), "owned_by": "qoder-cn"})
			}
		}
		if hasWB {
			models, err := b.Models(r.Context())
			if err != nil {
				writeWorkBuddyError(w, err)
				return
			}
			for _, m := range models {
				entry := map[string]interface{}{"id": "workbuddy/" + m.ID, "object": "model", "created": time.Now().Unix(), "owned_by": "workbuddy-cn", "context_window": m.MaxInputTokens, "max_output_tokens": m.MaxOutputTokens}
				if m.SupportsToolCall != nil {
					entry["supports_tool_calls"] = *m.SupportsToolCall
				}
				list = append(list, entry)
			}
		}
		if hasTrae {
			models, err := tb.Models(r.Context())
			if err != nil {
				writeWorkBuddyError(w, err)
				return
			}
			for _, m := range models {
				list = append(list, map[string]interface{}{"id": "trae/" + m.ID, "object": "model", "created": time.Now().Unix(), "owned_by": "trae-cn", "context_window": m.ContextWindow})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"object": "list", "data": list})
	}
}

func channelProviders(providers []any) (*trae.Bridge, *qoder.Provider) {
	var tb *trae.Bridge
	var qb *qoder.Provider
	for _, p := range providers {
		switch v := p.(type) {
		case *trae.Bridge:
			tb = v
		case *qoder.Provider:
			qb = v
		}
	}
	return tb, qb
}

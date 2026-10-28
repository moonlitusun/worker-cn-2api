package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
	"worker-cn-2api/qoder"
	"worker-cn-2api/stats"
	"worker-cn-2api/trae"
	"worker-cn-2api/workbuddy"
)

var credentialPattern = regexp.MustCompile(`(?i)(?:Bearer|Cloud-IDE-JWT)\s+[^\s"',;]+|trae-lt-[a-z0-9_-]+|eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`)
var sensitiveJSONPattern = regexp.MustCompile(`(?i)("(?:access_token|refresh_token|token|authorization|api_key|password)"\s*:\s*")[^"]*`)

func safeFailureText(s string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "[redacted]")
		}
	}
	s = credentialPattern.ReplaceAllString(s, "[redacted]")
	s = sensitiveJSONPattern.ReplaceAllString(s, `${1}[redacted]`)
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 1000 {
		s = s[:1000]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	return s
}

func classifyRequestFailure(err error, requestContexts ...context.Context) (int, string) {
	status := 502
	var qe *qoder.Error
	var te *trae.Error
	var we *workbuddy.UpstreamError
	var invalid *workbuddy.RequestError
	var queue *workbuddy.QueueTimeoutError
	switch {
	case errors.As(err, &qe):
		status = qe.Status
	case errors.As(err, &te):
		status = te.Status
	case errors.As(err, &we):
		status = we.Status
	case errors.As(err, &invalid):
		status = 400
	case errors.As(err, &queue):
		return 429, "queue_timeout"
	}
	if errors.Is(err, context.Canceled) || strings.Contains(strings.ToLower(err.Error()), "context canceled") {
		if len(requestContexts) > 0 && requestContexts[0] != nil && requestContexts[0].Err() == nil {
			return 504, "timeout"
		}
		return 499, "client_cancelled"
	}
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) || strings.Contains(strings.ToLower(err.Error()), "timeout") || strings.Contains(strings.ToLower(err.Error()), "deadline exceeded") || strings.Contains(strings.ToLower(err.Error()), "idle for over") {
		return 504, "timeout"
	}
	if status < 400 || status > 599 {
		status = 502
	}
	switch status {
	case 401, 403:
		return status, "authentication"
	case 429:
		return status, "rate_limit"
	case 400, 422:
		return status, "invalid_request"
	case 504, 408:
		return status, "timeout"
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || regexp.MustCompile(`(?i)\b(?:SSE|stream ended|stream error|stream has|stream broke|stream interrupted)\b`).MatchString(err.Error()) {
		return status, "stream_error"
	}
	return status, "upstream_error"
}

func recordChatResult(rec *stats.Recorder, provider, model string, started time.Time, err error, secrets []string, requestContexts ...context.Context) {
	if rec == nil {
		return
	}
	if !strings.HasPrefix(model, provider+"/") {
		model = provider + "/" + strings.TrimPrefix(model, "workbuddy/")
	}
	rec.Record(model, err == nil)
	if err == nil {
		return
	}
	status, kind := classifyRequestFailure(err, requestContexts...)
	f := stats.Failure{At: time.Now().UTC().Format(time.RFC3339), Provider: provider, Model: safeFailureText(model, secrets), DurationMS: time.Since(started).Milliseconds(), Status: status, Kind: kind, Reason: safeFailureText(err.Error(), secrets)}
	if e := rec.RecordFailure(f); e != nil {
		log.Printf("[request-failure] persistence failed: provider=%s status=%d kind=%s", provider, status, kind)
	}
	log.Printf("[request-failure] provider=%s model=%q status=%d kind=%s duration_ms=%d reason=%q", provider, f.Model, status, kind, f.DurationMS, f.Reason)
}

// Collect supplied text values for exact-match redaction if upstream echoes them.
func requestTextSecrets(v any) []string {
	out := []string{}
	var visit func(any)
	visit = func(value any) {
		switch x := value.(type) {
		case string:
			if len(x) >= 4 {
				out = append(out, x)
			}
		case []any:
			for _, v := range x {
				visit(v)
			}
		case map[string]any:
			for _, v := range x {
				visit(v)
			}
		}
	}
	visit(v)
	return out
}

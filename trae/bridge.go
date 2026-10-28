package trae

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"worker-cn-2api/auth"
	"worker-cn-2api/stats"
)

type ReportedError struct{ Err error }

func (e *ReportedError) Error() string { return e.Err.Error() }
func (e *ReportedError) Unwrap() error { return e.Err }

var errTerminal = errors.New("Trae terminal event")

func (b *Bridge) HandleChat(ctx context.Context, w http.ResponseWriter, body map[string]any, recorders ...*stats.Recorder) (retErr error) {
	o := b.observe()
	defer func() { b.record("chat", o, retErr) }()
	req, err := BuildRequest(body)
	if err != nil {
		return err
	}
	cfg, err := b.ensureCredentials(ctx)
	if err != nil {
		return err
	}
	if cfg.Token == "" {
		return &Error{Status: 503, Message: "auth token is not configured"}
	}
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	b.waiters.Add(1)
	select {
	case b.slots <- struct{}{}:
		b.waiters.Add(-1)
	case <-timer.C:
		b.waiters.Add(-1)
		return &Error{Status: 429, Message: "upstream queue timeout"}
	case <-ctx.Done():
		b.waiters.Add(-1)
		return ctx.Err()
	}
	defer func() { <-b.slots }()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	path := "/api/agent/v3/llm_utils_chat"
	if cfg.ClientID == EnterpriseClientID {
		models, e := b.Models(ctx)
		if e != nil {
			return e
		}
		model, _ := req["model"].(string)
		backend := ""
		function := ""
		for _, m := range models {
			if m.ID == model {
				backend = m.BackendModel
				function = m.Channel
				break
			}
		}
		if backend == "" {
			return bad("model is not in the Trae enterprise catalog")
		}
		path = "/api/ide/v2/llm_raw_chat"
		delete(req, "model")
		req["function"] = function
		req["model_name"] = backend
		req["user_input"] = ""
		session := uuid.NewString()
		req["session_id"] = session
		req["conversation_id"] = session
		req["metadata"] = map[string]any{"agent_type": "chat", "use_prompt_cache_pool": true}
		req["mode_type"] = 0
		req["access_type"] = 4
		if _, ok := req["tools"]; !ok {
			req["tools"] = []any{}
		}
	}
	resp, err := b.request(ctx, cfg, path, req, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var idleTimedOut atomic.Bool
	idle := time.AfterFunc(auth.CurrentStreamTimeouts().Idle, func() { idleTimedOut.Store(true); resp.Body.Close() })
	defer idle.Stop()
	reader := &activityReader{Reader: resp.Body, timer: idle, duration: auth.CurrentStreamTimeouts().Idle}
	type item struct {
		Event Event
		Err   error
	}
	ch := make(chan item)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		defer close(ch)
		err := ReadEvents(reader, func(e Event) error {
			select {
			case ch <- item{Event: e}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		if err != nil {
			select {
			case ch <- item{Err: err}:
			case <-ctx.Done():
			}
		}
	}()
	defer func() { cancel(); resp.Body.Close(); <-readerDone }()
	stream, _ := body["stream"].(bool)
	includeUsage := false
	if options, ok := body["stream_options"].(map[string]any); ok {
		includeUsage, _ = options["include_usage"].(bool)
	}
	id := "chatcmpl-" + uuid.NewString()
	created := time.Now().Unix()
	model, _ := body["model"].(string)
	base := func() map[string]any {
		return map[string]any{"id": id, "object": "chat.completion.chunk", "created": created, "model": model}
	}
	started := false
	roleSent := false
	write := func(data any) error {
		if !started {
			w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-transform")
			w.Header().Set("X-Accel-Buffering", "no")
			started = true
		}
		var frame string
		if text, ok := data.(string); ok {
			frame = text
		} else {
			raw, err := json.Marshal(data)
			if err != nil {
				return err
			}
			frame = "data: " + string(raw) + "\n\n"
		}
		if _, err := io.WriteString(w, frame); err != nil {
			return err
		}
		return http.NewResponseController(w).Flush()
	}
	// HTTP errors remain HTTP errors. Business errors after this point use SSE.
	if stream {
		if err = write(": worker-cn-2api connected\n\n"); err != nil {
			return &ReportedError{err}
		}
	}
	state := &completion{}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	run := func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				if stream {
					if err := write(": ping\n\n"); err != nil {
						return err
					}
				}
			case v, ok := <-ch:
				if !ok {
					return fmt.Errorf("Trae stream ended before done")
				}
				if v.Err != nil {
					return v.Err
				}
				delta, err := state.apply(v.Event)
				if err != nil {
					return err
				}
				if stream && len(delta) > 0 {
					if !roleSent {
						delta["role"] = "assistant"
						roleSent = true
					}
					chunk := base()
					chunk["choices"] = []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}
					if err := write(chunk); err != nil {
						return err
					}
				}
				if state.Done {
					return errTerminal
				}
			}
		}
	}
	err = run()
	if !errors.Is(err, errTerminal) {
		if idleTimedOut.Load() {
			err = fmt.Errorf("Trae stream idle timeout after %s", auth.CurrentStreamTimeouts().Idle)
		}
		var ue *Error
		if errors.As(err, &ue) {
			copy := *ue
			copy.Message = scrub(copy.Message, cfg)
			copy.Code = scrub(fmt.Sprint(copy.Code), cfg)
			err = &copy
		} else {
			err = fmt.Errorf("%s", scrub(err.Error(), cfg))
		}
		if !stream {
			return err
		}
		// Do not emit a successful finish or DONE after a broken upstream stream.
		_ = write(map[string]any{"error": map[string]any{"type": "upstream_error", "message": scrub(err.Error(), cfg)}})
		return &ReportedError{err}
	}
	if len(recorders) > 0 && recorders[0] != nil {
		defer func() {
			if retErr == nil {
				recorders[0].RecordUsageFor("trae", stats.UsageFromMap(state.Usage))
			}
		}()
	}
	if stream {
		delta := map[string]any{}
		if !roleSent {
			delta["role"] = "assistant"
		}
		chunk := base()
		chunk["choices"] = []any{map[string]any{"index": 0, "delta": delta, "finish_reason": state.Finish}}
		if err = write(chunk); err == nil && includeUsage && state.Usage != nil {
			u := base()
			u["choices"] = []any{}
			u["usage"] = state.Usage
			err = write(u)
		}
		if err == nil {
			err = write("data: [DONE]\n\n")
		}
		if err != nil {
			return &ReportedError{err}
		}
		return nil
	}
	result := map[string]any{"id": id, "object": "chat.completion", "created": created, "model": model, "choices": []any{map[string]any{"index": 0, "message": state.message(), "finish_reason": state.Finish}}}
	if state.Usage != nil {
		result["usage"] = state.Usage
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(result)
}

type activityReader struct {
	io.Reader
	timer    *time.Timer
	duration time.Duration
}

func (r *activityReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 {
		r.timer.Reset(r.duration)
	}
	return n, err
}

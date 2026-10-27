// trae-probe verifies the configured account without printing credentials.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"time"

	"worker-cn-2api/store"
	"worker-cn-2api/trae"
)

func main() {
	dataPath := flag.String("data", "data.json", "gateway runtime config (0600; saves rotated session)")
	model := flag.String("model", "", "exact upstream ID; default first visible model")
	modelsOnly := flag.Bool("models-only", false, "query catalog without generating a reply")
	tools := flag.Bool("tools", false, "also verify function calling and a tool-result continuation")
	flag.Parse()
	if err := run(*dataPath, *model, *modelsOnly, *tools); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path, model string, modelsOnly, tools bool) error {
	st, err := store.New(path)
	if err != nil {
		return err
	}
	saved := st.GetTraeSession()
	cfg := trae.ResolveConfig(saved.Token, saved.UserID)
	if saved.Token != "" {
		cfg.RefreshToken = saved.RefreshToken
		cfg.ExpiresAt = saved.ExpiresAt
		if saved.ClientID != "" {
			cfg.ClientID = saved.ClientID
		}
	}
	b := trae.NewBridge(cfg)
	b.SetCredentialSaver(func(c trae.Config) error {
		return st.SetTraeSession(store.TraeSession{Token: c.Token, UserID: c.UserID, RefreshToken: c.RefreshToken, ExpiresAt: c.ExpiresAt, ClientID: c.ClientID})
	})
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	models, err := b.Models(ctx)
	if err != nil {
		return err
	}
	ids := []string{}
	for _, m := range models {
		ids = append(ids, m.ID)
	}
	report := map[string]any{"model_count": len(models), "models": ids}
	if modelsOnly {
		return json.NewEncoder(os.Stdout).Encode(report)
	}
	if model == "" {
		model = models[0].ID
	}
	request := map[string]any{"model": "trae/" + model, "messages": []any{map[string]any{"role": "user", "content": "Reply with OK only."}}, "max_tokens": 64}
	syncCall := func(req map[string]any) (map[string]any, error) {
		w := httptest.NewRecorder()
		if err := b.HandleChat(ctx, w, req); err != nil {
			return nil, err
		}
		var result map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	if _, err = syncCall(request); err != nil {
		return err
	}
	report["sync"] = "passed"
	report["model"] = model
	request["stream"] = true
	request["stream_options"] = map[string]any{"include_usage": true}
	if err = b.HandleChat(ctx, httptest.NewRecorder(), request); err != nil {
		return err
	}
	report["stream"] = "passed"
	if tools {
		request["stream"] = false
		delete(request, "stream_options")
		request["max_tokens"] = 512
		request["messages"] = []any{map[string]any{"role": "user", "content": "Call the ping function with value hello, then report its result."}}
		request["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": "ping", "description": "Echo a value; no external side effects.", "parameters": map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string"}}, "required": []string{"value"}}}}}
		request["tool_choice"] = "required"
		result, err := syncCall(request)
		if err != nil {
			return err
		}
		choices, _ := result["choices"].([]any)
		if len(choices) != 1 {
			return fmt.Errorf("tool probe returned no choice")
		}
		choice, _ := choices[0].(map[string]any)
		msg, _ := choice["message"].(map[string]any)
		calls, _ := msg["tool_calls"].([]any)
		if len(calls) == 0 {
			return fmt.Errorf("model ignored required function call")
		}
		messages := request["messages"].([]any)
		messages = append(messages, msg)
		for _, raw := range calls {
			call, _ := raw.(map[string]any)
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": call["id"], "content": "hello"})
		}
		request["messages"] = messages
		request["tool_choice"] = "none"
		if _, err = syncCall(request); err != nil {
			return err
		}
		report["tools_and_continuation"] = "passed"
	}
	return json.NewEncoder(os.Stdout).Encode(report)
}

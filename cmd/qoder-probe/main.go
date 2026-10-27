package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"time"
	"worker-cn-2api/qoder"
)

func main() {
	path := flag.String("data", "", "read-only configuration file")
	chat := flag.Bool("chat", false, "verify short sync and streaming replies")
	flag.Parse()
	if err := run(*path, *chat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(path string, chat bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg struct {
		PAT string `json:"pat"`
	}
	if json.Unmarshal(data, &cfg) != nil || cfg.PAT == "" {
		return fmt.Errorf("Qoder PAT is missing")
	}
	p := qoder.NewProvider(cfg.PAT)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	models, err := p.Models(ctx)
	if err != nil {
		return err
	}
	result := map[string]any{"model_count": len(models), "models": models}
	if chat {
		model := models[0]
		for _, name := range models {
			if strings.Contains(name, "Flash") {
				model = name
				break
			}
		}
		for _, stream := range []bool{false, true} {
			w := httptest.NewRecorder()
			err := p.HandleChat(ctx, w, map[string]any{"model": "qoder/" + model, "stream": stream, "messages": []any{map[string]any{"role": "user", "content": "Reply with OK only."}}, "enable_thinking": false})
			if err != nil {
				return err
			}
			if stream && !strings.Contains(w.Body.String(), "[DONE]") {
				return fmt.Errorf("stream has no terminal marker")
			}
			if !stream {
				var response map[string]any
				if json.Unmarshal(w.Body.Bytes(), &response) != nil {
					return fmt.Errorf("invalid chat response")
				}
			}
		}
		result["chat_sync_and_stream"] = "passed"
		result["model"] = model
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

package stats

import (
	"encoding/json"
	"math"
)

// UsageFromMap reads upstream counters without estimating missing usage.
func UsageFromMap(raw map[string]any) *Usage {
	if raw == nil {
		return nil
	}
	number := func(v any) float64 {
		var n float64
		switch x := v.(type) {
		case float64:
			n = x
		case int:
			n = float64(x)
		case int64:
			n = float64(x)
		case json.Number:
			n, _ = x.Float64()
		}
		if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1e15 {
			return 0
		}
		return n
	}
	prompt := raw["prompt_tokens"]
	if prompt == nil {
		prompt = raw["input_tokens"]
	}
	completion := raw["completion_tokens"]
	if completion == nil {
		completion = raw["output_tokens"]
	}
	cached := raw["cached_tokens"]
	if details, ok := raw["prompt_tokens_details"].(map[string]any); ok && details["cached_tokens"] != nil {
		cached = details["cached_tokens"]
	}
	if details, ok := raw["input_tokens_details"].(map[string]any); ok && cached == nil {
		cached = details["cached_tokens"]
	}
	return &Usage{PromptTokens: int(number(prompt)), CompletionTokens: int(number(completion)), CachedTokens: int(number(cached)), Credits: number(raw["credits"])}
}

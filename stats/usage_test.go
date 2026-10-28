package stats

import "testing"

func TestUsageFromMap(t *testing.T) {
	if UsageFromMap(nil) != nil {
		t.Fatal("missing usage must not be estimated")
	}
	u := UsageFromMap(map[string]any{"prompt_tokens": float64(12), "completion_tokens": float64(7), "prompt_tokens_details": map[string]any{"cached_tokens": float64(4)}, "credits": 0.25})
	if u.PromptTokens != 12 || u.CompletionTokens != 7 || u.CachedTokens != 4 || u.Credits != 0.25 {
		t.Fatal(u)
	}
	u = UsageFromMap(map[string]any{"input_tokens": float64(6), "output_tokens": float64(3), "input_tokens_details": map[string]any{"cached_tokens": float64(2)}})
	if u.PromptTokens != 6 || u.CompletionTokens != 3 || u.CachedTokens != 2 {
		t.Fatal(u)
	}
	u = UsageFromMap(map[string]any{"prompt_tokens": float64(-1), "completion_tokens": "invalid"})
	if u.PromptTokens != 0 || u.CompletionTokens != 0 {
		t.Fatal(u)
	}
}

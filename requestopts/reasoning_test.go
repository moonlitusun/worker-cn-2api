package requestopts

import "testing"

func TestReasoningEnabled(t *testing.T) {
	cases := []struct {
		name  string
		input map[string]interface{}
		want  bool
		bad   bool
	}{
		{"legacy default", map[string]interface{}{}, true, false},
		{"off", map[string]interface{}{"reasoning_effort": "none"}, false, false},
		{"high", map[string]interface{}{"reasoning_effort": "high"}, true, false},
		{"deepseek off", map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}}, false, false},
		{"deepseek on", map[string]interface{}{"thinking": map[string]interface{}{"type": "enabled"}, "reasoning_effort": "medium"}, true, false},
		{"qwen off", map[string]interface{}{"enable_thinking": false}, false, false},
		{"template off", map[string]interface{}{"chat_template_kwargs": map[string]interface{}{"enable_thinking": false}}, false, false},
		{"invalid effort", map[string]interface{}{"reasoning_effort": "banana"}, false, true},
		{"invalid type", map[string]interface{}{"enable_thinking": "false"}, false, true},
		{"invalid thinking", map[string]interface{}{"thinking": map[string]interface{}{"type": "auto"}}, false, true},
		{"contradiction", map[string]interface{}{"enable_thinking": false, "reasoning_effort": "high"}, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ReasoningEnabled(c.input)
			if (err != nil) != c.bad {
				t.Fatalf("error=%v, want bad=%v", err, c.bad)
			}
			if err == nil && got != c.want {
				t.Fatalf("enabled=%v want %v", got, c.want)
			}
		})
	}
}

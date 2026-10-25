// Package requestopts normalizes client controls shared by upstream adapters.
package requestopts

import (
	"fmt"
	"sort"
	"strings"
)

// ReasoningEnabled maps explicit client controls to the upstream boolean switch.
// An omitted control preserves the bridge's existing reasoning-enabled behavior.
func ReasoningEnabled(req map[string]interface{}) (bool, error) {
	enabled := true
	specified := false
	set := func(value bool, field string) error {
		if specified && enabled != value {
			return fmt.Errorf("conflicting reasoning controls: %s", field)
		}
		enabled, specified = value, true
		return nil
	}
	if raw, ok := req["reasoning_effort"]; ok {
		effort, ok := raw.(string)
		if !ok {
			return false, fmt.Errorf("reasoning_effort must be a string")
		}
		var on bool
		switch effort {
		case "none", "off":
			on = false
		case "minimal", "low", "medium", "high", "xhigh", "max":
			on = true
		default:
			return false, fmt.Errorf("unsupported reasoning_effort")
		}
		if err := set(on, "reasoning_effort"); err != nil {
			return false, err
		}
	}
	if raw, ok := req["thinking"]; ok {
		thinking, ok := raw.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("thinking must be an object")
		}
		kind, _ := thinking["type"].(string)
		if kind != "enabled" && kind != "disabled" {
			return false, fmt.Errorf("thinking.type must be enabled or disabled")
		}
		if err := set(kind == "enabled", "thinking.type"); err != nil {
			return false, err
		}
	}
	if raw, ok := req["enable_thinking"]; ok {
		on, ok := raw.(bool)
		if !ok {
			return false, fmt.Errorf("enable_thinking must be a boolean")
		}
		if err := set(on, "enable_thinking"); err != nil {
			return false, err
		}
	}
	if raw, ok := req["chat_template_kwargs"]; ok {
		kwargs, ok := raw.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("chat_template_kwargs must be an object")
		}
		if raw, ok := kwargs["enable_thinking"]; ok {
			on, ok := raw.(bool)
			if !ok {
				return false, fmt.Errorf("chat_template_kwargs.enable_thinking must be a boolean")
			}
			if err := set(on, "chat_template_kwargs.enable_thinking"); err != nil {
				return false, err
			}
		}
	}
	return enabled, nil
}

// UnsupportedQoderParameters names controls with no verified upstream equivalent.
// Effort still toggles reasoning; its requested intensity cannot be represented.
func UnsupportedQoderParameters(req map[string]interface{}) string {
	fields := []string{}
	for _, field := range []string{"temperature", "top_p", "max_tokens", "max_completion_tokens", "stop", "frequency_penalty", "presence_penalty", "seed"} {
		if _, ok := req[field]; ok {
			fields = append(fields, field)
		}
	}
	if effort, ok := req["reasoning_effort"].(string); ok && effort != "none" && effort != "off" {
		fields = append(fields, "reasoning_effort")
	}
	sort.Strings(fields)
	return strings.Join(fields, ", ")
}

package workbuddy

import (
	"fmt"
	"strings"
)

// ModelInfo describes a model supported by WorkBuddy CN.
type ModelInfo struct {
	ID          string `json:"id"`
	UpstreamKey string `json:"upstream_key"`
	Description string `json:"description"`
	IsReasoning bool   `json:"is_reasoning"`
}

var supportedModels = []ModelInfo{
	{
		ID:          "deepseek-v3",
		UpstreamKey: "deepseek-v3",
		Description: "DeepSeek V3 深度求索通用编程大模型",
		IsReasoning: false,
	},
	{
		ID:          "deepseek-r1",
		UpstreamKey: "deepseek-r1",
		Description: "DeepSeek R1 具备原生思考过程（Reasoning）的推理模型",
		IsReasoning: true,
	},
	{
		ID:          "hunyuan-code",
		UpstreamKey: "hunyuan-standard-code",
		Description: "腾讯混元代码专精大模型",
		IsReasoning: false,
	},
	{
		ID:          "kimi-k2",
		UpstreamKey: "kimi-k2",
		Description: "Kimi 长上下文编程模型",
		IsReasoning: false,
	},
	{
		ID:          "glm-4",
		UpstreamKey: "glm-4-plus",
		Description: "智谱 GLM-4 旗舰大模型",
		IsReasoning: false,
	},
}

// SupportedModelIDs returns the list of all supported OpenAI model IDs.
func SupportedModelIDs() []string {
	ids := make([]string, len(supportedModels))
	for i, m := range supportedModels {
		ids[i] = m.ID
	}
	return ids
}

// IsWorkbuddyModel checks if the given model parameter belongs to WorkBuddy.
func IsWorkbuddyModel(modelParam string) bool {
	norm := strings.ToLower(strings.TrimSpace(modelParam))
	norm = strings.TrimPrefix(norm, "workbuddy/")
	for _, m := range supportedModels {
		if strings.ToLower(m.ID) == norm || strings.ToLower(m.UpstreamKey) == norm {
			return true
		}
	}
	return false
}

// ResolveModel maps an OpenAI model name to the WorkBuddy upstream model key.
func ResolveModel(modelParam string) (*ModelInfo, error) {
	norm := strings.ToLower(strings.TrimSpace(modelParam))
	norm = strings.TrimPrefix(norm, "workbuddy/")
	if norm == "" {
		// Default model
		return &supportedModels[0], nil
	}
	for _, m := range supportedModels {
		if strings.ToLower(m.ID) == norm || strings.ToLower(m.UpstreamKey) == norm {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("unsupported WorkBuddy model %q, available: %s", modelParam, strings.Join(SupportedModelIDs(), ", "))
}

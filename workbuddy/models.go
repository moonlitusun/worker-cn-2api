package workbuddy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ModelInfo is a model from the authenticated WorkBuddy CN catalog.
type ModelInfo struct {
	ID                string   `json:"id"`
	Name              string   `json:"name,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	SupportsToolCall  *bool    `json:"supportsToolCall,omitempty"`
	SupportsReasoning bool     `json:"supportsReasoning,omitempty"`
	MaxInputTokens    int      `json:"maxInputTokens,omitempty"`
	MaxOutputTokens   int      `json:"maxOutputTokens,omitempty"`
}

// Models caches only a successfully fetched catalog for the active credential.
func (b *Bridge) Models(ctx context.Context) ([]ModelInfo, error) {
	return b.models(ctx, false)
}

// TestModels bypasses the catalog cache for explicit connection tests.
func (b *Bridge) TestModels(ctx context.Context) ([]ModelInfo, error) {
	return b.models(ctx, true)
}

func (b *Bridge) models(ctx context.Context, fresh bool) ([]ModelInfo, error) {
	b.catalogMu.Lock()
	defer b.catalogMu.Unlock()
	cfg := b.Config()
	if cfg.Token == "" {
		return nil, fmt.Errorf("WorkBuddy CN token is not configured")
	}
	if !fresh && b.catalogToken == cfg.Token && time.Since(b.catalogAt) < time.Hour {
		return append([]ModelInfo(nil), b.catalog...), nil
	}
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	endpoint.Path = "/v2/enterprises/personal/models"
	endpoint.RawQuery = ""
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	setHeaders(req, cfg)
	resp, err := b.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, readUpstreamError(resp)
	}
	var raw struct {
		Models []ModelInfo `json:"models"`
		Data   struct {
			Models []ModelInfo `json:"models"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid model catalog: %w", err)
	}
	items := raw.Models
	if len(items) == 0 {
		items = raw.Data.Models
	}
	seen := map[string]bool{}
	result := []ModelInfo{}
	for _, m := range items {
		imageOnly := false
		for _, tag := range m.Tags {
			if tag == "text-to-image" {
				imageOnly = true
			}
		}
		if m.ID != "" && !seen[m.ID] && !imageOnly {
			seen[m.ID] = true
			result = append(result, m)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("WorkBuddy CN returned an empty model catalog")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	b.catalog, b.catalogAt, b.catalogToken = result, time.Now(), cfg.Token
	return append([]ModelInfo(nil), result...), nil
}

func normalizeModel(model string) (string, error) {
	model = strings.TrimPrefix(strings.TrimSpace(model), "workbuddy/")
	if model == "" {
		return "", fmt.Errorf("model is required; select an ID from /v1/models")
	}
	if strings.Contains(model, "/") {
		return "", fmt.Errorf("only WorkBuddy CN model IDs are supported")
	}
	return model, nil
}

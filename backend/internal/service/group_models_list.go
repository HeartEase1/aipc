package service

import (
	"slices"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// A partial mapping catalog must not hide models from unmapped OpenAI accounts.
// Keep an empty catalog unchanged so callers retain their existing discovery fallback.
func supplementUnmappedOpenAIModels(accounts []Account, models []string) []string {
	if len(models) == 0 {
		return models
	}
	for i := range accounts {
		account := &accounts[i]
		if account.Platform == PlatformOpenAI && len(account.GetModelMapping()) == 0 {
			return dedupeAndSortModelIDs(slices.Concat(models, openai.DefaultModelIDs()))
		}
	}
	return models
}

func normalizeGroupModelsListConfig(cfg GroupModelsListConfig) GroupModelsListConfig {
	return GroupModelsListConfig{
		Enabled:       cfg.Enabled,
		Models:        normalizeGroupModelIDs(cfg.Models),
		BlockedModels: normalizeGroupModelIDs(cfg.BlockedModels),
	}
}

func normalizeGroupModelIDs(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		out = append(out, model)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (g *Group) CustomModelsListEnabled() bool {
	return g != nil && g.ModelsListConfig.Enabled && len(g.ModelsListConfig.Models) > 0
}

func (g *Group) HasBlockedModels() bool {
	return g != nil && len(g.ModelsListConfig.BlockedModels) > 0
}

// IsModelBlocked checks the client-visible model before channel or account
// mapping. A trailing '*' is supported for model families. Gemini's optional
// "models/" resource prefix is ignored so the list and invocation APIs share
// the same rule.
func (g *Group) IsModelBlocked(model string) bool {
	if !g.HasBlockedModels() {
		return false
	}
	model = normalizeBlockedModelMatchValue(model)
	if model == "" {
		return false
	}
	for _, rule := range g.ModelsListConfig.BlockedModels {
		rule = normalizeBlockedModelMatchValue(rule)
		if rule == "" {
			continue
		}
		if strings.HasSuffix(rule, "*") {
			if prefix := strings.TrimSuffix(rule, "*"); prefix != "" && strings.HasPrefix(model, prefix) {
				return true
			}
			continue
		}
		if model == rule {
			return true
		}
	}
	return false
}

func normalizeBlockedModelMatchValue(model string) string {
	model = strings.TrimSpace(model)
	return strings.TrimPrefix(model, "models/")
}

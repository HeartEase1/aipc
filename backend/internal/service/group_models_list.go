package service

import "strings"

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

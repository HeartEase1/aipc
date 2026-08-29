package domain

// GroupModelsListConfig controls the optional custom /v1/models response list
// and the models that the group is not allowed to invoke.
type GroupModelsListConfig struct {
	Enabled       bool     `json:"enabled"`
	Models        []string `json:"models,omitempty"`
	BlockedModels []string `json:"blocked_models,omitempty"`
}

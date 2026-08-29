package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const groupModelDisabledMessage = "Model is disabled for this API key group"

func rejectBlockedAPIKeyModel(c *gin.Context, apiKey *service.APIKey, model string) bool {
	if c == nil || apiKey == nil || apiKey.Group == nil || !apiKey.Group.IsModelBlocked(model) {
		return false
	}
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"type":    "permission_error",
			"code":    "MODEL_DISABLED_BY_GROUP",
			"message": groupModelDisabledMessage,
		},
	})
	return true
}

// filterModelListEnvelope removes blocked entries while preserving the rest of
// an upstream model-list envelope. idFields are checked in order (for example
// Gemini uses name, while Codex manifests commonly use slug or id).
func filterModelListEnvelope(body []byte, listField string, group *service.Group, idFields ...string) ([]byte, error) {
	if group == nil || !group.HasBlockedModels() {
		return body, nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode model list envelope: %w", err)
	}
	rawList, ok := envelope[listField]
	if !ok {
		return nil, fmt.Errorf("model list envelope missing %q", listField)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(rawList, &entries); err != nil {
		return nil, fmt.Errorf("decode model list entries: %w", err)
	}

	filtered := make([]json.RawMessage, 0, len(entries))
	for _, entry := range entries {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(entry, &fields); err != nil {
			return nil, fmt.Errorf("decode model list entry: %w", err)
		}
		modelID := ""
		for _, field := range idFields {
			var candidate string
			if raw, exists := fields[field]; exists && json.Unmarshal(raw, &candidate) == nil {
				candidate = strings.TrimSpace(candidate)
				if candidate != "" {
					modelID = candidate
					break
				}
			}
		}
		if modelID == "" || !group.IsModelBlocked(modelID) {
			filtered = append(filtered, entry)
		}
	}

	filteredJSON, err := json.Marshal(filtered)
	if err != nil {
		return nil, fmt.Errorf("encode filtered model list: %w", err)
	}
	envelope[listField] = filteredJSON
	out, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode filtered model list envelope: %w", err)
	}
	return bytes.TrimSpace(out), nil
}

func filterBatchImageModelsByBlockedPolicy(models []service.BatchImagePublicModel, group *service.Group) []service.BatchImagePublicModel {
	if group == nil || !group.HasBlockedModels() || len(models) == 0 {
		return models
	}
	filtered := make([]service.BatchImagePublicModel, 0, len(models))
	for _, model := range models {
		if !group.IsModelBlocked(model.ID) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

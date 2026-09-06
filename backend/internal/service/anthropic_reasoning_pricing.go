package service

import (
	"math"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/tidwall/gjson"
)

const claudeFable51MaxReasoningEffortMultiplier = 3.0

// Conversion must not raise xhigh to max after the group policy has run.
func normalizeConvertedAnthropicEffort(request *apicompat.AnthropicRequest, sourceEffort *string) {
	if sourceEffort == nil || request.OutputConfig == nil {
		return
	}
	effort := NormalizeMaxReasoningEffort(*sourceEffort)
	if effort == "" {
		return
	}
	if effort == "minimal" {
		effort = "low"
	}
	if levels := claude.EffortLevelsForModel(request.Model); len(levels) > 0 {
		rank, _ := reasoningEffortRank(effort)
		for i := len(levels) - 1; i >= 0; i-- {
			if supportedRank, _ := reasoningEffortRank(levels[i]); supportedRank <= rank {
				effort = levels[i]
				break
			}
		}
	}
	request.OutputConfig.Effort = effort
	if effort == "low" {
		request.Thinking = nil
	} else if request.Thinking != nil {
		switch effort {
		case "medium":
			request.Thinking.BudgetTokens = 4096
		case "high":
			request.Thinking.BudgetTokens = 10240
		case "xhigh", "max":
			request.Thinking.BudgetTokens = 32768
		}
	}
}

func extractAnthropicReasoningEffort(body []byte) *string {
	effort := gjson.GetBytes(body, "output_config.effort")
	if effort.Type != gjson.String {
		return nil
	}
	return optionalTrimmedStringPtr(effort.String())
}

func isClaudeFable51Model(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, marker := range []string{"fable-5-1", "fable-5.1", "fable5.1", "fable51"} {
		if at := strings.Index(model, marker); at >= 0 {
			after := at + len(marker)
			if after == len(model) || model[after] < '0' || model[after] > '9' {
				return true
			}
		}
	}
	return false
}

// DefaultMaxReasoningEffortMultiplier returns the model's built-in Max surcharge.
func DefaultMaxReasoningEffortMultiplier(model string) *float64 {
	if !isClaudeFable51Model(model) {
		return nil
	}
	multiplier := claudeFable51MaxReasoningEffortMultiplier
	return &multiplier
}

func reasoningBillingModel(model, upstreamModel string) string {
	if strings.TrimSpace(upstreamModel) != "" {
		return upstreamModel
	}
	return model
}

func maxReasoningEffortBillingMultiplier(model, effort string, pricing *ModelPricing) float64 {
	if NormalizeMaxReasoningEffort(effort) != "max" {
		return 1
	}
	if pricing != nil && pricing.MaxReasoningEffortMultiplier != nil {
		value := *pricing.MaxReasoningEffortMultiplier
		if value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) {
			return value
		}
	}
	if multiplier := DefaultMaxReasoningEffortMultiplier(model); multiplier != nil {
		return *multiplier
	}
	return 1
}

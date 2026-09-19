package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGroupLongContextExemptionUsesBaseTierOnlyForListedModel(t *testing.T) {
	const catalog = `{"gpt-test":{"litellm_provider":"openai","mode":"chat",
		"input_cost_per_token":0.000001,"output_cost_per_token":0.000002,
		"long_context_input_token_threshold":100,
		"long_context_input_cost_multiplier":2,"long_context_output_cost_multiplier":1.5},
		"gpt-other":{"litellm_provider":"openai","mode":"chat",
		"input_cost_per_token":0.000001,"output_cost_per_token":0.000002,
		"long_context_input_token_threshold":100,
		"long_context_input_cost_multiplier":2,"long_context_output_cost_multiplier":1.5}}`
	billing := NewBillingService(&config.Config{}, newStubPricingServiceFromJSON(t, catalog))
	resolver := NewModelPricingResolver(nil, billing)
	group := &Group{LongContextPricingEnabled: true,
		LongContextPricingExemptModels: []string{"GPT-TEST"}}
	tokens := UsageTokens{InputTokens: 120, OutputTokens: 10}

	exempt, err := billing.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "gpt-test", Group: group,
		Tokens: tokens, RateMultiplier: 1, Resolver: resolver,
	})
	require.NoError(t, err)
	require.False(t, exempt.LongContextBillingApplied)
	require.InDelta(t, 120*0.000001, exempt.InputCost, 1e-12)
	require.InDelta(t, 10*0.000002, exempt.OutputCost, 1e-12)

	other, err := billing.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "gpt-other", Group: group,
		Tokens: tokens, RateMultiplier: 1, Resolver: resolver,
	})
	require.NoError(t, err)
	require.True(t, other.LongContextBillingApplied)
	require.InDelta(t, 120*0.000002, other.InputCost, 1e-12)
	require.InDelta(t, 10*0.000003, other.OutputCost, 1e-12)
}

func TestLongContextExemptionsRejectWildcardsAndDeduplicate(t *testing.T) {
	models := normalizeLongContextPricingExemptModels([]string{" gpt-test ", "GPT-TEST", "gpt-*", "", "gpt-other"})
	require.Equal(t, []string{"gpt-test", "gpt-other"}, models)
}

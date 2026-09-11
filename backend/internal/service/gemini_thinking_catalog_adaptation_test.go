//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Based on upstream 3bdb869fc: tier suffixes select reasoning, not a new rate.
func TestGatewayServiceRecordUsage_GeminiFlashThinkingTierUsesCatalogPrice(t *testing.T) {
	for _, baseModel := range []string{"gemini-3.7-flash", "gemini-3.8-flash"} {
		t.Run(baseModel, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			userRepo := &openAIRecordUsageUserRepoStub{}
			svc := newGatewayRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{})
			svc.billingService = NewBillingService(svc.cfg, &PricingService{pricingData: map[string]*LiteLLMModelPricing{
				baseModel: {InputCostPerToken: 0.75e-6, OutputCostPerToken: 3.75e-6, CacheReadInputTokenCost: 0.075e-6},
			}})
			svc.resolver = NewModelPricingResolver(nil, svc.billingService)
			group := &Group{ID: 27, Platform: PlatformGemini, RateMultiplier: 0.15}
			model := baseModel + "-medium"
			err := svc.RecordUsage(context.Background(), &RecordUsageInput{
				Result: &ForwardResult{
					RequestID: "gemini_thinking_tier", Model: model, UpstreamModel: model,
					Usage:    ClaudeUsage{InputTokens: 8498, OutputTokens: 469, CacheReadInputTokens: 159248},
					Duration: time.Second,
				},
				APIKey:  &APIKey{ID: 501, GroupID: &group.ID, Group: group},
				User:    &User{ID: 601},
				Account: &Account{ID: 701, Platform: PlatformGemini, Type: AccountTypeAPIKey},
			})
			require.NoError(t, err)
			require.NotNil(t, usageRepo.lastLog)
			require.Equal(t, model, usageRepo.lastLog.Model)
			require.InDelta(t, 0.02007585, usageRepo.lastLog.TotalCost, 1e-12)
			require.InDelta(t, 0.0030113775, usageRepo.lastLog.ActualCost, 1e-12)
			require.InDelta(t, 0.0030113775, userRepo.lastAmount, 1e-12)
		})
	}
}

func TestGeminiThinkingCatalogUnknownTierIsNotIdentified(t *testing.T) {
	svc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gemini-3.8-flash": {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6},
	}}
	require.Nil(t, svc.GetIdentifiedModelPricing("gemini-3.8-flash-unknown"))
	require.Nil(t, svc.GetIdentifiedModelPricing("gemini-99-flash-high"))
}

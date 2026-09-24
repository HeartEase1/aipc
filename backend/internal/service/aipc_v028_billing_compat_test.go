package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAIPCEffortPricingStacksOnceWithExemptionAndMarketing(t *testing.T) {
	now := time.Now()
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	previous := defaultDiscountCampaignService.Swap(&DiscountCampaignService{
		campaigns: []runtimeDiscountCampaign{{id: 9, scheduleType: DiscountScheduleOneTime,
			location: time.UTC, startsAt: &start, endsAt: &end, factor: 0.8}},
	})
	t.Cleanup(func() { defaultDiscountCampaignService.Store(previous) })
	bs := NewBillingService(nil, nil)
	bs.fallbackPrices["gpt-5.4"] = &ModelPricing{
		InputPricePerToken: 1e-6, OutputPricePerToken: 2e-6,
		LongContextInputThreshold: 100, LongContextInputMultiplier: 2, LongContextOutputMultiplier: 1.5,
		ReasoningEffortMultipliers: map[string]float64{"high": 1.5},
	}
	for _, withResolver := range []bool{false, true} {
		for _, exempt := range []bool{false, true} {
			group := &Group{ID: 1, Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, LongContextPricingEnabled: true}
			if exempt {
				group.LongContextPricingExemptModels = []string{"gpt-5.4"}
			}
			gateway := &OpenAIGatewayService{billingService: bs}
			if withResolver {
				gateway.resolver = NewModelPricingResolver(nil, bs)
			}
			cost, err := gateway.calculateOpenAIRecordUsageTokenCost(context.Background(), &APIKey{Group: group},
				"gpt-5.4", 0.5, now, UsageTokens{InputTokens: 120, OutputTokens: 10}, "", "high", nil)
			require.NoError(t, err)
			base := 120*2e-6 + 10*3e-6
			if exempt {
				base = 120*1e-6 + 10*2e-6
			}
			require.Equal(t, !exempt, cost.LongContextBillingApplied)
			require.Equal(t, string(BillingModeToken), cost.BillingMode)
			require.InDelta(t, base*1.5, cost.TotalCost, 1e-12)
			resolution := applyTokenMarketingDiscount(group, 7, now, 0.5, cost, 0)
			require.NotNil(t, resolution)
			require.InDelta(t, base*1.5*0.5*0.8, cost.ActualCost, 1e-12)
			require.InDelta(t, base*1.5, cost.TotalCost, 1e-12, "marketing must preserve the upstream charge")
			usage := &UsageLog{}
			applyDiscountResolutionToUsageLog(usage, cost, resolution)
			require.InDelta(t, base*1.5*0.5*0.2, usage.DiscountAmount, 1e-12)
		}
	}
}

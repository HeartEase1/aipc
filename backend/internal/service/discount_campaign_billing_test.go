package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenMarketingDiscountPreservesOfficialCostAndFixedFees(t *testing.T) {
	now := time.Now()
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	svc := &DiscountCampaignService{
		campaigns: []runtimeDiscountCampaign{{id: 9, scheduleType: DiscountScheduleOneTime, location: time.UTC, startsAt: &start, endsAt: &end, factor: 0.8}},
		excluded:  map[int64]map[string]struct{}{2: {"usage": {}}},
	}
	previous := defaultDiscountCampaignService.Swap(svc)
	t.Cleanup(func() { defaultDiscountCampaignService.Store(previous) })
	for _, tc := range []struct {
		name, mode, subscription        string
		userID                          int64
		fixed, wantCharge, wantDiscount float64
	}{
		{"token", "token", SubscriptionTypeStandard, 1, 0, 8, 2},
		{"token plus search", "token", SubscriptionTypeStandard, 1, 2, 8.4, 1.6},
		{"search only", "token", SubscriptionTypeStandard, 1, 10, 10, 0},
		{"fixed request", "per_request", SubscriptionTypeStandard, 1, 0, 10, 0},
		{"image", "image", SubscriptionTypeStandard, 1, 0, 10, 0},
		{"video", "video", SubscriptionTypeStandard, 1, 0, 10, 0},
		{"subscription", "token", SubscriptionTypeSubscription, 1, 0, 10, 0},
		{"excluded user", "token", SubscriptionTypeStandard, 2, 0, 10, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Free Fast can make the customer charge differ from the upstream tier cost.
			cost := &CostBreakdown{BillingMode: tc.mode, TotalCost: 20, InputCost: 6, OutputCost: 14, ActualCost: 10}
			resolution := applyTokenMarketingDiscount(&Group{SubscriptionType: tc.subscription}, tc.userID, now, 2, cost, tc.fixed)
			usage := &UsageLog{}
			applyDiscountResolutionToUsageLog(usage, cost, resolution)
			require.InDelta(t, tc.wantCharge, cost.ActualCost, 1e-9)
			require.InDelta(t, tc.wantDiscount, usage.DiscountAmount, 1e-9)
			require.Equal(t, float64(20), cost.TotalCost)
			require.Equal(t, float64(6), cost.InputCost)
			require.Equal(t, float64(14), cost.OutputCost)
			if tc.wantDiscount == 0 {
				require.Nil(t, resolution)
			}
		})
	}
}

func TestTokenMarketingFallbackPricingIsTagged(t *testing.T) {
	billing := NewBillingService(nil, nil)
	apiKey := &APIKey{Group: &Group{SubscriptionType: SubscriptionTypeStandard}}
	general := &GatewayService{billingService: billing}
	cost := general.calculateTokenCost(context.Background(), &ForwardResult{Usage: ClaudeUsage{InputTokens: 100, OutputTokens: 50}}, apiKey, "claude-sonnet-4-5", 1, time.Now())
	require.Positive(t, cost.ActualCost)
	require.Equal(t, string(BillingModeToken), cost.BillingMode)
	openai := &OpenAIGatewayService{billingService: billing}
	cost, err := openai.calculateOpenAIRecordUsageTokenCost(context.Background(), apiKey, "gpt-5", 1, time.Now(), UsageTokens{InputTokens: 100, OutputTokens: 50}, "", "", nil)
	require.NoError(t, err)
	require.Positive(t, cost.ActualCost)
	require.Equal(t, string(BillingModeToken), cost.BillingMode)
}

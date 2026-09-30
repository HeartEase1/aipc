//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A reservation is an admission estimate, not a second debit. The balance cache
// must reflect AIPC's final marketing charge before the billing reference drops.
func TestInflightReservation_AIPCMarketingSettlement(t *testing.T) {
	for _, tc := range []struct {
		name      string
		excluded  bool
		fixedCost float64
		mode      BillingMode
		want      float64
	}{
		{name: "token discount", mode: BillingModeToken, want: 0.2},
		{name: "excluded user", excluded: true, mode: BillingModeToken, want: 0.4},
		{name: "fixed charge stays undiscounted", fixedCost: 0.1, mode: BillingModeToken, want: 0.25},
		{name: "per request stays undiscounted", mode: BillingModePerRequest, want: 0.4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
			start, end := at.Add(-time.Hour), at.Add(time.Hour)
			campaigns := &DiscountCampaignService{campaigns: []runtimeDiscountCampaign{{
				id: 11, scheduleType: DiscountScheduleOneTime, location: time.UTC,
				startsAt: &start, endsAt: &end, factor: 0.5,
			}}}
			if tc.excluded {
				campaigns.excluded = map[int64]map[string]struct{}{1: {"usage": {}}}
			}
			previous := defaultDiscountCampaignService.Swap(campaigns)
			t.Cleanup(func() { defaultDiscountCampaignService.Store(previous) })
			cache := newMemInflightCache(1)
			svc := newInflightSvc(t, cache, 60)
			user := &User{ID: 1}
			group := &Group{ID: 2, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1}
			ctx := context.Background()
			reservation, err := svc.ReserveInflight(ctx, user, group, nil, 0.9)
			require.NoError(t, err)
			require.NotNil(t, reservation)
			t.Cleanup(reservation.Release)
			done := reservation.Acquire()
			reservation.HandlerDone()
			require.Equal(t, 1, cache.count())
			balance, err := cache.GetUserBalance(ctx, user.ID)
			require.NoError(t, err)
			require.Equal(t, 1.0, balance, "reservation must not debit the balance")

			cost := &CostBreakdown{BillingMode: string(tc.mode), TotalCost: 0.4, ActualCost: 0.4}
			applyOpenAITokenMarketingDiscount(group, user.ID, &OpenAIForwardResult{Model: "gpt-6.1-sol"},
				[]string{"gpt-6.1-sol"}, at, 1, cost, tc.fixedCost)
			require.InDelta(t, tc.want, cost.ActualCost, 1e-12)
			require.Equal(t, 0.4, cost.TotalCost, "marketing must not rewrite the upstream basis")
			syncBalanceCacheAfterDeduction(ctx, &postUsageBillingParams{User: user, Cost: cost},
				&billingDeps{billingCacheService: svc}, nil)
			balance, err = cache.GetUserBalance(ctx, user.ID)
			require.NoError(t, err)
			require.InDelta(t, 1-tc.want, balance, 1e-12)
			require.Equal(t, 1, cache.count(), "hold until the billing task finishes")
			done()
			done()
			require.Zero(t, cache.count())
			require.Equal(t, int32(1), cache.deducts.Load())
			require.Equal(t, int32(1), cache.releaseCt.Load())
		})
	}
}

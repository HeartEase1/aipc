package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestDiscountCampaignResolveChoosesLowestEffectiveMultiplier(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	svc := &DiscountCampaignService{campaigns: []runtimeDiscountCampaign{
		{id: 3, name: "ten percent", scheduleType: DiscountScheduleOneTime, location: time.UTC, startsAt: &start, endsAt: &end, factor: 0.9},
		{id: 2, name: "twenty percent", description: "Weekend offer", scheduleType: DiscountScheduleOneTime, location: time.UTC, startsAt: &start, endsAt: &end, factor: 0.8},
	}}

	resolved := svc.Resolve(&Group{SubscriptionType: SubscriptionTypeStandard}, now, 2)
	require.NotNil(t, resolved)
	require.Equal(t, int64(2), resolved.CampaignID)
	require.Equal(t, "Weekend offer", resolved.CampaignDescription)
	require.InDelta(t, 1.6, resolved.EffectiveRateMultiplier, 1e-9)
}

func TestDiscountCampaignResolveHonorsGroupScope(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	svc := &DiscountCampaignService{campaigns: []runtimeDiscountCampaign{
		{id: 1, scheduleType: DiscountScheduleOneTime, location: time.UTC, startsAt: &start, endsAt: &end, factor: 0.9},
		{id: 2, scheduleType: DiscountScheduleOneTime, location: time.UTC, startsAt: &start, endsAt: &end, factor: 0.8, groupIDs: map[int64]struct{}{20: {}}},
	}}

	allGroupsOnly := svc.Resolve(&Group{ID: 10, SubscriptionType: SubscriptionTypeStandard}, now, 2)
	require.NotNil(t, allGroupsOnly)
	require.Equal(t, int64(1), allGroupsOnly.CampaignID)

	selectedGroup := svc.Resolve(&Group{ID: 20, SubscriptionType: SubscriptionTypeStandard}, now, 2)
	require.NotNil(t, selectedGroup)
	require.Equal(t, int64(2), selectedGroup.CampaignID)
}

func TestDiscountCampaignGroupIDsAreNormalizedAndValidated(t *testing.T) {
	input := DiscountCampaignInput{
		Name: "test", ActorID: 1, GroupIDs: []int64{9, 3, 9},
		ScheduleType: DiscountScheduleOneTime, StartsAt: "2026-08-09T00:00:00Z",
		EndsAt: "2026-08-10T00:00:00Z", DiscountFactor: "0.9",
	}
	validated, err := validateDiscountCampaignInput(input)
	require.NoError(t, err)
	require.Equal(t, []int64{3, 9}, validated.GroupIDs)

	input.GroupIDs = []int64{0}
	_, err = validateDiscountCampaignInput(input)
	require.Error(t, err)
	require.Equal(t, "INVALID_DISCOUNT_GROUPS", infraerrors.Reason(err))
}

func TestDiscountCampaignGroupScopeRejectsNonBalanceGroups(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc := NewDiscountCampaignService(db)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM groups").
		WithArgs(pq.Array([]int64{3, 9}), SubscriptionTypeStandard).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	err = svc.validateGroupScope(context.Background(), []int64{3, 9})
	require.Error(t, err)
	require.Equal(t, "INVALID_DISCOUNT_GROUPS", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscountCampaignDescriptionValidation(t *testing.T) {
	input := DiscountCampaignInput{
		Name: "test", Description: strings.Repeat("a", 501), ActorID: 1,
		ScheduleType: DiscountScheduleOneTime, StartsAt: "2026-08-09T00:00:00Z",
		EndsAt: "2026-08-10T00:00:00Z", DiscountFactor: "0.9",
	}

	_, err := validateDiscountCampaignInput(input)
	require.Error(t, err)
}

func TestDiscountCampaignResolveExcludesSubscriptionAndExhaustedBudget(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	svc := &DiscountCampaignService{campaigns: []runtimeDiscountCampaign{{
		id: 1, scheduleType: DiscountScheduleOneTime, location: time.UTC,
		startsAt: &start, endsAt: &end, factor: 0.9, budgetCap: 10, discountSpent: 10,
	}}}

	require.Nil(t, svc.Resolve(&Group{SubscriptionType: SubscriptionTypeStandard}, now, 2))
	svc.campaigns[0].discountSpent = 0
	require.Nil(t, svc.Resolve(&Group{SubscriptionType: SubscriptionTypeSubscription}, now, 2))
}

func TestDiscountCampaignResolveAppliesMinimumMultiplier(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	svc := &DiscountCampaignService{campaigns: []runtimeDiscountCampaign{{
		id: 1, scheduleType: DiscountScheduleOneTime, location: time.UTC,
		startsAt: &start, endsAt: &end, factor: 0.5, minEffectiveMultiplier: 1.5,
	}}}

	resolved := svc.Resolve(&Group{SubscriptionType: SubscriptionTypeStandard}, now, 2)
	require.NotNil(t, resolved)
	require.InDelta(t, 1.5, resolved.EffectiveRateMultiplier, 1e-9)
	require.InDelta(t, 0.75, resolved.DiscountFactor, 1e-9)
}

func TestWeeklyDiscountCampaignMatchesCrossMidnightWindow(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	campaign := runtimeDiscountCampaign{
		scheduleType: DiscountScheduleWeekly,
		location:     location,
		weekdays:     map[time.Weekday]struct{}{time.Monday: {}},
		startMinute:  23 * 60,
		endMinute:    2 * 60,
	}

	require.True(t, campaign.matches(time.Date(2026, 8, 3, 23, 30, 0, 0, location)))
	require.True(t, campaign.matches(time.Date(2026, 8, 4, 1, 30, 0, 0, location)))
	require.False(t, campaign.matches(time.Date(2026, 8, 4, 2, 0, 0, 0, location)))
	require.False(t, campaign.matches(time.Date(2026, 8, 4, 23, 30, 0, 0, location)))
}

func TestApplyDiscountResolutionAuditsOnlyTokenBilling(t *testing.T) {
	campaignID := int64(9)
	resolution := &DiscountResolution{
		CampaignID: campaignID, DiscountFactor: 0.8,
		OriginalRateMultiplier: 2, EffectiveRateMultiplier: 1.6,
	}
	usage := &UsageLog{}
	applyDiscountResolutionToUsageLog(usage, &CostBreakdown{BillingMode: string(BillingModeImage), ActualCost: 8}, resolution)
	require.Nil(t, usage.DiscountCampaignID)

	applyDiscountResolutionToUsageLog(usage, &CostBreakdown{BillingMode: string(BillingModeToken), ActualCost: 8}, resolution)
	require.Equal(t, &campaignID, usage.DiscountCampaignID)
	require.InDelta(t, 2, usage.DiscountAmount, 1e-9)
}

func TestResolveOpenAIUsageTokenDiscountExcludesVideoEvenWithTokenPricing(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	svc := &DiscountCampaignService{campaigns: []runtimeDiscountCampaign{{
		id: 1, scheduleType: DiscountScheduleOneTime, location: time.UTC,
		startsAt: &start, endsAt: &end, factor: 0.9,
	}}}
	previous := defaultDiscountCampaignService.Swap(svc)
	t.Cleanup(func() { defaultDiscountCampaignService.Store(previous) })
	group := &Group{SubscriptionType: SubscriptionTypeStandard}

	video := &OpenAIForwardResult{Model: "grok-imagine-video", VideoCount: 1}
	require.Nil(t, resolveOpenAIUsageTokenDiscount(group, video, []string{"grok-imagine-video"}, now, 2))

	text := &OpenAIForwardResult{Model: "gpt-5"}
	resolved := resolveOpenAIUsageTokenDiscount(group, text, []string{"gpt-5"}, now, 2)
	require.NotNil(t, resolved)
	require.InDelta(t, 1.8, resolved.EffectiveRateMultiplier, 1e-9)
}

package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestResolveRechargePromotionPriorityAndAmount(t *testing.T) {
	amount := decimal.NewFromInt(100)
	result := ResolveRechargePromotion(RechargePromotionPricingInput{
		Amount: amount, Currency: "CNY", IsFirstRecharge: true,
		MembershipDiscount: decimal.NewFromInt(10),
	}, []RechargePromotionCandidate{
		{ID: 2, Kind: "recharge", Currency: "CNY", Enabled: true, DiscountPercent: decimal.NewFromInt(20)},
		{ID: 1, Kind: "first_recharge", Currency: "CNY", Enabled: true, DiscountPercent: decimal.NewFromInt(5)},
	})
	require.Equal(t, "first_recharge", result.Source)
	require.True(t, result.DiscountAmount.Equal(decimal.NewFromInt(5)))
	require.True(t, result.DiscountedAmount.Equal(decimal.NewFromInt(95)))
}

func TestResolveRechargePromotionBudgetAndBoundaries(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	pointer := func(value string) *decimal.Decimal { d := decimal.RequireFromString(value); return &d }
	base := RechargePromotionCandidate{ID: 1, Kind: "first_recharge", Currency: "CNY", Enabled: true, DiscountPercent: decimal.NewFromInt(10)}
	cases := []struct {
		name         string
		mutate       func(*RechargePromotionCandidate)
		wantSource   string
		wantID       int64
		wantDiscount string
	}{
		{"budget exact", func(c *RechargePromotionCandidate) { c.BudgetRemaining = pointer("10") }, "first_recharge", 1, "10"},
		{"budget insufficient", func(c *RechargePromotionCandidate) { c.BudgetRemaining = pointer("9.99") }, "campaign", 2, "5"},
		{"zero budget", func(c *RechargePromotionCandidate) { c.BudgetRemaining = pointer("0") }, "campaign", 2, "5"},
		{"cap makes affordable", func(c *RechargePromotionCandidate) { c.MaxDiscount = pointer("3"); c.BudgetRemaining = pointer("3") }, "first_recharge", 1, "3"},
		{"start inclusive", func(c *RechargePromotionCandidate) { c.StartsAt = &now }, "first_recharge", 1, "10"},
		{"end exclusive", func(c *RechargePromotionCandidate) { c.EndsAt = &now }, "campaign", 2, "5"},
		{"minimum inclusive", func(c *RechargePromotionCandidate) { c.MinAmount = pointer("100") }, "first_recharge", 1, "10"},
		{"maximum inclusive", func(c *RechargePromotionCandidate) { c.MaxAmount = pointer("100") }, "first_recharge", 1, "10"},
		{"below minimum", func(c *RechargePromotionCandidate) { c.MinAmount = pointer("101") }, "campaign", 2, "5"},
		{"above maximum", func(c *RechargePromotionCandidate) { c.MaxAmount = pointer("99") }, "campaign", 2, "5"},
		{"currency mismatch", func(c *RechargePromotionCandidate) { c.Currency = "USD" }, "campaign", 2, "5"},
		{"disabled", func(c *RechargePromotionCandidate) { c.Enabled = false }, "campaign", 2, "5"},
		{"unknown kind", func(c *RechargePromotionCandidate) { c.Kind = "other" }, "campaign", 2, "5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base
			tc.mutate(&candidate)
			result := ResolveRechargePromotion(RechargePromotionPricingInput{Amount: decimal.NewFromInt(100), Currency: "CNY", Now: now, IsFirstRecharge: true, MembershipDiscount: decimal.NewFromInt(30)}, []RechargePromotionCandidate{
				candidate, {ID: 2, Kind: "recharge", Currency: "CNY", Enabled: true, DiscountPercent: decimal.NewFromInt(5)},
			})
			require.Equal(t, tc.wantSource, result.Source)
			require.Equal(t, tc.wantID, result.PromotionID)
			require.Equal(t, tc.wantDiscount, result.DiscountAmount.String())
			require.True(t, result.DiscountAmount.Add(result.DiscountedAmount).Equal(result.OriginalAmount))
		})
	}
}

func TestRechargeQuoteCurrencyAndUnchangedCredit(t *testing.T) {
	for _, tc := range []struct{ currency, amount, discounted, fee, pay string }{
		{"CNY", "100", "90.00", "2.25", "92.25"},
		{"JPY", "101", "91", "3", "94"},
		{"KWD", "12.345", "11.110", "0.278", "11.388"},
	} {
		t.Run(tc.currency, func(t *testing.T) {
			pricing := ResolveRechargePromotion(RechargePromotionPricingInput{Amount: decimal.RequireFromString(tc.amount), Currency: tc.currency, MembershipDiscount: decimal.NewFromInt(10)}, nil)
			quote := buildRechargeQuote(pricing, tc.currency, 2.5, 0.14)
			require.Equal(t, tc.discounted, quote.DiscountedAmount)
			require.Equal(t, tc.fee, quote.FeeAmount)
			require.Equal(t, tc.pay, quote.PayAmount)
			require.Equal(t, decimal.NewFromFloat(calculateCreditedBalance(pricing.OriginalAmount.InexactFloat64(), 0.14)).StringFixed(2), quote.CreditedAmount)
			_, err := payment.AmountToMinorUnit(quote.PayAmount, tc.currency)
			require.NoError(t, err)
			withoutDiscount := ResolveRechargePromotion(RechargePromotionPricingInput{Amount: pricing.OriginalAmount, Currency: tc.currency}, nil)
			plain := buildRechargeQuote(withoutDiscount, tc.currency, 2.5, 0.14)
			require.Equal(t, payment.CalculatePayAmountForCurrency(pricing.OriginalAmount.InexactFloat64(), 2.5, tc.currency), plain.PayAmount)
		})
	}
}

func TestRechargeCampaignStableSelectionAndNoInputMutation(t *testing.T) {
	cap := decimal.NewFromInt(10)
	candidates := []RechargePromotionCandidate{
		{ID: 9, Kind: "recharge", Currency: "CNY", Enabled: true, DiscountPercent: decimal.NewFromInt(20), MaxDiscount: &cap},
		{ID: 2, Kind: "recharge", Currency: "CNY", Enabled: true, DiscountPercent: decimal.NewFromInt(10)},
	}
	before := append([]RechargePromotionCandidate(nil), candidates...)
	result := ResolveRechargePromotion(RechargePromotionPricingInput{Amount: decimal.NewFromInt(100), Currency: "CNY"}, candidates)
	require.Equal(t, int64(2), result.PromotionID)
	require.Equal(t, "campaign", result.Source)
	require.Equal(t, before, candidates)
}

func TestResolveRechargePromotionFallsBackToMembership(t *testing.T) {
	result := ResolveRechargePromotion(RechargePromotionPricingInput{
		Amount: decimal.NewFromInt(100), Currency: "CNY", MembershipDiscount: decimal.NewFromInt(10),
	}, nil)
	require.Equal(t, "membership", result.Source)
	require.True(t, result.DiscountedAmount.Equal(decimal.NewFromInt(90)))
}

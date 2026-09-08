package service

import (
	"testing"

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

func TestResolveRechargePromotionFallsBackToMembership(t *testing.T) {
	result := ResolveRechargePromotion(RechargePromotionPricingInput{
		Amount: decimal.NewFromInt(100), Currency: "CNY", MembershipDiscount: decimal.NewFromInt(10),
	}, nil)
	require.Equal(t, "membership", result.Source)
	require.True(t, result.DiscountedAmount.Equal(decimal.NewFromInt(90)))
}

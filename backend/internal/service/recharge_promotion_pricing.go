package service

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

// RechargePromotionPricingInput contains only server-owned values. The client
// must never be allowed to provide the resulting discounted amount.
type RechargePromotionPricingInput struct {
	Amount             decimal.Decimal
	Currency           string
	Now                time.Time
	IsFirstRecharge    bool
	MembershipDiscount decimal.Decimal
}

type RechargePromotionCandidate struct {
	ID              int64
	Kind            string // first_recharge or recharge
	Currency        string
	Enabled         bool
	StartsAt        *time.Time
	EndsAt          *time.Time
	MinAmount       *decimal.Decimal
	MaxAmount       *decimal.Decimal
	DiscountPercent decimal.Decimal
	MaxDiscount     *decimal.Decimal
	BudgetRemaining *decimal.Decimal
}

type RechargePromotionPricingResult struct {
	OriginalAmount   decimal.Decimal
	DiscountAmount   decimal.Decimal
	DiscountedAmount decimal.Decimal
	Source           string
	PromotionID      int64
}

// ResolveRechargePromotion applies the fixed priority first_recharge >
// recharge campaign > membership. It returns a deterministic result and
// never changes the amount credited to the user's balance.
func ResolveRechargePromotion(input RechargePromotionPricingInput, candidates []RechargePromotionCandidate) RechargePromotionPricingResult {
	result := RechargePromotionPricingResult{OriginalAmount: input.Amount, DiscountedAmount: input.Amount}
	if input.Amount.LessThanOrEqual(decimal.Zero) || input.Currency == "" {
		return result
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	digits := int32(payment.CurrencyMaxFractionDigits(input.Currency))
	bestPriority := 0
	for _, candidate := range candidates {
		if !candidate.Enabled || candidate.Currency != input.Currency || candidate.DiscountPercent.LessThanOrEqual(decimal.Zero) || candidate.DiscountPercent.GreaterThanOrEqual(decimal.NewFromInt(100)) {
			continue
		}
		if candidate.StartsAt != nil && now.Before(*candidate.StartsAt) || candidate.EndsAt != nil && !now.Before(*candidate.EndsAt) {
			continue
		}
		if candidate.MinAmount != nil && input.Amount.LessThan(*candidate.MinAmount) || candidate.MaxAmount != nil && input.Amount.GreaterThan(*candidate.MaxAmount) {
			continue
		}
		priority := 1
		source := "campaign"
		switch candidate.Kind {
		case "first_recharge":
			if !input.IsFirstRecharge {
				continue
			}
			priority, source = 2, "first_recharge"
		case "recharge":
		default:
			continue
		}
		discount := input.Amount.Mul(candidate.DiscountPercent).Div(decimal.NewFromInt(100))
		if candidate.MaxDiscount != nil {
			discount = decimal.Min(discount, *candidate.MaxDiscount)
		}
		discount = discount.Round(digits)
		if discount.LessThanOrEqual(decimal.Zero) || discount.GreaterThanOrEqual(input.Amount) {
			continue
		}
		if candidate.BudgetRemaining != nil && discount.GreaterThan(*candidate.BudgetRemaining) {
			continue
		}
		if priority > bestPriority || (priority == bestPriority && (discount.GreaterThan(result.DiscountAmount) || (discount.Equal(result.DiscountAmount) && candidate.ID < result.PromotionID))) {
			bestPriority = priority
			result.DiscountAmount = discount
			result.DiscountedAmount = input.Amount.Sub(discount).Round(digits)
			result.Source = source
			result.PromotionID = candidate.ID
		}
	}
	if bestPriority > 0 {
		return result
	}
	if input.MembershipDiscount.GreaterThan(decimal.Zero) && input.MembershipDiscount.LessThan(decimal.NewFromInt(100)) {
		discount := input.Amount.Mul(input.MembershipDiscount).Div(decimal.NewFromInt(100)).Round(digits)
		if discount.GreaterThan(decimal.Zero) && discount.LessThan(input.Amount) {
			result.DiscountAmount = discount
			result.DiscountedAmount = input.Amount.Sub(discount).Round(digits)
			result.Source = "membership"
		}
	}
	return result
}

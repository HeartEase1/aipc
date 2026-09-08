package service

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// RechargePromotionPricingInput contains only server-owned values. The client
// must never be allowed to provide the resulting discounted amount.
type RechargePromotionPricingInput struct {
	Amount             decimal.Decimal
	Currency           string
	Now                time.Time
	IsFirstRecharge   bool
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
	OriginalAmount  decimal.Decimal
	DiscountAmount  decimal.Decimal
	DiscountedAmount decimal.Decimal
	Source          string
	PromotionID     int64
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
	valid := make([]RechargePromotionCandidate, 0, len(candidates))
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
		if candidate.BudgetRemaining != nil && candidate.BudgetRemaining.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if candidate.Kind == "first_recharge" && !input.IsFirstRecharge {
			continue
		}
		if candidate.Kind == "recharge" && input.IsFirstRecharge {
			// A first-recharge candidate has priority, but ordinary campaigns
			// remain eligible when no first-recharge candidate matches.
		}
		valid = append(valid, candidate)
	}
	first := make([]RechargePromotionCandidate, 0)
	ordinary := make([]RechargePromotionCandidate, 0)
	for _, candidate := range valid {
		if candidate.Kind == "first_recharge" {
			first = append(first, candidate)
		} else if candidate.Kind == "recharge" {
			ordinary = append(ordinary, candidate)
		}
	}
	pool := ordinary
	if len(first) > 0 {
		pool = first
	}
	if len(pool) > 0 {
		sort.SliceStable(pool, func(i, j int) bool {
			left := pool[i].DiscountPercent
			right := pool[j].DiscountPercent
			if pool[i].MaxDiscount != nil {
				left = decimal.Min(left, pool[i].MaxDiscount.Div(input.Amount).Mul(decimal.NewFromInt(100)))
			}
			if pool[j].MaxDiscount != nil {
				right = decimal.Min(right, pool[j].MaxDiscount.Div(input.Amount).Mul(decimal.NewFromInt(100)))
			}
			if !left.Equal(right) {
				return left.GreaterThan(right)
			}
			return pool[i].ID < pool[j].ID
		})
		candidate := pool[0]
		discount := input.Amount.Mul(candidate.DiscountPercent).Div(decimal.NewFromInt(100))
		if candidate.MaxDiscount != nil {
			discount = decimal.Min(discount, *candidate.MaxDiscount)
		}
		result.DiscountAmount = discount.Round(2)
		result.DiscountedAmount = input.Amount.Sub(result.DiscountAmount).Round(2)
		result.Source = candidate.Kind
		result.PromotionID = candidate.ID
		return result
	}
	if input.MembershipDiscount.GreaterThan(decimal.Zero) && input.MembershipDiscount.LessThan(decimal.NewFromInt(100)) {
		result.DiscountAmount = input.Amount.Mul(input.MembershipDiscount).Div(decimal.NewFromInt(100)).Round(2)
		result.DiscountedAmount = input.Amount.Sub(result.DiscountAmount).Round(2)
		result.Source = "membership"
	}
	return result
}

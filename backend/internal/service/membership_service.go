package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type MembershipSummary struct {
	Enabled               bool   `json:"enabled"`
	SettlementCurrency    string `json:"settlement_currency"`
	CurrentAmount         string `json:"current_amount"`
	CurrentTier           string `json:"current_tier,omitempty"`
	CurrentDiscount       string `json:"current_discount_percent"`
	NextTier              string `json:"next_tier,omitempty"`
	NextThreshold         string `json:"next_threshold,omitempty"`
	AmountToNext          string `json:"amount_to_next,omitempty"`
	ProgressPercent       string `json:"progress_percent"`
	FirstRechargeEligible bool   `json:"first_recharge_eligible"`
}

type RechargeQuote struct {
	OriginalAmount   string `json:"original_amount"`
	DiscountAmount   string `json:"discount_amount"`
	DiscountedAmount string `json:"discounted_amount"`
	FeeAmount        string `json:"fee_amount"`
	PayAmount        string `json:"pay_amount"`
	CreditedAmount   string `json:"credited_amount"`
	Currency         string `json:"currency"`
	DiscountSource   string `json:"discount_source,omitempty"`
	PromotionID      int64  `json:"promotion_id,omitempty"`
}

func (s *PaymentService) QuoteRecharge(ctx context.Context, userID int64, amount decimal.Decimal, paymentType string) (RechargeQuote, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return RechargeQuote{}, errors.New("amount must be positive")
	}
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return RechargeQuote{}, err
	}
	currency, err := s.configService.ValidateMethodCurrencyConsistency(ctx, paymentType)
	if err != nil {
		return RechargeQuote{}, err
	}
	membership, err := s.GetMembershipSummary(ctx, userID)
	if err != nil {
		return RechargeQuote{}, err
	}
	membershipDiscount, _ := decimal.NewFromString(membership.CurrentDiscount)
	candidates, err := s.listRechargePromotionCandidates(ctx, currency)
	if err != nil {
		return RechargeQuote{}, err
	}
	pricing := ResolveRechargePromotion(RechargePromotionPricingInput{Amount: amount, Currency: currency, IsFirstRecharge: membership.FirstRechargeEligible, MembershipDiscount: membershipDiscount}, candidates)
	feeRate := decimal.NewFromFloat(cfg.RechargeFeeRate)
	fee := pricing.DiscountedAmount.Mul(feeRate).Div(decimal.NewFromInt(100)).Round(2)
	pay := pricing.DiscountedAmount.Add(fee).Round(2)
	credited := amount.Mul(decimal.NewFromFloat(cfg.BalanceRechargeMultiplier)).Round(2)
	return RechargeQuote{OriginalAmount: amount.StringFixed(2), DiscountAmount: pricing.DiscountAmount.StringFixed(2), DiscountedAmount: pricing.DiscountedAmount.StringFixed(2), FeeAmount: fee.StringFixed(2), PayAmount: pay.StringFixed(2), CreditedAmount: credited.StringFixed(2), Currency: currency, DiscountSource: pricing.Source, PromotionID: pricing.PromotionID}, nil
}

func (s *PaymentService) listRechargePromotionCandidates(ctx context.Context, currency string) ([]RechargePromotionCandidate, error) {
	if s.sqlDB == nil {
		return nil, nil
	}
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT id, kind, settlement_currency, enabled, starts_at, ends_at, min_amount, max_amount, discount_percent, max_discount_amount, GREATEST(0, COALESCE(budget_amount, 999999999999) - reserved_amount - redeemed_amount) FROM recharge_promotions WHERE enabled = TRUE AND kind IN ('recharge', 'first_recharge') AND settlement_currency = $1`, currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RechargePromotionCandidate
	for rows.Next() {
		var item RechargePromotionCandidate
		var starts, ends *time.Time
		var minAmount, maxAmount, maxDiscount, budget *string
		if err := rows.Scan(&item.ID, &item.Kind, &item.Currency, &item.Enabled, &starts, &ends, &minAmount, &maxAmount, &item.DiscountPercent, &maxDiscount, &budget); err != nil {
			return nil, err
		}
		item.StartsAt, item.EndsAt = starts, ends
		parseOptional := func(raw *string) (*decimal.Decimal, error) {
			if raw == nil {
				return nil, nil
			}
			value, err := decimal.NewFromString(*raw)
			if err != nil {
				return nil, err
			}
			return &value, nil
		}
		if item.MinAmount, err = parseOptional(minAmount); err != nil {
			return nil, err
		}
		if item.MaxAmount, err = parseOptional(maxAmount); err != nil {
			return nil, err
		}
		if item.MaxDiscount, err = parseOptional(maxDiscount); err != nil {
			return nil, err
		}
		if item.BudgetRemaining, err = parseOptional(budget); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// GetMembershipSummary reads only completed balance orders. Subscription,
// redeem, grant and affiliate records are excluded by order_type.
func (s *PaymentService) GetMembershipSummary(ctx context.Context, userID int64) (MembershipSummary, error) {
	if s == nil || s.sqlDB == nil {
		return MembershipSummary{SettlementCurrency: "CNY", FirstRechargeEligible: true}, nil
	}
	const currency = "CNY"
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	var amount, refunded string
	err := s.sqlDB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(pay_amount), 0), COALESCE(SUM(refund_amount), 0)
		FROM payment_orders
		WHERE user_id = $1 AND order_type = 'balance' AND settlement_currency = $2
		  AND status IN ('COMPLETED', 'REFUNDED', 'PARTIALLY_REFUNDED') AND paid_at >= $3`, userID, currency, cutoff).Scan(&amount, &refunded)
	if err != nil {
		return MembershipSummary{}, err
	}
	paid, err := decimal.NewFromString(amount)
	if err != nil {
		return MembershipSummary{}, err
	}
	refund, err := decimal.NewFromString(refunded)
	if err != nil {
		return MembershipSummary{}, err
	}
	current := paid.Sub(refund).Round(8)
	var firstCount int
	if err := s.sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_orders WHERE user_id = $1 AND order_type = 'balance' AND status IN ('COMPLETED', 'REFUNDED', 'PARTIALLY_REFUNDED')`, userID).Scan(&firstCount); err != nil {
		return MembershipSummary{}, err
	}
	result := MembershipSummary{Enabled: false, SettlementCurrency: currency, CurrentAmount: current.StringFixed(8), CurrentDiscount: "0", ProgressPercent: "0", FirstRechargeEligible: firstCount == 0}
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT name, threshold_amount, discount_percent FROM balance_membership_tiers WHERE enabled = TRUE AND settlement_currency = $1 ORDER BY threshold_amount ASC, sort_order ASC, id ASC`, currency)
	if err != nil {
		return MembershipSummary{}, err
	}
	defer rows.Close()
	result.Enabled = true
	type tier struct {
		name                string
		threshold, discount decimal.Decimal
	}
	var tiers []tier
	for rows.Next() {
		var name, threshold, discount string
		if err := rows.Scan(&name, &threshold, &discount); err != nil {
			return MembershipSummary{}, err
		}
		thresholdDecimal, err := decimal.NewFromString(threshold)
		if err != nil {
			return MembershipSummary{}, err
		}
		discountDecimal, err := decimal.NewFromString(discount)
		if err != nil {
			return MembershipSummary{}, err
		}
		tiers = append(tiers, tier{name, thresholdDecimal, discountDecimal})
	}
	if err := rows.Err(); err != nil {
		return MembershipSummary{}, err
	}
	var next *tier
	for i := range tiers {
		if current.GreaterThanOrEqual(tiers[i].threshold) {
			result.CurrentTier = tiers[i].name
			result.CurrentDiscount = tiers[i].discount.StringFixed(4)
		} else if next == nil {
			next = &tiers[i]
		}
	}
	if next != nil {
		result.NextTier = next.name
		result.NextThreshold = next.threshold.StringFixed(8)
		result.AmountToNext = decimal.Max(next.threshold.Sub(current), decimal.Zero).StringFixed(8)
		if next.threshold.GreaterThan(decimal.Zero) {
			result.ProgressPercent = decimal.Min(current.Div(next.threshold).Mul(decimal.NewFromInt(100)), decimal.NewFromInt(100)).Round(2).StringFixed(2)
		}
	} else if len(tiers) > 0 {
		result.ProgressPercent = "100"
	}
	return result, nil
}

var errMembershipUnavailable = errors.New("membership service unavailable")

func (s *PaymentService) membershipDB() (*sql.DB, error) {
	if s == nil || s.sqlDB == nil {
		return nil, errMembershipUnavailable
	}
	return s.sqlDB, nil
}

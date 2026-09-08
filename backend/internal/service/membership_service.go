package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type MembershipSummary struct {
	Enabled            bool   `json:"enabled"`
	SettlementCurrency string `json:"settlement_currency"`
	CurrentAmount      string `json:"current_amount"`
	CurrentTier        string `json:"current_tier,omitempty"`
	CurrentDiscount    string `json:"current_discount_percent"`
	NextTier           string `json:"next_tier,omitempty"`
	NextThreshold      string `json:"next_threshold,omitempty"`
	AmountToNext       string `json:"amount_to_next,omitempty"`
	ProgressPercent    string `json:"progress_percent"`
	FirstRechargeEligible bool `json:"first_recharge_eligible"`
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
	if err != nil { return MembershipSummary{}, err }
	refund, err := decimal.NewFromString(refunded)
	if err != nil { return MembershipSummary{}, err }
	current := paid.Sub(refund).Round(8)
	var firstCount int
	if err := s.sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_orders WHERE user_id = $1 AND order_type = 'balance' AND status IN ('COMPLETED', 'REFUNDED', 'PARTIALLY_REFUNDED')`, userID).Scan(&firstCount); err != nil {
		return MembershipSummary{}, err
	}
	result := MembershipSummary{Enabled: false, SettlementCurrency: currency, CurrentAmount: current.StringFixed(8), CurrentDiscount: "0", ProgressPercent: "0", FirstRechargeEligible: firstCount == 0}
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT name, threshold_amount, discount_percent FROM balance_membership_tiers WHERE enabled = TRUE AND settlement_currency = $1 ORDER BY threshold_amount ASC, sort_order ASC, id ASC`, currency)
	if err != nil { return MembershipSummary{}, err }
	defer rows.Close()
	result.Enabled = true
	type tier struct { name string; threshold, discount decimal.Decimal }
	var tiers []tier
	for rows.Next() {
		var name, threshold, discount string
		if err := rows.Scan(&name, &threshold, &discount); err != nil { return MembershipSummary{}, err }
		thresholdDecimal, err := decimal.NewFromString(threshold); if err != nil { return MembershipSummary{}, err }
		discountDecimal, err := decimal.NewFromString(discount); if err != nil { return MembershipSummary{}, err }
		tiers = append(tiers, tier{name, thresholdDecimal, discountDecimal})
	}
	if err := rows.Err(); err != nil { return MembershipSummary{}, err }
	var next *tier
	for i := range tiers {
		if current.GreaterThanOrEqual(tiers[i].threshold) {
			result.CurrentTier = tiers[i].name
			result.CurrentDiscount = tiers[i].discount.StringFixed(4)
		} else if next == nil { next = &tiers[i] }
	}
	if next != nil {
		result.NextTier = next.name
		result.NextThreshold = next.threshold.StringFixed(8)
		result.AmountToNext = decimal.Max(next.threshold.Sub(current), decimal.Zero).StringFixed(8)
		if next.threshold.GreaterThan(decimal.Zero) { result.ProgressPercent = decimal.Min(current.Div(next.threshold).Mul(decimal.NewFromInt(100)), decimal.NewFromInt(100)).Round(2).StringFixed(2) }
	} else if len(tiers) > 0 { result.ProgressPercent = "100" }
	return result, nil
}

var errMembershipUnavailable = errors.New("membership service unavailable")

func (s *PaymentService) membershipDB() (*sql.DB, error) {
	if s == nil || s.sqlDB == nil { return nil, errMembershipUnavailable }
	return s.sqlDB, nil
}

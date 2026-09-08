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

type MembershipTierAdminInput struct {
	Name               string `json:"name"`
	SettlementCurrency string `json:"settlement_currency"`
	ThresholdAmount    string `json:"threshold_amount"`
	DiscountPercent    string `json:"discount_percent"`
	SortOrder          int    `json:"sort_order"`
	Enabled            bool   `json:"enabled"`
}

type RechargePromotionAdminInput struct {
	Kind               string     `json:"kind"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	SettlementCurrency string     `json:"settlement_currency"`
	Enabled            bool       `json:"enabled"`
	StartsAt           *time.Time `json:"starts_at,omitempty"`
	EndsAt             *time.Time `json:"ends_at,omitempty"`
	Timezone           string     `json:"timezone"`
	MinAmount          *string    `json:"min_amount,omitempty"`
	MaxAmount          *string    `json:"max_amount,omitempty"`
	DiscountPercent    string     `json:"discount_percent"`
	MaxDiscountAmount  *string    `json:"max_discount_amount,omitempty"`
	BudgetAmount       *string    `json:"budget_amount,omitempty"`
}

func (s *PaymentService) ListMembershipTiers(ctx context.Context) ([]map[string]any, error) {
	if s == nil || s.sqlDB == nil {
		return []map[string]any{}, nil
	}
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT id, name, settlement_currency, threshold_amount, discount_percent, sort_order, enabled, created_at, updated_at FROM balance_membership_tiers ORDER BY settlement_currency, threshold_amount ASC, sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var name, currency, threshold, discount string
		var order int
		var enabled bool
		var created, updated time.Time
		if err := rows.Scan(&id, &name, &currency, &threshold, &discount, &order, &enabled, &created, &updated); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "name": name, "settlement_currency": currency, "threshold_amount": threshold, "discount_percent": discount, "sort_order": order, "enabled": enabled, "created_at": created, "updated_at": updated})
	}
	return out, rows.Err()
}

func (s *PaymentService) CreateMembershipTier(ctx context.Context, in MembershipTierAdminInput) (int64, error) {
	if s == nil || s.sqlDB == nil {
		return 0, errMembershipUnavailable
	}
	threshold, err := decimal.NewFromString(in.ThresholdAmount)
	if err != nil || threshold.LessThan(decimal.Zero) {
		return 0, errors.New("invalid threshold_amount")
	}
	discount, err := decimal.NewFromString(in.DiscountPercent)
	if err != nil || discount.LessThan(decimal.Zero) || discount.GreaterThanOrEqual(decimal.NewFromInt(100)) {
		return 0, errors.New("invalid discount_percent")
	}
	var id int64
	err = s.sqlDB.QueryRowContext(ctx, `INSERT INTO balance_membership_tiers (name, settlement_currency, threshold_amount, discount_percent, sort_order, enabled) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, in.Name, in.SettlementCurrency, threshold.String(), discount.String(), in.SortOrder, in.Enabled).Scan(&id)
	return id, err
}

func (s *PaymentService) UpdateMembershipTier(ctx context.Context, id int64, in MembershipTierAdminInput) error {
	if s == nil || s.sqlDB == nil {
		return errMembershipUnavailable
	}
	threshold, err := decimal.NewFromString(in.ThresholdAmount)
	if err != nil || threshold.LessThan(decimal.Zero) {
		return errors.New("invalid threshold_amount")
	}
	discount, err := decimal.NewFromString(in.DiscountPercent)
	if err != nil || discount.LessThan(decimal.Zero) || discount.GreaterThanOrEqual(decimal.NewFromInt(100)) {
		return errors.New("invalid discount_percent")
	}
	result, err := s.sqlDB.ExecContext(ctx, `UPDATE balance_membership_tiers SET name=$1, settlement_currency=$2, threshold_amount=$3, discount_percent=$4, sort_order=$5, enabled=$6, updated_at=NOW() WHERE id=$7`, in.Name, in.SettlementCurrency, threshold.String(), discount.String(), in.SortOrder, in.Enabled, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *PaymentService) DeleteMembershipTier(ctx context.Context, id int64) error {
	if s == nil || s.sqlDB == nil {
		return errMembershipUnavailable
	}
	result, err := s.sqlDB.ExecContext(ctx, `DELETE FROM balance_membership_tiers WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *PaymentService) reserveRechargePromotion(ctx context.Context, userID int64, quote *RechargeQuote) (int64, error) {
	if s == nil || s.sqlDB == nil || quote == nil || quote.PromotionID <= 0 || quote.DiscountAmount == "0.00" {
		return 0, nil
	}
	discount, err := decimal.NewFromString(quote.DiscountAmount)
	if err != nil || discount.LessThanOrEqual(decimal.Zero) {
		return 0, err
	}
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var updated int64
	err = tx.QueryRowContext(ctx, `UPDATE recharge_promotions SET reserved_amount = reserved_amount + $1, updated_at = NOW() WHERE id = $2 AND enabled = TRUE AND (budget_amount IS NULL OR reserved_amount + redeemed_amount + $1 <= budget_amount) RETURNING id`, discount.String(), quote.PromotionID).Scan(&updated)
	if err != nil {
		return 0, err
	}
	var claimID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO recharge_promotion_claims (promotion_id, user_id, source, discount_amount) VALUES ($1, $2, $3, $4) RETURNING id`, quote.PromotionID, userID, quote.DiscountSource, discount.String()).Scan(&claimID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return claimID, nil
}

func (s *PaymentService) bindRechargePromotionOrder(ctx context.Context, claimID, orderID int64) error {
	if claimID <= 0 || orderID <= 0 || s == nil || s.sqlDB == nil {
		return nil
	}
	_, err := s.sqlDB.ExecContext(ctx, `UPDATE recharge_promotion_claims SET order_id = $1 WHERE id = $2 AND status = 'reserved'`, orderID, claimID)
	return err
}

func (s *PaymentService) releaseRechargePromotion(ctx context.Context, claimID int64) error {
	if claimID <= 0 || s == nil || s.sqlDB == nil {
		return nil
	}
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var promotionID int64
	var discount string
	err = tx.QueryRowContext(ctx, `UPDATE recharge_promotion_claims SET status = 'released', released_at = NOW() WHERE id = $1 AND status = 'reserved' RETURNING promotion_id, discount_amount`, claimID).Scan(&promotionID, &discount)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE recharge_promotions SET reserved_amount = GREATEST(0, reserved_amount - $1), updated_at = NOW() WHERE id = $2`, discount, promotionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PaymentService) redeemRechargePromotionByOrder(ctx context.Context, orderID int64) error {
	if orderID <= 0 || s == nil || s.sqlDB == nil {
		return nil
	}
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var promotionID int64
	var discount string
	err = tx.QueryRowContext(ctx, `UPDATE recharge_promotion_claims SET status = 'redeemed', redeemed_at = NOW() WHERE order_id = $1 AND status = 'reserved' RETURNING promotion_id, discount_amount`, orderID).Scan(&promotionID, &discount)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE recharge_promotions SET reserved_amount = GREATEST(0, reserved_amount - $1), redeemed_amount = redeemed_amount + $1, updated_at = NOW() WHERE id = $2`, discount, promotionID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PaymentService) releaseRechargePromotionByOrder(ctx context.Context, orderID int64) error {
	if orderID <= 0 || s == nil || s.sqlDB == nil {
		return nil
	}
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var promotionID int64
	var discount string
	err = tx.QueryRowContext(ctx, `UPDATE recharge_promotion_claims SET status = 'released', released_at = NOW() WHERE order_id = $1 AND status = 'reserved' RETURNING promotion_id, discount_amount`, orderID).Scan(&promotionID, &discount)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE recharge_promotions SET reserved_amount = GREATEST(0, reserved_amount - $1), updated_at = NOW() WHERE id = $2`, discount, promotionID); err != nil {
		return err
	}
	return tx.Commit()
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

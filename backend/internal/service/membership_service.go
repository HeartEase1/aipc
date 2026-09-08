package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

type MembershipSummary struct {
	Rules                 MembershipRules `json:"rules"`
	Eligible              bool            `json:"-"`
	Enabled               bool            `json:"enabled"`
	SettlementCurrency    string          `json:"settlement_currency"`
	CurrentAmount         string          `json:"current_amount"`
	CurrentTier           string          `json:"current_tier,omitempty"`
	CurrentDiscount       string          `json:"current_discount_percent"`
	NextTier              string          `json:"next_tier,omitempty"`
	NextThreshold         string          `json:"next_threshold,omitempty"`
	AmountToNext          string          `json:"amount_to_next,omitempty"`
	ProgressPercent       string          `json:"progress_percent"`
	FirstRechargeEligible bool            `json:"first_recharge_eligible"`
}

type MembershipRules struct {
	WindowHours             int                 `json:"window_hours"`
	Priority                []string            `json:"priority"`
	SettlementCurrency      string              `json:"settlement_currency"`
	AffiliateCommissionRate string              `json:"affiliate_commission_rate"`
	Tiers                   []map[string]string `json:"tiers"`
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
	defer func() { _ = rows.Close() }()
	out := make([]map[string]any, 0)
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
	in, err := validateMembershipTierInput(in)
	if err != nil {
		return 0, err
	}
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize rule edits, including two concurrent inserts into an empty set.
	if _, err := tx.ExecContext(ctx, `LOCK TABLE balance_membership_tiers IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return 0, err
	}
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO balance_membership_tiers (name, settlement_currency, threshold_amount, discount_percent, sort_order, enabled) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, in.Name, in.SettlementCurrency, in.ThresholdAmount, in.DiscountPercent, in.SortOrder, in.Enabled).Scan(&id)
	if err != nil {
		return 0, err
	}
	if err := validateMembershipTierOrder(ctx, tx, in.SettlementCurrency); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *PaymentService) UpdateMembershipTier(ctx context.Context, id int64, in MembershipTierAdminInput) error {
	if s == nil || s.sqlDB == nil {
		return errMembershipUnavailable
	}
	in, err := validateMembershipTierInput(in)
	if err != nil {
		return err
	}
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `LOCK TABLE balance_membership_tiers IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE balance_membership_tiers SET name=$1, settlement_currency=$2, threshold_amount=$3, discount_percent=$4, sort_order=$5, enabled=$6, updated_at=NOW() WHERE id=$7`, in.Name, in.SettlementCurrency, in.ThresholdAmount, in.DiscountPercent, in.SortOrder, in.Enabled, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return infraerrors.NotFound("MEMBERSHIP_TIER_NOT_FOUND", "membership tier not found")
	}
	if err := validateMembershipTierOrder(ctx, tx, in.SettlementCurrency); err != nil {
		return err
	}
	return tx.Commit()
}

func validateMembershipTierInput(in MembershipTierAdminInput) (MembershipTierAdminInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 64 {
		return in, infraerrors.BadRequest("INVALID_MEMBERSHIP_TIER", "name must contain 1 to 64 characters")
	}
	currency, err := payment.NormalizePaymentCurrency(in.SettlementCurrency)
	if err != nil {
		return in, infraerrors.BadRequest("INVALID_MEMBERSHIP_CURRENCY", err.Error())
	}
	in.SettlementCurrency = currency
	if len(in.ThresholdAmount) > 32 || len(in.DiscountPercent) > 16 || strings.ContainsAny(in.ThresholdAmount+in.DiscountPercent, "eE") {
		return in, infraerrors.BadRequest("INVALID_MEMBERSHIP_AMOUNT", "amount is too long")
	}
	threshold, err := decimal.NewFromString(in.ThresholdAmount)
	if err != nil || threshold.IsNegative() || threshold.GreaterThanOrEqual(decimal.New(1, 12)) || !threshold.Equal(threshold.Truncate(8)) {
		return in, infraerrors.BadRequest("INVALID_MEMBERSHIP_THRESHOLD", "threshold must be nonnegative with at most 8 decimal places")
	}
	discount, err := decimal.NewFromString(in.DiscountPercent)
	if err != nil || discount.IsNegative() || discount.GreaterThanOrEqual(decimal.NewFromInt(100)) || !discount.Equal(discount.Truncate(4)) {
		return in, infraerrors.BadRequest("INVALID_MEMBERSHIP_DISCOUNT", "discount must be between 0 and 100 percent, exclusive of 100, with at most 4 decimal places")
	}
	in.ThresholdAmount, in.DiscountPercent = threshold.String(), discount.String()
	return in, nil
}

func validateMembershipTierOrder(ctx context.Context, tx *sql.Tx, currency string) error {
	rows, err := tx.QueryContext(ctx, `SELECT threshold_amount, discount_percent FROM balance_membership_tiers WHERE settlement_currency = $1 ORDER BY threshold_amount ASC, id ASC`, currency)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	var previousThreshold, previousDiscount decimal.Decimal
	hasPrevious := false
	for rows.Next() {
		var threshold, discount decimal.Decimal
		if err := rows.Scan(&threshold, &discount); err != nil {
			return err
		}
		if hasPrevious && (!threshold.GreaterThan(previousThreshold) || discount.LessThan(previousDiscount)) {
			return infraerrors.BadRequest("INVALID_MEMBERSHIP_TIER_ORDER", "thresholds must strictly increase and higher tiers cannot offer lower discounts")
		}
		previousThreshold, previousDiscount, hasPrevious = threshold, discount, true
	}
	return rows.Err()
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

func (s *PaymentService) QuoteRecharge(ctx context.Context, userID int64, amount decimal.Decimal, paymentType string) (RechargeQuote, error) {
	return s.quoteRecharge(ctx, userID, amount, paymentType, false)
}

func (s *PaymentService) quoteRecharge(ctx context.Context, userID int64, amount decimal.Decimal, paymentType string, skipPromotions bool) (RechargeQuote, error) {
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
	return s.quoteRechargeForConfig(ctx, userID, amount, currency, cfg, skipPromotions)
}

func (s *PaymentService) quoteRechargeForConfig(ctx context.Context, userID int64, amount decimal.Decimal, currency string, cfg *PaymentConfig, skipPromotions bool) (RechargeQuote, error) {
	if !cfg.Enabled || cfg.BalanceDisabled {
		return RechargeQuote{}, infraerrors.Forbidden("BALANCE_PAYMENT_DISABLED", "balance payment is disabled")
	}
	if !amount.IsPositive() || amount.GreaterThanOrEqual(decimal.New(1, 12)) || (cfg.MinAmount > 0 && amount.LessThan(decimal.NewFromFloat(cfg.MinAmount))) || (cfg.MaxAmount > 0 && amount.GreaterThan(decimal.NewFromFloat(cfg.MaxAmount))) {
		return RechargeQuote{}, infraerrors.BadRequest("INVALID_AMOUNT", "amount out of range")
	}
	if _, err := payment.AmountToMinorUnit(amount.String(), currency); err != nil {
		return RechargeQuote{}, err
	}
	membership, err := s.GetMembershipSummary(ctx, userID)
	if err != nil {
		return RechargeQuote{}, err
	}
	membershipDiscount := decimal.Zero
	if !skipPromotions {
		membershipDiscount, _ = decimal.NewFromString(membership.CurrentDiscount)
	}
	// Marketing exclusions are enforced server-side for both quote and order creation.
	if !skipPromotions {
		var excluded bool
		if err := s.sqlDB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM marketing_user_exclusions WHERE user_id=$1 AND enabled=TRUE AND scope IN ('all','membership'))`, userID).Scan(&excluded); err != nil {
			return RechargeQuote{}, err
		}
		if excluded {
			membershipDiscount = decimal.Zero
		}
	}
	var candidates []RechargePromotionCandidate
	if membership.Eligible && currency == membership.SettlementCurrency && !skipPromotions {
		candidates, err = s.listRechargePromotionCandidates(ctx, currency)
		if err != nil {
			return RechargeQuote{}, err
		}
	} else if !membership.Eligible || currency != membership.SettlementCurrency {
		membershipDiscount = decimal.Zero
	}
	if !skipPromotions {
		var excluded bool
		if err := s.sqlDB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM marketing_user_exclusions WHERE user_id=$1 AND enabled=TRUE AND scope IN ('all','recharge'))`, userID).Scan(&excluded); err != nil {
			return RechargeQuote{}, err
		}
		if excluded {
			candidates = nil
			membershipDiscount = decimal.Zero
		}
	}
	pricing := ResolveRechargePromotion(RechargePromotionPricingInput{Amount: amount, Currency: currency, IsFirstRecharge: membership.FirstRechargeEligible, MembershipDiscount: membershipDiscount}, candidates)
	return buildRechargeQuote(pricing, currency, cfg.RechargeFeeRate, cfg.BalanceRechargeMultiplier), nil
}

func buildRechargeQuote(pricing RechargePromotionPricingResult, currency string, rechargeFeeRate, multiplier float64) RechargeQuote {
	digits := int32(payment.CurrencyMaxFractionDigits(currency))
	feeRate := decimal.NewFromFloat(rechargeFeeRate)
	fee := decimal.Zero
	if feeRate.IsPositive() {
		fee = pricing.DiscountedAmount.Mul(feeRate).Div(decimal.NewFromInt(100)).RoundUp(digits)
	}
	pay := pricing.DiscountedAmount.Add(fee).Round(digits)
	credited := pricing.OriginalAmount.Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(multiplier))).Round(2)
	return RechargeQuote{OriginalAmount: pricing.OriginalAmount.StringFixed(digits), DiscountAmount: pricing.DiscountAmount.StringFixed(digits), DiscountedAmount: pricing.DiscountedAmount.StringFixed(digits), FeeAmount: fee.StringFixed(digits), PayAmount: pay.StringFixed(digits), CreditedAmount: credited.StringFixed(2), Currency: currency, DiscountSource: pricing.Source, PromotionID: pricing.PromotionID}
}

func (s *PaymentService) listRechargePromotionCandidates(ctx context.Context, currency string) ([]RechargePromotionCandidate, error) {
	if s.sqlDB == nil {
		return nil, nil
	}
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT id, kind, settlement_currency, enabled, starts_at, ends_at, min_amount, max_amount, discount_percent, max_discount_amount, CASE WHEN budget_amount IS NULL THEN NULL ELSE GREATEST(0, budget_amount - reserved_amount - redeemed_amount) END FROM recharge_promotions WHERE enabled = TRUE AND kind IN ('recharge', 'first_recharge') AND settlement_currency = $1`, currency)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
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
	cfg, err := s.GetBalanceMarketingConfig(ctx)
	if err != nil {
		return MembershipSummary{}, err
	}
	currency := cfg.SettlementCurrency
	result := MembershipSummary{SettlementCurrency: currency, CurrentAmount: "0.00000000", CurrentDiscount: "0", ProgressPercent: "0"}
	result.Rules = MembershipRules{WindowHours: 720, Priority: []string{"first_recharge", "campaign", "membership"}, SettlementCurrency: currency, AffiliateCommissionRate: cfg.AffiliateCommissionRate, Tiers: []map[string]string{}}
	if s == nil || s.sqlDB == nil {
		return result, nil
	}
	if err := s.sqlDB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND role = 'user' AND status = 'active' AND deleted_at IS NULL)`, userID).Scan(&result.Eligible); err != nil {
		return MembershipSummary{}, err
	}
	if !result.Eligible {
		return result, nil
	}
	now := time.Now()
	cutoff := now.Add(-30 * 24 * time.Hour)
	var amount string
	// refund_amount is wallet credit, not gateway money. Match the existing
	// proportional gateway refund calculation and only deduct settled refunds.
	err = s.sqlDB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE
		  WHEN status = 'REFUNDED' THEN 0
		  WHEN status = 'PARTIALLY_REFUNDED' AND amount > 0 THEN
		    GREATEST(0, pay_amount - ROUND(pay_amount * LEAST(GREATEST(refund_amount, 0), amount) / amount, $5))
		  ELSE pay_amount END), 0)
		FROM payment_orders
		WHERE user_id = $1 AND order_type = 'balance' AND settlement_currency = $2
		  AND paid_at >= $3 AND paid_at <= $4`, userID, currency, cutoff, now, payment.CurrencyMaxFractionDigits(currency)).Scan(&amount)
	if err != nil {
		return MembershipSummary{}, err
	}
	paid, err := decimal.NewFromString(amount)
	if err != nil {
		return MembershipSummary{}, err
	}
	current := decimal.Max(paid, decimal.Zero).Round(8)
	var firstUnavailable bool
	if err := s.sqlDB.QueryRowContext(ctx, `SELECT
		EXISTS(SELECT 1 FROM payment_orders WHERE user_id = $1 AND order_type = 'balance'
		  AND (paid_at IS NOT NULL OR completed_at IS NOT NULL OR status IN ('COMPLETED', 'REFUNDED', 'PARTIALLY_REFUNDED')))
		OR EXISTS(SELECT 1 FROM recharge_promotion_claims WHERE user_id = $1 AND source = 'first_recharge' AND status IN ('reserved', 'redeemed'))`, userID).Scan(&firstUnavailable); err != nil {
		return MembershipSummary{}, err
	}
	result.CurrentAmount = current.StringFixed(8)
	result.FirstRechargeEligible = !firstUnavailable
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT name, threshold_amount, discount_percent FROM balance_membership_tiers WHERE enabled = TRUE AND settlement_currency = $1 ORDER BY threshold_amount ASC, sort_order ASC, id ASC`, currency)
	if err != nil {
		return MembershipSummary{}, err
	}
	defer func() { _ = rows.Close() }()
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
		result.Rules.Tiers = append(result.Rules.Tiers, map[string]string{"name": name, "threshold_amount": thresholdDecimal.String(), "discount_percent": discountDecimal.String()})
	}
	if err := rows.Err(); err != nil {
		return MembershipSummary{}, err
	}
	result.Enabled = len(tiers) > 0
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

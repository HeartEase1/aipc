package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

var errRechargeQuoteChanged = errors.New("recharge quote changed")

// Balance payment confirmations and order creation share the same user lock.
// This also serializes a first-recharge reservation against ordinary payments.
func lockRechargeUser(ctx context.Context, client *dbent.Client, userID int64) error {
	rows, err := client.QueryContext(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return fmt.Errorf("recharge user not found")
	}
	return rows.Err()
}

func reserveRechargePromotionInTx(ctx context.Context, client *dbent.Client, userID, orderID int64, quote *RechargeQuote) error {
	if quote.PromotionID <= 0 {
		return nil
	}
	rows, err := client.QueryContext(ctx, `SELECT kind, settlement_currency, enabled, starts_at, ends_at, min_amount, max_amount, discount_percent, max_discount_amount,
	 CASE WHEN budget_amount IS NULL THEN NULL ELSE budget_amount-reserved_amount-redeemed_amount END
	 FROM recharge_promotions WHERE id=$1 FOR UPDATE`, quote.PromotionID)
	if err != nil {
		return err
	}
	var candidate RechargePromotionCandidate
	candidate.ID = quote.PromotionID
	if !rows.Next() {
		rows.Close()
		return errRechargeQuoteChanged
	}
	var min, max, cap, budget *string
	err = rows.Scan(&candidate.Kind, &candidate.Currency, &candidate.Enabled, &candidate.StartsAt, &candidate.EndsAt,
		&min, &max, &candidate.DiscountPercent, &cap, &budget)
	rows.Close()
	if err != nil {
		return err
	}
	for _, field := range []struct {
		raw    *string
		target **decimal.Decimal
	}{
		{min, &candidate.MinAmount}, {max, &candidate.MaxAmount}, {cap, &candidate.MaxDiscount}, {budget, &candidate.BudgetRemaining},
	} {
		if field.raw != nil {
			v, err := decimal.NewFromString(*field.raw)
			if err != nil {
				return err
			}
			*field.target = &v
		}
	}
	first := false
	if candidate.Kind == "first_recharge" {
		rows, err := client.QueryContext(ctx, `SELECT
		 EXISTS(SELECT 1 FROM payment_orders WHERE user_id=$1 AND order_type='balance' AND (paid_at IS NOT NULL OR completed_at IS NOT NULL OR status IN ('COMPLETED','REFUNDED','PARTIALLY_REFUNDED')))
		 OR EXISTS(SELECT 1 FROM recharge_promotion_claims WHERE user_id=$1 AND source='first_recharge' AND status IN ('reserved','redeemed'))`, userID)
		if err != nil {
			return err
		}
		var unavailable bool
		if !rows.Next() {
			rows.Close()
			return errRechargeQuoteChanged
		}
		err = rows.Scan(&unavailable)
		rows.Close()
		if err != nil {
			return err
		}
		first = !unavailable
	}
	amount, err := decimal.NewFromString(quote.OriginalAmount)
	if err != nil {
		return err
	}
	result := ResolveRechargePromotion(RechargePromotionPricingInput{Amount: amount, Currency: quote.Currency, IsFirstRecharge: first, Now: time.Now()}, []RechargePromotionCandidate{candidate})
	discount, err := decimal.NewFromString(quote.DiscountAmount)
	if err != nil {
		return err
	}
	if result.PromotionID != quote.PromotionID || !result.DiscountAmount.Equal(discount) || result.Source != quote.DiscountSource {
		return errRechargeQuoteChanged
	}
	if _, err = client.ExecContext(ctx, `UPDATE recharge_promotions SET reserved_amount=reserved_amount+$1, updated_at=NOW() WHERE id=$2`, discount.String(), quote.PromotionID); err != nil {
		return err
	}
	_, err = client.ExecContext(ctx, `INSERT INTO recharge_promotion_claims (promotion_id,user_id,order_id,source,discount_amount) VALUES ($1,$2,$3,$4,$5)`, quote.PromotionID, userID, orderID, quote.DiscountSource, discount.String())
	if err != nil && dbent.IsConstraintError(err) {
		// Another balance order won the first-recharge reservation while this
		// request was being priced. Roll back the order and obtain a fresh quote.
		return errRechargeQuoteChanged
	}
	return err
}

func redeemRechargePromotionInTx(ctx context.Context, client *dbent.Client, orderID int64) error {
	rows, err := client.QueryContext(ctx, `SELECT promotion_id, discount_amount, status FROM recharge_promotion_claims WHERE order_id=$1 FOR UPDATE`, orderID)
	if err != nil {
		return err
	}
	if !rows.Next() {
		err = rows.Err()
		rows.Close()
		return err
	}
	var promotionID int64
	var discount, status string
	err = rows.Scan(&promotionID, &discount, &status)
	rows.Close()
	if err != nil {
		return err
	}
	if status == "released" {
		return fmt.Errorf("payment arrived after promotion release; manual reconciliation required")
	}
	if status == "redeemed" {
		return nil
	}
	r, err := client.ExecContext(ctx, `UPDATE recharge_promotions SET reserved_amount=reserved_amount-$1, redeemed_amount=redeemed_amount+$1, updated_at=NOW() WHERE id=$2 AND reserved_amount >= $1`, discount, promotionID)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("promotion budget invariant violated")
	}
	_, err = client.ExecContext(ctx, `UPDATE recharge_promotion_claims SET status='redeemed', redeemed_at=NOW() WHERE order_id=$1`, orderID)
	return err
}

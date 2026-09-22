package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

// This snapshot is frozen at checkout. Confirmation only changes the decision/result fields.
type RechargeBonusSnapshot struct {
	CampaignID        int64                `json:"campaign_id"`
	Title             string               `json:"title"`
	Frequency         string               `json:"frequency"`
	Timezone          string               `json:"timezone"`
	Percent           string               `json:"percent"`
	Multiplier        string               `json:"multiplier"`
	Preview           string               `json:"preview"`
	Expected          string               `json:"expected"`
	Confirmed         bool                 `json:"confirmed"`
	Awarded           bool                 `json:"awarded"`
	Credited          bool                 `json:"credited"`
	Refunded          string               `json:"refunded"`
	Reason            string               `json:"reason,omitempty"`
	RefundedPrincipal string               `json:"refunded_principal,omitempty"`
	Refund            *RechargeBonusRefund `json:"refund,omitempty"`
}

func orderBonus(o *dbent.PaymentOrder) *RechargeBonusSnapshot {
	if o == nil || o.PricingSnapshot == nil || o.PricingSnapshot["bonus"] == nil {
		return nil
	}
	raw, err := json.Marshal(o.PricingSnapshot["bonus"])
	if err != nil {
		return nil
	}
	var b RechargeBonusSnapshot
	if json.Unmarshal(raw, &b) != nil || b.CampaignID <= 0 {
		return nil
	}
	return &b
}
func bonusPeriod(frequency, timezone string, now time.Time, orderID int64) (string, error) {
	switch frequency {
	case "every_payment":
		return "order:" + strconv.FormatInt(orderID, 10), nil
	case "campaign_first":
		return "campaign", nil
	case "daily_first":
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return "", err
		}
		return "day:" + now.In(loc).Format("2006-01-02"), nil
	default:
		return "", errors.New("invalid bonus frequency")
	}
}
func bonusBalance(amount, multiplier, percent decimal.Decimal) decimal.Decimal {
	return amount.Mul(multiplier).Mul(percent).Div(decimal.NewFromInt(100)).Round(2)
}
func (s *PaymentService) attachRechargeBonus(ctx context.Context, userID int64, amount decimal.Decimal, currency string, cfg *PaymentConfig, q *RechargeQuote) error {
	c, err := s.ActiveRechargeBonus(ctx, userID, currency)
	if err != nil {
		return err
	}
	if c == nil {
		return nil
	}
	for _, t := range c.Tiers {
		min, _ := decimal.NewFromString(t.MinAmount)
		if !amount.GreaterThan(min) {
			continue
		}
		if t.MaxAmount != nil {
			max, _ := decimal.NewFromString(*t.MaxAmount)
			if amount.GreaterThan(max) {
				continue
			}
		}
		percent, _ := decimal.NewFromString(t.BonusPercent)
		m := decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(cfg.BalanceRechargeMultiplier))
		credit := bonusBalance(amount, m, percent)
		if !credit.IsPositive() {
			return nil
		}
		q.Bonus = &RechargeBonusSnapshot{CampaignID: c.ID, Title: c.Title, Frequency: c.Frequency, Timezone: c.Timezone, Percent: percent.String(), Multiplier: m.String(), Expected: credit.StringFixed(2), Refunded: "0"}
		q.BonusCampaignID = c.ID
		q.BonusPercent = percent.String()
		q.BonusTitle = c.Title
		q.BonusAmount = amount.Mul(percent).Div(decimal.NewFromInt(100)).String()
		q.BonusCreditedAmount = credit.StringFixed(2)
		if !c.Eligible {
			q.BonusCreditedAmount = "0"
		}
		q.Bonus.Preview = q.BonusCreditedAmount
		principal, _ := decimal.NewFromString(q.CreditedAmount)
		preview, _ := decimal.NewFromString(q.BonusCreditedAmount)
		q.TotalCreditedAmount = principal.Add(preview).StringFixed(2)
		return nil
	}
	return nil
}

// Caller holds the user lock and has successfully changed PENDING to PAID in this transaction.
func claimRechargeBonus(ctx context.Context, client *dbent.Client, o *dbent.PaymentOrder, now time.Time) error {
	b := orderBonus(o)
	if b == nil {
		return nil
	}
	b.Confirmed = true
	b.Awarded = false
	b.Reason = "资格已使用"
	if !now.Before(o.ExpiresAt) {
		b.Reason = "订单有效期已结束"
	} else {
		key, err := bonusPeriod(b.Frequency, b.Timezone, now, o.ID)
		if err != nil {
			return err
		}
		r, err := client.ExecContext(ctx, `INSERT INTO recharge_bonus_claims(order_id,campaign_id,user_id,period_key,bonus_amount) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, o.ID, b.CampaignID, o.UserID, key, b.Expected)
		if err != nil {
			return err
		}
		n, err := r.RowsAffected()
		if err != nil {
			return err
		}
		b.Awarded = n == 1
		if b.Awarded {
			b.Reason = "充值赠送订单整笔不参与邀请返利"
		}
	}
	return saveOrderBonus(ctx, client, o, b)
}
func saveOrderBonus(ctx context.Context, client *dbent.Client, o *dbent.PaymentOrder, b *RechargeBonusSnapshot) error {
	snapshot := make(map[string]any, len(o.PricingSnapshot)+1)
	for k, v := range o.PricingSnapshot {
		snapshot[k] = v
	}
	snapshot["bonus"] = b
	credited := decimal.Zero
	if b.Credited {
		credited, _ = decimal.NewFromString(b.Expected)
	}
	snapshot["bonus_credited_amount"] = credited.StringFixed(2)
	snapshot["total_credited_amount"] = decimal.NewFromFloat(o.Amount).Add(credited).StringFixed(2)
	// Preserve the fulfillment lease timestamp: setting pricing must not steal the lease.
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if _, err = client.ExecContext(ctx, `UPDATE payment_orders SET pricing_snapshot=$1 WHERE id=$2`, string(raw), o.ID); err != nil {
		return err
	}
	o.PricingSnapshot = snapshot
	return nil
}
func (s *PaymentService) applyRechargeBonus(ctx context.Context, o *dbent.PaymentOrder, lease *paymentFulfillmentLease) error {
	b := orderBonus(o)
	if b == nil {
		return nil
	}
	if !b.Confirmed {
		return errors.New("bonus confirmation missing")
	}
	if !b.Awarded {
		return nil
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = lockRechargeUser(ctx, tx.Client(), o.UserID); err != nil {
		return err
	}
	if lease == nil {
		return errors.New("missing bonus fulfillment lease")
	}
	guard, err := tx.Client().QueryContext(ctx, `SELECT id FROM payment_orders WHERE id=$1 AND status='RECHARGING' AND updated_at=$2 FOR UPDATE`, o.ID, lease.version)
	if err != nil {
		return err
	}
	owned := guard.Next()
	_ = guard.Close()
	if !owned {
		return errors.New("bonus fulfillment lease lost")
	}
	rows, err := tx.Client().QueryContext(ctx, `SELECT credited_at FROM recharge_bonus_claims WHERE order_id=$1 FOR UPDATE`, o.ID)
	if err != nil {
		return err
	}
	if !rows.Next() {
		_ = rows.Close()
		return errors.New("bonus claim missing")
	}
	var creditedAt *time.Time
	err = rows.Scan(&creditedAt)
	_ = rows.Close()
	if err != nil {
		return err
	}
	if creditedAt == nil {
		amount, err := decimal.NewFromString(b.Expected)
		if err != nil || !amount.IsPositive() {
			return errors.New("invalid bonus amount")
		}
		if _, err = s.userRepo.AdjustBalance(dbent.NewTxContext(ctx, tx), o.UserID, amount.InexactFloat64()); err != nil {
			return err
		}
		if _, err = tx.Client().ExecContext(ctx, `UPDATE recharge_bonus_claims SET credited_at=NOW() WHERE order_id=$1`, o.ID); err != nil {
			return err
		}
		detail, _ := json.Marshal(map[string]any{"bonus_amount": b.Expected, "campaign_id": b.CampaignID, "affiliate_excluded": true})
		if err = tx.PaymentAuditLog.Create().SetOrderID(strconv.FormatInt(o.ID, 10)).SetAction("RECHARGE_BONUS_CREDITED").SetDetail(string(detail)).SetOperator("system").Exec(ctx); err != nil {
			return err
		}
	}
	b.Credited = true
	if err = saveOrderBonus(ctx, tx.Client(), o, b); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if s.redeemService != nil {
		s.redeemService.invalidateRedeemCaches(ctx, o.UserID, &RedeemCode{Type: RedeemTypeBalance})
	}
	return nil
}
func bonusRefundDue(o *dbent.PaymentOrder, principal float64) float64 {
	b := orderBonus(o)
	if b == nil || !b.Credited || o.Amount <= 0 {
		return 0
	}
	gift, err := decimal.NewFromString(b.Expected)
	if err != nil {
		return 0
	}
	recovered, _ := decimal.NewFromString(b.Refunded)
	target := gift.Mul(decimal.NewFromFloat(principal)).Div(decimal.NewFromFloat(o.Amount)).Round(2)
	return decimal.Max(decimal.Zero, decimal.Min(gift, target).Sub(recovered)).InexactFloat64()
}
func bonusDescription(o *dbent.PaymentOrder) string {
	b := orderBonus(o)
	if b == nil || !b.Credited {
		return ""
	}
	return fmt.Sprintf(" + 活动赠送 %s（%s%%），合计 %s", b.Expected, b.Percent, decimal.NewFromFloat(o.Amount).Add(decimalValue(b.Expected)).StringFixed(2))
}

func revalidateBonusOrder(ctx context.Context, client *dbent.Client, userID int64, q *RechargeQuote) error {
	if q.Bonus == nil {
		return nil
	}
	if _, err := client.ExecContext(ctx, `SELECT pg_advisory_xact_lock(71412541)`); err != nil {
		return err
	}
	b := q.Bonus
	rows, err := client.QueryContext(ctx, `SELECT EXISTS(SELECT 1 FROM recharge_bonus_campaigns c JOIN recharge_bonus_tiers t ON t.campaign_id=c.id WHERE c.id=$1 AND c.deleted_at IS NULL AND c.enabled AND c.starts_at<=NOW() AND c.ends_at>NOW() AND c.settlement_currency=$2 AND c.frequency=$3 AND c.timezone=$4 AND t.bonus_percent=$5 AND $6::numeric>t.min_amount AND (t.max_amount IS NULL OR $6::numeric<=t.max_amount) AND NOT EXISTS(SELECT 1 FROM marketing_user_exclusions WHERE user_id=$7 AND enabled AND scope IN ('all','recharge')))`, b.CampaignID, q.Currency, b.Frequency, b.Timezone, b.Percent, q.OriginalAmount, userID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return errors.New("bonus validation missing")
	}
	var valid bool
	if err = rows.Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return errRechargeQuoteChanged
	}
	return nil
}

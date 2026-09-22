package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// The deduction and this record commit together. Restoring or finalizing uses the same record.
type RechargeBonusRefund struct {
	ID               string `json:"id"`
	Reason           string `json:"reason"`
	Principal        string `json:"principal"`
	Before           string `json:"before"`
	BonusDue         string `json:"bonus_due"`
	Deducted         string `json:"deducted"`
	Deduct           bool   `json:"deduct"`
	Force            bool   `json:"force"`
	Restored         bool   `json:"restored"`
	Finished         bool   `json:"finished"`
	ProviderRefundID string `json:"provider_refund_id,omitempty"`
}

func bonusPrincipalRecovered(b *RechargeBonusSnapshot) float64 {
	if b == nil {
		return 0
	}
	v, _ := decimal.NewFromString(b.RefundedPrincipal)
	return v.InexactFloat64()
}
func bonusRefundRequestID(o *dbent.PaymentOrder) string {
	b := orderBonus(o)
	if b == nil || b.Refund == nil {
		return ""
	}
	return b.Refund.ID
}
func decimalValue(s string) decimal.Decimal { v, _ := decimal.NewFromString(s); return v }
func bonusRefundID(o *dbent.PaymentOrder, amount float64) string {
	sum := sha256.Sum256([]byte(o.OutTradeNo + ":" + strconv.FormatFloat(bonusPrincipalRecovered(orderBonus(o)), 'f', 2, 64) + ":" + strconv.FormatFloat(amount, 'f', 2, 64)))
	return "br" + hex.EncodeToString(sum[:15])
}
func (s *PaymentService) lockBonusRefund(ctx context.Context, tx *dbent.Tx, p *RefundPlan) (*dbent.PaymentOrder, *RechargeBonusSnapshot, error) {
	if err := lockRechargeUser(ctx, tx.Client(), p.Order.UserID); err != nil {
		return nil, nil, err
	}
	rows, err := tx.Client().QueryContext(ctx, `SELECT id FROM payment_orders WHERE id=$1 FOR UPDATE`, p.OrderID)
	if err != nil {
		return nil, nil, err
	}
	ok := rows.Next()
	_ = rows.Close()
	if !ok {
		return nil, nil, errors.New("refund order missing")
	}
	o, err := tx.PaymentOrder.Get(ctx, p.OrderID)
	if err != nil {
		return nil, nil, err
	}
	b := orderBonus(o)
	if b == nil || !b.Credited {
		return nil, nil, errors.New("credited bonus missing")
	}
	return o, b, nil
}
func (s *PaymentService) bonusRefundDebit(ctx context.Context, tx *dbent.Tx, o *dbent.PaymentOrder, r *RechargeBonusRefund) error {
	if !r.Deduct {
		r.Deducted = "0"
		r.Restored = false
		return nil
	}
	rows, err := tx.Client().QueryContext(ctx, `SELECT balance-frozen_balance FROM users WHERE id=$1`, o.UserID)
	if err != nil {
		return err
	}
	if !rows.Next() {
		_ = rows.Close()
		return ErrUserNotFound
	}
	var available decimal.Decimal
	err = rows.Scan(&available)
	_ = rows.Close()
	if err != nil {
		return err
	}
	need := decimalValue(r.Principal).Add(decimalValue(r.BonusDue))
	if available.LessThan(need) && !r.Force {
		return ErrBalanceNegative
	}
	actual := decimal.Max(decimal.Zero, decimal.Min(need, available))
	if actual.IsPositive() {
		if _, err = s.userRepo.AdjustBalance(dbent.NewTxContext(ctx, tx), o.UserID, -actual.InexactFloat64()); err != nil {
			return err
		}
	}
	r.Deducted = actual.StringFixed(2)
	r.Restored = false
	return nil
}
func (s *PaymentService) executeBonusRefund(ctx context.Context, p *RefundPlan) (*RefundResult, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	o, b, err := s.lockBonusRefund(ctx, tx, p)
	if err != nil {
		return nil, err
	}
	if !psSliceContains([]string{OrderStatusCompleted, OrderStatusPartiallyRefunded, OrderStatusRefundRequested, OrderStatusRefundFailed}, o.Status) {
		return nil, infraerrors.Conflict("REFUND_BUSY", "refund already processing; query its status")
	}
	if p.RefundAmount <= 0 || decimalValue(b.RefundedPrincipal).Add(decimal.NewFromFloat(p.RefundAmount)).GreaterThan(decimal.NewFromFloat(o.Amount)) {
		return nil, infraerrors.BadRequest("REFUND_AMOUNT_EXCEEDED", "refund exceeds remaining principal")
	}
	if bonusPrincipalRecovered(orderBonus(p.Order)) != bonusPrincipalRecovered(b) {
		return nil, infraerrors.Conflict("REFUND_CHANGED", "another refund completed; refresh the order")
	}
	if !decimal.NewFromFloat(p.RefundAmount).Equal(decimal.NewFromFloat(p.RefundAmount).Round(2)) {
		return nil, infraerrors.BadRequest("INVALID_REFUND_PRECISION", "refund principal supports two decimal places")
	}
	r := b.Refund
	if r == nil || r.Finished {
		r = &RechargeBonusRefund{ID: bonusRefundID(o, p.RefundAmount), Reason: p.Reason, Principal: decimal.NewFromFloat(p.RefundAmount).StringFixed(2), Before: decimalValue(b.RefundedPrincipal).String(), BonusDue: decimal.NewFromFloat(bonusRefundDue(o, p.RefundAmount+bonusPrincipalRecovered(b))).StringFixed(2), Deduct: p.DeductBalance, Force: p.Force, Restored: true}
		b.Refund = r
	} else if decimalValue(r.Principal).InexactFloat64() != p.RefundAmount || r.Deduct != p.DeductBalance {
		return nil, infraerrors.Conflict("REFUND_RETRY_MISMATCH", "retry the same refund amount and deduction option")
	}
	r.Force = p.Force
	// Gateway idempotency also requires the original request parameters on retries.
	p.Reason = r.Reason
	if r.Restored {
		if err = s.bonusRefundDebit(ctx, tx, o, r); err != nil {
			return nil, err
		}
	}
	if err = saveOrderBonus(ctx, tx.Client(), o, b); err != nil {
		return nil, err
	}
	if err = tx.PaymentOrder.UpdateOneID(o.ID).SetStatus(OrderStatusRefunding).SetRefundReason(p.Reason).SetForceRefund(p.Force).Exec(ctx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	p.Order = o
	p.GatewayAmount = bonusGatewayRefundAmount(o, r)
	p.BonusToDeduct = decimalValue(r.BonusDue).InexactFloat64()
	p.BalanceToDeduct = decimalValue(r.Deducted).InexactFloat64()
	s.invalidateBonusBalance(ctx, o.UserID)
	resp, err := s.gwRefund(ctx, p)
	if err != nil {
		// An ambiguous network error must not start a different refund. Persist and expose it for reconciliation.
		resp = &payment.RefundResponse{Status: payment.ProviderStatusPending, RefundID: r.ProviderRefundID}
	}
	return s.finishBonusRefund(ctx, p, resp)
}
func (s *PaymentService) finishBonusRefund(ctx context.Context, p *RefundPlan, resp *payment.RefundResponse) (*RefundResult, error) {
	if resp == nil {
		return nil, errors.New("missing refund response")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	o, b, err := s.lockBonusRefund(ctx, tx, p)
	if err != nil {
		return nil, err
	}
	r := b.Refund
	if r == nil || r.ID != bonusRefundRequestID(p.Order) {
		return nil, infraerrors.Conflict("REFUND_CHANGED", "refund operation changed")
	}
	if r.Finished {
		return &RefundResult{Success: true, BalanceDeducted: decimalValue(r.Deducted).InexactFloat64()}, nil
	}
	if !psSliceContains([]string{OrderStatusRefunding, OrderStatusRefundPending, OrderStatusRefundFailed}, o.Status) {
		return nil, infraerrors.Conflict("REFUND_CHANGED", "refund status changed")
	}
	if resp.RefundID != "" {
		r.ProviderRefundID = resp.RefundID
	}
	success := resp.Status == payment.ProviderStatusSuccess || resp.Status == payment.ProviderStatusRefunded
	status := OrderStatusRefundPending
	action := "BONUS_REFUND_PENDING"
	if success {
		if r.Restored {
			if err = s.bonusRefundDebit(ctx, tx, o, r); err != nil {
				return nil, err
			}
		}
		recovered := decimal.Max(decimal.Zero, decimalValue(r.Deducted).Sub(decimalValue(r.Principal)))
		b.Refunded = decimalValue(b.Refunded).Add(recovered).StringFixed(2)
		b.RefundedPrincipal = decimalValue(r.Before).Add(decimalValue(r.Principal)).StringFixed(2)
		r.Finished = true
		status = OrderStatusPartiallyRefunded
		if decimalValue(b.RefundedPrincipal).GreaterThanOrEqual(decimal.NewFromFloat(o.Amount)) {
			status = OrderStatusRefunded
		}
		action = "REFUND_SUCCESS"
	} else {
		if resp.Status == payment.ProviderStatusFailed && !r.Restored {
			amount := decimalValue(r.Deducted)
			if amount.IsPositive() {
				if _, err = s.userRepo.AdjustBalance(dbent.NewTxContext(ctx, tx), o.UserID, amount.InexactFloat64()); err != nil {
					return nil, err
				}
			}
			r.Restored = true
		}
		if resp.Status == payment.ProviderStatusFailed {
			status = OrderStatusRefundFailed
			action = "BONUS_REFUND_FAILED"
		}
	}
	if err = saveOrderBonus(ctx, tx.Client(), o, b); err != nil {
		return nil, err
	}
	update := tx.PaymentOrder.UpdateOneID(o.ID).SetStatus(status).SetRefundAmount(decimalValue(b.RefundedPrincipal).InexactFloat64())
	if success {
		update.SetRefundAt(time.Now())
	}
	if err = update.Exec(ctx); err != nil {
		return nil, err
	}
	detail, _ := json.Marshal(map[string]any{"refund_id": r.ID, "principal": r.Principal, "bonus_due": r.BonusDue, "balance_deducted": r.Deducted, "restored": r.Restored, "deduct_balance": r.Deduct, "force": r.Force})
	if err = tx.PaymentAuditLog.Create().SetOrderID(strconv.FormatInt(o.ID, 10)).SetAction(action).SetDetail(string(detail)).SetOperator("admin").Exec(ctx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	s.invalidateBonusBalance(ctx, o.UserID)
	result := &RefundResult{Success: success}
	if success {
		result.BalanceDeducted = decimalValue(r.Deducted).InexactFloat64()
	} else {
		result.Warning = "refund pending reconciliation; principal and bonus deduction retained until confirmation"
		if status == OrderStatusRefundFailed {
			result.Warning = "refund failed; deduction restored"
		}
	}
	return result, nil
}
func (s *PaymentService) queryBonusRefund(ctx context.Context, o *dbent.PaymentOrder, provider payment.Provider) (*RefundResult, error) {
	b := orderBonus(o)
	r := b.Refund
	p := &RefundPlan{OrderID: o.ID, Order: o, RefundAmount: decimalValue(r.Principal).InexactFloat64(), Force: r.Force, DeductBalance: r.Deduct, Reason: r.Reason}
	p.GatewayAmount = bonusGatewayRefundAmount(o, r)
	if r.ProviderRefundID == "" && psSliceContains([]string{"stripe", "alipay", "wxpay", "airwallex"}, provider.ProviderKey()) {
		resp, err := s.gwRefund(ctx, p)
		if err != nil {
			return nil, err
		}
		return s.finishBonusRefund(ctx, p, resp)
	}
	query, ok := provider.(payment.RefundQueryProvider)
	if !ok {
		return nil, infraerrors.BadRequest("REFUND_QUERY_UNSUPPORTED", "provider requires manual refund verification")
	}
	resp, err := query.QueryRefund(ctx, payment.RefundQueryRequest{TradeNo: o.PaymentTradeNo, OrderID: o.OutTradeNo, RefundID: r.ProviderRefundID, Amount: formatGatewayRefundAmount(p.GatewayAmount, o)})
	if err != nil {
		return nil, err
	}
	if resp == nil || !psSliceContains([]string{payment.ProviderStatusSuccess, payment.ProviderStatusRefunded, payment.ProviderStatusFailed, payment.ProviderStatusPending}, strings.TrimSpace(resp.Status)) {
		return nil, errors.New("unknown refund state")
	}
	return s.finishBonusRefund(ctx, p, resp)
}
func (s *PaymentService) invalidateBonusBalance(ctx context.Context, userID int64) {
	if s.redeemService != nil {
		s.redeemService.invalidateRedeemCaches(ctx, userID, &RedeemCode{Type: RedeemTypeBalance})
	}
}

func bonusGatewayRefundAmount(o *dbent.PaymentOrder, r *RechargeBonusRefund) float64 {
	before := decimalValue(r.Before)
	after := before.Add(decimalValue(r.Principal))
	digits := int32(payment.CurrencyMaxFractionDigits(PaymentOrderCurrency(o)))
	total := decimal.NewFromFloat(o.Amount)
	paid := decimal.NewFromFloat(o.PayAmount)
	return paid.Mul(after).Div(total).Round(digits).Sub(paid.Mul(before).Div(total).Round(digits)).InexactFloat64()
}

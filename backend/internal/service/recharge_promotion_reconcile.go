package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// Only a successful upstream close followed by a second unpaid check after the
// callback grace period can release a reservation. Network errors never do so.
func (s *PaymentService) reconcileRechargeReservations(ctx context.Context) error {
	if s.sqlDB == nil {
		return nil
	}
	rows, err := s.sqlDB.QueryContext(ctx, `SELECT c.order_id, c.closed_confirmed_at FROM recharge_promotion_claims c JOIN payment_orders o ON o.id=c.order_id
	 WHERE c.status='reserved' AND o.status IN ('CANCELLED','EXPIRED','FAILED') AND o.paid_at IS NULL
	 ORDER BY c.checked_at NULLS FIRST, c.id LIMIT 20`)
	if err != nil {
		return err
	}
	type pending struct {
		id     int64
		closed *time.Time
	}
	var items []pending
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.id, &item.closed); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, err := s.sqlDB.ExecContext(ctx, `UPDATE recharge_promotion_claims SET checked_at=NOW() WHERE order_id=$1`, item.id); err != nil {
			return err
		}
		o, err := s.entClient.PaymentOrder.Get(ctx, item.id)
		if err != nil {
			continue
		}
		prov, err := s.getOrderProvider(ctx, o)
		if err != nil {
			continue
		}
		ref := paymentOrderQueryReference(o, prov)
		if ref == "" {
			continue
		}
		checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		resp, err := prov.QueryOrder(checkCtx, ref)
		cancel()
		if err != nil || resp == nil {
			continue
		}
		if resp.Status == payment.ProviderStatusPaid {
			s.reconcilePaid(ctx, o)
			continue
		}
		if resp.Status != payment.ProviderStatusPending && resp.Status != payment.ProviderStatusFailed {
			continue
		}
		if item.closed == nil {
			closer, ok := prov.(payment.CancelableProvider)
			if !ok {
				continue
			}
			closeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := closer.CancelPayment(closeCtx, ref)
			cancel()
			if err != nil {
				continue
			}
			if _, err := s.sqlDB.ExecContext(ctx, `UPDATE recharge_promotion_claims SET closed_confirmed_at=NOW() WHERE order_id=$1 AND status='reserved' AND closed_confirmed_at IS NULL`, item.id); err != nil {
				return err
			}
			continue
		}
		if time.Since(*item.closed) < paymentGraceMinutes*time.Minute {
			continue
		}
		if err := s.releaseClosedRechargeReservation(ctx, item.id, o.UserID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PaymentService) releaseClosedRechargeReservation(ctx context.Context, orderID, userID int64) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	c := tx.Client()
	if err := lockRechargeUser(ctx, c, userID); err != nil {
		return err
	}
	rows, err := c.QueryContext(ctx, `UPDATE recharge_promotion_claims c SET status='released', released_at=NOW() FROM payment_orders o
	 WHERE c.order_id=o.id AND o.id=$1 AND c.status='reserved' AND c.closed_confirmed_at <= $2
	 AND o.paid_at IS NULL AND o.status IN ('CANCELLED','EXPIRED','FAILED') RETURNING c.promotion_id,c.discount_amount`, orderID, time.Now().Add(-paymentGraceMinutes*time.Minute))
	if err != nil {
		return err
	}
	if !rows.Next() {
		err = rows.Err()
		rows.Close()
		return err
	}
	var promotionID int64
	var discount string
	err = rows.Scan(&promotionID, &discount)
	rows.Close()
	if err != nil {
		return err
	}
	r, err := c.ExecContext(ctx, `UPDATE recharge_promotions SET reserved_amount=reserved_amount-$1, updated_at=NOW() WHERE id=$2 AND reserved_amount >= $1`, discount, promotionID)
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
	return tx.Commit()
}

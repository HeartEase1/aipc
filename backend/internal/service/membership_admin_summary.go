package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/lib/pq"
)

type AdminMembershipSummary struct {
	CurrentTier        string `json:"current_tier"`
	CurrentAmount      string `json:"current_amount"`
	SettlementCurrency string `json:"settlement_currency"`
}

// Load only the displayed page, in one query, rather than invoking the full
// membership/first-recharge calculation separately for every user.
func (s *PaymentService) GetAdminMembershipSummaries(ctx context.Context, userIDs []int64) (map[int64]AdminMembershipSummary, error) {
	if len(userIDs) == 0 || len(userIDs) > 1000 {
		return nil, infraerrors.BadRequest("INVALID_USER_IDS", "provide 1 to 1000 user IDs")
	}
	for _, id := range userIDs {
		if id <= 0 {
			return nil, infraerrors.BadRequest("INVALID_USER_IDS", "user IDs must be positive")
		}
	}
	db, err := s.membershipDB()
	if err != nil {
		return nil, err
	}
	cfg, err := s.GetBalanceMarketingConfig(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	rows, err := db.QueryContext(ctx, `WITH selected AS (
	 SELECT id, role, status FROM users WHERE id=ANY($1) AND deleted_at IS NULL
	), paid AS (
	 SELECT o.user_id, GREATEST(0, SUM(CASE
	 WHEN o.status='REFUNDED' THEN 0
	 WHEN o.status='PARTIALLY_REFUNDED' AND o.amount > 0 THEN
	 GREATEST(0, o.pay_amount - ROUND(o.pay_amount * LEAST(GREATEST(o.refund_amount,0),o.amount) / o.amount,$5))
	 ELSE o.pay_amount END)) AS amount
	 FROM payment_orders o JOIN selected u ON u.id=o.user_id
	 WHERE o.order_type='balance' AND o.settlement_currency=$2 AND o.paid_at >= $3 AND o.paid_at <= $4
	 GROUP BY o.user_id
	)
	SELECT u.id, COALESCE(t.name,''), CASE WHEN u.role='user' AND u.status='active' THEN COALESCE(p.amount,0) ELSE 0 END::text
	FROM selected u LEFT JOIN paid p ON p.user_id=u.id
	LEFT JOIN LATERAL (
	 SELECT name FROM balance_membership_tiers
	 WHERE enabled=TRUE AND settlement_currency=$2 AND threshold_amount <= COALESCE(p.amount,0)
	 AND u.role='user' AND u.status='active'
	 ORDER BY threshold_amount DESC, sort_order DESC, id DESC LIMIT 1
	) t ON TRUE`, pq.Array(userIDs), cfg.SettlementCurrency, now.Add(-30*24*time.Hour), now, payment.CurrencyMaxFractionDigits(cfg.SettlementCurrency))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make(map[int64]AdminMembershipSummary, len(userIDs))
	for rows.Next() {
		var id int64
		item := AdminMembershipSummary{SettlementCurrency: cfg.SettlementCurrency}
		if err := rows.Scan(&id, &item.CurrentTier, &item.CurrentAmount); err != nil {
			return nil, err
		}
		result[id] = item
	}
	return result, rows.Err()
}

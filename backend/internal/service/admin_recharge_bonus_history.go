package service

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// Read the committed claim ledger so past credits are visible without creating
// redeem codes or counting promotional balances as paid recharges.
func (s *adminServiceImpl) listRechargeBonusBalanceHistory(ctx context.Context, userID int64, params pagination.PaginationParams) ([]RedeemCode, int64, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return nil, 0, nil
	}
	rows, err := s.entClient.QueryContext(ctx, `
SELECT c.order_id, c.bonus_amount::double precision, c.credited_at,
       COALESCE(o.pricing_snapshot->'bonus'->>'title', ''),
       COALESCE(o.out_trade_no, '')
FROM recharge_bonus_claims c
LEFT JOIN payment_orders o ON o.id = c.order_id AND o.user_id = c.user_id
WHERE c.user_id = $1 AND c.credited_at IS NOT NULL
ORDER BY c.credited_at DESC, c.order_id DESC
OFFSET $2 LIMIT $3`, userID, params.Offset(), params.Limit())
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	codes := make([]RedeemCode, 0, params.Limit())
	for rows.Next() {
		var orderID int64
		var amount float64
		var creditedAt time.Time
		var title, orderNo string
		if err := rows.Scan(&orderID, &amount, &creditedAt, &title, &orderNo); err != nil {
			return nil, 0, err
		}
		if orderNo == "" {
			orderNo = strconv.FormatInt(orderID, 10)
		}
		codes = append(codes, RedeemCode{
			ID: -orderID, Code: orderNo, Type: RedeemTypeRechargeBonus,
			Value: amount, Status: StatusUsed, UsedBy: &userID,
			UsedAt: &creditedAt, CreatedAt: creditedAt, Notes: title,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	countRows, err := s.entClient.QueryContext(ctx, `
SELECT COUNT(*) FROM recharge_bonus_claims
WHERE user_id = $1 AND credited_at IS NOT NULL`, userID)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = countRows.Close() }()
	var total int64
	if countRows.Next() {
		if err := countRows.Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	return codes, total, countRows.Err()
}

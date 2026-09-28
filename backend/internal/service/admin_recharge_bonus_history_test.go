package service

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type bonusHistoryRedeemRepo struct {
	RedeemCodeRepository
	codes []RedeemCode
}

func (r *bonusHistoryRedeemRepo) ListByUserPaginated(_ context.Context, _ int64, _ pagination.PaginationParams, _ string) ([]RedeemCode, *pagination.PaginationResult, error) {
	return r.codes, &pagination.PaginationResult{Total: int64(len(r.codes))}, nil
}

func (r *bonusHistoryRedeemRepo) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	return 1000, nil
}

const bonusHistoryQuery = `SELECT c.order_id, c.bonus_amount::double precision, c.credited_at, COALESCE(o.pricing_snapshot->'bonus'->>'title', ''), COALESCE(o.out_trade_no, '') FROM recharge_bonus_claims c LEFT JOIN payment_orders o ON o.id = c.order_id AND o.user_id = c.user_id WHERE c.user_id = $1 AND c.credited_at IS NOT NULL ORDER BY c.credited_at DESC, c.order_id DESC OFFSET $2 LIMIT $3`
const bonusHistoryCountQuery = `SELECT COUNT(*) FROM recharge_bonus_claims WHERE user_id = $1 AND credited_at IS NOT NULL`

func bonusHistoryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"order_id", "bonus_amount", "credited_at", "title", "out_trade_no"})
}

func TestRechargeBonusHistoryUsesActualCreditAndExcludesBonusFromRechargeTotal(t *testing.T) {
	client, mock := bonusMock(t)
	now := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(bonusHistoryQuery)).WithArgs(int64(7), 15, 15).
		WillReturnRows(bonusHistoryRows().AddRow(8, 100, now, "Original campaign title", "ORDER-888"))
	mock.ExpectQuery(regexp.QuoteMeta(bonusHistoryCountQuery)).WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(16))
	svc := &adminServiceImpl{entClient: client, redeemCodeRepo: &bonusHistoryRedeemRepo{}}
	items, total, paid, err := svc.GetUserBalanceHistory(context.Background(), 7, 2, 15, RedeemTypeRechargeBonus)
	require.NoError(t, err)
	require.EqualValues(t, 16, total)
	require.Equal(t, 1000.0, paid)
	require.Len(t, items, 1)
	require.Equal(t, RedeemTypeRechargeBonus, items[0].Type)
	require.Equal(t, 100.0, items[0].Value)
	require.Equal(t, "ORDER-888", items[0].Code)
	require.Equal(t, "Original campaign title", items[0].Notes)
	require.Equal(t, now, *items[0].UsedAt)
	require.EqualValues(t, 7, *items[0].UsedBy)
}

func TestRechargeBonusHistoryCombinedPagination(t *testing.T) {
	for _, page := range []int{1, 2} {
		t.Run(strconv.Itoa(page), func(t *testing.T) {
			client, mock := bonusMock(t)
			now := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
			mock.ExpectQuery(`SELECT id, amount::double precision, created_at FROM user_affiliate_ledger`).
				WithArgs(int64(7), 0, 1000).WillReturnRows(sqlmock.NewRows([]string{"id", "amount", "created_at"}).AddRow(8, 5, now.Add(-time.Hour)))
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM user_affiliate_ledger`)).WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			mock.ExpectQuery(regexp.QuoteMeta(bonusHistoryQuery)).WithArgs(int64(7), 0, 1000).
				WillReturnRows(bonusHistoryRows().AddRow(8, 100, now, "Gift", "ORDER-888"))
			mock.ExpectQuery(regexp.QuoteMeta(bonusHistoryCountQuery)).WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			svc := &adminServiceImpl{entClient: client, redeemCodeRepo: &bonusHistoryRedeemRepo{codes: []RedeemCode{
				{ID: 1, Type: RedeemTypeBalance, Value: 1000, CreatedAt: now.Add(-2 * time.Hour)},
				{ID: 2, Type: RedeemTypeConcurrency, Value: 2, CreatedAt: now.Add(-3 * time.Hour)},
			}}}
			items, total, paid, err := svc.GetUserBalanceHistory(context.Background(), 7, page, 2, "")
			require.NoError(t, err)
			require.EqualValues(t, 4, total)
			require.Equal(t, 1000.0, paid)
			require.Len(t, items, 2)
			if page == 1 {
				require.Equal(t, RedeemTypeRechargeBonus, items[0].Type)
				require.Equal(t, RedeemTypeAffiliateBalance, items[1].Type)
			} else {
				require.Equal(t, RedeemTypeBalance, items[0].Type)
				require.Equal(t, RedeemTypeConcurrency, items[1].Type)
			}
		})
	}
}

func TestRechargeBonusHistoryEmptyPagePreservesTotal(t *testing.T) {
	client, mock := bonusMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(bonusHistoryQuery)).WithArgs(int64(7), 20, 20).WillReturnRows(bonusHistoryRows())
	mock.ExpectQuery(regexp.QuoteMeta(bonusHistoryCountQuery)).WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	svc := &adminServiceImpl{entClient: client}
	items, total, err := svc.listRechargeBonusBalanceHistory(context.Background(), 7, pagination.PaginationParams{Page: 2, PageSize: 20})
	require.NoError(t, err)
	require.Empty(t, items)
	require.EqualValues(t, 1, total)
}

func TestRechargeBonusHistoryQueryFailureIsReported(t *testing.T) {
	client, mock := bonusMock(t)
	failure := errors.New("history unavailable")
	mock.ExpectQuery(regexp.QuoteMeta(bonusHistoryQuery)).WillReturnError(failure)
	svc := &adminServiceImpl{entClient: client}
	_, _, _, err := svc.GetUserBalanceHistory(context.Background(), 7, 1, 15, RedeemTypeRechargeBonus)
	require.ErrorIs(t, err, failure)
}

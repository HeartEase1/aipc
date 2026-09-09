package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestAdminMembershipSummariesBatchAndValidation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	s := &PaymentService{sqlDB: db}
	for _, ids := range [][]int64{nil, {0}, {-1}, make([]int64, 1001)} {
		_, err := s.GetAdminMembershipSummaries(context.Background(), ids)
		require.Error(t, err)
	}
	mock.ExpectQuery("WITH selected AS").WithArgs(sqlmock.AnyArg(), "CNY", sqlmock.AnyArg(), sqlmock.AnyArg(), 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tier", "amount"}).AddRow(7, "VIP", "500").AddRow(8, "", "99"))
	result, err := s.GetAdminMembershipSummaries(context.Background(), []int64{7, 8})
	require.NoError(t, err)
	require.Equal(t, "VIP", result[7].CurrentTier)
	require.Equal(t, "500", result[7].CurrentAmount)
	require.Equal(t, "CNY", result[7].SettlementCurrency)
	require.Empty(t, result[8].CurrentTier)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFirstRechargeStatusSeparatesReservationFromSuccessfulPayment(t *testing.T) {
	for _, tc := range []struct {
		name           string
		used, reserved bool
		status         string
		eligible       bool
	}{
		{"unpaid_or_cancelled_order_reserved", false, true, "reserved", false},
		{"cancelled_order_reservation_released", false, false, "eligible", true},
		{"successful_or_refunded_payment", true, false, "used", false},
		{"successful_payment_with_stale_reservation", true, true, "used", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			mock.ExpectQuery("SELECT EXISTS.*FROM users").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			mock.ExpectQuery("SELECT COALESCE\\(SUM\\(CASE").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow("0"))
			mock.ExpectQuery("SELECT.*EXISTS.*FROM payment_orders.*status = 'redeemed'.*status = 'reserved'").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"used", "reserved"}).AddRow(tc.used, tc.reserved))
			mock.ExpectQuery("SELECT name, threshold_amount, discount_percent FROM balance_membership_tiers").WillReturnRows(sqlmock.NewRows([]string{"name", "threshold", "discount"}))
			result, err := (&PaymentService{sqlDB: db}).GetMembershipSummary(context.Background(), 7)
			require.NoError(t, err)
			require.Equal(t, tc.status, result.FirstRechargeStatus)
			require.Equal(t, tc.eligible, result.FirstRechargeEligible)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

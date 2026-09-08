package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestMembershipSummaryExcludesIneligibleUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectQuery("SELECT EXISTS.*FROM users").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	s := &PaymentService{sqlDB: db}
	result, err := s.GetMembershipSummary(context.Background(), 7)
	require.NoError(t, err)
	require.False(t, result.Eligible)
	require.False(t, result.Enabled)
	require.False(t, result.FirstRechargeEligible)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMembershipSummaryEmptyTiersRemainDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectQuery("SELECT EXISTS.*FROM users").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(CASE").WithArgs(int64(7), "CNY", sqlmock.AnyArg(), sqlmock.AnyArg(), 2).WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow("75.00000000"))
	mock.ExpectQuery("SELECT.*EXISTS.*FROM payment_orders").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"unavailable"}).AddRow(true))
	mock.ExpectQuery("SELECT name, threshold_amount, discount_percent FROM balance_membership_tiers").WithArgs("CNY").WillReturnRows(sqlmock.NewRows([]string{"name", "threshold", "discount"}))
	result, err := (&PaymentService{sqlDB: db}).GetMembershipSummary(context.Background(), 7)
	require.NoError(t, err)
	require.False(t, result.Enabled)
	require.True(t, result.Eligible)
	require.Equal(t, "75.00000000", result.CurrentAmount)
	require.False(t, result.FirstRechargeEligible)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMembershipSummaryChoosesHighestReachedTier(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectQuery("SELECT EXISTS.*FROM users").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(CASE").WithArgs(int64(7), "CNY", sqlmock.AnyArg(), sqlmock.AnyArg(), 2).WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow("300"))
	mock.ExpectQuery("SELECT.*EXISTS.*FROM payment_orders").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"unavailable"}).AddRow(true))
	mock.ExpectQuery("SELECT name, threshold_amount, discount_percent FROM balance_membership_tiers").WithArgs("CNY").WillReturnRows(sqlmock.NewRows([]string{"name", "threshold", "discount"}).AddRow("VIP", "100", "1").AddRow("VIP1", "300", "3").AddRow("SVIP", "1000", "10"))
	result, err := (&PaymentService{sqlDB: db}).GetMembershipSummary(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, result.Enabled)
	require.Equal(t, "VIP1", result.CurrentTier)
	require.Equal(t, "3.0000", result.CurrentDiscount)
	require.Equal(t, "SVIP", result.NextTier)
	require.Equal(t, "700.00000000", result.AmountToNext)
	require.Equal(t, "30.00", result.ProgressPercent)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMembershipTierValidation(t *testing.T) {
	base := MembershipTierAdminInput{Name: " VIP ", SettlementCurrency: "cny", ThresholdAmount: "100.000", DiscountPercent: "3.00"}
	valid, err := validateMembershipTierInput(base)
	require.NoError(t, err)
	require.Equal(t, "VIP", valid.Name)
	require.Equal(t, "CNY", valid.SettlementCurrency)
	require.Equal(t, "100", valid.ThresholdAmount)
	for _, tc := range []struct {
		name   string
		mutate func(*MembershipTierAdminInput)
	}{
		{"empty name", func(in *MembershipTierAdminInput) { in.Name = "  " }},
		{"invalid currency", func(in *MembershipTierAdminInput) { in.SettlementCurrency = "CN" }},
		{"negative threshold", func(in *MembershipTierAdminInput) { in.ThresholdAmount = "-1" }},
		{"threshold scale", func(in *MembershipTierAdminInput) { in.ThresholdAmount = "0.000000001" }},
		{"threshold overflow", func(in *MembershipTierAdminInput) { in.ThresholdAmount = "1000000000000" }},
		{"unbounded exponent", func(in *MembershipTierAdminInput) { in.ThresholdAmount = "1e999999999" }},
		{"negative discount", func(in *MembershipTierAdminInput) { in.DiscountPercent = "-1" }},
		{"free order", func(in *MembershipTierAdminInput) { in.DiscountPercent = "100" }},
		{"discount scale", func(in *MembershipTierAdminInput) { in.DiscountPercent = "1.00001" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			_, err := validateMembershipTierInput(in)
			require.Error(t, err)
		})
	}
}

func TestMembershipTierCreateRollsBackInconsistentOrdering(t *testing.T) {
	for _, tc := range []struct{ name, threshold, discount string }{
		{"duplicate threshold", "100", "3"},
		{"worse discount", "300", "0.5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectExec("LOCK TABLE balance_membership_tiers IN SHARE ROW EXCLUSIVE MODE").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("INSERT INTO balance_membership_tiers").WithArgs("VIP2", "CNY", tc.threshold, tc.discount, 0, true).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
			mock.ExpectQuery("SELECT threshold_amount, discount_percent FROM balance_membership_tiers").WithArgs("CNY").WillReturnRows(sqlmock.NewRows([]string{"threshold", "discount"}).AddRow("100", "1").AddRow(tc.threshold, tc.discount))
			mock.ExpectRollback()
			_, err = (&PaymentService{sqlDB: db}).CreateMembershipTier(context.Background(), MembershipTierAdminInput{Name: "VIP2", SettlementCurrency: "CNY", ThresholdAmount: tc.threshold, DiscountPercent: tc.discount, Enabled: true})
			require.ErrorContains(t, err, "INVALID_MEMBERSHIP_TIER_ORDER")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

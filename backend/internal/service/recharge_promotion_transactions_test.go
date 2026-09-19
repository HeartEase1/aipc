package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestRechargeReservationRevalidatesFrozenOffer(t *testing.T) {
	for _, tc := range []struct {
		name, kind, budget   string
		unavailable, success bool
	}{
		{"campaign", "recharge", "10", false, true},
		{"budget exhausted", "recharge", "9.99", false, false},
		{"first recharge", "first_recharge", "10", false, true},
		{"first already used", "first_recharge", "10", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			mock.ExpectBegin()
			tx, err := client.Tx(context.Background())
			require.NoError(t, err)
			mock.ExpectQuery("SELECT kind.*FROM recharge_promotions.*FOR UPDATE").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"kind", "currency", "enabled", "start", "end", "min", "max", "discount", "cap", "budget"}).AddRow(tc.kind, "CNY", true, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), nil, nil, "10", nil, tc.budget))
			source := "campaign"
			if tc.kind == "first_recharge" {
				source = tc.kind
				mock.ExpectQuery("SELECT.*EXISTS.*FROM payment_orders").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"unavailable"}).AddRow(tc.unavailable))
			}
			if tc.success {
				mock.ExpectExec("UPDATE recharge_promotions SET reserved_amount").WithArgs("10", int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO recharge_promotion_claims.*order_id").WithArgs(int64(1), int64(7), int64(8), source, "10").WillReturnResult(sqlmock.NewResult(1, 1))
			}
			err = reserveRechargePromotionInTx(context.Background(), tx.Client(), 7, 8, &RechargeQuote{PromotionID: 1, OriginalAmount: "100", DiscountAmount: "10", DiscountSource: source, Currency: "CNY"})
			if tc.success {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, errRechargeQuoteChanged)
			}
			// Rolling the order transaction back also rolls back its budget and claim.
			mock.ExpectRollback()
			require.NoError(t, tx.Rollback())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRechargeRedemptionIsIdempotentAndRejectsReleasedClaims(t *testing.T) {
	for _, status := range []string{"reserved", "redeemed", "released"} {
		t.Run(status, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			mock.ExpectQuery("SELECT promotion_id, discount_amount, status.*FOR UPDATE").WithArgs(int64(8)).WillReturnRows(sqlmock.NewRows([]string{"promotion_id", "discount", "status"}).AddRow(1, "10", status))
			if status == "reserved" {
				mock.ExpectExec("UPDATE recharge_promotions SET reserved_amount").WithArgs("10", int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("UPDATE recharge_promotion_claims SET status='redeemed'").WithArgs(int64(8)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			err = redeemRechargePromotionInTx(context.Background(), client, 8)
			if status == "released" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRechargePromotionAdminValidation(t *testing.T) {
	start, end := time.Now(), time.Now().Add(time.Hour)
	base := RechargePromotionAdminInput{Kind: "recharge", Name: "Offer", SettlementCurrency: "CNY", Timezone: "Asia/Shanghai", StartsAt: &start, EndsAt: &end, DiscountPercent: "10"}
	_, err := validateRechargePromotionInput(base)
	require.NoError(t, err)
	for _, change := range []func(*RechargePromotionAdminInput){
		func(v *RechargePromotionAdminInput) { v.EndsAt = &start },
		func(v *RechargePromotionAdminInput) { v.Timezone = "not/a/zone" },
		func(v *RechargePromotionAdminInput) { v.DiscountPercent = "100" },
		func(v *RechargePromotionAdminInput) { v.DiscountPercent = "1e999999" },
		func(v *RechargePromotionAdminInput) { v.Kind = "subscription" },
		func(v *RechargePromotionAdminInput) { x := "-1"; v.BudgetAmount = &x },
		func(v *RechargePromotionAdminInput) { x := "0.001"; v.BudgetAmount = &x },
	} {
		in := base
		change(&in)
		_, err := validateRechargePromotionInput(in)
		require.Error(t, err)
	}
}

func TestRechargeRedemptionRollsBackWhenPaidTransitionFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	svc := &PaymentService{entClient: client, sqlDB: db}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM users.*FOR UPDATE").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	mock.ExpectQuery("SELECT promotion_id, discount_amount, status.*FOR UPDATE").WithArgs(int64(8)).WillReturnRows(sqlmock.NewRows([]string{"promotion_id", "discount", "status"}).AddRow(1, "10", "reserved"))
	mock.ExpectExec("UPDATE recharge_promotions SET reserved_amount").WithArgs("10", int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE recharge_promotion_claims SET status='redeemed'").WithArgs(int64(8)).WillReturnResult(sqlmock.NewResult(0, 1))
	writeErr := errors.New("payment write failed")
	mock.ExpectExec(`UPDATE "payment_orders" SET`).WillReturnError(writeErr)
	mock.ExpectRollback()
	err = svc.toPaid(context.Background(), &dbent.PaymentOrder{ID: 8, UserID: 7, OrderType: payment.OrderTypeBalance, Status: OrderStatusPending}, "trade", 90, payment.TypeAlipay)
	require.ErrorIs(t, err, writeErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRechargeReservationReleaseRechecksPaymentAndBudgetAtomically(t *testing.T) {
	for _, tc := range []struct {
		name       string
		releasable bool
		budgetRows int64
		wantError  bool
	}{
		{"closed and unpaid", true, 1, false},
		{"callback already paid or grace not reached", false, 0, false},
		{"budget invariant violated", true, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			svc := &PaymentService{entClient: client, sqlDB: db}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT id FROM users.*FOR UPDATE").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
			rows := sqlmock.NewRows([]string{"promotion_id", "discount_amount"})
			if tc.releasable {
				rows.AddRow(1, "10")
			}
			mock.ExpectQuery("UPDATE recharge_promotion_claims.*closed_confirmed_at <=.*o.paid_at IS NULL.*RETURNING").WithArgs(int64(8), sqlmock.AnyArg()).WillReturnRows(rows)
			if tc.releasable {
				mock.ExpectExec("UPDATE recharge_promotions SET reserved_amount").WithArgs("10", int64(1)).WillReturnResult(sqlmock.NewResult(0, tc.budgetRows))
			}
			if tc.releasable && !tc.wantError {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			err = svc.releaseClosedRechargeReservation(context.Background(), 8, 7)
			if tc.wantError {
				require.ErrorContains(t, err, "budget invariant")
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

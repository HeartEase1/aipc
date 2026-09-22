package service

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func bonusInput() RechargeBonusCampaignInput {
	max := "100"
	return RechargeBonusCampaignInput{Title: "活动", SettlementCurrency: "CNY", Frequency: "daily_first", StartsAt: time.Now(), EndsAt: time.Now().Add(time.Hour), Tiers: []RechargeBonusTierInput{{MinAmount: "0", MaxAmount: &max, BonusPercent: "10"}, {MinAmount: "100", BonusPercent: "15"}}}
}
func TestBonusValidationAndBoundaries(t *testing.T) {
	in, err := validateBonus(bonusInput())
	require.NoError(t, err)
	require.Equal(t, "Asia/Shanghai", in.Timezone)
	for _, change := range []func(*RechargeBonusCampaignInput){
		func(v *RechargeBonusCampaignInput) { v.Tiers[1].MinAmount = "99.99" }, func(v *RechargeBonusCampaignInput) { v.Tiers[0].MaxAmount = nil },
		func(v *RechargeBonusCampaignInput) { v.Tiers[0].BonusPercent = "101" }, func(v *RechargeBonusCampaignInput) { v.Tiers[0].BonusPercent = "1e2" },
		func(v *RechargeBonusCampaignInput) { v.Tiers[0].MinAmount = "-1" }, func(v *RechargeBonusCampaignInput) { v.EndsAt = v.StartsAt },
		func(v *RechargeBonusCampaignInput) { v.Timezone = "bad-zone" }, func(v *RechargeBonusCampaignInput) { v.Frequency = "invalid" },
	} {
		v := bonusInput()
		change(&v)
		_, err := validateBonus(v)
		require.Error(t, err)
	}
	for _, tc := range []struct{ face, multiplier, percent, want string }{{"100", "10", "10", "100"}, {"100", "0.14", "5", "0.7"}, {"0.01", "1", "5", "0"}, {"123.45", "1.3", "7.25", "11.64"}} {
		amount, _ := decimal.NewFromString(tc.face)
		m, _ := decimal.NewFromString(tc.multiplier)
		p, _ := decimal.NewFromString(tc.percent)
		require.Equal(t, tc.want, bonusBalance(amount, m, p).String())
	}
}
func TestBonusPeriodsAndRefunds(t *testing.T) {
	a := time.Date(2026, 9, 22, 15, 59, 59, 0, time.UTC)
	b := a.Add(time.Second)
	p1, err := bonusPeriod("daily_first", "Asia/Shanghai", a, 1)
	require.NoError(t, err)
	p2, err := bonusPeriod("daily_first", "Asia/Shanghai", b, 2)
	require.NoError(t, err)
	require.NotEqual(t, p1, p2)
	p1, _ = bonusPeriod("campaign_first", "Asia/Shanghai", a, 1)
	p2, _ = bonusPeriod("campaign_first", "Asia/Shanghai", b, 2)
	require.Equal(t, p1, p2)
	p1, _ = bonusPeriod("every_payment", "Asia/Shanghai", a, 1)
	p2, _ = bonusPeriod("every_payment", "Asia/Shanghai", a, 2)
	require.NotEqual(t, p1, p2)
	order := &dbent.PaymentOrder{Amount: 1000, PricingSnapshot: map[string]any{"bonus": &RechargeBonusSnapshot{CampaignID: 1, Expected: "100", Credited: true, Refunded: "25"}}}
	require.Equal(t, 25.0, bonusRefundDue(order, 500))
	require.Equal(t, 75.0, bonusRefundDue(order, 1000))
	require.Zero(t, bonusRefundDue(order, 250))
	require.Zero(t, bonusRefundDue(&dbent.PaymentOrder{Amount: 1000}, 1000))
}
func bonusMock(t *testing.T) (*dbent.Client, sqlmock.Sqlmock) {
	t.Helper()
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, m.ExpectationsWereMet()); _ = db.Close() })
	return dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db))), m
}
func TestBonusConfirmationClaimsOnlyOnePeriodAndFreezesResult(t *testing.T) {
	for _, won := range []bool{true, false} {
		t.Run(map[bool]string{true: "winner", false: "already claimed"}[won], func(t *testing.T) {
			client, m := bonusMock(t)
			now := time.Now()
			o := &dbent.PaymentOrder{ID: 8, UserID: 7, Amount: 1000, ExpiresAt: now.Add(time.Minute), PricingSnapshot: map[string]any{"bonus": &RechargeBonusSnapshot{CampaignID: 4, Frequency: "campaign_first", Timezone: "Asia/Shanghai", Expected: "100"}}}
			affected := int64(0)
			if won {
				affected = 1
			}
			m.ExpectExec("INSERT INTO recharge_bonus_claims").WithArgs(int64(8), int64(4), int64(7), "campaign", "100").WillReturnResult(sqlmock.NewResult(0, affected))
			m.ExpectExec("UPDATE payment_orders SET pricing_snapshot").WillReturnResult(sqlmock.NewResult(0, 1))
			require.NoError(t, claimRechargeBonus(context.Background(), client, o, now))
			b := orderBonus(o)
			require.True(t, b.Confirmed)
			require.Equal(t, won, b.Awarded)
			require.False(t, b.Credited)
		})
	}
}
func TestBonusLatePaymentDoesNotClaim(t *testing.T) {
	client, m := bonusMock(t)
	o := &dbent.PaymentOrder{ID: 1, ExpiresAt: time.Now().Add(-time.Minute), PricingSnapshot: map[string]any{"bonus": &RechargeBonusSnapshot{CampaignID: 4, Expected: "100"}}}
	m.ExpectExec("UPDATE payment_orders SET pricing_snapshot").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, claimRechargeBonus(context.Background(), client, o, time.Now()))
	require.False(t, orderBonus(o).Awarded)
}

type bonusBalanceRepo struct {
	UserRepository
	calls int
	err   error
}

func (r *bonusBalanceRepo) AdjustBalance(ctx context.Context, id int64, amount float64) (BalanceChange, error) {
	r.calls++
	if dbent.TxFromContext(ctx) == nil {
		return BalanceChange{}, errors.New("missing transaction")
	}
	return BalanceChange{New: amount}, r.err
}
func TestBonusCreditRetriesWithoutDoubleCredit(t *testing.T) {
	client, m := bonusMock(t)
	repo := &bonusBalanceRepo{}
	svc := &PaymentService{entClient: client, userRepo: repo}
	now := time.Now()
	o := &dbent.PaymentOrder{ID: 8, UserID: 7, Amount: 1000, PricingSnapshot: map[string]any{"bonus": &RechargeBonusSnapshot{CampaignID: 4, Confirmed: true, Awarded: true, Expected: "100", Percent: "10"}}}
	for attempt := 0; attempt < 2; attempt++ {
		m.ExpectBegin()
		m.ExpectQuery("SELECT id FROM users").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
		m.ExpectQuery("SELECT id FROM payment_orders").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(8))
		row := sqlmock.NewRows([]string{"credited_at"})
		if attempt == 0 {
			row.AddRow(nil)
		} else {
			row.AddRow(now)
		}
		m.ExpectQuery("SELECT credited_at").WillReturnRows(row)
		if attempt == 0 {
			m.ExpectExec("UPDATE recharge_bonus_claims SET credited_at").WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery("INSERT INTO .payment_audit_logs.").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		}
		m.ExpectExec("UPDATE payment_orders SET pricing_snapshot").WillReturnResult(sqlmock.NewResult(0, 1))
		m.ExpectCommit()
		require.NoError(t, svc.applyRechargeBonus(context.Background(), o, &paymentFulfillmentLease{version: now}))
		require.True(t, orderBonus(o).Credited)
	}
	require.Equal(t, 1, repo.calls)
	// A gifted order skips the entire affiliate path, including when retried.
	require.NoError(t, svc.applyAffiliateRebateForOrder(context.Background(), o))
}
func TestBonusPosterRejectsCorruptAndOversizedImages(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	mime, err := validateBonusPoster(buf.Bytes())
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)
	for _, data := range [][]byte{[]byte("<svg/>"), buf.Bytes()[:26], make([]byte, BonusPosterMaxBytes+1)} {
		_, err := validateBonusPoster(data)
		require.Error(t, err)
	}
}

func TestBonusQuoteUsesFaceAmountAndTierEdges(t *testing.T) {
	for _, tc := range []struct{ amount, want string }{{"100", "100.00"}, {"100.01", "150.02"}, {"0.01", ""}} {
		db, m, err := sqlmock.New()
		require.NoError(t, err)
		m.ExpectQuery("SELECT c.id.*NOT EXISTS.*marketing_user_exclusions").WithArgs("CNY", int64(7)).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "copy", "poster_id", "currency", "enabled", "frequency", "starts", "ends", "timezone", "tiers"}).AddRow(1, "gift", "", nil, "CNY", true, "every_payment", time.Now(), time.Now().Add(time.Hour), "Asia/Shanghai", `[{"min_amount":"0","max_amount":"100","bonus_percent":"10"},{"min_amount":"100","max_amount":null,"bonus_percent":"15"}]`))
		amount, _ := decimal.NewFromString(tc.amount)
		q := RechargeQuote{CreditedAmount: amount.Mul(decimal.NewFromInt(10)).String(), PayAmount: "discounted price unrelated to gift"}
		err = (&PaymentService{sqlDB: db}).attachRechargeBonus(context.Background(), 7, amount, "CNY", &PaymentConfig{BalanceRechargeMultiplier: 10}, &q)
		require.NoError(t, err)
		// 0.01 face * 10 * 10% = 0.01 balance.
		want := tc.want
		if tc.amount == "0.01" {
			want = "0.01"
		}
		require.Equal(t, want, q.BonusCreditedAmount)
		require.NoError(t, m.ExpectationsWereMet())
		_ = db.Close()
	}
}

type bonusIdentityEncryptor struct{}

func (bonusIdentityEncryptor) Encrypt(s string) (string, error) { return s, nil }
func (bonusIdentityEncryptor) Decrypt(s string) (string, error) { return s, nil }

type bonusTestStorage struct {
	deleted []string
	err     error
}

func (s *bonusTestStorage) Save(context.Context, string, string, []byte) (string, error) {
	return "url", nil
}
func (s *bonusTestStorage) URL(context.Context, string) (string, error) { return "fresh-url", nil }
func (s *bonusTestStorage) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return s.err
}
func TestBonusPosterCleanupRetainsFailuresForRetry(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{true: "failure", false: "success"}[fail], func(t *testing.T) {
			db, m, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			store := &bonusTestStorage{}
			if fail {
				store.err = errors.New("unavailable")
			}
			svc := &PaymentService{sqlDB: db, bonusImages: &ImageStorageSettingService{encryptor: bonusIdentityEncryptor{}, factory: func(context.Context, *config.ImageStorageConfig) (ImageStorage, error) { return store, nil }}}
			m.ExpectBegin()
			m.ExpectExec("SELECT pg_advisory_xact_lock").WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec("UPDATE recharge_bonus_campaigns.*30 days").WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery("SELECT p.id.*NOT EXISTS.*poster_id=p.id").WillReturnRows(sqlmock.NewRows([]string{"id", "key", "cfg"}).AddRow(7, "images/recharge-bonus/old", "{}"))
			m.ExpectExec("UPDATE recharge_bonus_posters SET state='deleting'").WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec("DELETE FROM recharge_bonus_impressions").WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit()
			if fail {
				m.ExpectExec("UPDATE recharge_bonus_posters SET failures=failures\\+1").WillReturnResult(sqlmock.NewResult(0, 1))
			} else {
				m.ExpectExec("UPDATE recharge_bonus_posters SET state='deleted'").WillReturnResult(sqlmock.NewResult(0, 1))
			}
			require.NoError(t, svc.CleanupBonusPosters(context.Background()))
			require.Equal(t, []string{"images/recharge-bonus/old"}, store.deleted)
			require.NoError(t, m.ExpectationsWereMet())
		})
	}
}

// Capture the transaction's persisted snapshot for replay assertions.
type bonusSnapshotArg struct{ snapshot *map[string]any }

func (a bonusSnapshotArg) Match(v driver.Value) bool {
	raw, ok := v.(string)
	if !ok {
		return false
	}
	return json.Unmarshal([]byte(raw), a.snapshot) == nil
}
func expectBonusRefundLock(m sqlmock.Sqlmock, o *dbent.PaymentOrder) {
	m.ExpectBegin()
	m.ExpectQuery("SELECT id FROM users").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(o.UserID))
	m.ExpectQuery("SELECT id FROM payment_orders").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(o.ID))
	raw, _ := json.Marshal(o.PricingSnapshot)
	m.ExpectQuery("SELECT .*payment_orders.*WHERE").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "amount", "pay_amount", "status", "pricing_snapshot"}).AddRow(o.ID, o.UserID, o.Amount, o.PayAmount, o.Status, string(raw)))
}
func TestBonusRefundCompletionAndDuplicateAreIdempotent(t *testing.T) {
	client, m := bonusMock(t)
	svc := &PaymentService{entClient: client}
	o := &dbent.PaymentOrder{ID: 8, UserID: 7, Amount: 1000, PayAmount: 100, Status: OrderStatusRefunding, PricingSnapshot: map[string]any{"bonus": &RechargeBonusSnapshot{CampaignID: 4, Credited: true, Awarded: true, Expected: "100", Refunded: "0", Refund: &RechargeBonusRefund{ID: "stable-refund", Principal: "500", Before: "0", BonusDue: "50", Deducted: "550", Deduct: true}}}}
	plan := &RefundPlan{OrderID: o.ID, Order: o}
	expectBonusRefundLock(m, o)
	var snapshot map[string]any
	m.ExpectExec("UPDATE payment_orders SET pricing_snapshot").WithArgs(bonusSnapshotArg{&snapshot}, o.ID).WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectExec("UPDATE .payment_orders.").WillReturnResult(sqlmock.NewResult(0, 1))
	m.ExpectQuery("SELECT .*payment_orders.*WHERE").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(o.ID))
	m.ExpectQuery("INSERT INTO .payment_audit_logs.").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	m.ExpectCommit()
	result, err := svc.finishBonusRefund(context.Background(), plan, &payment.RefundResponse{Status: payment.ProviderStatusSuccess})
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, 550.0, result.BalanceDeducted)
	o.PricingSnapshot = snapshot
	o.Status = OrderStatusPartiallyRefunded
	require.Equal(t, "50.00", orderBonus(o).Refunded)
	require.Equal(t, "500.00", orderBonus(o).RefundedPrincipal)
	expectBonusRefundLock(m, o)
	m.ExpectRollback()
	result, err = svc.finishBonusRefund(context.Background(), plan, &payment.RefundResponse{Status: payment.ProviderStatusSuccess})
	require.NoError(t, err)
	require.True(t, result.Success)
}
func TestBonusRefundGatewayRoundingUsesCumulativeAmounts(t *testing.T) {
	o := &dbent.PaymentOrder{Amount: 3, PayAmount: 1, SettlementCurrency: "CNY"}
	require.InDelta(t, .33, bonusGatewayRefundAmount(o, &RechargeBonusRefund{Before: "0", Principal: "1"}), 1e-9)
	require.InDelta(t, .34, bonusGatewayRefundAmount(o, &RechargeBonusRefund{Before: "1", Principal: "1"}), 1e-9)
	require.InDelta(t, .33, bonusGatewayRefundAmount(o, &RechargeBonusRefund{Before: "2", Principal: "1"}), 1e-9)
}

func TestBonusRefundFailureRestoresOnlyOnce(t *testing.T) {
	client, m := bonusMock(t)
	repo := &bonusBalanceRepo{}
	svc := &PaymentService{entClient: client, userRepo: repo}
	o := &dbent.PaymentOrder{ID: 8, UserID: 7, Amount: 1000, Status: OrderStatusRefunding, PricingSnapshot: map[string]any{"bonus": &RechargeBonusSnapshot{CampaignID: 4, Credited: true, Expected: "100", Refunded: "0", Refund: &RechargeBonusRefund{ID: "stable", Principal: "500", Before: "0", BonusDue: "50", Deducted: "550", Deduct: true}}}}
	plan := &RefundPlan{OrderID: o.ID, Order: o}
	for attempt := 0; attempt < 2; attempt++ {
		expectBonusRefundLock(m, o)
		var saved map[string]any
		m.ExpectExec("UPDATE payment_orders SET pricing_snapshot").WithArgs(bonusSnapshotArg{&saved}, o.ID).WillReturnResult(sqlmock.NewResult(0, 1))
		m.ExpectExec("UPDATE .payment_orders.").WillReturnResult(sqlmock.NewResult(0, 1))
		m.ExpectQuery("SELECT .*payment_orders.*WHERE").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(o.ID))
		m.ExpectQuery("INSERT INTO .payment_audit_logs.").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		m.ExpectCommit()
		result, err := svc.finishBonusRefund(context.Background(), plan, &payment.RefundResponse{Status: payment.ProviderStatusFailed})
		require.NoError(t, err)
		require.False(t, result.Success)
		o.PricingSnapshot = saved
		o.Status = OrderStatusRefundFailed
		require.True(t, orderBonus(o).Refund.Restored)
		require.Equal(t, "0", orderBonus(o).Refunded)
	}
	require.Equal(t, 1, repo.calls)
}
func TestBonusRefundDebitRespectsFrozenBalanceAndForce(t *testing.T) {
	for _, force := range []bool{false, true} {
		client, m := bonusMock(t)
		repo := &bonusBalanceRepo{}
		svc := &PaymentService{entClient: client, userRepo: repo}
		m.ExpectBegin()
		tx, err := client.Tx(context.Background())
		require.NoError(t, err)
		m.ExpectQuery("SELECT balance-frozen_balance").WillReturnRows(sqlmock.NewRows([]string{"available"}).AddRow("40"))
		r := &RechargeBonusRefund{Principal: "500", BonusDue: "50", Deduct: true, Force: force}
		err = svc.bonusRefundDebit(context.Background(), tx, &dbent.PaymentOrder{UserID: 7}, r)
		if force {
			require.NoError(t, err)
			require.Equal(t, "40.00", r.Deducted)
			require.Equal(t, 1, repo.calls)
		} else {
			require.ErrorIs(t, err, ErrBalanceNegative)
			require.Zero(t, repo.calls)
		}
		m.ExpectRollback()
		require.NoError(t, tx.Rollback())
	}
}
func TestBonusCreditFailureRollsBackAndDoesNotSuppressRetry(t *testing.T) {
	client, m := bonusMock(t)
	repo := &bonusBalanceRepo{err: errors.New("database unavailable")}
	svc := &PaymentService{entClient: client, userRepo: repo}
	now := time.Now()
	o := &dbent.PaymentOrder{ID: 8, UserID: 7, PricingSnapshot: map[string]any{"bonus": &RechargeBonusSnapshot{CampaignID: 4, Confirmed: true, Awarded: true, Expected: "100"}}}
	m.ExpectBegin()
	m.ExpectQuery("SELECT id FROM users").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	m.ExpectQuery("SELECT id FROM payment_orders").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(8))
	m.ExpectQuery("SELECT credited_at").WillReturnRows(sqlmock.NewRows([]string{"credited_at"}).AddRow(nil))
	m.ExpectRollback()
	require.Error(t, svc.applyRechargeBonus(context.Background(), o, &paymentFulfillmentLease{version: now}))
	require.False(t, orderBonus(o).Credited)
}

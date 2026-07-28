package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func validBenefitGrantInput() BenefitGrantPreviewInput {
	return BenefitGrantPreviewInput{
		GrantType:           BenefitGrantTypeCompensation,
		GrantMode:           BenefitGrantModePercentage24h,
		AudienceType:        BenefitGrantAudienceSelected,
		UserIDs:             []int64{3, 2, 3},
		Percentage:          "12.34567891",
		MinAmount:           "0.1",
		PerUserCap:          "10",
		TotalBudgetCap:      "100",
		Reason:              "service incident",
		NotificationTitle:   "Balance received",
		NotificationContent: "{{amount}} / {{reason}} / {{balance}} / {{site_name}}",
		ActorID:             9,
	}
}

func TestValidateBenefitGrantInputNormalizesDecimalRulesAndRecipients(t *testing.T) {
	validated, err := validateBenefitGrantInput(validBenefitGrantInput())
	require.NoError(t, err)
	require.Equal(t, []int64{2, 3}, validated.UserIDs)
	require.Equal(t, "12.34567891", validated.percentage.StringFixed(8))
	require.Equal(t, "0.10000000", validated.minAmount.StringFixed(8))
	require.Equal(t, "10.00000000", validated.perUserCap.StringFixed(8))

	input := validBenefitGrantInput()
	input.Percentage = "0.009"
	_, err = validateBenefitGrantInput(input)
	require.Error(t, err)

	input = validBenefitGrantInput()
	input.Percentage = "100.00000001"
	_, err = validateBenefitGrantInput(input)
	require.Error(t, err)

	input = validBenefitGrantInput()
	input.MinAmount = "11"
	_, err = validateBenefitGrantInput(input)
	require.Error(t, err)
}

func TestParseBenefitDecimalRoundsAmountsToEightPlaces(t *testing.T) {
	value, err := parseBenefitDecimal("1.234567895", "amount", true)
	require.NoError(t, err)
	require.Equal(t, "1.23456790", value.StringFixed(8))

	_, err = parseBenefitDecimal("0.000000001", "amount", true)
	require.Error(t, err)
}

func TestRenderBenefitGrantTemplateReplacesOnlySupportedVariables(t *testing.T) {
	rendered := renderBenefitGrantTemplate(
		"{{amount}}|{{reason}}|{{balance}}|{{site_name}}|{{unknown}}",
		"1.25", "incident", "8.75", "IPCAI",
	)
	require.Equal(t, "1.25|incident|8.75|IPCAI|{{unknown}}", rendered)
}

func TestBenefitGrantOverBudgetUsesDecimalComparison(t *testing.T) {
	capValue := "10.00000000"
	require.False(t, benefitGrantOverBudget(&BenefitGrantBatch{TotalAmount: "10.00000000", TotalBudgetCap: &capValue}))
	require.True(t, benefitGrantOverBudget(&BenefitGrantBatch{TotalAmount: "10.00000001", TotalBudgetCap: &capValue}))
}

func TestProcessOneBenefitGrantItemCreditsBalanceAndCompletesItemInOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT i.id, i.batch_id, i.user_id, i.amount::text`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "batch_id", "user_id", "amount"}).AddRow(5, 12, 42, "2.50000000"))
	mock.ExpectExec(`UPDATE benefit_grant_batches`).WithArgs(int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT balance::text`).WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("10.00000000"))
	mock.ExpectQuery(`UPDATE users\s+SET balance = balance \+ \$1::numeric`).
		WithArgs("2.50000000", int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("12.50000000"))
	mock.ExpectExec(`UPDATE benefit_grant_items`).
		WithArgs(int64(5), "10.00000000", "12.50000000").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	grantService := NewBenefitGrantService(db, nil, nil, nil)
	processed, err := grantService.processOneBenefitGrantItem(context.Background())
	require.NoError(t, err)
	require.True(t, processed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProcessOneBenefitGrantItemSkipsUserWhoBecameIneligible(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT i.id, i.batch_id, i.user_id, i.amount::text`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "batch_id", "user_id", "amount"}).AddRow(5, 12, 42, "2.50000000"))
	mock.ExpectExec(`UPDATE benefit_grant_batches`).WithArgs(int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT balance::text`).WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(`UPDATE benefit_grant_items`).WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SET skipped_count = skipped_count \+ 1`).WithArgs(int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	grantService := NewBenefitGrantService(db, nil, nil, nil)
	processed, err := grantService.processOneBenefitGrantItem(context.Background())
	require.NoError(t, err)
	require.True(t, processed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteUsesDatabaseClockAtPreviewExpiryBoundary(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expiresAt := time.Date(2026, 7, 28, 12, 10, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status, expires_at, total_amount::text, total_budget_cap::text`).WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "expires_at", "total_amount", "total_budget_cap"}).
			AddRow(BenefitGrantStatusDraft, expiresAt, "5.00000000", nil))
	mock.ExpectQuery(`SELECT NOW\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"now"}).AddRow(expiresAt))
	mock.ExpectExec(`UPDATE benefit_grant_batches SET status = 'expired'`).WithArgs(int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	grantService := NewBenefitGrantService(db, nil, nil, nil)
	_, err = grantService.Execute(context.Background(), 12, 9)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFixedGrantCapUsesExactDecimalMinimum(t *testing.T) {
	fixed := decimal.RequireFromString("12.00000000")
	capValue := decimal.RequireFromString("5.00000000")
	require.Equal(t, "5.00000000", capBenefitGrantAmount(fixed, &capValue).StringFixed(8))
	require.Equal(t, "12.00000000", capBenefitGrantAmount(fixed, nil).StringFixed(8))
}

func TestPreviewPercentageSnapshotsOnlyWalletSpendingInsideLockedWindow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	windowStart := now.Add(-24 * time.Hour)
	expiresAt := now.Add(10 * time.Minute)
	input := validBenefitGrantInput()
	input.UserIDs = []int64{2, 3}
	input.Percentage = "10"
	input.MinAmount = "1"
	input.PerUserCap = "5"

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT NOW\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"now"}).AddRow(now))
	mock.ExpectQuery(`INSERT INTO benefit_grant_batches`).
		WithArgs(
			BenefitGrantTypeCompensation, BenefitGrantModePercentage24h, BenefitGrantAudienceSelected,
			nil, "10.00000000", "1.00000000", "5.00000000", "100.00000000",
			input.Reason, input.NotificationTitle, input.NotificationContent,
			windowStart, now, input.ActorID, expiresAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(77))
	mock.ExpectExec(`WITH eligible AS .*ul.billing_type = 0.*ul.actual_cost > 0.*ROUND\(final_amount, 8\)`).
		WithArgs(int64(77), windowStart, now, "10.00000000", "1.00000000", "5.00000000", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(`SELECT COUNT\(\*\)::integer`).WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "base", "amount", "average", "max"}).
			AddRow(2, "30.00000000", "3.00000000", "1.50000000", "2.00000000"))
	mock.ExpectExec(`UPDATE benefit_grant_batches`).
		WithArgs(int64(77), 2, 0, "30.00000000", "3.00000000", "1.50000000", "2.00000000").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	batchColumns := []string{
		"id", "grant_type", "grant_mode", "audience_type", "fixed_amount", "percentage",
		"min_amount", "per_user_cap", "total_budget_cap", "reason", "notification_title",
		"notification_content", "window_start", "window_end", "status", "eligible_count",
		"skipped_count", "success_count", "failed_count", "total_base_cost", "total_amount",
		"distributed_amount", "average_amount", "max_amount", "created_by", "executed_by",
		"expires_at", "started_at", "completed_at", "created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT id, grant_type, grant_mode, audience_type`).WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(batchColumns).AddRow(
			77, BenefitGrantTypeCompensation, BenefitGrantModePercentage24h, BenefitGrantAudienceSelected,
			nil, "10.00000000", "1.00000000", "5.00000000", "100.00000000",
			input.Reason, input.NotificationTitle, input.NotificationContent, windowStart, now,
			BenefitGrantStatusDraft, 2, 0, 0, 0, "30.00000000", "3.00000000", "0.00000000",
			"1.50000000", "2.00000000", input.ActorID, nil, expiresAt, nil, nil, now, now,
		))

	grantService := NewBenefitGrantService(db, nil, nil, nil)
	batch, err := grantService.Preview(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, int64(77), batch.ID)
	require.Equal(t, "3.00000000", batch.TotalAmount)
	require.Equal(t, windowStart, *batch.WindowStart)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteRejectsBatchAboveBudgetBeforeAnyBalanceWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status, expires_at, total_amount::text, total_budget_cap::text`).WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "expires_at", "total_amount", "total_budget_cap"}).
			AddRow(BenefitGrantStatusDraft, now.Add(time.Minute), "10.00000001", "10.00000000"))
	mock.ExpectQuery(`SELECT NOW\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"now"}).AddRow(now))
	mock.ExpectRollback()

	grantService := NewBenefitGrantService(db, nil, nil, nil)
	_, err = grantService.Execute(context.Background(), 12, 9)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAndReadUserGrantsAreScopedToAuthenticatedUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	createdAt := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM benefit_grant_items i WHERE i.user_id = \$1 AND i.status = 'succeeded' AND i.read_at IS NULL`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT i.id, i.batch_id, b.grant_type, i.amount::text`).
		WithArgs(int64(42), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "batch_id", "grant_type", "amount", "balance_after", "reason",
			"title", "content", "read_at", "created_at",
		}).AddRow(
			8, 7, BenefitGrantTypeWelfare, "1.50000000", "11.50000000", "welcome",
			"{{site_name}} grant", "{{amount}} / {{balance}} / {{reason}}", nil, createdAt,
		))
	mock.ExpectExec(`UPDATE benefit_grant_items`).WithArgs(int64(8), int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	grantService := NewBenefitGrantService(db, nil, nil, nil)
	result, err := grantService.ListUserGrants(context.Background(), 42, 1, 20, true)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "Sub2API grant", result.Items[0].Title)
	require.Equal(t, "1.50000000 / 11.50000000 / welcome", result.Items[0].Content)
	require.NoError(t, grantService.MarkUserGrantRead(context.Background(), 42, 8))
	require.NoError(t, mock.ExpectationsWereMet())
}

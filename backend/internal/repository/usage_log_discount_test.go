package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogMarketingFieldsRoundTrip(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	campaignID, factor, original := int64(17), 0.8, 2.0
	log := &service.UsageLog{
		UserID: 1, APIKeyID: 2, AccountID: 3, RequestID: "discount-round-trip", Model: "gpt-5",
		InputTokens: 100, OutputTokens: 10, TotalCost: 5, ActualCost: 8,
		DiscountCampaignID: &campaignID, DiscountFactor: &factor, OriginalRateMultiplier: &original,
		DiscountAmount: 2, CreatedAt: time.Now().UTC(),
	}
	prepared := prepareUsageLogInsert(log)
	args := make([]driver.Value, len(prepared.args))
	for i, arg := range prepared.args {
		var err error
		args[i], err = driver.DefaultParameterConverter.ConvertValue(arg)
		require.NoError(t, err)
	}
	require.Equal(t, campaignID, args[usageLogArgumentIndex(t, "discount_campaign_id")])
	require.Equal(t, factor, args[usageLogArgumentIndex(t, "discount_factor")])
	require.Equal(t, original, args[usageLogArgumentIndex(t, "original_rate_multiplier")])
	require.Equal(t, float64(2), args[usageLogArgumentIndex(t, "discount_amount")])
	values := append([]driver.Value{int64(99)}, args...)
	mock.ExpectQuery("SELECT .* FROM usage_logs WHERE id").WithArgs(int64(99)).WillReturnRows(
		sqlmock.NewRows(strings.Split(usageLogSelectColumns, ", ")).AddRow(values...),
	)
	got, err := repo.GetByID(context.Background(), 99)
	require.NoError(t, err)
	require.Equal(t, &campaignID, got.DiscountCampaignID)
	require.Equal(t, &factor, got.DiscountFactor)
	require.Equal(t, &original, got.OriginalRateMultiplier)
	require.Equal(t, float64(2), got.DiscountAmount)
	require.Equal(t, float64(8), got.ActualCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

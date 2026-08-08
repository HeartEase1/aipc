package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenefitSubscriptionAndDiscountCampaignMigration(t *testing.T) {
	content, err := FS.ReadFile("196_benefit_subscription_and_discount_campaigns.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "include_subscription BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "subscription_percentage NUMERIC(11,8)")
	require.Contains(t, sql, "balance_amount = amount")
	require.Contains(t, sql, "base_cost <> 0 OR amount <> 0")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS discount_campaigns")
	require.Contains(t, sql, "schedule_type IN ('one_time', 'weekly')")
	require.Contains(t, sql, "discount_factor > 0 AND discount_factor <= 1")
	require.Contains(t, sql, "discount_campaign_id BIGINT REFERENCES discount_campaigns(id)")
	require.Contains(t, sql, "ADD CONSTRAINT usage_logs_discount_campaign_id_fkey FOREIGN KEY (discount_campaign_id)")
	require.Contains(t, sql, "original_rate_multiplier NUMERIC(10,6)")
	require.Contains(t, sql, "discount_amount NUMERIC(20,10) NOT NULL DEFAULT 0")
}

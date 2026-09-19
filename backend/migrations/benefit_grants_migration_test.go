package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenefitGrantMigrationKeepsGrantsIndependentFromRechargeAccounting(t *testing.T) {
	content, err := FS.ReadFile("191_benefit_grants.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS benefit_grant_batches")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS benefit_grant_items")
	require.Contains(t, sql, "percentage NUMERIC(11,8)")
	require.Contains(t, sql, "amount NUMERIC(20,8) NOT NULL CHECK (amount > 0)")
	require.Contains(t, sql, "UNIQUE (batch_id, user_id)")
	require.Contains(t, sql, "status IN ('pending', 'succeeded', 'failed', 'skipped_ineligible')")
	require.NotContains(t, sql, "ALTER TABLE users")
	require.NotContains(t, sql, "user_affiliate_ledger")
	require.NotContains(t, sql, "redeem_codes")
}

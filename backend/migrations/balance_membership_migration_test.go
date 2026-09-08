package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBalanceMembershipMigrationRemainsImmutableAndSafetyFollowupExists(t *testing.T) {
	base, err := FS.ReadFile("238_balance_membership_and_recharge_promotions.sql")
	require.NoError(t, err)
	baseSQL := strings.ToLower(string(base))
	require.Contains(t, baseSQL, "on delete cascade")
	require.Contains(t, baseSQL, "coalesce(original_amount, amount)")
	require.NotContains(t, baseSQL, "closed_confirmed_at")
	followup, err := FS.ReadFile("239_balance_membership_promotion_claim_safety.sql")
	require.NoError(t, err)
	followupSQL := strings.ToLower(string(followup))
	require.Contains(t, followupSQL, "add column if not exists closed_confirmed_at")
	require.Contains(t, followupSQL, "on delete restrict")
	require.Contains(t, followupSQL, "idx_recharge_promotion_claims_reconcile")
}

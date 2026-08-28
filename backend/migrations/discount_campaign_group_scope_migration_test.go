package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscountCampaignGroupScopeMigration(t *testing.T) {
	content, err := FS.ReadFile("230_discount_campaign_group_scope.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS group_ids BIGINT[] NOT NULL DEFAULT '{}'")
}

package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscountCampaignDescriptionMigration(t *testing.T) {
	content, err := FS.ReadFile("197_add_discount_campaign_description.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT ''")
}

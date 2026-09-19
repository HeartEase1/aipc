package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeysFastModeMigration(t *testing.T) {
	content, err := FS.ReadFile("229_api_keys_fast_mode.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS fast_mode BOOLEAN NOT NULL DEFAULT FALSE")
}

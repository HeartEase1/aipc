package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration183RestoresOptionalUsernameColumn(t *testing.T) {
	content, err := FS.ReadFile("183_restore_users_username.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(100) NOT NULL DEFAULT ''")
}

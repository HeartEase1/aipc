package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorV2CNPlatformsMigration(t *testing.T) {
	content, err := FS.ReadFile("232_channel_monitor_v2_add_cn_platforms.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	for _, platform := range []string{"kimi", "zhipu", "deepseek"} {
		require.Contains(t, sql, "('"+platform+"',")
	}
	require.Contains(t, sql, "config.platforms ||")
	require.Contains(t, sql, "jsonb_array_elements(config.platforms)")
	require.Contains(t, sql, "LOWER(TRIM(existing ->> 'platform')) = required.platform")
	require.Contains(t, sql, "WHERE NOT EXISTS")
	require.Contains(t, sql, "jsonb_typeof(config.platforms) = 'array'")
	require.Contains(t, sql, "version = config.version + 1")
	require.NotContains(t, sql, "jsonb_set(")
}

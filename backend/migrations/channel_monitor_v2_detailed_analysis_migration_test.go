package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorV2DetailedAnalysisMigration(t *testing.T) {
	content, err := FS.ReadFile("231_channel_monitor_v2_detailed_analysis.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "VALUES ('channel_monitor_v2_detailed_analysis_enabled', 'false')")
	require.Contains(t, sql, "ON CONFLICT (key) DO NOTHING")
}

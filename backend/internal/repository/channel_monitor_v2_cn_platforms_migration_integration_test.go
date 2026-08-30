//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorV2CNPlatformsMigrationPreservesCustomConfig(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	custom := `[
		{"platform":"openai","enabled":false,"models":["gpt-custom"]},
		{"platform":"kimi","enabled":false,"models":["moonshot-v1"],"note":"keep"},
		{"platform":"private-provider","enabled":true,"models":[]}
	]`
	originalUpdatedAt := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	_, err := tx.ExecContext(ctx, `
		UPDATE channel_monitor_v2_config
		SET platforms = $1::jsonb, version = 41, updated_at = $2
		WHERE id = 1`, custom, originalUpdatedAt)
	require.NoError(t, err)

	migrationSQL, err := dbmigrations.FS.ReadFile("232_channel_monitor_v2_add_cn_platforms.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var raw []byte
	var version int
	var firstUpdatedAt time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT platforms, version, updated_at
		FROM channel_monitor_v2_config
		WHERE id = 1`).Scan(&raw, &version, &firstUpdatedAt)
	require.NoError(t, err)
	require.Equal(t, 42, version)
	require.True(t, firstUpdatedAt.After(originalUpdatedAt))

	var platforms []map[string]any
	require.NoError(t, json.Unmarshal(raw, &platforms))
	require.Len(t, platforms, 5)
	require.Equal(t, []string{"openai", "kimi", "private-provider", "zhipu", "deepseek"}, platformNames(platforms))
	require.Equal(t, false, platforms[0]["enabled"])
	require.Equal(t, []any{"gpt-custom"}, platforms[0]["models"])
	require.Equal(t, false, platforms[1]["enabled"])
	require.Equal(t, []any{"moonshot-v1"}, platforms[1]["models"])
	require.Equal(t, "keep", platforms[1]["note"])
	for _, platform := range platforms[3:] {
		require.Equal(t, true, platform["enabled"])
		require.Equal(t, []any{}, platform["models"])
	}

	// Reapplying the migration must be a complete no-op, including metadata.
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)
	var secondRaw []byte
	var secondVersion int
	var secondUpdatedAt time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT platforms, version, updated_at
		FROM channel_monitor_v2_config
		WHERE id = 1`).Scan(&secondRaw, &secondVersion, &secondUpdatedAt)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(secondRaw))
	require.Equal(t, version, secondVersion)
	require.Equal(t, firstUpdatedAt, secondUpdatedAt)
}

func platformNames(platforms []map[string]any) []string {
	names := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		name, _ := platform["platform"].(string)
		names = append(names, name)
	}
	return names
}

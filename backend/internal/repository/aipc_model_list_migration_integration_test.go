//go:build integration

package repository

import (
	"context"
	"testing"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestAIPCModelListUpgradePreservesDisplayOnlyAndDenials(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	_, err := tx.ExecContext(ctx, `ALTER TABLE groups RENAME COLUMN model_allowlist TO models_list_config;
 DELETE FROM schema_migrations WHERE filename='235_group_model_allowlist.sql';`)
	require.NoError(t, err)
	var id int64
	require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO groups(name,platform,rate_multiplier,status,models_list_config)
 VALUES('aipc-upgrade-policy','openai',1,'active','{"enabled":true,"models":["gpt-6"],"blocked_models":["gpt-image-*"]}') RETURNING id`).Scan(&id))
	sql, err := dbmigrations.FS.ReadFile("234_zz_aipc_model_list_compat.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(sql))
	require.NoError(t, err)
	applyGroupModelAllowlistRepair(ctx, t, tx)
	var got string
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT model_allowlist::text FROM groups WHERE id=$1", id).Scan(&got))
	require.JSONEq(t, `{"enabled":true,"legacy_list_only":true,"models":["gpt-6"],"blocked_models":["gpt-image-*"]}`, got)
	// Replaying the compatibility migration after the rename is a no-op.
	_, err = tx.ExecContext(ctx, string(sql))
	require.NoError(t, err)
}

func TestAIPCModelListUpgradeDoesNotRelaxOfficialPolicy(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	_, err := tx.ExecContext(ctx, `ALTER TABLE groups ADD COLUMN models_list_config JSONB NOT NULL DEFAULT '{}';`)
	require.NoError(t, err)
	var id int64
	require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO groups(name,platform,rate_multiplier,status,model_allowlist,models_list_config)
 VALUES('official-policy','openai',1,'active','{"enabled":true,"models":["gpt-6"]}','{"enabled":true,"models":["legacy"]}') RETURNING id`).Scan(&id))
	sql, err := dbmigrations.FS.ReadFile("234_zz_aipc_model_list_compat.sql")
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(sql))
	require.NoError(t, err)
	applyGroupModelAllowlistRepair(ctx, t, tx)
	var got string
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT model_allowlist::text FROM groups WHERE id=$1", id).Scan(&got))
	require.JSONEq(t, `{"enabled":true,"models":["gpt-6"]}`, got)
}

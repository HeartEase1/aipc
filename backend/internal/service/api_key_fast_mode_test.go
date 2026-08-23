package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthSnapshotPreservesFastMode(t *testing.T) {
	svc := &APIKeyService{}
	key := &APIKey{
		ID: 4, UserID: 9, Name: "fast", Status: StatusActive, FastMode: true,
		User: &User{ID: 9, Status: StatusActive, Role: RoleUser},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), key)
	require.NotNil(t, snapshot)
	require.Equal(t, apiKeyAuthSnapshotVersion, snapshot.Version)
	require.True(t, snapshot.FastMode)

	restored := svc.snapshotToAPIKey("sk-fast", snapshot)
	require.True(t, restored.FastMode)
	require.Equal(t, "sk-fast", restored.Key)
}

func TestAPIKeyFastModeContextDefaultsOff(t *testing.T) {
	require.False(t, APIKeyFastModeEnabled(context.Background()))
	require.True(t, APIKeyFastModeEnabled(WithAPIKeyFastMode(context.Background(), true)))
}

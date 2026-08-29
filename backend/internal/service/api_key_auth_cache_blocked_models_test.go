package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthSnapshotPreservesBlockedModels(t *testing.T) {
	groupID := int64(17)
	svc := &APIKeyService{}
	apiKey := &APIKey{
		ID:      8,
		UserID:  9,
		GroupID: &groupID,
		Status:  StatusActive,
		User:    &User{ID: 9, Status: StatusActive, Role: RoleUser},
		Group: &Group{
			ID:               groupID,
			Name:             "restricted",
			Platform:         PlatformOpenAI,
			Status:           StatusActive,
			SubscriptionType: SubscriptionTypeStandard,
			ModelsListConfig: GroupModelsListConfig{BlockedModels: []string{"gpt-5.6-luna", "gpt-image-*"}},
		},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)
	payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: snapshot})
	require.NoError(t, err)
	var cached APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal(payload, &cached))

	restored, used, err := svc.applyAuthCacheEntry("sk-test", &cached)
	require.NoError(t, err)
	require.True(t, used)
	require.NotNil(t, restored.Group)
	require.Equal(t, []string{"gpt-5.6-luna", "gpt-image-*"}, restored.Group.ModelsListConfig.BlockedModels)
	require.True(t, restored.Group.IsModelBlocked("gpt-image-2"))
}

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
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

func TestAPIKeyFastModeUsesFinalUpstreamTierAndRespectsGlobalPolicy(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	ctx := WithAPIKeyFastMode(context.Background(), true)
	body := []byte(`{"model":"gpt-5.6-sol","service_tier":"flex"}`)

	allow := newOpenAIGatewayServiceWithSettings(t, DefaultOpenAIFastPolicySettings())
	forwarded, err := allow.applyOpenAIFastPolicyToBody(ctx, account, "gpt-5.6-sol", body)
	require.NoError(t, err)
	require.Equal(t, OpenAIFastTierPriority, gjson.GetBytes(forwarded, "service_tier").String())

	filter := newOpenAIGatewayServiceWithSettings(t, openAIFastFilterPriorityPolicy())
	forwarded, err = filter.applyOpenAIFastPolicyToBody(ctx, account, "gpt-5.6-sol", body)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(forwarded, "service_tier").Exists())

	forwarded, err = allow.applyOpenAIFastPolicyToBody(ctx,
		&Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}, "claude-sonnet-4-5", body)
	require.NoError(t, err)
	require.Equal(t, string(body), string(forwarded), "the key preference must not affect another platform")
}

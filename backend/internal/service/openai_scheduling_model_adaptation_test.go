//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAISchedulingChannelModelPreservesExplicitAliasWhitelist(t *testing.T) {
	ctx := WithOpenAIForwardModel(context.Background(), "gpt-5.6-sol", false, "public-alias")
	for _, mapping := range []map[string]any{
		{"public-alias": "gpt-5.6-terra"},
		{"gpt-5.6-sol": "gpt-5.6-sol"},
	} {
		account := &Account{
			Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
			Credentials: map[string]any{"model_mapping": mapping},
		}
		require.True(t, isOpenAICompatibleAccountEligibleForRequest(ctx, account, PlatformOpenAI, "gpt-5.6-sol", false, ""))
		scheduler := &defaultOpenAIAccountScheduler{}
		require.True(t, scheduler.isAccountRequestCompatible(ctx, account, OpenAIAccountScheduleRequest{RequestedModel: "gpt-5.6-sol"}))
	}
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"model_mapping": map[string]any{"another-alias": "gpt-5.6-sol"},
	}}
	require.False(t, openAIAccountSupportsSchedulingModel(ctx, account, "gpt-5.6-sol"))
	account.Credentials["model_mapping"] = map[string]any{"public-alias": "gpt-5.6-terra"}
	require.False(t, openAIAccountSupportsSchedulingModel(context.Background(), account, "gpt-5.6-sol"))
	require.False(t, openAIAccountSupportsSchedulingModel(ctx, account, "gpt-6-astra"))

	oauth := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	ctx = WithOpenAIForwardModel(context.Background(), "glm-5.3", false, "public-alias")
	require.False(t, openAIAccountSupportsSchedulingModel(ctx, oauth, "glm-5.3"), "an unrestricted alias must not bypass OAuth provider support")
}

func TestOpenAISchedulingMappedModelKeepsChannelBillingRestrictionSource(t *testing.T) {
	groupID := int64(10)
	for _, tc := range []struct{ source, allowed string }{
		{BillingModelSourceRequested, "public-alias"},
		{BillingModelSourceChannelMapped, "gpt-5.6-sol"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			channelSvc := newTestChannelService(makeStandardRepo(Channel{
				ID: 1, Status: StatusActive, GroupIDs: []int64{groupID}, RestrictModels: true,
				BillingModelSource: tc.source,
				ModelMapping: map[string]map[string]string{PlatformOpenAI: {
					"public-alias": "gpt-5.6-sol", "gpt-5.6-sol": "gpt-6-astra",
				}},
				ModelPricing: []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{tc.allowed}}},
			}, map[int64]string{groupID: PlatformOpenAI}))
			svc := &OpenAIGatewayService{channelService: channelSvc}
			ctx := WithOpenAIForwardModel(context.Background(), "gpt-5.6-sol", false, "public-alias")
			require.False(t, svc.checkChannelPricingRestriction(ctx, &groupID, "gpt-5.6-sol"))
			require.True(t, svc.checkChannelPricingRestriction(ctx, &groupID, "unlisted"))
		})
	}
}

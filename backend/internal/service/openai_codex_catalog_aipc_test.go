//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexCatalogAIPCChannelMappingAndCapabilityIntersection(t *testing.T) {
	group := &Group{ID: 10, Platform: PlatformOpenAI}
	channels := newTestChannelService(makeStandardRepo(Channel{
		ID: 1, Status: StatusActive, GroupIDs: []int64{group.ID},
		ModelMapping: map[string]map[string]string{PlatformOpenAI: {"public-alias": "routed-model"}},
	}, map[int64]string{group.ID: PlatformOpenAI}))
	reasoning := true
	newAccount := func(levels []string, window, output int64, search bool) Account {
		account := Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{"model_mapping": map[string]any{"public-alias": "unused", "routed-model": "actual-model"}},
		}
		capability, err := json.Marshal(search)
		require.NoError(t, err)
		account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
			"actual-model": {ID: "actual-model", Reasoning: &reasoning, SupportedReasoningLevels: levels,
				InputModalities: []string{"text"}, ContextWindow: window, MaxOutputTokens: output,
				CodexToolCapabilities: map[string]json.RawMessage{"supports_search_tool": capability}},
		}})
		return account
	}
	accounts := []Account{newAccount([]string{"low", "high", "max"}, 128000, 8192, true), newAccount([]string{"low", "high"}, 64000, 4096, false)}
	body, err := buildCodexCatalogForGroup(context.Background(), channels, group, PlatformOpenAI, []string{"public-alias"}, accounts, nil, true)
	require.NoError(t, err)
	models := decodeCodexManifestModels(t, body)
	require.Len(t, models, 1)
	require.Equal(t, "public-alias", models[0]["slug"])
	require.EqualValues(t, 64000, models[0]["context_window"])
	require.EqualValues(t, 4096, models[0]["max_output_tokens"])
	require.Equal(t, false, models[0]["supports_search_tool"])
	require.Len(t, models[0]["supported_reasoning_levels"], 2)
	require.Equal(t, "unused", accounts[0].GetMappedModel("public-alias"), "catalog construction must not mutate account mapping")
}

func TestCodexCatalogAIPCCompositeRequiresActualRouteForUnknownAlias(t *testing.T) {
	group := &Group{ID: 10, Platform: PlatformComposite}
	accounts := []Account{{Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"model_mapping": map[string]any{"public-alias": "custom-model"}},
		Extra:       map[string]any{"openai_responses_supported": false},
	}}
	body, err := buildCodexCatalogForGroup(context.Background(), nil, group, PlatformComposite, []string{"public-alias"}, accounts, nil, true)
	require.NoError(t, err)
	require.Empty(t, decodeCodexManifestModels(t, body))
	routes := []CompositeModelRoute{{PublicModel: "public-alias", TargetPlatform: PlatformOpenAI, Endpoint: CompositeRouteEndpointResponses}}
	body, err = buildCodexCatalogForGroup(context.Background(), nil, group, PlatformComposite, []string{"public-alias"}, accounts, routes, true)
	require.NoError(t, err)
	models := decodeCodexManifestModels(t, body)
	require.Len(t, models, 1)
	require.Equal(t, true, models[0]["supports_search_tool"])
	group.ModelsListConfig.BlockedModels = []string{"public-alias"}
	body, err = buildCodexCatalogForGroup(context.Background(), nil, group, PlatformComposite, []string{"public-alias"}, accounts, routes, true)
	require.NoError(t, err)
	require.Empty(t, decodeCodexManifestModels(t, body))
}

func TestCodexCatalogSnapshotDoesNotFollowChangedProvider(t *testing.T) {
	account := newCodexModelsAPIKeyTestAccount("https://first.example/v1")
	account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{BaseURL: "https://first.example/v1", Models: map[string]UpstreamModelMetadata{"same-name": {ID: "same-name"}}})
	_, exists := account.GetUpstreamModelMetadata("same-name")
	require.True(t, exists)
	account.Credentials["base_url"] = "https://second.example/v1"
	_, exists = account.GetUpstreamModelMetadata("same-name")
	require.False(t, exists)
}

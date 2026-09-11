package service

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAstraCodexToolCapabilitiesUseAccountScopeAndSharedDeclarations(t *testing.T) {
	newAccount := func(baseURL string) Account {
		return Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
			"base_url": baseURL, "model_mapping": map[string]any{"public-astra": "gpt-6-astra"},
		}}
	}
	official := newAccount("https://api.openai.com/v1")
	custom := newAccount("https://relay.example/v1")
	bridge := newAccount("https://bridge.example/v1")
	bridge.Extra = map[string]any{"openai_responses_supported": false}
	for _, tt := range []struct {
		name     string
		accounts []Account
		search   bool
	}{
		{"official fallback", []Account{official}, true},
		{"custom host has no guessed capability", []Account{custom}, false},
		{"missing peer capability", []Account{official, custom}, false},
		{"implemented chat bridge", []Account{bridge}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{"public-astra"}, tt.accounts, nil, nil, true)
			require.NoError(t, err)
			model := decodeCodexManifestModels(t, body)[0]
			require.Equal(t, "public-astra", model["slug"])
			require.Equal(t, tt.search, model["supports_search_tool"])
			require.Equal(t, false, model["use_responses_lite"])
			if tt.name == "official fallback" {
				require.Equal(t, "freeform", model["apply_patch_tool_type"])
				require.Equal(t, "3000", model["comp_hash"])
			}
		})
	}
	custom.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
		"gpt-6-astra": {CodexToolCapabilities: map[string]json.RawMessage{
			"supports_search_tool": json.RawMessage("true"), "apply_patch_tool_type": json.RawMessage("null"),
			"comp_hash": json.RawMessage(`"3000"`), "tool_mode": json.RawMessage("null"), "use_responses_lite": json.RawMessage("false"),
		}},
	}})
	body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{"public-astra"}, []Account{official, custom}, nil, nil, true)
	require.NoError(t, err)
	model := decodeCodexManifestModels(t, body)[0]
	require.Equal(t, true, model["supports_search_tool"])
	require.Nil(t, model["apply_patch_tool_type"], "conflicting tool protocols must not be advertised")
	require.Equal(t, "3000", model["comp_hash"])
}

func TestAstraCodexToolCapabilitiesPreserveLiveNullAndFalse(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.openai.com/v1"}}
	account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
		"gpt-6-astra": {CodexToolCapabilities: map[string]json.RawMessage{
			"supports_search_tool": json.RawMessage("true"), "apply_patch_tool_type": json.RawMessage(`"freeform"`),
			"comp_hash": json.RawMessage(`"3000"`), "tool_mode": json.RawMessage(`"code_mode_only"`),
		}},
	}})
	for _, source := range []string{
		`{"models":[{"slug":"gpt-6-astra","supports_search_tool":false,"apply_patch_tool_type":null,"tool_mode":null}]}`,
		`{"data":[{"id":"gpt-6-astra","supports_search_tool":false,"apply_patch_tool_type":null,"tool_mode":null}]}`,
	} {
		converted := convertOpenAIModelListToCodexManifest([]byte(source), account)
		body, err := applySyncedAPIKeyCodexModelMetadata(converted, account, source[2:6] == "data")
		require.NoError(t, err)
		body, err = completeAPIKeyCodexModelsManifestMetadata(body, true, account)
		require.NoError(t, err)
		model := decodeCodexManifestModels(t, body)[0]
		require.Equal(t, false, model["supports_search_tool"])
		require.Nil(t, model["apply_patch_tool_type"])
		require.Nil(t, model["tool_mode"])
		require.Equal(t, "3000", model["comp_hash"])
	}
	fields := make(map[string]json.RawMessage)
	applyCodexToolCapabilities(fields, map[string]json.RawMessage{
		"supports_search_tool": json.RawMessage(`"true"`), "comp_hash": json.RawMessage(`{"unexpected":true}`),
		"model_messages": json.RawMessage(`"do not persist prompts"`),
	}, true)
	require.Empty(t, fields)
}

// Scenario: mixed groups prefer capability metadata synced for the routed account.
func TestBuildCodexModelsManifestForGroupAdvertisesSearchOnlyForChatBridgeRoutes(t *testing.T) {
	t.Parallel()

	newAccount := func(id int64, nativeResponses bool) Account {
		return Account{
			ID: id, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{
				"base_url":      "https://provider.example/v1",
				"model_mapping": map[string]any{"company-coding-model": "company-coding-model"},
			},
			Extra: map[string]any{"openai_responses_supported": nativeResponses},
		}
	}

	for _, tc := range []struct {
		name     string
		accounts []Account
		want     bool
	}{
		{name: "chat bridge", accounts: []Account{newAccount(80, false)}, want: true},
		{name: "native responses", accounts: []Account{newAccount(81, true)}, want: false},
		{name: "mixed routes", accounts: []Account{newAccount(82, false), newAccount(83, true)}, want: false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := buildCodexModelsManifestForAccounts(
				PlatformOpenAI, []string{"company-coding-model"}, tc.accounts, nil, nil, true,
			)
			require.NoError(t, err)
			models := decodeCodexManifestModels(t, body)
			require.Len(t, models, 1)
			require.Equal(t, tc.want, models[0]["supports_search_tool"])
		})
	}
}

func TestCompleteAPIKeyCodexManifestSearchCapabilityPreservesUpstreamAndFailsClosed(t *testing.T) {
	t.Parallel()

	nativeAccount := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://provider.example/v1",
		},
		Extra: map[string]any{"openai_responses_supported": true},
	}
	body, err := completeAPIKeyCodexModelsManifestMetadata([]byte(`{"models":[
		{"slug":"explicit","supports_search_tool":true},
		{"slug":"missing"}
	]}`), true, nativeAccount)
	require.NoError(t, err)
	models := decodeCodexManifestModels(t, body)
	require.Equal(t, true, models[0]["supports_search_tool"])
	require.Equal(t, false, models[1]["supports_search_tool"])

	chatAccount := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://provider.example/v1",
		},
		Extra: map[string]any{"openai_responses_supported": false},
	}
	body, err = completeAPIKeyCodexModelsManifestMetadata(
		[]byte(`{"models":[
			{"slug":"explicit-disabled","supports_search_tool":false},
			{"slug":"generated"}
		]}`), true, chatAccount,
	)
	require.NoError(t, err)
	models = decodeCodexManifestModels(t, body)
	require.Equal(t, false, models[0]["supports_search_tool"])
	require.Equal(t, true, models[1]["supports_search_tool"])
}

// Scenario: multiple schedulable accounts advertise only their shared capabilities.
func TestAstraCodexToolCapabilitiesFollowAPIKeyAlias(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"base_url": "https://relay.example/v1", "model_mapping": map[string]any{"my-astra": "gpt-6-astra"},
	}}
	account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
		"gpt-6-astra": {CodexToolCapabilities: map[string]json.RawMessage{
			"supports_search_tool": json.RawMessage("true"), "apply_patch_tool_type": json.RawMessage(`"freeform"`),
			"use_responses_lite": json.RawMessage("false"),
		}},
	}})
	for _, source := range []string{`{"data":[{"id":"my-astra"}]}`, `{"models":[{"slug":"my-astra"}]}`} {
		converted := convertOpenAIModelListToCodexManifest([]byte(source), account)
		body, err := completeAPIKeyCodexModelsManifestMetadata(converted, true, account)
		require.NoError(t, err)
		models := decodeCodexManifestModels(t, body)
		require.Len(t, models, 1)
		model := models[0]
		require.Equal(t, "my-astra", model["slug"])
		require.Equal(t, "my-astra", model["display_name"])
		require.Equal(t, true, model["supports_search_tool"])
		require.Equal(t, "freeform", model["apply_patch_tool_type"])
		require.Equal(t, false, model["use_responses_lite"])
	}
}

func TestAstraCodexToolCapabilitiesKeepAPIKeyResponsesLiteGuard(t *testing.T) {
	account := Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"base_url": "https://relay.example/v1", "model_mapping": map[string]any{"my-astra": "gpt-6-astra"},
	}}
	account.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
		"gpt-6-astra": {CodexToolCapabilities: map[string]json.RawMessage{"use_responses_lite": json.RawMessage("true")}},
	}})
	body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{"my-astra"}, []Account{account}, nil, nil, true)
	require.NoError(t, err)
	require.Equal(t, false, decodeCodexManifestModels(t, body)[0]["use_responses_lite"])
	body, err = adjustAPIKeyCodexModelsManifest([]byte(`{"models":[{"slug":"my-astra","use_responses_lite":true}]}`), &account)
	require.NoError(t, err)
	require.Equal(t, false, decodeCodexManifestModels(t, body)[0]["use_responses_lite"])
}

// Scenario: mixed groups prefer capability metadata synced for the routed account.

func decodeCodexManifestModels(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var result struct {
		Models []map[string]any `json:"models"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	return result.Models
}

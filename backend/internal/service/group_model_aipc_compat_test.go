package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAIPCModelPolicyCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name            string
		policy          GroupModelAllowlist
		model           string
		allowed, listed bool
	}{
		{"official admits selected", GroupModelAllowlist{Enabled: true, Models: []string{"gpt-*"}}, "gpt-6", true, true},
		{"official rejects unselected", GroupModelAllowlist{Enabled: true, Models: []string{"gpt-*"}}, "claude-opus", false, false},
		{"legacy selection only affects listing", GroupModelAllowlist{Enabled: true, LegacyListOnly: true, Models: []string{"gpt-*"}}, "claude-opus", true, false},
		{"legacy empty selection does not restrict", GroupModelAllowlist{Enabled: true, LegacyListOnly: true}, "gpt-6", true, true},
		{"deny without allowlist", GroupModelAllowlist{BlockedModels: []string{"gpt-*"}}, "gpt-6", false, false},
		{"deny leaves unrelated models available", GroupModelAllowlist{BlockedModels: []string{"gpt-*"}}, "claude-opus", true, true},
		{"deny wins over allow", GroupModelAllowlist{Enabled: true, Models: []string{"gpt-*"}, BlockedModels: []string{"gpt-6"}}, "gpt-6", false, false},
		{"deny wins over legacy selection", GroupModelAllowlist{Enabled: true, LegacyListOnly: true, Models: []string{"gpt-*"}, BlockedModels: []string{"gpt-6"}}, "gpt-6", false, false},
		{"gemini resource prefix", GroupModelAllowlist{BlockedModels: []string{"gemini-*"}}, "models/gemini-pro", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.allowed, tc.policy.Allows(tc.model))
			require.Equal(t, tc.listed, tc.policy.AllowsForListing(tc.model))
			if tc.listed {
				require.Contains(t, tc.policy.FilterForListing([]string{tc.model}), tc.model)
			} else {
				require.NotContains(t, tc.policy.FilterForListing([]string{tc.model}), tc.model)
			}
		})
	}
}

func TestAIPCModelPolicySurvivesPersistenceAndNormalization(t *testing.T) {
	policy := GroupModelAllowlist{Enabled: true, LegacyListOnly: true, Models: []string{"gpt-6"}, BlockedModels: []string{"gemini-*"}}
	require.Equal(t, policy, GroupModelAllowlistFromDomain(DomainGroupModelAllowlist(policy)))
	normalized, err := normalizeGroupModelAllowlist(policy)
	require.NoError(t, err)
	require.Equal(t, policy, normalized)
	require.True(t, (&Group{ModelAllowlist: GroupModelAllowlist{BlockedModels: []string{"gpt-6"}}}).ModelAllowlistEnabled())
	snapshot := APIKeyAuthGroupSnapshot{ModelAllowlist: policy}
	encoded, err := json.Marshal(snapshot)
	require.NoError(t, err)
	var restored APIKeyAuthGroupSnapshot
	require.NoError(t, json.Unmarshal(encoded, &restored))
	require.Equal(t, policy, restored.ModelAllowlist)
	svc := &APIKeyService{}
	_, ok, err := svc.applyAuthCacheEntry("stale-aipc-policy", &APIKeyAuthCacheEntry{Snapshot: &APIKeyAuthSnapshot{Version: 24}})
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCodexManifestDenialsWithoutSelection(t *testing.T) {
	body := []byte(`{"models":[{"slug":"gpt-6","display_name":"Keep metadata"},{"slug":"blocked-model"}]}`)
	filtered, changed, err := mergeConfiguredCodexModelsManifest(body, []string{"blocked-alias"}, nil, false, []string{"blocked-*"})
	require.NoError(t, err)
	require.True(t, changed)
	require.JSONEq(t, `{"models":[{"slug":"gpt-6","display_name":"Keep metadata"}]}`, string(filtered))
}

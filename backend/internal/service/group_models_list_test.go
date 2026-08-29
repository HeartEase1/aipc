package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupModelsListConfigNormalizesBothLists(t *testing.T) {
	got := normalizeGroupModelsListConfig(GroupModelsListConfig{
		Enabled:       true,
		Models:        []string{" gpt-5.6-sol ", "gpt-5.6-sol", ""},
		BlockedModels: []string{" gpt-5.6-luna ", "gpt-5.6-luna", "gpt-5.6-*"},
	})

	require.Equal(t, GroupModelsListConfig{
		Enabled:       true,
		Models:        []string{"gpt-5.6-sol"},
		BlockedModels: []string{"gpt-5.6-luna", "gpt-5.6-*"},
	}, got)
}

func TestGroupIsModelBlocked(t *testing.T) {
	group := &Group{ModelsListConfig: GroupModelsListConfig{
		BlockedModels: []string{"gpt-5.6-luna", "claude-opus-*", "models/gemini-3-pro-preview"},
	}}

	require.True(t, group.IsModelBlocked("gpt-5.6-luna"))
	require.True(t, group.IsModelBlocked("claude-opus-4-6"))
	require.True(t, group.IsModelBlocked("gemini-3-pro-preview"))
	require.True(t, group.IsModelBlocked("models/gemini-3-pro-preview"))
	require.False(t, group.IsModelBlocked("gpt-5.6-sol"))
	require.False(t, group.IsModelBlocked("Claude-Opus-4-6"), "model IDs remain case-sensitive")
}

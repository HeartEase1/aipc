package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPelicanPayloadsDoNotChangeDefaultAccountTestPayloads(t *testing.T) {
	defaultClaude, err := createTestPayload("claude-sonnet-4-6")
	require.NoError(t, err)
	defaultMessages, ok := defaultClaude["messages"].([]map[string]any)
	require.True(t, ok)
	defaultContent, ok := defaultMessages[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "hi", defaultContent[0]["text"])

	defaultOpenAI := createOpenAITestPayload("gpt-6-astra", true)
	defaultInput, ok := defaultOpenAI["input"].([]map[string]any)
	require.True(t, ok)
	defaultOpenAIContent, ok := defaultInput[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "hi", defaultOpenAIContent[0]["text"])
	require.NotContains(t, defaultOpenAI, "reasoning")

	pelicanClaude, err := createPelicanClaudePayload("claude-sonnet-4-6", "draw the pelican animation")
	require.NoError(t, err)
	pelicanMessages, ok := pelicanClaude["messages"].([]map[string]any)
	require.True(t, ok)
	pelicanContent, ok := pelicanMessages[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "draw the pelican animation", pelicanContent[0]["text"])

	pelicanOpenAI := createPelicanOpenAIPayload("gpt-6-astra", true, "draw the pelican animation", "medium")
	pelicanInput, ok := pelicanOpenAI["input"].([]map[string]any)
	require.True(t, ok)
	pelicanOpenAIContent, ok := pelicanInput[0]["content"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "draw the pelican animation", pelicanOpenAIContent[0]["text"])
	require.Equal(t, map[string]any{"effort": "medium"}, pelicanOpenAI["reasoning"])
}

func TestPelicanCapabilitiesRespectMappingsAndKnownModels(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"model_mapping": map[string]any{"alias": "gpt-6-astra", "picture": "gpt-image-2"}}}
	options, err := ResolvePelicanTestOptions(account, "alias")
	require.NoError(t, err)
	require.Contains(t, options.SupportedReasoningLevels, "max")
	require.NotContains(t, options.SupportedReasoningLevels, "ultra")
	_, err = ResolvePelicanTestOptions(account, "not-allowed")
	require.Error(t, err)
	_, err = ResolvePelicanTestOptions(account, "picture")
	require.Error(t, err)
	account.Credentials = nil
	options, err = ResolvePelicanTestOptions(account, "unknown-model")
	require.NoError(t, err)
	require.Empty(t, options.SupportedReasoningLevels)
}

func TestPelicanClaudeAndGeminiPayloadsCarryReasoningAndOutputBudget(t *testing.T) {
	claudePayload, err := createPelicanClaudePayload("claude-opus-4-6", "draw HTML", "max")
	require.NoError(t, err)
	require.Equal(t, map[string]any{"effort": "max"}, claudePayload["output_config"])
	require.Equal(t, map[string]any{"type": "adaptive"}, claudePayload["thinking"])
	require.Equal(t, pelicanMaxOutputTokens, claudePayload["max_tokens"])
	require.NotContains(t, claudePayload, "temperature")
	for _, model := range []string{"gemini-2.5-pro", "gemini-3-flash-preview"} {
		var payload map[string]any
		require.NoError(t, json.Unmarshal(createPelicanGeminiPayload(model, "draw HTML", "high"), &payload))
		config, ok := payload["generationConfig"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, float64(pelicanMaxOutputTokens), config["maxOutputTokens"])
		if model == "gemini-2.5-pro" {
			require.Equal(t, map[string]any{"thinkingBudget": float64(8192)}, config["thinkingConfig"])
		} else {
			require.Equal(t, map[string]any{"thinkingLevel": "high"}, config["thinkingConfig"])
		}
	}
}

func TestPelicanConcurrencyLimitAndRelease(t *testing.T) {
	svc := &AccountTestService{}
	for id := int64(1); id <= 4; id++ {
		for i := 0; i < 8; i++ {
			require.True(t, svc.acquirePelicanTest(id))
		}
		require.False(t, svc.acquirePelicanTest(id))
	}
	require.False(t, svc.acquirePelicanTest(5))
	svc.releasePelicanTest(1)
	require.True(t, svc.acquirePelicanTest(5))
	for i := 0; i < 7; i++ {
		svc.releasePelicanTest(1)
	}
	require.NotContains(t, svc.pelicanActive, int64(1))
}

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCodexListConversionPreservesExplicitCapabilities(t *testing.T) {
	body := []byte(`{"data":[{"id":"gpt-6-astra","supports_search_tool":false,"future_tool":{"enabled":false},"context_window":1050000,"service_tiers":[]}]}`)
	converted := convertOpenAIModelListToCodexManifest(body)
	require.Equal(t, "gpt-6-astra", gjson.GetBytes(converted, "models.0.slug").String())
	require.Equal(t, "false", gjson.GetBytes(converted, "models.0.supports_search_tool").Raw)
	require.Equal(t, "false", gjson.GetBytes(converted, "models.0.future_tool.enabled").Raw)
	require.Equal(t, int64(1050000), gjson.GetBytes(converted, "models.0.context_window").Int())
	require.Equal(t, "[]", gjson.GetBytes(converted, "models.0.service_tiers").Raw)
}

func TestMappedAstraDisablesResponsesLiteWithoutChangingCapabilities(t *testing.T) {
	account := &Account{Credentials: map[string]any{"model_mapping": map[string]any{"custom-astra": "gpt-6-astra"}}}
	body, err := adjustAPIKeyCodexModelsManifest([]byte(`{"models":[{"slug":"custom-astra","use_responses_lite":true,"supports_search_tool":false}]}`), account)
	require.NoError(t, err)
	require.Equal(t, "false", gjson.GetBytes(body, "models.0.use_responses_lite").Raw)
	require.Equal(t, "false", gjson.GetBytes(body, "models.0.supports_search_tool").Raw)
}

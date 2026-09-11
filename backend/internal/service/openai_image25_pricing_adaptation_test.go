package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestImage25MissingCatalogDoesNotUseOlderImageRates(t *testing.T) {
	old := &LiteLLMModelPricing{InputCostPerToken: 5e-6, OutputCostPerImageToken: 3e-5}
	svc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{"gpt-image-2": old}}
	for _, model := range []string{"gpt-image-2.5-flare", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare-2026-09-08", "gpt-image-2.5-unknown"} {
		require.Nil(t, svc.GetModelPricing(model), model)
	}
	require.Same(t, old, svc.GetModelPricing("gpt-image-2"))
	configured := &LiteLLMModelPricing{InputCostPerToken: 7e-6, OutputCostPerImageToken: 4e-5}
	svc.pricingData["gpt-image-2.5-flare"] = configured
	require.Same(t, configured, svc.GetModelPricing("gpt-image-2.5-flare"))
}

func TestImageToolUsagePreservesImageInputTokens(t *testing.T) {
	for _, tt := range []struct {
		name    string
		details string
		want    int
	}{
		{"image input", `,"input_tokens_details":{"image_tokens":60}`, 60},
		{"bounded by total", `,"input_tokens_details":{"image_tokens":150}`, 100},
		{"missing", "", 0},
		{"negative", `,"input_tokens_details":{"image_tokens":-1}`, 0},
		{"overflow", `,"input_tokens_details":{"image_tokens":1e999}`, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usage, ok := openAIImagesToolUsageFromGJSON(gjson.Parse(`{"input_tokens":100,"output_tokens":80,"output_tokens_details":{"image_tokens":80}` + tt.details + `}`))
			require.True(t, ok)
			require.Equal(t, 100, usage.InputTokens)
			require.Equal(t, tt.want, usage.ImageInputTokens)
			require.Equal(t, 80, usage.ImageOutputTokens)
		})
	}
}

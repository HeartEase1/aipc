package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestImage25CatalogAndFallbackUseOfficialRates(t *testing.T) {
	old := &LiteLLMModelPricing{InputCostPerToken: 9e-6, OutputCostPerImageToken: 9e-5}
	svc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{"gpt-image-2": old}}

	for _, model := range []string{"gpt-image-2.5-flare", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare-2026-09-08", "gpt-image-2.5-sunburst-2026-09-08"} {
		got := svc.GetModelPricing(model)
		require.Same(t, openAIGPTImage25FallbackPricing, got, model)
		require.InDelta(t, 5e-6, got.InputCostPerToken, 1e-12)
		require.InDelta(t, 1.25e-6, got.CacheReadInputTokenCost, 1e-12)
		require.InDelta(t, 8e-6, got.InputCostPerImageToken, 1e-12)
		require.InDelta(t, 3e-5, got.OutputCostPerImageToken, 1e-12)
	}
	require.Nil(t, svc.GetModelPricing("gpt-image-2.5-unknown"))
	require.Same(t, old, svc.GetModelPricing("gpt-image-2"))

	configured := &LiteLLMModelPricing{InputCostPerToken: 7e-6, OutputCostPerImageToken: 4e-5}
	svc.pricingData["gpt-image-2.5-flare"] = configured
	require.Same(t, configured, svc.GetModelPricing("gpt-image-2.5-flare"))
}

func TestBundledPricingIncludesImage25Entries(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "resources", "model-pricing", "model_prices_and_context_window.json"))
	require.NoError(t, err)
	pricingSvc := &PricingService{}
	parsed, err := pricingSvc.parsePricingData(data)
	require.NoError(t, err)
	for _, model := range []string{"gpt-image-2.5-flare", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare-2026-09-08", "gpt-image-2.5-sunburst-2026-09-08"} {
		got := parsed[model]
		require.NotNil(t, got, model)
		require.InDelta(t, 5e-6, got.InputCostPerToken, 1e-12)
		require.InDelta(t, 8e-6, got.InputCostPerImageToken, 1e-12)
		require.InDelta(t, 3e-5, got.OutputCostPerImageToken, 1e-12)
	}
}

func TestImage25TokenBillingDoesNotUsePerImageDefault(t *testing.T) {
	billing := NewBillingService(nil, &PricingService{})
	tokens := UsageTokens{InputTokens: 1000, ImageInputTokens: 200, OutputTokens: 300, ImageOutputTokens: 300}
	cost, err := billing.CalculateCost("gpt-image-2.5-flare", tokens, 1)
	require.NoError(t, err)
	require.InDelta(t, 0.004, cost.InputCost, 1e-12)
	require.InDelta(t, 0.0016, cost.ImageInputCost, 1e-12)
	require.InDelta(t, 0.009, cost.ImageOutputCost, 1e-12)
	require.NotEqual(t, defaultImageGenerationPrice, cost.TotalCost)
}

func TestImage25UsesGroupPerImagePriceWhenConfigured(t *testing.T) {
	price := 0.2
	svc := &OpenAIGatewayService{billingService: NewBillingService(nil, &PricingService{})}
	cost, err := svc.calculateOpenAIRecordUsageCost(
		context.Background(),
		&OpenAIForwardResult{ImageCount: 2, ImageSize: "1K"},
		&APIKey{Group: &Group{ImagePrice1K: &price}},
		[]string{"gpt-image-2.5-flare"},
		1, 1, 1, 1,
		UsageTokens{},
		"",
		nil,
		time.Time{},
	)
	require.NoError(t, err)
	require.Equal(t, string(BillingModeImage), cost.BillingMode)
	require.InDelta(t, 0.4, cost.TotalCost, 1e-12)
}

func TestImage25UsesIndependentImageMultiplier(t *testing.T) {
	price := 0.1
	svc := &OpenAIGatewayService{billingService: NewBillingService(nil, &PricingService{})}
	cost, err := svc.calculateOpenAIRecordUsageCost(
		context.Background(),
		&OpenAIForwardResult{ImageCount: 1, ImageSize: "1K"},
		&APIKey{Group: &Group{ImagePrice1K: &price, ImageRateIndependent: true, ImageRateMultiplier: 2}},
		[]string{"gpt-image-2.5-flare"},
		1, 2, 1, 1,
		UsageTokens{},
		"",
		nil,
		time.Time{},
	)
	require.NoError(t, err)
	require.Equal(t, string(BillingModeImage), cost.BillingMode)
	require.InDelta(t, 0.2, cost.ActualCost, 1e-12)
}

func TestImage25MissingTokenUsageDoesNotBillZero(t *testing.T) {
	svc := &OpenAIGatewayService{billingService: NewBillingService(nil, &PricingService{})}
	_, err := svc.calculateOpenAIRecordUsageCost(
		context.Background(),
		&OpenAIForwardResult{ImageCount: 1},
		&APIKey{},
		[]string{"gpt-image-2.5-flare"},
		1, 1, 1, 1,
		UsageTokens{},
		"",
		nil,
		time.Time{},
	)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrModelPricingUnavailable)
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

//go:build unit

package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAnthropicReasoningConvertedLevelsNeverExceedPolicy(t *testing.T) {
	for _, tc := range []struct{ model, source, want string }{
		{"claude-fable-5-1", "xhigh", "xhigh"},
		{"claude-fable-5-1", "max", "max"},
		{"claude-opus-4-6", "xhigh", "high"},
		{"claude-opus-4-5", "max", "high"},
		{"claude-fable-5-1", "minimal", "low"},
	} {
		request := &apicompat.AnthropicRequest{Model: tc.model, OutputConfig: &apicompat.AnthropicOutputConfig{Effort: "max"}, Thinking: &apicompat.AnthropicThinking{Type: "enabled", BudgetTokens: 32768}}
		normalizeConvertedAnthropicEffort(request, &tc.source)
		require.Equal(t, tc.want, request.OutputConfig.Effort)
		if tc.want == "low" {
			require.Nil(t, request.Thinking)
		}
	}
}

func newAnthropicReasoningBillingService(prices map[string]*ModelPricing) *BillingService {
	catalog := make(map[string]*LiteLLMModelPricing, len(prices))
	for name, price := range prices {
		catalog[name] = &LiteLLMModelPricing{
			InputCostPerToken: price.InputPricePerToken, OutputCostPerToken: price.OutputPricePerToken,
			LongContextInputTokenThreshold:  price.LongContextInputThreshold,
			LongContextInputCostMultiplier:  price.LongContextInputMultiplier,
			LongContextOutputCostMultiplier: price.LongContextOutputMultiplier,
		}
	}
	return &BillingService{pricingService: &PricingService{pricingData: catalog}}
}

func TestAnthropicReasoningPolicyPreservesProfitAdmission(t *testing.T) {
	group := profitControlTestGroup(1, 0.2, 0)
	group.Platform = PlatformAnthropic
	ctx, _ := WithGatewayTokenRequestPricing(profitControlTestCtx(group))
	gateway := &GatewayService{}
	ctx = gateway.withGatewayProfitControlGate(ctx, &group.ID)
	_, changed := ApplyOpenAIReasoningEffortPolicy([]byte(`{"output_config":{"effort":"max"}}`), "high", nil)
	require.True(t, changed)
	veto, _ := OpenAIProfitControlVeto(ctx, &Account{Platform: PlatformAnthropic, RateMultiplier: ptrFloat64(0.9)})
	require.True(t, veto)
	veto, _ = OpenAIProfitControlVeto(ctx, &Account{Platform: PlatformAnthropic, RateMultiplier: ptrFloat64(0.7)})
	require.False(t, veto)
}

func TestAnthropicReasoningMultiplierDefaultsAndValidation(t *testing.T) {
	for _, tc := range []struct {
		model, effort string
		override      *float64
		want          float64
	}{
		{"claude-fable-5-1", "max", nil, 3},
		{"anthropic/claude-fable-5.1-20260901", "max", nil, 3},
		{"claude-fable-5-10", "max", nil, 1},
		{"claude-fable-5.10", "max", nil, 1},
		{"claude-fable-5-1", "high", nil, 1},
		{"claude-fable-5-1", "", nil, 1},
		{"claude-fable-5-1", "max", ptrFloat64(1), 1},
		{"public-alias", "max", ptrFloat64(2), 2},
	} {
		require.Equal(t, tc.want, maxReasoningEffortBillingMultiplier(tc.model, tc.effort, &ModelPricing{MaxReasoningEffortMultiplier: tc.override}), tc.model)
	}
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		require.Error(t, checkPricesNotNegative(ChannelModelPricing{MaxReasoningEffortMultiplier: &value}))
	}
	require.NoError(t, checkPricesNotNegative(ChannelModelPricing{MaxReasoningEffortMultiplier: ptrFloat64(1)}))
}

func TestAnthropicReasoningPricingAliasAndPrecedence(t *testing.T) {
	bs := newAnthropicReasoningBillingService(map[string]*ModelPricing{"public-alias": {InputPricePerToken: 1}})
	cs := newTestChannelServiceForStats(t, &Channel{ID: 1, Status: StatusActive, ModelPricing: []ChannelModelPricing{{
		Platform: PlatformAnthropic, Models: []string{"public-alias"}, InputPrice: ptrFloat64(1), MaxReasoningEffortMultiplier: ptrFloat64(2),
	}}}, 1, PlatformAnthropic)
	cache := cs.cache.Load().(*channelCache)
	expandPricingToCache(cache, cache.channelByGroupID[1], 1, PlatformAnthropic)
	resolver := NewModelPricingResolver(cs, bs)
	group := &Group{ID: 1, Platform: PlatformAnthropic}
	input := CostInput{Ctx: context.Background(), Model: "public-alias", ReasoningModel: "claude-fable-5-1", ReasoningEffort: "max", Tokens: UsageTokens{InputTokens: 10}, RateMultiplier: 0.5, GroupID: &group.ID, Group: group, Resolver: resolver}
	cost, err := bs.CalculateCostUnified(input)
	require.NoError(t, err)
	require.Equal(t, 20.0, cost.TotalCost)
	require.Equal(t, 10.0, cost.ActualCost)
	group.ModelPricing = []ChannelModelPricing{{Models: []string{"public-alias"}, InputPrice: ptrFloat64(1), MaxReasoningEffortMultiplier: ptrFloat64(1)}}
	cost, err = bs.CalculateCostUnified(input)
	require.NoError(t, err)
	require.Equal(t, 10.0, cost.TotalCost)
	input.Resolver = nil
	cost, err = bs.CalculateCostUnified(input)
	require.NoError(t, err)
	require.Equal(t, 30.0, cost.TotalCost)
	require.Equal(t, 15.0, cost.ActualCost)
}

func TestAnthropicReasoningPricingStacksOnce(t *testing.T) {
	bs := newAnthropicReasoningBillingService(map[string]*ModelPricing{"claude-fable-5-1": {
		InputPricePerToken: 1, OutputPricePerToken: 2,
		LongContextInputThreshold: 10, LongContextInputMultiplier: 2, LongContextOutputMultiplier: 1.5,
	}})
	resolver := NewModelPricingResolver(nil, bs)
	input := CostInput{Model: "claude-fable-5-1", ReasoningEffort: "max", Tokens: UsageTokens{InputTokens: 20, OutputTokens: 10}, RateMultiplier: 0.5, Resolver: resolver, PricingAt: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)}
	input.Resolved = resolver.Resolve(context.Background(), PricingInput{Model: input.Model})
	input.Resolved.longContextPricingEnabled = true
	input.Resolved.Source = PricingSourceChannel
	input.Resolved.channelPricing = &ChannelModelPricing{TimePricing: &ChannelTimePricing{Timezone: "UTC", Periods: []ChannelTimePricingPeriod{{StartTime: "00:00", EndTime: "23:59", Multiplier: 2}}}}
	cost, err := bs.CalculateCostUnified(input)
	require.NoError(t, err)
	// (20*2 + 10*2*1.5) * time(2) * max(3), then user discount(0.5).
	require.Equal(t, 420.0, cost.TotalCost)
	require.Equal(t, 210.0, cost.ActualCost)
	body, changed := ApplyOpenAIReasoningEffortPolicy([]byte(`{"output_config":{"effort":"max"}}`), "high", nil)
	require.True(t, changed)
	input.ReasoningEffort = gjson.GetBytes(body, "output_config.effort").String()
	cost, err = bs.CalculateCostUnified(input)
	require.NoError(t, err)
	require.Equal(t, 140.0, cost.TotalCost)
	input.Resolved = &ResolvedPricing{Mode: BillingModePerRequest, DefaultPerRequestPrice: 7}
	input.RequestCount = 2
	input.ReasoningEffort = "max"
	cost, err = bs.CalculateCostUnified(input)
	require.NoError(t, err)
	require.Equal(t, 14.0, cost.TotalCost)
}

func TestAnthropicReasoningAccountStatsIndependent(t *testing.T) {
	model := "claude-fable-5-1"
	bs := newAnthropicReasoningBillingService(map[string]*ModelPricing{model: {InputPricePerToken: 1}})
	for _, tc := range []struct {
		name        string
		rules       []AccountStatsPricingRule
		useCustomer bool
		want        float64
	}{
		{name: "catalog cost", want: 30},
		{name: "customer total already includes max", useCustomer: true, want: 60},
		{name: "independent cost multiplier", rules: []AccountStatsPricingRule{{AccountIDs: []int64{1}, Pricing: []ChannelModelPricing{{Models: []string{model}, InputPrice: ptrFloat64(1), MaxReasoningEffortMultiplier: ptrFloat64(2)}}}}, useCustomer: true, want: 20},
		{name: "per request cost unchanged", rules: []AccountStatsPricingRule{{AccountIDs: []int64{1}, Pricing: []ChannelModelPricing{{Models: []string{model}, BillingMode: BillingModePerRequest, PerRequestPrice: ptrFloat64(5), MaxReasoningEffortMultiplier: ptrFloat64(2)}}}}, want: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cs := newTestChannelServiceForStats(t, &Channel{ID: 1, Status: StatusActive, ApplyPricingToAccountStats: tc.useCustomer, AccountStatsPricingRules: tc.rules}, 1, PlatformAnthropic)
			cost := resolveAccountStatsCost(context.Background(), cs, bs, 1, 1, model, UsageTokens{InputTokens: 10}, 1, 60, "", "max")
			require.NotNil(t, cost)
			require.Equal(t, tc.want, *cost)
		})
	}
}

func TestAnthropicReasoningCompositeAccountCostRule(t *testing.T) {
	model := "claude-fable-5-1"
	channel := &Channel{ID: 1, Status: StatusActive, AccountStatsPricingRules: []AccountStatsPricingRule{{
		GroupIDs: []int64{1}, Pricing: []ChannelModelPricing{{Platform: PlatformAnthropic, Models: []string{model}, InputPrice: ptrFloat64(1), MaxReasoningEffortMultiplier: ptrFloat64(2)}},
	}}}
	cs := newTestChannelServiceForStats(t, channel, 1, PlatformComposite)
	cost := resolveAccountStatsCost(context.Background(), cs, nil, 1, 1, model, UsageTokens{InputTokens: 10}, 1, 30, "", "max")
	require.NotNil(t, cost)
	require.Equal(t, 20.0, *cost)
}

func TestAnthropicReasoningGatewayFallbackAndUnified(t *testing.T) {
	bs := newAnthropicReasoningBillingService(map[string]*ModelPricing{"public-alias": {InputPricePerToken: 1}})
	for _, unified := range []bool{false, true} {
		gateway := &GatewayService{billingService: bs}
		openai := &OpenAIGatewayService{billingService: bs}
		if unified {
			gateway.resolver = NewModelPricingResolver(nil, bs)
			openai.resolver = gateway.resolver
		}
		key := &APIKey{Group: &Group{ID: 1}}
		result := &ForwardResult{Model: "public-alias", UpstreamModel: "claude-fable-5-1", Usage: ClaudeUsage{InputTokens: 10}, ReasoningEffort: optionalTrimmedStringPtr("max")}
		cost := gateway.calculateTokenCost(context.Background(), result, key, result.Model, 0.5, time.Time{}, &recordUsageOpts{})
		require.Equal(t, 30.0, cost.TotalCost)
		require.Equal(t, 15.0, cost.ActualCost)
		oc, err := openai.calculateOpenAIRecordUsageTokenCost(context.Background(), key, result.Model, 0.5, time.Time{}, UsageTokens{InputTokens: 10}, "", "max", result.UpstreamModel, nil)
		require.NoError(t, err)
		require.Equal(t, cost.TotalCost, oc.TotalCost)
	}
}

func TestAnthropicReasoningPolicyPlatformValidation(t *testing.T) {
	_, err := normalizeMaxReasoningEffortForPlatform(PlatformAnthropic, "max")
	require.NoError(t, err)
	_, err = normalizeMaxReasoningEffortForPlatform(PlatformAnthropic, "minimal")
	require.Error(t, err)
	_, err = NormalizeReasoningEffortMappings(PlatformAnthropic, []ReasoningEffortMapping{{From: "max", To: "high"}})
	require.NoError(t, err)
}

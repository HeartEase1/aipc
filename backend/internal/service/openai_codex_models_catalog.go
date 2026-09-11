package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"maps"
	"net/url"
	"sort"
	"strings"
)

const codexAutoModelPrefix = "codex-auto-"

func CodexModelsManifestETag(body []byte) string {
	return codexModelsManifestBodyETag(body)
}

func (s *GatewayService) CompositeCodexModelIDs(ctx context.Context, groupID int64) ([]string, error) {
	if s == nil || s.compositeResolver == nil || s.compositeResolver.repo == nil {
		return nil, nil
	}
	routes, err := s.compositeResolver.repo.ListByGroup(ctx, groupID, false)
	if err != nil {
		return nil, err
	}
	var models []string
	for _, route := range routes {
		if normalizeCompositeRouteMatchType(route.MatchType) == CompositeRouteMatchExact &&
			(normalizeCompositeRouteEndpoint(route.Endpoint) == CompositeRouteEndpointResponses || normalizeCompositeRouteEndpoint(route.Endpoint) == CompositeRouteEndpointAny) {
			models = append(models, route.PublicModel)
		}
	}
	return dedupeAndSortModelIDs(models), nil
}

func FilterCodexModelIDsForGroup(modelIDs []string, group *Group) []string {
	explicitlyEnabled := make(map[string]struct{})
	if group != nil && group.CustomModelsListEnabled() {
		for _, modelID := range group.ModelsListConfig.Models {
			modelID = strings.TrimSpace(modelID)
			if strings.HasPrefix(modelID, codexAutoModelPrefix) {
				explicitlyEnabled[modelID] = struct{}{}
			}
		}
	}

	filtered := make([]string, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if isCodexDedicatedMediaModel(modelID) {
			continue
		}
		if strings.Contains(modelID, "*") {
			continue
		}
		if strings.HasPrefix(modelID, codexAutoModelPrefix) {
			if _, ok := explicitlyEnabled[modelID]; !ok {
				continue
			}
		}
		filtered = append(filtered, modelID)
	}
	return filtered
}

func isCodexDedicatedMediaModel(modelID string) bool {
	canonical := codexProviderQualifiedModelID(modelID)
	return IsGPTImageGenerationModel(canonical) ||
		isImageGenerationModel(canonical) ||
		strings.HasPrefix(strings.ToLower(xai.StripGrokProviderPrefix(modelID)), "grok-imagine") ||
		strings.HasPrefix(strings.ToLower(xai.StripGrokProviderPrefix(modelID)), "imagine") ||
		isGrokVideoGenerationModel(canonical)
}

func codexProviderQualifiedModelID(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if slash := strings.LastIndexByte(modelID, '/'); slash >= 0 {
		modelID = strings.TrimSpace(modelID[slash+1:])
	}
	return strings.TrimPrefix(modelID, "models/")
}

// BuildGroupConfiguredCodexModelsManifest builds a Codex catalog exclusively
// from the public model names configured on accounts in an OpenAI group. The
// boolean result distinguishes "no explicit configuration" from a configured
// catalog that becomes empty after group-level filtering.
func (s *OpenAIGatewayService) BuildGroupConfiguredCodexModelsManifest(
	ctx context.Context,
	group *Group,
	ifNoneMatch string,
) (*CodexModelsManifest, bool, error) {
	if s == nil || s.accountRepo == nil || group == nil || group.Platform != PlatformOpenAI {
		return nil, false, nil
	}

	visible, catalog, err := loadCodexGroupCatalogAccounts(ctx, s.accountRepo, group.ID)
	if err != nil {
		return nil, false, fmt.Errorf("load group configured Codex models: %w", err)
	}
	configuredModels := openAIConfiguredCodexModelIDsForGroup(visible, group)
	if len(configuredModels) == 0 {
		return nil, false, nil
	}

	body, err := buildCodexCatalogForGroup(ctx, s.channelService, group, PlatformOpenAI, configuredModels, catalog, nil, true)
	if err != nil {
		return nil, false, fmt.Errorf("initialize group configured Codex models: %w", err)
	}
	body, _, err = mergeConfiguredCodexModelsManifest(
		body,
		nil,
		group.ModelsListConfig.Models,
		group.CustomModelsListEnabled(),
	)
	if err != nil {
		return nil, false, fmt.Errorf("build group configured Codex models: %w", err)
	}
	manifest := &CodexModelsManifest{
		Body: body,
		ETag: codexModelsManifestBodyETag(body),
	}
	if codexModelsManifestETagMatches(ifNoneMatch, manifest.ETag) {
		manifest.Body = nil
		manifest.NotModified = true
	}
	return manifest, true, nil
}

// MergeGroupConfiguredCodexModels adds account model aliases that are visible
// to the authenticated OpenAI group without discarding metadata from upstream
// Codex model entries. A group's custom models list also filters the picker,
// matching the standard /v1/models display policy.
func (s *OpenAIGatewayService) MergeGroupConfiguredCodexModels(
	ctx context.Context,
	group *Group,
	manifest *CodexModelsManifest,
	ifNoneMatch string,
) error {
	if s == nil || s.accountRepo == nil || group == nil || manifest == nil || manifest.NotModified {
		return nil
	}
	if group.Platform != PlatformOpenAI || len(manifest.Body) == 0 {
		return nil
	}

	configuredModels, err := s.groupConfiguredCodexModelIDs(ctx, group)
	if err != nil {
		return fmt.Errorf("load group configured Codex models: %w", err)
	}
	body, changed, err := mergeConfiguredCodexModelsManifest(
		manifest.Body,
		configuredModels,
		group.ModelsListConfig.Models,
		group.CustomModelsListEnabled(),
	)
	if err != nil {
		return fmt.Errorf("merge group configured Codex models: %w", err)
	}
	if changed {
		manifest.Body = body
		manifest.ETag = codexModelsManifestBodyETag(body)
	}
	if codexModelsManifestETagMatches(ifNoneMatch, manifest.ETag) {
		manifest.Body = nil
		manifest.NotModified = true
	}
	return nil
}

func (s *OpenAIGatewayService) groupConfiguredCodexModelIDs(ctx context.Context, group *Group) ([]string, error) {
	if group == nil {
		return nil, nil
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupID(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	return openAIConfiguredCodexModelIDsForGroup(accounts, group), nil
}

// loadCodexGroupCatalogAccounts separates picker membership from capability
// intersection. visible accounts are currently schedulable and decide which
// public aliases appear. catalog accounts are persistently enabled group
// members; the availability query ignores transient rate-limit, overload, and
// temporary-unschedulable state so those conditions cannot widen advertised
// capabilities. Persistently disabled accounts are excluded because routing
// cannot select them. A failed availability query must not widen capabilities.
func loadCodexGroupCatalogAccounts(ctx context.Context, repo AccountRepository, groupID int64) (visible []Account, catalog []Account, err error) {
	if repo == nil {
		return nil, nil, nil
	}
	visible, err = repo.ListSchedulableByGroupID(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}
	groupAccounts, listErr := repo.ListModelAvailabilityCandidates(
		ctx,
		&groupID,
		[]string{
			PlatformAnthropic,
			PlatformOpenAI,
			PlatformGemini,
			PlatformAntigravity,
			PlatformGrok,
			PlatformKimi,
			PlatformZhipu,
			PlatformDeepseek,
		},
		false,
	)
	if listErr != nil {
		return nil, nil, listErr
	}
	return visible, groupAccounts, nil
}

func openAIConfiguredCodexModelIDs(accounts []Account) []string {
	seen := make(map[string]struct{})
	models := make([]string, 0)
	for i := range accounts {
		account := &accounts[i]
		if account.Platform != PlatformOpenAI {
			continue
		}
		for modelID := range account.GetModelMapping() {
			modelID = strings.TrimSpace(modelID)
			if modelID == "" || strings.Contains(modelID, "*") {
				continue
			}
			if _, exists := seen[modelID]; exists {
				continue
			}
			seen[modelID] = struct{}{}
			models = append(models, modelID)
		}
	}
	sort.Strings(models)
	return models
}

func openAIConfiguredCodexModelIDsForGroup(accounts []Account, group *Group) []string {
	models := openAIConfiguredCodexModelIDs(accounts)
	if group == nil || !group.CustomModelsListEnabled() {
		return models
	}

	seen := make(map[string]struct{}, len(models)+len(group.ModelsListConfig.Models))
	for _, modelID := range models {
		seen[modelID] = struct{}{}
	}
	for _, selectedModel := range group.ModelsListConfig.Models {
		selectedModel = strings.TrimSpace(selectedModel)
		if selectedModel == "" || strings.Contains(selectedModel, "*") {
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account.Platform != PlatformOpenAI {
				continue
			}
			mappedModel, matched := account.ResolveMappedModel(selectedModel)
			if !matched || strings.TrimSpace(mappedModel) == "" {
				continue
			}
			if _, exists := seen[selectedModel]; !exists {
				seen[selectedModel] = struct{}{}
				models = append(models, selectedModel)
			}
			break
		}
	}
	sort.Strings(models)
	return models
}

const (
	configuredCodexModelPriority       = 50
	configuredCodexCustomDescription   = "Custom model routed through aipc."
	configuredCodexFallbackContext     = 272_000
	configuredCodexDeepSeekV4Context   = 1_000_000
	configuredCodexGrokContext         = 500_000
	configuredCodexGrokBuildContext    = 256_000
	configuredCodexGPT56MaxContext     = 872_000
	configuredCodexGPT6AstraContext    = 1_050_000
	configuredCodexToolOutputMaxTokens = 10_000
)

type configuredCodexReasoningLevel struct {
	Effort      string `json:"effort"`
	Description string `json:"description"`
}

type configuredCodexTruncationPolicy struct {
	Mode  string `json:"mode"`
	Limit int64  `json:"limit"`
}

type configuredCodexServiceTier struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type configuredCodexModelMessages struct {
	InstructionsTemplate  string `json:"instructions_template"`
	InstructionsVariables any    `json:"instructions_variables"`
	Approvals             any    `json:"approvals"`
	CollaborationModes    any    `json:"collaboration_modes"`
	AutoReview            any    `json:"auto_review"`
	Permissions           any    `json:"permissions"`
	MultiAgent            any    `json:"multi_agent"`
	TokenBudget           any    `json:"token_budget"`
	GuardianV2            any    `json:"guardian_v2"`
}

// configuredCodexModelDescriptor is the minimum complete ModelInfo contract
// understood by current Codex clients. Several nullable fields are intentionally
// emitted: unlike ordinary OpenAI /v1/models entries, the Codex manifest parser
// requires them to be present.
type configuredCodexModelDescriptor struct {
	Slug                              string                          `json:"slug"`
	DisplayName                       string                          `json:"display_name"`
	Description                       string                          `json:"description"`
	DefaultReasoningLevel             *string                         `json:"default_reasoning_level,omitempty"`
	SupportedReasoningLevels          []configuredCodexReasoningLevel `json:"supported_reasoning_levels"`
	MultiAgentReasoningEffort         *string                         `json:"multi_agent_reasoning_effort,omitempty"`
	ShellType                         string                          `json:"shell_type"`
	Visibility                        string                          `json:"visibility"`
	SupportedInAPI                    bool                            `json:"supported_in_api"`
	Priority                          int                             `json:"priority"`
	AdditionalSpeedTiers              []string                        `json:"additional_speed_tiers"`
	ServiceTiers                      []configuredCodexServiceTier    `json:"service_tiers"`
	DefaultServiceTier                any                             `json:"default_service_tier"`
	AvailabilityNUX                   any                             `json:"availability_nux"`
	Upgrade                           any                             `json:"upgrade"`
	ModelMessages                     configuredCodexModelMessages    `json:"model_messages"`
	IncludeSkillsUsageInstructions    bool                            `json:"include_skills_usage_instructions"`
	IncludePluginUsageInstructions    bool                            `json:"include_plugin_usage_instructions"`
	IncludeAppsUsageInstructions      bool                            `json:"include_apps_usage_instructions"`
	SupportsReasoningSummaryParameter bool                            `json:"supports_reasoning_summary_parameter"`
	DefaultReasoningSummary           string                          `json:"default_reasoning_summary"`
	SupportVerbosity                  bool                            `json:"support_verbosity"`
	DefaultVerbosity                  *string                         `json:"default_verbosity"`
	ApplyPatchToolType                *string                         `json:"apply_patch_tool_type"`
	WebSearchToolType                 string                          `json:"web_search_tool_type"`
	TruncationPolicy                  configuredCodexTruncationPolicy `json:"truncation_policy"`
	SupportsImageDetailOriginal       bool                            `json:"supports_image_detail_original"`
	SupportsParallelToolCalls         bool                            `json:"supports_parallel_tool_calls"`
	ContextWindow                     int64                           `json:"context_window"`
	MaxContextWindow                  int64                           `json:"max_context_window"`
	MaxOutputTokens                   int64                           `json:"max_output_tokens,omitempty"`
	AutoCompactTokenLimit             any                             `json:"auto_compact_token_limit"`
	CompHash                          any                             `json:"comp_hash"`
	EffectiveContextWindowPercent     int64                           `json:"effective_context_window_percent"`
	ExperimentalSupportedTools        []string                        `json:"experimental_supported_tools"`
	InputModalities                   []string                        `json:"input_modalities"`
	SupportsSearchTool                bool                            `json:"supports_search_tool"`
	UseResponsesLite                  bool                            `json:"use_responses_lite"`
	NodeREPLAutoReviewRequired        bool                            `json:"node_repl_auto_review_required"`
	NodeREPLDisabled                  bool                            `json:"node_repl_disabled"`
	AutoReviewModelOverride           any                             `json:"auto_review_model_override"`
	ModelSpecialty                    any                             `json:"model_specialty"`
	ToolMode                          any                             `json:"tool_mode"`
	MultiAgentVersion                 any                             `json:"multi_agent_version"`
}

type codexModelMetadataOverride struct {
	UpstreamModelMetadata
	reasoningConflict       bool
	inputModalitiesConflict bool
}

func newConfiguredCodexModelDescriptor(modelID string) configuredCodexModelDescriptor {
	modelID = strings.TrimSpace(modelID)
	noReasoningLevel := "none"
	descriptor := configuredCodexModelDescriptor{
		Slug:                  modelID,
		DisplayName:           modelID,
		Description:           configuredCodexCustomDescription,
		DefaultReasoningLevel: &noReasoningLevel,
		SupportedReasoningLevels: []configuredCodexReasoningLevel{
			{Effort: "none", Description: configuredCodexReasoningLevelDescription("none")},
		},
		ShellType:                         "unified_exec",
		Visibility:                        "list",
		SupportedInAPI:                    true,
		Priority:                          configuredCodexModelPriority,
		AdditionalSpeedTiers:              []string{},
		ServiceTiers:                      []configuredCodexServiceTier{},
		ModelMessages:                     configuredCodexModelMessages{InstructionsTemplate: openai.CodexBaseInstructionsForModel(modelID)},
		SupportsReasoningSummaryParameter: true,
		DefaultReasoningSummary:           "auto",
		WebSearchToolType:                 "text",
		TruncationPolicy:                  configuredCodexTruncationPolicy{Mode: "bytes", Limit: configuredCodexToolOutputMaxTokens},
		ContextWindow:                     configuredCodexFallbackContext,
		MaxContextWindow:                  configuredCodexFallbackContext,
		EffectiveContextWindowPercent:     95,
		ExperimentalSupportedTools:        []string{},
		InputModalities:                   []string{"text"},
	}

	if isDeepSeekCodexModel(modelID) {
		defaultReasoningLevel := "high"
		descriptor.DisplayName = deepSeekCodexDisplayName(modelID)
		descriptor.Description = "DeepSeek coding and reasoning model routed through aipc."
		descriptor.DefaultReasoningLevel = &defaultReasoningLevel
		descriptor.SupportedReasoningLevels = []configuredCodexReasoningLevel{
			{Effort: "low", Description: "Fast responses with lighter reasoning"},
			{Effort: "high", Description: "Greater reasoning depth for coding and agent tasks"},
			{Effort: "max", Description: "Maximum reasoning depth for complex tasks"},
		}
		descriptor.SupportsParallelToolCalls = true
		descriptor.ContextWindow = configuredCodexDeepSeekV4Context
		descriptor.MaxContextWindow = configuredCodexDeepSeekV4Context
	}

	if isGrokCodexModel(modelID) {
		descriptor.DisplayName = grokCodexDisplayName(modelID)
		descriptor.Description = "Grok coding and reasoning model routed through aipc."
		descriptor.SupportsParallelToolCalls = true
		descriptor.ContextWindow = grokCodexContextWindow(modelID)
		descriptor.MaxContextWindow = descriptor.ContextWindow
		if grokCodexSupportsReasoningEffort(modelID) {
			defaultReasoningLevel := "high"
			descriptor.DefaultReasoningLevel = &defaultReasoningLevel
			descriptor.SupportedReasoningLevels = configuredCodexGrokReasoningLevels(modelID)
		}
	}

	if isClaudeCodexModel(modelID) {
		descriptor.DisplayName = claudeCodexDisplayName(modelID)
		descriptor.Description = "Claude coding and reasoning model routed through aipc."
		descriptor.SupportsParallelToolCalls = true
		if levels := configuredCodexClaudeReasoningLevels(modelID); len(levels) > 0 {
			defaultReasoningLevel := claudeCodexDefaultReasoningLevel(levels)
			descriptor.DefaultReasoningLevel = &defaultReasoningLevel
			descriptor.SupportedReasoningLevels = levels
		}
	}

	if isOpenAICodexGPTModel(modelID) {
		descriptor.DisplayName = openaiCodexDisplayName(modelID)
		descriptor.Description = "OpenAI GPT coding model routed through aipc."
		descriptor.SupportsParallelToolCalls = true
		descriptor.ServiceTiers = configuredCodexServiceTiersForModel(modelID)
		if isOpenAICodexReasoningGPTModel(modelID) {
			defaultReasoningLevel := "medium"
			if getNormalizedCodexModel(modelID) == "gpt-5.6-sol" {
				defaultReasoningLevel = "low"
			}
			descriptor.DefaultReasoningLevel = &defaultReasoningLevel
			descriptor.SupportedReasoningLevels = configuredCodexGPTReasoningLevels(modelID)
			descriptor.DefaultReasoningSummary = "none"
			descriptor.TruncationPolicy = configuredCodexTruncationPolicy{Mode: "tokens", Limit: configuredCodexToolOutputMaxTokens}
			if isOpenAIGPT56Model(modelID) {
				descriptor.MaxContextWindow = configuredCodexGPT56MaxContext
			}
			if isOpenAIGPT6AstraModel(modelID) {
				multiAgentEffort := "xhigh"
				descriptor.MultiAgentReasoningEffort = &multiAgentEffort
				descriptor.MultiAgentVersion = "v2"
				descriptor.ContextWindow = configuredCodexGPT6AstraContext
				descriptor.MaxContextWindow = configuredCodexGPT6AstraContext
			}
		}
		if SupportsVerbosity(modelID) {
			defaultVerbosity := "low"
			descriptor.SupportVerbosity = true
			descriptor.DefaultVerbosity = &defaultVerbosity
		}
	}

	return descriptor
}

func configuredCodexServiceTiersForModel(modelID string) []configuredCodexServiceTier {
	tiers := make([]configuredCodexServiceTier, 0, 2)
	if configuredCodexSupportsPriorityServiceTier(modelID) {
		tiers = append(tiers, configuredCodexServiceTier{
			ID:          OpenAIFastTierPriority,
			Name:        "Fast",
			Description: "Priority processing for lower latency.",
		})
	}
	if configuredCodexSupportsUltrafastServiceTier(modelID) {
		tiers = append(tiers, configuredCodexServiceTier{
			ID:          OpenAIFastTierUltrafast,
			Name:        "Ultrafast",
			Description: "Ultra-low latency processing.",
		})
	}
	return tiers
}

func configuredCodexSupportsPriorityServiceTier(modelID string) bool {
	normalized := canonicalizeOpenAIModelAliasSpelling(modelID)
	for _, family := range []string{"gpt-5.4", "gpt-5.5", "gpt-5.6"} {
		if normalized == family || strings.HasPrefix(normalized, family+"-") {
			return true
		}
	}
	// GPT-6 Astra advertises Fast via service_tier=priority in public model metadata.
	return isOpenAIGPT6AstraModel(modelID)
}

func configuredCodexSupportsUltrafastServiceTier(modelID string) bool {
	return normalizeKnownOpenAICodexModel(modelID) == "gpt-5.6-sol"
}

func configuredCodexGrokReasoningLevels(modelID string) []configuredCodexReasoningLevel {
	levels := []configuredCodexReasoningLevel{
		{Effort: "low", Description: "Fast responses with lighter reasoning"},
		{Effort: "medium", Description: "Balanced reasoning for most coding tasks"},
		{Effort: "high", Description: "Greater reasoning depth for coding and agent tasks"},
	}
	if grokSupportsXHighReasoningEffort(modelID) {
		levels = append(levels, configuredCodexReasoningLevel{
			Effort:      "xhigh",
			Description: "Extra-high reasoning depth for difficult tasks",
		})
	}
	return levels
}

func configuredCodexClaudeReasoningLevels(modelID string) []configuredCodexReasoningLevel {
	descriptions := map[string]string{
		"low":    "Fast responses with lighter reasoning",
		"medium": "Balanced reasoning for most coding tasks",
		"high":   "Greater reasoning depth for coding and agent tasks",
		"xhigh":  "Extra-high reasoning depth for difficult tasks",
		"max":    "Maximum reasoning depth for complex tasks",
	}
	levels := claude.EffortLevelsForModel(modelID)
	out := make([]configuredCodexReasoningLevel, 0, len(levels))
	for _, effort := range levels {
		out = append(out, configuredCodexReasoningLevel{
			Effort:      effort,
			Description: descriptions[effort],
		})
	}
	return out
}

func claudeCodexDefaultReasoningLevel(levels []configuredCodexReasoningLevel) string {
	for _, preferred := range []string{"medium", "high", "low"} {
		for _, level := range levels {
			if level.Effort == preferred {
				return preferred
			}
		}
	}
	if len(levels) == 0 {
		return ""
	}
	return levels[0].Effort
}

func configuredCodexGPTReasoningLevels(modelID string) []configuredCodexReasoningLevel {
	levels := []configuredCodexReasoningLevel{
		{Effort: "low", Description: "Fast responses with lighter reasoning"},
		{Effort: "medium", Description: "Balanced reasoning for most coding tasks"},
		{Effort: "high", Description: "Greater reasoning depth for coding and agent tasks"},
		{Effort: "xhigh", Description: "Extra-high reasoning depth for difficult tasks"},
	}
	normalized := getNormalizedCodexModel(modelID)
	if isOpenAIGPT56Model(modelID) || isOpenAIGPT6AstraModel(modelID) {
		levels = append(levels, configuredCodexReasoningLevel{
			Effort:      "max",
			Description: "Maximum reasoning depth for complex tasks",
		})
	}
	if isOpenAIGPT6AstraModel(modelID) || normalized == "gpt-5.6-sol" || normalized == "gpt-5.6-terra" {
		levels = append(levels, configuredCodexReasoningLevel{
			Effort:      "ultra",
			Description: "Maximum reasoning with automatic task delegation",
		})
	}
	return levels
}

func isOpenAICodexGPTModel(modelID string) bool {
	normalized := canonicalizeOpenAIModelAliasSpelling(modelID)
	if normalized == "" || strings.HasPrefix(normalized, "gpt-image") {
		return false
	}
	return strings.HasPrefix(normalized, "gpt-")
}

func isOpenAICodexReasoningGPTModel(modelID string) bool {
	normalized := canonicalizeOpenAIModelAliasSpelling(modelID)
	return isOpenAIGPT6AstraModel(normalized) || strings.HasPrefix(normalized, "gpt-5")
}

func isOpenAICodexImageInputModel(modelID string) bool {
	normalized := canonicalizeOpenAIModelAliasSpelling(modelID)
	return isOpenAIGPT6AstraModel(normalized) ||
		strings.HasPrefix(normalized, "gpt-5") ||
		strings.HasPrefix(normalized, "gpt-4o") ||
		strings.HasPrefix(normalized, "gpt-4.1") ||
		strings.HasPrefix(normalized, "gpt-4.5") ||
		strings.HasPrefix(normalized, "gpt-4-turbo") ||
		strings.HasPrefix(normalized, "gpt-4-vision")
}

func isOfficialOpenAICodexCatalogModel(modelID string) bool {
	normalized := strings.ToLower(codexProviderQualifiedModelID(modelID))
	if normalized == "" || isCodexDedicatedMediaModel(normalized) {
		return false
	}
	if strings.HasPrefix(normalized, "codex-") {
		return true
	}
	if strings.HasPrefix(normalized, "o1") || strings.HasPrefix(normalized, "o3") || strings.HasPrefix(normalized, "o4") {
		return true
	}
	if !strings.HasPrefix(normalized, "gpt-") {
		return false
	}
	for _, incompatibleFamily := range []string{"audio", "realtime", "transcribe", "tts"} {
		if strings.Contains(normalized, incompatibleFamily) {
			return false
		}
	}
	return true
}

func openaiCodexDisplayName(modelID string) string {
	normalized := canonicalizeOpenAIModelAliasSpelling(modelID)
	if normalized == "" {
		return modelID
	}
	for _, model := range openai.DefaultModels {
		if strings.EqualFold(model.ID, normalized) && strings.TrimSpace(model.DisplayName) != "" {
			return model.DisplayName
		}
	}
	return modelID
}

func deepSeekCodexDisplayName(modelID string) string {
	switch strings.ToLower(strings.TrimSpace(modelID)) {
	case "deepseek-v4-pro", "deepseek-4-pro":
		return "DeepSeek V4 Pro"
	case "deepseek-v4-flash", "deepseek-4-flash":
		return "DeepSeek V4 Flash"
	default:
		return modelID
	}
}

func isDeepSeekCodexModel(modelID string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(modelID)), "deepseek-")
}

func isGrokCodexModel(modelID string) bool {
	return xai.IsGrokModelID(modelID)
}

func grokCodexSupportsReasoningEffort(modelID string) bool {
	if grokSupportsReasoningEffort(modelID) {
		return true
	}
	canonical := xai.ResolveGrokTextResponsesModelID(modelID)
	if canonical == "" || strings.EqualFold(canonical, modelID) {
		return false
	}
	return grokSupportsReasoningEffort(canonical)
}

func grokCodexDisplayName(modelID string) string {
	normalized := strings.ToLower(xai.StripGrokProviderPrefix(strings.TrimSpace(modelID)))
	if normalized == "" {
		return modelID
	}
	if name := grokDefaultDisplayName(normalized); name != "" {
		return name
	}
	canonical := strings.ToLower(xai.ResolveGrokTextResponsesModelID(normalized))
	if canonical != "" && canonical != normalized {
		if name := grokDefaultDisplayName(canonical); name != "" {
			return name
		}
	}
	return modelID
}

func grokDefaultDisplayName(modelID string) string {
	for _, model := range xai.DefaultModels() {
		if model.ID == modelID {
			return strings.TrimSpace(model.DisplayName)
		}
	}
	return ""
}

func grokCodexContextWindow(modelID string) int64 {
	normalized := strings.ToLower(xai.StripGrokProviderPrefix(strings.TrimSpace(modelID)))
	if strings.HasPrefix(normalized, "grok-build") {
		return configuredCodexGrokBuildContext
	}
	return configuredCodexGrokContext
}

func isClaudeCodexModel(modelID string) bool {
	platform, detected := DetectModelPlatform(modelID)
	return detected && platform == PlatformAnthropic
}

func claudeCodexDisplayName(modelID string) string {
	normalized := strings.ToLower(codexProviderQualifiedModelID(modelID))
	normalized = strings.TrimPrefix(normalized, "anthropic.")
	if normalized == "" {
		return modelID
	}
	for _, model := range claude.DefaultModels {
		if strings.EqualFold(model.ID, normalized) && strings.TrimSpace(model.DisplayName) != "" {
			return model.DisplayName
		}
	}
	if canonical, ok := claude.ModelIDOverrides[normalized]; ok {
		for _, model := range claude.DefaultModels {
			if model.ID == canonical && strings.TrimSpace(model.DisplayName) != "" {
				return model.DisplayName
			}
		}
	}
	return modelID
}

// BuildCodexModelsManifest builds a standalone Codex model catalog for models
// routed through a custom provider. The response is also suitable for saving
// as model_catalog_json in clients that do not refresh custom-provider catalogs.
func BuildCodexModelsManifest(modelIDs []string) ([]byte, error) {
	return buildCodexModelsManifest(modelIDs, nil, nil, nil, nil)
}

// BuildCodexModelsManifestForGroup derives input capabilities from the
// concrete Responses route and group accounts behind a group. Unknown or mixed
// capabilities fail closed to the text-only descriptor used by the standalone
// builder. Caller-supplied model IDs still decide which slugs appear; advertised
// capabilities intersect all active group members that map the alias, including
// accounts that are not currently schedulable.
func (s *GatewayService) BuildCodexModelsManifestForGroup(
	ctx context.Context,
	group *Group,
	platformOverride string,
	modelIDs []string,
) ([]byte, error) {
	if s == nil || s.accountRepo == nil || group == nil {
		return BuildCodexModelsManifest(modelIDs)
	}
	effectivePlatform := strings.TrimSpace(platformOverride)
	if effectivePlatform == "" {
		effectivePlatform = group.Platform
	}
	if effectivePlatform != PlatformComposite && !isConcreteRequestPlatform(effectivePlatform) {
		return BuildCodexModelsManifest(modelIDs)
	}

	_, catalog, err := loadCodexGroupCatalogAccounts(ctx, s.accountRepo, group.ID)
	if err != nil {
		return BuildCodexModelsManifest(modelIDs)
	}
	var compositeRoutes []CompositeModelRoute
	compositeRoutesAvailable := true
	if effectivePlatform == PlatformComposite && s.compositeResolver != nil && s.compositeResolver.repo != nil {
		compositeRoutes, err = s.compositeResolver.repo.ListByGroup(ctx, group.ID, false)
		if err != nil {
			compositeRoutesAvailable = false
		}
	}
	return buildCodexCatalogForGroup(ctx, s.channelService, group, effectivePlatform, modelIDs, catalog, compositeRoutes, compositeRoutesAvailable)
}

// Resolve the same composite, channel, and account mapping chain used by forwarding.
func buildCodexCatalogForGroup(ctx context.Context, channels *ChannelService, group *Group, platform string, modelIDs []string, accounts []Account, routes []CompositeModelRoute, routesAvailable bool) ([]byte, error) {
	var bodies [][]byte
	for _, modelID := range dedupeAndSortModelIDs(modelIDs) {
		if group.IsModelBlocked(modelID) {
			continue
		}
		targetPlatform, routedModel := platform, modelID
		if platform == PlatformComposite {
			var resolved bool
			targetPlatform, routedModel, resolved = resolveCodexCompositeModelTarget(modelID, accounts, routes, routesAvailable)
			if !resolved {
				continue
			}
		}
		requestCtx := WithResolvedTargetPlatform(ctx, targetPlatform)
		mappedModel := routedModel
		if channels != nil {
			mappedModel = channels.ResolveChannelMapping(requestCtx, group.ID, routedModel).MappedModel
		}
		requestCtx = WithOpenAIForwardModel(requestCtx, mappedModel, false, routedModel)
		var candidates []Account
		for i := range accounts {
			account := &accounts[i]
			if account.Platform != targetPlatform {
				continue
			}
			supported := account.IsModelSupported(mappedModel)
			if targetPlatform == PlatformOpenAI {
				supported = openAIAccountSupportsSchedulingModel(requestCtx, account, mappedModel)
			}
			if !supported {
				continue
			}
			candidate := *account
			candidate.Credentials = maps.Clone(account.Credentials)
			if candidate.Credentials == nil {
				candidate.Credentials = make(map[string]any)
			}
			candidate.Credentials["model_mapping"] = map[string]any{modelID: account.GetMappedModel(mappedModel)}
			candidates = append(candidates, candidate)
		}
		if len(candidates) == 0 {
			continue
		}
		body, err := buildCodexModelsManifestForAccounts(targetPlatform, []string{modelID}, candidates, group, nil, true)
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, body)
	}
	if len(bodies) == 0 {
		return []byte(`{"models":[]}`), nil
	}
	return mergeCodexModelsManifestBodies(bodies)
}

func buildCodexModelsManifestForAccounts(
	effectivePlatform string,
	modelIDs []string,
	accounts []Account,
	group *Group,
	compositeRoutes []CompositeModelRoute,
	compositeRoutesAvailable bool,
) ([]byte, error) {
	imageInputModels := make(map[string]bool, len(modelIDs))
	searchToolModels := make(map[string]bool, len(modelIDs))
	metadataModels := codexCatalogMetadataModels(
		effectivePlatform,
		modelIDs,
		accounts,
		compositeRoutes,
		compositeRoutesAvailable,
	)
	modelMetadata := make(map[string]codexModelMetadataOverride, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if groupCodexModelSupportsImageInput(
			effectivePlatform,
			modelID,
			accounts,
			compositeRoutes,
			compositeRoutesAvailable,
		) {
			imageInputModels[modelID] = true
		}
		if groupCodexModelSupportsSearchTool(
			effectivePlatform,
			modelID,
			accounts,
			compositeRoutes,
			compositeRoutesAvailable,
		) {
			searchToolModels[modelID] = true
		}
		if metadata, ok := groupCodexModelMetadata(
			effectivePlatform,
			modelID,
			accounts,
			group,
			compositeRoutes,
			compositeRoutesAvailable,
		); ok {
			modelMetadata[modelID] = metadata
		}
	}
	return buildCodexModelsManifest(modelIDs, imageInputModels, searchToolModels, metadataModels, modelMetadata)
}

func buildCodexModelsManifest(
	modelIDs []string,
	imageInputModels map[string]bool,
	searchToolModels map[string]bool,
	metadataModels map[string]string,
	modelMetadata map[string]codexModelMetadataOverride,
) ([]byte, error) {
	seen := make(map[string]struct{}, len(modelIDs))
	models := make([]json.RawMessage, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		metadataModelID := strings.TrimSpace(metadataModels[modelID])
		if metadataModelID == "" {
			metadataModelID = modelID
		}
		if isCodexDedicatedMediaModel(modelID) || isCodexDedicatedMediaModel(metadataModelID) {
			continue
		}
		seen[modelID] = struct{}{}
		descriptor := newConfiguredCodexModelDescriptor(metadataModelID)
		descriptor.Slug = modelID
		if imageInputModels[modelID] {
			descriptor.InputModalities = []string{"text", "image"}
		}
		descriptor.SupportsSearchTool = searchToolModels[modelID]
		if metadata, ok := modelMetadata[modelID]; ok {
			applyUpstreamModelMetadataToCodexDescriptor(&descriptor, metadata)
		}
		if metadataModelID != modelID {
			descriptor.DisplayName = modelID
			descriptor.Description = configuredCodexCustomDescription
		}
		encoded, err := json.Marshal(descriptor)
		if err != nil {
			return nil, err
		}
		if capabilities := modelMetadata[modelID].CodexToolCapabilities; len(capabilities) > 0 {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				return nil, err
			}
			applyCodexToolCapabilities(fields, capabilities, true)
			encoded, err = json.Marshal(fields)
			if err != nil {
				return nil, err
			}
		}
		models = append(models, encoded)
	}
	return json.Marshal(struct {
		Models []json.RawMessage `json:"models"`
	}{Models: models})
}

func codexCatalogMetadataModels(
	platform string,
	modelIDs []string,
	accounts []Account,
	compositeRoutes []CompositeModelRoute,
	compositeRoutesAvailable bool,
) map[string]string {
	metadataModels := make(map[string]string, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		metadataModelID := resolveCodexCatalogMetadataModel(
			platform,
			modelID,
			accounts,
			compositeRoutes,
			compositeRoutesAvailable,
		)
		if metadataModelID != "" && metadataModelID != modelID {
			metadataModels[modelID] = metadataModelID
		}
	}
	return metadataModels
}

func resolveCodexCatalogMetadataModel(
	platform string,
	modelID string,
	accounts []Account,
	compositeRoutes []CompositeModelRoute,
	compositeRoutesAvailable bool,
) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	if platform == PlatformComposite {
		if !compositeRoutesAvailable {
			return modelID
		}
		if route, matched := matchCompositeRoute(compositeRoutes, modelID, CompositeRouteEndpointResponses); matched {
			if upstreamModel := strings.TrimSpace(route.UpstreamModel); upstreamModel != "" {
				return upstreamModel
			}
			return modelID
		}
		if codexCompositeRouteMatchesModel(compositeRoutes, modelID) {
			return modelID
		}

		claimedPlatforms := make(map[string]struct{})
		for _, account := range accounts {
			accountPlatform := strings.TrimSpace(account.Platform)
			if !isConcreteRequestPlatform(accountPlatform) || !codexExplicitModelMappingClaims(account, modelID) {
				continue
			}
			claimedPlatforms[accountPlatform] = struct{}{}
		}
		if len(claimedPlatforms) > 1 {
			return modelID
		}
		for accountPlatform := range claimedPlatforms {
			return uniqueCodexMappedModel(accounts, accountPlatform, modelID)
		}

		detectedPlatform, detected := DetectModelPlatform(modelID)
		if !detected {
			return modelID
		}
		platform = detectedPlatform
	}
	return uniqueCodexMappedModel(accounts, platform, modelID)
}

func uniqueCodexMappedModel(accounts []Account, platform string, modelID string) string {
	targets := make(map[string]struct{})
	for i := range accounts {
		account := &accounts[i]
		if account.Platform != platform {
			continue
		}
		mappedModel, matched := account.ResolveMappedModel(modelID)
		mappedModel = strings.TrimSpace(mappedModel)
		if !matched || mappedModel == "" {
			continue
		}
		targets[mappedModel] = struct{}{}
	}
	if len(targets) != 1 {
		return modelID
	}
	for target := range targets {
		return target
	}
	return modelID
}

func groupCodexModelSupportsImageInput(
	platform string,
	modelID string,
	accounts []Account,
	compositeRoutes []CompositeModelRoute,
	compositeRoutesAvailable bool,
) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	upstreamModel := modelID
	if platform == PlatformComposite {
		var resolved bool
		platform, upstreamModel, resolved = resolveCodexCompositeModelTarget(
			modelID,
			accounts,
			compositeRoutes,
			compositeRoutesAvailable,
		)
		if !resolved {
			return false
		}
	}
	if platform != PlatformOpenAI && platform != PlatformGrok {
		return false
	}

	candidates := 0
	for i := range accounts {
		account := &accounts[i]
		if account.Platform != platform || !account.IsModelSupported(upstreamModel) {
			continue
		}
		candidates++
		if !accountCodexModelSupportsImageInput(account, account.GetMappedModel(upstreamModel)) {
			return false
		}
	}
	return candidates > 0
}

// groupCodexModelSupportsSearchTool advertises client-side tool discovery only
// when every account that may serve the model uses the gateway's Responses to
// Chat Completions bridge. Native Responses routes must declare the capability
// in their upstream Codex manifest instead of having it inferred here.
func groupCodexModelSupportsSearchTool(
	platform string,
	modelID string,
	accounts []Account,
	compositeRoutes []CompositeModelRoute,
	compositeRoutesAvailable bool,
) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	upstreamModel := modelID
	if platform == PlatformComposite {
		var resolved bool
		platform, upstreamModel, resolved = resolveCodexCompositeModelTarget(
			modelID,
			accounts,
			compositeRoutes,
			compositeRoutesAvailable,
		)
		if !resolved {
			return false
		}
	}
	if platform != PlatformOpenAI {
		return false
	}

	candidates := 0
	for i := range accounts {
		account := &accounts[i]
		if account.Platform != platform || !account.IsModelSupported(upstreamModel) {
			continue
		}
		candidates++
		if !shouldForwardOpenAIResponsesViaRawChatCompletions(account) {
			return false
		}
	}
	return candidates > 0
}

func resolveCodexCompositeModelTarget(
	modelID string,
	accounts []Account,
	routes []CompositeModelRoute,
	routesAvailable bool,
) (string, string, bool) {
	if !routesAvailable {
		return "", "", false
	}
	if route, matched := matchCompositeRoute(routes, modelID, CompositeRouteEndpointResponses); matched {
		upstreamModel := strings.TrimSpace(route.UpstreamModel)
		if upstreamModel == "" {
			upstreamModel = modelID
		}
		return route.TargetPlatform, upstreamModel, true
	}
	if codexCompositeRouteMatchesModel(routes, modelID) {
		return "", "", false
	}

	platform, detected := DetectModelPlatform(modelID)
	if !detected {
		return "", "", false
	}
	return platform, modelID, true
}

func codexCompositeRouteMatchesModel(routes []CompositeModelRoute, modelID string) bool {
	for _, route := range routes {
		publicModel := strings.TrimSpace(route.PublicModel)
		if publicModel == "" {
			continue
		}
		switch normalizeCompositeRouteMatchType(route.MatchType) {
		case CompositeRouteMatchPrefix:
			if strings.HasPrefix(modelID, publicModel) {
				return true
			}
		default:
			if modelID == publicModel {
				return true
			}
		}
	}
	return false
}

func codexExplicitModelMappingClaims(account Account, modelID string) bool {
	if account.Credentials == nil || strings.TrimSpace(modelID) == "" {
		return false
	}
	mapped := strings.TrimSpace(account.GetModelMapping()[modelID])
	return mapped != ""
}

func accountCodexModelSupportsImageInput(account *Account, upstreamModel string) bool {
	if account == nil {
		return false
	}
	switch account.Platform {
	case PlatformOpenAI:
		if metadata, ok := account.GetUpstreamModelMetadata(upstreamModel); ok {
			if modalities := normalizeCodexInputModalities(metadata.InputModalities); len(modalities) > 0 {
				return stringSliceContains(modalities, "image")
			}
		}
		if !isOpenAICodexImageInputModel(upstreamModel) {
			return false
		}
		if account.IsOpenAIOAuth() {
			return true
		}
		if !account.IsOpenAIApiKey() {
			return false
		}
		// Compatible model lists often omit modalities. Preserve the known GPT
		// fallback unless a synced snapshot above explicitly narrows it.
		return true
	case PlatformGrok:
		if !isOfficialGrokCodexBaseURL(account.GetGrokBaseURL()) {
			return false
		}
		canonical := xai.ResolveGrokTextResponsesModelID(upstreamModel)
		return isGrokCodexImageInputModel(canonical)
	default:
		return false
	}
}

func isGrokCodexImageInputModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "grok-4.3",
		"grok-4.5",
		"grok-4.6",
		"grok-build-0.1",
		"grok-4.20-0309-reasoning",
		"grok-4.20-0309-non-reasoning",
		"grok-4.20-multi-agent-0309":
		return true
	default:
		return false
	}
}

func isOfficialGrokCodexBaseURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	return xai.IsOfficialBaseURLHost(strings.TrimSuffix(parsed.Hostname(), "."))
}

// BuildDeepSeekCodexModelsManifest preserves the historical entry point for
// callers that still use the provider-specific function name.
func BuildDeepSeekCodexModelsManifest(modelIDs []string) ([]byte, error) {
	return BuildCodexModelsManifest(modelIDs)
}

func mergeConfiguredCodexModelsManifest(
	body []byte,
	configuredModels []string,
	selectedModels []string,
	filterBySelection bool,
) ([]byte, bool, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, false, err
	}
	var upstreamModels []json.RawMessage
	if err := json.Unmarshal(envelope["models"], &upstreamModels); err != nil {
		return nil, false, err
	}

	selected := make(map[string]struct{}, len(selectedModels))
	for _, modelID := range selectedModels {
		modelID = strings.TrimSpace(modelID)
		if modelID != "" {
			selected[modelID] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(upstreamModels)+len(configuredModels))
	merged := make([]json.RawMessage, 0, len(upstreamModels)+len(configuredModels))
	changed := false
	for _, rawModel := range upstreamModels {
		var descriptor struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(rawModel, &descriptor); err != nil || strings.TrimSpace(descriptor.Slug) == "" {
			if filterBySelection {
				changed = true
				continue
			}
			merged = append(merged, rawModel)
			continue
		}
		descriptor.Slug = strings.TrimSpace(descriptor.Slug)
		if isCodexDedicatedMediaModel(descriptor.Slug) {
			changed = true
			continue
		}
		if filterBySelection {
			if _, allowed := selected[descriptor.Slug]; !allowed {
				changed = true
				continue
			}
		}
		if strings.HasPrefix(descriptor.Slug, codexAutoModelPrefix) {
			_, explicitlyEnabled := selected[descriptor.Slug]
			explicitlyEnabled = filterBySelection && explicitlyEnabled
			if !explicitlyEnabled {
				changed = true
				continue
			}
			visibleModel, visibilityChanged, err := codexModelWithVisibility(rawModel, "list")
			if err != nil {
				return nil, false, err
			}
			rawModel = visibleModel
			changed = changed || visibilityChanged
		}
		seen[descriptor.Slug] = struct{}{}
		merged = append(merged, rawModel)
	}

	for _, modelID := range configuredModels {
		if isCodexDedicatedMediaModel(modelID) {
			continue
		}
		if filterBySelection {
			if _, allowed := selected[modelID]; !allowed {
				continue
			}
		}
		if strings.HasPrefix(modelID, codexAutoModelPrefix) {
			if _, explicitlyEnabled := selected[modelID]; !filterBySelection || !explicitlyEnabled {
				continue
			}
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		rawModel, err := json.Marshal(newConfiguredCodexModelDescriptor(modelID))
		if err != nil {
			return nil, false, err
		}
		merged = append(merged, rawModel)
		seen[modelID] = struct{}{}
		changed = true
	}
	if !changed {
		return body, false, nil
	}

	rawModels, err := json.Marshal(merged)
	if err != nil {
		return nil, false, err
	}
	envelope["models"] = rawModels
	mergedBody, err := json.Marshal(envelope)
	if err != nil {
		return nil, false, err
	}
	return mergedBody, true, nil
}

func codexModelWithVisibility(rawModel json.RawMessage, visibility string) (json.RawMessage, bool, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawModel, &fields); err != nil {
		return nil, false, err
	}
	var current string
	if rawVisibility, ok := fields["visibility"]; ok {
		if err := json.Unmarshal(rawVisibility, &current); err == nil && current == visibility {
			return rawModel, false, nil
		}
	}
	rawVisibility, err := json.Marshal(visibility)
	if err != nil {
		return nil, false, err
	}
	fields["visibility"] = rawVisibility
	updated, err := json.Marshal(fields)
	if err != nil {
		return nil, false, err
	}
	return updated, true, nil
}

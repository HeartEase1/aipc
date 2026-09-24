package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/gin-gonic/gin"
)

type pelicanTestContextKey struct{}

type pelicanTestOptions struct {
	prompt          string
	reasoningEffort string
}

const pelicanMaxOutputTokens = 16384

type PelicanTestCapabilities struct {
	ModelID                  string   `json:"model_id"`
	SupportedReasoningLevels []string `json:"supported_reasoning_levels"`
}

// ResolvePelicanTestOptions reuses the account's mapped-model metadata. Ultra is
// a Codex client orchestration mode, not a single inference effort, so it is not
// advertised by this single-request test.
func ResolvePelicanTestOptions(account *Account, modelID string) (*PelicanTestCapabilities, error) {
	modelID = strings.TrimSpace(modelID)
	if account == nil || modelID == "" || len(modelID) > 256 || strings.ContainsAny(modelID, "\r\n\x00") {
		return nil, fmt.Errorf("a valid text model is required")
	}
	if !account.IsOpenAIPassthroughEnabled() && !account.IsModelSupported(modelID) {
		return nil, fmt.Errorf("model %q is not allowed by this account", modelID)
	}
	target := modelID
	if !account.IsOpenAIPassthroughEnabled() {
		target = account.GetMappedModel(modelID)
	}
	lower := strings.ToLower(target)
	if isOpenAIImageGenerationModel(lower) || isGrokImageGenerationModel(lower) || isGrokVideoGenerationModel(lower) || isGeminiCompatibleImageModel(lower) {
		return nil, fmt.Errorf("pelican tests require a text-generation model")
	}
	for _, marker := range []string{"embedding", "rerank", "tts", "transcribe", "realtime", "whisper", "-audio", "grok-voice"} {
		if strings.Contains(lower, marker) {
			return nil, fmt.Errorf("pelican tests require a text-generation model")
		}
	}
	levels := []string{}
	switch {
	case strings.Contains(lower, "gemini-3"):
		levels = []string{"low", "high"}
		if strings.Contains(lower, "flash") {
			levels = []string{"minimal", "low", "medium", "high"}
		}
	case strings.Contains(lower, "gemini-2.5"):
		levels = []string{"low", "medium", "high"}
	default:
		for _, level := range newConfiguredCodexModelDescriptor(target).SupportedReasoningLevels {
			if level.Effort != "none" {
				levels = append(levels, level.Effort)
			}
		}
	}
	if metadata, ok := account.GetUpstreamModelMetadata(target); ok {
		if metadata.Reasoning != nil && !*metadata.Reasoning {
			levels = nil
		} else if len(metadata.SupportedReasoningLevels) > 0 {
			levels = metadata.SupportedReasoningLevels
		}
	}
	// The Antigravity Claude adapter exposes thinking budgets, not output_config.
	if account.Platform == PlatformAntigravity && account.Type != AccountTypeAPIKey && isClaudeCodexModel(target) {
		levels = []string{"low", "medium", "high"}
	}
	result := &PelicanTestCapabilities{ModelID: target, SupportedReasoningLevels: []string{}}
	for _, level := range levels {
		if value := normalizePelicanReasoningEffort(level); value != "" && !slices.Contains(result.SupportedReasoningLevels, value) {
			result.SupportedReasoningLevels = append(result.SupportedReasoningLevels, value)
		}
	}
	return result, nil
}

func withPelicanTestOptions(ctx context.Context, options pelicanTestOptions) context.Context {
	return context.WithValue(ctx, pelicanTestContextKey{}, options)
}

func pelicanTestOptionsFromContext(ctx context.Context) (pelicanTestOptions, bool) {
	options, ok := ctx.Value(pelicanTestContextKey{}).(pelicanTestOptions)
	return options, ok
}

// TestPelicanAccountConnection is the dedicated account test path for the
// Pelican UI. The existing /test endpoint deliberately keeps its historical
// probe payload; only this endpoint opts into the user prompt and reasoning.
func (s *AccountTestService) TestPelicanAccountConnection(c *gin.Context, accountID int64, modelID, prompt, reasoningEffort string) error {
	if strings.TrimSpace(prompt) == "" || utf8.RuneCountInString(prompt) > 16000 || strings.TrimSpace(modelID) == "" {
		return s.sendErrorAndEnd(c, "A model and prompt (at most 16000 characters) are required")
	}
	if strings.TrimSpace(reasoningEffort) != "" && normalizePelicanReasoningEffort(reasoningEffort) == "" {
		return s.sendErrorAndEnd(c, "Unsupported reasoning effort")
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()
	options := pelicanTestOptions{
		prompt:          strings.TrimSpace(prompt),
		reasoningEffort: normalizePelicanReasoningEffort(reasoningEffort),
	}
	ctx = withPelicanTestOptions(ctx, options)
	c.Request = c.Request.WithContext(ctx)
	return s.TestAccountConnection(c, accountID, modelID, options.prompt, AccountTestModeDefault)
}

func normalizePelicanReasoningEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func createPelicanClaudePayload(modelID, prompt string, efforts ...string) (map[string]any, error) {
	payload, err := createTestPayload(modelID)
	if err != nil {
		return nil, err
	}
	messages, ok := payload["messages"].([]map[string]any)
	if !ok || len(messages) == 0 {
		return payload, nil
	}
	content, ok := messages[0]["content"].([]map[string]any)
	if ok && len(content) > 0 {
		content[0]["text"] = promptOrDefault(prompt)
	}
	payload["max_tokens"] = pelicanMaxOutputTokens
	delete(payload, "temperature")
	if len(efforts) > 0 && efforts[0] != "" {
		if efforts[0] == "none" {
			payload["thinking"] = map[string]any{"type": "disabled"}
		} else {
			payload["output_config"] = map[string]any{"effort": efforts[0]}
			// Opus 4.5 supports effort, but predates adaptive thinking.
			if len(claude.EffortLevelsForModel(modelID)) > 0 && !strings.Contains(modelID, "opus-4-5") && !strings.Contains(modelID, "opus-4.5") {
				payload["thinking"] = map[string]any{"type": "adaptive"}
			}
		}
	}
	return payload, nil
}

func createPelicanOpenAIPayload(modelID string, isOAuth bool, prompt, reasoningEffort string) map[string]any {
	payload := createOpenAITestPayload(modelID, isOAuth)
	input, ok := payload["input"].([]map[string]any)
	if ok && len(input) > 0 {
		content, ok := input[0]["content"].([]map[string]any)
		if ok && len(content) > 0 {
			content[0]["text"] = promptOrDefault(prompt)
		}
	}
	if effort := normalizePelicanReasoningEffort(reasoningEffort); effort != "" {
		payload["reasoning"] = map[string]any{"effort": effort}
	}
	if !isOAuth {
		payload["max_output_tokens"] = pelicanMaxOutputTokens
	}
	return payload
}

func createAccountClaudeTestPayload(ctx context.Context, modelID string) (map[string]any, error) {
	if options, ok := pelicanTestOptionsFromContext(ctx); ok {
		return createPelicanClaudePayload(modelID, options.prompt, options.reasoningEffort)
	}
	return createTestPayload(modelID)
}

func createPelicanGeminiPayload(modelID, prompt, effort string) []byte {
	var payload map[string]any
	_ = json.Unmarshal(createGeminiTestPayload(modelID, prompt), &payload)
	config := map[string]any{"maxOutputTokens": pelicanMaxOutputTokens}
	if effort != "" {
		if strings.Contains(strings.ToLower(modelID), "gemini-3") {
			config["thinkingConfig"] = map[string]any{"thinkingLevel": effort}
		} else {
			budget := map[string]int{"none": 0, "minimal": 512, "low": 1024, "medium": 4096, "high": 8192}[effort]
			config["thinkingConfig"] = map[string]any{"thinkingBudget": budget}
		}
	}
	payload["generationConfig"] = config
	body, _ := json.Marshal(payload)
	return body
}

func (s *AntigravityGatewayService) buildPelicanAntigravityRequest(projectID, model string, options pelicanTestOptions) ([]byte, error) {
	if strings.HasPrefix(model, "gemini-") {
		var payload map[string]any
		_ = json.Unmarshal(createPelicanGeminiPayload(model, options.prompt, options.reasoningEffort), &payload)
		payload["systemInstruction"] = map[string]any{"parts": []map[string]any{{"text": antigravity.GetDefaultIdentityPatch()}}}
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return s.wrapV1InternalRequest(projectID, model, body)
	}
	content, _ := json.Marshal(options.prompt)
	req := &antigravity.ClaudeRequest{Model: model, Messages: []antigravity.ClaudeMessage{{Role: "user", Content: content}}, MaxTokens: pelicanMaxOutputTokens}
	if options.reasoningEffort != "" {
		req.Thinking = &antigravity.ThinkingConfig{Type: "enabled", BudgetTokens: map[string]int{"low": 1024, "medium": 4096, "high": 8192}[options.reasoningEffort]}
	}
	return antigravity.TransformClaudeToGemini(req, projectID, model)
}

func (s *AccountTestService) acquirePelicanTest(accountID int64) bool {
	s.pelicanMu.Lock()
	defer s.pelicanMu.Unlock()
	if s.pelicanActive == nil {
		s.pelicanActive = make(map[int64]int)
	}
	if s.pelicanTotal >= 32 || s.pelicanActive[accountID] >= 8 {
		return false
	}
	s.pelicanActive[accountID]++
	s.pelicanTotal++
	return true
}

func (s *AccountTestService) releasePelicanTest(accountID int64) {
	s.pelicanMu.Lock()
	defer s.pelicanMu.Unlock()
	s.pelicanTotal--
	s.pelicanActive[accountID]--
	if s.pelicanActive[accountID] == 0 {
		delete(s.pelicanActive, accountID)
	}
}

func promptOrDefault(prompt string) string {
	if value := strings.TrimSpace(prompt); value != "" {
		return value
	}
	return "hi"
}

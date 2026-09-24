//go:build unit

package service

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestPelicanForwardingUsesSelectedSettings(t *testing.T) {
	for _, tc := range []struct{ name, platform, protocol, model, effort, response, promptPath, effortPath string }{
		{"responses", PlatformOpenAI, "responses", "gpt-6-astra", "max", "data: {\"type\":\"response.completed\"}\n\n", "input.0.content.0.text", "reasoning.effort"},
		{"chat", PlatformDeepseek, "chat_completions", "deepseek-v4-flash", "high", "data: [DONE]\n\n", "messages.0.content", "reasoning_effort"},
		{"adaptive single request", PlatformDeepseek, "adaptive", "deepseek-v4-flash", "high", "data: [DONE]\n\n", "messages.0.content", "reasoning_effort"},
		{"anthropic", PlatformAnthropic, "anthropic", "claude-opus-4-6", "max", "data: {\"type\":\"message_stop\"}\n\n", "messages.0.content.0.text", "output_config.effort"},
		{"gemini", PlatformGemini, "", "gemini-3-flash-preview", "high", "data: [DONE]\n\n", "contents.0.parts.0.text", "generationConfig.thinkingConfig.thinkingLevel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newTestContext()
			account := &Account{ID: 1, Platform: tc.platform, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test-key", "api_protocol": tc.protocol, "base_url": "https://api.example.com", "model_mapping": map[string]any{tc.model: tc.model}}}
			repo := &mockAccountRepoForGemini{accountsByID: map[int64]*Account{1: account}}
			upstream := &queuedHTTPUpstream{responses: []*http.Response{newJSONResponse(200, tc.response)}}
			svc := &AccountTestService{accountRepo: repo, httpUpstream: upstream, cfg: &config.Config{
				Security: config.SecurityConfig{
					URLAllowlist: config.URLAllowlistConfig{Enabled: false},
				},
			}}
			require.NoError(t, svc.TestPelicanAccountConnection(c, 1, tc.model, "draw a pelican HTML", tc.effort), rec.Body.String())
			require.Len(t, upstream.requests, 1)
			body, err := io.ReadAll(upstream.requests[0].Body)
			require.NoError(t, err)
			require.Equal(t, "draw a pelican HTML", gjson.GetBytes(body, tc.promptPath).String())
			require.Equal(t, tc.effort, gjson.GetBytes(body, tc.effortPath).String())
			require.Equal(t, 0, svc.pelicanTotal)
		})
	}
}

func TestPelicanRejectsUnsupportedEffortBeforeUpstream(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	svc := &AccountTestService{accountRepo: &mockAccountRepoForGemini{accountsByID: map[int64]*Account{1: account}}}
	for _, effort := range []string{"max", "ultra", "garbage"} {
		c, rec := newTestContext()
		require.Error(t, svc.TestPelicanAccountConnection(c, 1, "gpt-5.4", "draw HTML", effort))
		require.Contains(t, strings.ToLower(rec.Body.String()), "reasoning effort")
		require.Zero(t, svc.pelicanTotal)
	}
}

func TestPelicanPromptLimit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prompt string
		valid  bool
	}{
		{"empty", "", false},
		{"whitespace", " \n\t", false},
		{"ascii at limit", strings.Repeat("x", 16000), true},
		{"ascii above limit", strings.Repeat("x", 16001), false},
		{"unicode at limit", strings.Repeat("鹈", 16000), true},
		{"unicode above limit", strings.Repeat("鹈", 16001), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test-key", "api_protocol": "responses"}}
			upstream := &queuedHTTPUpstream{responses: []*http.Response{newJSONResponse(200, "data: {\"type\":\"response.completed\"}\n\n")}}
			svc := &AccountTestService{accountRepo: &mockAccountRepoForGemini{accountsByID: map[int64]*Account{1: account}}, httpUpstream: upstream, cfg: &config.Config{}}
			c, rec := newTestContext()
			err := svc.TestPelicanAccountConnection(c, 1, "gpt-6-astra", tc.prompt, "")
			if tc.valid {
				require.NoError(t, err, rec.Body.String())
				require.Len(t, upstream.requests, 1)
				body, readErr := io.ReadAll(upstream.requests[0].Body)
				require.NoError(t, readErr)
				require.Equal(t, tc.prompt, gjson.GetBytes(body, "input.0.content.0.text").String())
			} else {
				require.Error(t, err)
				require.Contains(t, rec.Body.String(), "16000")
				require.Empty(t, upstream.requests)
			}
			require.Zero(t, svc.pelicanTotal)
		})
	}
}

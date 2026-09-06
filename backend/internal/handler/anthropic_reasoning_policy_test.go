package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAnthropicReasoningPolicyResolvedPlatform(t *testing.T) {
	for _, platform := range []string{service.PlatformAnthropic, service.PlatformOpenAI, service.PlatformGemini} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
		c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(c.Request.Context(), platform))
		group := &service.Group{Platform: service.PlatformComposite, MaxReasoningEffort: "minimal", ReasoningEffortMappings: []service.ReasoningEffortMapping{{From: "max", To: "xhigh"}, {From: "xhigh", To: "high"}}}
		key := &service.APIKey{Group: group}
		body := []byte(`{"output_config":{"effort":"max"}}`)
		result, changed := applyOpenAIReasoningEffortPolicyForRequest(c, key, body)
		switch platform {
		case service.PlatformAnthropic:
			require.True(t, changed)
			require.Equal(t, "low", gjson.GetBytes(result, "output_config.effort").String())
		case service.PlatformOpenAI:
			require.True(t, changed)
			require.Equal(t, "minimal", gjson.GetBytes(result, "output_config.effort").String())
		default:
			require.False(t, changed)
		}
		require.Equal(t, "minimal", group.MaxReasoningEffort)
		require.Equal(t, "xhigh", group.ReasoningEffortMappings[0].To)
	}
}

func TestAnthropicReasoningPolicyMapsOnceAndLeavesOmitted(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	key := &service.APIKey{Group: &service.Group{Platform: service.PlatformAnthropic, ReasoningEffortMappings: []service.ReasoningEffortMapping{{From: "max", To: "xhigh"}, {From: "xhigh", To: "high"}}}}
	body, changed := applyOpenAIReasoningEffortPolicyForRequest(c, key, []byte(`{"output_config":{"effort":"max"}}`))
	require.True(t, changed)
	require.Equal(t, "xhigh", gjson.GetBytes(body, "output_config.effort").String())
	bindOpenAIReasoningEffortPolicyForMessagesRequest(c, key, body)
	_, changed = service.ApplyOpenAIReasoningEffortPolicyFromContext(c.Request.Context(), body)
	require.False(t, changed)
	omitted := []byte(`{"model":"claude-fable-5-1"}`)
	body, changed = applyOpenAIReasoningEffortPolicyForRequest(c, key, omitted)
	require.False(t, changed)
	require.Equal(t, omitted, body)
}

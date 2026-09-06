package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAnthropicReasoningForwardRecordsEffectiveEffort(t *testing.T) {
	for _, ceiling := range []string{"", "high"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		body, _ := ApplyOpenAIReasoningEffortPolicy([]byte(`{"model":"claude-fable-5-1","max_tokens":64,"messages":[{"role":"user","content":"hi"}],"output_config":{"effort":"max"}}`), ceiling, nil)
		parsed, err := ParseGatewayRequest(NewRequestBodyRef(body), PlatformAnthropic)
		require.NoError(t, err)
		upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
			StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"type":"message","model":"claude-fable-5-1","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":10,"output_tokens":2}}`)),
		}}
		cfg := &config.Config{}
		gateway := &GatewayService{cfg: cfg, httpUpstream: upstream, responseHeaderFilter: compileResponseHeaderFilter(cfg), rateLimitService: &RateLimitService{}, deferredService: &DeferredService{}}
		result, err := gateway.Forward(context.Background(), c, newAnthropicAPIKeyAccountForTest(), parsed)
		require.NoError(t, err)
		require.NotNil(t, result.ReasoningEffort)
		want := "max"
		if ceiling != "" {
			want = ceiling
		}
		require.Equal(t, want, *result.ReasoningEffort)
		require.Equal(t, want, gjson.GetBytes(upstream.lastBody, "output_config.effort").String())
	}
}

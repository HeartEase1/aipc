package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAIPCWebSocketDenialsWithoutAllowlist(t *testing.T) {
	for _, mode := range []string{service.OpenAIWSIngressModePassthrough, service.OpenAIWSIngressModeDedicated} {
		t.Run(mode+"/first", func(t *testing.T) {
			group := wsAllowlistGroup(false)
			group.ModelAllowlist.BlockedModels = []string{"gpt-4.1"}
			runOpenAIResponsesWebSocketUsageLogCase(t, openAIResponsesWSUsageLogCase{
				firstPayload: `{"type":"response.create","model":"gpt-4.1","stream":false}`,
				group:        group, ingressMode: mode, firstFrameCloseExpected: true,
			})
		})
		t.Run(mode+"/subsequent", func(t *testing.T) {
			group := wsAllowlistGroup(false)
			group.ModelAllowlist.BlockedModels = []string{"gpt-4.1"}
			runOpenAIResponsesWebSocketUsageLogCase(t, openAIResponsesWSUsageLogCase{
				firstPayload:  `{"type":"response.create","model":"gpt-5.4","stream":false}`,
				secondPayload: `{"type":"response.create","model":"gpt-4.1","stream":false}`,
				group:         group, ingressMode: mode, secondTurnCloseExpected: true,
			})
		})
	}
}

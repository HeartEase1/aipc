package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesArgumentsDeltaDoesNotOverwriteToolName(t *testing.T) {
	state := NewResponsesEventToChatState()
	first := ResponsesEventToChatChunks(&ResponsesStreamEvent{
		Type:        "response.output_item.added",
		OutputIndex: 0,
		Item: &ResponsesOutput{
			Type:   "function_call",
			CallID: "call_1",
			Name:   "search",
		},
	}, state)
	require.Len(t, first, 1)

	delta := ResponsesEventToChatChunks(&ResponsesStreamEvent{
		Type:        "response.function_call_arguments.delta",
		OutputIndex: 0,
		Delta:       `{"query":"test"}`,
	}, state)
	require.Len(t, delta, 1)

	raw, err := json.Marshal(delta[0])
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"name"`)
	require.Contains(t, string(raw), `"arguments"`)
}

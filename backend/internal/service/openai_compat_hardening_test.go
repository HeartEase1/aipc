package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEmptyOpenAICapabilitiesBehaveAsUnconfigured(t *testing.T) {
	for name, capabilities := range map[string]any{
		"object": map[string]any{},
		"slice":  []any{},
	} {
		t.Run(name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					openAIEndpointCapabilitiesCredentialKey: capabilities,
				},
			}
			require.True(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponses))
		})
	}

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			openAIEndpointCapabilitiesCredentialKey: map[string]any{
				string(OpenAIEndpointCapabilityResponses): false,
			},
		},
	}
	require.False(t, account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponses), "an explicit false value must still restrict scheduling")
}

func TestStripEmptyChatToolCallIdentityKeepsArgumentDeltas(t *testing.T) {
	payload := []byte(`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"","type":"function","function":{"name":"","arguments":"{\"q\":"}}]}}]}`)
	rewritten, changed := stripEmptyChatToolCallIdentity(payload)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(rewritten, "choices.0.delta.tool_calls.0.id").Exists())
	require.False(t, gjson.GetBytes(rewritten, "choices.0.delta.tool_calls.0.function.name").Exists())
	require.Equal(t, `{"q":`, gjson.GetBytes(rewritten, "choices.0.delta.tool_calls.0.function.arguments").String())
	require.Equal(t, "function", gjson.GetBytes(rewritten, "choices.0.delta.tool_calls.0.type").String())
}

func TestRawChatToolCallIdentityNormalizationSkipsGrok(t *testing.T) {
	line := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"","function":{"name":"","arguments":"{}"}}]}}]}`

	grok := &Account{Platform: PlatformGrok}
	require.Equal(t, line, sanitizeRawChatToolCallIdentityForAccount(grok, line))

	openAI := &Account{Platform: PlatformOpenAI}
	normalized := sanitizeRawChatToolCallIdentityForAccount(openAI, line)
	require.NotEqual(t, line, normalized)
	require.NotContains(t, normalized, `"id":""`)
	require.NotContains(t, normalized, `"name":""`)
}

func TestStreamingTerminalOutputUsesReportedDoneItems(t *testing.T) {
	doneItems := newResponsesStreamOutputItems()
	doneItems.Observe([]byte(`{"type":"response.output_item.done","output_index":1,"item":{"id":"msg_real","type":"message","status":"completed","phase":"final_answer","role":"assistant","content":[{"type":"output_text","text":"done"}]}}`))
	doneItems.Observe([]byte(`{"type":"response.output_item.done","output_index":0,"item":{"id":"rs_real","type":"reasoning","encrypted_content":"opaque"}}`))

	terminal := []byte(`{"type":"response.completed","response":{"status":"completed","output":[]}}`)
	rewritten, changed := normalizeResponsesStreamingTerminalOutput(terminal, nil, doneItems, nil)
	require.True(t, changed)
	require.Equal(t, "rs_real", gjson.GetBytes(rewritten, "response.output.0.id").String())
	require.Equal(t, "opaque", gjson.GetBytes(rewritten, "response.output.0.encrypted_content").String())
	require.Equal(t, "msg_real", gjson.GetBytes(rewritten, "response.output.1.id").String())
	require.Equal(t, "final_answer", gjson.GetBytes(rewritten, "response.output.1.phase").String())
}

func TestStreamingTerminalOutputUsesImageDoneItemWithoutSeparateCopy(t *testing.T) {
	doneItems := newResponsesStreamOutputItems()
	doneItems.Observe([]byte(`{"type":"response.output_item.done","output_index":0,"item":{"id":"img_1","type":"image_generation_call","status":"completed","result":"base64-image-data"}}`))

	terminal := []byte(`{"type":"response.completed","response":{"status":"completed","output":[]}}`)
	rewritten, changed := normalizeResponsesStreamingTerminalOutput(terminal, nil, doneItems, nil)
	require.True(t, changed)
	require.Len(t, gjson.GetBytes(rewritten, "response.output").Array(), 1)
	require.Equal(t, "image_generation_call", gjson.GetBytes(rewritten, "response.output.0.type").String())
	require.Equal(t, "base64-image-data", gjson.GetBytes(rewritten, "response.output.0.result").String())
}

func TestRejectedStatusRetryClearsAllItemsOfTheSameType(t *testing.T) {
	body := []byte(`{"input":[
		{"type":"tool_search_output","status":"completed","call_id":"a"},
		{"type":"message","status":"completed","role":"user","content":"hi"},
		{"type":"tool_search_output","status":"completed","call_id":"b"}
	]}`)
	upstreamError := []byte(`{"error":{"code":"unknown_parameter","message":"Unknown parameter: 'input[2].status'.","param":"input[2].status"}}`)

	retryBody, _, changed, err := normalizeOpenAIResponsesRejectedFieldRetryBody(http.StatusBadRequest, body, upstreamError)
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(retryBody, "input.0.status").Exists())
	require.Equal(t, "completed", gjson.GetBytes(retryBody, "input.1.status").String())
	require.False(t, gjson.GetBytes(retryBody, "input.2.status").Exists())
	require.Equal(t, "b", gjson.GetBytes(retryBody, "input.2.call_id").String())
}

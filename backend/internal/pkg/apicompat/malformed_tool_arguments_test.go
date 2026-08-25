package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesToChatDropsMalformedFunctionCallPair(t *testing.T) {
	req := &ResponsesRequest{
		Model: "gpt-5.5",
		Input: json.RawMessage(`[
			{"type":"function_call","call_id":"bad","name":"lookup","arguments":"{\"q\":"},
			{"type":"function_call_output","call_id":"bad","output":"ignored"},
			{"type":"function_call","call_id":"good","name":"lookup","arguments":"{\"q\":\"ok\"}"},
			{"type":"function_call_output","call_id":"good","output":"result"},
			{"role":"user","content":"continue"}
		]`),
	}

	chat, err := ResponsesToChatCompletionsRequest(req)
	require.NoError(t, err)
	require.Len(t, chat.Messages, 3)
	require.Len(t, chat.Messages[0].ToolCalls, 1)
	require.Equal(t, "good", chat.Messages[0].ToolCalls[0].ID)
	require.Equal(t, "good", chat.Messages[1].ToolCallID)
	require.Equal(t, "user", chat.Messages[2].Role)
}

func TestChatResponseDoesNotPersistMalformedFunctionArguments(t *testing.T) {
	resp := &ChatCompletionsResponse{
		ID: "chat_1",
		Choices: []ChatChoice{{
			Message: ChatMessage{
				Role: "assistant",
				ToolCalls: []ChatToolCall{{
					ID: "bad",
					Function: ChatFunctionCall{
						Name:      "lookup",
						Arguments: `{"q":`,
					},
				}},
			},
		}},
	}

	converted := ChatCompletionsResponseToResponses(resp, "gpt-5.5", nil, false, nil)
	for _, output := range converted.Output {
		require.NotEqual(t, "function_call", output.Type)
	}
}

func TestStreamToolArgumentValidationOnlyAppliesToFunctions(t *testing.T) {
	ordinary := NewChatCompletionsToResponsesStreamState("gpt-5.5")
	ordinary.ToolCalls[0] = &ChatToolCall{ID: "bad", Function: ChatFunctionCall{Name: "lookup", Arguments: `{"q":`}}
	require.Error(t, ordinary.ValidateToolCallArguments())

	custom := NewChatCompletionsToResponsesStreamState("gpt-5.5")
	custom.ToolCalls[0] = &ChatToolCall{ID: "custom", Function: ChatFunctionCall{Name: "shell", Arguments: "free text"}}
	custom.toolIsCustom[0] = true
	require.NoError(t, custom.ValidateToolCallArguments())
}

//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCodexDelegationWithHistoryPreservesCallPair(t *testing.T) {
	delegation := map[string]any{
		"type": "function_call_output", "namespace": "codex_app", "name": "send_message_to_thread",
		"output": "<codex_delegation><source_thread_id>source</source_thread_id><input>Continue</input></codex_delegation>",
	}
	request := map[string]any{
		"previous_response_id": "resp_previous",
		"input": []any{
			map[string]any{"type": "function_call", "call_id": "call_history", "name": "shell", "arguments": "{}"},
			map[string]any{"type": "function_call_output", "call_id": "call_history", "output": "done"},
			delegation,
		},
	}
	body, err := json.Marshal(request)
	require.NoError(t, err)
	got, changed := normalizeCodexDelegationBootstrap(body)
	require.True(t, changed)
	require.Equal(t, "resp_previous", gjson.GetBytes(got, "previous_response_id").String())
	require.Equal(t, "call_history", gjson.GetBytes(got, "input.0.call_id").String())
	require.Equal(t, "call_history", gjson.GetBytes(got, "input.1.call_id").String())
	require.Equal(t, "user", gjson.GetBytes(got, "input.2.role").String())
	require.Equal(t, delegation["output"], gjson.GetBytes(got, "input.2.content.0.text").String())
	delegation["call_id"] = "real_tool_call"
	body, err = json.Marshal(request)
	require.NoError(t, err)
	got, changed = normalizeCodexDelegationBootstrap(body)
	require.False(t, changed, "a real tool result must retain its call association")
	require.Equal(t, body, got)
}

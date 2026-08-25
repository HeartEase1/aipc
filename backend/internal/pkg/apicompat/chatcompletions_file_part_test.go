package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatCompletionsToResponsesPreservesFileParts(t *testing.T) {
	req := &ChatCompletionsRequest{
		Model: "gpt-5.5",
		Messages: []ChatMessage{{
			Role: "user",
			Content: json.RawMessage(`[
				{"type":"text","text":"summarize this"},
				{"type":"file","file":{"filename":"report.pdf","file_data":"data:application/pdf;base64,JVBERg=="}},
				{"type":"file","file":{"file_id":"file_123"}},
				{"type":"file","file":{"filename":"empty.pdf"}}
			]`),
		}},
	}

	converted, err := ChatCompletionsToResponses(req)
	require.NoError(t, err)

	var input []ResponsesInputItem
	require.NoError(t, json.Unmarshal(converted.Input, &input))
	require.Len(t, input, 1)

	var parts []ResponsesContentPart
	require.NoError(t, json.Unmarshal(input[0].Content, &parts))
	require.Len(t, parts, 3, "empty file parts must not produce invalid upstream input")
	require.Equal(t, "input_text", parts[0].Type)
	require.Equal(t, "input_file", parts[1].Type)
	require.Equal(t, "report.pdf", parts[1].Filename)
	require.Equal(t, "data:application/pdf;base64,JVBERg==", parts[1].FileData)
	require.Equal(t, "input_file", parts[2].Type)
	require.Equal(t, "file_123", parts[2].FileID)
}

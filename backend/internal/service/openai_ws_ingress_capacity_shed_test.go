package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIWSIngressMessageForClient(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		payload   string
		wantCode  string
	}{
		{
			name:      "capacity error becomes retryable",
			eventType: "error",
			payload:   `{"type":"error","error":{"code":"server_is_overloaded","message":"busy"}}`,
			wantCode:  `"code":"server_error"`,
		},
		{
			name:      "capacity response failure becomes retryable",
			eventType: "response.failed",
			payload:   `{"type":"response.failed","response":{"error":{"code":"slow_down","message":"busy"}}}`,
			wantCode:  `"code":"server_error"`,
		},
		{
			name:      "non-capacity error is unchanged",
			eventType: "error",
			payload:   `{"type":"error","error":{"code":"workspace_suspended","message":"suspended"}}`,
			wantCode:  `"code":"workspace_suspended"`,
		},
		{
			name:      "non-error event is unchanged",
			eventType: "response.output_text.delta",
			payload:   `{"type":"response.output_text.delta","error":{"code":"server_is_overloaded"}}`,
			wantCode:  `"code":"server_is_overloaded"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := []byte(tt.payload)
			got := openAIWSIngressMessageForClient(tt.eventType, original)
			require.Contains(t, string(got), tt.wantCode)
			require.Equal(t, tt.payload, string(original), "the upstream payload must remain unchanged")
		})
	}
}

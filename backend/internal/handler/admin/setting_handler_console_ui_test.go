package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestConsoleUIModeSettingsPreserveValidateAndSave(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   map[string]any
		status int
		want   string
	}{
		{"omitted", map[string]any{"site_name": "AIPC"}, http.StatusOK, "modern"},
		{"switch", map[string]any{"console_ui_mode": "legacy"}, http.StatusOK, "legacy"},
		{"invalid", map[string]any{"console_ui_mode": "custom"}, http.StatusBadRequest, "modern"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newStepUpSwitchTestHandler(t, map[string]string{service.SettingKeyConsoleUIMode: "modern"})
			rec := doUpdateSettings(t, h, tc.body, nil)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.Equal(t, tc.want, repo.values[service.SettingKeyConsoleUIMode])
			if tc.status == http.StatusOK {
				require.Contains(t, rec.Body.String(), `"console_ui_mode":"`+tc.want+`"`)
			}
		})
	}
}

func TestConsoleUIModeAuditRecordsOnlyChanges(t *testing.T) {
	before := &service.SystemSettings{ConsoleUIMode: "legacy"}
	after := &service.SystemSettings{ConsoleUIMode: "modern"}
	require.Contains(t, diffSettings(before, after, nil, nil, UpdateSettingsRequest{}), service.SettingKeyConsoleUIMode)
	require.NotContains(t, diffSettings(after, after, nil, nil, UpdateSettingsRequest{}), service.SettingKeyConsoleUIMode)
}

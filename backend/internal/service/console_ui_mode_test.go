//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeConsoleUIMode(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "missing falls back legacy", raw: "", want: ConsoleUIModeLegacy},
		{name: "modern", raw: ConsoleUIModeModern, want: ConsoleUIModeModern},
		{name: "modern normalized", raw: " MODERN ", want: ConsoleUIModeModern},
		{name: "legacy", raw: ConsoleUIModeLegacy, want: ConsoleUIModeLegacy},
		{name: "invalid falls back legacy", raw: "classic", want: ConsoleUIModeLegacy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, NormalizeConsoleUIMode(tt.raw))
		})
	}
}

func TestParseConsoleUIMode(t *testing.T) {
	mode, ok := ParseConsoleUIMode(" MODERN ")
	require.True(t, ok)
	require.Equal(t, ConsoleUIModeModern, mode)

	mode, ok = ParseConsoleUIMode("unexpected")
	require.False(t, ok)
	require.Empty(t, mode)
}

package service

import "strings"

const (
	ConsoleUIModeLegacy = "legacy"
	ConsoleUIModeModern = "modern"
)

// ParseConsoleUIMode normalizes a value supplied by an administrator while
// preserving whether it was valid. Stored legacy data uses Normalize below,
// whereas write APIs reject invalid enum values instead of silently changing
// the active interface.
func ParseConsoleUIMode(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ConsoleUIModeModern:
		return ConsoleUIModeModern, true
	case ConsoleUIModeLegacy:
		return ConsoleUIModeLegacy, true
	default:
		return "", false
	}
}

// NormalizeConsoleUIMode fails closed to the proven legacy console when the
// setting is absent or malformed. Fresh installations persist modern during
// default-settings initialization.
func NormalizeConsoleUIMode(raw string) string {
	if mode, ok := ParseConsoleUIMode(raw); ok {
		return mode
	}
	return ConsoleUIModeLegacy
}

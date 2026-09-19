package service

import "strings"

const (
	ConsoleUIModeLegacy = "legacy"
	ConsoleUIModeModern = "modern"
)

func ParseConsoleUIMode(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ConsoleUIModeLegacy:
		return ConsoleUIModeLegacy, true
	case ConsoleUIModeModern:
		return ConsoleUIModeModern, true
	default:
		return "", false
	}
}

func NormalizeConsoleUIMode(raw string) string {
	if mode, ok := ParseConsoleUIMode(raw); ok {
		return mode
	}
	return ConsoleUIModeLegacy
}

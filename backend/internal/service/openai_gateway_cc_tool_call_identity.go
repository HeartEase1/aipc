package service

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// sanitizeRawChatToolCallIdentityForAccount keeps Grok's raw stream untouched
// while applying OpenAI-compatible tool-delta normalization elsewhere.
func sanitizeRawChatToolCallIdentityForAccount(account *Account, line string) string {
	if account == nil || account.Platform == PlatformGrok {
		return line
	}
	return stripEmptyChatToolCallIdentityFromSSELine(line)
}

// stripEmptyChatToolCallIdentityFromSSELine removes empty identity fields from
// argument-only tool-call deltas. Missing fields preserve the valid id and name
// that clients accumulated from the first delta; empty strings can overwrite it.
func stripEmptyChatToolCallIdentityFromSSELine(line string) string {
	payload, ok := extractOpenAISSEDataLine(line)
	if !ok || strings.TrimSpace(payload) == "" || strings.TrimSpace(payload) == "[DONE]" {
		return line
	}
	rewritten, changed := stripEmptyChatToolCallIdentity([]byte(payload))
	if !changed {
		return line
	}
	prefixLength := len(line) - len(payload)
	if prefixLength < 0 {
		return line
	}
	return line[:prefixLength] + string(rewritten)
}

func stripEmptyChatToolCallIdentity(payload []byte) ([]byte, bool) {
	if len(payload) == 0 || !bytes.Contains(payload, []byte("tool_calls")) || !gjson.ValidBytes(payload) {
		return payload, false
	}
	choices := gjson.GetBytes(payload, "choices")
	if !choices.IsArray() {
		return payload, false
	}

	updated := payload
	changed := false
	for choiceIndex, choice := range choices.Array() {
		toolCalls := choice.Get("delta.tool_calls")
		if !toolCalls.IsArray() {
			continue
		}
		for toolIndex, toolCall := range toolCalls.Array() {
			basePath := "choices." + strconv.Itoa(choiceIndex) + ".delta.tool_calls." + strconv.Itoa(toolIndex)
			if id := toolCall.Get("id"); id.Exists() && id.Type == gjson.String && id.String() == "" {
				next, err := sjson.DeleteBytes(updated, basePath+".id")
				if err != nil {
					return payload, false
				}
				updated = next
				changed = true
			}
			if name := toolCall.Get("function.name"); name.Exists() && name.Type == gjson.String && name.String() == "" {
				next, err := sjson.DeleteBytes(updated, basePath+".function.name")
				if err != nil {
					return payload, false
				}
				updated = next
				changed = true
			}
		}
	}
	return updated, changed
}

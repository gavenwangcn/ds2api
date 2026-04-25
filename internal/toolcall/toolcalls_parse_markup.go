package toolcall

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
)

var xmlToolsWrapperPattern = regexp.MustCompile(`(?is)<tools\b[^>]*>\s*(.*?)\s*</tools>`)
var xmlToolCallPattern = regexp.MustCompile(`(?is)<tool_call\b[^>]*>\s*(.*?)\s*</tool_call>`)
var xmlCanonicalToolCallBodyPattern = regexp.MustCompile(`(?is)^\s*<(?:[a-z0-9_:-]+:)?tool_name\b[^>]*>(.*?)</(?:[a-z0-9_:-]+:)?tool_name>\s*<(?:[a-z0-9_:-]+:)?param\b[^>]*>(.*?)</(?:[a-z0-9_:-]+:)?param>\s*$`)

func parseXMLToolCalls(text string) []ParsedToolCall {
	wrappers := xmlToolsWrapperPattern.FindAllStringSubmatch(text, -1)
	if len(wrappers) == 0 {
		return nil
	}
	out := make([]ParsedToolCall, 0, len(wrappers))
	for _, wrapper := range wrappers {
		if len(wrapper) < 2 {
			continue
		}
		for _, block := range xmlToolCallPattern.FindAllString(wrapper[1], -1) {
			call, ok := parseSingleXMLToolCall(block)
			if !ok {
				continue
			}
			out = append(out, call)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseSingleXMLToolCall(block string) (ParsedToolCall, bool) {
	inner := strings.TrimSpace(block)
	inner = strings.TrimPrefix(inner, "<tool_call>")
	inner = strings.TrimSuffix(inner, "</tool_call>")
	inner = strings.TrimSpace(inner)
	if strings.HasPrefix(inner, "{") {
		var payload map[string]any
		if err := json.Unmarshal([]byte(inner), &payload); err == nil {
			name := strings.TrimSpace(asString(payload["name"]))
			if name != "" {
				input := map[string]any{}
				if params, ok := payload["input"].(map[string]any); ok {
					input = params
				}
				return ParsedToolCall{Name: name, Input: input}, true
			}
		}
	}

	m := xmlCanonicalToolCallBodyPattern.FindStringSubmatch(inner)
	if len(m) < 3 {
		return ParsedToolCall{}, false
	}
	name := strings.TrimSpace(html.UnescapeString(extractRawTagValue(m[1])))
	if strings.TrimSpace(name) == "" {
		return ParsedToolCall{}, false
	}
	return ParsedToolCall{Name: name, Input: parseStructuredToolCallInput(m[2])}, true
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

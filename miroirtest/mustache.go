package miroirtest

import (
	"fmt"
	"regexp"
)

var doubleBracePattern = regexp.MustCompile(`{{\s*([^}]+?)\s*}}`)

func extractDoubleBracePatterns(args []any) (any, error) {
	input, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("extractDoubleBracePatterns: expected string argument")
	}
	matches := doubleBracePattern.FindAllStringSubmatchIndex(input, -1)
	out := make([]any, 0, len(matches))
	for _, loc := range matches {
		content := input[loc[2]:loc[3]]
		// TS: match[1].trim()
		trimmed := trimSpace(content)
		if trimmed == "" {
			return nil, fmt.Errorf("Empty pattern found")
		}
		out = append(out, map[string]any{
			"content": trimmed,
			"start":   loc[0],
			"end":     loc[1] - 1,
		})
	}
	return out, nil
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

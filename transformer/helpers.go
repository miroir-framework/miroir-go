package transformer

import (
	"encoding/json"
	"fmt"
	"sort"
)

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func schemaType(schema any) string {
	m, ok := asMap(schema)
	if !ok || m == nil {
		return ""
	}
	if t, ok := m["type"]; ok && t != nil {
		return fmt.Sprint(t)
	}
	return ""
}

func isTypedTransformer(v any) bool {
	m, ok := asMap(v)
	if !ok || m == nil {
		return false
	}
	_, ok = m["transformerType"].(string)
	return ok
}

func transformerType(v any) string {
	m, ok := asMap(v)
	if !ok {
		return ""
	}
	s, _ := m["transformerType"].(string)
	return s
}

func isFailed(v any) bool {
	m, ok := asMap(v)
	if !ok || m == nil {
		return false
	}
	if m["status"] != "error" {
		return false
	}
	_, has := m["failureKind"]
	return has
}

func jsonEqual(a, b any) bool {
	ab, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(ab) == string(bb)
}

func fail(kind, err string, extras map[string]any) map[string]any {
	out := map[string]any{
		"status":      "error",
		"failureKind": kind,
		"error":       err,
		"typePath":    []any{},
	}
	for k, v := range extras {
		if skipEmptyExtra(k, v) {
			continue
		}
		out[k] = v
	}
	return out
}

func skipEmptyExtra(k string, v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok && s == "" {
		return true
	}
	if list, ok := v.([]any); ok && len(list) == 0 && k != "typePath" {
		return true
	}
	return false
}

func wrapOperandFailure(result any, parent string, key any) any {
	m, ok := asMap(result)
	if !ok {
		return result
	}
	out := copyMap(m)
	out["transformerPath"] = []any{parent, key}
	out["innerError"] = result
	return out
}

func stringSlice(v any) []string {
	switch list := v.(type) {
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case []string:
		return list
	default:
		return nil
	}
}

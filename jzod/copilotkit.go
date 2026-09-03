package jzod

import "sort"

// JzodToCopilotKitParameter converts a Jzod element to a CopilotKit parameter
// descriptor (name plus JSON-Schema-like fields).
func JzodToCopilotKitParameter(name string, element any, context map[string]any) map[string]any {
	if element == nil {
		return map[string]any{"name": name}
	}
	el, ok := element.(map[string]any)
	if !ok {
		return map[string]any{"name": name}
	}
	if context == nil {
		context = map[string]any{}
	}
	base := map[string]any{"name": name}
	if d, ok := el["description"].(string); ok && d != "" {
		base["description"] = d
	}
	if optional, _ := el["optional"].(bool); optional {
		base["required"] = false
	}
	withType := func(t string) map[string]any {
		out := copyMap(base)
		out["type"] = t
		return out
	}
	switch el["type"] {
	case "string":
		return withType("string")
	case "number":
		return withType("number")
	case "boolean":
		return withType("boolean")
	case "uuid", "bigint", "date":
		return withType("string")
	case "any", "unknown", "never", "undefined", "void":
		return copyMap(base)
	case "literal":
		out := withType("string")
		out["enum"] = []any{el["definition"]}
		return out
	case "enum":
		out := withType("string")
		out["enum"] = el["definition"]
		return out
	case "array":
		inner, _ := el["definition"].(map[string]any)
		innerType, _ := inner["type"].(string)
		switch innerType {
		case "string", "uuid", "bigint", "date":
			return withType("string[]")
		case "number":
			return withType("number[]")
		case "boolean":
			return withType("boolean[]")
		case "object":
			out := withType("object[]")
			def, _ := inner["definition"].(map[string]any)
			attrs := objectAttributes(def, context)
			out["attributes"] = attrs
			return out
		default:
			return withType("object[]")
		}
	case "object":
		out := withType("object")
		def, _ := el["definition"].(map[string]any)
		out["attributes"] = objectAttributes(def, context)
		return out
	case "record":
		return withType("object")
	case "union":
		list, _ := el["definition"].([]any)
		allLiterals := true
		enums := []any{}
		for _, d := range list {
			m, _ := d.(map[string]any)
			if m["type"] != "literal" {
				allLiterals = false
				break
			}
			enums = append(enums, m["definition"])
		}
		out := withType("string")
		if allLiterals {
			out["enum"] = enums
		}
		return out
	case "intersection":
		return withType("object")
	case "schemaReference":
		def, _ := el["definition"].(map[string]any)
		merged := mergeContext(context, contextOf(el))
		refPath, _ := def["relativePath"].(string)
		if resolved, ok := merged[refPath]; ok {
			return JzodToCopilotKitParameter(name, resolved, merged)
		}
		return copyMap(base)
	case "lazy":
		return JzodToCopilotKitParameter(name, el["definition"], context)
	default:
		return copyMap(base)
	}
}

func objectAttributes(def map[string]any, context map[string]any) []any {
	keys := make([]string, 0, len(def))
	for k := range def {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	attrs := make([]any, 0, len(keys))
	for _, k := range keys {
		attrs = append(attrs, JzodToCopilotKitParameter(k, def[k], context))
	}
	return attrs
}

func copyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

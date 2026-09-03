package jzod

import "sort"

// JzodToJsonSchema converts a Jzod element to a JSON Schema object.
// Object required arrays follow [KeysOf] insertion order.
func JzodToJsonSchema(element any, context map[string]any) map[string]any {
	if element == nil {
		return map[string]any{}
	}
	el, ok := element.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	if context == nil {
		context = map[string]any{}
	}
	description, _ := el["description"].(string)
	nullable, _ := el["nullable"].(bool)
	withMeta := func(schema map[string]any) map[string]any {
		if description != "" {
			schema["description"] = description
		}
		if nullable {
			schema["nullable"] = true
		}
		return schema
	}
	switch el["type"] {
	case "string":
		return withMeta(map[string]any{"type": "string"})
	case "number":
		return withMeta(map[string]any{"type": "number"})
	case "boolean":
		return withMeta(map[string]any{"type": "boolean"})
	case "any", "unknown", "undefined", "void":
		return withMeta(map[string]any{})
	case "never":
		return withMeta(map[string]any{"not": map[string]any{}})
	case "uuid", "bigint":
		return withMeta(map[string]any{"type": "string"})
	case "date":
		return withMeta(map[string]any{"type": "string", "format": "date-time"})
	case "literal":
		return withMeta(map[string]any{"const": el["definition"]})
	case "enum":
		return withMeta(map[string]any{"enum": el["definition"]})
	case "array":
		return withMeta(map[string]any{"type": "array", "items": JzodToJsonSchema(el["definition"], context)})
	case "object":
		def, _ := el["definition"].(map[string]any)
		properties := map[string]any{}
		required := []any{}
		partial, _ := el["partial"].(bool)
		for _, k := range KeysOf(def) {
			v := def[k]
			properties[k] = JzodToJsonSchema(v, context)
			child, _ := v.(map[string]any)
			optional, _ := child["optional"].(bool)
			if !optional && !partial {
				required = append(required, k)
			}
		}
		schema := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			schema["required"] = required
		}
		if el["nonStrict"] != true {
			schema["additionalProperties"] = false
		}
		return withMeta(schema)
	case "record":
		return withMeta(map[string]any{
			"type":                 "object",
			"additionalProperties": JzodToJsonSchema(el["definition"], context),
		})
	case "union":
		list, _ := el["definition"].([]any)
		anyOf := make([]any, len(list))
		for i, d := range list {
			anyOf[i] = JzodToJsonSchema(d, context)
		}
		return withMeta(map[string]any{"anyOf": anyOf})
	case "intersection":
		def, _ := el["definition"].(map[string]any)
		return withMeta(map[string]any{"allOf": []any{
			JzodToJsonSchema(def["left"], context),
			JzodToJsonSchema(def["right"], context),
		}})
	case "schemaReference":
		def, _ := el["definition"].(map[string]any)
		merged := mergeContext(context, contextOf(el))
		refPath, _ := def["relativePath"].(string)
		if resolved, ok := merged[refPath]; ok {
			return JzodToJsonSchema(resolved, merged)
		}
		return withMeta(map[string]any{"$ref": "#/$defs/" + refPath})
	case "lazy":
		return JzodToJsonSchema(el["definition"], context)
	case "tuple":
		list, _ := el["definition"].([]any)
		items := make([]any, len(list))
		for i, d := range list {
			items[i] = JzodToJsonSchema(d, context)
		}
		return withMeta(map[string]any{
			"type":     "array",
			"items":    items,
			"minItems": len(list),
			"maxItems": len(list),
		})
	case "map", "set":
		return withMeta(map[string]any{"type": "object"})
	default:
		return withMeta(map[string]any{})
	}
}

func objectKeyOrder(def map[string]any) []string {
	preferred := []string{"name", "uuid", "type", "id"}
	seen := map[string]bool{}
	var keys []string
	for _, k := range preferred {
		if _, ok := def[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range def {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

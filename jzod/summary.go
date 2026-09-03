package jzod

// JzodToJzodSummary returns a depth-limited structural summary of a Jzod schema.
func JzodToJzodSummary(schema any, _ any, depth int) any {
	el, ok := schema.(map[string]any)
	if !ok {
		return schema
	}
	base := buildSummaryBase(el)
	switch el["type"] {
	case "literal", "enum":
		base["definition"] = el["definition"]
		return base
	case "any", "bigint", "boolean", "never", "number", "string", "uuid", "undefined", "unknown", "void", "date":
		if v, ok := el["validations"]; ok {
			base["validations"] = v
		}
		return base
	case "object":
		def, ok := el["definition"].(map[string]any)
		if !ok {
			return base
		}
		allDef := map[string]any{}
		for k, v := range def {
			if depth <= 0 {
				allDef[k] = minimalSummary(v)
			} else {
				allDef[k] = JzodToJzodSummary(v, nil, depth-1)
			}
		}
		ordered := map[string]any{}
		for k, v := range allDef {
			if child, ok := v.(map[string]any); ok && child["type"] == "literal" {
				ordered[k] = v
			}
		}
		for k, v := range allDef {
			if _, exists := ordered[k]; !exists {
				ordered[k] = v
			}
		}
		base["definition"] = ordered
		if v, ok := el["nonStrict"]; ok {
			base["nonStrict"] = v
		}
		if v, ok := el["partial"]; ok {
			base["partial"] = v
		}
		if v, ok := el["extend"]; ok {
			base["extend"] = v
		}
		return base
	case "array", "record", "set", "promise":
		if el["definition"] == nil {
			return base
		}
		if depth <= 0 {
			base["definition"] = minimalSummary(el["definition"])
		} else {
			base["definition"] = JzodToJzodSummary(el["definition"], nil, depth-1)
		}
		return base
	case "tuple", "union":
		list, _ := el["definition"].([]any)
		items := make([]any, len(list))
		for i, item := range list {
			if depth <= 0 {
				items[i] = minimalSummary(item)
			} else {
				items[i] = JzodToJzodSummary(item, nil, depth-1)
			}
		}
		base["definition"] = items
		if el["type"] == "union" {
			if v, ok := el["discriminator"]; ok {
				base["discriminator"] = v
			}
			if v, ok := el["discriminatorNew"]; ok {
				base["discriminatorNew"] = v
			}
		}
		return base
	case "schemaReference":
		base["definition"] = el["definition"]
		return base
	case "intersection":
		def, _ := el["definition"].(map[string]any)
		left, right := def["left"], def["right"]
		if depth <= 0 {
			base["definition"] = map[string]any{"left": minimalSummary(left), "right": minimalSummary(right)}
		} else {
			base["definition"] = map[string]any{
				"left":  JzodToJzodSummary(left, nil, depth-1),
				"right": JzodToJzodSummary(right, nil, depth-1),
			}
		}
		return base
	case "map":
		list, _ := el["definition"].([]any)
		if len(list) < 2 {
			return base
		}
		if depth <= 0 {
			base["definition"] = []any{minimalSummary(list[0]), minimalSummary(list[1])}
		} else {
			base["definition"] = []any{JzodToJzodSummary(list[0], nil, depth-1), JzodToJzodSummary(list[1], nil, depth-1)}
		}
		return base
	default:
		return base
	}
}

func buildSummaryBase(el map[string]any) map[string]any {
	base := map[string]any{"type": el["type"]}
	if v, ok := el["optional"]; ok {
		base["optional"] = v
	}
	if v, ok := el["nullable"]; ok {
		base["nullable"] = v
	}
	if v, ok := el["description"]; ok {
		base["description"] = v
	}
	if tag, ok := el["tag"].(map[string]any); ok {
		if value, ok := tag["value"].(map[string]any); ok {
			stripped := map[string]any{}
			for _, k := range []string{"defaultLabel", "description", "foreignKeyParams", "formValidation"} {
				if v, ok := value[k]; ok {
					stripped[k] = v
				}
			}
			if len(stripped) > 0 {
				base["tag"] = map[string]any{"value": stripped}
			}
		}
	}
	return base
}

func minimalSummary(el any) any {
	m, ok := el.(map[string]any)
	if !ok {
		return el
	}
	switch m["type"] {
	case "literal", "enum", "schemaReference":
		return map[string]any{"type": m["type"], "definition": m["definition"]}
	default:
		return map[string]any{"type": m["type"]}
	}
}

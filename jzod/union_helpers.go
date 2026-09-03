package jzod

import "encoding/json"

// UnionObjectChoices returns the object and record branches of a union, after
// unfolding nested unions and references (TS jzodUnion_objectChoices).
func UnionObjectChoices(schemas []any, envMap any, relativeContext map[string]any) []any {
	var out []any
	for _, j := range schemas {
		m, _ := j.(map[string]any)
		if m["type"] == "record" {
			out = append(out, j)
		}
	}
	for _, j := range schemas {
		m, _ := j.(map[string]any)
		if m["type"] == "object" {
			flat, err := JzodObjectFlatten(j, envMap, relativeContext)
			if err == nil {
				out = append(out, flat)
			}
		}
	}
	for _, j := range schemas {
		m, _ := j.(map[string]any)
		if m["type"] != "union" {
			continue
		}
		def, _ := m["definition"].([]any)
		for _, k := range def {
			km, _ := k.(map[string]any)
			if km["type"] == "object" {
				flat, err := JzodObjectFlatten(k, envMap, relativeContext)
				if err == nil {
					out = append(out, flat)
				}
			}
		}
		for _, k := range def {
			km, _ := k.(map[string]any)
			if km["type"] != "schemaReference" {
				continue
			}
			resolved, err := ResolveJzodSchemaReferenceInContext(k, mergeContext(relativeContext, contextOf(km)), envMap)
			if err != nil {
				continue
			}
			rm, _ := resolved.(map[string]any)
			if rm["type"] == "object" {
				flat, err := JzodObjectFlatten(resolved, envMap, relativeContext)
				if err == nil {
					out = append(out, flat)
				}
			}
		}
	}
	return out
}

// UnionArrayChoices returns the array branches of a union (TS
// jzodUnion_arrayChoices).
func UnionArrayChoices(schemas []any, envMap any, relativeContext map[string]any) []any {
	var out []any
	for _, j := range schemas {
		m, _ := j.(map[string]any)
		if m["type"] == "array" || m["type"] == "tuple" {
			out = append(out, j)
		}
	}
	for _, j := range schemas {
		m, _ := j.(map[string]any)
		if m["type"] != "union" {
			continue
		}
		def, _ := m["definition"].([]any)
		for _, k := range def {
			km, _ := k.(map[string]any)
			if km["type"] == "array" || km["type"] == "tuple" {
				out = append(out, k)
			}
		}
		for _, k := range def {
			km, _ := k.(map[string]any)
			if km["type"] != "schemaReference" {
				continue
			}
			resolved, err := RecursiveResolveJzodSchemaReferenceInContext(k, mergeContext(relativeContext, contextOf(km)), envMap)
			if err != nil {
				continue
			}
			rm, _ := resolved.(map[string]any)
			if rm["type"] == "array" || rm["type"] == "tuple" {
				out = append(out, resolved)
			}
		}
	}
	return out
}

// SelectUnionBranchFromDiscriminator picks a union object branch using
// discriminator attribute values (TS selectUnionBranchFromDiscriminator).
func SelectUnionBranchFromDiscriminator(
	objectUnionChoices []any,
	effectiveRawSchema any,
	discriminator any,
	valueObject map[string]any,
	valueObjectPath []any,
	typePath []any,
	envMap any,
	relativeContext map[string]any,
) map[string]any {
	if discriminator == nil {
		return map[string]any{
			"status":             "error",
			"error":              "selectUnionBranchFromDiscriminator called for union-type value object without discriminator",
			"discriminator":      discriminator,
			"valuePath":          valueObjectPath,
			"typePath":           typePath,
			"value":              valueObject,
			"objectUnionChoices": objectUnionChoices,
		}
	}
	discriminators := normalizeDiscriminators(discriminator)
	var flattened []any
	for _, choice := range objectUnionChoices {
		cm, _ := choice.(map[string]any)
		if cm["extend"] != nil {
			extension, err := ResolveJzodSchemaReferenceInContext(cm["extend"], relativeContext, envMap)
			if err != nil {
				return map[string]any{
					"status": "error", "error": "selectUnionBranchFromDiscriminator object extend clause schema is not an object",
					"discriminator": discriminator, "valuePath": valueObjectPath, "typePath": typePath,
					"value": valueObject, "objectUnionChoices": objectUnionChoices,
				}
			}
			em, _ := extension.(map[string]any)
			if em["type"] != "object" {
				return map[string]any{
					"status": "error", "error": "selectUnionBranchFromDiscriminator object extend clause schema is not an object",
					"discriminator": discriminator, "valuePath": valueObjectPath, "typePath": typePath,
					"value": valueObject, "objectUnionChoices": objectUnionChoices,
				}
			}
			merged := map[string]any{}
			if def, ok := em["definition"].(map[string]any); ok {
				for k, v := range def {
					merged[k] = v
				}
			}
			if def, ok := cm["definition"].(map[string]any); ok {
				for k, v := range def {
					merged[k] = v
				}
			}
			flattened = append(flattened, map[string]any{"type": "object", "definition": merged})
		} else {
			flattened = append(flattened, choice)
		}
	}
	var discriminatorValues []any
	for _, d := range discriminators {
		if list, ok := d.([]any); ok {
			var nonNull []any
			for _, sub := range list {
				v := valueObject[fmtSprint(sub)]
				if v != nil {
					nonNull = append(nonNull, v)
				}
			}
			if len(nonNull) > 0 {
				discriminatorValues = append(discriminatorValues, nonNull[0])
			} else {
				discriminatorValues = append(discriminatorValues, nil)
			}
		} else {
			discriminatorValues = append(discriminatorValues, valueObject[fmtSprint(d)])
		}
	}
	filtered := flattened
	var chosen []any
	if len(discriminators) == 0 {
		filtered = filterChoicesByValue(flattened, valueObject)
	} else {
		flatDisc := flattenDisc(discriminators)
		hasValues := false
		for _, d := range flatDisc {
			if _, ok := valueObject[d]; ok {
				hasValues = true
				break
			}
		}
		if !hasValues {
			var none []any
			for _, choice := range flattened {
				cm, _ := choice.(map[string]any)
				def, _ := cm["definition"].(map[string]any)
				ok := true
				for key := range def {
					if containsStr(flatDisc, key) {
						ok = false
						break
					}
				}
				if ok {
					none = append(none, choice)
				}
			}
			if len(none) == 1 {
				return map[string]any{
					"status":                               "ok",
					"currentDiscriminatedObjectJzodSchema": filtered[0],
					"flattenedUnionChoices":                filtered,
					"chosenDiscriminator":                  []any{},
				}
			}
			return map[string]any{
				"status":        "error",
				"error":         "selectUnionBranchFromDiscriminator: no discriminator values found in valueObject and multiple choices exist",
				"discriminator": discriminator, "effectiveRawSchema": effectiveRawSchema,
				"valuePath": valueObjectPath, "typePath": typePath, "value": valueObject,
				"objectUnionChoices": objectUnionChoices, "flattenedUnionChoices": none,
			}
		}
		i := 0
		for i < len(discriminators) && len(filtered) > 1 {
			local := discriminators[i]
			var localChosen string
			if list, ok := local.([]any); ok {
				for _, d := range list {
					key := fmtSprint(d)
					if _, exists := valueObject[key]; exists {
						localChosen = key
						break
					}
				}
			} else {
				localChosen = fmtSprint(local)
			}
			var next []any
			for _, a := range filtered {
				am, _ := a.(map[string]any)
				def, _ := am["definition"].(map[string]any)
				field, _ := def[localChosen].(map[string]any)
				if field["type"] == "literal" && valuesEqual(field["definition"], valueObject[localChosen]) {
					next = append(next, a)
				} else if field["type"] == "enum" {
					if enumContains(field["definition"], valueObject[localChosen]) {
						next = append(next, a)
					}
				}
			}
			chosen = append(chosen, map[string]any{"discriminator": localChosen, "value": valueObject[localChosen]})
			filtered = next
			i++
		}
	}
	if len(filtered) == 0 {
		return map[string]any{
			"status":        "error",
			"error":         "selectUnionBranchFromDiscriminator called for union-type value object found no match with discriminator(s)=" + jsonStringify(discriminators),
			"discriminator": discriminators, "discriminatorValues": discriminatorValues,
			"valuePath": valueObjectPath, "typePath": typePath, "value": valueObject,
			"objectUnionChoices": objectUnionChoices, "flattenedUnionChoices": flattened,
		}
	}
	if len(filtered) > 1 {
		return map[string]any{
			"status":        "error",
			"error":         "selectUnionBranchFromDiscriminator called for union-type value object found many matches with discriminator(s)=" + jsonStringify(discriminators) + " found " + itoa(len(filtered)) + " matches.",
			"discriminator": discriminators, "discriminatorValues": discriminatorValues,
			"valuePath": valueObjectPath, "typePath": typePath, "value": valueObject,
			"objectUnionChoices": objectUnionChoices, "flattenedUnionChoices": filtered,
		}
	}
	return map[string]any{
		"status":                               "ok",
		"currentDiscriminatedObjectJzodSchema": filtered[0],
		"flattenedUnionChoices":                filtered,
		"chosenDiscriminator":                  chosen,
	}
}

// JzodUnionResolvedTypeForArray picks the array branch that matches valueArray
// (TS jzodUnionResolvedTypeForArray).
func JzodUnionResolvedTypeForArray(schemas []any, raw any, discriminator any, valueArray any, valuePath, typePath []any, envMap any, relativeContext map[string]any) map[string]any {
	choices := UnionArrayChoices(schemas, envMap, relativeContext)
	if len(choices) == 1 {
		return map[string]any{"status": "ok", "resolvedJzodObjectSchema": choices[0], "arrayUnionChoices": choices}
	}
	if len(choices) == 0 {
		return map[string]any{
			"status":    "error",
			"error":     "jzodUnionResolvedTypeForArray could not find object type for given array value in resolved union",
			"rawSchema": raw, "discriminator": discriminator, "valuePath": valuePath, "typePath": typePath,
			"value": valueArray, "concreteUnrolledJzodSchemas": schemas, "unionChoices": choices,
		}
	}
	return map[string]any{
		"status":        "error",
		"error":         "jzodUnionResolvedTypeForArray called for union-type value array with discriminator(s)=" + jsonStringify(discriminator) + " found " + itoa(len(choices)) + " matches.",
		"discriminator": discriminator, "valuePath": valuePath, "typePath": typePath,
		"value": valueArray, "concreteUnrolledJzodSchemas": schemas, "unionChoices": choices,
	}
}

// JzodUnionResolvedTypeForObject picks the object branch that matches
// valueObject (TS jzodUnionResolvedTypeForObject).
func JzodUnionResolvedTypeForObject(schemas []any, raw any, discriminator any, valueObject map[string]any, valuePath, typePath []any, envMap any, relativeContext map[string]any) map[string]any {
	choices := UnionObjectChoices(schemas, envMap, relativeContext)
	if len(choices) == 1 {
		return map[string]any{"status": "ok", "resolvedJzodObjectSchema": choices[0], "objectUnionChoices": choices}
	}
	rawMap, _ := raw.(map[string]any)
	if len(choices) == 0 && rawMap["optInDiscriminator"] != true {
		return map[string]any{
			"status":        "error",
			"error":         "jzodUnionResolvedTypeForObject could not find object type for given object value in resolved union",
			"discriminator": discriminator, "valuePath": valuePath, "typePath": typePath,
			"value": valueObject, "concreteUnrolledJzodSchemas": schemas, "unionChoices": choices,
		}
	}
	selected := SelectUnionBranchFromDiscriminator(choices, raw, discriminator, valueObject, valuePath, typePath, envMap, relativeContext)
	if selected["status"] == "error" {
		if rawMap["optInDiscriminator"] == true {
			return map[string]any{
				"status":                   "ok",
				"resolvedJzodObjectSchema": map[string]any{"type": "record", "definition": map[string]any{"type": "any"}},
				"objectUnionChoices":       choices,
				"chosenDiscriminator":      []any{map[string]any{"discriminator": "optInDiscriminator", "value": "optInDiscriminator"}},
			}
		}
		return map[string]any{
			"status":        "error",
			"error":         "jzodUnionResolvedTypeForObject failed to select union branch",
			"discriminator": discriminator, "valuePath": valuePath, "typePath": typePath,
			"innerError": selected, "value": valueObject, "concreteUnrolledJzodSchemas": schemas, "unionChoices": choices,
		}
	}
	return map[string]any{
		"status":                   "ok",
		"resolvedJzodObjectSchema": selected["currentDiscriminatedObjectJzodSchema"],
		"objectUnionChoices":       choices,
		"chosenDiscriminator":      selected["chosenDiscriminator"],
	}
}

// LocalizeJzodSchemaReferenceContext copies relative context onto nested
// schemaReferences so later unfold/resolve steps stay local (TS
// localizeJzodSchemaReferenceContext).
func LocalizeJzodSchemaReferenceContext(fundamental, element, currentModel, metaModel, relativeContext any) any {
	el, ok := element.(map[string]any)
	if !ok {
		return element
	}
	rel, _ := relativeContext.(map[string]any)
	switch el["type"] {
	case "object":
		out := copyMap(el)
		def, _ := el["definition"].(map[string]any)
		next := map[string]any{}
		for k, v := range def {
			next[k] = LocalizeJzodSchemaReferenceContext(fundamental, v, currentModel, metaModel, rel)
		}
		out["definition"] = next
		return out
	case "schemaReference":
		var localized any
		if ctx, ok := el["context"].(map[string]any); ok {
			merged := mergeContext(rel, ctx)
			loc := map[string]any{}
			for k, v := range ctx {
				loc[k] = LocalizeJzodSchemaReferenceContext(fundamental, v, currentModel, metaModel, merged)
			}
			localized = loc
		} else {
			localized = rel
		}
		out := copyMap(el)
		out["context"] = localized
		return out
	case "union":
		out := copyMap(el)
		list, _ := el["definition"].([]any)
		next := make([]any, len(list))
		for i, v := range list {
			next[i] = LocalizeJzodSchemaReferenceContext(fundamental, v, currentModel, metaModel, rel)
		}
		out["definition"] = next
		return out
	case "array":
		out := copyMap(el)
		out["definition"] = LocalizeJzodSchemaReferenceContext(fundamental, el["definition"], currentModel, metaModel, rel)
		return out
	default:
		return el
	}
}

func normalizeDiscriminators(v any) []any {
	if s, ok := v.(string); ok {
		return []any{s}
	}
	if list, ok := v.([]any); ok {
		return list
	}
	return []any{}
}

func flattenDisc(discs []any) []string {
	var out []string
	for _, d := range discs {
		if list, ok := d.([]any); ok {
			for _, x := range list {
				out = append(out, fmtSprint(x))
			}
		} else {
			out = append(out, fmtSprint(d))
		}
	}
	return out
}

func filterChoicesByValue(choices []any, valueObject map[string]any) []any {
	var out []any
	for _, choice := range choices {
		cm, _ := choice.(map[string]any)
		def, _ := cm["definition"].(map[string]any)
		ok := true
		for key := range valueObject {
			field, exists := def[key]
			if !exists {
				ok = false
				break
			}
			fm, _ := field.(map[string]any)
			if fm["type"] == "literal" && !valuesEqual(fm["definition"], valueObject[key]) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, choice)
		}
	}
	return out
}

func containsStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	return jsonStringify(n)
}

func jsonRaw(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

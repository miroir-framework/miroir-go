package jzod

// JzodUnionRecursivelyUnfold expands nested unions and schemaReferences into
// a flat list of concrete branches (TS jzodUnion_recursivelyUnfold).
func JzodUnionRecursivelyUnfold(jzodUnion any, expanded any, envMap any, relativeContext map[string]any) (any, error) {
	union, ok := jzodUnion.(map[string]any)
	if !ok {
		return map[string]any{"status": "error", "error": "expected union"}, nil
	}
	expandedSet := setFrom(expanded)
	def, _ := union["definition"].([]any)
	result := []any{}
	var references []any
	for _, a := range def {
		m, _ := a.(map[string]any)
		if m["type"] != "schemaReference" && m["type"] != "union" {
			result = append(result, a)
			continue
		}
		if m["type"] == "schemaReference" {
			rel, _ := refDefinition(m)["relativePath"].(string)
			if !expandedSet[rel] {
				references = append(references, a)
			}
		}
	}
	var resolved []any
	for _, a := range references {
		m, _ := a.(map[string]any)
		item, err := RecursiveResolveJzodSchemaReferenceInContext(a, mergeContext(relativeContext, contextOf(m)), envMap)
		if err != nil {
			return map[string]any{"status": "error", "error": "Error while recursively unfolding JzodUnion: " + err.Error()}, nil
		}
		resolved = append(resolved, item)
	}
	for _, r := range resolved {
		m, _ := r.(map[string]any)
		if m["type"] != "union" {
			result = append(result, r)
		}
	}
	newExpanded := copySet(expandedSet)
	for _, a := range references {
		m, _ := a.(map[string]any)
		if rel, _ := refDefinition(m)["relativePath"].(string); rel != "" {
			newExpanded[rel] = true
		}
	}
	var unions []any
	for _, a := range def {
		m, _ := a.(map[string]any)
		if m["type"] == "union" {
			unions = append(unions, a)
		}
	}
	for _, r := range resolved {
		m, _ := r.(map[string]any)
		if m["type"] == "union" {
			unions = append(unions, r)
		}
	}
	for _, r := range unions {
		sub, err := JzodUnionRecursivelyUnfold(r, newExpanded, envMap, relativeContext)
		if err != nil {
			return map[string]any{"status": "error", "error": err.Error()}, nil
		}
		sm, _ := sub.(map[string]any)
		if sm["status"] == "error" {
			return sub, nil
		}
		if list, ok := sm["result"].([]any); ok {
			result = append(result, list...)
		}
		if refs, ok := sm["expandedReferences"].([]any); ok {
			for _, ref := range refs {
				newExpanded[fmtSprint(ref)] = true
			}
		}
	}
	expandedOut := make([]any, 0, len(newExpanded))
	for k := range newExpanded {
		expandedOut = append(expandedOut, k)
	}
	out := map[string]any{
		"status":             "ok",
		"result":             result,
		"expandedReferences": expandedOut,
	}
	if disc, ok := union["discriminator"]; ok {
		out["discriminator"] = disc
	}
	return out, nil
}

func setFrom(v any) map[string]bool {
	out := map[string]bool{}
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			out[fmtSprint(item)] = true
		}
	case map[string]bool:
		return copySet(t)
	}
	return out
}

func copySet(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func fmtSprint(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return jsonStringify(v)
}

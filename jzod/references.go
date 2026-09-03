package jzod

import "fmt"

// SchemaReferencesList collects every schemaReference (and optionally extend
// references) found under element, including duplicates.
func SchemaReferencesList(element any, includeExtend bool) []any {
	var refs []any
	traverseRefs(element, &refs, nil, includeExtend)
	return refs
}

// SchemaReferencesSet is [SchemaReferencesList] with duplicate relative/absolute
// paths removed.
func SchemaReferencesSet(element any, includeExtend bool) []any {
	seen := map[string]bool{}
	var refs []any
	traverseRefs(element, &refs, seen, includeExtend)
	return refs
}

func traverseRefs(node any, refs *[]any, seen map[string]bool, includeExtend bool) {
	el, ok := node.(map[string]any)
	if !ok {
		return
	}
	switch el["type"] {
	case "object":
		if includeExtend {
			if ext, ok := el["extend"]; ok && ext != nil {
				if list, ok := ext.([]any); ok {
					for _, item := range list {
						if item != nil {
							traverseRefs(item, refs, seen, includeExtend)
						}
					}
				} else {
					traverseRefs(ext, refs, seen, includeExtend)
				}
			}
		}
		if def, ok := el["definition"].(map[string]any); ok {
			for _, v := range def {
				traverseRefs(v, refs, seen, includeExtend)
			}
		}
	case "function":
		if def, ok := el["definition"].(map[string]any); ok {
			if args, ok := def["args"].([]any); ok {
				for _, a := range args {
					traverseRefs(a, refs, seen, includeExtend)
				}
			}
			if ret := def["returns"]; ret != nil {
				traverseRefs(ret, refs, seen, includeExtend)
			}
		}
	case "array", "lazy", "promise", "record", "set":
		traverseRefs(el["definition"], refs, seen, includeExtend)
	case "intersection":
		if def, ok := el["definition"].(map[string]any); ok {
			traverseRefs(def["left"], refs, seen, includeExtend)
			traverseRefs(def["right"], refs, seen, includeExtend)
		}
	case "map", "tuple", "union":
		if list, ok := el["definition"].([]any); ok {
			for _, v := range list {
				traverseRefs(v, refs, seen, includeExtend)
			}
		}
	case "schemaReference":
		if seen != nil {
			key := jsonStringify(el)
			if seen[key] {
				return
			}
			seen[key] = true
		}
		*refs = append(*refs, el)
	}
}

// TransitiveDependencySet walks fundamental.definition.context from
// contextElementName and returns reachable context key names (TS
// jzodSchemaTransitiveDependencySet).
func TransitiveDependencySet(fundamental any, contextElementName string, includeExtend bool, filterPrefix string) ([]any, error) {
	fund, _ := fundamental.(map[string]any)
	if fund == nil {
		return nil, fmt.Errorf("miroirFundamentalJzodSchema.context is not defined")
	}
	def, _ := fund["definition"].(map[string]any)
	context, _ := fund["context"].(map[string]any)
	if context == nil && def != nil {
		context, _ = def["context"].(map[string]any)
	}
	if context == nil {
		return nil, fmt.Errorf("miroirFundamentalJzodSchema.context is not defined")
	}
	visited := map[string]bool{}
	toVisit := []string{contextElementName}
	for len(toVisit) > 0 {
		element := toVisit[0]
		toVisit = toVisit[1:]
		if visited[element] {
			continue
		}
		visited[element] = true
		node, ok := context[element]
		if !ok {
			keys := make([]string, 0, len(context))
			for k := range context {
				keys = append(keys, k)
			}
			return nil, fmt.Errorf("jzodTransitiveDependencySet Element %s not found in context:%s", element, jsonStringify(keys))
		}
		refs := SchemaReferencesSet(node, includeExtend)
		for _, ref := range refs {
			rm, _ := ref.(map[string]any)
			rdef, _ := rm["definition"].(map[string]any)
			rel, _ := rdef["relativePath"].(string)
			if rel != "" && !visited[rel] {
				toVisit = append(toVisit, rel)
			}
		}
	}
	out := make([]any, 0, len(visited))
	for k := range visited {
		if filterPrefix == "" || len(k) >= len(filterPrefix) {
			out = append(out, k)
		}
	}
	return out, nil
}

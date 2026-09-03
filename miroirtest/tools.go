package miroirtest

import (
	"encoding/json"
	"fmt"
	"sort"
)

func stringTuple(args []any) (any, error) {
	return args, nil
}

func domainStateToReduxDeploymentsState(args []any) (any, error) {
	domain, _ := args[0].(map[string]any)
	result := map[string]any{}
	for deploymentUUID, sectionRaw := range domain {
		sections, _ := sectionRaw.(map[string]any)
		for section, entitiesRaw := range sections {
			entities, _ := entitiesRaw.(map[string]any)
			for entityUUID, instancesRaw := range entities {
				instances, _ := instancesRaw.(map[string]any)
				ids := make([]any, 0, len(instances))
				for id := range instances {
					ids = append(ids, id)
				}
				sortStrings(ids)
				result[deploymentUUID+"_"+section+"_"+entityUUID] = map[string]any{
					"ids":      ids,
					"entities": instances,
				}
			}
		}
	}
	return result, nil
}

func safeResolvePathOnObject(args []any) (any, error) {
	valueObject := argAt(args, 0)
	path, _ := argAt(args, 1).([]any)
	if len(path) == 0 || isJSONUndefined(valueObject) || valueObject == nil {
		if isJSONUndefined(valueObject) {
			return jsonUndefined{}, nil
		}
		return valueObject, nil
	}
	acc := valueObject
	for _, curr := range path {
		if acc == nil {
			return jsonUndefined{}, nil
		}
		next, ok := indexPath(acc, curr)
		if !ok {
			return jsonUndefined{}, nil
		}
		acc = next
	}
	return acc, nil
}

func resolvePathOnObject(args []any) (any, error) {
	valueObject := argAt(args, 0)
	path, _ := argAt(args, 1).([]any)
	acc := valueObject
	for _, curr := range path {
		next, ok := indexPath(acc, curr)
		if !ok {
			return nil, QueryFailureError{
				QueryFailure: "ReferenceNotFound",
				Message:      "resolvePathOnObject value object=" + jsonString(acc) + " path segment " + fmt.Sprint(curr),
			}
		}
		acc = next
	}
	return acc, nil
}

func resolveRelativePath(args []any) (any, error) {
	valueObject := argAt(args, 0)
	initialPath, _ := argAt(args, 1).([]any)
	path, _ := argAt(args, 2).([]any)
	stack := []any{}
	current := valueObject
	for _, segment := range initialPath {
		stack = append(stack, current)
		next, ok := indexPath(current, segment)
		if !ok {
			if isMapSeg(segment) && !isArray(current) {
				return map[string]any{"error": "INITIAL_PATH_NON_ARRAY_MAP_SEGMENT", "segment": segment, "current": current}, nil
			}
			if !isObject(current) {
				return map[string]any{"error": "INITIAL_PATH_NOT_OBJECT", "segment": segment, "current": current}, nil
			}
			if isArray(current) {
				return map[string]any{"error": "INITIAL_PATH_ARRAY_INDEX_OUT_OF_BOUNDS", "segment": segment, "current": current}, nil
			}
			return map[string]any{"error": "INITIAL_PATH_SEGMENT_NOT_FOUND", "segment": segment, "current": current}, nil
		}
		current = next
	}
	acc := current
	parentIndex := len(stack) - 1
	for _, segment := range path {
		if fmt.Sprint(segment) == "#" {
			if parentIndex < 0 {
				return map[string]any{"error": "NO_PARENT_TO_GO_UP", "parentIndex": parentIndex, "stack": stack}, nil
			}
			acc = stack[parentIndex]
			parentIndex--
			continue
		}
		next, ok := indexPath(acc, segment)
		if !ok {
			if isMapSeg(segment) && !isArray(acc) {
				return map[string]any{"error": "MAP_SEGMENT_ON_NON_ARRAY", "segment": segment, "acc": acc}, nil
			}
			if !isObject(acc) {
				return map[string]any{"error": "PATH_NOT_OBJECT", "segment": segment, "acc": acc}, nil
			}
			if isArray(acc) {
				return map[string]any{"error": "PATH_ARRAY_INDEX_OUT_OF_BOUNDS", "segment": segment, "acc": acc}, nil
			}
			return map[string]any{"error": "PATH_SEGMENT_NOT_FOUND", "segment": segment, "acc": acc}, nil
		}
		acc = next
	}
	return acc, nil
}

func indexPath(acc any, curr any) (any, bool) {
	if acc == nil {
		return nil, false
	}
	if isMapSeg(curr) {
		list, ok := acc.([]any)
		if !ok {
			return nil, false
		}
		key := mapSegKey(curr)
		out := make([]any, len(list))
		for i, item := range list {
			m, _ := item.(map[string]any)
			out[i] = m[key]
		}
		return out, true
	}
	switch obj := acc.(type) {
	case map[string]any:
		key := pathKey(curr)
		v, ok := obj[key]
		return v, ok
	case []any:
		i, ok := pathIndex(curr)
		if !ok || i < 0 || i >= len(obj) {
			return nil, false
		}
		return obj[i], true
	default:
		return nil, false
	}
}

func isMapSeg(seg any) bool {
	m, ok := seg.(map[string]any)
	if !ok {
		return false
	}
	return m["type"] == "map"
}

func mapSegKey(seg any) string {
	m, _ := seg.(map[string]any)
	return fmt.Sprint(m["key"])
}

func pathKey(seg any) string {
	switch n := seg.(type) {
	case float64:
		if n == float64(int(n)) {
			return fmt.Sprintf("%d", int(n))
		}
	}
	return fmt.Sprint(seg)
}

func pathIndex(seg any) (int, bool) {
	switch n := seg.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	case string:
		var i int
		_, err := fmt.Sscanf(n, "%d", &i)
		return i, err == nil
	default:
		return 0, false
	}
}

func isArray(v any) bool {
	_, ok := v.([]any)
	return ok
}

func isObject(v any) bool {
	if v == nil {
		return false
	}
	switch v.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func sortStrings(ids []any) {
	sort.Slice(ids, func(i, j int) bool { return fmt.Sprint(ids[i]) < fmt.Sprint(ids[j]) })
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

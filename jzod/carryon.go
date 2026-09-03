package jzod

// MergePositionBased merges two carry-on / position-based collections (TS
// mergePositionBased).
func MergePositionBased(a any, b any) any {
	if isEmptyMerge(a) && isEmptyMerge(b) {
		return nil
	}
	na := normalizeMerge(a)
	nb := normalizeMerge(b)
	max := len(na)
	if len(nb) > max {
		max = len(nb)
	}
	result := make([]any, 0, max)
	for i := 0; i < max; i++ {
		var itemA, itemB any
		if i < len(na) {
			itemA = na[i]
		}
		if i < len(nb) {
			itemB = nb[i]
		}
		if truthyMerge(itemA) && truthyMerge(itemB) {
			result = append(result, append(asStringList(itemA), asStringList(itemB)...))
		} else if truthyMerge(itemA) {
			result = append(result, itemA)
		} else if truthyMerge(itemB) {
			result = append(result, itemB)
		}
	}
	return result
}

func isEmptyMerge(v any) bool {
	if v == nil {
		return true
	}
	if _, ok := v.(struct{}); ok {
		return true
	}
	if s, ok := v.(string); ok {
		return s == ""
	}
	return false
}

func truthyMerge(v any) bool {
	return !isEmptyMerge(v)
}

func normalizeMerge(input any) []any {
	if input == nil {
		return []any{}
	}
	if s, ok := input.(string); ok {
		return []any{s}
	}
	if list, ok := input.([]any); ok {
		return list
	}
	return []any{}
}

func asStringList(v any) []any {
	if list, ok := v.([]any); ok {
		out := make([]any, len(list))
		copy(out, list)
		return out
	}
	return []any{v}
}

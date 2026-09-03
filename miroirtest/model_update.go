package miroirtest

import "fmt"

func registerModelUpdate() {
	registry["miroir-core/1_core/model/ModelUpdate"] = map[string]Fn{
		"getModelUpdate": func(args []any) (any, error) {
			return getModelUpdate(argAt(args, 0), argAt(args, 1), argAt(args, 2))
		},
	}
}

func getModelUpdate(application, before, after any) (any, error) {
	beforeMap, _ := before.(map[string]any)
	afterMap, _ := after.(map[string]any)
	if fmtSprint(beforeMap["uuid"]) != fmtSprint(afterMap["uuid"]) {
		return nil, fmt.Errorf("EntityDefinitions must have the same UUID to compute a ModelUpdate.")
	}
	changes := jsonDiff(beforeMap, afterMap)
	if changes == nil {
		return nil, nil
	}
	ml, _ := changes["mlSchema"].(map[string]any)
	if ml == nil {
		return nil, fmt.Errorf("getModelUpdate: Only mlSchema.definition changes are currently supported.")
	}
	defChanges, _ := ml["definition"].(map[string]any)
	if defChanges == nil {
		return nil, fmt.Errorf("getModelUpdate: Only mlSchema.definition changes are currently supported.")
	}
	var addColumns []any
	var removeColumns []any
	hasStructural := false
	for key, value := range defChanges {
		if len(key) > 7 && key[len(key)-7:] == "__added" {
			addColumns = append(addColumns, map[string]any{"name": key[:len(key)-7], "definition": value})
			hasStructural = true
			continue
		}
		if len(key) > 9 && key[len(key)-9:] == "__deleted" {
			removeColumns = append(removeColumns, key[:len(key)-9])
			hasStructural = true
			continue
		}
		if attr, ok := value.(map[string]any); ok {
			for k := range attr {
				if k != "tag" && k != "tag__added" && k != "tag__deleted" {
					hasStructural = true
					break
				}
			}
		}
	}
	if !hasStructural {
		return nil, nil
	}
	payload := map[string]any{
		"application": application,
		"entityName":  beforeMap["name"],
		"entityUuid":  beforeMap["entityUuid"],
	}
	if len(addColumns) > 0 {
		payload["addColumns"] = addColumns
	}
	if len(removeColumns) > 0 {
		payload["removeColumns"] = removeColumns
	}
	return map[string]any{
		"actionType": "alterEntityAttribute",
		"endpoint":   "7947ae40-eb34-4149-887b-15a9021e714e",
		"payload":    payload,
	}, nil
}

func jsonDiff(before, after map[string]any) map[string]any {
	if valuesEqual(before, after, nil) {
		return nil
	}
	out := map[string]any{}
	seen := map[string]bool{}
	for k, bv := range before {
		seen[k] = true
		av, exists := after[k]
		if !exists {
			out[k+"__deleted"] = bv
			continue
		}
		if bm, ok := bv.(map[string]any); ok {
			if am, ok := av.(map[string]any); ok {
				if child := jsonDiff(bm, am); child != nil {
					out[k] = child
				}
				continue
			}
		}
		if !valuesEqual(bv, av, nil) {
			out[k] = av
		}
	}
	for k, av := range after {
		if seen[k] {
			continue
		}
		out[k+"__added"] = av
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func init() {
	registerModelUpdate()
}

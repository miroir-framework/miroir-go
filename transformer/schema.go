package transformer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/miroir-framework/miroir/go/jzod"
)

func defaultValueForSchema(schema any, a ApplyArgs) any {
	el, ok := asMap(schema)
	if !ok || el == nil {
		return nil
	}
	if opt, _ := el["optional"].(bool); opt {
		return nil
	}
	if tag, _ := asMap(el["tag"]); tag != nil {
		if value, _ := asMap(tag["value"]); value != nil {
			if init, _ := asMap(value["initializeTo"]); init != nil && init["initializeToType"] == "value" {
				return init["value"]
			}
		}
	}
	switch el["type"] {
	case "string", "uuid", "bigint":
		return ""
	case "number":
		return float64(0)
	case "boolean":
		return false
	case "literal":
		return el["definition"]
	case "array", "tuple", "set":
		return []any{}
	case "object":
		def, _ := asMap(el["definition"])
		out := map[string]any{}
		keys := KeysOfOrSorted(def)
		for _, k := range keys {
			child := defaultValueForSchema(def[k], a)
			if child != nil || !isOptionalSchema(def[k]) {
				if !isOptionalSchema(def[k]) {
					out[k] = child
				}
			}
		}
		jzod.RememberKeys(out, keys)
		return out
	case "record":
		return map[string]any{}
	case "schemaReference":
		resolved := resolveRef(el, a)
		if rm, ok := asMap(resolved); ok && rm["type"] != "schemaReference" {
			return defaultValueForSchema(resolved, a)
		}
		return map[string]any{}
	default:
		return nil
	}
}

func isOptionalSchema(schema any) bool {
	m, _ := asMap(schema)
	opt, _ := m["optional"].(bool)
	return opt
}

func resolveRef(el map[string]any, _ ApplyArgs) any {
	def, _ := asMap(el["definition"])
	rel, _ := def["relativePath"].(string)
	ctx := map[string]any{}
	if local, ok := asMap(el["context"]); ok {
		for k, v := range local {
			ctx[k] = v
		}
	}
	fund := jzod.FundamentalSchema()
	if fund != nil {
		if fctx, ok := asMap(fund["context"]); ok {
			for k, v := range fctx {
				if _, exists := ctx[k]; !exists {
					ctx[k] = v
				}
			}
		}
		if fdef, ok := asMap(fund["definition"]); ok {
			if fc, ok := asMap(fdef["context"]); ok {
				for k, v := range fc {
					if _, exists := ctx[k]; !exists {
						ctx[k] = v
					}
				}
			}
		}
	}
	if rel != "" {
		if v, ok := ctx[rel]; ok {
			return v
		}
	}
	return el
}

func unfoldSchemaOnce(schema any, a ApplyArgs) any {
	el, err := unfoldElement(schema, map[string]any{}, a)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	return map[string]any{"status": "ok", "element": el}
}

func unfoldElement(schema any, ctx map[string]any, a ApplyArgs) (any, error) {
	el, ok := asMap(schema)
	if !ok {
		return schema, nil
	}
	local := copyMap(ctx)
	if c, ok := asMap(el["context"]); ok {
		for k, v := range c {
			local[k] = v
		}
	}
	switch el["type"] {
	case "schemaReference":
		localized := localizeSchema(el, local)
		resolved := resolveLocalizedRef(localized, a)
		out := copyMap(mustMap(resolved))
		if _, has := el["optional"]; has {
			out["optional"] = el["optional"]
		}
		if _, has := el["nullable"]; has {
			out["optional"] = el["nullable"]
		}
		if el["tag"] != nil {
			out["tag"] = el["tag"]
		}
		return out, nil
	case "object":
		def, _ := asMap(el["definition"])
		newDef := map[string]any{}
		keys := KeysOfOrSorted(def)
		for _, k := range keys {
			child, err := unfoldElement(def[k], local, a)
			if err != nil {
				return nil, err
			}
			newDef[k] = child
		}
		out := copyMap(el)
		out["definition"] = newDef
		return out, nil
	case "array":
		child, err := unfoldElement(el["definition"], local, a)
		if err != nil {
			return nil, err
		}
		out := copyMap(el)
		out["definition"] = child
		return out, nil
	case "union":
		list, _ := el["definition"].([]any)
		newList := make([]any, len(list))
		for i, item := range list {
			child, err := unfoldElement(item, local, a)
			if err != nil {
				return nil, err
			}
			newList[i] = child
		}
		out := copyMap(el)
		out["definition"] = newList
		return out, nil
	default:
		return el, nil
	}
}

func localizeSchema(el map[string]any, relCtx map[string]any) map[string]any {
	if el == nil {
		return el
	}
	switch el["type"] {
	case "object":
		def, _ := asMap(el["definition"])
		newDef := map[string]any{}
		keys := KeysOfOrSorted(def)
		for _, k := range keys {
			if child, ok := asMap(def[k]); ok && child != nil {
				newDef[k] = localizeSchema(child, relCtx)
			} else {
				newDef[k] = def[k]
			}
		}
		out := copyMap(el)
		out["definition"] = newDef
		jzod.RememberKeys(newDef, keys)
		return out
	case "union":
		list, _ := el["definition"].([]any)
		newList := make([]any, len(list))
		for i, item := range list {
			if m, ok := asMap(item); ok {
				newList[i] = localizeSchema(m, relCtx)
			} else {
				newList[i] = item
			}
		}
		out := copyMap(el)
		out["definition"] = newList
		return out
	case "array", "record":
		if m, ok := asMap(el["definition"]); ok && m != nil {
			out := copyMap(el)
			out["definition"] = localizeSchema(m, relCtx)
			return out
		}
		return el
	case "schemaReference":
		var localizedCtx map[string]any
		if local, ok := asMap(el["context"]); ok && local != nil && len(local) > 0 {
			localizedCtx = map[string]any{}
			merged := copyMap(relCtx)
			for k, v := range local {
				merged[k] = v
			}
			keys := KeysOfOrSorted(local)
			for _, k := range keys {
				if cm, ok := asMap(local[k]); ok && cm != nil {
					localizedCtx[k] = localizeSchema(cm, merged)
				} else {
					localizedCtx[k] = local[k]
				}
			}
			jzod.RememberKeys(localizedCtx, keys)
		} else {
			localizedCtx = relCtx
		}
		out := copyMap(el)
		out["context"] = localizedCtx
		return out
	default:
		return el
	}
}

func resolveLocalizedRef(el map[string]any, a ApplyArgs) any {
	ctx := map[string]any{}
	if local, ok := asMap(el["context"]); ok {
		for k, v := range local {
			ctx[k] = v
		}
	}
	return resolveJzodRef(el, ctx, a)
}

func resolveConditionalSchema(xf map[string]any, a ApplyArgs) any {
	schema := xf["schema"]
	if schema == nil {
		schema = xf["mlSchema"]
	}
	el, ok := asMap(schema)
	if !ok || el == nil {
		return schema
	}
	tag, _ := asMap(el["tag"])
	if tag == nil {
		return schema
	}
	value, _ := asMap(tag["value"])
	if value == nil {
		return schema
	}
	if isTemplate, _ := value["isTemplate"].(bool); isTemplate {
		return schema
	}
	cfg, _ := asMap(value["ifThenElseMMLS"])
	if cfg == nil {
		return schema
	}
	effective := schema
	if parentSpec := cfg["parentUuid"]; parentSpec != nil {
		if pm, isMap := asMap(parentSpec); isMap {
			var pathToUse string
			if _, hasDual := pm["defaultValuePath"]; hasDual && pm["typeCheckPath"] != nil {
				if fmt.Sprint(xf["context"]) == "defaultValue" {
					pathToUse = fmt.Sprint(pm["defaultValuePath"])
				} else {
					pathToUse = fmt.Sprint(pm["typeCheckPath"])
				}
			} else if p, ok := pm["path"].(string); ok && p != "" {
				pathToUse = p
			} else {
				return map[string]any{
					"error":   "INVALID_PARENT_UUID_CONFIG",
					"details": prettyJSON(pm),
				}
			}
			parentUuid := resolveRelativePathValue(xf["valueObject"], xf["valuePath"], strings.Split(pathToUse, "."))
			if parentUuid == nil {
				return invalidParentUuid(parentUuid)
			}
			if em, ok := asMap(parentUuid); ok && em["error"] != nil {
				return invalidParentUuid(em)
			}
			parentStr := fmt.Sprint(parentUuid)
			if isTypedTransformer(parentUuid) {
				parentStr = fmt.Sprint(mustApply(a, "parentUuid", parentUuid))
			}
			env, _ := asMap(a.Env)
			currentModel, _ := asMap(env["currentModel"])
			found := findEntityFromUuid(currentModel, parentStr)
			if found == nil || found["mlSchema"] == nil {
				dep := "undefined"
				if env != nil && env["deploymentUuid"] != nil && fmt.Sprint(env["deploymentUuid"]) != "" {
					dep = fmt.Sprint(env["deploymentUuid"])
				}
				return map[string]any{
					"error":   "PARENT_NOT_FOUND",
					"details": fmt.Sprintf("No present Entity (or EntityVersion fallback) found for parentUuid %s in deployment %s", parentStr, dep),
				}
			}
			effective = found["mlSchema"]
		}
	}
	if ref := cfg["mmlsReference"]; ref != nil {
		effective = map[string]any{"type": "schemaReference", "definition": ref}
	}
	return effective
}

func invalidParentUuid(parentUuid any) map[string]any {
	return map[string]any{
		"error":   "INVALID_PARENT_UUID_CONFIG",
		"details": "parentUuid resolution failed: " + prettyJSON(parentUuid),
	}
}

func prettyJSON(v any) string {
	if em, ok := asMap(v); ok && em != nil {
		if em["error"] == "INITIAL_PATH_SEGMENT_NOT_FOUND" {
			seg, _ := json.Marshal(em["segment"])
			cur, _ := json.Marshal(em["current"])
			errb, _ := json.Marshal(em["error"])
			return "{\n  \"error\": " + string(errb) + ",\n  \"segment\": " + string(seg) + ",\n  \"current\": " + string(cur) + "\n}"
		}
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func resolveRelativePathValue(valueObject any, initialPath any, path []string) any {
	stack := []any{}
	current := valueObject
	for _, segment := range toAnySlice(initialPath) {
		stack = append(stack, current)
		next, ok := walkSegment(current, segment)
		if !ok {
			if !isJSONObject(current) {
				return map[string]any{"error": "INITIAL_PATH_NOT_OBJECT", "segment": segment, "current": current}
			}
			if _, isArr := asList(current); isArr {
				return map[string]any{"error": "INITIAL_PATH_ARRAY_INDEX_OUT_OF_BOUNDS", "segment": segment, "current": current}
			}
			return map[string]any{"error": "INITIAL_PATH_SEGMENT_NOT_FOUND", "segment": segment, "current": current}
		}
		current = next
	}
	acc := current
	parentIndex := len(stack) - 1
	for _, segment := range path {
		if segment == "#" {
			if parentIndex < 0 {
				return map[string]any{"error": "NO_PARENT_TO_GO_UP", "parentIndex": parentIndex, "stack": stack}
			}
			acc = stack[parentIndex]
			parentIndex--
			continue
		}
		next, ok := walkSegment(acc, segment)
		if !ok {
			if !isJSONObject(acc) {
				return map[string]any{"error": "PATH_NOT_OBJECT", "segment": segment, "acc": acc}
			}
			if _, isArr := asList(acc); isArr {
				return map[string]any{"error": "PATH_ARRAY_INDEX_OUT_OF_BOUNDS", "segment": segment, "acc": acc}
			}
			return map[string]any{"error": "PATH_SEGMENT_NOT_FOUND", "segment": segment, "acc": acc}
		}
		acc = next
	}
	return acc
}

func isJSONObject(v any) bool {
	if v == nil {
		return false
	}
	if _, ok := asMap(v); ok {
		return true
	}
	_, ok := asList(v)
	return ok
}

func toAnySlice(v any) []any {
	if v == nil {
		return nil
	}
	if list, ok := v.([]any); ok {
		return list
	}
	if list, ok := v.([]string); ok {
		out := make([]any, len(list))
		for i, s := range list {
			out[i] = s
		}
		return out
	}
	return nil
}

func findEntityFromUuid(model map[string]any, uuid string) map[string]any {
	if model == nil || uuid == "" {
		return nil
	}
	list, _ := model["entities"].([]any)
	for _, item := range list {
		if m, ok := asMap(item); ok && fmt.Sprint(m["uuid"]) == uuid {
			return m
		}
	}
	return nil
}

func resolveSchemaReferenceInContext(xf map[string]any, a ApplyArgs) any {
	ref := xf["jzodReference"]
	rel, _ := asMap(xf["relativeReferenceJzodContext"])
	if ref == nil {
		ref = xf["mlSchema"]
	}
	return resolveJzodRef(ref, rel, a)
}

func resolveJzodRef(ref any, relative map[string]any, a ApplyArgs) any {
	if list, ok := ref.([]any); ok {
		merged := map[string]any{}
		for _, item := range list {
			resolved := resolveJzodRef(item, relative, a)
			if rm, ok := asMap(resolved); ok && rm["type"] == "object" {
				if def, ok := asMap(rm["definition"]); ok {
					for k, v := range def {
						merged[k] = v
					}
				}
			}
		}
		return map[string]any{"type": "object", "definition": merged}
	}
	el := mustMap(ref)
	if el["type"] == "object" {
		return el
	}
	ctx := copyMap(relative)
	if local, ok := asMap(el["context"]); ok {
		for k, v := range local {
			ctx[k] = v
		}
	}
	def, _ := asMap(el["definition"])
	relPath, _ := def["relativePath"].(string)
	if relPath != "" {
		if v, ok := ctx[relPath]; ok {
			return v
		}
	}
	return resolveRef(el, a)
}

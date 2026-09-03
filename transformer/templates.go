package transformer

import (
	"encoding/json"
	"fmt"
)

// ResolveQueryTemplate is [ResolveQueryTemplateWithExtractorCombinerTransformer].
func ResolveQueryTemplate(queryTemplate any, modelEnvironment any) (any, error) {
	return ResolveQueryTemplateWithExtractorCombinerTransformer(queryTemplate, modelEnvironment)
}

// ResolveQueryTemplateWithExtractorCombinerTransformer applies build-step
// transformers inside a boxed query template (TS
// resolveQueryTemplateWithExtractorCombinerTransformer).
func ResolveQueryTemplateWithExtractorCombinerTransformer(queryTemplate any, _modelEnvironment any) (any, error) {
	qt, ok := asMap(queryTemplate)
	if !ok || qt == nil {
		return nil, fmt.Errorf("resolveQueryTemplateWithExtractorCombinerTransformer: queryTemplate is not an object")
	}
	params := map[string]any{}
	if page, ok := asMap(qt["pageParams"]); ok {
		for k, v := range page {
			params[k] = v
		}
	}
	if query, ok := asMap(qt["queryParams"]); ok {
		for k, v := range query {
			params[k] = v
		}
	}
	context, _ := asMap(qt["contextResults"])
	if context == nil {
		context = map[string]any{}
	}

	extractorsIn, _ := asMap(qt["extractorTemplates"])
	resolvedQueries := map[string]any{}
	for _, key := range sortedKeys(extractorsIn) {
		resolvedQueries[key] = resolveExtractorTemplate(extractorsIn[key], params, context)
	}
	var failedQueries []any
	for _, v := range resolvedQueries {
		if m, ok := asMap(v); ok && m["queryFailure"] != nil {
			failedQueries = append(failedQueries, v)
		}
	}
	if len(failedQueries) > 0 {
		raw, _ := json.Marshal(failedQueries)
		return nil, fmt.Errorf("resolveQueryTemplateWithExtractorCombinerTransformer QueryNotExecutable failedQueries: %s", raw)
	}

	combinersIn, _ := asMap(qt["combinerTemplates"])
	combiners := map[string]any{}
	for _, key := range sortedKeys(combinersIn) {
		combiners[key] = resolveExtractorTemplate(combinersIn[key], params, context)
	}

	out := map[string]any{
		"queryType":  "boxedQueryWithExtractorCombinerTransformer",
		"extractors": resolvedQueries,
		"combiners":  combiners,
	}
	copyIfPresent(out, qt, "pageParams")
	copyIfPresent(out, qt, "queryParams")
	copyIfPresent(out, qt, "contextResults")
	copyIfPresent(out, qt, "application")
	copyIfPresent(out, qt, "runtimeTransformers")
	return out, nil
}

func copyIfPresent(dst, src map[string]any, key string) {
	if v, ok := src[key]; ok {
		dst[key] = v
	}
}

func resolveExtractorTemplate(extractor any, params, context map[string]any) any {
	src, ok := asMap(extractor)
	if !ok || src == nil {
		return queryNotExecutable(extractor)
	}
	clean := copyMap(src)
	delete(clean, "extractorOrCombinerType")
	kind, _ := src["extractorOrCombinerType"].(string)
	label := firstString(src["label"], kind)

	switch kind {
	case "literal":
		return map[string]any{
			"extractorOrCombinerType": "literal",
			"definition":              src["definition"],
		}
	case "extractorInstancesByEntity":
		out := copyMap(clean)
		out["extractorOrCombinerType"] = "extractorInstancesByEntity"
		out["parentUuid"] = resolveMaybeTransformer(src["parentUuid"], params, context, label)
		if filter, ok := asMap(src["filter"]); ok && filter != nil {
			f := copyMap(filter)
			val, _ := Apply("build", filter["value"], params, context)
			f["value"] = val
			out["filter"] = f
		}
		return out
	case "extractorByPrimaryKey":
		out := copyMap(clean)
		out["extractorOrCombinerType"] = "extractorByPrimaryKey"
		out["parentUuid"] = resolveMaybeTransformer(src["parentUuid"], params, context, label)
		inst, _ := Apply("build", src["instanceUuid"], params, context)
		out["instanceUuid"] = inst
		return out
	case "extractorWrapperReturningObject":
		defIn, _ := asMap(src["definition"])
		defOut := map[string]any{}
		for _, key := range sortedKeys(defIn) {
			defOut[key] = resolveExtractorTemplate(defIn[key], params, context)
		}
		out := copyMap(clean)
		out["extractorOrCombinerType"] = "extractorWrapperReturningObject"
		out["definition"] = defOut
		return out
	case "extractorWrapperReturningList":
		list, _ := src["definition"].([]any)
		defOut := make([]any, len(list))
		for i, e := range list {
			em, _ := asMap(e)
			itemLabel := firstString(em["label"], "extractorWrapperReturningList label missing")
			val, _ := Apply("build", e, params, context)
			_ = itemLabel
			defOut[i] = val
		}
		out := copyMap(clean)
		out["extractorOrCombinerType"] = "extractorWrapperReturningList"
		out["definition"] = defOut
		return out
	case "combinerOneToMany", "combinerOneToOne":
		out := copyMap(clean)
		out["extractorOrCombinerType"] = kind
		out["parentUuid"] = resolveMaybeTransformer(src["parentUuid"], params, context, label)
		ref, _ := Apply("build", src["objectReference"], params, context)
		out["objectReference"] = ref
		return out
	case "combinerManyToMany":
		out := copyMap(clean)
		out["extractorOrCombinerType"] = kind
		out["parentUuid"] = resolveMaybeTransformer(src["parentUuid"], params, context, label)
		ref, _ := Apply("build", src["objectListReference"], params, context)
		out["objectListReference"] = ref
		return out
	case "combinerByHeteronomousManyToMany":
		out := copyMap(clean)
		out["extractorOrCombinerType"] = kind
		if s, ok := src["rootExtractorOrReference"].(string); ok {
			out["rootExtractorOrReference"] = s
		} else {
			out["rootExtractorOrReference"] = resolveExtractorTemplate(src["rootExtractorOrReference"], params, context)
		}
		return out
	default:
		return queryNotExecutable(extractor)
	}
}

func resolveMaybeTransformer(v any, params, context map[string]any, _label string) any {
	if s, ok := v.(string); ok {
		return s
	}
	out, _ := Apply("build", v, params, context)
	return out
}

func queryNotExecutable(query any) map[string]any {
	raw, _ := json.Marshal(query)
	return map[string]any{
		"queryFailure":  "QueryNotExecutable",
		"failureOrigin": []any{"AsyncQuerySelectors", "resolveExtractorTemplate"},
		"query":         string(raw),
	}
}

func firstString(values ...any) string {
	for _, v := range values {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

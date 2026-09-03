package transformer

import (
	"fmt"
	"strconv"
)

func resolveInnerReference(a ApplyArgs, xf map[string]any) any {
	tt, _ := xf["transformerType"].(string)
	interp, _ := xf["interpolation"].(string)
	if interp == "" {
		interp = "build"
	}
	if a.Step == "build" && interp == "runtime" {
		return xf
	}
	switch tt {
	case "returnValue":
		return xf["value"]
	case "generateUuid":
		return handleGenerateUuidRaw()
	case "mustacheStringTemplate":
		v, err := handleMustache(a, xf)
		if err != nil {
			throw(AsFailureOrWrap(err))
		}
		return v
	case "getFromContext":
		if a.Step == "build" {
			return xf
		}
		return resolveNamedOrPath(a, xf, "context", a.Context)
	case "getFromParameters":
		return resolveNamedOrPath(a, xf, "param", a.Params)
	case "accessDynamicPath":
		v, err := handleAccessDynamicPath(a, xf)
		if err != nil {
			throw(AsFailureOrWrap(err))
		}
		return v
	default:
		throw(Failure{
			QueryFailure:    "FailedTransformer",
			TransformerPath: pathAny(a.Path),
			FailureOrigin:   []any{"transformer_InnerReference_resolve"},
			FailureMessage:  "transformer_InnerReference_resolve failed, unknown transformerType for transformer=" + fmt.Sprint(xf),
			QueryContext:    "transformer_InnerReference_resolve failed, unknown transformerType for transformer=" + fmt.Sprint(xf),
			QueryParameters: xf,
		})
		return nil
	}
}

func resolveNamedOrPath(a ApplyArgs, xf map[string]any, kind string, bank map[string]any) any {
	if bank == nil {
		bank = map[string]any{}
	}
	if name, ok := xf["referenceName"].(string); ok && name != "" {
		if _, exists := bank[name]; !exists {
			throw(Failure{
				QueryFailure:    "ReferenceNotFound",
				TransformerPath: append(pathAny(a.Path), name),
				FailureOrigin:   []any{"transformer_resolveReference"},
				QueryReference:  name,
				FailureMessage:  "no referenceName " + name + " in " + kind,
				QueryContext:    stringify(sortedKeys(bank)),
			})
		}
		return bank[name]
	}
	if path, ok := xf["referencePath"].([]any); ok && len(path) > 0 {
		acc := any(bank)
		for _, seg := range path {
			next, ok := walkSegment(acc, seg)
			if !ok {
				throw(Failure{
					QueryFailure:    "FailedTransformer_getFromContext",
					TransformerPath: pathAny(a.Path),
					FailureOrigin:   []any{"transformer_resolveReference"},
					QueryReference:  path,
					FailureMessage:  "no referencePath " + joinSegs(path) + " found in queryContext",
				})
			}
			acc = next
		}
		return acc
	}
	throw(Failure{
		QueryFailure:    "FailedTransformer",
		TransformerPath: pathAny(a.Path),
		FailureOrigin:   []any{"transformer_resolveReference"},
		FailureMessage:  "no referenceName or referencePath in " + kind,
	})
	return nil
}

func walkSegment(acc any, seg any) (any, bool) {
	if list, ok := asList(acc); ok {
		idx, ok := pathIndex(seg)
		if !ok || idx < 0 || idx >= len(list) {
			return nil, false
		}
		return list[idx], true
	}
	m, ok := asMap(acc)
	if !ok || m == nil {
		return nil, false
	}
	v, exists := m[fmt.Sprint(seg)]
	return v, exists
}

func pathIndex(seg any) (int, bool) {
	switch t := seg.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		if t == float64(int(t)) {
			return int(t), true
		}
		return 0, false
	case string:
		n, err := strconv.Atoi(t)
		return n, err == nil
	default:
		return 0, false
	}
}

func joinSegs(path []any) string {
	out := ""
	for i, s := range path {
		if i > 0 {
			out += "."
		}
		out += fmt.Sprint(s)
	}
	return out
}

// AsFailureOrWrap returns err as [Failure], or wraps a plain error as
// queryFailure FailedTransformer.
func AsFailureOrWrap(err error) Failure {
	if f, ok := AsFailure(err); ok {
		return f
	}
	return Failure{QueryFailure: "FailedTransformer", FailureMessage: err.Error()}
}

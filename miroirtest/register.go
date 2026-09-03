package miroirtest

import (
	"github.com/miroir-framework/miroir/go/jzod"
	"github.com/miroir-framework/miroir/go/jzodgen"
	"github.com/miroir-framework/miroir/go/transformer"
)

func registerJzodHelpers() {
	registry["miroir-core/1_core/jzod/JzodToJsonSchema"] = map[string]Fn{
		"jzodToJsonSchema": func(args []any) (any, error) {
			ctx, _ := asMapOK(argAt(args, 1))
			return jzod.JzodToJsonSchema(argAt(args, 0), ctx), nil
		},
	}
	registry["miroir-core/1_core/jzod/JzodToCopilotKitParameter"] = map[string]Fn{
		"jzodToCopilotKitParameter": func(args []any) (any, error) {
			ctx, _ := asMapOK(argAt(args, 2))
			return jzod.JzodToCopilotKitParameter(fmtSprint(argAt(args, 0)), argAt(args, 1), ctx), nil
		},
	}
	registry["miroir-core/1_core/jzod/JzodToJzod_CarryOn"] = map[string]Fn{
		"mergePositionBased": func(args []any) (any, error) {
			return jzod.MergePositionBased(undefToNil(argAt(args, 0)), undefToNil(argAt(args, 1))), nil
		},
	}
	registry["miroir-core/1_core/jzod/JzodSchemaReferences"] = map[string]Fn{
		"JzodSchemaReferencesList": func(args []any) (any, error) {
			return jzod.SchemaReferencesList(argAt(args, 0), boolOr(argAt(args, 1), true)), nil
		},
		"JzodSchemaReferencesSet": func(args []any) (any, error) {
			return jsonSet(jzod.SchemaReferencesSet(argAt(args, 0), boolOr(argAt(args, 1), true))), nil
		},
		"jzodTransitiveDependencySet": func(args []any) (any, error) {
			include := boolOr(argAt(args, 2), false)
			filter, _ := argAt(args, 3).(string)
			set, err := jzod.TransitiveDependencySet(argAt(args, 0), fmtSprint(argAt(args, 1)), include, filter)
			if err != nil {
				return nil, err
			}
			return jsonSet(set), nil
		},
	}
	registry["miroir-core/1_core/jzod/JzodToJzod_Summary"] = map[string]Fn{
		"jzodToJzod_Summary": func(args []any) (any, error) {
			depth := 1
			if len(args) > 2 && !isJSONUndefined(argAt(args, 2)) {
				depth = intFrom(argAt(args, 2), 1)
			}
			return jzod.JzodToJzodSummary(argAt(args, 0), argAt(args, 1), depth), nil
		},
	}
	registry["miroir-core/1_core/jzod/jzodObjectFlatten"] = map[string]Fn{
		"jzodObjectFlatten": func(args []any) (any, error) {
			rel, _ := asMapOK(argAt(args, 2))
			return jzod.JzodObjectFlatten(argAt(args, 0), argAt(args, 1), rel)
		},
	}
	registry["miroir-core/1_core/jzod/jzodTypeCheck"] = map[string]Fn{
		"buildAnyObjectEntry": func(args []any) (any, error) {
			obj, _ := asMapOK(argAt(args, 0))
			path, _ := argAt(args, 1).([]any)
			typePath, _ := argAt(args, 2).([]any)
			return jzod.BuildAnyObjectEntry(obj, path, typePath), nil
		},
		"buildAnySubnodeKeyMap": func(args []any) (any, error) {
			obj, _ := asMapOK(argAt(args, 0))
			path, _ := argAt(args, 1).([]any)
			typePath, _ := argAt(args, 2).([]any)
			return jzod.BuildAnySubnodeKeyMap(obj, path, typePath), nil
		},
		"selectUnionBranchFromDiscriminator": func(args []any) (any, error) {
			choices, _ := argAt(args, 0).([]any)
			valueObject, _ := asMapOK(argAt(args, 3))
			valuePath, _ := argAt(args, 4).([]any)
			typePath, _ := argAt(args, 5).([]any)
			rel, _ := asMapOK(argAt(args, 7))
			return jzod.SelectUnionBranchFromDiscriminator(choices, argAt(args, 1), undefToNil(argAt(args, 2)), valueObject, valuePath, typePath, argAt(args, 6), rel), nil
		},
		"unionObjectChoices": func(args []any) (any, error) {
			list, _ := argAt(args, 0).([]any)
			rel, _ := asMapOK(argAt(args, 2))
			return jzod.UnionObjectChoices(list, argAt(args, 1), rel), nil
		},
		"unionArrayChoices": func(args []any) (any, error) {
			list, _ := argAt(args, 0).([]any)
			rel, _ := asMapOK(argAt(args, 2))
			return jzod.UnionArrayChoices(list, argAt(args, 1), rel), nil
		},
		"jzodUnionResolvedTypeForObject": func(args []any) (any, error) {
			list, _ := argAt(args, 0).([]any)
			valueObject, _ := asMapOK(argAt(args, 3))
			valuePath, _ := argAt(args, 4).([]any)
			typePath, _ := argAt(args, 5).([]any)
			rel, _ := asMapOK(argAt(args, 7))
			return jzod.JzodUnionResolvedTypeForObject(list, argAt(args, 1), undefToNil(argAt(args, 2)), valueObject, valuePath, typePath, argAt(args, 6), rel), nil
		},
		"jzodUnionResolvedTypeForArray": func(args []any) (any, error) {
			list, _ := argAt(args, 0).([]any)
			valuePath, _ := argAt(args, 4).([]any)
			typePath, _ := argAt(args, 5).([]any)
			rel, _ := asMapOK(argAt(args, 7))
			return jzod.JzodUnionResolvedTypeForArray(list, argAt(args, 1), undefToNil(argAt(args, 2)), argAt(args, 3), valuePath, typePath, argAt(args, 6), rel), nil
		},
	}
	registry["miroir-core/1_core/jzod/jzodUnion_RecursivelyUnfold"] = map[string]Fn{
		"jzodUnion_recursivelyUnfold": func(args []any) (any, error) {
			rel, _ := asMapOK(argAt(args, 3))
			expanded := setArg(argAt(args, 1))
			out, err := jzod.JzodUnionRecursivelyUnfold(argAt(args, 0), expanded, argAt(args, 2), rel)
			if err != nil {
				return out, err
			}
			if m, ok := out.(map[string]any); ok {
				if refs, ok := m["expandedReferences"].([]any); ok {
					m["expandedReferences"] = jsonSet(refs)
				}
			}
			return out, nil
		},
	}
	registry["miroir-core/1_core/jzod/JzodUnfoldSchemaOnce"] = map[string]Fn{
		"localizeJzodSchemaReferenceContext": func(args []any) (any, error) {
			return jzod.LocalizeJzodSchemaReferenceContext(argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), argAt(args, 4)), nil
		},
	}
	registry["miroir-core/1_core/jzod/getAttributeTypesFromJzodSchema"] = map[string]Fn{
		"getAttributeTypesFromJzodSchema": func(args []any) (any, error) {
			return jzod.GetAttributeTypesFromJzodSchema(argAt(args, 0))
		},
	}
	registry["miroir-store-postgres/1_core/mlSchema"] = map[string]Fn{
		"getAttributeTypesFromJzodSchema": func(args []any) (any, error) {
			return jzod.GetAttributeTypesFromJzodSchema(argAt(args, 0))
		},
	}
	registry["miroir-core/1_core/ansiColumnsToJzodSchema"] = map[string]Fn{
		"ansiColumnsToJzodSchema": func(args []any) (any, error) {
			return jzod.AnsiColumnsToJzodSchema(argAt(args, 0))
		},
	}
	registry["miroir-core/2_domain/Templates"] = map[string]Fn{
		"resolveQueryTemplateWithExtractorCombinerTransformer": func(args []any) (any, error) {
			out, err := transformer.ResolveQueryTemplate(argAt(args, 0), argAt(args, 1))
			if err != nil {
				return nil, err
			}
			if m, ok := out.(map[string]any); ok {
				if m["contextResults"] == nil {
					m["contextResults"] = jsonUndefined{}
				}
				if m["runtimeTransformers"] == nil {
					m["runtimeTransformers"] = jsonUndefined{}
				}
			}
			return out, nil
		},
	}
	registry["miroir-core/2_domain/Transformer_ResultSchema"] = map[string]Fn{
		"resolveTransformerResultSchema": func(args []any) (any, error) {
			ctx, _ := asMapOK(argAt(args, 1))
			defs, _ := asMapOK(argAt(args, 2))
			if defs == nil || len(defs) == 0 {
				defs = transformer.TransformerDefinitions()
			}
			return transformer.ResolveResultSchema(argAt(args, 0), ctx, defs), nil
		},
	}
	registry["miroir-core/2_domain/TransformerInterfaceInference"] = map[string]Fn{
		"inferTransformerOutputTypeFromSchema": func(args []any) (any, error) {
			return transformer.InferOutputTypeFromSchema(argAt(args, 0), undefToNil(argAt(args, 1))), nil
		},
		"inferElementTransformerOutputType": func(args []any) (any, error) {
			out := transformer.InferElementTransformerOutputType(argAt(args, 0), undefToNil(argAt(args, 1)), undefToNil(argAt(args, 2)))
			if out == nil {
				return jsonUndefined{}, nil
			}
			return out, nil
		},
	}
	registry["miroir-core/2_domain/VirtualAttributes"] = map[string]Fn{
		"evaluateVirtualAttributesOnInstance": func(args []any) (any, error) {
			return transformer.EvaluateVirtualAttributesOnInstance(argAt(args, 0), argAt(args, 1), argAt(args, 2), argAt(args, 3), undefToNil(argAt(args, 4)))
		},
		"stripVirtualAttributesFromInstance": func(args []any) (any, error) {
			return transformer.StripVirtualAttributesFromInstance(argAt(args, 0), argAt(args, 1))
		},
	}
	registry["miroir-go/jzodgen"] = map[string]Fn{
		"JzodToGoType": func(args []any) (any, error) {
			name := fmtSprint(undefToNil(argAt(args, 0)))
			env, _ := asMapOK(undefToNil(argAt(args, 2)))
			return jzodgen.JzodToGoType(name, undefToNil(argAt(args, 1)), env)
		},
	}
}

func asMapOK(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func boolOr(v any, fallback bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return fallback
}

func undefToNil(v any) any {
	if isJSONUndefined(v) {
		return nil
	}
	return v
}

func setArg(v any) any {
	if s, ok := v.(jsonSet); ok {
		return []any(s)
	}
	return v
}

func init() {
	registerJzodHelpers()
}

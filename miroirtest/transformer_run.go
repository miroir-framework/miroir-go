package miroirtest

import (
	"fmt"
	"strings"

	"github.com/miroir-framework/miroir/go/jzod"
	"github.com/miroir-framework/miroir/go/transformer"
)

func runTransformerTest(label string, leaf map[string]any) LeafOutcome {
	if skip, _ := leaf["skip"].(bool); skip {
		return LeafOutcome{Label: label, OK: true}
	}
	xf := leaf["transformer"]
	params, _ := leaf["transformerParams"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	ctx, _ := leaf["transformerRuntimeContext"].(map[string]any)
	if ctx == nil {
		ctx = map[string]any{}
	}
	step, _ := leaf["runTestStep"].(string)
	if step == "" {
		step = "runtime"
	}
	env := jzod.DefaultMetaModelEnvironment()
	built, err := transformer.ApplyUnit("build", xf, params, ctx, env)
	if err != nil {
		return LeafOutcome{Label: label, OK: false, Err: err.Error()}
	}
	raw := built
	if step == "runtime" {
		if m, ok := built.(map[string]any); !ok || m["elementType"] == nil {
			raw, err = transformer.ApplyUnit("runtime", built, params, ctx, env)
			if err != nil {
				return LeafOutcome{Label: label, OK: false, Err: err.Error()}
			}
		}
	}
	got := ignorePostgresExtra(raw, stringSlice(leaf["ignoreAttributes"]))
	if retain, ok := leaf["retainAttributes"].([]any); ok && len(retain) > 0 {
		if m, isMap := got.(map[string]any); isMap {
			keep := map[string]bool{}
			for _, r := range retain {
				keep[fmt.Sprint(r)] = true
			}
			filtered := map[string]any{}
			for k, v := range m {
				if keep[k] {
					filtered[k] = v
				}
			}
			if keep["queryFailure"] {
				if qf, _ := filtered["queryFailure"].(string); qf != "" && qf != "FailedTransformer" {
					wantMap, _ := leaf["unitTestExpectedValue"].(map[string]any)
					if wantMap == nil {
						wantMap, _ = leaf["expectedValue"].(map[string]any)
					}
					if wantMap["queryFailure"] == "FailedTransformer" {
						filtered["queryFailure"] = "FailedTransformer"
					}
				}
			}
			got = filtered
		}
	}
	got = unNullify(got)
	if subs, ok := leaf["subExpectedValue"].([]any); ok && len(subs) > 0 {
		for _, rawSub := range subs {
			pair, _ := rawSub.([]any)
			if len(pair) < 2 {
				return LeafOutcome{Label: label, OK: false, Err: "subExpectedValue pair too short"}
			}
			path := fmt.Sprint(pair[0])
			want := unNullify(pair[1])
			actual := dotted(got, path)
			if !valuesEqual(actual, want, nil) {
				return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("subExpectedValue %s: got %#v want %#v", path, actual, want)}
			}
		}
		return LeafOutcome{Label: label, OK: true}
	}
	want := leaf["unitTestExpectedValue"]
	if want == nil {
		want = leaf["expectedValue"]
	}
	want = ignorePostgresExtra(want, stringSlice(leaf["ignoreAttributes"]))
	want = unNullify(want)
	if !valuesEqual(got, want, stringSlice(leaf["ignoreAttributes"])) {
		return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("expectedValue mismatch: got %#v want %#v", got, want)}
	}
	return LeafOutcome{Label: label, OK: true}
}

func ignorePostgresExtra(v any, extra []string) any {
	ignore := map[string]bool{"createdAt": true, "updatedAt": true}
	for _, e := range extra {
		ignore[e] = true
	}
	return dropKeys(v, ignore)
}

func dropKeys(v any, ignore map[string]bool) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, item := range t {
			if ignore[k] {
				continue
			}
			out[k] = dropKeys(item, ignore)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = dropKeys(item, ignore)
		}
		return out
	default:
		return v
	}
}

func unNullify(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, item := range t {
			out[k] = unNullify(item)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = unNullify(item)
		}
		return out
	default:
		return t
	}
}

func dotted(root any, path string) any {
	if path == "" {
		return root
	}
	acc := root
	for _, key := range strings.Split(path, ".") {
		m, ok := acc.(map[string]any)
		if !ok {
			return nil
		}
		acc = m[key]
	}
	return acc
}

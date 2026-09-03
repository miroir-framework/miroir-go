package miroirtest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/miroir-framework/miroir/go/jzod"
)

// Fn is a functionCallTest whitelist entry: the same (args) contract as
// TypeScript FUNCTION_CALL_REGISTRY exports.
type Fn func(args []any) (any, error)

var registry = map[string]map[string]Fn{
	"miroir-core/1_core/mustache": {
		"extractDoubleBracePatterns": extractDoubleBracePatterns,
	},
	"miroir-core/tools": {
		"alterObjectAtPath":                  alterObjectAtPath,
		"stringTuple":                        stringTuple,
		"domainStateToReduxDeploymentsState": domainStateToReduxDeploymentsState,
		"safeResolvePathOnObject":            safeResolvePathOnObject,
		"resolvePathOnObject":                resolvePathOnObject,
		"resolveRelativePath":                resolveRelativePath,
	},
}

// LeafOutcome is the result of one walked MiroirTest leaf.
type LeafOutcome struct {
	Label string
	OK    bool
	Err   string
}

// RunSuite walks doc and runs every functionCallTest and transformerTest leaf.
// Other miroirTestType values fail closed.
func RunSuite(doc map[string]any) []LeafOutcome {
	var out []LeafOutcome
	for _, leaf := range WalkLeaves(doc) {
		out = append(out, runLeaf(leaf))
	}
	return out
}

// RunFile loads path with [LoadFile] and runs [RunSuite].
func RunFile(path string) ([]LeafOutcome, error) {
	doc, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	return RunSuite(doc), nil
}

func runLeaf(leaf map[string]any) LeafOutcome {
	label, _ := leaf["miroirTestLabel"].(string)
	switch leaf["miroirTestType"] {
	case "functionCallTest":
		return runFunctionCall(label, leaf)
	case "transformerTest":
		return runTransformerTest(label, leaf)
	default:
		return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("miroirTestType %v not implemented", leaf["miroirTestType"])}
	}
}

func runFunctionCall(label string, leaf map[string]any) LeafOutcome {
	ref, _ := leaf["functionRef"].(map[string]any)
	module, _ := ref["module"].(string)
	export, _ := ref["export"].(string)
	fn := lookup(module, export)
	if fn == nil {
		return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("unknown functionRef %s / %s", module, export)}
	}
	args, err := prepareArguments(leaf)
	if err != nil {
		return LeafOutcome{Label: label, OK: false, Err: err.Error()}
	}
	got, err := fn(args)
	if expectedErr, ok := leaf["expectedError"].(string); ok && expectedErr != "" {
		if err == nil {
			return LeafOutcome{Label: label, OK: false, Err: "expected error " + expectedErr}
		}
		if !errorMatches(err, expectedErr) {
			return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("error: got %q want contains %q", err.Error(), expectedErr)}
		}
		return LeafOutcome{Label: label, OK: true}
	}
	if err != nil {
		return LeafOutcome{Label: label, OK: false, Err: err.Error()}
	}
	if expectedType, ok := leaf["expectedAction2ErrorType"].(string); ok && expectedType != "" {
		m, isMap := got.(map[string]any)
		if !isMap || m["errorType"] != expectedType {
			return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("expectedAction2ErrorType %s: got %#v", expectedType, got)}
		}
		return LeafOutcome{Label: label, OK: true}
	}
	if expectUndef, _ := leaf["expectUndefinedResult"].(bool); expectUndef {
		if got != nil && !isJSONUndefined(got) {
			return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("expected undefined, got %#v", got)}
		}
		return LeafOutcome{Label: label, OK: true}
	}
	if assertions, ok := leaf["assertions"].([]any); ok && len(assertions) > 0 {
		return runAssertions(label, got, assertions, stringSlice(leaf["ignoreAttributes"]))
	}
	want := deserializeFunctionCallValue(leaf["expectedValue"])
	if isJSONUndefined(want) {
		if got != nil && !isJSONUndefined(got) {
			return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("expected undefined, got %#v", got)}
		}
		return LeafOutcome{Label: label, OK: true}
	}
	ignore := stringSlice(leaf["ignoreAttributes"])
	if !valuesEqual(got, want, ignore) {
		return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("expectedValue mismatch: got %#v want %#v", got, want)}
	}
	return LeafOutcome{Label: label, OK: true}
}

func runAssertions(label string, got any, assertions []any, ignore []string) LeafOutcome {
	for _, raw := range assertions {
		a, _ := raw.(map[string]any)
		sub, _ := a["label"].(string)
		path, _ := a["resultAccessPath"].([]any)
		actual, err := resolvePathValue(got, path)
		if err != nil {
			return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("%s: %s", sub, err)}
		}
		want := deserializeFunctionCallValue(a["expectedValue"])
		if pattern, ok := matchPattern(want); ok {
			if !regexp.MustCompile(pattern).MatchString(fmt.Sprint(actual)) {
				return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("%s: %q did not match %q", sub, actual, pattern)}
			}
			continue
		}
		if !valuesEqual(actual, want, ignore) {
			return LeafOutcome{Label: label, OK: false, Err: fmt.Sprintf("%s: got %#v want %#v", sub, actual, want)}
		}
	}
	return LeafOutcome{Label: label, OK: true}
}

func matchPattern(want any) (string, bool) {
	m, ok := want.(map[string]any)
	if !ok {
		return "", false
	}
	p, ok := m["__miroirMatchPattern"]
	if !ok {
		return "", false
	}
	return fmt.Sprint(p), true
}

func prepareArguments(leaf map[string]any) ([]any, error) {
	rawArgs, _ := leaf["arguments"].([]any)
	if rawArgs == nil {
		rawArgs = []any{}
	}
	args, _ := deserializeFunctionCallValue(rawArgs).([]any)
	if args == nil {
		args = []any{}
	}
	if fixtureRef, ok := leaf["fixtureRef"].(string); ok && fixtureRef != "" {
		fixture, err := resolveFixture(fixtureRef)
		if err != nil {
			return nil, err
		}
		prop, _ := leaf["fixtureProperty"].(string)
		value, err := resolveFixtureProperty(fixture, prop)
		if err != nil {
			return nil, err
		}
		index := intFrom(leaf["fixtureArgumentIndex"], 0)
		args = insertAt(args, index, value)
	}
	if envRef, ok := leaf["environmentRef"].(string); ok && envRef != "" {
		env, err := resolveEnvironment(envRef)
		if err != nil {
			return nil, err
		}
		index := intFrom(leaf["environmentArgumentIndex"], 1)
		args = insertAt(args, index, env)
	}
	return args, nil
}

func insertAt(args []any, index int, value any) []any {
	if index < 0 {
		index = 0
	}
	if index > len(args) {
		index = len(args)
	}
	out := make([]any, 0, len(args)+1)
	out = append(out, args[:index]...)
	out = append(out, value)
	out = append(out, args[index:]...)
	return out
}

func intFrom(v any, fallback int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return fallback
	}
}

type jsonUndefined struct{}

type jsonSet []any

func isJSONUndefined(v any) bool {
	_, ok := v.(jsonUndefined)
	return ok
}

func deserializeFunctionCallValue(value any) any {
	if m, ok := value.(map[string]any); ok {
		if flag, ok := m["__miroirJsonUndefined"].(bool); ok && flag {
			return jsonUndefined{}
		}
		if items, ok := m["__miroirJsonSet"]; ok {
			list, _ := items.([]any)
			out := make(jsonSet, len(list))
			for i, v := range list {
				out[i] = deserializeFunctionCallValue(v)
			}
			return out
		}
		if ref, ok := m["__miroirEnvironmentRef"].(string); ok {
			env, err := resolveEnvironment(ref)
			if err != nil {
				return err
			}
			return env
		}
		if ref, ok := m["__fixtureRef"].(string); ok {
			fixture, err := resolveFixture(ref)
			if err != nil {
				return err
			}
			prop, _ := m["__fixtureProperty"].(string)
			value, err := resolveFixtureProperty(fixture, prop)
			if err != nil {
				return err
			}
			return value
		}
		out := make(map[string]any, len(m))
		if keys, ok := jzod.RememberedKeys(m); ok {
			for _, k := range keys {
				if v, exists := m[k]; exists {
					out[k] = deserializeFunctionCallValue(v)
				}
			}
			jzod.RememberKeys(out, keys)
			return out
		}
		for k, v := range m {
			out[k] = deserializeFunctionCallValue(v)
		}
		return out
	}
	if list, ok := value.([]any); ok {
		out := make([]any, len(list))
		for i, v := range list {
			out[i] = deserializeFunctionCallValue(v)
		}
		return out
	}
	return value
}

func lookup(module, export string) Fn {
	if m := registry[module]; m != nil {
		return m[export]
	}
	return nil
}

func valuesEqual(got, want any, ignore []string) bool {
	return reflect.DeepEqual(normalizeForCompare(got, ignore), normalizeForCompare(want, ignore))
}

func normalizeForCompare(value any, ignore []string) any {
	switch v := value.(type) {
	case jsonUndefined:
		return nil
	case jsonSet:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = normalizeForCompare(item, ignore)
		}
		if allStringable(items) {
			sort.Slice(items, func(i, j int) bool {
				return fmt.Sprint(items[i]) < fmt.Sprint(items[j])
			})
		}
		return items
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			if containsString(ignore, k) || isJSONUndefined(item) {
				continue
			}
			out[k] = normalizeForCompare(item, ignore)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalizeForCompare(item, ignore)
		}
		return out
	case jzod.Result:
		b, _ := json.Marshal(v)
		var m any
		_ = json.Unmarshal(b, &m)
		return normalizeForCompare(m, ignore)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return v
		}
		var norm any
		if err := json.Unmarshal(b, &norm); err != nil {
			return v
		}
		if normMap, ok := norm.(map[string]any); ok {
			return normalizeForCompare(normMap, ignore)
		}
		if normList, ok := norm.([]any); ok {
			return normalizeForCompare(normList, ignore)
		}
		return norm
	}
}

func allStringable(items []any) bool {
	for _, item := range items {
		switch item.(type) {
		case string, float64, bool, nil:
		default:
			return false
		}
	}
	return true
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func stringSlice(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, fmt.Sprint(item))
	}
	return out
}

func errorMatches(err error, expected string) bool {
	if qf, ok := err.(QueryFailureError); ok && qf.QueryFailure == expected {
		return true
	}
	return strings.Contains(err.Error(), expected)
}

// QueryFailureError is a functionCallTest expectedError whose queryFailure
// string is compared exactly.
type QueryFailureError struct {
	QueryFailure string
	Message      string
}

// Error implements error.
func (e QueryFailureError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.QueryFailure
}

func resolvePathValue(root any, path []any) (any, error) {
	acc := root
	for _, seg := range path {
		next, ok := indexPath(acc, seg)
		if !ok {
			return nil, fmt.Errorf("resultAccessPath segment %v not found", seg)
		}
		acc = next
	}
	return acc, nil
}

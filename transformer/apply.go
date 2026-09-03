package transformer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/miroir-framework/miroir/go/jzod"
)

const defaultTransformerInput = "defaultInput"

var mustachePattern = regexp.MustCompile(`{{\s*([^}]+?)\s*}}`)

// ApplyArgs is the dispatcher context for one transformer application
// (step, resolveBuildTransformersTo, params, context, model env).
type ApplyArgs struct {
	Step        string
	Path        []string
	Label       string
	Transformer any
	ResolveTo   string
	Env         any
	Params      map[string]any
	Context     map[string]any
}

// Apply runs xf at step ("build" or "runtime") with resolveBuildTransformersTo
// "value", matching transformer_extended_apply used by Templates / virtual
// attributes.
func Apply(step string, xf any, params, context map[string]any) (any, error) {
	return apply(ApplyArgs{
		Step:        step,
		Transformer: xf,
		ResolveTo:   "value",
		Env:         jzod.DefaultMiroirModelEnvironment(),
		Params:      params,
		Context:     context,
	})
}

// ApplyUnit is the unit MiroirTest apply path: resolveBuildTransformersTo
// "constantTransformer", matching transformer_extended_apply_wrapper.
// Failures are returned as maps with elementType "failure", not as errors.
func ApplyUnit(step string, xf any, params, context map[string]any, env any) (any, error) {
	if env == nil {
		env = jzod.DefaultMiroirModelEnvironment()
	}
	v, err := apply(ApplyArgs{
		Step:        step,
		Transformer: xf,
		ResolveTo:   "constantTransformer",
		Env:         env,
		Params:      params,
		Context:     context,
	})
	if f, ok := AsFailure(err); ok {
		return f.Map(), nil
	}
	if f, ok := AsFailure(v); ok {
		return f.Map(), nil
	}
	return v, err
}

func apply(a ApplyArgs) (out any, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			if f, ok := rec.(Failure); ok {
				out, err = f, f
				return
			}
			err = fmt.Errorf("%v", rec)
		}
	}()
	if a.Params == nil {
		a.Params = map[string]any{}
	}
	if a.Context == nil {
		a.Context = map[string]any{}
	}
	if a.ResolveTo == "" {
		a.ResolveTo = "value"
	}
	if a.Step == "" {
		a.Step = "runtime"
	}
	return applyValue(a)
}

func applyValue(a ApplyArgs) (any, error) {
	xf := a.Transformer
	if xf == nil {
		return nil, nil
	}
	if list, ok := xf.([]any); ok {
		return applyArray(a, list)
	}
	m, ok := asMap(xf)
	if !ok || m == nil {
		return xf, nil
	}
	if _, hasType := m["transformerType"]; !hasType {
		return applyPlainObject(a, m)
	}

	interpRaw := m["interpolation"]
	newResolve := a.ResolveTo
	var interpTruth any
	if interpRaw == nil {
		interpTruth = a.Step == "build"
	} else {
		interpTruth = interpRaw
	}
	if jsTruthy(interpTruth) && a.ResolveTo == "constantTransformer" {
		newResolve = "value"
	}
	child := a
	child.ResolveTo = newResolve

	interp := "build"
	if s, ok := interpRaw.(string); ok && s != "" {
		interp = s
	}
	if a.Step == "runtime" || interp == "build" {
		return dispatchTyped(child, m)
	}
	return xf, nil
}

func dispatchTyped(a ApplyArgs, xf map[string]any) (any, error) {
	tt, _ := xf["transformerType"].(string)
	defs := Definitions()
	rawDef, ok := defs[tt]
	if !ok {
		return nil, Failure{
			QueryFailure:    "TransformerNotFound",
			TransformerPath: pathAny(a.Path),
			FailureOrigin:   []any{"transformer_extended_apply"},
			QueryContext:    "transformer " + tt + " does not exist",
			QueryParameters: xf,
		}
	}
	def, _ := asMap(rawDef)
	impl, _ := asMap(def["transformerImplementation"])
	if impl == nil {
		return nil, Failure{
			QueryFailure:    "FailedTransformer",
			TransformerPath: pathAny(a.Path),
			FailureOrigin:   []any{"transformer_extended_apply"},
			QueryContext:    "transformerImplementation for transformer" + fmt.Sprint(xf) + " not found",
			QueryParameters: xf,
		}
	}
	switch impl["transformerImplementationType"] {
	case "libraryImplementation":
		fnName, _ := impl["inMemoryImplementationFunctionName"].(string)
		h := lookupHandler(fnName, tt)
		if h == nil {
			return nil, Failure{
				QueryFailure:    "TransformerNotFound",
				TransformerPath: pathAny(a.Path),
				FailureOrigin:   []any{"transformer_extended_apply"},
				QueryContext:    "transformerImplementation " + fnName + " not found",
				QueryParameters: xf,
			}
		}
		return h(a, xf)
	case "transformer":
		return applyComposite(a, xf, def)
	default:
		return nil, Failure{
			QueryFailure:    "FailedTransformer",
			TransformerPath: pathAny(a.Path),
			FailureOrigin:   []any{"transformer_extended_apply"},
			QueryContext:    "transformerImplementation " + fmt.Sprint(impl["transformerImplementationType"]) + " not found",
			QueryParameters: xf,
		}
	}
}

func applyComposite(a ApplyArgs, xf, def map[string]any) (any, error) {
	iface, _ := asMap(def["transformerInterface"])
	if iface == nil {
		return nil, Failure{
			QueryFailure:    "FailedTransformer",
			TransformerPath: pathAny(a.Path),
			FailureOrigin:   []any{"transformer_extended_apply"},
			QueryContext:    "transformer " + transformerType(xf) + " not found",
			QueryParameters: xf,
		}
	}
	paramSchema, _ := asMap(iface["transformerParameterSchema"])
	td, _ := asMap(paramSchema["transformerDefinition"])
	fields, _ := asMap(td["definition"])
	evaluated := map[string]any{}
	for _, name := range KeysOfOrSorted(fields) {
		v, err := applyValue(childArgs(a, name, xf[name]))
		if err != nil {
			return nil, err
		}
		if isFailed(v) {
			return Failure{
				QueryFailure:    "FailedTransformer",
				TransformerPath: pathAny(a.Path),
				FailureOrigin:   []any{"transformer_extended_apply"},
				QueryContext:    "errors in parameters for transformer " + transformerType(xf),
				QueryParameters: xf,
			}, nil
		}
		evaluated[name] = v
	}
	impl, _ := asMap(def["transformerImplementation"])
	ctx := copyMap(a.Context)
	for k, v := range evaluated {
		ctx[k] = v
	}
	inner := a
	inner.Step = "runtime"
	inner.Context = ctx
	inner.Transformer = impl["definition"]
	inner.Path = append(append([]string{}, a.Path...), "transformerImplementation")
	return applyValue(inner)
}

func applyArray(a ApplyArgs, list []any) (any, error) {
	out := make([]any, len(list))
	for i, item := range list {
		v, err := applyValue(childArgs(a, fmt.Sprint(i), item))
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func applyPlainObject(a ApplyArgs, m map[string]any) (any, error) {
	out := map[string]any{}
	for _, k := range KeysOfOrSorted(m) {
		v, err := applyValue(childArgs(a, k, m[k]))
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	jzod.RememberKeys(out, KeysOfOrSorted(m))
	return out, nil
}

func childArgs(a ApplyArgs, name string, xf any) ApplyArgs {
	c := a
	c.Transformer = xf
	c.Label = name
	c.Path = append(append([]string{}, a.Path...), name)
	return c
}

func lookupHandler(fnName, tt string) handler {
	if h, ok := handlers[fnName]; ok {
		return h
	}
	if h, ok := handlers[tt]; ok {
		return h
	}
	return nil
}

// KeysOfOrSorted returns [jzod.RememberedKeys] when valid, otherwise sorted
// keys. Used wherever TypeScript Object.keys / insertion order is observable.
func KeysOfOrSorted(m map[string]any) []string {
	if keys, ok := jzod.RememberedKeys(m); ok {
		return keys
	}
	return sortedKeys(m)
}

func renderMustache(template string, bank map[string]any) string {
	if bank == nil {
		bank = map[string]any{}
	}
	return mustachePattern.ReplaceAllStringFunc(template, func(match string) string {
		sub := mustachePattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		key := strings.TrimSpace(sub[1])
		v, ok := lookupDotted(bank, key)
		if !ok || v == nil {
			return ""
		}
		return mustachePrint(v)
	})
}

func lookupDotted(bank map[string]any, key string) (any, bool) {
	if bank == nil {
		return nil, false
	}
	if v, ok := bank[key]; ok {
		return v, true
	}
	acc := any(bank)
	for _, part := range strings.Split(key, ".") {
		m, ok := asMap(acc)
		if !ok {
			return nil, false
		}
		v, exists := m[part]
		if !exists {
			return nil, false
		}
		acc = v
	}
	return acc, true
}

func mustachePrint(v any) string {
	switch n := v.(type) {
	case float64:
		if n == float64(int64(n)) {
			return fmt.Sprintf("%d", int64(n))
		}
		return fmt.Sprint(n)
	case float32:
		if float64(n) == float64(int64(n)) {
			return fmt.Sprintf("%d", int64(n))
		}
		return fmt.Sprint(n)
	default:
		return fmt.Sprint(v)
	}
}

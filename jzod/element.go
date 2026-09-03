package jzod

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/miroir-framework/miroir/go/jzod/generated"
)

type unsupportedTypeError struct {
	kind string
}

func (e *unsupportedTypeError) Error() string {
	return "jzodTypeCheck unsupported type " + e.kind
}

// AsElement converts a JSON-decoded schema (or an already-typed value) into a
// generated JzodElement. Discriminated-union validity is not checked here;
// TypeCheck does that.
func AsElement(v any) (generated.JzodElement, error) {
	if v == nil {
		return nil, fmt.Errorf("jzodTypeCheck expected a schema object")
	}
	if el, ok := v.(generated.JzodElement); ok {
		return el, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("jzodTypeCheck expected a schema object")
	}
	kind, _ := m["type"].(string)
	if kind == "" {
		return nil, fmt.Errorf("jzodTypeCheck expected a schema object")
	}
	switch kind {
	case "literal":
		var out generated.JzodLiteral
		if err := unmarshalInto(m, &out); err != nil {
			return nil, err
		}
		return out, nil
	case "enum":
		var out generated.JzodEnum
		if err := unmarshalInto(m, &out); err != nil {
			return nil, err
		}
		return out, nil
	case "string":
		if m["validations"] != nil {
			var out generated.JzodAttributePlainStringWithValidations
			if err := unmarshalInto(m, &out); err != nil {
				return nil, err
			}
			return out, nil
		}
		return convertPlain(m)
	case "number":
		if m["validations"] != nil {
			var out generated.JzodAttributePlainNumberWithValidations
			if err := unmarshalInto(m, &out); err != nil {
				return nil, err
			}
			return out, nil
		}
		return convertPlain(m)
	case "date":
		if m["validations"] != nil {
			var out generated.JzodAttributePlainDateWithValidations
			if err := unmarshalInto(m, &out); err != nil {
				return nil, err
			}
			return out, nil
		}
		return convertPlain(m)
	case "boolean", "any", "uuid", "bigint", "undefined", "never", "unknown", "void", "null":
		return convertPlain(m)
	case "object":
		return convertObject(m)
	case "schemaReference":
		return convertReference(m)
	case "union":
		return convertUnion(m)
	case "array":
		return convertArray(m)
	case "tuple":
		return convertTuple(m)
	case "record":
		return convertRecord(m)
	case "set":
		return convertSet(m)
	case "lazy":
		return convertLazy(m)
	case "promise":
		return convertPromise(m)
	case "map":
		return convertMap(m)
	case "function":
		return convertFunction(m)
	case "intersection":
		return convertIntersection(m)
	default:
		return nil, &unsupportedTypeError{kind: kind}
	}
}

func convertPlain(m map[string]any) (generated.JzodPlainAttribute, error) {
	var out generated.JzodPlainAttribute
	if err := unmarshalInto(m, &out); err != nil {
		return out, err
	}
	return out, nil
}

func convertObject(m map[string]any) (generated.JzodObject, error) {
	var out generated.JzodObject
	if err := unmarshalInto(m, &out, "definition", "extend"); err != nil {
		return out, err
	}
	if def, ok := m["definition"].(map[string]any); ok {
		converted, err := convertElementMap(def)
		if err != nil {
			return out, err
		}
		out.Definition = converted
	}
	if m["extend"] != nil {
		ext := m["extend"]
		out.Extend = &ext
	}
	return out, nil
}

func convertReference(m map[string]any) (generated.JzodReference, error) {
	var out generated.JzodReference
	if err := unmarshalInto(m, &out, "context"); err != nil {
		return out, err
	}
	if ctx, ok := m["context"].(map[string]any); ok {
		converted, err := convertElementMap(ctx)
		if err != nil {
			return out, err
		}
		out.Context = &converted
	}
	return out, nil
}

func convertUnion(m map[string]any) (generated.JzodUnion, error) {
	var out generated.JzodUnion
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	list, _ := m["definition"].([]any)
	converted, err := convertElementSlice(list)
	if err != nil {
		return out, err
	}
	out.Definition = converted
	return out, nil
}

func convertArray(m map[string]any) (generated.JzodArray, error) {
	var out generated.JzodArray
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	inner, err := AsElement(m["definition"])
	if err != nil {
		return out, err
	}
	out.Definition = inner
	return out, nil
}

func convertTuple(m map[string]any) (generated.JzodTuple, error) {
	var out generated.JzodTuple
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	list, _ := m["definition"].([]any)
	converted, err := convertElementSlice(list)
	if err != nil {
		return out, err
	}
	out.Definition = converted
	return out, nil
}

func convertRecord(m map[string]any) (generated.JzodRecord, error) {
	var out generated.JzodRecord
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	inner, err := AsElement(m["definition"])
	if err != nil {
		return out, err
	}
	out.Definition = inner
	return out, nil
}

func convertSet(m map[string]any) (generated.JzodSet, error) {
	var out generated.JzodSet
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	inner, err := AsElement(m["definition"])
	if err != nil {
		return out, err
	}
	out.Definition = inner
	return out, nil
}

func convertLazy(m map[string]any) (generated.JzodLazy, error) {
	var out generated.JzodLazy
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	if m["definition"] == nil {
		return out, nil
	}
	inner, err := AsElement(m["definition"])
	if err != nil {
		return out, err
	}
	if fn, ok := unwrapElement(inner).(generated.JzodFunction); ok {
		out.Definition = &fn
	}
	return out, nil
}

func convertPromise(m map[string]any) (generated.JzodPromise, error) {
	var out generated.JzodPromise
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	inner, err := AsElement(m["definition"])
	if err != nil {
		return out, err
	}
	out.Definition = inner
	return out, nil
}

func convertMap(m map[string]any) (generated.JzodMap, error) {
	var out generated.JzodMap
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	list, _ := m["definition"].([]any)
	if len(list) >= 1 {
		e0, err := AsElement(list[0])
		if err != nil {
			return out, err
		}
		out.Definition.E0 = e0
	}
	if len(list) >= 2 {
		e1, err := AsElement(list[1])
		if err != nil {
			return out, err
		}
		out.Definition.E1 = e1
	}
	return out, nil
}

func convertFunction(m map[string]any) (generated.JzodFunction, error) {
	var out generated.JzodFunction
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	def, _ := m["definition"].(map[string]any)
	if args, ok := def["args"].([]any); ok {
		converted, err := convertElementSlice(args)
		if err != nil {
			return out, err
		}
		out.Definition.Args = converted
	}
	if def["returns"] != nil {
		ret, err := AsElement(def["returns"])
		if err != nil {
			return out, err
		}
		out.Definition.Returns = ret
	}
	return out, nil
}

func convertIntersection(m map[string]any) (generated.JzodIntersection, error) {
	var out generated.JzodIntersection
	if err := unmarshalInto(m, &out, "definition"); err != nil {
		return out, err
	}
	def, _ := m["definition"].(map[string]any)
	if def["left"] != nil {
		left, err := AsElement(def["left"])
		if err != nil {
			return out, err
		}
		out.Definition.Left = left
	}
	if def["right"] != nil {
		right, err := AsElement(def["right"])
		if err != nil {
			return out, err
		}
		out.Definition.Right = right
	}
	return out, nil
}

func convertElementMap(m map[string]any) (map[string]generated.JzodElement, error) {
	out := make(map[string]generated.JzodElement, len(m))
	for _, k := range KeysOf(m) {
		el, err := AsElement(m[k])
		if err != nil {
			return nil, err
		}
		out[k] = el
	}
	return out, nil
}

func convertElementSlice(list []any) ([]generated.JzodElement, error) {
	out := make([]generated.JzodElement, 0, len(list))
	for _, item := range list {
		el, err := AsElement(item)
		if err != nil {
			return nil, err
		}
		out = append(out, el)
	}
	return out, nil
}

func unmarshalInto(m map[string]any, dest any, skip ...string) error {
	filtered := m
	if len(skip) > 0 {
		omit := map[string]bool{}
		for _, k := range skip {
			omit[k] = true
		}
		filtered = make(map[string]any, len(m))
		for k, v := range m {
			if !omit[k] {
				filtered[k] = v
			}
		}
	}
	raw, err := json.Marshal(filtered)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func unwrapElement(el generated.JzodElement) generated.JzodElement {
	if el == nil {
		return nil
	}
	v := reflect.ValueOf(el)
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		inner := v.Elem().Interface()
		if e, ok := inner.(generated.JzodElement); ok {
			return e
		}
	}
	return el
}

// ElementKind returns the Jzod `type` string of el.
func ElementKind(el generated.JzodElement) string {
	if el == nil {
		return ""
	}
	v := reflect.ValueOf(el)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	f := v.FieldByName("Type")
	if f.IsValid() && f.Kind() == reflect.String {
		return f.String()
	}
	return ""
}

// ElementOptional reports whether el is marked optional.
func ElementOptional(el generated.JzodElement) bool {
	return boolField(el, "Optional")
}

// ElementNullable reports whether el is marked nullable.
func ElementNullable(el generated.JzodElement) bool {
	return boolField(el, "Nullable")
}

func boolField(el generated.JzodElement, name string) bool {
	if el == nil {
		return false
	}
	v := reflect.ValueOf(el)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	f := v.FieldByName(name)
	if !f.IsValid() {
		return false
	}
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return false
		}
		f = f.Elem()
	}
	return f.Kind() == reflect.Bool && f.Bool()
}

func contextFromRef(ref generated.JzodReference) map[string]any {
	if ref.Context == nil {
		return nil
	}
	out := make(map[string]any, len(*ref.Context))
	for k, v := range *ref.Context {
		out[k] = v
	}
	return out
}

func ElementToMap(el generated.JzodElement) (map[string]any, error) {
	raw, err := json.Marshal(el)
	if err != nil {
		return nil, err
	}
	decoded, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	m, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("element did not marshal to object")
	}
	return m, nil
}

func asElementError(schema any, err error, valuePath, typePath []any, value any) Result {
	if u, ok := err.(*unsupportedTypeError); ok {
		return errorResult(u.Error(), u.kind, valuePath, typePath, value, schema)
	}
	m, ok := schema.(map[string]any)
	if !ok {
		return errorResult("jzodTypeCheck expected a schema object", "", valuePath, typePath, value, schema)
	}
	kind, _ := m["type"].(string)
	if kind == "" {
		return errorResult("jzodTypeCheck expected a schema object", "", valuePath, typePath, value, schema)
	}
	return errorResult("jzodTypeCheck unsupported type "+kind, kind, valuePath, typePath, value, schema)
}

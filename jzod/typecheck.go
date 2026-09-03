package jzod

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/miroir-framework/miroir/go/jzod/generated"
)

// ModelEnvironment holds absolute-reference MlSchemas for schemaReference lookup.
// [EnvironmentFromMap] registers bootstrap / fundamental schemas here.
type ModelEnvironment struct {
	AbsoluteSchemas []map[string]any
}

// Result is the Go shape of TypeScript ResolvedJzodSchemaReturnType
// (status ok|error, rawSchema, resolvedSchema, keyMap, paths).
type Result struct {
	Status              string         `json:"status"`
	SchemaReferenceName string         `json:"schemaReferenceName,omitempty"`
	ValuePath           []any          `json:"valuePath"`
	TypePath            []any          `json:"typePath"`
	RawSchema           any            `json:"rawSchema"`
	ResolvedSchema      any            `json:"resolvedSchema,omitempty"`
	KeyMap              map[string]any `json:"keyMap,omitempty"`
	Error               string         `json:"error,omitempty"`
	RawJzodSchemaType   string         `json:"rawJzodSchemaType,omitempty"`
	Value               any            `json:"value,omitempty"`
}

// TypeCheck checks value against a Jzod schema. Semantics follow miroir-core
// jzodTypeCheck (packages/miroir-core/src/1_core/jzod/jzodTypeCheck.ts).
// schema may be JSON-decoded map[string]any or a generated.JzodElement.
func TypeCheck(
	schema any,
	value any,
	valuePath []any,
	typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
) Result {
	return typeCheck(schema, value, copyPath(valuePath), copyPath(typePath), env, relativeContext, "")
}

func typeCheck(
	schema any,
	value any,
	valuePath []any,
	typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
	schemaReferenceName string,
) Result {
	el, err := AsElement(schema)
	if err != nil {
		return asElementError(schema, err, valuePath, typePath, value)
	}
	el = unwrapElement(el)
	kind := ElementKind(el)

	if value == nil {
		if !ElementOptional(el) && !ElementNullable(el) && kind != "any" && kind != "undefined" {
			return errorResult("jzodTypeCheck expected a value but got null for non-optional schema", kind, valuePath, typePath, value, schema)
		}
		resolved := schema
		if kind == "any" {
			resolved = valueToJzod(value)
		}
		return okResult(schema, resolved, valuePath, typePath, schemaReferenceName)
	}

	switch e := el.(type) {
	case generated.JzodLiteral:
		if valuesEqual(value, e.Definition) {
			return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
		}
		return mismatch("literal", valuePath, typePath, value, schema)
	case generated.JzodPlainAttribute:
		return typeCheckPlain(e, schema, value, valuePath, typePath, schemaReferenceName)
	case generated.JzodAttributePlainStringWithValidations:
		if _, ok := value.(string); !ok {
			return mismatch("string", valuePath, typePath, value, schema)
		}
		return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
	case generated.JzodAttributePlainNumberWithValidations:
		if !isJSONNumber(value) {
			return mismatch("number", valuePath, typePath, value, schema)
		}
		return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
	case generated.JzodAttributePlainDateWithValidations:
		return typeCheckDate(schema, value, valuePath, typePath, schemaReferenceName)
	case generated.JzodReference:
		return typeCheckSchemaReference(e, schema, value, valuePath, typePath, env, relativeContext)
	case generated.JzodObject:
		return typeCheckObject(e, schema, value, valuePath, typePath, env, relativeContext, schemaReferenceName)
	case generated.JzodUnion:
		return typeCheckUnion(e, schema, value, valuePath, typePath, env, relativeContext, schemaReferenceName)
	case generated.JzodEnum:
		if enumContains(e.Definition, value) {
			return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
		}
		return mismatch("enum", valuePath, typePath, value, schema)
	case generated.JzodArray:
		return typeCheckArray(e, schema, value, valuePath, typePath, env, relativeContext, schemaReferenceName)
	case generated.JzodTuple:
		return typeCheckTuple(e, schema, value, valuePath, typePath, env, relativeContext, schemaReferenceName)
	case generated.JzodRecord:
		return typeCheckRecord(e, schema, value, valuePath, typePath, env, relativeContext, schemaReferenceName)
	case generated.JzodIntersection, generated.JzodPromise, generated.JzodSet, generated.JzodFunction, generated.JzodMap, generated.JzodLazy:
		return stubAccept(schema, valuePath, typePath, schemaReferenceName)
	default:
		return errorResult("jzodTypeCheck unsupported type "+kind, kind, valuePath, typePath, value, schema)
	}
}

func typeCheckPlain(
	e generated.JzodPlainAttribute,
	schema, value any,
	valuePath, typePath []any,
	schemaReferenceName string,
) Result {
	switch e.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return mismatch("string", valuePath, typePath, value, schema)
		}
		return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
	case "boolean":
		if _, ok := value.(bool); !ok {
			return mismatch("boolean", valuePath, typePath, value, schema)
		}
		return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
	case "number":
		if !isJSONNumber(value) {
			return mismatch("number", valuePath, typePath, value, schema)
		}
		return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
	case "any":
		return typeCheckAny(schema, value, valuePath, typePath, schemaReferenceName)
	case "uuid":
		s, ok := value.(string)
		if !ok || !uuidV4Pattern.MatchString(s) {
			return mismatch("uuid", valuePath, typePath, value, schema)
		}
		return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
	case "bigint":
		return mismatch("bigint", valuePath, typePath, value, schema)
	case "date":
		return typeCheckDate(schema, value, valuePath, typePath, schemaReferenceName)
	case "undefined", "never", "unknown", "void":
		return stubAccept(schema, valuePath, typePath, schemaReferenceName)
	default:
		return errorResult("jzodTypeCheck unsupported type "+e.Type, e.Type, valuePath, typePath, value, schema)
	}
}

func stubAccept(schema any, valuePath, typePath []any, schemaReferenceName string) Result {
	return Result{
		Status:              "ok",
		SchemaReferenceName: schemaReferenceName,
		ValuePath:           valuePath,
		TypePath:            []any{},
		RawSchema:           schema,
		ResolvedSchema:      schema,
		KeyMap: map[string]any{
			joinPath(valuePath): map[string]any{
				"rawSchema":      schema,
				"resolvedSchema": schema,
				"valuePath":      valuePath,
				"typePath":       typePath,
			},
		},
	}
}

func typeCheckSchemaReference(
	ref generated.JzodReference,
	raw any,
	value any,
	valuePath, typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
) Result {
	newContext := mergeContext(relativeContext, contextFromRef(ref))
	resolved, err := resolveElement(ref, newContext, env)
	if err != nil {
		return errorResult("jzodTypeCheck failed to resolve schemaReference", "schemaReference", valuePath, typePath, value, raw)
	}
	rel := ref.Definition.RelativePath
	if rel == "" {
		rel = "NO_RELATIVE_PATH"
	}
	inner := typeCheck(resolved, value, valuePath, appendPath(typePath, "ref:"+rel), env, newContext, rel)
	if inner.Status == "error" {
		return errorResult("jzodTypeCheck failed to resolve schemaReference", "schemaReference", valuePath, typePath, value, raw)
	}
	inner.SchemaReferenceName = rel
	inner.RawSchema = raw
	if inner.KeyMap == nil {
		inner.KeyMap = map[string]any{}
	}
	inner.KeyMap[joinPath(valuePath)] = map[string]any{
		"rawSchema":                        raw,
		"resolvedReferenceSchemaInContext": resolved,
		"resolvedSchema":                   inner.ResolvedSchema,
		"valuePath":                        valuePath,
		"typePath":                         typePath,
	}
	return inner
}

func resolveElement(ref generated.JzodReference, relativeContext map[string]any, env ModelEnvironment) (generated.JzodElement, error) {
	target, err := lookupRef(ref, relativeContext, env)
	if err != nil {
		return nil, err
	}
	el, err := AsElement(target)
	if err != nil {
		return nil, err
	}
	el = unwrapElement(el)
	if next, ok := el.(generated.JzodReference); ok {
		return resolveElement(next, mergeContext(relativeContext, contextFromRef(next)), env)
	}
	return el, nil
}

func lookupRef(ref generated.JzodReference, relativeContext map[string]any, env ModelEnvironment) (any, error) {
	lookup := relativeContext
	if ref.Definition.AbsolutePath != nil && *ref.Definition.AbsolutePath != "" {
		found := false
		lookup, found = env.contextForUUID(*ref.Definition.AbsolutePath)
		if !found {
			return nil, fmt.Errorf("absolutePath %s not registered", *ref.Definition.AbsolutePath)
		}
	}
	rel := ref.Definition.RelativePath
	if rel == "" {
		return nil, fmt.Errorf("schemaReference missing relativePath")
	}
	target, ok := lookup[rel]
	if !ok {
		return nil, fmt.Errorf("unresolved relativePath %s", rel)
	}
	return target, nil
}

func typeCheckObject(
	objSchema generated.JzodObject,
	raw any,
	value any,
	valuePath, typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
	schemaReferenceName string,
) Result {
	obj, ok := value.(map[string]any)
	if !ok {
		return errorResult("jzodTypeCheck failed for object schema to match non-object value", "object", valuePath, typePath, value, raw)
	}
	schemaMap, err := schemaMapForObject(raw, objSchema)
	if err != nil {
		return errorResult("jzodTypeCheck failed to flatten object extend: "+err.Error(), "object", valuePath, typePath, value, raw)
	}
	flattened, err := flattenObject(schemaMap, relativeContext, env)
	if err != nil {
		return errorResult("jzodTypeCheck failed to flatten object extend: "+err.Error(), "object", valuePath, typePath, value, raw)
	}
	flatEl, err := AsElement(flattened)
	if err != nil {
		return errorResult("jzodTypeCheck failed to flatten object extend: "+err.Error(), "object", valuePath, typePath, value, raw)
	}
	flatObj, ok := unwrapElement(flatEl).(generated.JzodObject)
	if !ok {
		return errorResult("jzodTypeCheck failed to flatten object extend: flattened schema is not an object", "object", valuePath, typePath, value, raw)
	}
	definition := flatObj.Definition
	if definition == nil {
		definition = map[string]generated.JzodElement{}
	}
	nonStrict := false
	if flatObj.NonStrict != nil {
		nonStrict = *flatObj.NonStrict
	}
	resolvedDef := map[string]any{}
	for key, attrValue := range obj {
		attrSchema, found := definition[key]
		if !found {
			if nonStrict {
				resolvedDef[key] = map[string]any{"type": "any"}
				continue
			}
			return errorResult("jzodTypeCheck failed to match some object value attribute(s) with the schema of that attribute(s)", "object", valuePath, typePath, value, raw)
		}
		attrResult := typeCheck(attrSchema, attrValue, appendPath(valuePath, key), appendPath(typePath, key), env, relativeContext, "")
		if attrResult.Status == "error" {
			return errorResult("jzodTypeCheck failed to match some object value attribute(s) with the schema of that attribute(s)", "object", valuePath, typePath, value, raw)
		}
		resolvedDef[key] = attrResult.ResolvedSchema
	}
	for key, attrSchema := range definition {
		if _, present := obj[key]; present {
			continue
		}
		if ElementOptional(attrSchema) || ElementNullable(attrSchema) {
			continue
		}
		return errorResult("jzodTypeCheck failed to match some mandatory object value attribute(s) with the schema of that attribute(s)", "object", valuePath, typePath, value, raw)
	}
	resolvedSchema := map[string]any{
		"type":       "object",
		"definition": resolvedDef,
	}
	return okResult(raw, resolvedSchema, valuePath, typePath, schemaReferenceName)
}

func schemaMapForObject(raw any, obj generated.JzodObject) (map[string]any, error) {
	if m, ok := raw.(map[string]any); ok {
		return m, nil
	}
	return ElementToMap(obj)
}

var uuidV4Pattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func typeCheckArray(
	arr generated.JzodArray,
	raw any,
	value any,
	valuePath, typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
	schemaReferenceName string,
) Result {
	list, ok := value.([]any)
	if !ok {
		return mismatch("array", valuePath, typePath, value, raw)
	}
	items := make([]any, len(list))
	for i, item := range list {
		r := typeCheck(arr.Definition, item, appendPath(valuePath, i), appendPath(typePath, i), env, relativeContext, "")
		if r.Status == "error" {
			return errorResult("jzodTypeCheck failed to match value with array schema", "array", valuePath, typePath, value, raw)
		}
		items[i] = r.ResolvedSchema
	}
	return okResult(raw, map[string]any{"type": "tuple", "definition": items}, valuePath, typePath, schemaReferenceName)
}

func typeCheckTuple(
	tup generated.JzodTuple,
	raw any,
	value any,
	valuePath, typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
	schemaReferenceName string,
) Result {
	list, ok := value.([]any)
	if !ok {
		return mismatch("tuple", valuePath, typePath, value, raw)
	}
	defs := tup.Definition
	if len(list) != len(defs) {
		return mismatch("tuple", valuePath, typePath, value, raw)
	}
	items := make([]any, len(list))
	for i, item := range list {
		r := typeCheck(defs[i], item, appendPath(valuePath, i), appendPath(typePath, i), env, relativeContext, "")
		if r.Status == "error" {
			return errorResult("jzodTypeCheck failed to match value with tuple schema", "tuple", valuePath, typePath, value, raw)
		}
		items[i] = r.ResolvedSchema
	}
	return okResult(raw, map[string]any{"type": "tuple", "definition": items}, valuePath, typePath, schemaReferenceName)
}

func typeCheckRecord(
	rec generated.JzodRecord,
	raw any,
	value any,
	valuePath, typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
	schemaReferenceName string,
) Result {
	obj, ok := value.(map[string]any)
	if !ok {
		return errorResult("jzodTypeCheck record schema for value is not an object", "record", valuePath, typePath, value, raw)
	}
	elemSchema := rec.Definition
	if _, isUnion := unwrapElement(elemSchema).(generated.JzodUnion); isUnion {
		elemSchema = generated.JzodPlainAttribute{Type: "any"}
	}
	resolvedDef := map[string]any{}
	for key, attrValue := range obj {
		attrResult := typeCheck(elemSchema, attrValue, appendPath(valuePath, key), appendPath(typePath, key), env, relativeContext, "")
		if attrResult.Status == "error" {
			return errorResult("jzodTypeCheck failed to match value with record schema", "record", valuePath, typePath, value, raw)
		}
		resolvedDef[key] = attrResult.ResolvedSchema
	}
	return okResult(raw, map[string]any{"type": "object", "definition": resolvedDef}, valuePath, typePath, schemaReferenceName)
}

func flattenObject(schema map[string]any, relativeContext map[string]any, env ModelEnvironment) (map[string]any, error) {
	if schema["extend"] == nil {
		return schema, nil
	}
	parent, err := extendProperties(schema["extend"], relativeContext, env)
	if err != nil {
		return nil, err
	}
	own, _ := schema["definition"].(map[string]any)
	out := map[string]any{
		"type":       "object",
		"definition": mergeContext(parent, own),
	}
	for _, key := range []string{"optional", "nullable", "tag", "nonStrict"} {
		if v, ok := schema[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}

func extendProperties(extend any, relativeContext map[string]any, env ModelEnvironment) (map[string]any, error) {
	switch e := extend.(type) {
	case []any:
		out := map[string]any{}
		for _, item := range e {
			part, err := extendProperties(item, relativeContext, env)
			if err != nil {
				return nil, err
			}
			out = mergeContext(out, part)
		}
		return out, nil
	case map[string]any:
		switch e["type"] {
		case "schemaReference":
			resolved, err := recursiveResolve(e, mergeContext(relativeContext, contextOf(e)), env)
			if err != nil {
				return nil, err
			}
			resolvedMap, ok := asSchemaMap(resolved)
			if !ok {
				return nil, fmt.Errorf("extend schemaReference resolved to non-object")
			}
			return extendProperties(resolvedMap, relativeContext, env)
		case "object":
			own, _ := e["definition"].(map[string]any)
			if e["extend"] == nil {
				if own == nil {
					return map[string]any{}, nil
				}
				return own, nil
			}
			parent, err := extendProperties(e["extend"], relativeContext, env)
			if err != nil {
				return nil, err
			}
			return mergeContext(parent, own), nil
		default:
			return nil, fmt.Errorf("extend clause type %v cannot flatten", e["type"])
		}
	default:
		return nil, fmt.Errorf("unsupported extend clause")
	}
}

func typeCheckDate(schema any, value any, valuePath, typePath []any, schemaReferenceName string) Result {
	if isJSDate(value) {
		return okResult(schema, schema, valuePath, typePath, schemaReferenceName)
	}
	return errorResult(
		fmt.Sprintf("jzodTypeCheck failed to match value with date schema. %s could not be converted to Date. Value: %s", jsTypeof(value), jsonStringify(value)),
		"date",
		valuePath,
		typePath,
		value,
		schema,
	)
}

func isJSDate(value any) bool {
	switch v := value.(type) {
	case string:
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02", "2006-01-02T15:04:05", time.RFC1123} {
			if _, err := time.Parse(layout, v); err == nil {
				return true
			}
		}
		return false
	case float64, json.Number, int, int32, int64:
		return true
	default:
		return false
	}
}

func jsTypeof(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, json.Number, int, int32, int64:
		return "number"
	default:
		return "object"
	}
}

func jsonStringify(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(raw)
}

func typeCheckAny(schema, value any, valuePath, typePath []any, schemaReferenceName string) Result {
	resolved := valueToJzod(value)
	result := okResult(schema, resolved, valuePath, typePath, schemaReferenceName)
	if obj, ok := value.(map[string]any); ok {
		for k, v := range buildAnySubnodeKeyMap(obj, valuePath, typePath) {
			result.KeyMap[k] = v
		}
	}
	return result
}

func valueToJzod(value any) any {
	if value == nil {
		return map[string]any{"type": "null"}
	}
	switch v := value.(type) {
	case string:
		return map[string]any{"type": "string"}
	case bool:
		return map[string]any{"type": "boolean"}
	case float64, json.Number, int, int32, int64:
		return map[string]any{"type": "number"}
	case []any:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = valueToJzod(item)
		}
		return map[string]any{"type": "tuple", "definition": items}
	case map[string]any:
		def := make(map[string]any, len(v))
		for k, item := range v {
			def[k] = valueToJzod(item)
		}
		return map[string]any{"type": "object", "definition": def}
	default:
		return map[string]any{"type": "never"}
	}
}

// BuildAnyObjectEntry builds a keyMap entry for an object attribute whose
// schema is type any (valueToJzod of that child).
func BuildAnyObjectEntry(v map[string]any, childPath, childTypePath []any) map[string]any {
	return buildAnyObjectEntry(v, childPath, childTypePath)
}

// BuildAnySubnodeKeyMap builds a keyMap for every attribute of an object
// typed as any.
func BuildAnySubnodeKeyMap(obj map[string]any, basePath, baseTypePath []any) map[string]any {
	return buildAnySubnodeKeyMap(obj, basePath, baseTypePath)
}

func buildAnyObjectEntry(v map[string]any, childPath, childTypePath []any) map[string]any {
	entry := map[string]any{
		"rawSchema":      map[string]any{"type": "any"},
		"resolvedSchema": valueToJzod(v),
		"valuePath":      childPath,
		"typePath":       childTypePath,
	}
	for k2, v2 := range v {
		subPath := appendPath(childPath, k2)
		subTypePath := appendPath(childTypePath, k2)
		if nested, ok := v2.(map[string]any); ok {
			entry[k2] = buildAnyObjectEntry(nested, subPath, subTypePath)
			continue
		}
		entry[k2] = map[string]any{
			"rawSchema":      map[string]any{"type": "any"},
			"resolvedSchema": valueToJzod(v2),
			"valuePath":      subPath,
			"typePath":       subTypePath,
		}
	}
	return entry
}

func buildAnySubnodeKeyMap(obj map[string]any, basePath, baseTypePath []any) map[string]any {
	result := map[string]any{}
	for k, v := range obj {
		childPath := appendPath(basePath, k)
		childTypePath := appendPath(baseTypePath, k)
		flatKey := joinPath(childPath)
		if nested, ok := v.(map[string]any); ok {
			result[flatKey] = buildAnyObjectEntry(nested, childPath, childTypePath)
			continue
		}
		result[flatKey] = map[string]any{
			"rawSchema":      map[string]any{"type": "any"},
			"resolvedSchema": valueToJzod(v),
			"valuePath":      childPath,
			"typePath":       childTypePath,
		}
	}
	return result
}

func typeCheckUnion(
	union generated.JzodUnion,
	raw any,
	value any,
	valuePath, typePath []any,
	env ModelEnvironment,
	relativeContext map[string]any,
	schemaReferenceName string,
) Result {
	unfolded, err := unfoldUnion(union, map[string]bool{}, relativeContext, env)
	if err != nil {
		return errorResult("jzodTypeCheck failed to recursively unfold schema", "union", valuePath, typePath, value, raw)
	}
	if obj, isObj := value.(map[string]any); isObj {
		for _, b := range unfolded {
			if ElementKind(unwrapElement(b)) != "object" {
				continue
			}
			inner := typeCheck(b, obj, valuePath, typePath, env, relativeContext, "")
			if inner.Status == "ok" {
				inner.RawSchema = raw
				inner.SchemaReferenceName = schemaReferenceName
				return inner
			}
		}
		return errorResult("jzodTypeCheck failed to resolve union for object", "union", valuePath, typePath, value, raw)
	}
	chosen, ok := pickUnionBranch(unfolded, value)
	if !ok {
		return errorResult("jzodTypeCheck could not find type for value in resolved union", "union", valuePath, typePath, value, raw)
	}
	return okResult(raw, chosen, valuePath, typePath, schemaReferenceName)
}

func pickUnionBranch(branches []generated.JzodElement, value any) (any, bool) {
	switch value.(type) {
	case string:
		for _, b := range branches {
			el := unwrapElement(b)
			switch ElementKind(el) {
			case "any", "string", "uuid":
				return el, true
			case "literal":
				if lit, ok := el.(generated.JzodLiteral); ok && valuesEqual(value, lit.Definition) {
					return el, true
				}
			case "enum":
				if en, ok := el.(generated.JzodEnum); ok && enumContains(en.Definition, value) {
					return el, true
				}
			}
		}
	case bool:
		for _, b := range branches {
			el := unwrapElement(b)
			if ElementKind(el) == "boolean" {
				return el, true
			}
		}
	default:
		if isJSONNumber(value) {
			for _, b := range branches {
				el := unwrapElement(b)
				kind := ElementKind(el)
				if kind == "number" || kind == "bigint" {
					return el, true
				}
			}
		}
	}
	return nil, false
}

func unfoldUnion(union generated.JzodUnion, expanded map[string]bool, relativeContext map[string]any, env ModelEnvironment) ([]generated.JzodElement, error) {
	var result []generated.JzodElement
	var refs []generated.JzodReference
	var nested []generated.JzodUnion
	for _, raw := range union.Definition {
		switch b := unwrapElement(raw).(type) {
		case generated.JzodReference:
			refs = append(refs, b)
		case generated.JzodUnion:
			nested = append(nested, b)
		default:
			result = append(result, unwrapElement(raw))
		}
	}
	newExpanded := copyExpanded(expanded)
	for _, ref := range refs {
		rel := ref.Definition.RelativePath
		if rel != "" && newExpanded[rel] {
			continue
		}
		if rel != "" {
			newExpanded[rel] = true
		}
		resolved, err := resolveElement(ref, mergeContext(relativeContext, contextFromRef(ref)), env)
		if err != nil {
			return nil, err
		}
		resolved = unwrapElement(resolved)
		if u, ok := resolved.(generated.JzodUnion); ok {
			nested = append(nested, u)
			continue
		}
		result = append(result, resolved)
	}
	for _, u := range nested {
		sub, err := unfoldUnion(u, newExpanded, relativeContext, env)
		if err != nil {
			return nil, err
		}
		result = append(result, sub...)
	}
	return result, nil
}

func recursiveResolve(schema map[string]any, relativeContext map[string]any, env ModelEnvironment) (any, error) {
	resolved, err := resolveReference(schema, relativeContext, env)
	if err != nil {
		return nil, err
	}
	if m, ok := asSchemaMap(resolved); ok && m["type"] == "schemaReference" {
		return recursiveResolve(m, mergeContext(relativeContext, contextOf(m)), env)
	}
	return resolved, nil
}

func asSchemaMap(v any) (map[string]any, bool) {
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	if el, ok := v.(generated.JzodElement); ok {
		m, err := ElementToMap(unwrapElement(el))
		return m, err == nil
	}
	return nil, false
}

func resolveReference(schema map[string]any, relativeContext map[string]any, env ModelEnvironment) (any, error) {
	def := refDefinition(schema)
	abs, _ := def["absolutePath"].(string)
	rel, _ := def["relativePath"].(string)
	lookup := relativeContext
	if abs != "" {
		found := false
		lookup, found = env.contextForUUID(abs)
		if !found {
			return nil, fmt.Errorf("absolutePath %s not registered", abs)
		}
	}
	if rel == "" {
		return nil, fmt.Errorf("schemaReference missing relativePath")
	}
	target, ok := lookup[rel]
	if !ok {
		return nil, fmt.Errorf("unresolved relativePath %s", rel)
	}
	return target, nil
}

func (env ModelEnvironment) contextForUUID(uuid string) (map[string]any, bool) {
	for _, schema := range env.AbsoluteSchemas {
		if schema["uuid"] != uuid {
			continue
		}
		definition, _ := schema["definition"].(map[string]any)
		context, _ := definition["context"].(map[string]any)
		if context == nil {
			return map[string]any{}, true
		}
		return context, true
	}
	return nil, false
}

func okResult(rawSchema, resolvedSchema any, valuePath, typePath []any, schemaReferenceName string) Result {
	return Result{
		Status:              "ok",
		SchemaReferenceName: schemaReferenceName,
		ValuePath:           valuePath,
		TypePath:            typePath,
		RawSchema:           rawSchema,
		ResolvedSchema:      resolvedSchema,
		KeyMap: map[string]any{
			joinPath(valuePath): map[string]any{
				"rawSchema":      rawSchema,
				"resolvedSchema": resolvedSchema,
				"valuePath":      valuePath,
				"typePath":       typePath,
			},
		},
	}
}

func mismatch(schemaKind string, valuePath, typePath []any, value, rawSchema any) Result {
	return errorResult("jzodTypeCheck failed to match value with "+schemaKind+" schema", schemaKind, valuePath, typePath, value, rawSchema)
}

func errorResult(message, rawType string, valuePath, typePath []any, value, rawSchema any) Result {
	return Result{
		Status:            "error",
		Error:             message,
		RawJzodSchemaType: rawType,
		ValuePath:         valuePath,
		TypePath:          typePath,
		Value:             value,
		RawSchema:         rawSchema,
	}
}

func joinPath(path []any) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, len(path))
	for i, p := range path {
		switch v := p.(type) {
		case string:
			parts[i] = v
		case int:
			parts[i] = strconv.Itoa(v)
		case float64:
			parts[i] = strconv.Itoa(int(v))
		default:
			parts[i] = fmt.Sprint(v)
		}
	}
	return strings.Join(parts, ".")
}

func copyPath(p []any) []any {
	if p == nil {
		return []any{}
	}
	out := make([]any, len(p))
	copy(out, p)
	return out
}

func appendPath(p []any, seg any) []any {
	out := make([]any, len(p)+1)
	copy(out, p)
	out[len(p)] = seg
	return out
}

func valuesEqual(a, b any) bool {
	return a == b
}

func isJSONNumber(v any) bool {
	switch v.(type) {
	case float64, json.Number, int, int32, int64:
		return true
	default:
		return false
	}
}

func contextOf(schema map[string]any) map[string]any {
	c, _ := schema["context"].(map[string]any)
	return c
}

func refDefinition(schema map[string]any) map[string]any {
	d, _ := schema["definition"].(map[string]any)
	if d == nil {
		return map[string]any{}
	}
	return d
}

func mergeContext(base, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func enumContains(definition any, value any) bool {
	switch list := definition.(type) {
	case []any:
		for _, item := range list {
			if valuesEqual(item, value) {
				return true
			}
		}
	case []string:
		for _, item := range list {
			if valuesEqual(item, value) {
				return true
			}
		}
	}
	return false
}

func copyExpanded(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

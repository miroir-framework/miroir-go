package miroirtest

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	compositeKeySeparator = "|"
	compositeKeyEscape    = `\`
)

func registerEntityPrimaryKey() {
	registry["miroir-core/1_core/EntityPrimaryKey"] = map[string]Fn{
		"getEntityPrimaryKeyAttribute":  wrap1(getEntityPrimaryKeyAttribute),
		"getEntityPrimaryKeyAttributes": wrap1(getEntityPrimaryKeyAttributes),
		"entityHasCompositePrimaryKey":  wrap1(entityHasCompositePrimaryKey),
		"entityHasUuidPrimaryKey":       wrap1(entityHasUuidPrimaryKey),
		"serializeCompositeKeyValue":    wrap2(serializeCompositeKeyValue),
		"parseCompositeKeyValue":        wrap2(parseCompositeKeyValue),
		"getInstancePrimaryKeyValue":    wrap2(getInstancePrimaryKeyValue),
		"getForeignKeyValue":            wrap2(getForeignKeyValue),
		"instanceMatchesForeignKey":     wrap3(instanceMatchesForeignKey),
		"resolveInstanceParentUuid":     wrap2(resolveInstanceParentUuid),
	}
}

func wrap1(fn func(any) any) Fn {
	return func(args []any) (any, error) {
		return fn(argAt(args, 0)), nil
	}
}

func wrap2(fn func(any, any) any) Fn {
	return func(args []any) (any, error) {
		return fn(argAt(args, 0), argAt(args, 1)), nil
	}
}

func wrap3(fn func(any, any, any) any) Fn {
	return func(args []any) (any, error) {
		return fn(argAt(args, 0), argAt(args, 1), argAt(args, 2)), nil
	}
}

func argAt(args []any, i int) any {
	if i >= len(args) {
		return jsonUndefined{}
	}
	return args[i]
}

func asSource(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func idAttribute(source map[string]any) any {
	if v, ok := source["idAttribute"]; ok {
		return v
	}
	return nil
}

func getEntityPrimaryKeyAttribute(source any) any {
	attr := idAttribute(asSource(source))
	if attr == nil {
		return "uuid"
	}
	return attr
}

func getEntityPrimaryKeyAttributes(source any) any {
	attr := getEntityPrimaryKeyAttribute(source)
	if list, ok := attr.([]any); ok {
		return list
	}
	return []any{attr}
}

func entityHasCompositePrimaryKey(source any) any {
	_, ok := idAttribute(asSource(source)).([]any)
	return ok
}

func entityHasUuidPrimaryKey(source any) any {
	return getEntityPrimaryKeyAttribute(source) == "uuid"
}

func pkAttrStrings(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, len(t))
		for i, x := range t {
			out[i] = fmt.Sprint(x)
		}
		return out
	case []string:
		return t
	default:
		return []string{fmt.Sprint(v)}
	}
}

func instanceMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func escapeKeyComponent(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `|`, `\|`)
}

func unescapeKeyComponent(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			b.WriteByte(value[i+1])
			i++
			continue
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

func serializeCompositeKeyValue(pkAttributes any, instance any) any {
	attrs := pkAttrStrings(pkAttributes)
	inst := instanceMap(instance)
	if len(attrs) == 1 {
		return fmt.Sprint(inst[attrs[0]])
	}
	parts := make([]string, len(attrs))
	for i, attr := range attrs {
		parts[i] = escapeKeyComponent(fmt.Sprint(inst[attr]))
	}
	return strings.Join(parts, compositeKeySeparator)
}

func parseCompositeKeyValue(pkAttributes any, serializedKey any) any {
	attrs := pkAttrStrings(pkAttributes)
	key := fmt.Sprint(serializedKey)
	if len(attrs) == 1 {
		return map[string]any{attrs[0]: key}
	}
	parts := splitEscaped(key)
	out := map[string]any{}
	for i, attr := range attrs {
		part := ""
		if i < len(parts) {
			part = parts[i]
		}
		out[attr] = unescapeKeyComponent(part)
	}
	return out
}

func splitEscaped(serializedKey string) []string {
	var parts []string
	var current strings.Builder
	for i := 0; i < len(serializedKey); i++ {
		if serializedKey[i] == '\\' && i+1 < len(serializedKey) {
			current.WriteByte(serializedKey[i])
			current.WriteByte(serializedKey[i+1])
			i++
			continue
		}
		if serializedKey[i] == '|' {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(serializedKey[i])
	}
	parts = append(parts, current.String())
	return parts
}

func getInstancePrimaryKeyValue(source any, instance any) any {
	return serializeCompositeKeyValue(getEntityPrimaryKeyAttributes(source), instance)
}

func getForeignKeyValue(fkAttribute any, referenceObject any) any {
	ref := instanceMap(referenceObject)
	if list, ok := fkAttribute.([]any); ok {
		for _, attr := range list {
			if ref[fmt.Sprint(attr)] == nil {
				return jsonUndefined{}
			}
		}
		parts := make([]string, len(list))
		for i, attr := range list {
			parts[i] = escapeKeyComponent(fmt.Sprint(ref[fmt.Sprint(attr)]))
		}
		return strings.Join(parts, compositeKeySeparator)
	}
	key := fmt.Sprint(fkAttribute)
	val := ref[key]
	if val == nil {
		return jsonUndefined{}
	}
	return fmt.Sprint(val)
}

func instanceMatchesForeignKey(fkAttribute any, instance any, referenceValue any) any {
	inst := instanceMap(instance)
	ref := fmt.Sprint(referenceValue)
	if list, ok := fkAttribute.([]any); ok {
		parts := parseCompositeKeyValue(list, ref).(map[string]any)
		for _, attr := range list {
			a := fmt.Sprint(attr)
			if fmt.Sprint(inst[a]) != fmt.Sprint(parts[a]) {
				return false
			}
		}
		return true
	}
	return inst[fmt.Sprint(fkAttribute)] == referenceValue
}

func resolveInstanceParentUuid(instance any, payloadParentUuid any) any {
	inst := instanceMap(instance)
	if s, ok := inst["parentUuid"].(string); ok && s != "" {
		return s
	}
	switch p := payloadParentUuid.(type) {
	case jsonUndefined:
		// fallthrough
	case nil:
	case string:
		if p != "" {
			return p
		}
	}
	raw, _ := json.Marshal(inst)
	return map[string]any{
		"status":       "error",
		"errorType":    "FailedToResolveParentUuid",
		"errorMessage": fmt.Sprintf("Could not resolve parentUuid for instance %s: neither instance.parentUuid nor action payload.parentUuid is defined.", raw),
	}
}

func init() {
	registerEntityPrimaryKey()
}

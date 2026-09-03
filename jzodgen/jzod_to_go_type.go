package jzodgen

import (
	"fmt"
	"go/format"
	"strings"
	"unicode"

	"github.com/miroir-framework/miroir/go/jzod"
)

// JzodToGoType emits a Go defined type with the given name for a Jzod schema.
// env is unused until schemaReference conversion (relativePath names and
// absolutePath uuids in the same map).
func JzodToGoType(name string, schema any, env map[string]any) (string, error) {
	expr, err := goTypeExpr(schema, env)
	if err != nil {
		return "", err
	}
	decl := "type " + name + " " + expr
	formatted, err := format.Source([]byte(decl))
	if err != nil {
		return "", fmt.Errorf("JzodToGoType format: %w", err)
	}
	return strings.TrimRight(string(formatted), "\n"), nil
}

func goTypeExpr(schema any, env map[string]any) (string, error) {
	el, ok := schema.(map[string]any)
	if !ok {
		return "", fmt.Errorf("JzodToGoType expected a schema object")
	}
	switch el["type"] {
	case "string", "bigint", "uuid", "date":
		return "string", nil
	case "boolean":
		return "bool", nil
	case "number":
		return "float64", nil
	case "any", "unknown", "undefined", "void", "null":
		return "any", nil
	case "never":
		return "struct{}", nil
	case "literal":
		return goTypeOfValue(el["definition"]), nil
	case "enum":
		return "string", nil
	case "union":
		list, ok := el["definition"].([]any)
		if !ok || len(list) == 0 {
			return "any", nil
		}
		first, err := goTypeExpr(list[0], env)
		if err != nil {
			return "", err
		}
		for _, item := range list[1:] {
			next, err := goTypeExpr(item, env)
			if err != nil {
				return "", err
			}
			if next != first {
				return "any", nil
			}
		}
		return first, nil
	case "array":
		inner, err := goTypeExpr(el["definition"], env)
		if err != nil {
			return "", err
		}
		return "[]" + inner, nil
	case "record":
		inner, err := goTypeExpr(el["definition"], env)
		if err != nil {
			return "", err
		}
		return "map[string]" + inner, nil
	case "tuple":
		list, ok := el["definition"].([]any)
		if !ok {
			return "", fmt.Errorf("JzodToGoType tuple definition must be an array")
		}
		fields := make([]structField, len(list))
		for i, item := range list {
			inner, err := goTypeExpr(item, env)
			if err != nil {
				return "", err
			}
			fields[i] = structField{name: fmt.Sprintf("E%d", i), typ: inner}
		}
		return structExpr(fields), nil
	case "object":
		def, err := objectDefinition(el, env)
		if err != nil {
			return "", err
		}
		if len(def) == 0 {
			return "struct{}", nil
		}
		partial, _ := el["partial"].(bool)
		fields := make([]structField, 0, len(def))
		for _, k := range jzod.KeysOf(def) {
			inner, err := goTypeExpr(def[k], env)
			if err != nil {
				return "", err
			}
			child, _ := def[k].(map[string]any)
			typ := inner
			tag := "json:\"" + k + "\""
			optional, _ := child["optional"].(bool)
			nullable, _ := child["nullable"].(bool)
			if optional || nullable || partial {
				typ = "*" + inner
			}
			if optional || partial {
				tag = "json:\"" + k + ",omitempty\""
			}
			fields = append(fields, structField{
				name: exportedIdent(k),
				typ:  typ,
				tag:  tag,
			})
		}
		return structExpr(fields), nil
	default:
		return "", fmt.Errorf("JzodToGoType unsupported type %v", el["type"])
	}
}

type structField struct {
	name string
	typ  string
	tag  string
}

func structExpr(fields []structField) string {
	if len(fields) == 0 {
		return "struct{}"
	}
	var b strings.Builder
	b.WriteString("struct {\n")
	for _, f := range fields {
		b.WriteString("\t")
		b.WriteString(f.name)
		b.WriteString(" ")
		b.WriteString(f.typ)
		if f.tag != "" {
			b.WriteString(" `")
			b.WriteString(f.tag)
			b.WriteString("`")
		}
		b.WriteString("\n")
	}
	b.WriteString("}")
	return b.String()
}

func exportedIdent(key string) string {
	parts := strings.FieldsFunc(key, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var b strings.Builder
	for _, p := range parts {
		rs := []rune(p)
		if len(rs) == 0 {
			continue
		}
		rs[0] = unicode.ToUpper(rs[0])
		b.WriteString(string(rs))
	}
	s := b.String()
	if s == "" || !unicode.IsLetter([]rune(s)[0]) {
		return "F" + s
	}
	return s
}

func goTypeOfValue(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "float64"
	default:
		return "any"
	}
}

func objectDefinition(el map[string]any, env map[string]any) (map[string]any, error) {
	own, _ := el["definition"].(map[string]any)
	if own == nil {
		own = map[string]any{}
	}
	if el["extend"] == nil {
		return own, nil
	}
	parent, err := extendDefinition(el["extend"], env)
	if err != nil {
		return nil, err
	}
	return mergeObjectDefs(parent, own), nil
}

func extendDefinition(extend any, env map[string]any) (map[string]any, error) {
	switch e := extend.(type) {
	case []any:
		out := map[string]any{}
		for _, item := range e {
			part, err := extendDefinition(item, env)
			if err != nil {
				return nil, err
			}
			out = mergeObjectDefs(out, part)
		}
		return out, nil
	case map[string]any:
		switch e["type"] {
		case "object":
			return objectDefinition(e, env)
		default:
			return nil, fmt.Errorf("JzodToGoType extend clause type %v", e["type"])
		}
	default:
		return nil, fmt.Errorf("JzodToGoType unsupported extend clause")
	}
}

func mergeObjectDefs(parent, child map[string]any) map[string]any {
	if parent == nil {
		parent = map[string]any{}
	}
	if child == nil {
		child = map[string]any{}
	}
	out := map[string]any{}
	var keys []string
	seen := map[string]bool{}
	for _, k := range jzod.KeysOf(parent) {
		out[k] = parent[k]
		keys = append(keys, k)
		seen[k] = true
	}
	for _, k := range jzod.KeysOf(child) {
		out[k] = child[k]
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	jzod.RememberKeys(out, keys)
	return out
}

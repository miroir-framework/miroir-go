package jzodgen

import (
	"fmt"
	"go/format"
	"strings"
	"unicode"

	"github.com/miroir-framework/miroir/go/jzod"
)

type namedDecl struct {
	name string
	expr string
}

type generator struct {
	env      map[string]any
	names    map[string]any
	emitted  []namedDecl
	didEmit  map[string]bool
	emitting map[string]bool
}

// JzodToGoType emits a Go defined type with the given name for a Jzod schema.
// env is a mixed lookup: relativePath names map to Jzod elements; absolutePath
// uuids map to an MlSchema (or its definition.context), matching jzod-ts
// lazy/eager references plus TypeCheck uuid lookup.
func JzodToGoType(name string, schema any, env map[string]any) (string, error) {
	g := newGenerator(env)
	expr, err := g.goTypeExprAt(schema, false)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, d := range g.emitted {
		b.WriteString("type ")
		b.WriteString(d.name)
		b.WriteString(" ")
		b.WriteString(d.expr)
		b.WriteString("\n")
	}
	if !g.didEmit[name] {
		b.WriteString("type ")
		b.WriteString(name)
		b.WriteString(" ")
		b.WriteString(expr)
		b.WriteString("\n")
	}
	return formatDecls(b.String())
}

func newGenerator(env map[string]any) *generator {
	if env == nil {
		env = map[string]any{}
	}
	names := map[string]any{}
	for k, v := range env {
		if isJzodElement(v) && !(looksLikeUUID(k) && isMlSchema(v)) {
			names[k] = v
		}
	}
	return &generator{
		env:      env,
		names:    names,
		didEmit:  map[string]bool{},
		emitting: map[string]bool{},
	}
}

func formatDecls(src string) (string, error) {
	wrapped := "package p\n\n" + strings.TrimSpace(src) + "\n"
	formatted, err := format.Source([]byte(wrapped))
	if err != nil {
		return "", fmt.Errorf("JzodToGoType format: %w", err)
	}
	out := strings.TrimPrefix(string(formatted), "package p")
	return strings.TrimSpace(out), nil
}

func (g *generator) goTypeExprAt(schema any, inField bool) (string, error) {
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
		first, err := g.goTypeExprAt(list[0], inField)
		if err != nil {
			return "", err
		}
		for _, item := range list[1:] {
			next, err := g.goTypeExprAt(item, inField)
			if err != nil {
				return "", err
			}
			if next != first {
				return "any", nil
			}
		}
		return first, nil
	case "array":
		inner, err := g.goTypeExprAt(el["definition"], true)
		if err != nil {
			return "", err
		}
		return "[]" + inner, nil
	case "record":
		inner, err := g.goTypeExprAt(el["definition"], true)
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
			inner, err := g.goTypeExprAt(item, true)
			if err != nil {
				return "", err
			}
			fields[i] = structField{name: fmt.Sprintf("E%d", i), typ: inner}
		}
		return structExpr(fields), nil
	case "object":
		return g.objectExpr(el)
	case "schemaReference":
		return g.schemaRefExpr(el, inField)
	default:
		return "", fmt.Errorf("JzodToGoType unsupported type %v", el["type"])
	}
}

func (g *generator) objectExpr(el map[string]any) (string, error) {
	def, err := g.objectDefinition(el)
	if err != nil {
		return "", err
	}
	if len(def) == 0 {
		return "struct{}", nil
	}
	partial, _ := el["partial"].(bool)
	fields := make([]structField, 0, len(def))
	for _, k := range jzod.KeysOf(def) {
		inner, err := g.goTypeExprAt(def[k], true)
		if err != nil {
			return "", err
		}
		child, _ := def[k].(map[string]any)
		typ := inner
		tag := "json:\"" + k + "\""
		optional, _ := child["optional"].(bool)
		nullable, _ := child["nullable"].(bool)
		if (optional || nullable || partial) && !strings.HasPrefix(inner, "*") {
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
}

func (g *generator) schemaRefExpr(el map[string]any, inField bool) (string, error) {
	if ctx := contextMap(el); len(ctx) > 0 {
		for _, k := range jzod.KeysOf(ctx) {
			g.names[k] = ctx[k]
		}
		for _, k := range jzod.KeysOf(ctx) {
			if err := g.emitNamed(k, ctx[k]); err != nil {
				return "", err
			}
		}
	}
	def := refDefinition(el)
	rel, _ := def["relativePath"].(string)
	abs, _ := def["absolutePath"].(string)
	eager, _ := def["eager"].(bool)
	if eager {
		target, err := g.resolve(abs, rel)
		if err != nil {
			return "", err
		}
		return g.goTypeExprAt(target, inField)
	}
	if rel == "" {
		return "", fmt.Errorf("JzodToGoType schemaReference missing relativePath")
	}
	if abs != "" {
		if _, ok := g.env[abs]; !ok {
			return "", fmt.Errorf("JzodToGoType absolutePath %s not registered", abs)
		}
	}
	return g.lazyRefExpr(rel, inField), nil
}

func (g *generator) emitNamed(key string, schema any) error {
	typeName := exportedIdent(key)
	if g.didEmit[typeName] || g.emitting[typeName] {
		return nil
	}
	g.emitting[typeName] = true
	expr, err := g.goTypeExprAt(schema, false)
	g.emitting[typeName] = false
	if err != nil {
		return err
	}
	g.emitted = append(g.emitted, namedDecl{name: typeName, expr: expr})
	g.didEmit[typeName] = true
	return nil
}

func (g *generator) lazyRefExpr(rel string, inField bool) string {
	name := exportedIdent(rel)
	if g.emitting[name] {
		return "*" + name
	}
	if inField {
		if target, ok := g.lookupRel(rel); ok && isObjectType(target) {
			return "*" + name
		}
	}
	return name
}

func isObjectType(v any) bool {
	m, ok := v.(map[string]any)
	return ok && m["type"] == "object"
}

func (g *generator) resolve(abs, rel string) (any, error) {
	if abs != "" {
		raw, ok := g.env[abs]
		if !ok {
			return nil, fmt.Errorf("JzodToGoType absolutePath %s not registered", abs)
		}
		ctx := mlContext(raw)
		if rel == "" {
			return map[string]any{"type": "object", "definition": ctx}, nil
		}
		got, ok := ctx[rel]
		if !ok {
			return nil, fmt.Errorf("JzodToGoType unresolved relativePath %s", rel)
		}
		return got, nil
	}
	if rel == "" {
		return nil, fmt.Errorf("JzodToGoType schemaReference missing relativePath")
	}
	got, ok := g.lookupRel(rel)
	if !ok {
		return nil, fmt.Errorf("JzodToGoType unresolved relativePath %s", rel)
	}
	return got, nil
}

func (g *generator) lookupRel(rel string) (any, bool) {
	if v, ok := g.names[rel]; ok {
		return v, true
	}
	if v, ok := g.env[rel]; ok && isJzodElement(v) {
		return v, true
	}
	return nil, false
}

func (g *generator) objectDefinition(el map[string]any) (map[string]any, error) {
	own, _ := el["definition"].(map[string]any)
	if own == nil {
		own = map[string]any{}
	}
	if el["extend"] == nil {
		return own, nil
	}
	parent, err := g.extendDefinition(el["extend"])
	if err != nil {
		return nil, err
	}
	return mergeObjectDefs(parent, own), nil
}

func (g *generator) extendDefinition(extend any) (map[string]any, error) {
	switch e := extend.(type) {
	case []any:
		out := map[string]any{}
		for _, item := range e {
			part, err := g.extendDefinition(item)
			if err != nil {
				return nil, err
			}
			out = mergeObjectDefs(out, part)
		}
		return out, nil
	case map[string]any:
		switch e["type"] {
		case "object":
			return g.objectDefinition(e)
		case "schemaReference":
			if ctx := contextMap(e); len(ctx) > 0 {
				for _, k := range jzod.KeysOf(ctx) {
					g.names[k] = ctx[k]
				}
			}
			def := refDefinition(e)
			rel, _ := def["relativePath"].(string)
			abs, _ := def["absolutePath"].(string)
			target, err := g.resolve(abs, rel)
			if err != nil {
				return nil, err
			}
			tm, ok := target.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("JzodToGoType extend schemaReference resolved to non-object")
			}
			if tm["type"] != "object" {
				return nil, fmt.Errorf("JzodToGoType extend schemaReference resolved to non-object")
			}
			return g.objectDefinition(tm)
		default:
			return nil, fmt.Errorf("JzodToGoType extend clause type %v", e["type"])
		}
	default:
		return nil, fmt.Errorf("JzodToGoType unsupported extend clause")
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

func isJzodElement(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	t, ok := m["type"].(string)
	return ok && t != ""
}

func isMlSchema(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	def, ok := m["definition"].(map[string]any)
	if !ok {
		return false
	}
	_, hasCtx := def["context"]
	return hasCtx
}

func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

func contextMap(el map[string]any) map[string]any {
	c, _ := el["context"].(map[string]any)
	return c
}

func refDefinition(el map[string]any) map[string]any {
	d, _ := el["definition"].(map[string]any)
	if d == nil {
		return map[string]any{}
	}
	return d
}

func mlContext(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	if def, ok := m["definition"].(map[string]any); ok {
		if ctx, ok := def["context"].(map[string]any); ok {
			return ctx
		}
	}
	if ctx, ok := m["context"].(map[string]any); ok {
		return ctx
	}
	if _, hasType := m["type"]; !hasType {
		return m
	}
	return map[string]any{}
}

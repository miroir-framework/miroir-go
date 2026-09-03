package miroirtest

import (
	"regexp"
	"sort"
	"strings"
)

var entityUUIDRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func registerInterfaceCheck() {
	registry["miroir-core/2_domain/TransformerInterfaceCheck"] = map[string]Fn{
		"inputOutputTypesCompatible":                          wrap2(inputOutputTypesCompatible),
		"checkTransformerInterfaceCompatibility":              wrap2(checkTransformerInterfaceCompatibility),
		"checkTransformerInterfaceCompatibilityWithInference": wrap3(checkTransformerInterfaceCompatibilityWithInference),
		"getTransformerDefinitionInputOutput":                 wrapGetTransformerDefinitionInputOutput,
		"findInvalidStockTransformerInputOutputs":             wrapFindInvalidStock,
	}
}

func wrapGetTransformerDefinitionInputOutput(args []any) (any, error) {
	name := fmtSprint(argAt(args, 0))
	defs := transformerDefinitions()
	if len(args) > 1 {
		if override, ok := args[1].(map[string]any); ok {
			defs = override
		}
	}
	def, ok := defs[name].(map[string]any)
	if !ok {
		return jsonUndefined{}, nil
	}
	iface, _ := def["transformerInterface"].(map[string]any)
	if iface == nil {
		return jsonUndefined{}, nil
	}
	io := iface["inputOutput"]
	if io == nil {
		return jsonUndefined{}, nil
	}
	return io, nil
}

func wrapFindInvalidStock(args []any) (any, error) {
	defs := transformerDefinitions()
	if len(args) > 0 {
		if override, ok := args[0].(map[string]any); ok {
			defs = override
		}
	}
	var invalid []string
	for name, raw := range defs {
		def, _ := raw.(map[string]any)
		iface, _ := def["transformerInterface"].(map[string]any)
		if iface == nil {
			continue
		}
		io := iface["inputOutput"]
		if io == nil {
			continue
		}
		if !validInputOutputObject(io) {
			invalid = append(invalid, name)
		}
	}
	sort.Strings(invalid)
	out := make([]any, len(invalid))
	for i, n := range invalid {
		out[i] = n
	}
	return out, nil
}

func inputOutputTypesCompatible(actual, expected any) any {
	if isAny(actual) || isAny(expected) {
		return true
	}
	a := normalizeIOType(actual)
	e := normalizeIOType(expected)
	switch a.kind {
	case "entityUuid":
		return (e.kind == "entityUuid" && a.uuid == e.uuid) || (e.kind == "object" && e.payload == "any")
	case "primitive":
		return e.kind == "primitive" && e.value == a.value
	case "object", "array":
		return e.kind == a.kind && payloadsCompatible(a.payload, e.payload)
	}
	return false
}

type ioType struct {
	kind    string
	value   string
	uuid    string
	payload string
}

func normalizeIOType(t any) ioType {
	if m, ok := t.(map[string]any); ok {
		kind, _ := m["type"].(string)
		payload := "any"
		if p, ok := m["payload"].(string); ok {
			payload = p
		}
		return ioType{kind: kind, payload: payload}
	}
	s := fmtSprint(t)
	if s == "object" || s == "array" {
		return ioType{kind: s, payload: "any"}
	}
	if entityUUIDRe.MatchString(s) {
		return ioType{kind: "entityUuid", uuid: strings.ToLower(s)}
	}
	return ioType{kind: "primitive", value: s}
}

func isAny(t any) bool {
	return fmtSprint(t) == "any"
}

func payloadsCompatible(actual, expected string) bool {
	if actual == "any" || expected == "any" {
		return true
	}
	if entityUUIDRe.MatchString(actual) || entityUUIDRe.MatchString(expected) {
		return entityUUIDRe.MatchString(actual) && entityUUIDRe.MatchString(expected) && strings.ToLower(actual) == strings.ToLower(expected)
	}
	return actual == expected
}

func checkTransformerInterfaceCompatibility(given, declared any) any {
	decl := declaredInputOutput(declared)
	givenMap, _ := given.(map[string]any)
	var failures []any
	if fmtSprint(decl["input"]) != "undefined" && inputOutputTypesCompatible(givenMap["input"], decl["input"]) != true {
		failures = append(failures, map[string]any{"direction": "input", "given": givenMap["input"], "declared": decl["input"]})
	}
	if inputOutputTypesCompatible(decl["output"], givenMap["output"]) != true {
		failures = append(failures, map[string]any{"direction": "output", "given": givenMap["output"], "declared": decl["output"]})
	}
	if len(failures) == 0 {
		return map[string]any{"status": "ok"}
	}
	return map[string]any{"status": "incompatible", "failures": failures}
}

func checkTransformerInterfaceCompatibilityWithInference(given, declared, inferred any) any {
	base := checkTransformerInterfaceCompatibility(given, declared).(map[string]any)
	givenMap, _ := given.(map[string]any)
	if isJSONUndefined(inferred) || inferred == nil {
		return base
	}
	if inputOutputTypesCompatible(inferred, givenMap["output"]) == true {
		return base
	}
	inferredFailure := map[string]any{
		"direction": "output",
		"given":     givenMap["output"],
		"declared":  inferred,
		"source":    "inferred",
	}
	if base["status"] == "ok" {
		return map[string]any{"status": "incompatible", "failures": []any{inferredFailure}}
	}
	failures, _ := base["failures"].([]any)
	return map[string]any{"status": "incompatible", "failures": append(append([]any{}, failures...), inferredFailure)}
}

func declaredInputOutput(declared any) map[string]any {
	if m, ok := declared.(map[string]any); ok {
		return m
	}
	return map[string]any{"input": "any", "output": "any"}
}

func validInputOutputObject(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	if len(m) != 2 {
		return false
	}
	return validInputOutputType(m["input"]) && validInputOutputType(m["output"])
}

func validInputOutputType(v any) bool {
	if s, ok := v.(string); ok {
		switch s {
		case "any", "undefined", "bigint", "number", "string", "boolean", "object", "array":
			return true
		}
		return entityUUIDRe.MatchString(s)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	t, _ := m["type"].(string)
	if t != "object" && t != "array" {
		return false
	}
	if payload, exists := m["payload"]; exists {
		return validPayload(payload)
	}
	return true
}

func validPayload(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if s == "any" || s == "undefined" || s == "bigint" || s == "number" || s == "string" || s == "boolean" {
		return true
	}
	return entityUUIDRe.MatchString(s)
}

func fmtSprint(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(jsonString(v), `"`, ""), "\n", ""))
}

func init() {
	registerInterfaceCheck()
}

package jzod

import (
	"testing"

	"github.com/miroir-framework/miroir/go/jzod/generated"
)

func TestTypeCheckLiteralFromGeneratedStruct(t *testing.T) {
	schema := generated.JzodLiteral{Type: "literal", Definition: "myLiteral"}
	result := TypeCheck(schema, "myLiteral", nil, nil, ModelEnvironment{}, nil)
	if result.Status != "ok" {
		t.Fatalf("status: got %q error %q", result.Status, result.Error)
	}
	jsonEqual(t, result.ResolvedSchema, schema, "resolvedSchema")
	jsonEqual(t, result.RawSchema, schema, "rawSchema")
}

func TestTypeCheckStringFromGeneratedStruct(t *testing.T) {
	schema := generated.JzodPlainAttribute{Type: "string"}
	result := TypeCheck(schema, "hello", nil, nil, ModelEnvironment{}, nil)
	if result.Status != "ok" {
		t.Fatalf("status: got %q error %q", result.Status, result.Error)
	}
	result = TypeCheck(schema, 42.0, nil, nil, ModelEnvironment{}, nil)
	if result.Status != "error" {
		t.Fatalf("string vs number: got status %q", result.Status)
	}
}

func TestTypeCheckObjectFromGeneratedStruct(t *testing.T) {
	schema := generated.JzodObject{
		Type: "object",
		Definition: map[string]generated.JzodElement{
			"name": generated.JzodPlainAttribute{Type: "string"},
		},
	}
	result := TypeCheck(schema, map[string]any{"name": "Ada"}, nil, nil, ModelEnvironment{}, nil)
	if result.Status != "ok" {
		t.Fatalf("status: got %q error %q", result.Status, result.Error)
	}
	want := map[string]any{
		"type": "object",
		"definition": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}
	jsonEqual(t, result.ResolvedSchema, want, "resolvedSchema")
}

func TestTypeCheckSchemaReferenceFromGeneratedStruct(t *testing.T) {
	ctx := map[string]generated.JzodElement{
		"a": generated.JzodPlainAttribute{Type: "string"},
	}
	schema := generated.JzodReference{Type: "schemaReference", Context: &ctx}
	schema.Definition.RelativePath = "a"
	result := TypeCheck(schema, "ok", nil, nil, ModelEnvironment{}, nil)
	if result.Status != "ok" {
		t.Fatalf("status: got %q error %q", result.Status, result.Error)
	}
	if result.SchemaReferenceName != "a" {
		t.Fatalf("schemaReferenceName: got %q", result.SchemaReferenceName)
	}
}

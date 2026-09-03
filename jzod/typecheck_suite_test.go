package jzod

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func typecheckSuiteLeaf(t *testing.T, index int) map[string]any {
	t.Helper()
	doc := loadJSONObject(t, "..", "packages", "miroir-test-app_deployment-miroir", "assets", "miroir_data", "a311f363-e238-4203-bdfc-29e8c160c26b", "3aff508a-8a9f-4384-ba50-cc696411eba5.json")
	definition := asMap(t, doc["definition"], "definition")
	leaves := asSlice(t, definition["miroirTests"], "definition.miroirTests")
	if index < 0 || index >= len(leaves) {
		t.Fatalf("leaf index %d out of range (n=%d)", index, len(leaves))
	}
	return asMap(t, leaves[index], "leaf")
}

func transformerPayload(t *testing.T, leaf map[string]any) (schema any, value any) {
	t.Helper()
	transformer := asMap(t, leaf["transformer"], "transformer")
	return transformer["mlSchema"], transformer["valueObject"]
}

func jsonEqual(t *testing.T, got, want any, field string) {
	t.Helper()
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("%s: marshal got: %v", field, err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("%s: marshal want: %v", field, err)
	}
	var gotNorm, wantNorm any
	if err := json.Unmarshal(gotJSON, &gotNorm); err != nil {
		t.Fatalf("%s: unmarshal got: %v", field, err)
	}
	if err := json.Unmarshal(wantJSON, &wantNorm); err != nil {
		t.Fatalf("%s: unmarshal want: %v", field, err)
	}
	if !reflect.DeepEqual(gotNorm, wantNorm) {
		t.Fatalf("%s:\n got %s\nwant %s", field, gotJSON, wantJSON)
	}
}

func TestTypeCheckLiteralFromSuiteLeaf0(t *testing.T) {
	leaf := typecheckSuiteLeaf(t, 0)
	if got := leaf["miroirTestLabel"]; got != "test010_literal" {
		t.Fatalf("leaf 0 label: got %v", got)
	}
	schema, value := transformerPayload(t, leaf)
	result := TypeCheck(schema, value, nil, nil, ModelEnvironment{}, nil)

	expected := asMap(t, leaf["expectedValue"], "expectedValue")
	if result.Status != "ok" {
		t.Fatalf("status: got %q error %q", result.Status, result.Error)
	}
	jsonEqual(t, result.ResolvedSchema, expected["resolvedSchema"], "resolvedSchema")
	jsonEqual(t, result.RawSchema, expected["rawSchema"], "rawSchema")
	jsonEqual(t, result.ValuePath, expected["valuePath"], "valuePath")
	jsonEqual(t, result.TypePath, expected["typePath"], "typePath")
}

func TestTypeCheckUnsupportedTypeErrors(t *testing.T) {
	result := TypeCheck(map[string]any{"type": "notAJzodType"}, "x", nil, nil, ModelEnvironment{}, nil)
	if result.Status != "error" {
		t.Fatalf("unsupported type: got status %q", result.Status)
	}
}

func assertSuiteLeafStatusAndSchemas(t *testing.T, index int, wantLabel string) {
	t.Helper()
	leaf := typecheckSuiteLeaf(t, index)
	if got := leaf["miroirTestLabel"]; got != wantLabel {
		t.Fatalf("leaf %d label: got %v want %s", index, got, wantLabel)
	}
	schema, value := transformerPayload(t, leaf)
	result := TypeCheck(schema, value, nil, nil, ModelEnvironment{}, nil)
	expected := asMap(t, leaf["expectedValue"], "expectedValue")
	if result.Status != "ok" {
		t.Fatalf("leaf %d %s status: got %q error %q", index, wantLabel, result.Status, result.Error)
	}
	jsonEqual(t, result.ResolvedSchema, expected["resolvedSchema"], "resolvedSchema")
	jsonEqual(t, result.RawSchema, expected["rawSchema"], "rawSchema")
}

func TestTypeCheckStringFromSuiteLeaf1(t *testing.T) {
	assertSuiteLeafStatusAndSchemas(t, 1, "test020_string")
}

func TestTypeCheckBooleanTrueFromSuiteLeaf2(t *testing.T) {
	assertSuiteLeafStatusAndSchemas(t, 2, "test022_boolean_true")
}

func TestTypeCheckBooleanFalseFromSuiteLeaf3(t *testing.T) {
	assertSuiteLeafStatusAndSchemas(t, 3, "test024_boolean_false")
}

func TestTypeCheckStringRejectsNumber(t *testing.T) {
	leaf := typecheckSuiteLeaf(t, 1)
	schema, _ := transformerPayload(t, leaf)
	result := TypeCheck(schema, 42.0, nil, nil, ModelEnvironment{}, nil)
	if result.Status != "error" {
		t.Fatalf("string schema vs number: got status %q", result.Status)
	}
}

func TestTypeCheckSchemaReferenceFromSuiteLeaf4(t *testing.T) {
	leaf := typecheckSuiteLeaf(t, 4)
	if got := leaf["miroirTestLabel"]; got != "test030_schemaReference" {
		t.Fatalf("leaf 4 label: got %v", got)
	}
	schema, value := transformerPayload(t, leaf)
	result := TypeCheck(schema, value, nil, nil, ModelEnvironment{}, nil)
	expected := asMap(t, leaf["expectedValue"], "expectedValue")
	if result.Status != "ok" {
		t.Fatalf("status: got %q error %q", result.Status, result.Error)
	}
	if result.SchemaReferenceName != "a" {
		t.Fatalf("schemaReferenceName: got %q", result.SchemaReferenceName)
	}
	jsonEqual(t, result.ResolvedSchema, expected["resolvedSchema"], "resolvedSchema")
	jsonEqual(t, result.TypePath, expected["typePath"], "typePath")
}

func TestTypeCheckSimpleObjectFromSuiteLeaf5(t *testing.T) {
	leaf := typecheckSuiteLeaf(t, 5)
	if got := leaf["miroirTestLabel"]; got != "test040_simple_object" {
		t.Fatalf("leaf 5 label: got %v", got)
	}
	schema, value := transformerPayload(t, leaf)
	result := TypeCheck(schema, value, nil, nil, ModelEnvironment{}, nil)
	expected := asMap(t, leaf["expectedValue"], "expectedValue")
	if result.Status != "ok" {
		t.Fatalf("status: got %q error %q", result.Status, result.Error)
	}
	if result.SchemaReferenceName != "myObject" {
		t.Fatalf("schemaReferenceName: got %q", result.SchemaReferenceName)
	}
	jsonEqual(t, result.ResolvedSchema, expected["resolvedSchema"], "resolvedSchema")
	jsonEqual(t, result.TypePath, expected["typePath"], "typePath")
}

func TestTypeCheckObjectWithUnionFromSuiteLeaf6(t *testing.T) {
	assertSuiteLeafStatusAndSchemas(t, 6, "test050_object_with_union")
}

func TestTypeCheckSimpleUnionFromSuiteLeaf9(t *testing.T) {
	assertSuiteLeafStatusAndSchemas(t, 9, "test120_simple_union")
}

func TestTypeCheckAllSuiteLeavesSubExpected(t *testing.T) {
	for index, want := range typecheckSuiteLeafLabels {
		t.Run(fmt.Sprintf("%02d_%s", index, want), func(t *testing.T) {
			assertSuiteLeafSubExpected(t, index, want)
		})
	}
}

func TestTypeCheckAnyLeavesFromSuite(t *testing.T) {
	for index := 10; index <= 17; index++ {
		want := typecheckSuiteLeafLabels[index]
		t.Run(want, func(t *testing.T) {
			assertSuiteLeafSubExpected(t, index, want)
		})
	}
}

func assertSuiteLeafSubExpected(t *testing.T, index int, wantLabel string) {
	t.Helper()
	leaf := typecheckSuiteLeaf(t, index)
	if got := leaf["miroirTestLabel"]; got != wantLabel {
		t.Fatalf("leaf %d label: got %v want %s", index, got, wantLabel)
	}
	schema, value := transformerPayload(t, leaf)
	result := TypeCheck(schema, value, nil, nil, ModelEnvironment{}, nil)
	pairs, ok := leaf["subExpectedValue"].([]any)
	if !ok {
		t.Fatal("leaf has no subExpectedValue")
	}
	gotRoot := resultAsMap(t, result)
	for _, raw := range pairs {
		pair, ok := raw.([]any)
		if !ok || len(pair) != 2 {
			t.Fatalf("bad subExpectedValue pair: %#v", raw)
		}
		path, _ := pair[0].(string)
		jsonEqual(t, dotted(gotRoot, path), pair[1], path)
	}
}

func resultAsMap(t *testing.T, result Result) map[string]any {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return out
}

func dotted(root any, path string) any {
	cur := root
	for _, seg := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[seg]
	}
	return cur
}

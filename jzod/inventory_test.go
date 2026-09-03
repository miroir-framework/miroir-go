package jzod

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	bootstrapSchemaUUID = "1e8dab4b-65a3-4686-922e-ce89a2d62aa9"
	typecheckSuiteUUID  = "3aff508a-8a9f-4384-ba50-cc696411eba5"
)

var bootstrapContextKeys = []string{
	"jzodArray",
	"jzodAttributeDateValidations",
	"jzodAttributeNumberValidations",
	"jzodAttributePlainDateWithValidations",
	"jzodAttributePlainNumberWithValidations",
	"jzodAttributePlainStringWithValidations",
	"jzodAttributeStringValidations",
	"jzodBaseObject",
	"jzodElement",
	"jzodEnum",
	"jzodEnumAttributeTypes",
	"jzodEnumElementTypes",
	"jzodFunction",
	"jzodIntersection",
	"jzodLazy",
	"jzodLiteral",
	"jzodMap",
	"jzodObject",
	"jzodPlainAttribute",
	"jzodPromise",
	"jzodRecord",
	"jzodReference",
	"jzodSet",
	"jzodTuple",
	"jzodUnion",
}

var jzodElementBranchRelativePaths = []string{
	"jzodArray",
	"jzodPlainAttribute",
	"jzodAttributePlainDateWithValidations",
	"jzodAttributePlainNumberWithValidations",
	"jzodAttributePlainStringWithValidations",
	"jzodEnum",
	"jzodFunction",
	"jzodLazy",
	"jzodLiteral",
	"jzodIntersection",
	"jzodMap",
	"jzodObject",
	"jzodPromise",
	"jzodRecord",
	"jzodReference",
	"jzodSet",
	"jzodTuple",
	"jzodUnion",
}

var typecheckSuiteLeafLabels = []string{
	"test010_literal",
	"test020_string",
	"test022_boolean_true",
	"test024_boolean_false",
	"test030_schemaReference",
	"test040_simple_object",
	"test050_object_with_union",
	"test060_recursive_object",
	"test070",
	"test120_simple_union",
	"test130_any_string",
	"test140_any_number",
	"test150_any_object",
	"test160_any_object_of_object",
	"test170_any_array",
	"test180_any_null",
	"test181_any_boolean",
	"test182_any_empty_object",
	"test010",
	"test011",
	"test012",
	"test013",
	"test014",
	"test015",
	"test016",
	"test017",
	"test018",
	"test019",
	"test020",
	"test021",
	"test022",
	"test024",
	"test025",
	"test026",
	"test027",
	"test028",
	"test029",
	"test030",
	"test100",
	"test110",
	"test120",
	"test130",
}

func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func loadJSONObject(t *testing.T, parts ...string) map[string]any {
	t.Helper()
	path := filepath.Join(append([]string{packageDir(t)}, parts...)...)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return out
}

func asMap(t *testing.T, v any, path string) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%s: want object, got %T", path, v)
	}
	return m
}

func asSlice(t *testing.T, v any, path string) []any {
	t.Helper()
	s, ok := v.([]any)
	if !ok {
		t.Fatalf("%s: want array, got %T", path, v)
	}
	return s
}

func TestInventoryBootstrapSchema(t *testing.T) {
	doc := loadJSONObject(t, "..", "packages", "miroir-test-app_deployment-miroir", "assets", "miroir_data", "5e81e1b9-38be-487c-b3e5-53796c57fccf", "1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json")
	if got := doc["name"]; got != "jzodMiroirBootstrapSchema" {
		t.Fatalf("name: got %v", got)
	}
	if got := doc["uuid"]; got != bootstrapSchemaUUID {
		t.Fatalf("uuid: got %v", got)
	}

	definition := asMap(t, doc["definition"], "definition")
	context := asMap(t, definition["context"], "definition.context")
	if len(context) != len(bootstrapContextKeys) {
		t.Fatalf("context key count: got %d want %d", len(context), len(bootstrapContextKeys))
	}
	for _, key := range bootstrapContextKeys {
		if _, ok := context[key]; !ok {
			t.Fatalf("missing context key %q", key)
		}
	}

	jzodElement := asMap(t, context["jzodElement"], "jzodElement")
	branches := asSlice(t, jzodElement["definition"], "jzodElement.definition")
	if len(branches) != len(jzodElementBranchRelativePaths) {
		t.Fatalf("jzodElement branches: got %d want %d", len(branches), len(jzodElementBranchRelativePaths))
	}
	for i, want := range jzodElementBranchRelativePaths {
		branch := asMap(t, branches[i], "jzodElement.definition[%d]")
		if got := branch["type"]; got != "schemaReference" {
			t.Fatalf("branch %d type: got %v", i, got)
		}
		def := asMap(t, branch["definition"], "jzodElement.definition[%d].definition")
		if got := def["relativePath"]; got != want {
			t.Fatalf("branch %d relativePath: got %v want %s", i, got, want)
		}
	}
}

func TestInventoryTypecheckSuiteLeaves(t *testing.T) {
	doc := loadJSONObject(t, "..", "packages", "miroir-test-app_deployment-miroir", "assets", "miroir_data", "a311f363-e238-4203-bdfc-29e8c160c26b", "3aff508a-8a9f-4384-ba50-cc696411eba5.json")
	if got := doc["uuid"]; got != typecheckSuiteUUID {
		t.Fatalf("uuid: got %v", got)
	}
	definition := asMap(t, doc["definition"], "definition")
	leaves := asSlice(t, definition["miroirTests"], "definition.miroirTests")
	if len(leaves) != len(typecheckSuiteLeafLabels) {
		t.Fatalf("leaf count: got %d want %d", len(leaves), len(typecheckSuiteLeafLabels))
	}
	for i, want := range typecheckSuiteLeafLabels {
		leaf := asMap(t, leaves[i], "leaf")
		if got := leaf["miroirTestType"]; got != "transformerTest" {
			t.Fatalf("leaf %d type: got %v", i, got)
		}
		if got := leaf["miroirTestLabel"]; got != want {
			t.Fatalf("leaf %d label: got %v want %s", i, got, want)
		}
	}
}

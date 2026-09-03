package jzod

import "testing"

func TestBootstrapSelfParse(t *testing.T) {
	doc := loadJSONObject(t, "..", "packages", "miroir-test-app_deployment-miroir", "assets", "miroir_data", "5e81e1b9-38be-487c-b3e5-53796c57fccf", "1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json")
	if got := doc["uuid"]; got != bootstrapSchemaUUID {
		t.Fatalf("uuid: got %v", got)
	}
	definition := doc["definition"]
	env := ModelEnvironment{AbsoluteSchemas: []map[string]any{doc}}
	result := TypeCheck(definition, definition, nil, nil, env, nil)
	if result.Status != "ok" {
		t.Fatalf("bootstrap self-parse: status=%q error=%q", result.Status, result.Error)
	}
}

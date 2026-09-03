package transformer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	defsOnce sync.Once
	defs     map[string]any
)

// TransformerDefinitions loads TransformerDefinition JSON (entity
// a557419d-…) keyed by name and by transformerType literal.
func TransformerDefinitions() map[string]any {
	defsOnce.Do(func() {
		defs = map[string]any{}
		_, file, _, _ := runtime.Caller(0)
		dir := filepath.Join(filepath.Dir(file), "..", "packages", "miroir-test-app_deployment-miroir", "assets", "miroir_data", "a557419d-a288-4fb8-8a1e-971c86c113b8")
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				continue
			}
			if name, _ := doc["name"].(string); name != "" {
				defs[name] = doc
			}
			if tType := transformerTypeOfDef(doc); tType != "" {
				defs[tType] = doc
			}
		}
	})
	return defs
}

// Definitions is an alias of [TransformerDefinitions].
func Definitions() map[string]any {
	return TransformerDefinitions()
}

func transformerTypeOfDef(doc map[string]any) string {
	iface, _ := doc["transformerInterface"].(map[string]any)
	if iface == nil {
		return ""
	}
	schema, _ := iface["transformerParameterSchema"].(map[string]any)
	if schema == nil {
		return ""
	}
	tt := schema["transformerType"]
	if m, ok := tt.(map[string]any); ok {
		if m["type"] == "literal" {
			if s, ok := m["definition"].(string); ok {
				return s
			}
		}
		if def, ok := m["definition"].([]any); ok && len(def) > 0 {
			if s, ok := def[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

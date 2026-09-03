package jzod

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	// BootstrapSchemaUUID is jzodMiroirBootstrapSchema (language self-description).
	BootstrapSchemaUUID = "1e8dab4b-65a3-4686-922e-ce89a2d62aa9"
	// FundamentalSchemaUUID is miroirFundamentalJzodSchema (absolutePath target).
	FundamentalSchemaUUID = "fe9b7d99-f216-44de-bb6e-60e1a1ebb739"
	// MiroirDeploymentUUID is deployment_Miroir.uuid on defaultMiroirModelEnvironment.
	MiroirDeploymentUUID = "10ff36f2-50cc-480d-9973-6d1a0482342e"
)

var (
	envOnce   sync.Once
	envCache  map[string]any
	fundCache map[string]any
	metaCache map[string]any
)

func envPackageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

func goRoot() string {
	return filepath.Join(envPackageDir(), "..")
}

// FundamentalSchema returns the generated miroirFundamentalJzodSchema JSON
// (uuid [FundamentalSchemaUUID]), falling back to bootstrap if the local
// JSON copy cannot be parsed.
func FundamentalSchema() map[string]any {
	loadEnv()
	return fundCache
}

// DefaultMiroirModelEnvironment returns a map shaped like TypeScript
// defaultMiroirModelEnvironment, including deploymentUuid.
func DefaultMiroirModelEnvironment() map[string]any {
	loadEnv()
	return envCache
}

// DefaultMetaModelEnvironment returns the same map as
// [DefaultMiroirModelEnvironment] without deploymentUuid, matching TypeScript
// defaultMetaModelEnvironment (the unit transformerTest host).
func DefaultMetaModelEnvironment() map[string]any {
	loadEnv()
	out := map[string]any{}
	for k, v := range envCache {
		if k == "deploymentUuid" {
			continue
		}
		out[k] = v
	}
	return out
}

// DefaultMiroirMetaModel returns the sparse currentModel / miroirMetaModel
// placeholder used by the Go unit environments (entities empty unless loaded
// elsewhere).
func DefaultMiroirMetaModel() map[string]any {
	loadEnv()
	return metaCache
}

func loadEnv() {
	envOnce.Do(func() {
		fundCache = loadFundamentalSchemaFile()
		metaCache = map[string]any{
			"entities":          []any{},
			"jzodSchemas":       []any{},
			"endpoints":         []any{},
			"entityDefinitions": []any{},
		}
		envCache = map[string]any{
			"miroirFundamentalJzodSchema": fundCache,
			"miroirMetaModel":             metaCache,
			"currentModel":                metaCache,
			"endpointsByUuid":             map[string]any{},
			"deploymentUuid":              MiroirDeploymentUUID,
		}
	})
}

func loadFundamentalSchemaFile() map[string]any {
	path := filepath.Join(goRoot(), "packages", "miroir-core", "src", "0_interfaces", "1_core", "preprocessor-generated", "miroirFundamentalJzodSchema.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return loadBootstrapAsFundamental()
	}
	decoded, err := Decode(raw)
	if err != nil {
		return loadBootstrapAsFundamental()
	}
	doc, ok := decoded.(map[string]any)
	if !ok {
		return loadBootstrapAsFundamental()
	}
	return doc
}

func loadBootstrapAsFundamental() map[string]any {
	path := filepath.Join(goRoot(), "packages", "miroir-test-app_deployment-miroir", "assets", "miroir_data", "5e81e1b9-38be-487c-b3e5-53796c57fccf", BootstrapSchemaUUID+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{"uuid": FundamentalSchemaUUID, "definition": map[string]any{"type": "schemaReference", "context": map[string]any{}}}
	}
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	doc["uuid"] = FundamentalSchemaUUID
	return doc
}

// EnvironmentFromMap builds a [ModelEnvironment] from a default-* environment
// map, registering the fundamental schema and any currentModel.jzodSchemas.
func EnvironmentFromMap(v any) ModelEnvironment {
	env := ModelEnvironment{}
	m, _ := v.(map[string]any)
	if m == nil {
		return env
	}
	if fund, ok := m["miroirFundamentalJzodSchema"].(map[string]any); ok {
		env.AbsoluteSchemas = append(env.AbsoluteSchemas, fund)
	}
	if current, ok := m["currentModel"].(map[string]any); ok {
		if schemas, ok := current["jzodSchemas"].([]any); ok {
			for _, s := range schemas {
				if sm, ok := s.(map[string]any); ok {
					env.AbsoluteSchemas = append(env.AbsoluteSchemas, sm)
				}
			}
		}
	}
	return env
}

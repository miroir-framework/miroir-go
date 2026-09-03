package jzod

import (
	"encoding/json"
	"fmt"
)

// ResolveJzodSchemaReferenceInContext resolves a schemaReference (or object)
// against relativeContext plus the fundamental / model environment (TS
// resolveJzodSchemaReferenceInContext).
func ResolveJzodSchemaReferenceInContext(jzodReference any, relativeContext map[string]any, envMap any) (any, error) {
	if list, ok := jzodReference.([]any); ok {
		resolved := make([]any, len(list))
		allObjects := true
		for i, ref := range list {
			if ref == nil {
				allObjects = false
				continue
			}
			item, err := ResolveJzodSchemaReferenceInContext(ref, relativeContext, envMap)
			if err != nil {
				return nil, err
			}
			resolved[i] = item
			im, ok := item.(map[string]any)
			if !ok || im["type"] != "object" {
				allObjects = false
			}
		}
		if allObjects {
			merged := map[string]any{}
			for _, item := range resolved {
				im, _ := item.(map[string]any)
				if def, ok := im["definition"].(map[string]any); ok {
					for k, v := range def {
						merged[k] = v
					}
				}
			}
			return map[string]any{"type": "object", "definition": merged}, nil
		}
		b, _ := json.Marshal(resolved)
		return nil, fmt.Errorf("resolveJzodSchemaReferenceInContext can not handle array of references with mixed types or non-object definitions: %s", b)
	}
	ref, ok := jzodReference.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("resolveJzodSchemaReferenceInContext expected object")
	}
	if ref["type"] == "object" {
		b, _ := json.Marshal(ref)
		return nil, fmt.Errorf("resolveJzodSchemaReferenceInContext can not handle object reference %s", b)
	}
	def := refDefinition(ref)
	env := envMapToLookup(envMap)
	var targetContext map[string]any
	if abs, _ := def["absolutePath"].(string); abs != "" {
		found, ok := env.contextForUUID(abs)
		if !ok {
			return nil, fmt.Errorf("resolveJzodSchemaReferenceInContext could not resolve reference %v", def)
		}
		targetContext = found
	} else if relativeContext != nil {
		targetContext = relativeContext
	} else {
		targetContext = contextOf(ref)
	}
	if rel, _ := def["relativePath"].(string); rel != "" {
		got, ok := targetContext[rel]
		if !ok {
			return nil, fmt.Errorf("resolveJzodSchemaReferenceInContext could not resolve reference %v", def)
		}
		return got, nil
	}
	return map[string]any{"type": "object", "definition": targetContext}, nil
}

// RecursiveResolveJzodSchemaReferenceInContext keeps resolving until the result
// is no longer a schemaReference (TS recursiveResolveJzodSchemaReferenceInContext).
func RecursiveResolveJzodSchemaReferenceInContext(jzodReference any, relativeContext map[string]any, envMap any) (any, error) {
	resolved, err := ResolveJzodSchemaReferenceInContext(jzodReference, relativeContext, envMap)
	if err != nil {
		return nil, err
	}
	if m, ok := resolved.(map[string]any); ok && m["type"] == "schemaReference" {
		return RecursiveResolveJzodSchemaReferenceInContext(resolved, relativeContext, envMap)
	}
	return resolved, nil
}

func envMapToLookup(envMap any) ModelEnvironment {
	if env, ok := envMap.(ModelEnvironment); ok {
		return env
	}
	return EnvironmentFromMap(envMap)
}

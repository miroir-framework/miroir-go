package jzod

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// JzodObjectFlatten resolves an object's extend clause and merges parent
// definition keys under the child (TS jzodObjectFlatten).
func JzodObjectFlatten(obj any, envMap any, relativeContext map[string]any) (any, error) {
	el, ok := obj.(map[string]any)
	if !ok {
		return obj, nil
	}
	if el["extend"] == nil {
		return obj, nil
	}
	parent, err := getAllProperties(el["extend"], nil, envMap, relativeContext)
	if err != nil {
		return nil, err
	}
	def := map[string]any{}
	for k, v := range parent.properties {
		def[k] = v
	}
	if own, ok := el["definition"].(map[string]any); ok {
		for k, v := range own {
			def[k] = v
		}
	}
	out := map[string]any{"type": "object", "definition": def}
	if v, ok := el["optional"]; ok {
		out["optional"] = v
	}
	if v, ok := el["nullable"]; ok {
		out["nullable"] = v
	}
	if v, ok := el["tag"]; ok {
		out["tag"] = v
	} else if parent.tag != nil {
		out["tag"] = parent.tag
	}
	return out, nil
}

type flattenProps struct {
	properties map[string]any
	tag        any
}

func getAllProperties(parent any, chain []any, envMap any, relativeContext map[string]any) (flattenProps, error) {
	if list, ok := parent.([]any); ok {
		all := map[string]any{}
		var tag any
		for _, p := range list {
			if p == nil {
				continue
			}
			props, err := getAllProperties(p, chain, envMap, relativeContext)
			if err != nil {
				return flattenProps{}, err
			}
			for k, v := range props.properties {
				all[k] = v
			}
			if tag == nil {
				if pm, ok := p.(map[string]any); ok && pm["tag"] != nil {
					tag = pm["tag"]
				} else if props.tag != nil {
					tag = props.tag
				}
			}
		}
		return flattenProps{properties: all, tag: tag}, nil
	}
	el, ok := parent.(map[string]any)
	if !ok {
		return flattenProps{properties: map[string]any{}}, nil
	}
	if el["type"] == "schemaReference" {
		for _, ref := range chain {
			if reflect.DeepEqual(ref, parent) {
				b, _ := json.Marshal(chain)
				return flattenProps{}, fmt.Errorf("jzodObjectFlatten: Circular reference detected. Reference chain: %s", b)
			}
		}
		env := envMapToLookup(envMap)
		if len(env.AbsoluteSchemas) == 0 && envMap == nil {
			return flattenProps{}, fmt.Errorf("jzodObjectFlatten: Cannot resolve schema reference without miroirFundamentalJzodSchema")
		}
		resolved, err := ResolveJzodSchemaReferenceInContext(el, orEmpty(relativeContext), envMap)
		if err != nil {
			return flattenProps{}, err
		}
		newChain := append(append([]any{}, chain...), el)
		if rm, ok := resolved.(map[string]any); ok && rm["type"] == "schemaReference" {
			return getAllProperties(resolved, newChain, envMap, relativeContext)
		}
		if rm, ok := resolved.(map[string]any); ok && rm["type"] != "object" {
			return flattenProps{}, fmt.Errorf("jzodObjectFlatten: Schema reference resolved to non-object type '%v'. Only object types can be used in extend clauses.", rm["type"])
		}
		return getAllProperties(resolved, newChain, envMap, relativeContext)
	}
	if el["type"] == "object" {
		properties := map[string]any{}
		if def, ok := el["definition"].(map[string]any); ok {
			for k, v := range def {
				properties[k] = v
			}
		}
		if el["extend"] != nil {
			parentProps, err := getAllProperties(el["extend"], chain, envMap, relativeContext)
			if err != nil {
				return flattenProps{}, err
			}
			merged := map[string]any{}
			for k, v := range parentProps.properties {
				merged[k] = v
			}
			for k, v := range properties {
				merged[k] = v
			}
			tag := el["tag"]
			if tag == nil {
				tag = parentProps.tag
			}
			return flattenProps{properties: merged, tag: tag}, nil
		}
		return flattenProps{properties: properties, tag: el["tag"]}, nil
	}
	return flattenProps{properties: map[string]any{}}, nil
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

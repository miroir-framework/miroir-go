package transformer

// InferTransformerOutputTypeFromSchema is [InferOutputTypeFromSchema].
func InferTransformerOutputTypeFromSchema(resultSchema any, options any) any {
	return InferOutputTypeFromSchema(resultSchema, options)
}

// InferOutputTypeFromSchema maps a transformerResultSchema to the compact
// output-type tag used by functionCallTest (TS inferTransformerOutputTypeFromSchema).
func InferOutputTypeFromSchema(resultSchema any, options any) any {
	m, ok := asMap(resultSchema)
	if !ok || m == nil {
		return "any"
	}
	t, _ := m["type"].(string)
	switch t {
	case "any":
		return "any"
	case "undefined":
		return "undefined"
	case "bigint", "number", "string", "boolean":
		return t
	case "object":
		opts, _ := asMap(options)
		if opts != nil {
			rowEntityUuid, _ := opts["rowEntityUuid"].(string)
			if rowEntityUuid != "" && opts["rowMlSchema"] != nil && jsonEqual(resultSchema, opts["rowMlSchema"]) {
				return rowEntityUuid
			}
		}
		return "object"
	case "array":
		var elementSchema any
		if def, ok := m["definition"].([]any); ok && len(def) > 0 {
			elementSchema = def[0]
		} else if defMap, ok := asMap(m["definition"]); ok && defMap != nil {
			if _, hasType := defMap["type"]; hasType {
				elementSchema = defMap
			}
		}
		if elementSchema == nil {
			return map[string]any{"type": "array", "payload": "any"}
		}
		payload := InferOutputTypeFromSchema(elementSchema, options)
		if _, isObj := asMap(payload); isObj {
			return map[string]any{"type": "array", "payload": "any"}
		}
		return map[string]any{"type": "array", "payload": payload}
	default:
		return "any"
	}
}

// InferElementTransformerOutputType infers the output type of a list-element
// transformer given the row schema (TS inferElementTransformerOutputType).
func InferElementTransformerOutputType(elementTransformer any, rowMlSchema any, rowEntityUuid any) any {
	if rowMlSchema == nil || !isTypedTransformer(elementTransformer) {
		return nil
	}
	resolved := ResolveResultSchema(elementTransformer, map[string]any{"row": rowMlSchema}, Definitions())
	if isFailed(resolved) {
		return nil
	}
	uuid, _ := rowEntityUuid.(string)
	return InferOutputTypeFromSchema(resolved, map[string]any{
		"rowEntityUuid": uuid,
		"rowMlSchema":   rowMlSchema,
	})
}

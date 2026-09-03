package transformer

import "fmt"

func resultSchemaOf(def map[string]any) map[string]any {
	iface, _ := def["transformerInterface"].(map[string]any)
	if iface == nil {
		return nil
	}
	rs, _ := iface["transformerResultSchema"].(map[string]any)
	return rs
}

// ResolveResultSchema evaluates transformerResultSchema for xf (TS
// resolveTransformerResultSchema).
func ResolveResultSchema(xf any, context map[string]any, defs map[string]any) any {
	if context == nil {
		context = map[string]any{}
	}
	if defs == nil {
		defs = Definitions()
	}
	if !isTypedTransformer(xf) {
		return fail("missingTransformerType", "resolveTransformerResultSchema: transformer missing transformerType", map[string]any{
			"typePath": []any{},
		})
	}
	tt := transformerType(xf)
	rawDef, ok := defs[tt]
	if !ok {
		return fail("unknownTransformerType", fmt.Sprintf("resolveTransformerResultSchema: unknown transformerType %q", tt), map[string]any{
			"transformerType": tt,
			"typePath":        []any{"transformerType"},
		})
	}
	def, _ := asMap(rawDef)
	resultSchema := resultSchemaOf(def)
	if resultSchema == nil {
		return fail("missingTransformerResultSchema", fmt.Sprintf("resolveTransformerResultSchema: transformer %q has no transformerResultSchema", tt), map[string]any{
			"transformerType": tt,
			"typePath":        []any{"transformerResultSchema"},
		})
	}

	if resultSchema["returns"] == "mlSchemaTransformer" {
		addAttrs, _ := resultSchema["addAttributesToContextBeingSubtypeOf"].(map[string]any)
		derivation := buildMlSchemaTransformerContext(xf, context, defs, addAttrs)
		if isFailed(derivation) {
			return derivation
		}
		derived, _ := asMap(derivation)
		for _, attr := range sortedKeys(addAttrs) {
			applyToSchema := derived[attr]
			applyToTransformer := field(xf, attr)
			if applyToSchema != nil && isTypedTransformer(applyToTransformer) {
				expected := addAttrs[attr]
				if expected == nil {
					expected = map[string]any{"type": "never"}
				}
				expectedRoot := schemaType(expected)
				if expectedRoot == "" {
					expectedRoot = "unknown"
				}
				if shape := validateApplyToSchemaShape(tt, applyToTransformer, applyToSchema, expectedRoot, expected); shape != nil {
					return shape
				}
			}
		}
		return ResolveResultSchema(resultSchema["definition"], derived, defs)
	}

	xfMap, _ := asMap(xf)
	switch tt {
	case "returnValue":
		if ml := xfMap["mlSchema"]; ml != nil {
			return ml
		}
	case "getFromContext":
		return resolveReferenceSchema(xfMap, context, resultSchema["definition"], "getFromContext")
	case "getFromParameters":
		return resolveReferenceSchema(xfMap, context, resultSchema["definition"], "getFromParameters")
	case "accessDynamicPath":
		return resolveAccessDynamicPathSchema(xfMap, context, defs)
	case "boolExpr":
		op, _ := xfMap["operator"].(string)
		if boolExprOperatorRequiresBooleanOperands(op) {
			if leftFail := validateBooleanOperand(xfMap["left"], context, defs, "boolExpr", "left"); leftFail != nil {
				return leftFail
			}
			if xfMap["right"] != nil && op != "!" {
				if rightFail := validateBooleanOperand(xfMap["right"], context, defs, "boolExpr", "right"); rightFail != nil {
					return rightFail
				}
			}
		}
	case "numericOp":
		args, _ := xfMap["args"].([]any)
		for i, arg := range args {
			argSchema := resolveOperandSchema(arg, context, defs, "numericOp", i)
			if isFailed(argSchema) {
				return argSchema
			}
			if argFail := requireSchemaRootType(argSchema, "number", map[string]any{"type": "number"}, map[string]any{
				"transformerType": "numericOp",
				"typePath":        []any{"numericOp", "args", i},
				"errorMessage":    fmt.Sprintf("numericOp arg %d must resolve to number schema but got %q", i, unknownType(argSchema)),
			}); argFail != nil {
				return argFail
			}
		}
		return map[string]any{"type": "number"}
	case "mapList":
		if applyTo := xfMap["applyTo"]; isTypedTransformer(applyTo) {
			applyToSchema := ResolveResultSchema(applyTo, context, defs)
			if isFailed(applyToSchema) {
				return applyToSchema
			}
			if shape := validateApplyToSchemaShape(tt, applyTo, applyToSchema, "array", map[string]any{"type": "array", "definition": map[string]any{"type": "any"}}); shape != nil {
				return shape
			}
		}
		elementSchema := resolveOperandSchema(xfMap["elementTransformer"], context, defs, "mapList", "elementTransformer")
		if isFailed(elementSchema) {
			return elementSchema
		}
		return map[string]any{"type": "array", "definition": elementSchema}
	case "stringOp":
		if xfMap["op"] == "length" && isTypedTransformer(xfMap["applyTo"]) {
			applyToSchema := ResolveResultSchema(xfMap["applyTo"], context, defs)
			if isFailed(applyToSchema) {
				return applyToSchema
			}
			if shape := validateApplyToSchemaShape(tt, xfMap["applyTo"], applyToSchema, "string", map[string]any{"type": "string"}); shape != nil {
				return shape
			}
		}
	case "dataflowObject":
		def, _ := xfMap["definition"].(map[string]any)
		return resolveRecordTransformerDefinitionSchema(def, context, defs, true, "dataflowObject")
	case "ifThenElse":
		if ifFail := validateBooleanOperand(xfMap["if"], context, defs, "ifThenElse", "if"); ifFail != nil {
			return ifFail
		}
		if xfMap["then"] != nil && xfMap["else"] != nil {
			thenSchema := ResolveResultSchema(xfMap["then"], context, defs)
			if isFailed(thenSchema) {
				return thenSchema
			}
			elseSchema := ResolveResultSchema(xfMap["else"], context, defs)
			if isFailed(elseSchema) {
				return elseSchema
			}
			return map[string]any{"type": "union", "definition": []any{thenSchema, elseSchema}}
		}
		if xfMap["then"] != nil {
			return ResolveResultSchema(xfMap["then"], context, defs)
		}
		if xfMap["else"] != nil {
			return ResolveResultSchema(xfMap["else"], context, defs)
		}
		return map[string]any{"type": "boolean"}
	case "createObject":
		def, _ := xfMap["definition"].(map[string]any)
		return resolveRecordTransformerDefinitionSchema(def, context, defs, false, "createObject")
	case "filterList":
		elementSchema := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, "filterList")
		if isFailed(elementSchema) {
			return elementSchema
		}
		if pred := resolveListPredicateBoolean(xfMap["predicate"], context, defs, "filterList"); pred != nil {
			return pred
		}
		return map[string]any{"type": "array", "definition": elementSchema}
	case "sortList":
		elementSchema := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, "sortList")
		if isFailed(elementSchema) {
			return elementSchema
		}
		return map[string]any{"type": "array", "definition": elementSchema}
	case "listLength":
		listFail := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, "listLength")
		if isFailed(listFail) {
			return listFail
		}
		return map[string]any{"type": "number"}
	case "find":
		elementSchema := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, "find")
		if isFailed(elementSchema) {
			return elementSchema
		}
		if pred := resolveListPredicateBoolean(xfMap["predicate"], context, defs, "find"); pred != nil {
			return pred
		}
		return elementSchema
	case "concatLists":
		lists, _ := xfMap["lists"].([]any)
		elementSchemas := []any{}
		for i, listTransformer := range lists {
			if !isTypedTransformer(listTransformer) {
				continue
			}
			listSchema := resolveOperandSchema(listTransformer, context, defs, "concatLists", i)
			if isFailed(listSchema) {
				return listSchema
			}
			if shape := validateApplyToSchemaShape("concatLists", listTransformer, listSchema, "array", map[string]any{"type": "array", "definition": map[string]any{"type": "any"}}); shape != nil {
				sm, _ := asMap(shape)
				out := copyMap(sm)
				out["typePath"] = []any{"concatLists", "lists", i}
				return out
			}
			elementSchemas = append(elementSchemas, unwrapArrayElementSchema(listSchema))
		}
		if len(elementSchemas) == 0 {
			return map[string]any{"type": "array", "definition": map[string]any{"type": "any"}}
		}
		allSame := len(elementSchemas) > 1
		if allSame {
			for _, schema := range elementSchemas[1:] {
				if !jsonEqual(schema, elementSchemas[0]) {
					allSame = false
					break
				}
			}
		}
		def := buildUnionSchema(elementSchemas)
		if allSame {
			def = elementSchemas[0]
		}
		return map[string]any{"type": "array", "definition": def}
	case "getObjectValues":
		applyToSchema := resolveApplyToObjectSchema(xfMap["applyTo"], context, defs, "getObjectValues")
		if isFailed(applyToSchema) {
			return applyToSchema
		}
		objectDefinition := getObjectDefinitionMap(applyToSchema)
		if objectDefinition == nil {
			return map[string]any{"type": "array", "definition": map[string]any{"type": "any"}}
		}
		return map[string]any{"type": "array", "definition": buildUnionSchema(mapValues(objectDefinition))}
	case "getObjectEntries":
		applyToSchema := resolveApplyToObjectSchema(xfMap["applyTo"], context, defs, "getObjectEntries")
		if isFailed(applyToSchema) {
			return applyToSchema
		}
		return map[string]any{"type": "array", "definition": map[string]any{"type": "any"}}
	case "getUniqueValues":
		elementSchema := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, "getUniqueValues")
		if isFailed(elementSchema) {
			return elementSchema
		}
		attribute := fmt.Sprint(xfMap["attribute"])
		objectDefinition := getObjectDefinitionMap(elementSchema)
		if objectDefinition != nil {
			if attrSchema, ok := objectDefinition[attribute]; ok {
				return map[string]any{"type": "array", "definition": attrSchema}
			}
		}
		return map[string]any{"type": "array", "definition": map[string]any{"type": "any"}}
	case "indexListBy", "listReducerToSpreadObject":
		elementSchema := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, tt)
		if isFailed(elementSchema) {
			return elementSchema
		}
		return map[string]any{"type": "record", "definition": elementSchema}
	case "object_fromEntries":
		entriesFail := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, "object_fromEntries")
		if isFailed(entriesFail) {
			return entriesFail
		}
		return map[string]any{"type": "record", "definition": map[string]any{"type": "any"}}
	case "mergeIntoObject":
		baseSchema := any(map[string]any{"type": "object", "definition": map[string]any{}})
		if isTypedTransformer(xfMap["applyTo"]) {
			applyToSchema := resolveApplyToObjectSchema(xfMap["applyTo"], context, defs, "mergeIntoObject")
			if isFailed(applyToSchema) {
				return applyToSchema
			}
			baseSchema = applyToSchema
		}
		overlaySchema := resolveOperandSchema(xfMap["definition"], context, defs, "mergeIntoObject", "definition")
		if isFailed(overlaySchema) {
			return overlaySchema
		}
		if isObjectLikeSchema(overlaySchema) || getObjectDefinitionMap(overlaySchema) != nil {
			return mergeObjectSchemas(baseSchema, overlaySchema)
		}
		return baseSchema
	case "createObjectFromPairs":
		pairs, _ := xfMap["definition"].([]any)
		objectDefinition := map[string]any{}
		for i, raw := range pairs {
			pair, _ := asMap(raw)
			key, _ := pair["attributeKey"].(string)
			if key == "" {
				key = fmt.Sprintf("key%d", i)
			}
			valueSchema := resolveOperandSchema(pair["attributeValue"], context, defs, "createObjectFromPairs", fmt.Sprintf("definition.%d.attributeValue", i))
			if isFailed(valueSchema) {
				return valueSchema
			}
			objectDefinition[key] = valueSchema
		}
		return map[string]any{"type": "object", "definition": objectDefinition}
	case "case":
		return resolveCaseBranchSchemas(xfMap, context, defs)
	case "constantAsExtractor":
		if schema := xfMap["valueJzodSchema"]; schema != nil {
			return schema
		}
	case "aggregate":
		aggregateFail := resolveApplyToArrayElementSchema(xfMap["applyTo"], context, defs, "aggregate")
		if isFailed(aggregateFail) {
			return aggregateFail
		}
	case "resolveTransformerResultSchema":
		nested := xfMap["transformer"]
		if nested == nil {
			return fail("schemaShapeMismatch", "resolveTransformerResultSchema: resolveTransformerResultSchema requires transformer parameter", map[string]any{
				"transformerType": tt,
				"typePath":        []any{"transformer"},
			})
		}
		nestedCtx, _ := xfMap["context"].(map[string]any)
		if nestedCtx == nil {
			nestedCtx = map[string]any{}
		}
		return ResolveResultSchema(nested, nestedCtx, defs)
	}

	return resultSchema["definition"]
}

func field(xf any, name string) any {
	m, ok := asMap(xf)
	if !ok {
		return nil
	}
	return m[name]
}

func buildMlSchemaTransformerContext(xf any, context map[string]any, defs map[string]any, attributeNames map[string]any) any {
	derivation := copyMap(context)
	for _, attributeName := range sortedKeys(attributeNames) {
		operand := field(xf, attributeName)
		if operand == nil || !isTypedTransformer(operand) {
			continue
		}
		operandSchema := ResolveResultSchema(operand, context, defs)
		if isFailed(operandSchema) {
			return operandSchema
		}
		derivation[attributeName] = operandSchema
	}
	return derivation
}

func resolveReferenceSchema(xf map[string]any, context map[string]any, fallback any, transformerType string) any {
	if name, ok := xf["referenceName"].(string); ok && name != "" {
		schema, exists := context[name]
		if !exists || schema == nil {
			return fail("contextMissingReference", fmt.Sprintf("resolveTransformerResultSchema: context missing reference %q", name), map[string]any{
				"transformerType": transformerType,
				"referenceName":   name,
				"referencePath":   xf["referencePath"],
				"typePath":        []any{transformerType, "referenceName"},
			})
		}
		return schema
	}
	path := stringSlice(xf["referencePath"])
	if len(path) > 0 {
		rootKey := path[0]
		current := context[rootKey]
		if current == nil {
			return fail("contextMissingReference", fmt.Sprintf("resolveTransformerResultSchema: context missing reference %q", rootKey), map[string]any{
				"transformerType": transformerType,
				"referenceName":   rootKey,
				"referencePath":   pathAny(path),
				"typePath":        []any{transformerType, "referencePath", 0},
			})
		}
		for i, segment := range path[1:] {
			m, ok := asMap(current)
			if !ok || m == nil {
				return fail("contextPathNotFound", fmt.Sprintf("resolveTransformerResultSchema: context path %q not found", joinPath(path)), map[string]any{
					"transformerType": transformerType,
					"referenceName":   xf["referenceName"],
					"referencePath":   pathAny(path),
					"actualSchema":    current,
					"typePath":        []any{transformerType, "referencePath", i + 1},
				})
			}
			def := m["definition"]
			defMap, ok := asMap(def)
			if !ok || defMap == nil {
				return fail("contextPathNotFound", fmt.Sprintf("resolveTransformerResultSchema: context path %q not found", joinPath(path)), map[string]any{
					"transformerType": transformerType,
					"referenceName":   xf["referenceName"],
					"referencePath":   pathAny(path),
					"actualSchema":    current,
					"typePath":        []any{transformerType, "referencePath", i + 1},
				})
			}
			current = defMap[segment]
			if current == nil {
				return fail("contextPathNotFound", fmt.Sprintf("resolveTransformerResultSchema: context path %q not found", joinPath(path)), map[string]any{
					"transformerType": transformerType,
					"referenceName":   xf["referenceName"],
					"referencePath":   pathAny(path),
					"typePath":        []any{transformerType, "referencePath", i + 1},
				})
			}
		}
		return current
	}
	return fallback
}

func resolveAccessDynamicPathSchema(xf map[string]any, context map[string]any, defs map[string]any) any {
	var current any
	path, _ := xf["objectAccessPath"].([]any)
	for index, segment := range path {
		if seg, ok := segment.(string); ok {
			if index == 0 {
				return fail("accessDynamicPathFailure", "resolveTransformerResultSchema: accessDynamicPath path must start with a transformer segment", map[string]any{
					"transformerType": "accessDynamicPath",
					"typePath":        []any{"accessDynamicPath", "objectAccessPath", index},
				})
			}
			if isFailed(current) {
				return current
			}
			m, ok := asMap(current)
			if !ok || m == nil {
				return fail("accessDynamicPathFailure", fmt.Sprintf("resolveTransformerResultSchema: accessDynamicPath segment %q on non-object schema", seg), map[string]any{
					"transformerType": "accessDynamicPath",
					"actualSchema":    current,
					"typePath":        []any{"accessDynamicPath", "objectAccessPath", index},
				})
			}
			next, exists := m[seg]
			if !exists {
				return fail("accessDynamicPathFailure", fmt.Sprintf("resolveTransformerResultSchema: accessDynamicPath segment %q not found", seg), map[string]any{
					"transformerType": "accessDynamicPath",
					"actualSchema":    current,
					"typePath":        []any{"accessDynamicPath", "objectAccessPath", index},
				})
			}
			current = next
			continue
		}
		resolved := ResolveResultSchema(segment, context, defs)
		if isFailed(resolved) {
			return resolved
		}
		current = resolved
	}
	if current == nil {
		return fail("accessDynamicPathFailure", "resolveTransformerResultSchema: accessDynamicPath resolved undefined", map[string]any{
			"transformerType": "accessDynamicPath",
			"typePath":        []any{"accessDynamicPath"},
		})
	}
	if isFailed(current) {
		return current
	}
	return current
}

func resolveRecordTransformerDefinitionSchema(definition map[string]any, context map[string]any, defs map[string]any, threadContext bool, transformerType string) any {
	if definition == nil {
		return map[string]any{"type": "object", "definition": map[string]any{}}
	}
	objectDefinition := map[string]any{}
	stepContext := copyMap(context)
	keys := sortedKeys(definition)
	if threadContext {
		remaining := append([]string{}, keys...)
		for len(remaining) > 0 {
			progress := false
			var next []string
			var lastFail any
			var lastKey string
			for _, key := range remaining {
				ctx := stepContext
				if !threadContext {
					ctx = context
				}
				nested := ResolveResultSchema(definition[key], ctx, defs)
				if isFailed(nested) && isContextMissing(nested) {
					lastFail = nested
					lastKey = key
					next = append(next, key)
					continue
				}
				if isFailed(nested) {
					return wrapRecordFailure(nested, transformerType, key)
				}
				objectDefinition[key] = nested
				stepContext[key] = nested
				progress = true
			}
			if !progress {
				if lastFail != nil {
					return wrapRecordFailure(lastFail, transformerType, lastKey)
				}
				break
			}
			remaining = next
		}
		return map[string]any{"type": "object", "definition": objectDefinition}
	}
	for _, key := range keys {
		nested := ResolveResultSchema(definition[key], context, defs)
		if isFailed(nested) {
			return wrapRecordFailure(nested, transformerType, key)
		}
		objectDefinition[key] = nested
	}
	return map[string]any{"type": "object", "definition": objectDefinition}
}

func wrapRecordFailure(nested any, transformerType, key string) any {
	m, _ := asMap(nested)
	out := copyMap(m)
	out["transformerPath"] = []any{transformerType, "definition", key}
	out["innerError"] = nested
	return out
}

func isContextMissing(v any) bool {
	m, ok := asMap(v)
	return ok && m["failureKind"] == "contextMissingReference"
}

func validateApplyToSchemaShape(transformerType string, applyToTransformer any, applyToSchema any, expectedRootType string, expectedSchema any) any {
	return requireSchemaRootType(applyToSchema, expectedRootType, expectedSchema, map[string]any{
		"transformerType": transformerType,
		"typePath":        []any{transformerType, "applyTo"},
		"referenceName":   referenceNameOf(applyToTransformer),
		"referencePath":   referencePathOf(applyToTransformer),
		"errorMessage":    fmt.Sprintf("%s expected applyTo schema type %q but got %q", transformerType, expectedRootType, unknownType(applyToSchema)),
	})
}

func referenceNameOf(xf any) any {
	m, ok := asMap(xf)
	if !ok {
		return nil
	}
	tt, _ := m["transformerType"].(string)
	if tt != "getFromContext" && tt != "getFromParameters" {
		return nil
	}
	return m["referenceName"]
}

func referencePathOf(xf any) any {
	m, ok := asMap(xf)
	if !ok {
		return nil
	}
	tt, _ := m["transformerType"].(string)
	if tt != "getFromContext" && tt != "getFromParameters" {
		return nil
	}
	return m["referencePath"]
}

func requireSchemaRootType(schema any, expectedRootType string, expectedSchema any, details map[string]any) any {
	actual := schemaType(schema)
	if actual == expectedRootType {
		return nil
	}
	msg, _ := details["errorMessage"].(string)
	if msg == "" {
		msg = fmt.Sprintf("%s expected schema type %q but got %q", details["transformerType"], expectedRootType, unknownType(schema))
	}
	return fail("schemaShapeMismatch", "resolveTransformerResultSchema: "+msg, map[string]any{
		"transformerType": details["transformerType"],
		"referenceName":   details["referenceName"],
		"referencePath":   details["referencePath"],
		"expectedSchema":  expectedSchema,
		"actualSchema":    schema,
		"typePath":        details["typePath"],
	})
}

func resolveOperandSchema(operand any, context map[string]any, defs map[string]any, parent string, key any) any {
	result := ResolveResultSchema(operand, context, defs)
	if isFailed(result) {
		return wrapOperandFailure(result, parent, key)
	}
	return result
}

func validateBooleanOperand(operand any, context map[string]any, defs map[string]any, parent string, key any) any {
	operandSchema := resolveOperandSchema(operand, context, defs, parent, key)
	if isFailed(operandSchema) {
		return operandSchema
	}
	return requireSchemaRootType(operandSchema, "boolean", map[string]any{"type": "boolean"}, map[string]any{
		"transformerType": parent,
		"typePath":        []any{parent, key},
		"referenceName":   referenceNameOf(operand),
		"referencePath":   referencePathOf(operand),
		"errorMessage":    fmt.Sprintf("%s operand %q must resolve to boolean schema but got %q", parent, fmt.Sprint(key), unknownType(operandSchema)),
	})
}

func boolExprOperatorRequiresBooleanOperands(operator string) bool {
	return operator == "&&" || operator == "||" || operator == "!"
}

func isObjectLikeSchema(schema any) bool {
	root := schemaType(schema)
	return root == "object" || root == "record"
}

func getObjectDefinitionMap(schema any) map[string]any {
	if schemaType(schema) != "object" {
		return nil
	}
	m, ok := asMap(schema)
	if !ok {
		return nil
	}
	def, ok := asMap(m["definition"])
	if !ok || def == nil {
		return nil
	}
	return def
}

func unwrapArrayElementSchema(schema any) any {
	if schemaType(schema) == "array" {
		if m, ok := asMap(schema); ok {
			if def, exists := m["definition"]; exists {
				return def
			}
		}
	}
	return map[string]any{"type": "any"}
}

func buildUnionSchema(schemas []any) any {
	nonFailed := make([]any, 0, len(schemas))
	for _, schema := range schemas {
		if !isFailed(schema) {
			nonFailed = append(nonFailed, schema)
		}
	}
	if len(nonFailed) == 0 {
		return map[string]any{"type": "any"}
	}
	if len(nonFailed) == 1 {
		return nonFailed[0]
	}
	return map[string]any{"type": "union", "definition": nonFailed}
}

func mergeObjectSchemas(base, overlay any) any {
	baseDef := getObjectDefinitionMap(base)
	if baseDef == nil {
		baseDef = map[string]any{}
	}
	overlayDef := getObjectDefinitionMap(overlay)
	if overlayDef == nil {
		overlayDef = map[string]any{}
	}
	merged := copyMap(baseDef)
	for k, v := range overlayDef {
		merged[k] = v
	}
	return map[string]any{"type": "object", "definition": merged}
}

func resolveApplyToArrayElementSchema(applyTo any, context map[string]any, defs map[string]any, parent string) any {
	if !isTypedTransformer(applyTo) {
		return fail("schemaShapeMismatch", fmt.Sprintf("resolveTransformerResultSchema: %s requires applyTo transformer", parent), map[string]any{
			"transformerType": parent,
			"typePath":        []any{parent, "applyTo"},
			"expectedSchema":  map[string]any{"type": "array", "definition": map[string]any{"type": "any"}},
		})
	}
	applyToSchema := resolveOperandSchema(applyTo, context, defs, parent, "applyTo")
	if isFailed(applyToSchema) {
		return applyToSchema
	}
	if shape := validateApplyToSchemaShape(parent, applyTo, applyToSchema, "array", map[string]any{"type": "array", "definition": map[string]any{"type": "any"}}); shape != nil {
		return shape
	}
	return unwrapArrayElementSchema(applyToSchema)
}

func resolveApplyToObjectSchema(applyTo any, context map[string]any, defs map[string]any, parent string) any {
	if !isTypedTransformer(applyTo) {
		return fail("schemaShapeMismatch", fmt.Sprintf("resolveTransformerResultSchema: %s requires applyTo transformer", parent), map[string]any{
			"transformerType": parent,
			"typePath":        []any{parent, "applyTo"},
			"expectedSchema":  map[string]any{"type": "object", "definition": map[string]any{}},
		})
	}
	applyToSchema := resolveOperandSchema(applyTo, context, defs, parent, "applyTo")
	if isFailed(applyToSchema) {
		return applyToSchema
	}
	if !isObjectLikeSchema(applyToSchema) {
		return fail("schemaShapeMismatch", fmt.Sprintf("resolveTransformerResultSchema: %s expected applyTo schema type \"object\" or \"record\" but got %q", parent, unknownType(applyToSchema)), map[string]any{
			"transformerType": parent,
			"typePath":        []any{parent, "applyTo"},
			"referenceName":   referenceNameOf(applyTo),
			"referencePath":   referencePathOf(applyTo),
			"expectedSchema":  map[string]any{"type": "object", "definition": map[string]any{}},
			"actualSchema":    applyToSchema,
		})
	}
	return applyToSchema
}

func resolveListPredicateBoolean(predicate any, context map[string]any, defs map[string]any, parent string) any {
	return validateBooleanOperand(predicate, context, defs, parent, "predicate")
}

func resolveCaseBranchSchemas(xf map[string]any, context map[string]any, defs map[string]any) any {
	whens, _ := xf["whens"].([]any)
	branchSchemas := []any{}
	for i, raw := range whens {
		when, _ := asMap(raw)
		thenSchema := resolveOperandSchema(when["then"], context, defs, "case", fmt.Sprintf("whens.%d.then", i))
		if isFailed(thenSchema) {
			return thenSchema
		}
		branchSchemas = append(branchSchemas, thenSchema)
	}
	if xf["else"] != nil {
		elseSchema := resolveOperandSchema(xf["else"], context, defs, "case", "else")
		if isFailed(elseSchema) {
			return elseSchema
		}
		branchSchemas = append(branchSchemas, elseSchema)
	}
	if len(branchSchemas) == 0 {
		return map[string]any{"type": "any"}
	}
	return buildUnionSchema(branchSchemas)
}

func mapValues(m map[string]any) []any {
	out := make([]any, 0, len(m))
	for _, k := range sortedKeys(m) {
		out = append(out, m[k])
	}
	return out
}

func unknownType(schema any) string {
	if t := schemaType(schema); t != "" {
		return t
	}
	return "unknown"
}

func pathAny(path []string) []any {
	out := make([]any, len(path))
	for i, s := range path {
		out[i] = s
	}
	return out
}

func joinPath(path []string) string {
	out := ""
	for i, s := range path {
		if i > 0 {
			out += "."
		}
		out += s
	}
	return out
}

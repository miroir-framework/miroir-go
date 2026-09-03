package transformer

var entityIdentityAttributeNames = map[string]struct{}{
	"uuid":                        {},
	"parentName":                  {},
	"parentUuid":                  {},
	"parentDefinitionVersionUuid": {},
	"conceptLevel":                {},
}

// EvaluateVirtualAttributesOnInstance computes named virtual attributes on
// instance (TS evaluateVirtualAttributesOnInstance).
func EvaluateVirtualAttributesOnInstance(entity, instance any, neededNames any, _modelEnvironment any, transformerParams any) (any, error) {
	ent, _ := asMap(entity)
	inst, _ := asMap(instance)
	if inst == nil {
		inst = map[string]any{}
	}
	virtualNames := listVirtualAttributeNames(ent)
	virtualSet := map[string]struct{}{}
	for _, n := range virtualNames {
		virtualSet[n] = struct{}{}
	}
	result := copyMap(inst)
	for name := range virtualSet {
		delete(result, name)
	}
	contextResults := storedFieldsContext(inst, virtualSet)
	params, _ := asMap(transformerParams)
	if params == nil {
		params = map[string]any{}
	}
	definition := mlSchemaDefinition(ent)
	for _, name := range stringSlice(neededNames) {
		if _, ok := virtualSet[name]; !ok {
			continue
		}
		schema := definition[name]
		xf := virtualAttributeTransformer(schema)
		if xf == nil {
			continue
		}
		evaluated, err := Apply("runtime", xf, params, contextResults)
		if err != nil {
			return nil, err
		}
		result[name] = evaluated
	}
	return result, nil
}

// StripVirtualAttributesFromInstance removes virtual attribute keys from
// instance (TS stripVirtualAttributesFromInstance).
func StripVirtualAttributesFromInstance(entity, instance any) (any, error) {
	ent, _ := asMap(entity)
	inst, _ := asMap(instance)
	if inst == nil {
		return instance, nil
	}
	virtualNames := listVirtualAttributeNames(ent)
	if len(virtualNames) == 0 {
		return instance, nil
	}
	result := copyMap(inst)
	changed := false
	for _, name := range virtualNames {
		if _, ok := result[name]; ok {
			delete(result, name)
			changed = true
		}
	}
	if changed {
		return result, nil
	}
	return instance, nil
}

func mlSchemaDefinition(entity map[string]any) map[string]any {
	ml, _ := asMap(entity["mlSchema"])
	if ml == nil {
		return map[string]any{}
	}
	def, _ := asMap(ml["definition"])
	if def == nil {
		return map[string]any{}
	}
	return def
}

func virtualAttributeTransformer(schema any) any {
	m, ok := asMap(schema)
	if !ok {
		return nil
	}
	tag, _ := asMap(m["tag"])
	if tag == nil {
		return nil
	}
	value, _ := asMap(tag["value"])
	if value == nil {
		return nil
	}
	return value["virtualAttribute"]
}

func isVirtualAttribute(schema any) bool {
	return virtualAttributeTransformer(schema) != nil
}

func listVirtualAttributeNames(entity map[string]any) []string {
	def := mlSchemaDefinition(entity)
	var out []string
	for _, name := range sortedKeys(def) {
		if _, skip := entityIdentityAttributeNames[name]; skip {
			continue
		}
		if isVirtualAttribute(def[name]) {
			out = append(out, name)
		}
	}
	return out
}

func storedFieldsContext(instance map[string]any, virtualNames map[string]struct{}) map[string]any {
	context := map[string]any{}
	for k, v := range instance {
		if _, virt := virtualNames[k]; !virt {
			context[k] = v
		}
	}
	return context
}

package jzod

import "fmt"

var jzodToSQL = map[string]string{
	"boolean": "boolean",
	"bigint":  "double precision",
	"date":    "date",
	"number":  "double precision",
	"string":  "text",
	"uuid":    "text",
	"enum":    "text",
}

// GetAttributeTypesFromJzodSchema maps object-schema attributes to SQL-ish
// type names (TS getAttributeTypesFromJzodSchema).
func GetAttributeTypesFromJzodSchema(element any) (any, error) {
	el, ok := element.(map[string]any)
	if !ok || el["type"] == nil {
		return nil, fmt.Errorf("MlSchema has no type")
	}
	if el["type"] != "object" {
		return nil, fmt.Errorf("MlSchema type is not object")
	}
	def, _ := el["definition"].(map[string]any)
	out := map[string]any{}
	for k, v := range def {
		child, _ := v.(map[string]any)
		t, _ := child["type"].(string)
		sql, ok := jzodToSQL[t]
		if !ok {
			return nil, fmt.Errorf("Jzod type %s not supported", t)
		}
		out[k] = sql
	}
	return out, nil
}

var postgresToJzod = map[string]string{
	"bigint":                      "bigint",
	"boolean":                     "boolean",
	"character":                   "string",
	"character varying":           "string",
	"date":                        "date",
	"double precision":            "number",
	"integer":                     "number",
	"json":                        "object",
	"jsonb":                       "object",
	"numeric":                     "number",
	"real":                        "number",
	"smallint":                    "number",
	"text":                        "string",
	"timestamp with time zone":    "date",
	"timestamp without time zone": "date",
	"uuid":                        "uuid",
}

// AnsiColumnsToJzodSchema builds an object Jzod schema from ANSI information-
// schema column rows (TS ansiColumnsToJzodSchema).
func AnsiColumnsToJzodSchema(columns any) (any, error) {
	list, _ := columns.([]any)
	sorted := append([]any{}, list...)
	// insertion order is enough if tests already sort; still sort by ordinal_position
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if ordinal(sorted[j]) < ordinal(sorted[i]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	def := map[string]any{}
	for _, raw := range sorted {
		col, _ := raw.(map[string]any)
		dataType, _ := col["data_type"].(string)
		jzodType, ok := postgresToJzod[dataType]
		if !ok {
			return nil, fmt.Errorf("Postgres data_type %s not supported", dataType)
		}
		name, _ := col["column_name"].(string)
		field := map[string]any{
			"type": jzodType,
			"tag": map[string]any{
				"value": map[string]any{
					"id":           ordinal(raw),
					"defaultLabel": name,
				},
			},
		}
		if jzodType == "object" {
			field["definition"] = map[string]any{}
		}
		if col["is_nullable"] == "YES" {
			field["optional"] = true
		}
		def[name] = field
	}
	return map[string]any{"type": "object", "definition": def}, nil
}

func ordinal(col any) float64 {
	m, _ := col.(map[string]any)
	switch n := m["ordinal_position"].(type) {
	case float64:
		return n
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}

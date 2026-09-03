package transformer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func jsTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case float32:
		return t != 0
	case int:
		return t != 0
	case json.Number:
		f, _ := t.Float64()
		return f != 0
	default:
		return true
	}
}

func jsLooseEqual(a, b any) bool {
	if jsStrictEqual(a, b) {
		return true
	}
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	as, aNum, aIsNum := jsNumber(a)
	bs, bNum, bIsNum := jsNumber(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}
	if aIsNum {
		if s, ok := b.(string); ok {
			if s == "" {
				return aNum == 0
			}
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				return aNum == n
			}
		}
		if b, ok := b.(bool); ok {
			return aNum == boolNum(b)
		}
	}
	if bIsNum {
		if s, ok := a.(string); ok {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				return bNum == n
			}
		}
		if a, ok := a.(bool); ok {
			return bNum == boolNum(a)
		}
	}
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			return sa == sb
		}
	}
	_ = as
	_ = bs
	return false
}

func jsStrictEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if _, ok := a.(map[string]any); ok {
		return false
	}
	if _, ok := b.(map[string]any); ok {
		return false
	}
	if _, ok := a.([]any); ok {
		return false
	}
	if _, ok := b.([]any); ok {
		return false
	}
	_, an, aok := jsNumber(a)
	_, bn, bok := jsNumber(b)
	if aok && bok {
		return an == bn
	}
	return reflect.DeepEqual(a, b)
}

func jsNumber(v any) (string, float64, bool) {
	switch n := v.(type) {
	case float64:
		return "", n, true
	case float32:
		return "", float64(n), true
	case int:
		return "", float64(n), true
	case int64:
		return "", float64(n), true
	case json.Number:
		f, err := n.Float64()
		return string(n), f, err == nil
	default:
		return fmt.Sprint(v), 0, false
	}
}

func boolNum(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func jsLess(a, b any) bool {
	if _, an, aok := jsNumber(a); aok {
		if _, bn, bok := jsNumber(b); bok {
			return an < bn
		}
	}
	return fmt.Sprint(a) < fmt.Sprint(b)
}

func asList(v any) ([]any, bool) {
	if v == nil {
		return nil, false
	}
	if list, ok := v.([]any); ok {
		return list, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out, true
	}
	return nil, false
}

func toFloat(v any) (float64, bool) {
	_, n, ok := jsNumber(v)
	return n, ok
}

func jsString(v any) string {
	if v == nil {
		return "undefined"
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func deepEqualJSON(a, b any) bool {
	return jsonEqual(a, b)
}

func outerName(xf map[string]any) string {
	if s, ok := xf["referenceToOuterObject"].(string); ok && s != "" {
		return s
	}
	return defaultTransformerInput
}

func mustMap(v any) map[string]any {
	m, _ := asMap(v)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func stringify(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func localeCompare(a, b string) int {
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

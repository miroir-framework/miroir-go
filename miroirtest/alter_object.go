package miroirtest

import "fmt"

func alterObjectAtPath(args []any) (any, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("alterObjectAtPath: expected object, path, value")
	}
	object := args[0]
	path, _ := args[1].([]any)
	return alterObjectAtPathRec(object, path, args[2])
}

func alterObjectAtPathRec(object any, path []any, value any) (any, error) {
	if len(path) == 0 {
		return value, nil
	}
	if object == nil {
		return nil, fmt.Errorf("alterObjectAtPath could not access attribute %v for undefined object", path[0])
	}
	head := path[0]
	child, err := alterObjectAtPathRec(indexAny(object, head), path[1:], value)
	if err != nil {
		return nil, err
	}
	switch obj := object.(type) {
	case map[string]any:
		out := make(map[string]any, len(obj)+1)
		for k, v := range obj {
			out[k] = v
		}
		out[fmt.Sprint(head)] = child
		return out, nil
	default:
		return map[string]any{fmt.Sprint(head): child}, nil
	}
}

func indexAny(object any, head any) any {
	switch obj := object.(type) {
	case map[string]any:
		return obj[fmt.Sprint(head)]
	case []any:
		switch n := head.(type) {
		case float64:
			i := int(n)
			if i >= 0 && i < len(obj) {
				return obj[i]
			}
		case int:
			if n >= 0 && n < len(obj) {
				return obj[n]
			}
		}
	}
	return nil
}

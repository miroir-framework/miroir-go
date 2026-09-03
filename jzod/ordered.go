package jzod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"
)

var keyOrder sync.Map

func mapPtr(m map[string]any) uintptr {
	return reflect.ValueOf(m).Pointer()
}

// RememberKeys records JSON insertion order for m so [KeysOf] can replay it.
// Stale pointer reuse is rejected by [RememberedKeys].
func RememberKeys(m map[string]any, keys []string) {
	if m == nil || keys == nil {
		return
	}
	cp := append([]string(nil), keys...)
	keyOrder.Store(mapPtr(m), cp)
}

// RememberedKeys returns the insertion-order keys previously stored by
// [RememberKeys], if they still match the live map.
func RememberedKeys(m map[string]any) ([]string, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := keyOrder.Load(mapPtr(m))
	if !ok {
		return nil, false
	}
	keys, _ := v.([]string)
	if !rememberedKeysMatch(keys, m) {
		keyOrder.Delete(mapPtr(m))
		return nil, false
	}
	return keys, len(keys) > 0 || len(m) == 0
}

func rememberedKeysMatch(keys []string, m map[string]any) bool {
	if len(keys) != len(m) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

// KeysOf returns remembered insertion order, or a stable fallback order.
func KeysOf(m map[string]any) []string {
	if keys, ok := RememberedKeys(m); ok {
		return keys
	}
	return objectKeyOrder(m)
}

// Decode unmarshals JSON while recording object key insertion order via
// [RememberKeys]. Use this instead of encoding/json when order is observable.
func Decode(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	v, err := decodeValue(dec)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			return decodeObject(dec)
		}
		if t == '[' {
			return decodeArray(dec)
		}
		return nil, fmt.Errorf("unexpected delim %v", t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return t, nil
	}
}

func decodeObject(dec *json.Decoder) (map[string]any, error) {
	m := map[string]any{}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("expected object key, got %T", tok)
		}
		val, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
		m[key] = val
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	RememberKeys(m, keys)
	return m, nil
}

func decodeArray(dec *json.Decoder) ([]any, error) {
	var out []any
	for dec.More() {
		val, err := decodeValue(dec)
		if err != nil {
			return nil, err
		}
		out = append(out, val)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return out, nil
}

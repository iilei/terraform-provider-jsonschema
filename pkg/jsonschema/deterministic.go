package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// MarshalDeterministic marshals data to JSON with deterministic key ordering
func MarshalDeterministic(data interface{}) ([]byte, error) {
	return json.Marshal(sortKeys(data))
}

// sortKeys recursively sorts all map keys in the data structure to ensure deterministic output
func sortKeys(data interface{}) interface{} {
	v := reflect.ValueOf(data)

	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			// Create a new map with sorted keys
			sortedMap := make(map[string]interface{})
			keys := make([]string, 0, v.Len())

			// Collect all keys
			for _, key := range v.MapKeys() {
				keys = append(keys, key.String())
			}

			// Sort keys
			sort.Strings(keys)

			// Rebuild map with sorted keys and recursively sort values
			for _, key := range keys {
				value := v.MapIndex(reflect.ValueOf(key))
				sortedMap[key] = sortKeys(value.Interface())
			}

			return sortedMap
		}
		// For non-string keyed maps, return as-is (shouldn't happen in JSON)
		return data

	case reflect.Slice, reflect.Array:
		// Process each element in the slice/array
		length := v.Len()
		result := make([]interface{}, length)
		for i := 0; i < length; i++ {
			result[i] = sortKeys(v.Index(i).Interface())
		}
		return result

	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return sortKeys(v.Elem().Interface())

	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return sortKeys(v.Elem().Interface())

	default:
		// For primitive types (string, int, bool, etc.), return as-is
		return data
	}
}

// MarshalDeterministicString is a convenience function that returns deterministic JSON as a string
func MarshalDeterministicString(data interface{}) (string, error) {
	jsonBytes, err := MarshalDeterministic(data)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// CompactDeterministicJSON marshals data to compact, deterministic JSON
func CompactDeterministicJSON(data interface{}) ([]byte, error) {
	jsonBytes, err := MarshalDeterministic(data)
	if err != nil {
		return nil, err
	}

	// Compact the JSON to remove unnecessary whitespace
	var buf bytes.Buffer
	if err := json.Compact(&buf, jsonBytes); err != nil {
		return nil, fmt.Errorf("failed to compact JSON: %w", err)
	}

	return buf.Bytes(), nil
}

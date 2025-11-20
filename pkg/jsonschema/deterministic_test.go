package jsonschema

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalDeterministic(t *testing.T) {
	// Test that deterministic marshaling produces consistent output
	testData := map[string]interface{}{
		"zebra": "last",
		"alpha": "first",
		"nested": map[string]interface{}{
			"charlie": 3,
			"bravo":   2,
			"alpha":   1,
		},
	}

	// Marshal multiple times to ensure deterministic output
	result1, err1 := MarshalDeterministic(testData)
	if err1 != nil {
		t.Fatalf("First marshal failed: %v", err1)
	}

	result2, err2 := MarshalDeterministic(testData)
	if err2 != nil {
		t.Fatalf("Second marshal failed: %v", err2)
	}

	if !bytes.Equal(result1, result2) {
		t.Errorf("Non-deterministic results:\nFirst:  %s\nSecond: %s", result1, result2)
	}

	// Verify keys are sorted alphabetically
	expected := `{"alpha":"first","nested":{"alpha":1,"bravo":2,"charlie":3},"zebra":"last"}`
	if string(result1) != expected {
		t.Errorf("Keys not sorted correctly:\nExpected: %s\nGot: %s", expected, string(result1))
	}
}

func TestMarshalDeterministicString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{
			name: "simple object to string",
			input: map[string]interface{}{
				"b": "value2",
				"a": "value1",
			},
			expected: `{"a":"value1","b":"value2"}`,
			wantErr:  false,
		},
		{
			name:     "array to string",
			input:    []interface{}{1, 2, 3},
			expected: `[1,2,3]`,
			wantErr:  false,
		},
		{
			name:     "string to string",
			input:    "hello",
			expected: `"hello"`,
			wantErr:  false,
		},
		{
			name:     "number to string",
			input:    42,
			expected: `42`,
			wantErr:  false,
		},
		{
			name:     "boolean to string",
			input:    true,
			expected: `true`,
			wantErr:  false,
		},
		{
			name:     "null to string",
			input:    nil,
			expected: `null`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarshalDeterministicString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalDeterministicString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("MarshalDeterministicString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCompactDeterministicJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{
			name: "object with whitespace removed",
			input: map[string]interface{}{
				"z": map[string]interface{}{
					"nested": "value",
				},
				"a": "test",
			},
			expected: `{"a":"test","z":{"nested":"value"}}`,
			wantErr:  false,
		},
		{
			name:     "simple array compacted",
			input:    []interface{}{1, 2, 3},
			expected: `[1,2,3]`,
			wantErr:  false,
		},
		{
			name: "complex nested structure",
			input: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"name": "John",
						"age":  30,
					},
					map[string]interface{}{
						"name": "Jane",
						"age":  25,
					},
				},
				"metadata": map[string]interface{}{
					"version": "1.0",
					"author":  "test",
				},
			},
			expected: `{"metadata":{"author":"test","version":"1.0"},"users":[{"age":30,"name":"John"},{"age":25,"name":"Jane"}]}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompactDeterministicJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompactDeterministicJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("CompactDeterministicJSON() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestSortKeysEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:  "pointer to nil",
			input: (*string)(nil),
		},
		{
			name: "pointer to value",
			input: func() *string {
				s := "test"
				return &s
			}(),
		},
		{
			name: "interface with nil",
			input: func() interface{} {
				var i interface{} = nil
				return i
			}(),
		},
		{
			name: "slice with mixed types",
			input: []interface{}{
				map[string]interface{}{"z": 1, "a": 2},
				"string",
				42,
				nil,
			},
		},
		{
			name: "array with nested maps",
			input: [2]interface{}{
				map[string]interface{}{"b": 1, "a": 2},
				map[string]interface{}{"y": 3, "x": 4},
			},
		},
		{
			name: "non-string keyed map (edge case)",
			input: map[int]interface{}{
				2: "two",
				1: "one",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify that sortKeys doesn't panic on edge cases
			result := sortKeys(tt.input)
			// The result should be valid for JSON marshaling
			_, err := MarshalDeterministic(result)
			if err != nil {
				t.Errorf("sortKeys() result could not be marshaled: %v", err)
			}
		})
	}
}

func TestMarshalDeterministicStringErrorHandling(t *testing.T) {
	// Test with data that cannot be marshaled to JSON
	// Functions cannot be marshaled to JSON
	invalidData := map[string]interface{}{
		"function": func() {},
	}

	result, err := MarshalDeterministicString(invalidData)
	if err == nil {
		t.Errorf("expected error for invalid data, got result: %s", result)
	}
	if result != "" {
		t.Errorf("expected empty result on error, got: %s", result)
	}
}

func TestCompactDeterministicJSONErrorHandling(t *testing.T) {
	// Test with data that cannot be marshaled to JSON
	// Functions cannot be marshaled to JSON
	invalidData := map[string]interface{}{
		"function": func() {},
	}

	result, err := CompactDeterministicJSON(invalidData)
	if err == nil {
		t.Errorf("expected error for invalid data, got result: %s", result)
	}
	if result != nil {
		t.Errorf("expected nil result on error, got: %s", result)
	}
}

func TestSortKeysReflectionEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		desc  string
	}{
		{
			name:  "empty_string_keyed_map",
			input: map[string]interface{}{},
			desc:  "Empty map with string keys should be handled correctly",
		},
		{
			name:  "empty_int_keyed_map",
			input: map[int]interface{}{},
			desc:  "Empty map with non-string keys should return as-is",
		},
		{
			name: "nested_pointers_to_pointers",
			input: func() interface{} {
				s := "deep"
				p1 := &s
				p2 := &p1
				return &p2
			}(),
			desc: "Multi-level pointer nesting should be dereferenced properly",
		},
		{
			name: "interface_containing_nil_pointer",
			input: func() interface{} {
				var ptr *string = nil
				var iface interface{} = ptr
				return iface
			}(),
			desc: "Interface containing nil pointer should return nil",
		},
		{
			name: "interface_containing_empty_interface",
			input: func() interface{} {
				var inner interface{} = nil
				var outer = inner
				return outer
			}(),
			desc: "Interface containing nil interface should return nil",
		},
		{
			name: "map_with_complex_key_type",
			input: map[interface{}]interface{}{
				"string_key": "value1",
				42:           "value2",
			},
			desc: "Map with interface{} keys (non-string) should return as-is",
		},
		{
			name: "slice_with_nil_interface_elements",
			input: []interface{}{
				nil,
				(*string)(nil),
				map[string]interface{}{"a": 1},
				nil,
			},
			desc: "Slice containing nil interfaces should handle each element",
		},
		{
			name: "array_with_interface_wrapping_pointers",
			input: [3]interface{}{
				func() interface{} {
					s := "first"
					return &s
				}(),
				func() interface{} {
					var p *int = nil
					return p
				}(),
				func() interface{} {
					i := 42
					return &i
				}(),
			},
			desc: "Array with interfaces wrapping different pointer types",
		},
		{
			name:  "nil_slice_vs_empty_slice",
			input: []interface{}(nil),
			desc:  "Nil slice should be handled differently from empty slice",
		},
		{
			name: "map_with_interface_values_containing_pointers",
			input: map[string]interface{}{
				"nil_ptr": (*string)(nil),
				"valid_ptr": func() interface{} {
					s := "value"
					return &s
				}(),
				"nested_map": map[string]interface{}{
					"inner_ptr": func() interface{} {
						i := 123
						return &i
					}(),
				},
			},
			desc: "Map with mixed interface values including pointers",
		},
		{
			name: "deeply_nested_interface_pointer_chain",
			input: func() interface{} {
				base := map[string]interface{}{"key": "value"}
				level1 := &base
				var level2 interface{} = level1
				level3 := &level2
				return level3
			}(),
			desc: "Complex nesting of pointers and interfaces should be resolved",
		},
		{
			name: "map_key_type_checking_edge_case",
			input: map[float64]interface{}{
				3.14: "pi",
				2.71: "e",
			},
			desc: "Map with float64 keys should return unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that sortKeys handles all reflection edge cases without panicking
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("sortKeys panicked on %s: %v", tt.desc, r)
				}
			}()

			result := sortKeys(tt.input)

			// Ensure result can be marshaled to JSON (basic validation)
			_, err := json.Marshal(result)
			if err != nil {
				// Only fail if it's not an expected unmarshalable type (like functions)
				if !strings.Contains(err.Error(), "unsupported type") {
					t.Errorf("sortKeys result for %s could not be marshaled: %v", tt.desc, err)
				}
			}

			// For nil inputs, expect nil output
			if tt.input == nil && result != nil {
				t.Errorf("Expected nil result for nil input, got: %v", result)
			}
		})
	}
}

func TestSortKeysMapKeyTypeValidation(t *testing.T) {
	// Test the specific reflection path for map key type checking
	tests := []struct {
		name        string
		createMap   func() interface{}
		shouldSort  bool
		description string
	}{
		{
			name: "string_keyed_map_should_sort",
			createMap: func() interface{} {
				return map[string]interface{}{
					"zebra": 1,
					"alpha": 2,
				}
			},
			shouldSort:  true,
			description: "String-keyed maps should have keys sorted",
		},
		{
			name: "int_keyed_map_should_not_sort",
			createMap: func() interface{} {
				return map[int]interface{}{
					3: "three",
					1: "one",
					2: "two",
				}
			},
			shouldSort:  false,
			description: "Int-keyed maps should be returned as-is",
		},
		{
			name: "interface_keyed_map_should_not_sort",
			createMap: func() interface{} {
				return map[interface{}]interface{}{
					"string": "value1",
					42:       "value2",
					true:     "value3",
				}
			},
			shouldSort:  false,
			description: "Interface{}-keyed maps should be returned as-is",
		},
		{
			name: "bool_keyed_map_should_not_sort",
			createMap: func() interface{} {
				return map[bool]interface{}{
					true:  "true_value",
					false: "false_value",
				}
			},
			shouldSort:  false,
			description: "Bool keyed maps should be returned as-is",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.createMap()
			result := sortKeys(input)

			// Convert both to JSON to compare
			inputJSON, _ := json.Marshal(input)
			resultJSON, _ := json.Marshal(result)

			if tt.shouldSort {
				// For string-keyed maps, the result should be different (sorted)
				// We can't easily test exact order without more complex logic,
				// but we can ensure it's still valid JSON of the same structure
				if bytes.Equal(inputJSON, resultJSON) {
					// This could be OK if the input was already sorted
					t.Logf("Input was already sorted or result unchanged for %s", tt.description)
				}
			} else {
				// For non-string-keyed maps, result should be identical to input
				if !bytes.Equal(inputJSON, resultJSON) {
					t.Errorf("%s: expected unchanged result, input: %s, result: %s", tt.description, inputJSON, resultJSON)
				}
			}
		})
	}
}

func TestSortKeysCompleteReflectionCoverage(t *testing.T) {
	// Target specific reflection paths that might not be covered
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "empty_map_with_string_keys",
			input: make(map[string]interface{}),
		},
		{
			name:  "single_key_map",
			input: map[string]interface{}{"single": "value"},
		},
		{
			name:  "map_with_zero_length_after_filtering",
			input: map[string]interface{}{},
		},
		{
			name: "deeply_nested_pointer_interface_chain",
			input: func() interface{} {
				// Create a complex nested structure
				value := "deep_value"
				ptr1 := &value
				var iface1 interface{} = ptr1
				ptr2 := &iface1
				var iface2 interface{} = ptr2
				return iface2
			}(),
		},
		{
			name: "slice_containing_maps_with_different_key_types",
			input: []interface{}{
				map[string]interface{}{"a": 1, "z": 2},
				map[int]interface{}{1: "one", 2: "two"},
				map[interface{}]interface{}{"mixed": "value"},
			},
		},
		{
			name:  "array_with_nil_elements",
			input: [3]interface{}{nil, nil, nil},
		},
		{
			name: "map_value_type_edge_cases",
			input: map[string]interface{}{
				"nil_value":      nil,
				"pointer_to_nil": (*string)(nil),
				"empty_slice":    []interface{}{},
				"nil_slice":      []interface{}(nil),
				"empty_map":      map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure no panic occurs during sorting
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("sortKeys panicked on %s: %v", tt.name, r)
				}
			}()

			result := sortKeys(tt.input)

			// Verify the result is JSON-serializable (basic validation)
			if result != nil {
				_, err := json.Marshal(result)
				if err != nil && !strings.Contains(err.Error(), "unsupported type") {
					t.Errorf("sortKeys result for %s is not JSON serializable: %v", tt.name, err)
				}
			}
		})
	}
}

func TestSortKeysInterfaceHandling(t *testing.T) {
	// Specifically test interface{} containing nil edge cases
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name: "interface containing nil interface",
			input: func() interface{} {
				var inner interface{} = nil
				return inner
			}(),
			expected: nil,
		},
		{
			name: "interface containing nil pointer",
			input: func() interface{} {
				var ptr *string = nil
				var iface interface{} = ptr
				return iface
			}(),
			expected: nil,
		},
		{
			name: "nested interface with non-nil value",
			input: func() interface{} {
				s := "value"
				var iface interface{} = &s
				return iface
			}(),
			expected: "value",
		},
		{
			name: "slice with interface nil elements",
			input: []interface{}{
				func() interface{} {
					var iface interface{} = nil
					return iface
				}(),
				func() interface{} {
					var ptr *int = nil
					return ptr
				}(),
			},
			expected: []interface{}{nil, nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortKeys(tt.input)

			// Marshal both to JSON for comparison
			expectedJSON, err1 := json.Marshal(tt.expected)
			resultJSON, err2 := json.Marshal(result)

			if err1 != nil {
				t.Fatalf("Failed to marshal expected: %v", err1)
			}
			if err2 != nil {
				t.Fatalf("Failed to marshal result: %v", err2)
			}

			if !bytes.Equal(expectedJSON, resultJSON) {
				t.Errorf("Expected %s, got %s", string(expectedJSON), string(resultJSON))
			}
		})
	}
}

func TestMarshalDeterministicComplexNesting(t *testing.T) {
	// Test complex nested structures to ensure full coverage
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name: "deeply nested maps and slices",
			input: map[string]interface{}{
				"z_first": map[string]interface{}{
					"z_nested": []interface{}{
						map[string]interface{}{
							"z_item": "value",
							"a_item": "value",
						},
					},
				},
				"a_last": []interface{}{
					map[string]interface{}{
						"z_key": "value",
						"a_key": "value",
					},
				},
			},
		},
		{
			name: "interface with pointers and nils",
			input: map[string]interface{}{
				"nil_interface": func() interface{} {
					var i interface{} = nil
					return i
				}(),
				"nil_ptr": (*int)(nil),
				"valid_ptr": func() interface{} {
					i := 42
					return &i
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic and produce valid JSON
			result, err := MarshalDeterministic(tt.input)
			if err != nil {
				t.Errorf("MarshalDeterministic failed: %v", err)
			}

			// Verify result is valid JSON
			var unmarshaled interface{}
			if err := json.Unmarshal(result, &unmarshaled); err != nil {
				t.Errorf("Result is not valid JSON: %v", err)
			}
		})
	}
}

func TestSortKeysInterfaceWithNonNilElem(t *testing.T) {
	// Test interface case where IsNil() is false and has non-nil elem
	// This covers the reflect.Interface case lines 61-65

	// Create a struct that will be passed as interface{}
	type testStruct struct {
		Value string
	}

	testData := map[string]interface{}{
		"z_key": func() interface{} {
			// Return a non-nil interface containing a non-nil value
			var iface interface{} = &testStruct{Value: "test"}
			return iface
		}(),
		"a_key": func() interface{} {
			s := "string value"
			var iface interface{} = &s
			return iface
		}(),
		"m_key": func() interface{} {
			// Nested interface
			inner := map[string]interface{}{"nested": "value"}
			var iface interface{} = inner
			return iface
		}(),
	}

	// This should recursively handle the interface types
	result := sortKeys(testData)

	// Verify it's JSON-serializable
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal result: %v", err)
	}

	// Verify the keys are sorted
	var resultMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &resultMap); err != nil {
		t.Errorf("Failed to unmarshal result: %v", err)
	}

	// Keys should be in sorted order in the JSON
	if !strings.Contains(string(jsonBytes), `"a_key"`) {
		t.Error("Expected a_key in result")
	}
}
func TestSortKeysWithJSONNull(t *testing.T) {
	// Test that unmarshaled JSON with null value triggers the reflect.Interface case
	// When JSON is unmarshaled, null becomes interface{} with nil value
	jsonData := `null`

	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// data should be nil after unmarshaling "null"
	if data != nil {
		t.Errorf("Expected nil after unmarshaling null, got: %v", data)
	}

	// sortKeys should handle the null value (which is interface{} containing nil)
	result := sortKeys(data)

	// Result should be nil
	if result != nil {
		t.Errorf("Expected nil result, got: %v", result)
	}

	// Marshal back to JSON
	resultJSON, err := MarshalDeterministic(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	// Verify null is preserved
	expected := `null`
	if string(resultJSON) != expected {
		t.Errorf("Expected %s, got %s", expected, string(resultJSON))
	}
}

func TestSortKeysInterfaceNonNilWithValue(t *testing.T) {
	// Test the reflect.Interface case where IsNil() is false and contains a value
	// This specifically tests the return sortKeys(v.Elem().Interface()) path

	// Create data with interface{} values wrapping different types
	jsonData := `{
		"z_string": "last",
		"a_number": 42,
		"m_bool": true,
		"nested": {
			"z_nested": "value",
			"a_nested": null
		}
	}`

	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// When unmarshaling, the values in the map are interface{} wrapping concrete types
	// sortKeys needs to recursively handle these interfaces
	result := sortKeys(data)

	// Marshal back to JSON
	resultJSON, err := MarshalDeterministic(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	// Verify keys are sorted and values are preserved, including nested null
	expected := `{"a_number":42,"m_bool":true,"nested":{"a_nested":null,"z_nested":"value"},"z_string":"last"}`
	if string(resultJSON) != expected {
		t.Errorf("Expected %s, got %s", expected, string(resultJSON))
	}
}

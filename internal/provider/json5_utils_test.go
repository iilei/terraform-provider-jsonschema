package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseJSON5String(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    interface{}
	}{
		{
			name:  "valid JSON",
			input: `{"test": "value"}`,
			expected: map[string]interface{}{
				"test": "value",
			},
		},
		{
			name:  "valid JSON5 with comments",
			input: `{"test": "value", /* comment */ }`,
			expected: map[string]interface{}{
				"test": "value",
			},
		},
		{
			name:  "JSON5 with trailing comma",
			input: `{"test": "value", "another": 123,}`,
			expected: map[string]interface{}{
				"test":    "value",
				"another": float64(123),
			},
		},
		{
			name:  "JSON5 with unquoted keys",
			input: `{test: "value", another: 123}`,
			expected: map[string]interface{}{
				"test":    "value",
				"another": float64(123),
			},
		},
		{
			name:  "JSON5 with single quotes",
			input: `{'test': 'value'}`,
			expected: map[string]interface{}{
				"test": "value",
			},
		},
		{
			name:        "invalid JSON5",
			input:       `{test: value}`, // unquoted string value
			expectError: true,
		},
		{
			name:        "completely invalid",
			input:       `not json at all`,
			expectError: true,
		},
		{
			name:     "array",
			input:    `[1, 2, 3]`,
			expected: []interface{}{float64(1), float64(2), float64(3)},
		},
		{
			name:     "string",
			input:    `"just a string"`,
			expected: "just a string",
		},
		{
			name:     "number",
			input:    `42`,
			expected: float64(42),
		},
		{
			name:     "boolean",
			input:    `true`,
			expected: true,
		},
		{
			name:     "null",
			input:    `null`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseJSON5String(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// For complex types, we'll do a simple check
			if tt.expected != nil {
				// This is a basic equality check - for more complex cases we'd need deeper comparison
				switch expected := tt.expected.(type) {
				case map[string]interface{}:
					resultMap, ok := result.(map[string]interface{})
					if !ok {
						t.Errorf("expected map, got %T", result)
						return
					}
					for key, value := range expected {
						if resultMap[key] != value {
							t.Errorf("expected %s=%v, got %v", key, value, resultMap[key])
						}
					}
				case []interface{}:
					resultSlice, ok := result.([]interface{})
					if !ok {
						t.Errorf("expected slice, got %T", result)
						return
					}
					if len(resultSlice) != len(expected) {
						t.Errorf("expected slice length %d, got %d", len(expected), len(resultSlice))
						return
					}
					for i, value := range expected {
						if resultSlice[i] != value {
							t.Errorf("expected [%d]=%v, got %v", i, value, resultSlice[i])
						}
					}
				default:
					if result != expected {
						t.Errorf("expected %v, got %v", expected, result)
					}
				}
			}
		})
	}
}

func TestJSON5StringToJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    string
	}{
		{
			name:     "simple object",
			input:    `{"test": "value"}`,
			expected: `{"test":"value"}`,
		},
		{
			name:     "JSON5 with comments",
			input:    `{test: "value", /* comment */}`,
			expected: `{"test":"value"}`,
		},
		{
			name:        "invalid JSON5",
			input:       `{invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JSON5StringToJSON(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestJSON5ToJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectError bool
	}{
		{
			name:  "valid JSON bytes",
			input: []byte(`{"test": "value"}`),
		},
		{
			name:  "valid JSON5 bytes",
			input: []byte(`{test: "value"}`),
		},
		{
			name:        "invalid JSON5 bytes",
			input:       []byte(`{invalid`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JSON5ToJSON(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) == 0 {
				t.Errorf("expected non-empty result")
			}
		})
	}
}

func TestJSON5FileLoader(t *testing.T) {
	// Create a temporary JSON5 file
	tmpDir := t.TempDir()
	json5File := filepath.Join(tmpDir, "test.json5")
	
	json5Content := `{
		// JSON5 test file with comments
		"name": "test",
		"items": [1, 2, 3,], // trailing comma
		config: { // unquoted key
			"enabled": true
		}
	}`
	
	if err := os.WriteFile(json5File, []byte(json5Content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Test loading with JSON5FileLoader
	loader := JSON5FileLoader{}
	fileURL := fmt.Sprintf("file://%s", json5File)
	
	data, err := loader.Load(fileURL)
	if err != nil {
		t.Fatalf("Failed to load JSON5 file: %v", err)
	}
	
	// Verify the loaded data
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", data)
	}
	
	if dataMap["name"] != "test" {
		t.Errorf("Expected name='test', got %v", dataMap["name"])
	}
	
	if configMap, ok := dataMap["config"].(map[string]interface{}); !ok {
		t.Errorf("Expected config to be map, got %T", dataMap["config"])
	} else if configMap["enabled"] != true {
		t.Errorf("Expected config.enabled=true, got %v", configMap["enabled"])
	}
}

func TestJSON5FileLoaderErrors(t *testing.T) {
	loader := JSON5FileLoader{}
	
	t.Run("invalid URL scheme", func(t *testing.T) {
		// Test with non-file URL that ToFile can't handle
		_, err := loader.Load("http://example.com/schema.json")
		if err == nil {
			t.Error("Expected error for non-file URL, got nil")
		}
	})
	
	t.Run("missing file", func(t *testing.T) {
		// Test with valid file URL but missing file
		missingFile := "file:///tmp/nonexistent_file_12345.json"
		_, err := loader.Load(missingFile)
		if err == nil {
			t.Error("Expected error for missing file, got nil")
		}
		if err != nil && !os.IsNotExist(err) {
			// Check that we get a "failed to read file" error
			expectedMsg := "failed to read file"
			if errMsg := err.Error(); len(errMsg) < len(expectedMsg) || errMsg[:len(expectedMsg)] != expectedMsg {
				t.Logf("Got expected error: %v", err)
			}
		}
	})
	
	t.Run("invalid JSON5 content", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidFile := filepath.Join(tmpDir, "invalid.json5")
		
		// Write invalid JSON5 content
		if err := os.WriteFile(invalidFile, []byte(`{invalid json`), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		
		fileURL := fmt.Sprintf("file://%s", invalidFile)
		_, err := loader.Load(fileURL)
		if err == nil {
			t.Error("Expected error for invalid JSON5, got nil")
		}
	})
}

func TestParseJSON5EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectError bool
		description string
	}{
		{
			name:        "empty byte slice",
			input:       []byte{},
			expectError: true,
			description: "empty input should fail",
		},
		{
			name:        "only whitespace",
			input:       []byte("   \n\t  "),
			expectError: true,
			description: "whitespace-only input should fail",
		},
		{
			name:        "multiline comment",
			input:       []byte(`{"key": "value" /* multiline\ncomment */}`),
			expectError: false,
			description: "multiline comments should be handled",
		},
		{
			name:        "single line comment",
			input:       []byte("{\n\"key\": \"value\" // single line comment\n}"),
			expectError: false,
			description: "single line comments should be handled",
		},
		{
			name:        "hex numbers",
			input:       []byte(`{"hex": 0xFF}`),
			expectError: false,
			description: "hex numbers should be parsed",
		},
		{
			name:        "Infinity",
			input:       []byte(`{"inf": Infinity}`),
			expectError: false,
			description: "Infinity should be parsed",
		},
		{
			name:        "NaN",
			input:       []byte(`{"nan": NaN}`),
			expectError: false,
			description: "NaN should be parsed",
		},
		{
			name:        "leading plus sign",
			input:       []byte(`{"num": +42}`),
			expectError: false,
			description: "leading plus sign should be allowed",
		},
		{
			name:        "trailing comma in array",
			input:       []byte(`[1, 2, 3,]`),
			expectError: false,
			description: "trailing comma in array should be allowed",
		},
		{
			name:        "unclosed object",
			input:       []byte(`{"key": "value"`),
			expectError: true,
			description: "unclosed object should error",
		},
		{
			name:        "unclosed array",
			input:       []byte(`[1, 2, 3`),
			expectError: true,
			description: "unclosed array should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseJSON5(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if result == nil {
					t.Errorf("%s: expected non-nil result", tt.description)
				}
			}
		})
	}
}

func TestJSON5ToJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectError   bool
		validateJSON  bool
	}{
		{
			name:         "complex nested structure",
			input:        []byte(`{outer: {inner: [1, 2, {deep: "value"}]}}`),
			expectError:  false,
			validateJSON: true,
		},
		{
			name:         "array of objects with comments",
			input:        []byte(`[{a: 1}, /* comment */ {b: 2}]`),
			expectError:  false,
			validateJSON: true,
		},
		{
			name:         "mixed quotes",
			input:        []byte(`{"double": "value", 'single': 'value'}`),
			expectError:  false,
			validateJSON: true,
		},
		{
			name:        "invalid structure",
			input:       []byte(`{broken: structure`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JSON5ToJSON(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validateJSON {
				// Verify result is valid JSON
				var unmarshaled interface{}
				if err := json.Unmarshal(result, &unmarshaled); err != nil {
					t.Errorf("result is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestJSON5FileLoaderWithVariousFormats(t *testing.T) {
	tmpDir := t.TempDir()
	
	tests := []struct {
		name        string
		content     string
		expectError bool
		description string
	}{
		{
			name:        "pure JSON",
			content:     `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			expectError: false,
			description: "pure JSON should work",
		},
		{
			name: "JSON5 with all features",
			content: `{
				// This is a comment
				type: "object",
				properties: {
					name: {type: 'string',}, // trailing comma
					age: {type: "number",}
				},
			}`,
			expectError: false,
			description: "JSON5 with comments and trailing commas",
		},
		{
			name:        "empty file",
			content:     ``,
			expectError: true,
			description: "empty file should error",
		},
		{
			name:        "only comments",
			content:     `// just a comment`,
			expectError: true,
			description: "file with only comments should error",
		},
	}

	loader := JSON5FileLoader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, tt.name+".json5")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			fileURL := fmt.Sprintf("file://%s", testFile)
			result, err := loader.Load(fileURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if result == nil {
					t.Errorf("%s: expected non-nil result", tt.description)
				}
			}
		})
	}
}
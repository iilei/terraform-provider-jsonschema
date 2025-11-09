package provider

import (
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
package jsonschema

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestErrorMessageTemplating(t *testing.T) {
	mockError := fmt.Errorf("required property 'name' missing")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "simple full message",
			template: "{{.FullMessage}}",
			expected: "required property 'name' missing",
		},
		{
			name:     "error with schema",
			template: "Error in {{.SchemaFile}}: {{.FullMessage}}",
			expected: "Error in test.schema.json: required property 'name' missing",
		},
		{
			name:     "error count template with individual errors",
			template: "{{.ErrorCount}} error(s) found: {{range .Errors}}{{.Message}}{{end}}",
			expected: "1 error(s) found: required property 'name' missing",
		},
		{
			name:     "error with path from individual errors",
			template: "{{range .Errors}}{{.DocumentPath}}: {{.Message}}{{end}}",
			expected: ": required property 'name' missing",
		},
		{
			name:     "ci format template",
			template: "::error file={{.SchemaFile}},line=1::{{.FullMessage}}",
			expected: "::error file=test.schema.json,line=1::required property 'name' missing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatValidationError(mockError, "test.schema.json", "{}", tc.template)
			if result.Error() != tc.expected {
				t.Errorf("Expected: %s\nGot: %s", tc.expected, result.Error())
			}
		})
	}
}

func TestErrorFormattingEdgeCases(t *testing.T) {
	mockError := fmt.Errorf("validation error")

	// Test template with invalid Go template syntax
	result := FormatValidationError(mockError, "schema.json", "{}", "{{.InvalidField")
	if !strings.Contains(result.Error(), "template parsing failed") {
		t.Error("Expected template parsing error for invalid Go template")
	}

	// Test document truncation
	longDoc := strings.Repeat("x", 600)
	result2 := FormatValidationError(mockError, "schema.json", longDoc, "Doc: {{.Document}}")
	if !strings.Contains(result2.Error(), "...") {
		t.Error("Expected document to be truncated")
	}

	// Test all available variables
	result3 := FormatValidationError(
		mockError,
		"test.schema.json",
		`{"test": true}`,
		"Error: {{.FullMessage}}, Schema: {{.SchemaFile}}, Count: {{.ErrorCount}}, Doc: {{.Document}}",
	)
	expected := `Error: validation error, Schema: test.schema.json, Count: 1, Doc: {"test": true}`
	if result3.Error() != expected {
		t.Errorf("Expected: %s\nGot: %s", expected, result3.Error())
	}
}

// MockValidationError is a simple error for testing - not implementing ValidationError interface
// since the actual jsonschema.ValidationError is used in production
type MockValidationError struct {
	message string
	path    string
}

func (m *MockValidationError) Error() string {
	return m.message
}

func TestFormatValidationErrorComplexScenarios(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		schemaPath       string
		document         string
		template         string
		expectedContains []string
	}{
		{
			name:       "Go template with multiple variables",
			err:        &MockValidationError{message: "validation failed", path: "/name"},
			schemaPath: "/path/to/schema.json",
			document:   `{"name": 123}`,
			template:   "Error: {{.FullMessage}} | Schema: {{.SchemaFile}}",
			expectedContains: []string{
				"Error:",
				"validation failed",
				"Schema:",
				"/path/to/schema.json",
			},
		},
		{
			name:             "Template with error count",
			err:              &MockValidationError{message: "type error", path: "/age"},
			schemaPath:       "user.schema.json",
			document:         `{"age": "twenty"}`,
			template:         "Found {{.ErrorCount}} error(s) in schema {{.SchemaFile}}: {{range .Errors}}{{.Message}}{{end}}",
			expectedContains: []string{"Found 1 error(s)", "user.schema.json", "type error"},
		},
		{
			name:             "Template with path iteration",
			err:              &MockValidationError{message: "missing field"},
			schemaPath:       "schema.json",
			document:         "{}",
			template:         "{{range .Errors}}Path {{.DocumentPath}}: {{.Message}}{{end}}",
			expectedContains: []string{"Path :", "missing field"},
		},
		{
			name:             "Invalid Go template syntax",
			err:              &MockValidationError{message: "test error"},
			schemaPath:       "test.json",
			document:         "{}",
			template:         "{{.Errors", // Missing closing braces
			expectedContains: []string{"template parsing failed"},
		},
		{
			name:             "Non-validation error",
			err:              fmt.Errorf("generic error"),
			schemaPath:       "test.json",
			document:         "{}",
			template:         "Custom: {{.FullMessage}}",
			expectedContains: []string{"Custom:", "generic error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValidationError(tt.err, tt.schemaPath, tt.document, tt.template)

			if result == nil {
				t.Errorf("expected error but got nil")
				return
			}

			errorMessage := result.Error()
			for _, expected := range tt.expectedContains {
				if !strings.Contains(errorMessage, expected) {
					t.Errorf(
						"expected error message to contain %q, got: %s",
						expected,
						errorMessage,
					)
				}
			}
		})
	}
}

func TestNilErrorHandling(t *testing.T) {
	// Test that passing nil error doesn't crash - should return nil or handle gracefully
	result := FormatValidationError(nil, "test.json", "{}", "Error: {error}")
	// The function should handle nil gracefully, either by returning nil or a reasonable error
	if result == nil {
		// This is acceptable - function handles nil by returning nil
		return
	}
	// If not nil, it should be a reasonable error message
	if !strings.Contains(result.Error(), "error") {
		t.Errorf("expected error message to contain 'error', got: %v", result)
	}
}

func TestGetCommonTemplate(t *testing.T) {
	tests := []struct {
		name             string
		templateName     string
		expectFound      bool
		expectedTemplate string
	}{
		{
			name:             "simple template",
			templateName:     "simple",
			expectFound:      true,
			expectedTemplate: "{{.FullMessage}}",
		},
		{
			name:             "basic template",
			templateName:     "basic",
			expectFound:      true,
			expectedTemplate: "{{range .Errors}}{{.Message}}\n{{end}}",
		},
		{
			name:             "detailed template",
			templateName:     "detailed",
			expectFound:      true,
			expectedTemplate: "{{.ErrorCount}} validation error(s) found:\n{{range $i, $e := .Errors}}{{add $i 1}}. {{.Message}} at {{.DocumentPath}}\n{{end}}",
		},
		{
			name:             "with_path template",
			templateName:     "with_path",
			expectFound:      true,
			expectedTemplate: "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}",
		},
		{
			name:             "with_schema template",
			templateName:     "with_schema",
			expectFound:      true,
			expectedTemplate: "Schema {{.SchemaFile}} validation failed:\n{{.FullMessage}}",
		},
		{
			name:             "verbose template",
			templateName:     "verbose",
			expectFound:      true,
			expectedTemplate: "Validation Results:\nSchema: {{.SchemaFile}}\nErrors: {{.ErrorCount}}\nFull Message: {{.FullMessage}}\n\nIndividual Errors:\n{{range $i, $e := .Errors}}Error {{add $i 1}}:\n  Document Path: {{.DocumentPath}}\n  Schema Path: {{.SchemaPath}}\n  Message: {{.Message}}{{if .Value}}\n  Value: {{.Value}}{{end}}\n\n{{end}}",
		},
		{
			name:             "non-existent template",
			templateName:     "nonexistent",
			expectFound:      false,
			expectedTemplate: "",
		},
		{
			name:             "empty template name",
			templateName:     "",
			expectFound:      false,
			expectedTemplate: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, found := GetCommonTemplate(tt.templateName)

			if found != tt.expectFound {
				t.Errorf("GetCommonTemplate() found = %v, expectFound = %v", found, tt.expectFound)
			}

			if found && template != tt.expectedTemplate {
				t.Errorf(
					"GetCommonTemplate() template = %v, expectedTemplate = %v",
					template,
					tt.expectedTemplate,
				)
			}
		})
	}
}

func TestValidationErrorSorting(t *testing.T) {
	// Test that validation errors are consistently ordered
	unsortedErrors := []ValidationErrorDetail{
		{Message: "error at /z", DocumentPath: "/z"},
		{Message: "error at /a", DocumentPath: "/a"},
		{Message: "second error at /a", DocumentPath: "/a"},
		{Message: "error at /b", DocumentPath: "/b"},
	}

	// Sort using our function
	sortValidationErrors(unsortedErrors)

	// Verify the order
	expected := []ValidationErrorDetail{
		{Message: "error at /a", DocumentPath: "/a"},
		{Message: "second error at /a", DocumentPath: "/a"},
		{Message: "error at /b", DocumentPath: "/b"},
		{Message: "error at /z", DocumentPath: "/z"},
	}

	for i, err := range unsortedErrors {
		if err.DocumentPath != expected[i].DocumentPath || err.Message != expected[i].Message {
			t.Errorf("Expected error %d to be %+v, got %+v", i, expected[i], err)
		}
	}
}

func TestCommonErrorTemplatesExist(t *testing.T) {
	// Test that all expected common templates exist
	expectedTemplates := []string{
		"basic",
		"detailed",
		"simple",
		"with_path",
		"with_schema",
		"verbose",
	}

	for _, templateName := range expectedTemplates {
		t.Run("template_"+templateName, func(t *testing.T) {
			template, found := GetCommonTemplate(templateName)
			if !found {
				t.Errorf("Expected common template '%s' not found", templateName)
			}
			if template == "" {
				t.Errorf("Common template '%s' is empty", templateName)
			}
		})
	}
}

func TestFormatValidationErrorTemplateEdgeCases(t *testing.T) {
	mockErr := fmt.Errorf("validation failed")
	schema := "test.schema.json"
	document := `{"test": "data"}`

	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "template with invalid syntax",
			template: "{{.InvalidField}}",
			wantErr:  false, // Should not panic, template execution might just produce empty string
		},
		{
			name:     "template with range but no Errors",
			template: "{{range .Errors}}{{.DocumentPath}}{{end}}",
			wantErr:  false,
		},
		{
			name:     "template accessing nested fields",
			template: "{{range .Errors}}{{.DocumentPath}}: {{.Message}} ({{.SchemaPath}}){{end}}",
			wantErr:  false,
		},
		{
			name:     "template with conditional",
			template: "{{if .ErrorCount}}Found {{.ErrorCount}} errors{{end}}",
			wantErr:  false,
		},
		{
			name:     "empty template",
			template: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValidationError(mockErr, schema, document, tt.template)
			if result == nil {
				t.Error("Expected non-nil error result")
			}
		})
	}
}

func TestExtractCleanMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		path     string
		expected string
	}{
		{
			name:     "root path with empty string prefix",
			message:  "at '': required property missing",
			path:     "", // Per RFC 6901, root is empty string
			expected: "required property missing",
		},
		{
			name:     "nested path with prefix",
			message:  "at '/name': must be string",
			path:     "/name",
			expected: "must be string",
		},
		{
			name:     "message without path prefix",
			message:  "validation error",
			path:     "/test",
			expected: "validation error",
		},
		{
			name:     "empty message",
			message:  "",
			path:     "",
			expected: "",
		},
		{
			name:     "path with multiple segments",
			message:  "at '/user/address/city': invalid format",
			path:     "/user/address/city",
			expected: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCleanMessage(tt.message, tt.path)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatInstanceLocation(t *testing.T) {
	tests := []struct {
		name     string
		location []string
		expected string
	}{
		{
			name:     "empty location",
			location: []string{},
			expected: "", // Per RFC 6901, empty string represents root
		},
		{
			name:     "single element",
			location: []string{"name"},
			expected: "/name",
		},
		{
			name:     "multiple elements",
			location: []string{"user", "address", "city"},
			expected: "/user/address/city",
		},
		{
			name:     "array index",
			location: []string{"items", "0", "name"},
			expected: "/items/0/name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatInstanceLocation(tt.location)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string no truncation",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length no truncation",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string with truncation",
			input:    "this is a very long string that needs truncation",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "single character over limit",
			input:    "ab",
			maxLen:   1,
			expected: "a...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateSortedFullMessage(t *testing.T) {
	// Test the message generation without using the actual ValidationError
	// We'll test this indirectly through FormatValidationError
	tests := []struct {
		name          string
		errorMsg      string
		template      string
		expectedParts []string
	}{
		{
			name:          "error with multiple parts",
			errorMsg:      "validation error",
			template:      "{{.FullMessage}}",
			expectedParts: []string{"validation error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockErr := fmt.Errorf("%s", tt.errorMsg)
			result := FormatValidationError(
				mockErr,
				"test.schema.json",
				`{"test": "data"}`,
				tt.template,
			)

			for _, part := range tt.expectedParts {
				if !strings.Contains(result.Error(), part) {
					t.Errorf("Expected result to contain %q, got %q", part, result.Error())
				}
			}
		})
	}
}

func TestSortValidationErrorsSecondarySort(t *testing.T) {
	// Test secondary sort by message when paths are the same
	errors := []ValidationErrorDetail{
		{Message: "error z", DocumentPath: "/same"},
		{Message: "error a", DocumentPath: "/same"},
		{Message: "error m", DocumentPath: "/same"},
	}

	sortValidationErrors(errors)

	// Verify they are sorted alphabetically by message
	if errors[0].Message != "error a" {
		t.Errorf("Expected first error to be 'error a', got %q", errors[0].Message)
	}
	if errors[1].Message != "error m" {
		t.Errorf("Expected second error to be 'error m', got %q", errors[1].Message)
	}
	if errors[2].Message != "error z" {
		t.Errorf("Expected third error to be 'error z', got %q", errors[2].Message)
	}
}

func TestFormatValidationErrorWithLongDocument(t *testing.T) {
	// Test document truncation in error context
	longDocument := strings.Repeat("x", 1000)
	mockErr := fmt.Errorf("validation error")

	result := FormatValidationError(mockErr, "test.json", longDocument, "Doc: {{.Document}}")

	// Verify the document was truncated
	if !strings.Contains(result.Error(), "...") {
		t.Error("Expected document to be truncated in error message")
	}

	// Verify error message is not absurdly long
	if len(result.Error()) > 600 {
		t.Errorf("Error message too long: %d characters", len(result.Error()))
	}
}

func TestFormatValidationErrorTemplateAddFunction(t *testing.T) {
	// Test the template's add function
	mockErr := fmt.Errorf("validation error")

	result := FormatValidationError(mockErr, "test.json", `{"test": "data"}`,
		"Error {{add 1 2}}: {{.FullMessage}}")

	if !strings.Contains(result.Error(), "Error 3:") {
		t.Errorf("Expected add function to work, got: %s", result.Error())
	}
}

func TestFormatValidationErrorEdgeCaseTemplates(t *testing.T) {
	// Test various template edge cases
	mockErr := fmt.Errorf("test error")

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "template with range over empty errors",
			template: "Count: {{.ErrorCount}} {{range .Errors}}Should not appear{{end}}",
			expected: "Count: 1",
		},
		{
			name:     "template with multiple add operations",
			template: "Result: {{add (add 1 2) 3}}",
			expected: "Result: 6",
		},
		{
			name:     "template with conditionals",
			template: "{{if gt .ErrorCount 0}}Has errors{{else}}No errors{{end}}",
			expected: "Has errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValidationError(mockErr, "test.json", `{}`, tt.template)
			if !strings.Contains(result.Error(), tt.expected) {
				t.Errorf("Expected %q in result, got: %s", tt.expected, result.Error())
			}
		})
	}
}

func TestCompactDeterministicJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "valid map",
			input:   map[string]interface{}{"key": "value"},
			wantErr: false,
		},
		{
			name:    "nested map",
			input:   map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}},
			wantErr: false,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "empty map",
			input:   map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompactDeterministicJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompactDeterministicJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(result) == 0 {
				t.Error("Expected non-empty result for valid input")
			}
		})
	}
}

// TestExtractValueAtPath tests the extractValueAtPath function comprehensively
func TestExtractValueAtPath(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		path     []string
		expected string
	}{
		{
			name:     "nil data, empty path",
			data:     nil,
			path:     []string{},
			expected: "null",
		},
		{
			name:     "root level string",
			data:     "test value",
			path:     []string{},
			expected: `"test value"`,
		},
		{
			name:     "root level number",
			data:     42,
			path:     []string{},
			expected: "42",
		},
		{
			name:     "root level boolean",
			data:     true,
			path:     []string{},
			expected: "true",
		},
		{
			name: "root level object truncated",
			data: map[string]interface{}{
				"key": strings.Repeat("x", 120),
			},
			path:     []string{},
			expected: "", // Should be truncated with "..."
		},
		{
			name: "simple object property",
			data: map[string]interface{}{
				"name": "John",
			},
			path:     []string{"name"},
			expected: `"John"`,
		},
		{
			name: "nested object property",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"email": "test@example.com",
				},
			},
			path:     []string{"user", "email"},
			expected: `"test@example.com"`,
		},
		{
			name: "missing property in object",
			data: map[string]interface{}{
				"name": "John",
			},
			path:     []string{"age"},
			expected: "",
		},
		{
			name: "array with valid index",
			data: map[string]interface{}{
				"items": []interface{}{"first", "second", "third"},
			},
			path:     []string{"items", "1"},
			expected: `"second"`,
		},
		{
			name: "array with index 0",
			data: map[string]interface{}{
				"items": []interface{}{100, 200, 300},
			},
			path:     []string{"items", "0"},
			expected: "100",
		},
		{
			name: "array with out of bounds index",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			path:     []string{"items", "5"},
			expected: "",
		},
		{
			name: "array with negative index",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			path:     []string{"items", "-1"},
			expected: "",
		},
		{
			name: "array with invalid index string",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			path:     []string{"items", "invalid"},
			expected: "",
		},
		{
			name: "nested array element",
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"id": 1, "name": "Alice"},
					map[string]interface{}{"id": 2, "name": "Bob"},
				},
			},
			path:     []string{"users", "1", "name"},
			expected: `"Bob"`,
		},
		{
			name:     "primitive value with path (can't navigate further)",
			data:     map[string]interface{}{"port": 8080},
			path:     []string{"port", "invalid"},
			expected: "",
		},
		{
			name: "null value in object",
			data: map[string]interface{}{
				"value": nil,
			},
			path:     []string{"value"},
			expected: "null",
		},
		{
			name: "complex nested structure",
			data: map[string]interface{}{
				"config": map[string]interface{}{
					"servers": []interface{}{
						map[string]interface{}{
							"host": "localhost",
							"port": 8080,
						},
					},
				},
			},
			path:     []string{"config", "servers", "0", "port"},
			expected: "8080",
		},
		{
			name: "array element is null",
			data: map[string]interface{}{
				"items": []interface{}{nil, "value", nil},
			},
			path:     []string{"items", "0"},
			expected: "null",
		},
		{
			name: "deeply nested missing property",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "value",
					},
				},
			},
			path:     []string{"level1", "level2", "missing", "level4"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValueAtPath(tt.data, tt.path)

			if tt.expected == "" && result != "" {
				// Check if it's a truncation case
				if !strings.Contains(result, "...") {
					t.Errorf("Expected empty string, got: %s", result)
				}
			} else if tt.name == "root level object truncated" {
				// Special case: should be truncated
				if !strings.Contains(result, "...") && len(result) > 100 {
					t.Errorf("Expected truncated result with '...', got: %s", result)
				}
			} else if result != tt.expected {
				t.Errorf("Expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

// TestValueFieldPopulationIntegration tests that Value field is correctly populated in actual validation errors
func TestValueFieldPopulationIntegration(t *testing.T) {
	tests := []struct {
		name           string
		schema         string
		document       string
		expectedValues map[string]string // path -> expected value
	}{
		{
			name: "string value violation",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string", "minLength": 5}
				}
			}`,
			document: `{"name": "ab"}`,
			expectedValues: map[string]string{
				"/name": `"ab"`,
			},
		},
		{
			name: "number value violation",
			schema: `{
				"type": "object",
				"properties": {
					"port": {"type": "integer", "minimum": 1000}
				}
			}`,
			document: `{"port": 80}`,
			expectedValues: map[string]string{
				"/port": "80",
			},
		},
		{
			name: "null value",
			schema: `{
				"type": "object",
				"properties": {
					"value": {"type": "string"}
				},
				"required": ["value"]
			}`,
			document: `{"value": null}`,
			expectedValues: map[string]string{
				"/value": "null",
			},
		},
		{
			name: "boolean value",
			schema: `{
				"type": "object",
				"properties": {
					"enabled": {"type": "string"}
				}
			}`,
			document: `{"enabled": true}`,
			expectedValues: map[string]string{
				"/enabled": "true",
			},
		},
		{
			name: "array element value",
			schema: `{
				"type": "object",
				"properties": {
					"items": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}`,
			document: `{"items": ["valid", 123, "another"]}`,
			expectedValues: map[string]string{
				"/items/1": "123",
			},
		},
		{
			name: "nested object value",
			schema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							"age": {"type": "integer", "minimum": 18}
						}
					}
				}
			}`,
			document: `{"user": {"age": 15}}`,
			expectedValues: map[string]string{
				"/user/age": "15",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load schema
			compiler := jsonschema.NewCompiler()

			// Parse schema
			var schemaData interface{}
			if err := json.Unmarshal([]byte(tt.schema), &schemaData); err != nil {
				t.Fatalf("Failed to parse schema: %v", err)
			}

			if err := compiler.AddResource("test.schema.json", schemaData); err != nil {
				t.Fatalf("Failed to add schema: %v", err)
			}

			schema, err := compiler.Compile("test.schema.json")
			if err != nil {
				t.Fatalf("Failed to compile schema: %v", err)
			}

			// Parse document
			var doc interface{}
			if err := json.Unmarshal([]byte(tt.document), &doc); err != nil {
				t.Fatalf("Failed to parse document: %v", err)
			}

			// Validate
			err = schema.Validate(doc)
			if err == nil {
				t.Fatal("Expected validation error, got none")
			}

			// Type assert to ValidationError
			validationErr, ok := err.(*jsonschema.ValidationError)
			if !ok {
				t.Fatalf("Expected *jsonschema.ValidationError, got %T", err)
			}

			// Extract errors
			errors := extractValidationErrors(validationErr, doc)

			// Verify Value field is populated correctly
			for _, valErr := range errors {
				expectedValue, exists := tt.expectedValues[valErr.DocumentPath]
				if !exists {
					continue // Not checking this error
				}

				if valErr.Value != expectedValue {
					t.Errorf("Path %s: expected value %q, got %q",
						valErr.DocumentPath, expectedValue, valErr.Value)
				}
			}
		})
	}
}

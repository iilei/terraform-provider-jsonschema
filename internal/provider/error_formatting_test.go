package provider

import (
	"fmt"
	"strings"
	"testing"
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
			template: "Error in {{.Schema}}: {{.FullMessage}}",
			expected: "Error in test.schema.json: required property 'name' missing",
		},
		{
			name:     "error count template with individual errors",
			template: "{{.ErrorCount}} error(s) found: {{range .Errors}}{{.Message}}{{end}}",
			expected: "1 error(s) found: required property 'name' missing",
		},
		{
			name:     "error with path from individual errors",
			template: "{{range .Errors}}{{.Path}}: {{.Message}}{{end}}",
			expected: ": required property 'name' missing",
		},
		{
			name:     "ci format template",
			template: "::error file={{.Schema}},line=1::{{.FullMessage}}",
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

func TestProviderConfigDefaults(t *testing.T) {
	// Test that NewProviderConfig sets sensible defaults
	config, err := NewProviderConfig("", "", false)
	if err != nil {
		t.Fatalf("Failed to create provider config: %v", err)
	}

	expectedErrorTemplate := "{{.FullMessage}}"
	if config.DefaultErrorTemplate != expectedErrorTemplate {
		t.Errorf("Expected default error template: %s\nGot: %s", expectedErrorTemplate, config.DefaultErrorTemplate)
	}

	// Test that custom template is preserved
	customTemplate := "Custom error: {{.FullMessage}}"
	config2, err := NewProviderConfig("", customTemplate, false)
	if err != nil {
		t.Fatalf("Failed to create provider config with custom template: %v", err)
	}

	if config2.DefaultErrorTemplate != customTemplate {
		t.Errorf("Expected custom error template: %s\nGot: %s", customTemplate, config2.DefaultErrorTemplate)
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
	result3 := FormatValidationError(mockError, "test.schema.json", `{"test": true}`, 
		"Error: {{.FullMessage}}, Schema: {{.Schema}}, Count: {{.ErrorCount}}, Doc: {{.Document}}")
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
			template:   "Error: {{.FullMessage}} | Schema: {{.Schema}}",
			expectedContains: []string{"Error:", "validation failed", "Schema:", "/path/to/schema.json"},
		},
		{
			name:       "Template with error count",
			err:        &MockValidationError{message: "type error", path: "/age"},
			schemaPath: "user.schema.json",
			document:   `{"age": "twenty"}`,
			template:   "Found {{.ErrorCount}} error(s) in schema {{.Schema}}: {{range .Errors}}{{.Message}}{{end}}",
			expectedContains: []string{"Found 1 error(s)", "user.schema.json", "type error"},
		},
		{
			name:       "Template with path iteration",
			err:        &MockValidationError{message: "missing field"},
			schemaPath: "schema.json",
			document:   "{}",
			template:   "{{range .Errors}}Path {{.Path}}: {{.Message}}{{end}}",
			expectedContains: []string{"Path :", "missing field"},
		},
		{
			name:       "Invalid Go template syntax",
			err:        &MockValidationError{message: "test error"},
			schemaPath: "test.json",
			document:   "{}",
			template:   "{{.Errors", // Missing closing braces
			expectedContains: []string{"template parsing failed"},
		},
		{
			name:       "Non-validation error",
			err:        fmt.Errorf("generic error"),
			schemaPath: "test.json",
			document:   "{}",
			template:   "Custom: {{.FullMessage}}",
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
					t.Errorf("expected error message to contain %q, got: %s", expected, errorMessage)
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
		name       string
		templateName string
		expectFound bool
		expectedTemplate string
	}{
		{
			name:         "simple template",
			templateName: "simple",
			expectFound:  true,
			expectedTemplate: "{{.FullMessage}}",
		},
		{
			name:         "basic template",
			templateName: "basic",
			expectFound:  true,
			expectedTemplate: "{{range .Errors}}{{.Message}}\n{{end}}",
		},
		{
			name:         "detailed template",
			templateName: "detailed",
			expectFound:  true,
			expectedTemplate: "{{.ErrorCount}} validation error(s) found:\n{{range $i, $e := .Errors}}{{add $i 1}}. {{.Message}} at {{.Path}}\n{{end}}",
		},
		{
			name:         "with_path template",
			templateName: "with_path",
			expectFound:  true,
			expectedTemplate: "{{range .Errors}}{{.Path}}: {{.Message}}\n{{end}}",
		},
		{
			name:         "with_schema template",
			templateName: "with_schema",
			expectFound:  true,
			expectedTemplate: "Schema {{.Schema}} validation failed:\n{{.FullMessage}}",
		},
		{
			name:         "verbose template",
			templateName: "verbose",
			expectFound:  true,
			expectedTemplate: "Validation Results:\nSchema: {{.Schema}}\nErrors: {{.ErrorCount}}\nFull Message: {{.FullMessage}}\n\nIndividual Errors:\n{{range $i, $e := .Errors}}Error {{add $i 1}}:\n  Path: {{.Path}}\n  Schema Path: {{.SchemaPath}}\n  Message: {{.Message}}{{if .Value}}\n  Value: {{.Value}}{{end}}\n\n{{end}}",
		},
		{
			name:         "non-existent template",
			templateName: "nonexistent",
			expectFound:  false,
			expectedTemplate: "",
		},
		{
			name:         "empty template name",
			templateName: "",
			expectFound:  false,
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
				t.Errorf("GetCommonTemplate() template = %v, expectedTemplate = %v", template, tt.expectedTemplate)
			}
		})
	}
}

func TestValidationErrorSorting(t *testing.T) {
	// Test that validation errors are consistently ordered
	unsortedErrors := []ValidationErrorDetail{
		{Message: "error at /z", Path: "/z"},
		{Message: "error at /a", Path: "/a"},  
		{Message: "second error at /a", Path: "/a"},
		{Message: "error at /b", Path: "/b"},
	}
	
	// Sort using our function
	sortValidationErrors(unsortedErrors)
	
	// Verify the order
	expected := []ValidationErrorDetail{
		{Message: "error at /a", Path: "/a"},
		{Message: "second error at /a", Path: "/a"},
		{Message: "error at /b", Path: "/b"},
		{Message: "error at /z", Path: "/z"},
	}
	
	for i, err := range unsortedErrors {
		if err.Path != expected[i].Path || err.Message != expected[i].Message {
			t.Errorf("Expected error %d to be %+v, got %+v", i, expected[i], err)
		}
	}
}

func TestCommonErrorTemplatesExist(t *testing.T) {
	// Test that all expected common templates exist
	expectedTemplates := []string{"basic", "detailed", "simple", "with_path", "with_schema", "verbose"}
	
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
			template: "{{range .Errors}}{{.Path}}{{end}}",
			wantErr:  false,
		},
		{
			name:     "template accessing nested fields",
			template: "{{range .Errors}}{{.Path}}: {{.Message}} ({{.SchemaPath}}){{end}}",
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
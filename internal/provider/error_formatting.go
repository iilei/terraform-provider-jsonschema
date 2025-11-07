package provider

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidationErrorDetail represents a single validation error with rich context
type ValidationErrorDetail struct {
	Message    string `json:"message"`    // Human-readable error message
	Path       string `json:"path"`       // JSON path where error occurred  
	SchemaPath string `json:"schemaPath"` // Path in the schema where validation failed
	Value      string `json:"value"`      // The actual value that failed validation (if available)
}

// ErrorContext holds data available for error message templating
type ErrorContext struct {
	Schema       string                  `json:"schema"`       // Path to the schema file
	Document     string                  `json:"document"`     // The document content (truncated if too long)
	Errors       []ValidationErrorDetail `json:"errors"`       // Individual validation errors with details
	ErrorCount   int                     `json:"errorCount"`   // Number of individual errors
	FullMessage  string                  `json:"fullMessage"`  // Complete formatted error message from jsonschema
}

// FormatValidationError creates a formatted error message using the provided template
func FormatValidationError(err error, schemaPath, document, errorTemplate string) error {
	if err == nil {
		return nil
	}

	var errors []ValidationErrorDetail
	var fullMessage string
	
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		errors = extractValidationErrors(validationErr)
		fullMessage = validationErr.Error()
	} else {
		// For non-validation errors, create a single error detail
		errors = []ValidationErrorDetail{{
			Message: err.Error(),
			Path:    "",
		}}
		fullMessage = err.Error()
	}

	// Create clean template context
	ctx := ErrorContext{
		Schema:      schemaPath,
		Document:    truncateString(document, 500),
		Errors:      errors,
		ErrorCount:  len(errors),
		FullMessage: fullMessage,
	}

	// Execute Go template with helper functions
	tmpl := template.New("error").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})
	
	parsed, err := tmpl.Parse(errorTemplate)
	if err != nil {
		return fmt.Errorf("template parsing failed: %v", err)
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, ctx); err != nil {
		return fmt.Errorf("template execution failed: %v", err)
	}

	return fmt.Errorf("%s", buf.String())
}

// Common error message templates that users can reference
var CommonErrorTemplates = map[string]string{
	"basic": "{{range .Errors}}{{.Message}}\n{{end}}",
	"detailed": "{{.ErrorCount}} validation error(s) found:\n{{range $i, $e := .Errors}}{{printf \"%d. %s at %s\" (add $i 1) .Message .Path}}\n{{end}}",
	"simple": "{{.FullMessage}}",
	"with_path": "{{range .Errors}}{{.Path}}: {{.Message}}\n{{end}}",
	"with_schema": "Schema {{.Schema}} validation failed:\n{{.FullMessage}}",
	"verbose": "Validation Results:\nSchema: {{.Schema}}\nErrors: {{.ErrorCount}}\nFull Message: {{.FullMessage}}\n\nIndividual Errors:\n{{range $i, $e := .Errors}}Error {{add $i 1}}:\n  Path: {{.Path}}\n  Schema Path: {{.SchemaPath}}\n  Message: {{.Message}}{{if .Value}}\n  Value: {{.Value}}{{end}}\n\n{{end}}",
}

// GetCommonTemplate returns a predefined error template by name
func GetCommonTemplate(name string) (string, bool) {
	template, exists := CommonErrorTemplates[name]
	return template, exists
}

// extractValidationErrors recursively extracts all validation errors from the error tree
func extractValidationErrors(err *jsonschema.ValidationError) []ValidationErrorDetail {
	var errors []ValidationErrorDetail
	
	// If there are child causes, extract them individually (they contain the specific errors)
	if len(err.Causes) > 0 {
		for _, child := range err.Causes {
			errors = append(errors, extractValidationErrors(child)...)
		}
		// Sort errors for consistent ordering
		sortValidationErrors(errors)
		return errors
	}
	
	// If no child causes, this is a leaf error - use it directly
	detail := ValidationErrorDetail{
		Message: err.Error(),
		Path:    formatInstanceLocation(err.InstanceLocation),
	}
	
	// Use schema URL if available
	if err.SchemaURL != "" {
		detail.SchemaPath = err.SchemaURL
	}
	
	errors = append(errors, detail)
	return errors
}

// sortValidationErrors sorts validation errors for consistent ordering
// Primary sort: by Path (field name)
// Secondary sort: by Message (for same field, different constraint violations)  
func sortValidationErrors(errors []ValidationErrorDetail) {
	sort.Slice(errors, func(i, j int) bool {
		// First, sort by path
		if errors[i].Path != errors[j].Path {
			return errors[i].Path < errors[j].Path
		}
		// If paths are the same, sort by message
		return errors[i].Message < errors[j].Message
	})
}

// formatInstanceLocation formats the instance location path
// Handles both string keys (objects) and converts any other types to strings (array indices, etc.)
func formatInstanceLocation(location []string) string {
	if len(location) == 0 {
		return "/"
	}
	
	// All elements in location are already strings in v6, but let's ensure proper formatting
	var pathParts []string
	for _, part := range location {
		// Handle array indices and object keys uniformly
		pathParts = append(pathParts, part)
	}
	
	return "/" + strings.Join(pathParts, "/")
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
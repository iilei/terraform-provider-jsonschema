package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidationErrorDetail represents a single validation error with rich context
type ValidationErrorDetail struct {
	Message      string `json:"message"`      // Human-readable error message
	DocumentPath string `json:"documentPath"` // JSON Pointer to location in document where error occurred
	SchemaPath   string `json:"schemaPath"`   // JSON Pointer to schema constraint that failed
	Value        string `json:"value"`        // The actual value that failed validation (if available)
}

// ErrorContext holds data available for error message templating
type ErrorContext struct {
	SchemaFile  string                  `json:"schemaFile"`  // Path to the schema file
	Document    string                  `json:"document"`    // The document content (truncated if too long)
	Errors      []ValidationErrorDetail `json:"errors"`      // Individual validation errors with details
	ErrorCount  int                     `json:"errorCount"`  // Number of individual errors
	FullMessage string                  `json:"fullMessage"` // Complete formatted error message from jsonschema
}

// FormatValidationError creates a formatted error message using the provided template
func FormatValidationError(err error, schemaPath, document, errorTemplate string) error {
	if err == nil {
		return nil
	}

	var errors []ValidationErrorDetail
	var fullMessage string

	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		// Parse the document to extract actual values for errors
		var documentData interface{}
		if parseErr := json.Unmarshal([]byte(document), &documentData); parseErr != nil {
			// If we can't parse, try JSON5
			if data, err := ParseJSON5String(document); err == nil {
				documentData = data
			}
		}

		errors = extractValidationErrors(validationErr, documentData)
		// Generate full message using sorted errors for consistency
		fullMessage = generateSortedFullMessage(validationErr, errors)
	} else {
		// For non-validation errors, create a single error detail
		errors = []ValidationErrorDetail{{
			Message:      err.Error(),
			DocumentPath: "",
		}}
		fullMessage = err.Error()
	}

	// Create clean template context
	ctx := ErrorContext{
		SchemaFile:  schemaPath,
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
	"basic":       "{{range .Errors}}{{.Message}}\n{{end}}",
	"detailed":    "{{.ErrorCount}} validation error(s) found:\n{{range $i, $e := .Errors}}{{add $i 1}}. {{.Message}} at {{.DocumentPath}}\n{{end}}",
	"simple":      "{{.FullMessage}}",
	"with_path":   "{{range .Errors}}{{.DocumentPath}}: {{.Message}}\n{{end}}",
	"with_schema": "Schema {{.SchemaFile}} validation failed:\n{{.FullMessage}}",
	"verbose":     "Validation Results:\nSchema: {{.SchemaFile}}\nErrors: {{.ErrorCount}}\nFull Message: {{.FullMessage}}\n\nIndividual Errors:\n{{range $i, $e := .Errors}}Error {{add $i 1}}:\n  Document Path: {{.DocumentPath}}\n  Schema Path: {{.SchemaPath}}\n  Message: {{.Message}}{{if .Value}}\n  Value: {{.Value}}{{end}}\n\n{{end}}",
}

// GetCommonTemplate returns a predefined error template by name
func GetCommonTemplate(name string) (string, bool) {
	template, exists := CommonErrorTemplates[name]
	return template, exists
}

// generateSortedFullMessage creates a full error message using sorted errors for consistency
func generateSortedFullMessage(err *jsonschema.ValidationError, sortedErrors []ValidationErrorDetail) string {
	// Use the main error prefix from the original error
	prefix := fmt.Sprintf("jsonschema validation failed with '%s'", err.SchemaURL)

	// Build the error list using our sorted errors
	var errorLines []string
	for _, detail := range sortedErrors {
		// Extract just the validation message part (remove path if present)
		message := extractCleanMessage(detail.Message, detail.DocumentPath)

		// Use path as-is - per RFC 6901, "" (empty string) is root, not "/"
		displayPath := detail.DocumentPath

		errorLine := fmt.Sprintf("- at '%s': %s", displayPath, message)
		errorLines = append(errorLines, errorLine)
	}

	if len(errorLines) > 0 {
		return prefix + "\n" + strings.Join(errorLines, "\n")
	}

	return prefix
}

// extractCleanMessage removes path information from error messages if present
func extractCleanMessage(message, path string) string {
	// Per RFC 6901, root path is "" (empty string), not "/"
	// Error messages from the library will use this format: "at '<path>': <message>"
	expectedPrefix := fmt.Sprintf("at '%s': ", path)

	if strings.HasPrefix(message, expectedPrefix) {
		return message[len(expectedPrefix):]
	}

	// If no path prefix found, return message as-is
	return message
}

// extractValidationErrors recursively extracts all validation errors from the error tree
func extractValidationErrors(err *jsonschema.ValidationError, documentData interface{}) []ValidationErrorDetail {
	var errors []ValidationErrorDetail

	// If there are child causes, extract them individually (they contain the specific errors)
	if len(err.Causes) > 0 {
		for _, child := range err.Causes {
			errors = append(errors, extractValidationErrors(child, documentData)...)
		}
		// Sort errors for consistent ordering
		sortValidationErrors(errors)
		return errors
	}

	// If no child causes, this is a leaf error - use it directly
	detail := ValidationErrorDetail{
		Message:      err.Error(),
		DocumentPath: formatInstanceLocation(err.InstanceLocation),
		SchemaPath:   err.SchemaURL,
		Value:        extractValueAtPath(documentData, err.InstanceLocation),
	}

	errors = append(errors, detail)
	return errors
}

// extractValueAtPath retrieves the value at the given JSON path from the document
func extractValueAtPath(data interface{}, path []string) string {
	if data == nil || len(path) == 0 {
		// For root-level errors, try to show the whole document (truncated)
		if jsonBytes, err := json.Marshal(data); err == nil {
			valueStr := string(jsonBytes)
			if len(valueStr) > 100 {
				return valueStr[:100] + "..."
			}
			return valueStr
		}
		return ""
	}

	// Navigate to the value at the path
	current := data
	for _, key := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[key]
			if !ok {
				return "" // Path doesn't exist (e.g., missing required field)
			}
		case []interface{}:
			// Try to parse key as array index
			var idx int
			if _, err := fmt.Sscanf(key, "%d", &idx); err == nil && idx >= 0 && idx < len(v) {
				current = v[idx]
			} else {
				return ""
			}
		default:
			return "" // Can't navigate further
		}
	}

	// Serialize the value to JSON
	if jsonBytes, err := json.Marshal(current); err == nil {
		return string(jsonBytes)
	}

	return ""
}

// sortValidationErrors sorts validation errors for consistent ordering
// Primary sort: by DocumentPath (field name)
// Secondary sort: by Message (for same field, different constraint violations)
func sortValidationErrors(errors []ValidationErrorDetail) {
	sort.Slice(errors, func(i, j int) bool {
		// First, sort by path
		if errors[i].DocumentPath != errors[j].DocumentPath {
			return errors[i].DocumentPath < errors[j].DocumentPath
		}
		// If paths are the same, sort by message
		return errors[i].Message < errors[j].Message
	})
}

// formatInstanceLocation formats the instance location path according to JSON Pointer (RFC 6901)
// Per RFC 6901: empty string "" represents the root/whole document, not "/"
// A path like "/" would represent a field with an empty string as its key
func formatInstanceLocation(location []string) string {
	if len(location) == 0 {
		return "" // Empty string for root, per RFC 6901
	}

	// Build JSON Pointer: "/" + each reference token
	// Note: RFC 6901 requires escaping "~" as "~0" and "/" as "~1" in tokens
	// The jsonschema library should provide already-decoded tokens
	var pathParts []string
	pathParts = append(pathParts, location...)

	return "/" + strings.Join(pathParts, "/")
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

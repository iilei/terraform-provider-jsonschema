package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/iilei/terraform-provider-jsonschema/internal/provider"
)

func main() {
	// Create temp directory for demo files
	tmpDir := "/tmp/jsonschema_templating_demo"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Create a schema file that will produce multiple validation errors
	schemaPath := filepath.Join(tmpDir, "demo-schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["name", "age", "email"],
		"properties": {
			"name": {
				"type": "string",
				"minLength": 2
			},
			"age": {
				"type": "integer",
				"minimum": 0,
				"maximum": 150
			},
			"email": {
				"type": "string"
			},
			"status": {
				"enum": ["active", "inactive", "pending"]
			}
		}
	}`

	// Write schema to file
	if err := ioutil.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		fmt.Printf("Failed to write schema: %v\n", err)
		return
	}

	// Create invalid document with multiple validation errors
	invalidDocument := `{
		"name": "",
		"age": -5,
		"email": 123,
		"status": "invalid-status"
	}`

	fmt.Println("Enhanced Templating Demo - Multiple Validation Errors")
	fmt.Println("============================================================")
	fmt.Printf("Schema: %s\n", schemaPath)
	fmt.Printf("Document: %s\n", strings.ReplaceAll(invalidDocument, "\n", " "))
	fmt.Println()

	// Create a mock validation error for demonstration
	mockError := fmt.Errorf("validation failed with multiple issues")
	
	fmt.Println("Demo Templates (using mock validation error):")
	fmt.Println(strings.Repeat("=", 50))

	// Demo 1: Simple full message (default)
	fmt.Println("\n1. Simple Full Message (Default Template):")
	fmt.Println("Template: {{.FullMessage}}")
	err1 := provider.FormatValidationError(mockError, schemaPath, invalidDocument, "{{.FullMessage}}")
	fmt.Printf("Result: %s\n", err1.Error())

	// Demo 2: Individual error iteration
	fmt.Println("\n2. Individual Error Iteration:")
	fmt.Println("Template: Found {{.ErrorCount}} validation errors:\\n{{range $i, $e := .Errors}}  {{printf \"%d. %s (at %s)\" (add $i 1) .Message .Path}}\\n{{end}}")
	template2 := `Found {{.ErrorCount}} validation errors:
{{range $i, $e := .Errors}}  {{printf "%d. %s (at %s)" (add $i 1) .Message .Path}}
{{end}}`
	err2 := provider.FormatValidationError(mockError, schemaPath, invalidDocument, template2)
	fmt.Printf("Result:\n%s\n", err2.Error())

	// Demo 3: Simple path iteration
	fmt.Println("\n3. Simple Path:Message Format:")
	fmt.Println("Template: {{range .Errors}}{{.Path}}: {{.Message}}\\n{{end}}")
	template3 := `{{range .Errors}}{{.Path}}: {{.Message}}
{{end}}`
	err3 := provider.FormatValidationError(mockError, schemaPath, invalidDocument, template3)
	fmt.Printf("Result:\n%s", err3.Error())

	// Demo 4: CI/CD Format with individual errors
	fmt.Println("\n4. CI/CD Format:")
	fmt.Println("Template: {{range .Errors}}::error file={{$.Schema}}::{{.Message}}{{if .Path}} at {{.Path}}{{end}}\\n{{end}}")
	template4 := `{{range .Errors}}::error file={{$.Schema}}::{{.Message}}{{if .Path}} at {{.Path}}{{end}}
{{end}}`
	err4 := provider.FormatValidationError(mockError, schemaPath, invalidDocument, template4)
	fmt.Printf("Result:\n%s", err4.Error())

	// Demo 5: JSON Format for structured logging
	fmt.Println("\n5. JSON Format for Structured Logging:")
	fmt.Println("Template: {\"validation_failed\": true, \"schema\": \"{{.Schema}}\", \"error_count\": {{.ErrorCount}}, \"errors\": [{{range $i, $e := .Errors}}{{if $i}}, {{end}}{\"message\": \"{{.Message}}\", \"path\": \"{{.Path}}\"}{{end}}]}")
	template5 := `{"validation_failed": true, "schema": "{{.Schema}}", "error_count": {{.ErrorCount}}, "errors": [{{range $i, $e := .Errors}}{{if $i}}, {{end}}{"message": "{{.Message}}", "path": "{{.Path}}"}{{end}}]}`
	err5 := provider.FormatValidationError(mockError, schemaPath, invalidDocument, template5)
	fmt.Printf("Result: %s\n", err5.Error())

	// Demo 6: Using predefined templates
	fmt.Println("\n6. Predefined Templates:")
	
	templates := []string{"basic", "detailed", "simple", "with_path", "with_schema", "verbose"}
	for _, templateName := range templates {
		if template, found := provider.GetCommonTemplate(templateName); found {
			fmt.Printf("\n   %s template:\n", templateName)
			fmt.Printf("   %s\n", strings.ReplaceAll(template, "\n", "\\n"))
			err := provider.FormatValidationError(mockError, schemaPath, invalidDocument, template)
			fmt.Printf("   Result: %s\n", strings.TrimSpace(err.Error()))
		}
	}

	fmt.Println("\nEnhanced templating supports both simple full messages and detailed individual error iteration!")
	fmt.Println("\nAvailable Template Variables:")
	fmt.Println("• {{.Schema}}      - Path to schema file")
	fmt.Println("• {{.Document}}    - Document content (truncated if long)")
	fmt.Println("• {{.FullMessage}} - Complete formatted error from jsonschema")
	fmt.Println("• {{.Errors}}      - Array of individual ValidationErrorDetail")
	fmt.Println("• {{.ErrorCount}}  - Number of individual errors")
	fmt.Println("\nEach error in {{.Errors}} has:")
	fmt.Println("• .Message    - Human-readable error message")
	fmt.Println("• .Path       - JSON path where error occurred")  
	fmt.Println("• .SchemaPath - Schema path where validation failed")
	fmt.Println("• .Value      - Actual value that failed validation")
}
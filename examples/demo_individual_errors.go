package main

import (
	"fmt"

	"github.com/iilei/terraform-provider-jsonschema/internal/provider"
)

func main() {
	// Create a mock error to demonstrate the template functionality
	mockError := fmt.Errorf("required property 'name' missing")
	
	fmt.Println("Individual Error Iteration Demo")
	fmt.Println("==================================")
	fmt.Println()
	
	// Template 1: Individual error iteration
	fmt.Println("Template: {{range .Errors}}{{.Path}}: {{.Message}}\\n{{end}}")
	template1 := `{{range .Errors}}{{.Path}}: {{.Message}}
{{end}}`
	
	result1 := provider.FormatValidationError(mockError, "test.schema.json", `{"incomplete": "data"}`, template1)
	fmt.Printf("Result:\n%s\n", result1.Error())
	
	// Template 2: Error count with individual errors
	fmt.Println("Template: {{.ErrorCount}} error(s) found: {{range .Errors}}{{.Message}}{{end}}")
	template2 := `{{.ErrorCount}} error(s) found: {{range .Errors}}{{.Message}}{{end}}`
	
	result2 := provider.FormatValidationError(mockError, "test.schema.json", `{"incomplete": "data"}`, template2)
	fmt.Printf("Result: %s\n\n", result2.Error())
	
	// Template 3: Detailed individual error formatting
	fmt.Println("Template: Found {{.ErrorCount}} errors:\\n{{range $i, $e := .Errors}}  {{add $i 1}}. {{.Message}} (at {{.Path}})\\n{{end}}")
	template3 := `Found {{.ErrorCount}} errors:
{{range $i, $e := .Errors}}  {{add $i 1}}. {{.Message}} (at {{.Path}})
{{end}}`
	
	result3 := provider.FormatValidationError(mockError, "test.schema.json", `{"incomplete": "data"}`, template3)
	fmt.Printf("Result:\n%s\n", result3.Error())
	
	// Template 4: Full message vs individual comparison
	fmt.Println("Full Message Template: {{.FullMessage}}")
	template4 := `{{.FullMessage}}`
	
	result4 := provider.FormatValidationError(mockError, "test.schema.json", `{"incomplete": "data"}`, template4)
	fmt.Printf("Result: %s\n\n", result4.Error())
	
	// Show available template variables
	fmt.Println("Available Template Variables in ErrorContext:")
	fmt.Println("• {{.Schema}}      - Path to schema file") 
	fmt.Println("• {{.Document}}    - Document content (truncated)")
	fmt.Println("• {{.FullMessage}} - Complete formatted error")
	fmt.Println("• {{.Errors}}      - Array of ValidationErrorDetail")
	fmt.Println("• {{.ErrorCount}}  - Number of errors")
	fmt.Println()
	fmt.Println("Each ValidationErrorDetail has:")
	fmt.Println("• .Message    - Human readable error message")
	fmt.Println("• .Path       - JSON path where error occurred") 
	fmt.Println("• .SchemaPath - Schema path where validation failed")
	fmt.Println("• .Value      - Actual value that failed validation")
	fmt.Println()
	fmt.Println("💡 Use {{range .Errors}} to iterate over individual validation errors!")
}
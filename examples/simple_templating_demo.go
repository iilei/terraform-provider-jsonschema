package main

import (
	"fmt"
	"log"

	"github.com/iilei/terraform-provider-jsonschema/internal/provider"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func main() {
	// Create a test schema that will produce multiple validation errors
	schemaData := `{
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

	// Create a test document with multiple validation errors
	invalidDocument := `{
		"name": "",
		"age": -5,
		"email": 123,
		"status": "invalid-status"
	}`

	// Compile schema
	compiler := jsonschema.NewCompiler()
	schemaInterface, err := provider.ParseJSON5String(schemaData)
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}
	if err := compiler.AddResource("test://schema.json", schemaInterface); err != nil {
		log.Fatalf("Failed to add schema resource: %v", err)
	}
	
	schema, err := compiler.Compile("test://schema.json")
	if err != nil {
		log.Fatalf("Failed to compile schema: %v", err)
	}

	// Parse document 
	documentData, err := provider.ParseJSON5String(invalidDocument)
	if err != nil {
		log.Fatalf("Failed to parse document: %v", err)
	}

	// Validate and get validation error
	validationErr := schema.Validate(documentData)
	if validationErr == nil {
		log.Fatalf("Expected validation error but got none")
	}

	fmt.Println("Enhanced Templating Demo - Individual Error Iteration")
	fmt.Println("============================================================")
	
	// Show the raw validation error first
	fmt.Println("\nRaw validation error from jsonschema:")
	fmt.Printf("%s\n", validationErr.Error())

	// Demo: Individual error iteration with path
	fmt.Println("\n✨ Individual Error Iteration Template:")
	fmt.Println(`Template: "{{range .Errors}}{{.Path}}: {{.Message}}\n{{end}}"`)
	fmt.Println("\nResult:")
	
	template := `{{range .Errors}}{{.Path}}: {{.Message}}
{{end}}`
	
	err1 := provider.FormatValidationError(validationErr, "test://schema.json", invalidDocument, template)
	fmt.Printf("%s", err1.Error())

	// Demo 2: More detailed individual iteration
	fmt.Println("\n✨ Detailed Individual Error Template:")
	fmt.Println(`Template: "Found {{.ErrorCount}} errors:{{range $i, $e := .Errors}}\n  {{add $i 1}}. {{.Message}} (at {{.Path}}){{end}}"`)
	fmt.Println("\nResult:")
	
	template2 := `Found {{.ErrorCount}} errors:{{range $i, $e := .Errors}}
  {{add $i 1}}. {{.Message}} (at {{.Path}}){{end}}`
	
	err2 := provider.FormatValidationError(validationErr, "test://schema.json", invalidDocument, template2)
	fmt.Printf("%s\n", err2.Error())

	// Demo 3: Compare with full message
	fmt.Println("\n✨ Full Message vs Individual Errors:")
	fmt.Println("Full Message Template: {{.FullMessage}}")
	
	fullTemplate := `{{.FullMessage}}`
	err3 := provider.FormatValidationError(validationErr, "test://schema.json", invalidDocument, fullTemplate)
	fmt.Printf("Result: %s\n", err3.Error())

	fmt.Println("\n💡 This shows how you can iterate over individual validation errors!")
	fmt.Println("Each error has its own .Path and .Message for precise formatting.")
}
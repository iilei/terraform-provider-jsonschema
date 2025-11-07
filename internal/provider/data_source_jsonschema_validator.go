package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
)



func dataSourceJsonschemaValidator() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceJsonschemaValidatorRead,

		Schema: map[string]*schema.Schema{
			"document": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "JSON or JSON5 document content to validate",
			},
			"schema": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to JSON or JSON5 schema file",
			},
			"schema_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JSON Schema version override for this validation (overrides provider default)",
			},
			"error_message_template": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Template for formatting validation error messages. Available variables: {{.Schema}}, {{.Document}}, {{.FullMessage}}, {{.Errors}}, {{.ErrorCount}}. Use {{range .Errors}} to iterate over individual errors.",
			},

			"validated": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The validated document in canonical JSON format",
			},
		},
	}
}

func dataSourceJsonschemaValidatorRead(d *schema.ResourceData, m interface{}) error {
	config, ok := m.(*ProviderConfig)
	if !ok {
		return fmt.Errorf("invalid provider configuration")
	}

	document := d.Get("document").(string)
	schemaPath := d.Get("schema").(string)
	schemaVersionOverride := d.Get("schema_version").(string)
	errorMessageTemplate := d.Get("error_message_template").(string)
	
	// Use provider default if no template specified
	if errorMessageTemplate == "" {
		errorMessageTemplate = config.DefaultErrorTemplate
	}



	// Parse document (supports JSON5)
	documentData, err := ParseJSON5String(document)
	if err != nil {
		return fmt.Errorf("failed to parse document: %w", err)
	}

	// Read and parse schema file (supports JSON5)
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file %q: %w", schemaPath, err)
	}

	schemaData, err := ParseJSON5(schemaBytes)
	if err != nil {
		return fmt.Errorf("failed to parse schema file %q: %w", schemaPath, err)
	}

	// Create a new compiler instance for this validation
	compiler := jsonschema.NewCompiler()
	
	// Enable JSON5 support for $ref loading
	compiler.UseLoader(jsonschema.SchemeURLLoader{
		"file": JSON5FileLoader{},
	})
	
	// Determine which schema version to use
	effectiveSchemaVersion := config.DefaultSchemaVersion
	if schemaVersionOverride != "" {
		effectiveSchemaVersion = schemaVersionOverride
	}

	// Set the appropriate draft using DefaultDraft method (v6 API)
	if effectiveSchemaVersion != "" {
		draft, err := GetDraftForVersion(effectiveSchemaVersion)
		if err != nil {
			return err
		}
		compiler.DefaultDraft(draft)
	} else if config.DefaultDraft != nil {
		compiler.DefaultDraft(config.DefaultDraft)
	} else {
		// Fallback to Draft2020 if no draft is set
		compiler.DefaultDraft(jsonschema.Draft2020)
	}

	// Convert schema data to deterministic JSON string
	schemaJSON, err := MarshalDeterministic(schemaData)
	if err != nil {
		return fmt.Errorf("failed to convert schema to JSON: %w", err)
	}

	// Generate schema URL based on the schema file path
	schemaDir, err := filepath.Abs(filepath.Dir(schemaPath))
	if err != nil {
		return fmt.Errorf("failed to get absolute path for schema directory: %w", err)
	}
	schemaURL := fmt.Sprintf("file://%s/schema.json", schemaDir)

	// Add schema resource and compile (v6 API)
	var parsedSchemaData interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &parsedSchemaData); err != nil {
		return fmt.Errorf("failed to parse schema JSON: %w", err)
	}
	
	if err := compiler.AddResource(schemaURL, parsedSchemaData); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}
	
	compiledSchema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	// Validate the document
	if err := compiledSchema.Validate(documentData); err != nil {
		return FormatValidationError(err, schemaPath, document, errorMessageTemplate)
	}

	// Convert document to deterministic canonical JSON
	canonicalJSON, err := MarshalDeterministic(documentData)
	if err != nil {
		return fmt.Errorf("failed to convert document to canonical JSON: %w", err)
	}

	// Set the validated document
	if err := d.Set("validated", string(canonicalJSON)); err != nil {
		return fmt.Errorf("failed to set validated field: %w", err)
	}

	// Generate ID based on document, schema, and configuration
	// schemaJSON is already available from earlier in the function
	
	compositeString := fmt.Sprintf("%s:%s:%s", 
		string(canonicalJSON), 
		string(schemaJSON), 
		effectiveSchemaVersion,
	)
	d.SetId(hash(compositeString))

	return nil
}

func hash(s string) string {
	sha := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sha[:])
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/pflag"

	"github.com/iilei/terraform-provider-jsonschema/pkg/config"
	validator "github.com/iilei/terraform-provider-jsonschema/pkg/jsonschema"
)

const (
	ExitSuccess        = 0
	ExitValidationFail = 1
	ExitUsageError     = 2
)

var version = "dev" // Set by goreleaser

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitUsageError)
	}
}

func run() error {
	// Define flags
	var (
		showVersion   bool
		showHelp      bool
		configFile    string
		schemaPath    string
		schemaVersion string
		errorTemplate string
		refOverrides  []string
		documents     []string
		envPrefix     string
		forceFiletype string
	)

	pflag.BoolVarP(&showVersion, "version", "v", false, "Show version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show help and exit")
	pflag.StringVarP(&configFile, "config", "c", "", "Path to configuration file (.yaml, .toml, or .json)")
	pflag.StringVarP(&schemaPath, "schema", "s", "", "Path to JSON Schema file (required unless in config)")
	pflag.StringVar(&schemaVersion, "schema-version", "", "JSON Schema version (draft/2020-12, draft/2019-09, draft-07, draft-06, draft-04)")
	pflag.StringVarP(&errorTemplate, "error-template", "e", "", "Go template for error formatting")
	pflag.StringArrayVarP(&refOverrides, "ref-override", "r", nil, "Override $ref URL with local file (format: url=path)")
	pflag.StringArrayVarP(&documents, "document", "d", nil, "Document file(s) to validate (supports globs)")
	pflag.StringVar(&envPrefix, "env-prefix", "JSONSCHEMA_VALIDATOR_", "Environment variable prefix (must end with underscore)")
	pflag.StringVar(&forceFiletype, "force-filetype", "", "Force file type for documents (json, json5, yaml, toml). Auto-detected from extension if not set")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, `jsonschema-validator - Validate JSON/JSON5/YAML/TOML documents against JSON Schema

Usage:
  jsonschema-validator [flags] [documents...]

Flags:
`)
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Validate a JSON document
  jsonschema-validator -s schema.json config.json

  # Validate a YAML document (auto-detected)
  jsonschema-validator -s schema.json deployment.yaml

  # Validate a TOML document (auto-detected)
  jsonschema-validator -s schema.json config.toml

  # Force file type override
  jsonschema-validator -s schema.json --force-filetype yaml data.txt

  # Validate multiple documents
  jsonschema-validator -s schema.json doc1.json doc2.yaml doc3.toml

  # Use glob patterns
  jsonschema-validator -s schema.json "configs/*.yaml"

  # Override remote $ref with local file
  jsonschema-validator -s schema.json -r https://example.com/schema.json=./local.json doc.json

  # Use configuration file
  jsonschema-validator -c .jsonschema-validator.yaml

  # Configuration auto-discovery (checks in order):
  #   1. .jsonschema-validator.yaml (or .yml, .toml, .json)
  #   2. pyproject.toml [tool.jsonschema-validator]
  #   3. package.json "jsonschema-validator" field
  #   4. ~/.jsonschema-validator.yaml

For more information, see: https://github.com/iilei/terraform-provider-jsonschema
`)
	}

	pflag.Parse()

	if showVersion {
		fmt.Printf("jsonschema-validator version %s\n", version)
		return nil
	}

	if showHelp {
		pflag.Usage()
		return nil
	}

	// Load configuration
	loader := config.NewLoader()

	// Set custom environment prefix if provided
	if envPrefix != "" {
		// Ensure prefix ends with underscore
		if !strings.HasSuffix(envPrefix, "_") {
			envPrefix += "_"
		}
		loader.SetEnvPrefix(envPrefix)
	}

	// Load from specific config file if provided
	var cfg *config.Config
	var err error

	if configFile != "" {
		cfg, err = loader.LoadFromFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Auto-discover configuration
		cfg, err = loader.Load(pflag.CommandLine)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	// Command-line arguments override configuration
	if schemaPath != "" || len(documents) > 0 {
		// Build schema config from command-line args
		schemaConfig := config.SchemaConfig{
			Path:      schemaPath,
			Documents: append(documents, pflag.Args()...),
		}

		if schemaVersion != "" {
			schemaConfig.SchemaVersion = schemaVersion
		}

		if errorTemplate != "" {
			schemaConfig.ErrorTemplate = errorTemplate
		}

		if len(refOverrides) > 0 {
			schemaConfig.RefOverrides = config.ParseRefOverridesFromSlice(refOverrides)
		}

		// If no schemas in config, use command-line schema
		if len(cfg.Schemas) == 0 {
			cfg.Schemas = []config.SchemaConfig{schemaConfig}
		} else {
			// Override first schema with command-line args
			cfg.Schemas[0] = schemaConfig
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Expand globs in document paths
	for i := range cfg.Schemas {
		expanded, err := cfg.Schemas[i].ExpandDocumentGlobs()
		if err != nil {
			return fmt.Errorf("failed to expand glob patterns: %w", err)
		}
		cfg.Schemas[i].Documents = expanded
	}

	// Validate all schemas
	hasErrors := false
	for _, schemaConfig := range cfg.Schemas {
		if err := validateSchema(schemaConfig, cfg, forceFiletype); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			hasErrors = true
		}
	}

	if hasErrors {
		os.Exit(ExitValidationFail)
	}

	return nil
}

func validateSchema(schemaConfig config.SchemaConfig, globalConfig *config.Config, forceFiletype string) error {
	// Read and parse schema (auto-detect format)
	schemaData, err := validator.ParseFile(schemaConfig.Path, validator.FileTypeAuto)
	if err != nil {
		return fmt.Errorf("failed to parse schema %q: %w", schemaConfig.Path, err)
	}

	// Create compiler
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(jsonschema.SchemeURLLoader{
		"file": validator.JSON5FileLoader{},
	})

	// Set schema version
	effectiveVersion := schemaConfig.GetEffectiveSchemaVersion(globalConfig.SchemaVersion)
	if effectiveVersion != "" {
		draft, err := getDraftForVersion(effectiveVersion)
		if err != nil {
			return err
		}
		compiler.DefaultDraft(draft)
	}

	// Merge and register ref overrides
	mergedOverrides := config.MergeRefOverrides(
		globalConfig.Schemas[0].RefOverrides, // Global overrides
		schemaConfig.RefOverrides,            // Schema-specific overrides
	)

	for remoteURL, localPath := range mergedOverrides {
		// Parse ref override file (auto-detect format)
		overrideData, err := validator.ParseFile(localPath, validator.FileTypeAuto)
		if err != nil {
			return fmt.Errorf("ref-override: failed to parse %q for URL %q: %w", localPath, remoteURL, err)
		}

		if err := compiler.AddResource(remoteURL, overrideData); err != nil {
			return fmt.Errorf("ref-override: failed to register %q -> %q: %w", remoteURL, localPath, err)
		}
	}

	// Add and compile schema
	schemaAbsPath, err := filepath.Abs(schemaConfig.Path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for schema: %w", err)
	}
	schemaURL := fmt.Sprintf("file://%s", schemaAbsPath)

	if err := compiler.AddResource(schemaURL, schemaData); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	compiledSchema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	// Validate each document
	hasErrors := false
	for _, docPath := range schemaConfig.Documents {
		if err := validateDocument(docPath, compiledSchema, schemaConfig, globalConfig, forceFiletype); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed for schema %q", schemaConfig.Path)
	}

	return nil
}

func validateDocument(docPath string, schema *jsonschema.Schema, schemaConfig config.SchemaConfig, globalConfig *config.Config, flagForceFiletype string) error {
	// Get effective force_filetype: command-line flag > config file > auto-detect
	effectiveForceFiletype := schemaConfig.GetEffectiveForceFiletype(flagForceFiletype)

	// Parse document file with optional forced file type
	fileType := validator.FileType(effectiveForceFiletype)
	if fileType == "" {
		fileType = validator.FileTypeAuto
	}

	docData, err := validator.ParseFile(docPath, fileType)
	if err != nil {
		return fmt.Errorf("failed to parse document %q: %w", docPath, err)
	}

	// Validate
	if err := schema.Validate(docData); err != nil {
		effectiveTemplate := schemaConfig.GetEffectiveErrorTemplate(globalConfig.ErrorTemplate)
		if effectiveTemplate == "" {
			effectiveTemplate = "{{.FullMessage}}"
		}

		formattedErr := validator.FormatValidationError(err, schemaConfig.Path, docPath, effectiveTemplate)
		return fmt.Errorf("document %q: %w", docPath, formattedErr)
	}

	fmt.Printf("✓ %s: valid\n", docPath)
	return nil
}

func getDraftForVersion(version string) (*jsonschema.Draft, error) {
	// Normalize version string
	version = strings.ToLower(strings.TrimSpace(version))

	switch version {
	case "draft/2020-12", "2020-12", "draft-2020-12":
		return jsonschema.Draft2020, nil
	case "draft/2019-09", "2019-09", "draft-2019-09":
		return jsonschema.Draft2019, nil
	case "draft-07", "draft/07", "7":
		return jsonschema.Draft7, nil
	case "draft-06", "draft/06", "6":
		return jsonschema.Draft6, nil
	case "draft-04", "draft/04", "4":
		return jsonschema.Draft4, nil
	default:
		return nil, fmt.Errorf("unsupported schema version: %q (supported: draft/2020-12, draft/2019-09, draft-07, draft-06, draft-04)", version)
	}
}

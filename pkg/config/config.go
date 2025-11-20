package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the complete configuration for jsonschema-validator
// All field names and structure match the Terraform provider for consistency
type Config struct {
	// SchemaVersion is the default JSON Schema draft version
	// Matches Terraform provider's "schema_version" field
	// Valid values: "draft-4", "draft-6", "draft-7", "draft/2019-09", "draft/2020-12"
	SchemaVersion string `koanf:"schema_version" json:"schemaVersion" yaml:"schema_version" toml:"schema_version" mapstructure:"schema_version"`

	// Schemas is a list of schema-document mappings
	// Each schema can validate multiple documents with glob pattern support
	Schemas []SchemaConfig `koanf:"schemas" json:"schemas" yaml:"schemas" toml:"schemas" mapstructure:"schemas"`

	// ErrorTemplate is a custom error message template using Go template syntax
	// Matches Terraform provider's "error_message_template" field
	ErrorTemplate string `koanf:"error_template" json:"errorTemplate" yaml:"error_template" toml:"error_template" mapstructure:"error_template"`
}

// SchemaConfig represents a single schema with its document mappings
// Matches Terraform provider's data source configuration
type SchemaConfig struct {
	// Path is the path to the JSON Schema file (supports JSON5)
	// Matches Terraform provider's "schema" field
	Path string `koanf:"path" json:"path" yaml:"path" toml:"path" mapstructure:"path"`

	// Documents is a list of document file paths or glob patterns
	// Matches Terraform provider's "document" field (but allows multiple)
	Documents []string `koanf:"documents" json:"documents" yaml:"documents" toml:"documents" mapstructure:"documents"`

	// ForceFiletype overrides automatic file type detection for documents
	// Matches Terraform provider's "force_filetype" field
	// Valid values: "json", "json5", "yaml", "toml"
	// Empty string means auto-detect from file extension
	ForceFiletype string `koanf:"force_filetype" json:"forceFiletype" yaml:"force_filetype" toml:"force_filetype" mapstructure:"force_filetype"`

	// RefOverrides maps remote $ref URLs to local file paths
	// Matches Terraform provider's "ref_overrides" map
	// Key: remote URL (e.g., "https://example.com/schema.json")
	// Value: local file path (e.g., "./schemas/local.json")
	RefOverrides map[string]string `koanf:"ref_overrides" json:"refOverrides" yaml:"ref_overrides" toml:"ref_overrides" mapstructure:"ref_overrides"`

	// SchemaVersion overrides the global schema version for this schema
	// Matches Terraform provider's "schema_version" field at resource level
	SchemaVersion string `koanf:"schema_version" json:"schemaVersion" yaml:"schema_version" toml:"schema_version" mapstructure:"schema_version"`

	// ErrorTemplate overrides the global error template for this schema
	// Matches Terraform provider's "error_message_template" field at resource level
	ErrorTemplate string `koanf:"error_template" json:"errorTemplate" yaml:"error_template" toml:"error_template" mapstructure:"error_template"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Schemas) == 0 {
		return fmt.Errorf("no schemas configured")
	}

	for i, schema := range c.Schemas {
		if err := schema.Validate(); err != nil {
			return fmt.Errorf("schema[%d]: %w", i, err)
		}
	}

	return nil
}

// Validate checks if a schema configuration is valid
func (s *SchemaConfig) Validate() error {
	if s.Path == "" {
		return fmt.Errorf("schema path is required")
	}

	if len(s.Documents) == 0 {
		return fmt.Errorf("at least one document is required")
	}

	// Check if schema file exists
	if _, err := os.Stat(s.Path); err != nil {
		return fmt.Errorf("schema file %q: %w", s.Path, err)
	}

	return nil
}

// GetEffectiveSchemaVersion returns the schema version to use
// Priority: schema-level > global-level > empty (use schema's $schema field)
func (s *SchemaConfig) GetEffectiveSchemaVersion(globalVersion string) string {
	if s.SchemaVersion != "" {
		return s.SchemaVersion
	}
	return globalVersion
}

// GetEffectiveErrorTemplate returns the error template to use
// Priority: schema-level > global-level > empty (use default)
func (s *SchemaConfig) GetEffectiveErrorTemplate(globalTemplate string) string {
	if s.ErrorTemplate != "" {
		return s.ErrorTemplate
	}
	return globalTemplate
}

// GetEffectiveForceFiletype returns the force filetype to use
// Priority: command-line flag > schema-level > empty (auto-detect)
func (s *SchemaConfig) GetEffectiveForceFiletype(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return s.ForceFiletype
}

// MergeRefOverrides merges multiple ref_override sources with priority
// Later sources override earlier ones (command-line > config file > defaults)
func MergeRefOverrides(sources ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, source := range sources {
		for key, value := range source {
			result[key] = value
		}
	}

	return result
}

// ExpandDocumentGlobs expands glob patterns in document paths
func (s *SchemaConfig) ExpandDocumentGlobs() ([]string, error) {
	var expanded []string

	for _, pattern := range s.Documents {
		// Check if pattern contains glob characters
		if !containsGlobChars(pattern) {
			// Not a glob pattern, add as-is
			expanded = append(expanded, pattern)
			continue
		}

		// Expand glob pattern
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}

		if len(matches) == 0 {
			// No matches found - this might be intentional (e.g., no files yet)
			// Don't treat as error, just skip
			continue
		}

		expanded = append(expanded, matches...)
	}

	return expanded, nil
}

// containsGlobChars checks if a string contains glob pattern characters
func containsGlobChars(s string) bool {
	for _, ch := range s {
		if ch == '*' || ch == '?' || ch == '[' {
			return true
		}
	}
	return false
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		SchemaVersion: "", // Empty means use schema's $schema field
		Schemas:       []SchemaConfig{},
		ErrorTemplate: "", // Empty means use default error formatting
	}
}

// NewSchemaConfig creates a new schema configuration
func NewSchemaConfig(schemaPath string, documents ...string) *SchemaConfig {
	return &SchemaConfig{
		Path:         schemaPath,
		Documents:    documents,
		RefOverrides: make(map[string]string),
	}
}

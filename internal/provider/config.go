package provider

import (
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ProviderConfig holds the provider-level configuration
type ProviderConfig struct {
	// DefaultSchemaVersion specifies the JSON Schema version to use when not specified in the schema
	DefaultSchemaVersion string
	
	// DefaultErrorTemplate is the default error message template
	DefaultErrorTemplate string
	
	// DetailedErrors enables detailed structured error output
	DetailedErrors bool
	
	// DefaultDraft is the default draft to use
	DefaultDraft *jsonschema.Draft
}

// NewProviderConfig creates a new provider configuration with defaults
func NewProviderConfig(schemaVersion, errorTemplate string, detailedErrors bool) (*ProviderConfig, error) {
	// Set sensible default for error template if empty
	if errorTemplate == "" {
		errorTemplate = "{{.FullMessage}}"
	}
	
	config := &ProviderConfig{
		DefaultSchemaVersion: schemaVersion,
		DefaultErrorTemplate: errorTemplate,
		DetailedErrors:       detailedErrors,
		DefaultDraft:         jsonschema.Draft2020, // Default to latest draft
	}
	
	// Set draft based on schema version if provided
	if schemaVersion != "" {
		draft, err := GetDraftForVersion(schemaVersion)
		if err != nil {
			return nil, err
		}
		config.DefaultDraft = draft
	}
	
	return config, nil
}

// GetDraftForVersion returns the appropriate draft for a given schema version string
func GetDraftForVersion(version string) (*jsonschema.Draft, error) {
	switch version {
	case "draft-04", "http://json-schema.org/draft-04/schema#":
		return jsonschema.Draft4, nil
	case "draft-06", "http://json-schema.org/draft-06/schema#":
		return jsonschema.Draft6, nil
	case "draft-07", "http://json-schema.org/draft-07/schema#":
		return jsonschema.Draft7, nil
	case "draft/2019-09", "https://json-schema.org/draft/2019-09/schema":
		return jsonschema.Draft2019, nil
	case "draft/2020-12", "https://json-schema.org/draft/2020-12/schema":
		return jsonschema.Draft2020, nil
	default:
		return nil, fmt.Errorf("unsupported JSON Schema version: %s", version)
	}
}
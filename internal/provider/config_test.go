package provider

import (
	"strings"
	"testing"
)

func TestNewProviderConfig(t *testing.T) {
	tests := []struct {
		name           string
		schemaVersion  string
		errorTemplate string
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid draft-07",
			schemaVersion: "draft-07",
			errorTemplate: "Error: {error}",
			expectError:   false,
		},
		{
			name:          "valid draft-04",
			schemaVersion: "draft-04",
			errorTemplate: "",
			expectError:   false,
		},
		{
			name:          "valid draft-06",
			schemaVersion: "draft-06",
			errorTemplate: "",
			expectError:   false,
		},
		{
			name:          "valid draft/2019-09",
			schemaVersion: "draft/2019-09",
			errorTemplate: "",
			expectError:   false,
		},
		{
			name:          "valid draft/2020-12",
			schemaVersion: "draft/2020-12",
			errorTemplate: "",
			expectError:   false,
		},
		{
			name:          "invalid schema version",
			schemaVersion: "invalid-version",
			errorTemplate: "",
			expectError:   true,
			errorContains: "unsupported JSON Schema version",
		},
		{
			name:          "default empty values",
			schemaVersion: "",
			errorTemplate: "",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewProviderConfig(tt.schemaVersion, tt.errorTemplate, false)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Errorf("expected config to be non-nil")
				return
			}

			// Test defaults
			if tt.schemaVersion == "" && config.DefaultSchemaVersion != "" {
				// Default schema version should be empty when not specified
				t.Errorf("expected empty default schema version, got %q", config.DefaultSchemaVersion)
			}

			if tt.errorTemplate == "" && config.DefaultErrorTemplate != "{{.FullMessage}}" {
				t.Errorf("expected default error template, got %q", config.DefaultErrorTemplate)
			}
		})
	}
}

func TestGetDraftForVersion(t *testing.T) {
	tests := []struct {
		version     string
		expectError bool
	}{
		{"draft-04", false},
		{"draft-06", false},
		{"draft-07", false},
		{"draft/2019-09", false},
		{"draft/2020-12", false},
		{"invalid", true},
		{"", true}, // empty should be invalid
		{"draft-05", true}, // non-existent version
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			_, err := GetDraftForVersion(tt.version)
			if tt.expectError && err == nil {
				t.Errorf("expected error for version %q", tt.version)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for version %q: %v", tt.version, err)
			}
		})
	}
}

func TestGetDraftForVersionURLFormats(t *testing.T) {
	// Test all URL format variants including http://json-schema.org URLs
	tests := []struct {
		version     string
		expectError bool
		description string
	}{
		{
			version:     "http://json-schema.org/draft-04/schema#",
			expectError: false,
			description: "draft-04 with full URL",
		},
		{
			version:     "http://json-schema.org/draft-06/schema#",
			expectError: false,
			description: "draft-06 with full URL",
		},
		{
			version:     "http://json-schema.org/draft-07/schema#",
			expectError: false,
			description: "draft-07 with full URL",
		},
		{
			version:     "https://json-schema.org/draft/2019-09/schema",
			expectError: false,
			description: "draft 2019-09 with https URL",
		},
		{
			version:     "https://json-schema.org/draft/2020-12/schema",
			expectError: false,
			description: "draft 2020-12 with https URL",
		},
		{
			version:     "http://json-schema.org/draft-08/schema#",
			expectError: true,
			description: "non-existent draft-08",
		},
		{
			version:     "https://json-schema.org/draft/2021-01/schema",
			expectError: true,
			description: "non-existent draft 2021-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			draft, err := GetDraftForVersion(tt.version)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for %s, but got none", tt.description)
				}
				if draft != nil {
					t.Errorf("expected nil draft for error case, got %v", draft)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %s: %v", tt.description, err)
				}
				if draft == nil {
					t.Errorf("expected non-nil draft for %s", tt.description)
				}
			}
		})
	}
}

func TestNewProviderConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		schemaVersion string
		errorTemplate string
		detailedErrs  bool
		expectError   bool
		validateFunc  func(*testing.T, *ProviderConfig)
	}{
		{
			name:          "empty template gets default",
			schemaVersion: "",
			errorTemplate: "",
			detailedErrs:  false,
			expectError:   false,
			validateFunc: func(t *testing.T, cfg *ProviderConfig) {
				if cfg.DefaultErrorTemplate != "{{.FullMessage}}" {
					t.Errorf("expected default template, got %q", cfg.DefaultErrorTemplate)
				}
			},
		},
		{
			name:          "custom template preserved",
			schemaVersion: "",
			errorTemplate: "Custom: {{.Errors}}",
			detailedErrs:  false,
			expectError:   false,
			validateFunc: func(t *testing.T, cfg *ProviderConfig) {
				if cfg.DefaultErrorTemplate != "Custom: {{.Errors}}" {
					t.Errorf("expected custom template, got %q", cfg.DefaultErrorTemplate)
				}
			},
		},
		{
			name:          "detailed errors flag",
			schemaVersion: "",
			errorTemplate: "",
			detailedErrs:  true,
			expectError:   false,
			validateFunc: func(t *testing.T, cfg *ProviderConfig) {
				if !cfg.DetailedErrors {
					t.Error("expected DetailedErrors to be true")
				}
			},
		},
		{
			name:          "all parameters set",
			schemaVersion: "draft-07",
			errorTemplate: "Error: {{.FullMessage}}",
			detailedErrs:  true,
			expectError:   false,
			validateFunc: func(t *testing.T, cfg *ProviderConfig) {
				if cfg.DefaultSchemaVersion != "draft-07" {
					t.Errorf("expected schema version draft-07, got %q", cfg.DefaultSchemaVersion)
				}
				if cfg.DefaultErrorTemplate != "Error: {{.FullMessage}}" {
					t.Errorf("expected custom template, got %q", cfg.DefaultErrorTemplate)
				}
				if !cfg.DetailedErrors {
					t.Error("expected DetailedErrors to be true")
				}
				if cfg.DefaultDraft == nil {
					t.Error("expected DefaultDraft to be set")
				}
			},
		},
		{
			name:          "invalid schema version",
			schemaVersion: "draft-99",
			errorTemplate: "",
			detailedErrs:  false,
			expectError:   true,
			validateFunc:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewProviderConfig(tt.schemaVersion, tt.errorTemplate, tt.detailedErrs)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Fatal("expected non-nil config")
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, cfg)
			}
		})
	}
}
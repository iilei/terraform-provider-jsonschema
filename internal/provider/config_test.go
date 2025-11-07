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
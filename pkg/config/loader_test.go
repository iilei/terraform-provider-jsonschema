package config

import (
	"os"
	"path/filepath"
	"testing"

	flag "github.com/spf13/pflag"
)

func TestLoader_LoadDefaults(t *testing.T) {
	loader := NewLoader()
	cfg, err := loader.Load(nil)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.SchemaVersion != "" {
		t.Errorf("default schema_version = %q, want empty", cfg.SchemaVersion)
	}
	if cfg.ErrorTemplate != "" {
		t.Errorf("default error_template = %q, want empty", cfg.ErrorTemplate)
	}
	if len(cfg.Schemas) != 0 {
		t.Errorf("default schemas has %d items, want 0", len(cfg.Schemas))
	}
}

func TestLoader_LoadFromYAMLFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `
schema_version: "draft/2020-12"
error_template: "Error: {{.FullMessage}}"
schemas:
  - path: "test.schema.json"
    documents:
      - "test.json"
      - "test.*.json"
    ref_overrides:
      "https://example.com/user.json": "./local/user.json"
      "https://example.com/product.json": "./local/product.json"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	cfg, err := loader.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	if cfg.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q", cfg.SchemaVersion, "draft/2020-12")
	}

	if cfg.ErrorTemplate != "Error: {{.FullMessage}}" {
		t.Errorf("error_template = %q, want %q", cfg.ErrorTemplate, "Error: {{.FullMessage}}")
	}

	if len(cfg.Schemas) != 1 {
		t.Fatalf("schemas has %d items, want 1", len(cfg.Schemas))
	}

	schema := cfg.Schemas[0]
	if schema.Path != "test.schema.json" {
		t.Errorf("schema.path = %q, want %q", schema.Path, "test.schema.json")
	}

	if len(schema.Documents) != 2 {
		t.Errorf("schema.documents has %d items, want 2", len(schema.Documents))
	}

	if len(schema.RefOverrides) != 2 {
		t.Errorf("schema.ref_overrides has %d items, want 2", len(schema.RefOverrides))
	}

	if schema.RefOverrides["https://example.com/user.json"] != "./local/user.json" {
		t.Errorf("ref_override mismatch for user.json")
	}
}

func TestLoader_LoadFromTOMLFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `
schema_version = "draft/2020-12"

[[schemas]]
path = "test.schema.json"
documents = ["test.json"]

[schemas.ref_overrides]
"https://example.com/user.json" = "./local/user.json"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	cfg, err := loader.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	if cfg.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q", cfg.SchemaVersion, "draft/2020-12")
	}

	if len(cfg.Schemas) != 1 {
		t.Fatalf("schemas has %d items, want 1", len(cfg.Schemas))
	}
}

func TestLoader_LoadFromJSONFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	// JSON configs now use snake_case for consistency with YAML/TOML
	configContent := `{
  "schema_version": "draft/2020-12",
  "schemas": [
    {
      "path": "test.schema.json",
      "documents": ["test.json"],
      "ref_overrides": {
        "https://example.com/user.json": "./local/user.json"
      }
    }
  ]
}`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	cfg, err := loader.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	if cfg.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q", cfg.SchemaVersion, "draft/2020-12")
	}
}

func TestLoader_LoadProjectConfig(t *testing.T) {
	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create .jsonschema-validator.yaml
	configContent := `
schema_version: "draft/2020-12"
schemas:
  - path: "test.schema.json"
    documents: ["test.json"]
`

	if err := os.WriteFile(".jsonschema-validator.yaml", []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load should find the project config
	loader := NewLoader()
	cfg, err := loader.Load(nil)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q", cfg.SchemaVersion, "draft/2020-12")
	}
}

func TestLoader_LoadPyprojectTOML(t *testing.T) {
	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create pyproject.toml
	configContent := `
[project]
name = "test-project"

[tool.jsonschema-validator]
schema_version = "draft/2020-12"

[[tool.jsonschema-validator.schemas]]
path = "test.schema.json"
documents = ["test.json"]
`

	if err := os.WriteFile("pyproject.toml", []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load should find the pyproject.toml config
	loader := NewLoader()
	cfg, err := loader.Load(nil)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q", cfg.SchemaVersion, "draft/2020-12")
	}
}

func TestLoader_LoadEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("JSONSCHEMA_VALIDATOR_SCHEMA_VERSION", "draft/2019-09")
	defer os.Unsetenv("JSONSCHEMA_VALIDATOR_SCHEMA_VERSION")

	loader := NewLoader()
	cfg, err := loader.Load(nil)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.SchemaVersion != "draft/2019-09" {
		t.Errorf("schema_version = %q, want %q", cfg.SchemaVersion, "draft/2019-09")
	}
}

func TestLoader_CustomEnvPrefix(t *testing.T) {
	// Set environment variables with custom prefix
	os.Setenv("MY_APP_SCHEMA_VERSION", "draft/2020-12")
	os.Setenv("MY_APP_ERROR_TEMPLATE", "Custom: {{.FullMessage}}")
	defer os.Unsetenv("MY_APP_SCHEMA_VERSION")
	defer os.Unsetenv("MY_APP_ERROR_TEMPLATE")

	loader := NewLoader()
	loader.SetEnvPrefix("MY_APP_")

	cfg, err := loader.Load(nil)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q", cfg.SchemaVersion, "draft/2020-12")
	}

	if cfg.ErrorTemplate != "Custom: {{.FullMessage}}" {
		t.Errorf("error_template = %q, want %q", cfg.ErrorTemplate, "Custom: {{.FullMessage}}")
	}

	// Verify default prefix doesn't work
	os.Setenv("JSONSCHEMA_VALIDATOR_SCHEMA_VERSION", "draft/2019-09")
	defer os.Unsetenv("JSONSCHEMA_VALIDATOR_SCHEMA_VERSION")

	loader2 := NewLoader()
	loader2.SetEnvPrefix("MY_APP_")

	cfg2, err := loader2.Load(nil)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should still use MY_APP_ prefix, not JSONSCHEMA_VALIDATOR_
	if cfg2.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q (should ignore JSONSCHEMA_VALIDATOR_ prefix)", cfg2.SchemaVersion, "draft/2020-12")
	}
}

func TestLoader_LoadFlags(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	schemaVersion := flags.String("schema.version", "", "Schema version")

	flags.Parse([]string{"--schema.version", "draft/2019-09"})

	loader := NewLoader()

	// Load flags using posflag provider
	if err := loader.loadFlags(flags); err != nil {
		t.Fatalf("Load flags failed: %v", err)
	}

	var cfg Config
	if err := loader.k.UnmarshalWithConf("", &cfg, unmarshalConf); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify the flag was loaded
	if *schemaVersion != "draft/2019-09" {
		t.Errorf("flag value = %q, want %q", *schemaVersion, "draft/2019-09")
	}

	// Note: Flag to config field mapping happens in CLI main.go
	// This test verifies the flag loading mechanism works
}

func TestParseRefOverridesFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "empty string",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "single override",
			input: "https://example.com/user.json=./local/user.json",
			want: map[string]string{
				"https://example.com/user.json": "./local/user.json",
			},
		},
		{
			name:  "multiple overrides",
			input: "https://example.com/user.json=./local/user.json,https://example.com/product.json=./local/product.json",
			want: map[string]string{
				"https://example.com/user.json":    "./local/user.json",
				"https://example.com/product.json": "./local/product.json",
			},
		},
		{
			name:  "with whitespace",
			input: " https://example.com/user.json = ./local/user.json ",
			want: map[string]string{
				"https://example.com/user.json": "./local/user.json",
			},
		},
		{
			name:  "invalid format ignored",
			input: "invalid,https://example.com/user.json=./local/user.json",
			want: map[string]string{
				"https://example.com/user.json": "./local/user.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRefOverridesFromString(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseRefOverridesFromString() got %d items, want %d", len(got), len(tt.want))
			}
			for key, wantVal := range tt.want {
				if gotVal, ok := got[key]; !ok {
					t.Errorf("ParseRefOverridesFromString() missing key %q", key)
				} else if gotVal != wantVal {
					t.Errorf("ParseRefOverridesFromString()[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestParseRefOverridesFromSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  map[string]string
	}{
		{
			name:  "empty slice",
			input: []string{},
			want:  map[string]string{},
		},
		{
			name: "single override",
			input: []string{
				"https://example.com/user.json=./local/user.json",
			},
			want: map[string]string{
				"https://example.com/user.json": "./local/user.json",
			},
		},
		{
			name: "multiple overrides",
			input: []string{
				"https://example.com/user.json=./local/user.json",
				"https://example.com/product.json=./local/product.json",
			},
			want: map[string]string{
				"https://example.com/user.json":    "./local/user.json",
				"https://example.com/product.json": "./local/product.json",
			},
		},
		{
			name: "duplicate keys - last wins",
			input: []string{
				"https://example.com/user.json=./old.json",
				"https://example.com/user.json=./new.json",
			},
			want: map[string]string{
				"https://example.com/user.json": "./new.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRefOverridesFromSlice(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseRefOverridesFromSlice() got %d items, want %d", len(got), len(tt.want))
			}
			for key, wantVal := range tt.want {
				if gotVal, ok := got[key]; !ok {
					t.Errorf("ParseRefOverridesFromSlice() missing key %q", key)
				} else if gotVal != wantVal {
					t.Errorf("ParseRefOverridesFromSlice()[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestLoader_ConfigPriority(t *testing.T) {
	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create .jsonschema-validator.yaml with one value
	yamlConfig := `schema_version: "draft-7"`
	if err := os.WriteFile(".jsonschema-validator.yaml", []byte(yamlConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Set environment variable with higher priority
	os.Setenv("JSONSCHEMA_VALIDATOR_SCHEMA_VERSION", "draft/2020-12")
	defer os.Unsetenv("JSONSCHEMA_VALIDATOR_SCHEMA_VERSION")

	loader := NewLoader()
	cfg, err := loader.Load(nil)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Environment variable should override file config
	if cfg.SchemaVersion != "draft/2020-12" {
		t.Errorf("schema_version = %q, want %q (env var should override file)", cfg.SchemaVersion, "draft/2020-12")
	}
}

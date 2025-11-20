package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				SchemaVersion: "draft/2020-12",
				Schemas: []SchemaConfig{
					{
						Path:      "testdata/schema.json",
						Documents: []string{"test.json"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no schemas",
			config: &Config{
				SchemaVersion: "draft/2020-12",
				Schemas:       []SchemaConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test files for valid test
			if !tt.wantErr && len(tt.config.Schemas) > 0 {
				tempDir := t.TempDir()
				schemaPath := filepath.Join(tempDir, "schema.json")
				if err := os.WriteFile(schemaPath, []byte(`{"type": "object"}`), 0644); err != nil {
					t.Fatal(err)
				}
				tt.config.Schemas[0].Path = schemaPath
			}

			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaConfig_Validate(t *testing.T) {
	tempDir := t.TempDir()
	validSchemaPath := filepath.Join(tempDir, "schema.json")
	if err := os.WriteFile(validSchemaPath, []byte(`{"type": "object"}`), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		schema  *SchemaConfig
		wantErr bool
	}{
		{
			name: "valid schema config",
			schema: &SchemaConfig{
				Path:      validSchemaPath,
				Documents: []string{"test.json"},
			},
			wantErr: false,
		},
		{
			name: "empty path",
			schema: &SchemaConfig{
				Path:      "",
				Documents: []string{"test.json"},
			},
			wantErr: true,
		},
		{
			name: "no documents",
			schema: &SchemaConfig{
				Path:      validSchemaPath,
				Documents: []string{},
			},
			wantErr: true,
		},
		{
			name: "non-existent schema file",
			schema: &SchemaConfig{
				Path:      "/nonexistent/schema.json",
				Documents: []string{"test.json"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SchemaConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaConfig_GetEffectiveSchemaVersion(t *testing.T) {
	tests := []struct {
		name          string
		schemaVersion string
		globalVersion string
		want          string
	}{
		{
			name:          "schema level takes precedence",
			schemaVersion: "draft/2019-09",
			globalVersion: "draft/2020-12",
			want:          "draft/2019-09",
		},
		{
			name:          "falls back to global",
			schemaVersion: "",
			globalVersion: "draft/2020-12",
			want:          "draft/2020-12",
		},
		{
			name:          "both empty",
			schemaVersion: "",
			globalVersion: "",
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SchemaConfig{
				SchemaVersion: tt.schemaVersion,
			}
			if got := s.GetEffectiveSchemaVersion(tt.globalVersion); got != tt.want {
				t.Errorf("GetEffectiveSchemaVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchemaConfig_GetEffectiveErrorTemplate(t *testing.T) {
	tests := []struct {
		name           string
		schemaTemplate string
		globalTemplate string
		want           string
	}{
		{
			name:           "schema level takes precedence",
			schemaTemplate: "schema template",
			globalTemplate: "global template",
			want:           "schema template",
		},
		{
			name:           "falls back to global",
			schemaTemplate: "",
			globalTemplate: "global template",
			want:           "global template",
		},
		{
			name:           "both empty",
			schemaTemplate: "",
			globalTemplate: "",
			want:           "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SchemaConfig{
				ErrorTemplate: tt.schemaTemplate,
			}
			if got := s.GetEffectiveErrorTemplate(tt.globalTemplate); got != tt.want {
				t.Errorf("GetEffectiveErrorTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchemaConfig_GetEffectiveForceFiletype(t *testing.T) {
	tests := []struct {
		name            string
		configForceType string
		flagForceType   string
		want            string
	}{
		{
			name:            "flag takes precedence",
			configForceType: "json",
			flagForceType:   "yaml",
			want:            "yaml",
		},
		{
			name:            "falls back to config",
			configForceType: "json5",
			flagForceType:   "",
			want:            "json5",
		},
		{
			name:            "both empty",
			configForceType: "",
			flagForceType:   "",
			want:            "",
		},
		{
			name:            "flag only",
			configForceType: "",
			flagForceType:   "toml",
			want:            "toml",
		},
		{
			name:            "config only",
			configForceType: "yaml",
			flagForceType:   "",
			want:            "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SchemaConfig{
				ForceFiletype: tt.configForceType,
			}
			if got := s.GetEffectiveForceFiletype(tt.flagForceType); got != tt.want {
				t.Errorf("GetEffectiveForceFiletype() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeRefOverrides(t *testing.T) {
	tests := []struct {
		name    string
		sources []map[string]string
		want    map[string]string
	}{
		{
			name:    "empty sources",
			sources: []map[string]string{},
			want:    map[string]string{},
		},
		{
			name: "single source",
			sources: []map[string]string{
				{"url1": "path1", "url2": "path2"},
			},
			want: map[string]string{"url1": "path1", "url2": "path2"},
		},
		{
			name: "multiple sources with override",
			sources: []map[string]string{
				{"url1": "path1", "url2": "path2"},
				{"url2": "new_path2", "url3": "path3"},
			},
			want: map[string]string{
				"url1": "path1",
				"url2": "new_path2", // Overridden
				"url3": "path3",
			},
		},
		{
			name: "three sources with priority",
			sources: []map[string]string{
				{"url1": "default"},
				{"url1": "config"},
				{"url1": "cli"},
			},
			want: map[string]string{"url1": "cli"}, // CLI wins
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeRefOverrides(tt.sources...)
			if len(got) != len(tt.want) {
				t.Errorf("MergeRefOverrides() got %d items, want %d", len(got), len(tt.want))
			}
			for key, wantVal := range tt.want {
				if gotVal, ok := got[key]; !ok {
					t.Errorf("MergeRefOverrides() missing key %q", key)
				} else if gotVal != wantVal {
					t.Errorf("MergeRefOverrides()[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestSchemaConfig_ExpandDocumentGlobs(t *testing.T) {
	// Create test files
	tempDir := t.TempDir()
	files := []string{
		filepath.Join(tempDir, "config.json"),
		filepath.Join(tempDir, "config.dev.json"),
		filepath.Join(tempDir, "config.prod.json"),
		filepath.Join(tempDir, "data.json"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name      string
		documents []string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "no glob - single file",
			documents: []string{filepath.Join(tempDir, "config.json")},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "glob pattern",
			documents: []string{filepath.Join(tempDir, "config.*.json")},
			wantCount: 2, // config.dev.json, config.prod.json
			wantErr:   false,
		},
		{
			name: "mixed glob and non-glob",
			documents: []string{
				filepath.Join(tempDir, "data.json"),
				filepath.Join(tempDir, "config.*.json"),
			},
			wantCount: 3, // data.json + 2 matched files
			wantErr:   false,
		},
		{
			name:      "no matches",
			documents: []string{filepath.Join(tempDir, "nonexistent.*.json")},
			wantCount: 0, // No error, just no matches
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SchemaConfig{
				Documents: tt.documents,
			}
			got, err := s.ExpandDocumentGlobs()
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandDocumentGlobs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("ExpandDocumentGlobs() got %d files, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestContainsGlobChars(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"no glob chars", "config.json", false},
		{"asterisk", "config.*.json", true},
		{"question mark", "config?.json", true},
		{"bracket", "config[123].json", true},
		{"multiple glob chars", "**/*.json", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsGlobChars(tt.s); got != tt.want {
				t.Errorf("containsGlobChars(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg == nil {
		t.Fatal("NewConfig() returned nil")
	}
	if cfg.SchemaVersion != "" {
		t.Errorf("NewConfig().SchemaVersion = %q, want empty", cfg.SchemaVersion)
	}
	if cfg.Schemas == nil {
		t.Error("NewConfig().Schemas is nil, want empty slice")
	}
	if cfg.ErrorTemplate != "" {
		t.Errorf("NewConfig().ErrorTemplate = %q, want empty", cfg.ErrorTemplate)
	}
}

func TestNewSchemaConfig(t *testing.T) {
	schema := NewSchemaConfig("schema.json", "doc1.json", "doc2.json")
	if schema == nil {
		t.Fatal("NewSchemaConfig() returned nil")
	}
	if schema.Path != "schema.json" {
		t.Errorf("NewSchemaConfig().Path = %q, want %q", schema.Path, "schema.json")
	}
	if len(schema.Documents) != 2 {
		t.Errorf("NewSchemaConfig().Documents has %d items, want 2", len(schema.Documents))
	}
	if schema.RefOverrides == nil {
		t.Error("NewSchemaConfig().RefOverrides is nil, want empty map")
	}
}

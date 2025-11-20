package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

// Custom unmarshaler for handling both camelCase (JSON) and snake_case (YAML/TOML)
var unmarshalConf = koanf.UnmarshalConf{
	Tag: "koanf",
	// Use flatMap to handle nested structures
	FlatPaths: false,
}

// Loader handles configuration loading from multiple sources
type Loader struct {
	k         *koanf.Koanf
	envPrefix string
}

// NewLoader creates a new configuration loader with default environment prefix
func NewLoader() *Loader {
	return &Loader{
		k:         koanf.New("."),
		envPrefix: "JSONSCHEMA_VALIDATOR_",
	}
}

// SetEnvPrefix sets a custom environment variable prefix
// The prefix should end with an underscore (e.g., "MY_APP_")
func (l *Loader) SetEnvPrefix(prefix string) {
	l.envPrefix = prefix
}

// Load loads configuration from all available sources in priority order:
// 1. Command-line flags (highest priority)
// 2. Environment variables (customizable prefix, default: JSONSCHEMA_VALIDATOR_*)
// 3. .jsonschema-validator.yaml in current directory
// 4. pyproject.toml section [tool.jsonschema-validator]
// 5. Default values (lowest priority)
func (l *Loader) Load(flags *flag.FlagSet) (*Config, error) {
	// 1. Load defaults (lowest priority)
	if err := l.loadDefaults(); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// 2. Load from pyproject.toml (if exists)
	if err := l.loadPyprojectTOML(); err != nil {
		// Optional, don't fail if not found
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading pyproject.toml config: %w", err)
		}
	}

	// 3. Load from .jsonschema-validator.yaml (if exists)
	if err := l.loadProjectConfig(); err != nil {
		// Optional, don't fail if not found
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading project config: %w", err)
		}
	}

	// 4. Load from environment variables
	if err := l.loadEnvVars(); err != nil {
		return nil, fmt.Errorf("loading environment variables: %w", err)
	}

	// 5. Load from command-line flags (highest priority)
	if flags != nil {
		if err := l.loadFlags(flags); err != nil {
			return nil, fmt.Errorf("loading flags: %w", err)
		}
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := l.k.UnmarshalWithConf("", &cfg, unmarshalConf); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// LoadFromFile loads configuration from a specific file
func (l *Loader) LoadFromFile(path string) (*Config, error) {
	// Load defaults first
	if err := l.loadDefaults(); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// Determine parser based on file extension
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		if err := l.k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file %q: %w", path, err)
		}
	case ".toml":
		if err := l.k.Load(file.Provider(path), toml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file %q: %w", path, err)
		}
	case ".json":
		if err := l.k.Load(file.Provider(path), json.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file %q: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := l.k.UnmarshalWithConf("", &cfg, unmarshalConf); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// loadDefaults loads default configuration values
func (l *Loader) loadDefaults() error {
	defaults := map[string]interface{}{
		"schema_version": "", // Empty means use schema's $schema field
		"schemas":        []interface{}{},
		"error_template": "", // Empty means use default formatting
	}

	return l.k.Load(confmap.Provider(defaults, "."), nil)
}

// loadProjectConfig loads configuration from .jsonschema-validator.yaml in current directory
func (l *Loader) loadProjectConfig() error {
	// Try multiple file names in order of preference
	candidates := []string{
		".jsonschema-validator.yaml",
		".jsonschema-validator.yml",
		".jsonschema.yaml",
		".jsonschema.yml",
		"jsonschema-validator.yaml",
		"jsonschema-validator.yml",
	}

	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return l.k.Load(file.Provider(name), yaml.Parser())
		}
	}

	return os.ErrNotExist
}

// loadPyprojectTOML loads configuration from pyproject.toml [tool.jsonschema-validator]
func (l *Loader) loadPyprojectTOML() error {
	const configFile = "pyproject.toml"

	if _, err := os.Stat(configFile); err != nil {
		return err
	}

	// Load the entire TOML file
	tempK := koanf.New(".")
	if err := tempK.Load(file.Provider(configFile), toml.Parser()); err != nil {
		return err
	}

	// Extract only the [tool.jsonschema-validator] section
	toolConfig := tempK.Cut("tool.jsonschema-validator")
	if toolConfig == nil || toolConfig.Raw() == nil {
		// No jsonschema-validator section found
		return os.ErrNotExist
	}

	// Merge the tool section into main config
	return l.k.Merge(toolConfig)
}

// loadEnvVars loads configuration from environment variables
// Environment variables use SCREAMING_SNAKE_CASE (no camelCase!)
// Prefixed with the configured prefix (default: JSONSCHEMA_VALIDATOR_)
// Examples (with default prefix):
//
//	JSONSCHEMA_VALIDATOR_SCHEMA_VERSION=draft/2020-12
//	JSONSCHEMA_VALIDATOR_ERROR_TEMPLATE="..."
func (l *Loader) loadEnvVars() error {
	return l.k.Load(env.Provider(l.envPrefix, ".", func(s string) string {
		// Convert JSONSCHEMA_VALIDATOR_SCHEMA_VERSION to schema_version
		// Just remove prefix and lowercase - underscores stay as-is
		s = strings.TrimPrefix(s, l.envPrefix)
		s = strings.ToLower(s)
		return s
	}), nil)
}

// loadFlags loads configuration from command-line flags
func (l *Loader) loadFlags(flags *flag.FlagSet) error {
	return l.k.Load(posflag.Provider(flags, ".", l.k), nil)
}

// ParseRefOverridesFromString parses ref_override from string format
// Format: "url1=path1,url2=path2" or repeated --ref-override flags
func ParseRefOverridesFromString(s string) map[string]string {
	overrides := make(map[string]string)

	if s == "" {
		return overrides
	}

	// Split by comma
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		// Split by equals sign
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			url := strings.TrimSpace(parts[0])
			path := strings.TrimSpace(parts[1])
			if url != "" && path != "" {
				overrides[url] = path
			}
		}
	}

	return overrides
}

// ParseRefOverridesFromSlice parses ref_override from slice format
// Each element is in format "url=path"
func ParseRefOverridesFromSlice(slice []string) map[string]string {
	overrides := make(map[string]string)

	for _, item := range slice {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			url := strings.TrimSpace(parts[0])
			path := strings.TrimSpace(parts[1])
			if url != "" && path != "" {
				overrides[url] = path
			}
		}
	}

	return overrides
}

// GetKoanf returns the underlying koanf instance for advanced usage
func (l *Loader) GetKoanf() *koanf.Koanf {
	return l.k
}

package provider

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
    "encoding/json"
	"github.com/google/go-cmp/cmp"
)

func TestRefResolver_ResolveRefs(t *testing.T) {
    // Create temporary test files
    tmpDir := t.TempDir()
    t.Logf("Using temp dir: %s", tmpDir)
    
    // Create test schema files
    allowedPathLevel1 := filepath.Join(tmpDir, "schemas", "allowed1.json")
    allowedPathLevel2 := filepath.Join(tmpDir, "schemas", "allowed2.json")
    deniedPath := filepath.Join(tmpDir, "other",  "denied.json")
    mainSchemaPath := filepath.Join(tmpDir, "main.json")
    
    
    // Ensure directories exist
    if err := os.MkdirAll(filepath.Dir(allowedPathLevel1), 0755); err != nil {
        t.Fatal(err)
    }
    if err := os.MkdirAll(filepath.Dir(deniedPath), 0755); err != nil {
        t.Fatal(err)
    }
    
    // Write test files
    if err := os.WriteFile(allowedPathLevel1, []byte(`{"type": "object",  "properties": { "nested": {"$ref": "./allowed2.json"}} }`), 0644); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(allowedPathLevel2, []byte(`{"type": "object",  "properties": { "leaf": {"type": "string"}} }`), 0644); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(deniedPath, []byte(`{"type": "number"}`), 0644); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(mainSchemaPath, []byte(`{"type": "object"}`), 0644); err != nil {
        t.Fatal(err)
    }

    tests := []struct {
        name          string
        patterns      []string
        schema        map[string]interface{}
        basePath      string
        wantErr       bool
        errSubstring  string
        wantType      string // expected type after resolution
        wantLiteral   string // expected literal value after resolution (if applicable)
    }{
        {
            name:     "denied ref path returns error",
            patterns: []string{filepath.Join(tmpDir, "schemas/**/*.json")},
            schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "field": map[string]interface{}{
                        "$ref": deniedPath, // Remove file:// scheme
                    },
                },
            },
            basePath:     mainSchemaPath,
            wantErr:      true,
            errSubstring: "not allowed",  
        },
        {
            name:     "absolute ref resolves when pattern allows",
            patterns: []string{"./schemas/*.json"}, 
            schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "field": map[string]interface{}{
                        "$ref": allowedPathLevel1,  // Using absolute path
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            wantType: "object",
            wantLiteral: `{"properties":{"field":{"properties":{"nested":{"properties":{"leaf":{"type":"string"}},"type":"object"}},"type":"object"}},"type":"object"}`,
        },
        {
            name:     "relative ref resolves when pattern allows",
            patterns: []string{tmpDir + "/**/*.json"}, // Allow all under tmpDir
            schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "field": map[string]interface{}{
                        "$ref": filepath.Join(filepath.Dir(mainSchemaPath), "schemas/allowed1.json"),
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            wantType: "object",
            wantLiteral: `{"properties":{"field":{"properties":{"nested":{"properties":{"leaf":{"type":"string"}},"type":"object"}},"type":"object"}},"type":"object"}`,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resolver, err := NewRefResolver(tt.patterns, tt.basePath)
            if err != nil {
                t.Fatalf("NewRefResolver() error = %v", err)
            }

            t.Logf("Test %q: Using patterns %v", tt.name, tt.patterns)
            t.Logf("Test %q: Resolving schema with basePath %v", tt.name, tt.basePath)

            // Call ResolveRefs since that's the public API
            t.Logf("Schema before resolution: %#v", tt.schema)
            got, err := resolver.ResolveRefs(tt.schema)
            if err != nil {
                t.Logf("ResolveRefs returned error: %v", err)
            }
            t.Logf("Schema after resolution: %#v", got)

            if (err != nil) != tt.wantErr {
                t.Errorf("ResolveRefs() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if tt.wantErr && tt.errSubstring != "" && err != nil {
                if !strings.Contains(err.Error(), tt.errSubstring) {
                    t.Errorf("error %q should contain %q", err.Error(), tt.errSubstring)
                }
                return
            }
            if !tt.wantErr {
                // Verify the resolved schema contains the referenced content
                props, ok := got.(map[string]interface{})["properties"]
                if !ok {
                    t.Errorf("resolved schema missing 'properties': %#v", got)
                    return
                }
                propsMap, ok := props.(map[string]interface{})
                if !ok {
                    t.Errorf("'properties' is not a map: %#v", props)
                    return
                }
                field, ok := propsMap["field"].(map[string]interface{})
                if !ok {
                    t.Errorf("'field' is not a map: %#v", propsMap["field"])
                    return
                }
                if fieldType, ok := field["type"].(string); !ok || fieldType != tt.wantType {
                    t.Errorf("resolved ref should have type=%v, got %v", tt.wantType, field["type"])
                }
            }
            if tt.wantLiteral != "" {
                gotLiteralBytes, err := json.Marshal(got)
                if err != nil {
                    t.Errorf("failed to marshal resolved schema: %v", err)
                    return
                }
                if diff := cmp.Diff(tt.wantLiteral, string(gotLiteralBytes)); diff != "" {
                    t.Errorf("resolved schema mismatch (-want +got):\n%s", diff)
                }
            }
        })
    }
}

func TestRefResolver_HappyPathResolution(t *testing.T) {
    tmpDir := t.TempDir()

    // Setup file structure
    schemaDir := filepath.Join(tmpDir, "schemas")
    if err := os.MkdirAll(schemaDir, 0755); err != nil {
        t.Fatal(err)
    }

    allowedPathLevel1 := filepath.Join(schemaDir, "allowed.json")

    // Write allowed ref file
    if err := os.WriteFile(allowedPathLevel1, []byte(`{
        "type": "string",
        "enum": ["one", "two", "three"]
    }`), 0644); err != nil {
        t.Fatal(err)
    }

    originalSchema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "field": map[string]interface{}{
                "$ref": allowedPathLevel1,
            },
        },
    }
    expectedSchema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "field": map[string]interface{}{
                "type": "string",
                "enum": []interface{}{"one", "two", "three"},
            },
        },
    }

    patterns := []string{schemaDir + "/*.json"}
    resolver, err := NewRefResolver(patterns, allowedPathLevel1)

    resolved, err := resolver.ResolveRefs(originalSchema)
    if err != nil {
        t.Fatalf("ResolveRefs() error = %v", err)
    }


	if diff := cmp.Diff(expectedSchema, resolved); diff != "" {
		t.Errorf("resolved schema mismatch (-want +got):\n%s", diff)
	}


}

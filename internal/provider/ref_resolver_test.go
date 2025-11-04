package provider

import (
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "testing"
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
    
    // Create a file with definitions for fragment testing
    defsPath := filepath.Join(tmpDir, "schemas", "definitions.json")
    if err := os.WriteFile(defsPath, []byte(`{
        "definitions": {
            "StringType": {
                "type": "string",
                "minLength": 1
            },
            "NumberType": {
                "type": "number",
                "minimum": 0
            }
        }
    }`), 0644); err != nil {
        t.Fatal(err)
    }

    tests := []struct {
        name          string
        patterns      []string
        schema        map[string]interface{}
        basePath      string
        wantErr       bool
        errSubstring  string
        wantType      string      // expected type after resolution
        want          interface{} // expected structure after resolution
    }{
        {
            name:     "denied ref path returns error",
            patterns: []string{"./schemas/**/*.json"},
            schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "field": map[string]interface{}{
                        "$ref": "./other/denied.json",
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
                        "$ref": "./schemas/allowed1.json",
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            wantType: "object",
            want: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "field": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "nested": map[string]interface{}{
                                "type": "object",
                                "properties": map[string]interface{}{
                                    "leaf": map[string]interface{}{
                                        "type": "string",
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
        {
            name:     "relative ref resolves when pattern allows",
            patterns: []string{"./schemas/*.json"}, 
            schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "field": map[string]interface{}{
                        "$ref": "./schemas/allowed1.json",
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            wantType: "object",
            want: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "field": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "nested": map[string]interface{}{
                                "type": "object",
                                "properties": map[string]interface{}{
                                    "leaf": map[string]interface{}{
                                        "type": "string",
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
        {
            name:     "fragment ref resolves when pattern allows",
            patterns: []string{"./schemas/*.json"}, 
            schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "stringField": map[string]interface{}{
                        "$ref": "./schemas/definitions.json#definitions/StringType",
                    },
                    "numberField": map[string]interface{}{
                        "$ref": "./schemas/definitions.json#definitions/NumberType",
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            wantType: "",
            want: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "stringField": map[string]interface{}{
                        "type":      "string",
                        "minLength": float64(1),
                    },
                    "numberField": map[string]interface{}{
                        "type":    "number",
                        "minimum": float64(0),
                    },
                },
            },
        },
        {
            name:     "fragment ref with leading slash resolves when pattern allows",
            patterns: []string{"./schemas/*.json"}, 
            schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "stringField": map[string]interface{}{
                        "$ref": "./schemas/definitions.json#/definitions/StringType",
                    },
                    "numberField": map[string]interface{}{
                        "$ref": "./schemas/definitions.json#/definitions/NumberType",
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            wantType: "",
            want: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "stringField": map[string]interface{}{
                        "type":      "string",
                        "minLength": float64(1),
                    },
                    "numberField": map[string]interface{}{
                        "type":    "number",
                        "minimum": float64(0),
                    },
                },
            },
        },
        {
            name:     "fragment-only ref resolves against root document",
            patterns: []string{"./schemas/*.json"},
            schema: map[string]interface{}{
                "definitions": map[string]interface{}{
                    "Address": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "street": map[string]interface{}{"type": "string"},
                            "city":   map[string]interface{}{"type": "string"},
                        },
                    },
                    "Person": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "name": map[string]interface{}{"type": "string"},
                            "address": map[string]interface{}{
                                "$ref": "#/definitions/Address",
                            },
                        },
                    },
                },
                "type": "object",
                "properties": map[string]interface{}{
                    "user": map[string]interface{}{
                        "$ref": "#/definitions/Person",
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            want: map[string]interface{}{
                "definitions": map[string]interface{}{
                    "Address": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "street": map[string]interface{}{"type": "string"},
                            "city":   map[string]interface{}{"type": "string"},
                        },
                    },
                    "Person": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "name": map[string]interface{}{"type": "string"},
                            "address": map[string]interface{}{
                                "type": "object",
                                "properties": map[string]interface{}{
                                    "street": map[string]interface{}{"type": "string"},
                                    "city":   map[string]interface{}{"type": "string"},
                                },
                            },
                        },
                    },
                },
                "type": "object",
                "properties": map[string]interface{}{
                    "user": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "name": map[string]interface{}{"type": "string"},
                            "address": map[string]interface{}{
                                "type": "object",
                                "properties": map[string]interface{}{
                                    "street": map[string]interface{}{"type": "string"},
                                    "city":   map[string]interface{}{"type": "string"},
                                },
                            },
                        },
                    },
                },
            },
        },
        {
            name:     "fragment-only ref without leading slash resolves against root document",
            patterns: []string{"./schemas/*.json"},
            schema: map[string]interface{}{
                "definitions": map[string]interface{}{
                    "Color": map[string]interface{}{
                        "type": "string",
                        "enum": []interface{}{"red", "green", "blue"},
                    },
                },
                "type": "object",
                "properties": map[string]interface{}{
                    "favoriteColor": map[string]interface{}{
                        "$ref": "#definitions/Color",
                    },
                },
            },
            basePath: mainSchemaPath,
            wantErr:  false,
            want: map[string]interface{}{
                "definitions": map[string]interface{}{
                    "Color": map[string]interface{}{
                        "type": "string",
                        "enum": []interface{}{"red", "green", "blue"},
                    },
                },
                "type": "object",
                "properties": map[string]interface{}{
                    "favoriteColor": map[string]interface{}{
                        "type": "string",
                        "enum": []interface{}{"red", "green", "blue"},
                    },
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resolver, err := NewRefResolver(tt.patterns, filepath.Dir(tt.basePath))
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
                if tt.wantType != "" {
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
            }
            if tt.want != nil {
                // Compare structures directly
                if diff := cmp.Diff(tt.want, got); diff != "" {
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
                "$ref": "./schemas/allowed.json",
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

    // Use tmpDir as base directory and adjust pattern accordingly
    patterns := []string{"./schemas/*.json"}  // Add ./ to match our pattern normalization
    resolver, err := NewRefResolver(patterns, tmpDir)
    if err != nil {
        t.Fatal(err)
    }

    resolved, err := resolver.ResolveRefs(originalSchema)
    if err != nil {
        t.Fatalf("ResolveRefs() error = %v", err)
    }


	if diff := cmp.Diff(expectedSchema, resolved); diff != "" {
		t.Errorf("resolved schema mismatch (-want +got):\n%s", diff)
	}
}

func TestRefResolver_WindowsAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup file structure
	schemaDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}

	allowedPath := filepath.Join(schemaDir, "windows-test.json")

	// Write test file
	if err := os.WriteFile(allowedPath, []byte(`{
		"type": "string",
		"pattern": "^[A-Z]:"
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Test that filepath.VolumeName detects Windows-style paths
	// This tests our Windows path detection logic
	testCases := []struct {
		name        string
		ref         string
		shouldParse bool // whether url.Parse should be called
	}{
		{
			name:        "Unix absolute path",
			ref:         allowedPath,
			shouldParse: false, // filepath.IsAbs catches this
		},
	}

	// Add Windows-specific test case only on Windows
	if runtime.GOOS == "windows" {
		testCases = append(testCases, struct {
			name        string
			ref         string
			shouldParse bool
		}{
			name:        "Windows drive letter path",
			ref:         allowedPath, // Will be like C:\Users\...
			shouldParse: false,       // filepath.VolumeName catches this
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalSchema := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"field": map[string]interface{}{
						"$ref": tc.ref,
					},
				},
			}
			expectedSchema := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"field": map[string]interface{}{
						"type":    "string",
						"pattern": "^[A-Z]:",
					},
				},
			}

			patterns := []string{"./schemas/*.json"}
			resolver, err := NewRefResolver(patterns, tmpDir)
			if err != nil {
				t.Fatal(err)
			}

			resolved, err := resolver.ResolveRefs(originalSchema)
			if err != nil {
				t.Fatalf("ResolveRefs() error = %v", err)
			}

			if diff := cmp.Diff(expectedSchema, resolved); diff != "" {
				t.Errorf("resolved schema mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRefResolver_WindowsPathDetection(t *testing.T) {
	// Test that our Windows path detection logic works correctly
	tests := []struct {
		path          string
		isAbsolute    bool
		hasVolume     bool
		shouldSkipURL bool
	}{
		{"/unix/absolute/path", true, false, true},
		{"./relative/path", false, false, false},
		{"relative/path", false, false, false},
	}

	// Add Windows-specific test cases
	if runtime.GOOS == "windows" {
		tests = append(tests,
			struct {
				path          string
				isAbsolute    bool
				hasVolume     bool
				shouldSkipURL bool
			}{"C:\\Windows\\System32", true, true, true},
			struct {
				path          string
				isAbsolute    bool
				hasVolume     bool
				shouldSkipURL bool
			}{"D:\\data\\file.json", true, true, true},
			struct {
				path          string
				isAbsolute    bool
				hasVolume     bool
				shouldSkipURL bool
			}{"C:relative", false, true, true}, // Has volume but not absolute
		)
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			isAbs := filepath.IsAbs(tt.path)
			hasVol := filepath.VolumeName(tt.path) != ""
			shouldSkipURL := isAbs || hasVol

			if isAbs != tt.isAbsolute {
				t.Errorf("filepath.IsAbs(%q) = %v, want %v", tt.path, isAbs, tt.isAbsolute)
			}
			if hasVol != tt.hasVolume {
				t.Errorf("filepath.VolumeName(%q) != \"\" is %v, want %v", tt.path, hasVol, tt.hasVolume)
			}
			if shouldSkipURL != tt.shouldSkipURL {
				t.Errorf("should skip URL parsing for %q: %v, want %v", tt.path, shouldSkipURL, tt.shouldSkipURL)
			}
		})
	}
}

func TestRefResolver_CachingWithMultipleFragments(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup file structure
	schemaDir := filepath.Join(tmpDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a definitions file with multiple definitions
	defsPath := filepath.Join(schemaDir, "common.json")
	if err := os.WriteFile(defsPath, []byte(`{
		"definitions": {
			"Email": {
				"type": "string",
				"format": "email"
			},
			"URL": {
				"type": "string",
				"format": "uri"
			},
			"Age": {
				"type": "integer",
				"minimum": 0,
				"maximum": 150
			}
		}
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Schema that references different fragments from the same file
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"email": map[string]interface{}{
				"$ref": "./schemas/common.json#/definitions/Email",
			},
			"website": map[string]interface{}{
				"$ref": "./schemas/common.json#/definitions/URL",
			},
			"age": map[string]interface{}{
				"$ref": "./schemas/common.json#/definitions/Age",
			},
			// Reference the same fragment again to test cache hit
			"alternateEmail": map[string]interface{}{
				"$ref": "./schemas/common.json#/definitions/Email",
			},
		},
	}

	expectedSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"email": map[string]interface{}{
				"type":   "string",
				"format": "email",
			},
			"website": map[string]interface{}{
				"type":   "string",
				"format": "uri",
			},
			"age": map[string]interface{}{
				"type":    "integer",
				"minimum": float64(0),
				"maximum": float64(150),
			},
			"alternateEmail": map[string]interface{}{
				"type":   "string",
				"format": "email",
			},
		},
	}

	patterns := []string{"./schemas/*.json"}
	resolver, err := NewRefResolver(patterns, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := resolver.ResolveRefs(schema)
	if err != nil {
		t.Fatalf("ResolveRefs() error = %v", err)
	}

	if diff := cmp.Diff(expectedSchema, resolved); diff != "" {
		t.Errorf("resolved schema mismatch (-want +got):\n%s", diff)
	}

	// Verify caching: the file should be loaded once, but multiple cache entries for different fragments
	resolver.mu.RLock()
	loadedFilesCount := len(resolver.loadedFiles)
	loadedRefsCount := len(resolver.loadedRefs)
	resolver.mu.RUnlock()

	// Should have loaded exactly 1 file (common.json)
	if loadedFilesCount != 1 {
		t.Errorf("expected 1 loaded file, got %d", loadedFilesCount)
	}

	// Should have cached 3 distinct refs (Email, URL, Age) - the duplicate Email ref should hit cache
	if loadedRefsCount != 3 {
		t.Errorf("expected 3 cached refs (Email, URL, Age), got %d", loadedRefsCount)
	}
}


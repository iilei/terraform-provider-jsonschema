package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"encoding/json"
    "sort"
	"github.com/titanous/json5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/xeipuuv/gojsonschema"
)
func canonicalizeJSON(input string) (string, error) {
    var decoded interface{}
    if err := json.Unmarshal([]byte(input), &decoded); err != nil {
        return "", err
    }
    
    // Marshal with sorted keys
    canonical, err := json.Marshal(sortKeys(decoded))
    if err != nil {
        return "", err
    }
    
    return string(canonical), nil
}

func sortKeys(i interface{}) interface{} {
    switch x := i.(type) {
    case map[string]interface{}:
        // Get all keys
        keys := make([]string, 0, len(x))
        for k := range x {
            keys = append(keys, k)
        }
        sort.Strings(keys)
        
        // Create new sorted map
        m2 := make(map[string]interface{})
        for _, k := range keys {
            m2[k] = sortKeys(x[k])
        }
        return m2
    case []interface{}:
        for i, v := range x {
            x[i] = sortKeys(v)
        }
    }
    return i
}

func dataSourceJsonschemaValidator() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceJsonschemaValidatorRead,

		Schema: map[string]*schema.Schema{
			"document": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "body of a yaml or json file",
			},

			"schema": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "json schema to validate content by",
			},

			"validated": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceJsonschemaValidatorRead(d *schema.ResourceData, m interface{}) error {
    var (
        err    error = nil
        result       = new(gojsonschema.Result)
    )

	document := d.Get("document").(string)
	schemaJson5 := d.Get("schema").(string)

	// Convert JSON5 schema to regular JSON
	var schemaData interface{}
	if err := json5.Unmarshal([]byte(schemaJson5), &schemaData); err != nil {
		return fmt.Errorf("invalid JSON5 schema: %v", err)
	}

	// Get ref patterns from provider config
	var patterns []string
	if m != nil {
		if p, ok := m.(map[string]interface{})["ref_patterns"]; ok {
			for _, pattern := range p.(*schema.Set).List() {
				patterns = append(patterns, pattern.(string))
			}
		}
	}

	// Create resolver
	resolver, err := NewRefResolver(patterns)
	if err != nil {
		return fmt.Errorf("failed to create ref resolver: %v", err)
	}

	// Resolve refs in schema
	resolvedSchema, err := resolver.ResolveRefs(schemaData)
	if err != nil {
		return fmt.Errorf("failed to resolve refs: %v", err)
	}

	// Convert back to JSON string
	schemaJson, err := json.Marshal(resolvedSchema)
	if err != nil {
		return fmt.Errorf("error converting schema to JSON: %v", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaJson))
	documentLoader := gojsonschema.NewStringLoader(document)

	// Validate
	result, err = gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("error during validation: %v", err)
	}

	if result.Valid() {
		if err := d.Set("validated", document); err != nil {
			return fmt.Errorf("failed to set validated document: %v", err)
		}
	} else {
		// Build error message
		message := "The document is not valid. See errors:\n"

		// Parse schema for descriptions safely
		var schemaMap map[string]interface{}
		if schemaBytes, e := schemaLoader.LoadJSON(); e == nil {
			if sm, ok := schemaBytes.(map[string]interface{}); ok {
				schemaMap = sm
			}
		}

		lookupDescription := func(schema map[string]interface{}, field string) string {
			if schema == nil {
				return ""
			}
			props, ok := schema["properties"].(map[string]interface{})
			if !ok {
				return ""
			}
			prop, ok := props[field].(map[string]interface{})
			if !ok {
				return ""
			}
			if d, ok := prop["description"].(string); ok {
				return d
			}
			return ""
		}

		for _, verr := range result.Errors() {
			field := verr.Field()
			desc := lookupDescription(schemaMap, field)
			message += fmt.Sprintf("- %s: %s", field, verr.Description())
			if desc != "" {
				message += fmt.Sprintf(" -- %s", desc)
			}
			message += "\n"
		}

		return errors.New(message)
	}

	// Canonicalize document
	canonicalDoc, err := canonicalizeJSON(document)
	if err != nil {
		return fmt.Errorf("error canonicalizing document: %v", err)
	}

	// Use already marshaled schema
	compositeString := fmt.Sprintf("%s:%s", canonicalDoc, schemaJson)
	d.SetId(hash(compositeString))

	return nil
}


func hash(s string) string {
	sha := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sha[:])
}

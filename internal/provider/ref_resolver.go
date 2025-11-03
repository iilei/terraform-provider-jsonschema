package provider

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/url"
    "path/filepath"

    "github.com/gobwas/glob"
)

type RefResolver struct {
    allowedPatterns []glob.Glob
    loadedRefs     map[string]interface{}
}

func NewRefResolver(patterns []string) (*RefResolver, error) {
    globs := make([]glob.Glob, 0, len(patterns))
    for _, pattern := range patterns {
        g, err := glob.Compile(pattern)
        if err != nil {
            return nil, fmt.Errorf("invalid glob pattern %q: %v", pattern, err)
        }
        globs = append(globs, g)
    }
    return &RefResolver{
        allowedPatterns: globs,
        loadedRefs:     make(map[string]interface{}),
    }, nil
}

func (r *RefResolver) ResolveRefs(schema interface{}) (interface{}, error) {
    return r.resolveRefsRecursive(schema, "")
}

func (r *RefResolver) resolveRefsRecursive(v interface{}, basePath string) (interface{}, error) {
    switch x := v.(type) {
    case map[string]interface{}:
        if ref, ok := x["$ref"].(string); ok {
            return r.resolveRef(ref, basePath)
        }
        
        result := make(map[string]interface{})
        for k, v := range x {
            resolved, err := r.resolveRefsRecursive(v, basePath)
            if err != nil {
                return nil, err
            }
            result[k] = resolved
        }
        return result, nil
        
    case []interface{}:
        result := make([]interface{}, len(x))
        for i, v := range x {
            resolved, err := r.resolveRefsRecursive(v, basePath)
            if err != nil {
                return nil, err
            }
            result[i] = resolved
        }
        return result, nil
    }
    return v, nil
}

func (r *RefResolver) resolveRef(ref string, basePath string) (interface{}, error) {
    if cached, ok := r.loadedRefs[ref]; ok {
        return cached, nil
    }

    u, err := url.Parse(ref)
    if err != nil {
        return nil, fmt.Errorf("invalid $ref URL: %v", err)
    }

    // Handle only file references for now
    if u.Scheme != "" && u.Scheme != "file" {
        return nil, fmt.Errorf("only file:// and relative refs are supported")
    }

    path := u.Path
    if !filepath.IsAbs(path) {
        path = filepath.Join(filepath.Dir(basePath), path)
    }

    // Check if path is allowed
    allowed := false
    for _, pattern := range r.allowedPatterns {
        if pattern.Match(path) {
            allowed = true
            break
        }
    }
    if !allowed {
        return nil, fmt.Errorf("$ref path %q not allowed", path)
    }

    // Load and parse referenced file
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read $ref file: %v", err)
    }

    var parsed interface{}
    if err := json.Unmarshal(data, &parsed); err != nil {
        return nil, fmt.Errorf("failed to parse $ref file: %v", err)
    }

    // Cache the result
    r.loadedRefs[ref] = parsed
    return parsed, nil
}
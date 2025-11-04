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
    basePath       string
}

func NewRefResolver(patterns []string, basePath string) (*RefResolver, error) {
    globs := make([]glob.Glob, 0, len(patterns))
    for _, pattern := range patterns {
        // Clean and normalize the pattern
        cleanPattern := filepath.Clean(pattern)
        
        // Store whether the pattern was absolute for later use
        isAbs := filepath.IsAbs(cleanPattern)
        
        // Always make the pattern relative for consistent matching
        if isAbs && basePath != "" {
                if rel, err := filepath.Rel(filepath.Dir(basePath), cleanPattern); err == nil {
                cleanPattern = rel
            }
        }
        
        // Ensure all patterns start with ./ for consistent matching
        cleanPattern = filepath.Join(".", cleanPattern)
        
        g, err := glob.Compile(cleanPattern)
        if err != nil {
            return nil, fmt.Errorf("invalid glob pattern %q: %v", pattern, err)
        }
        globs = append(globs, g)
    }
    return &RefResolver{
        allowedPatterns: globs,
        loadedRefs:     make(map[string]interface{}),
        basePath:       basePath,
    }, nil
}

func (r *RefResolver) ResolveRefs(schema interface{}) (interface{}, error) {
    return r.resolveRefsRecursive(schema, r.basePath)
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
    // Use the full reference path as cache key
    fullPath := ref
    if !filepath.IsAbs(ref) {
        fullPath = filepath.Join(filepath.Dir(basePath), ref)
    }
    
    if cached, ok := r.loadedRefs[fullPath]; ok {
        return cached, nil
    }

    // Parse URL first to handle file:// scheme
    u, err := url.Parse(ref)
    if err != nil {
        return nil, fmt.Errorf("invalid $ref URL: %v", err)
    }

    if u.Scheme != "" && u.Scheme != "file" {
        return nil, fmt.Errorf("only file:// and relative refs are supported")
    }

    path := ref
    if u.Scheme == "file" {
        path = u.Path
    }

    baseDir := "."
    if basePath != "" {
        baseDir = filepath.Dir(basePath)
    }

    // Resolve paths relative to the base schema
    resolvedPath := path
    if !filepath.IsAbs(path) {
        // For relative refs, join with the base directory and clean the path
        resolvedPath = filepath.Clean(filepath.Join(baseDir, path))
    }

    // For pattern matching, make the resolved path relative to initial base path
    checkPath := resolvedPath
    if filepath.IsAbs(resolvedPath) {
        rel, err := filepath.Rel(filepath.Dir(r.basePath), resolvedPath)
        if err != nil {
            return nil, fmt.Errorf("failed to make path %q relative to initial base path %q: %v", 
                resolvedPath, r.basePath, err)
        }
        checkPath = rel
    }
    
    // Ensure pattern matching uses ./
    checkPath = filepath.Join(".", checkPath)
    
    // Check if path is allowed using the relative path from initial base
    var allowed bool
    for _, pattern := range r.allowedPatterns {
        if pattern.Match(checkPath) {
            allowed = true
            break
        }
    }
    if !allowed {
        return nil, fmt.Errorf("$ref path %q not allowed (relative to base: %q)", ref, checkPath)
    }

    // Load and parse referenced file using the absolute resolved path
    data, err := ioutil.ReadFile(filepath.Clean(resolvedPath))
    if err != nil {
        return nil, fmt.Errorf("failed to read $ref file %q: %v", ref, err)
    }

    var parsed interface{}
    if err := json.Unmarshal(data, &parsed); err != nil {
        return nil, fmt.Errorf("failed to parse $ref file %q: %v", ref, err)
    }

    // Before caching, resolve any nested refs in the loaded file using the resolved path as base
    resolved, err := r.resolveRefsRecursive(parsed, resolvedPath)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve nested refs in %q: %v", resolvedPath, err)
    }

    // Cache the resolved result
    r.loadedRefs[fullPath] = resolved
    return resolved, nil
}
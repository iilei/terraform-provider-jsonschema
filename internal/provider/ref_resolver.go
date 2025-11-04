package provider

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/url"
    "path/filepath"
    "strings"

    "github.com/gobwas/glob"
    "github.com/xeipuuv/gojsonpointer"
)

type RefResolver struct {
    allowedPatterns []glob.Glob
    loadedFiles    map[string]interface{} // Cache parsed files by absolute path
    loadedRefs     map[string]interface{} // Cache resolved refs (with fragments) by path#fragment
    baseDir        string                 // Directory to resolve relative paths from
}

func NewRefResolver(patterns []string, baseDir string) (*RefResolver, error) {
    globs := make([]glob.Glob, 0, len(patterns))
    for _, pattern := range patterns {
        // Clean and normalize the pattern
        cleanPattern := filepath.Clean(pattern)
        
        // Store whether the pattern was absolute for later use
        isAbs := filepath.IsAbs(cleanPattern)
        
        // Always make the pattern relative for consistent matching
        if isAbs && baseDir != "" {
            // Make absolute patterns relative to baseDir
            if rel, err := filepath.Rel(baseDir, cleanPattern); err == nil {
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
        loadedFiles:    make(map[string]interface{}),
        loadedRefs:     make(map[string]interface{}),
        baseDir:        baseDir,
    }, nil
}

func (r *RefResolver) ResolveRefs(schema interface{}) (interface{}, error) {
    // Start with no file context, only baseDir for initial resolution
    return r.resolveRefsRecursive(schema, r.baseDir)
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

func (r *RefResolver) resolveRef(ref string, currentPath string) (interface{}, error) {
    // Split ref into file path and fragment
    var fragment string
    refPath := ref
    if idx := strings.Index(ref, "#"); idx >= 0 {
        refPath = ref[:idx]
        fragment = ref[idx+1:]
    }

    // Handle Windows absolute paths before url.Parse
    // url.Parse("C:\path\file.json") would interpret "C" as a scheme
    var path string
    
    if filepath.IsAbs(refPath) || filepath.VolumeName(refPath) != "" {
        // This is a local absolute path (including Windows drive letters)
        path = refPath
    } else {
        // Parse URL for potential file:// scheme
        u, err := url.Parse(refPath)
        if err != nil {
            return nil, fmt.Errorf("invalid $ref URL: %v", err)
        }

        if u.Scheme != "" && u.Scheme != "file" {
            return nil, fmt.Errorf("only file:// and relative refs are supported")
        }

        if u.Scheme == "file" {
            path = u.Path
        } else {
            path = refPath
        }
    }

    // Resolve the absolute path of the referenced file
    var resolvedPath string
    if filepath.IsAbs(path) {
        resolvedPath = filepath.Clean(path)
    } else {
        resolvedPath = filepath.Clean(filepath.Join(currentPath, path))
    }

    // Use resolved path + fragment as cache key for resolved refs
    cacheKey := resolvedPath
    if fragment != "" {
        cacheKey = resolvedPath + "#" + fragment
    }
    
    // Check if we already resolved this exact ref (with fragment)
    if cached, ok := r.loadedRefs[cacheKey]; ok {
        return cached, nil
    }

    // Make the path relative to baseDir for pattern matching
    rel, err := filepath.Rel(r.baseDir, resolvedPath)
    if err != nil {
        return nil, fmt.Errorf("failed to make path %q relative to base directory %q: %v", 
            resolvedPath, r.baseDir, err)
    }
    
    // Ensure pattern matching uses ./
    checkPath := filepath.Join(".", rel)
    
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

    // Check if we already loaded and parsed this file
    var parsed interface{}
    if cached, ok := r.loadedFiles[resolvedPath]; ok {
        parsed = cached
    } else {
        // Load and parse the file for the first time
        data, err := ioutil.ReadFile(resolvedPath)
        if err != nil {
            return nil, fmt.Errorf("failed to read $ref file %q: %v", ref, err)
        }

        if err := json.Unmarshal(data, &parsed); err != nil {
            return nil, fmt.Errorf("failed to parse $ref file %q: %v", ref, err)
        }

        // Cache the parsed file content
        r.loadedFiles[resolvedPath] = parsed
    }

    // Apply JSON Pointer fragment if present
    var fragmentResult interface{}
    if fragment != "" {
        pointer, err := gojsonpointer.NewJsonPointer("/" + fragment)
        if err != nil {
            return nil, fmt.Errorf("invalid JSON Pointer fragment %q: %v", fragment, err)
        }
        fragmentResult, _, err = pointer.Get(parsed)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve fragment %q in file %q: %v", fragment, ref, err)
        }
    } else {
        fragmentResult = parsed
    }

    // Before caching, resolve any nested refs in the loaded content
    // Use the directory of the current file as the path context
    currentFileDir := filepath.Dir(resolvedPath)
    resolved, err := r.resolveRefsRecursive(fragmentResult, currentFileDir)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve nested refs in %q: %v", ref, err)
    }

    // Cache the fully resolved result (with fragment applied and nested refs resolved)
    r.loadedRefs[cacheKey] = resolved
    return resolved, nil
}
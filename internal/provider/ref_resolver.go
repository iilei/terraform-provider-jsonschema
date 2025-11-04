package provider

import (
    "encoding/json"
    "fmt"
    "net/url"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "github.com/gobwas/glob"
    "github.com/xeipuuv/gojsonpointer"
)

type RefResolver struct {
    allowedPatterns []glob.Glob
    loadedFiles    map[string]interface{} // Cache parsed files by absolute path
    loadedRefs     map[string]interface{} // Cache resolved refs (with fragments) by path#fragment
    baseDir        string                 // Directory to resolve relative paths from
    mu             sync.RWMutex           // Protects loadedFiles and loadedRefs maps
}

func NewRefResolver(patterns []string, baseDir string) (*RefResolver, error) {
    // Canonicalize baseDir to absolute path for consistent resolution and caching
    if baseDir != "" {
        absBaseDir, err := filepath.Abs(baseDir)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve absolute path for baseDir %q: %v", baseDir, err)
        }
        baseDir = absBaseDir
    }
    
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
        
        // Normalize to forward slashes for cross-platform glob matching
        // Most glob libraries expect Unix-style separators
        compilePattern := filepath.ToSlash(cleanPattern)
        
        g, err := glob.Compile(compilePattern)
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
    return r.resolveRefsRecursive(schema, schema, r.baseDir)
}

// resolveRefsRecursive recursively resolves all $ref fields in the schema.
// rootDoc is the root document for resolving fragment-only refs.
// currentDir is the directory path used to resolve relative refs within this schema.
func (r *RefResolver) resolveRefsRecursive(v interface{}, rootDoc interface{}, currentDir string) (interface{}, error) {
    switch x := v.(type) {
    case map[string]interface{}:
        if ref, ok := x["$ref"].(string); ok {
            // Handle fragment-only refs (e.g., "#/definitions/Foo")
            if strings.HasPrefix(ref, "#") {
                return r.resolveFragmentOnly(ref, rootDoc)
            }
            return r.resolveRef(ref, rootDoc, currentDir)
        }
        
        result := make(map[string]interface{})
        for k, v := range x {
            resolved, err := r.resolveRefsRecursive(v, rootDoc, currentDir)
            if err != nil {
                return nil, err
            }
            result[k] = resolved
        }
        return result, nil
        
    case []interface{}:
        result := make([]interface{}, len(x))
        for i, v := range x {
            resolved, err := r.resolveRefsRecursive(v, rootDoc, currentDir)
            if err != nil {
                return nil, err
            }
            result[i] = resolved
        }
        return result, nil
    }
    return v, nil
}

// resolveFragmentOnly resolves a fragment-only $ref (e.g., "#/definitions/Foo") against the root document.
func (r *RefResolver) resolveFragmentOnly(ref string, rootDoc interface{}) (interface{}, error) {
    fragment := strings.TrimPrefix(ref, "#")
    
    // Normalize fragment to ensure it starts with "/" for JSON Pointer
    fragmentNorm := fragment
    if fragmentNorm == "" {
        // Empty fragment means use whole document
        return rootDoc, nil
    }
    if !strings.HasPrefix(fragmentNorm, "/") {
        fragmentNorm = "/" + fragmentNorm
    }
    
    pointer, err := gojsonpointer.NewJsonPointer(fragmentNorm)
    if err != nil {
        return nil, fmt.Errorf("invalid JSON Pointer fragment %q: %v", fragment, err)
    }
    
    result, _, err := pointer.Get(rootDoc)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve fragment-only ref %q: %v", ref, err)
    }
    
    // Recursively resolve any nested refs in the result
    // Fragment-only refs are resolved against the same root document
    return r.resolveRefsRecursive(result, rootDoc, r.baseDir)
}

// resolveRef resolves a single $ref string.
// rootDoc is the root document for resolving fragment-only refs in nested content.
// currentDir is the directory path to resolve relative refs from (e.g., directory containing the current schema file).
func (r *RefResolver) resolveRef(ref string, rootDoc interface{}, currentDir string) (interface{}, error) {
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
        resolvedPath = filepath.Clean(filepath.Join(currentDir, path))
    }
    
    // Canonicalize to absolute path for consistent cache keys
    absResolvedPath, err := filepath.Abs(resolvedPath)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve absolute path for %q: %v", resolvedPath, err)
    }
    resolvedPath = absResolvedPath

    // Use resolved path + fragment as cache key for resolved refs
    cacheKey := resolvedPath
    if fragment != "" {
        cacheKey = resolvedPath + "#" + fragment
    }
    
    // Check if we already resolved this exact ref (with fragment)
    r.mu.RLock()
    cached, ok := r.loadedRefs[cacheKey]
    r.mu.RUnlock()
    if ok {
        return cached, nil
    }

    // Make the path relative to baseDir for pattern matching
    rel, err := filepath.Rel(r.baseDir, resolvedPath)
    if err != nil {
        return nil, fmt.Errorf("failed to make path %q relative to base directory %q: %v", 
            resolvedPath, r.baseDir, err)
    }
    
    // Ensure pattern matching uses ./ and normalize to forward slashes for cross-platform consistency
    checkPath := filepath.ToSlash(filepath.Join(".", rel))
    
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
    r.mu.RLock()
    parsed, ok := r.loadedFiles[resolvedPath]
    r.mu.RUnlock()
    
    if !ok {
        // Load and parse the file for the first time
        data, err := os.ReadFile(resolvedPath)
        if err != nil {
            return nil, fmt.Errorf("failed to read $ref file %q: %v", ref, err)
        }

        if err := json.Unmarshal(data, &parsed); err != nil {
            return nil, fmt.Errorf("failed to parse $ref file %q: %v", ref, err)
        }

        // Cache the parsed file content
        r.mu.Lock()
        r.loadedFiles[resolvedPath] = parsed
        r.mu.Unlock()
    }

    // Apply JSON Pointer fragment if present
    var fragmentResult interface{}
    if fragment != "" {
        // Normalize fragment to ensure it starts with "/" for JSON Pointer
        fragmentNorm := fragment
        if fragmentNorm == "" {
            // Empty fragment means use whole document
            fragmentResult = parsed
        } else {
            if !strings.HasPrefix(fragmentNorm, "/") {
                fragmentNorm = "/" + fragmentNorm
            }
            pointer, err := gojsonpointer.NewJsonPointer(fragmentNorm)
            if err != nil {
                return nil, fmt.Errorf("invalid JSON Pointer fragment %q: %v", fragment, err)
            }
            fragmentResult, _, err = pointer.Get(parsed)
            if err != nil {
                return nil, fmt.Errorf("failed to resolve fragment %q in file %q: %v", fragment, ref, err)
            }
        }
    } else {
        fragmentResult = parsed
    }

    // Before caching, resolve any nested refs in the loaded content
    // Use the directory of the current file as the path context
    // Use the parsed file as the new root document for fragment-only refs within it
    currentFileDir := filepath.Dir(resolvedPath)
    resolved, err := r.resolveRefsRecursive(fragmentResult, parsed, currentFileDir)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve nested refs in %q: %v", ref, err)
    }

    // Cache the fully resolved result (with fragment applied and nested refs resolved)
    r.mu.Lock()
    r.loadedRefs[cacheKey] = resolved
    r.mu.Unlock()
    
    return resolved, nil
}
package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-lsp/internal/document"
	"frugal-lsp/pkg/ast"
)

// IncludeResolver handles cross-file includes resolution and dependency tracking
type IncludeResolver struct {
	// Map from document URI to its dependencies (files it includes)
	dependencies map[string][]string
	
	// Map from document URI to its dependents (files that include it)
	dependents map[string][]string
	
	// Map from include path to resolved file URI
	includeCache map[string]string
	
	// Workspace root paths for resolving relative includes
	workspaceRoots []string
	
	mutex sync.RWMutex
}

// NewIncludeResolver creates a new include resolver
func NewIncludeResolver(workspaceRoots []string) *IncludeResolver {
	return &IncludeResolver{
		dependencies:   make(map[string][]string),
		dependents:     make(map[string][]string),
		includeCache:   make(map[string]string),
		workspaceRoots: workspaceRoots,
	}
}

// UpdateDocument updates the dependency tracking for a document
func (r *IncludeResolver) UpdateDocument(doc *document.Document) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Clear existing dependencies for this document
	r.clearDependencies(doc.URI)
	
	// Extract includes from the document
	includes := r.extractIncludes(doc)
	if len(includes) == 0 {
		return nil
	}
	
	// Resolve include paths to actual file URIs
	var resolvedDependencies []string
	for _, includePath := range includes {
		resolvedURI, err := r.resolveIncludePath(includePath, doc.URI)
		if err != nil {
			// Log warning but continue - include might not exist yet
			continue
		}
		resolvedDependencies = append(resolvedDependencies, resolvedURI)
	}
	
	// Update dependency mappings
	r.dependencies[doc.URI] = resolvedDependencies
	for _, depURI := range resolvedDependencies {
		if r.dependents[depURI] == nil {
			r.dependents[depURI] = make([]string, 0)
		}
		r.dependents[depURI] = append(r.dependents[depURI], doc.URI)
	}
	
	return nil
}

// RemoveDocument removes a document from dependency tracking
func (r *IncludeResolver) RemoveDocument(uri string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.clearDependencies(uri)
	
	// Remove this document from all dependents lists
	for depURI, dependentsList := range r.dependents {
		r.dependents[depURI] = r.removeFromSlice(dependentsList, uri)
		if len(r.dependents[depURI]) == 0 {
			delete(r.dependents, depURI)
		}
	}
}

// GetDependencies returns the files that the given document depends on
func (r *IncludeResolver) GetDependencies(uri string) []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	deps := r.dependencies[uri]
	result := make([]string, len(deps))
	copy(result, deps)
	return result
}

// GetDependents returns the files that depend on the given document
func (r *IncludeResolver) GetDependents(uri string) []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	deps := r.dependents[uri]
	result := make([]string, len(deps))
	copy(result, deps)
	return result
}

// GetAllSymbols returns symbols from a document and all its dependencies
func (r *IncludeResolver) GetAllSymbols(doc *document.Document, docManager *document.Manager) []ast.Symbol {
	var allSymbols []ast.Symbol
	visited := make(map[string]bool)
	
	r.collectSymbolsRecursive(doc.URI, docManager, &allSymbols, visited)
	return allSymbols
}

// HasCircularDependency checks if adding a dependency would create a circular reference
func (r *IncludeResolver) HasCircularDependency(fromURI, toURI string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	visited := make(map[string]bool)
	return r.hasCircularDependencyRecursive(toURI, fromURI, visited)
}

// extractIncludes extracts include statements from a document
func (r *IncludeResolver) extractIncludes(doc *document.Document) []string {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return nil
	}
	
	var includes []string
	root := doc.ParseResult.GetRootNode()
	
	r.findIncludesRecursive(root, doc.Content, &includes)
	return includes
}

// findIncludesRecursive recursively searches for include statements
func (r *IncludeResolver) findIncludesRecursive(node *tree_sitter.Node, source []byte, includes *[]string) {
	if node == nil {
		return
	}
	
	if node.Kind() == "include" {
		// Find the string literal in the include statement
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			if child.Kind() == "literal_string" {
				includeText := ast.GetText(child, source)
				// Remove quotes from the string literal
				if len(includeText) >= 2 && includeText[0] == '"' && includeText[len(includeText)-1] == '"' {
					includePath := includeText[1 : len(includeText)-1]
					*includes = append(*includes, includePath)
				}
				break
			}
		}
	}
	
	// Recursively search child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		r.findIncludesRecursive(child, source, includes)
	}
}

// resolveIncludePath resolves a relative include path to an absolute file URI
func (r *IncludeResolver) resolveIncludePath(includePath, fromURI string) (string, error) {
	// Check cache first
	cacheKey := fromURI + ":" + includePath
	if resolvedURI, exists := r.includeCache[cacheKey]; exists {
		return resolvedURI, nil
	}
	
	// Parse the source URI to get its directory
	fromPath, err := uriToPath(fromURI)
	if err != nil {
		return "", fmt.Errorf("invalid source URI: %w", err)
	}
	
	fromDir := filepath.Dir(fromPath)
	
	// Try resolving relative to the source file
	candidates := []string{
		filepath.Join(fromDir, includePath),
	}
	
	// Also try resolving relative to workspace roots
	for _, root := range r.workspaceRoots {
		candidates = append(candidates, filepath.Join(root, includePath))
	}
	
	// Find the first existing file
	for _, candidate := range candidates {
		// Clean and normalize the path
		cleanPath := filepath.Clean(candidate)
		
		// Convert back to file URI
		resolvedURI := pathToURI(cleanPath)
		
		// Cache the result
		r.includeCache[cacheKey] = resolvedURI
		
		return resolvedURI, nil
	}
	
	return "", fmt.Errorf("could not resolve include path: %s", includePath)
}

// clearDependencies removes all dependency mappings for a document
func (r *IncludeResolver) clearDependencies(uri string) {
	// Remove from dependents of our dependencies
	if deps, exists := r.dependencies[uri]; exists {
		for _, depURI := range deps {
			if dependentsList, exists := r.dependents[depURI]; exists {
				r.dependents[depURI] = r.removeFromSlice(dependentsList, uri)
				if len(r.dependents[depURI]) == 0 {
					delete(r.dependents, depURI)
				}
			}
		}
	}
	
	// Clear our dependencies
	delete(r.dependencies, uri)
}

// removeFromSlice removes a string from a slice
func (r *IncludeResolver) removeFromSlice(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// collectSymbolsRecursive collects symbols from a document and its dependencies
func (r *IncludeResolver) collectSymbolsRecursive(uri string, docManager *document.Manager, allSymbols *[]ast.Symbol, visited map[string]bool) {
	if visited[uri] {
		return // Avoid infinite recursion
	}
	visited[uri] = true
	
	// Get document and its symbols
	if doc, exists := docManager.GetDocument(uri); exists {
		symbols := doc.GetSymbols()
		*allSymbols = append(*allSymbols, symbols...)
		
		// Recursively collect from dependencies
		r.mutex.RLock()
		deps := r.dependencies[uri]
		r.mutex.RUnlock()
		
		for _, depURI := range deps {
			r.collectSymbolsRecursive(depURI, docManager, allSymbols, visited)
		}
	}
}

// hasCircularDependencyRecursive checks for circular dependencies
func (r *IncludeResolver) hasCircularDependencyRecursive(currentURI, targetURI string, visited map[string]bool) bool {
	if currentURI == targetURI {
		return true
	}
	
	if visited[currentURI] {
		return false // Already checked this path
	}
	visited[currentURI] = true
	
	// Check dependencies of current URI
	deps := r.dependencies[currentURI]
	for _, depURI := range deps {
		if r.hasCircularDependencyRecursive(depURI, targetURI, visited) {
			return true
		}
	}
	
	return false
}

// uriToPath converts a file URI to a file system path
func uriToPath(uri string) (string, error) {
	if strings.HasPrefix(uri, "file://") {
		return uri[7:], nil // Remove "file://" prefix
	}
	return uri, nil // Assume it's already a path
}

// pathToURI converts a file system path to a file URI
func pathToURI(path string) string {
	if strings.HasPrefix(path, "/") {
		return "file://" + path
	}
	return "file:///" + filepath.ToSlash(path) // Windows paths
}
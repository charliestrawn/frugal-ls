package workspace

import (
	"testing"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
)

func TestNewIncludeResolver(t *testing.T) {
	workspaceRoots := []string{"/workspace1", "/workspace2"}
	resolver := NewIncludeResolver(workspaceRoots)

	if resolver == nil {
		t.Fatal("Expected non-nil IncludeResolver")
	}

	if len(resolver.workspaceRoots) != 2 {
		t.Errorf("Expected 2 workspace roots, got %d", len(resolver.workspaceRoots))
	}

	if resolver.workspaceRoots[0] != "/workspace1" || resolver.workspaceRoots[1] != "/workspace2" {
		t.Errorf("Expected workspace roots [/workspace1, /workspace2], got %v", resolver.workspaceRoots)
	}

	if resolver.dependencies == nil {
		t.Error("Expected non-nil dependencies map")
	}

	if resolver.dependents == nil {
		t.Error("Expected non-nil dependents map")
	}

	if resolver.includeCache == nil {
		t.Error("Expected non-nil includeCache map")
	}
}

func TestUpdateDocumentNoIncludes(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	content := `struct User {
    1: i64 id,
    2: string name
}`

	doc := createTestDocument("file:///test.frugal", content)

	err := resolver.UpdateDocument(doc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Document with no includes should have no dependencies
	deps := resolver.GetDependencies("file:///test.frugal")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}
}

func TestUpdateDocumentWithIncludes(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	content := `include "common.frugal"
include "types.frugal"

struct User {
    1: i64 id
}`

	doc := createTestDocument("file:///test.frugal", content)

	err := resolver.UpdateDocument(doc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should find dependencies
	deps := resolver.GetDependencies("file:///test.frugal")
	if len(deps) == 0 {
		t.Error("Expected dependencies to be found")
	}
}

func TestGetDependenciesNotFound(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	deps := resolver.GetDependencies("file:///nonexistent.frugal")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for nonexistent file, got %d", len(deps))
	}
}

func TestGetDependents(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	// Create a document that includes another file
	content1 := `include "common.frugal"

struct User {
    1: i64 id
}`

	doc1 := createTestDocument("file:///user.frugal", content1)
	err := resolver.UpdateDocument(doc1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	content2 := `include "common.frugal"

struct Order {
    1: i64 id
}`

	doc2 := createTestDocument("file:///order.frugal", content2)
	err = resolver.UpdateDocument(doc2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Get dependents of common.frugal (files that include it)
	dependents := resolver.GetDependents("file:///common.frugal")

	// Both user.frugal and order.frugal should depend on common.frugal
	if len(dependents) < 1 {
		t.Error("Expected at least 1 dependent of common.frugal")
	}
}

func TestGetDependentsNotFound(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	dependents := resolver.GetDependents("file:///nonexistent.frugal")
	if len(dependents) != 0 {
		t.Errorf("Expected 0 dependents for nonexistent file, got %d", len(dependents))
	}
}

func TestResolveIncludePath(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	testCases := []struct {
		name        string
		includePath string
		fromFile    string
		expected    string
		expectError bool
	}{
		{
			name:        "relative path",
			includePath: "common.frugal",
			fromFile:    "file:///workspace/service.frugal",
			expected:    "file:///workspace/common.frugal",
			expectError: false,
		},
		{
			name:        "nested relative path",
			includePath: "../shared/types.frugal",
			fromFile:    "file:///workspace/service/user.frugal",
			expected:    "file:///workspace/shared/types.frugal",
			expectError: false,
		},
		{
			name:        "subdirectory path",
			includePath: "models/user.frugal",
			fromFile:    "file:///workspace/service.frugal",
			expected:    "file:///workspace/models/user.frugal",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := resolver.resolveIncludePath(tc.includePath, tc.fromFile)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if resolved != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, resolved)
			}
		})
	}
}

func TestClearDependencies(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	// Add some dependencies first
	content := `include "common.frugal"

struct User {
    1: i64 id
}`

	doc := createTestDocument("file:///test.frugal", content)
	err := resolver.UpdateDocument(doc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify dependencies exist
	deps := resolver.GetDependencies("file:///test.frugal")
	if len(deps) == 0 {
		t.Error("Expected dependencies to be found")
	}

	// Update with content that has no includes
	contentNoIncludes := `struct User {
    1: i64 id
}`

	docNoIncludes := createTestDocument("file:///test.frugal", contentNoIncludes)
	err = resolver.UpdateDocument(docNoIncludes)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Dependencies should be cleared
	deps = resolver.GetDependencies("file:///test.frugal")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies after clearing, got %d", len(deps))
	}
}

func TestConcurrentAccess(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	// Test concurrent access to resolver methods
	done := make(chan bool)

	// Concurrent updates
	go func() {
		for i := 0; i < 10; i++ {
			content := `include "common.frugal"
struct Test {}`
			doc := createTestDocument("file:///test1.frugal", content)
			resolver.UpdateDocument(doc)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			content := `include "shared.frugal"
struct Test {}`
			doc := createTestDocument("file:///test2.frugal", content)
			resolver.UpdateDocument(doc)
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 10; i++ {
			resolver.GetDependencies("file:///test1.frugal")
			resolver.GetDependents("file:///common.frugal")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestExtractIncludes(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	testCases := []struct {
		name            string
		content         string
		expectedCount   int
		expectedInclude string
	}{
		{
			name:            "single include",
			content:         `include "common.frugal"`,
			expectedCount:   1,
			expectedInclude: "common.frugal",
		},
		{
			name: "multiple includes",
			content: `include "common.frugal"
include "types.frugal"
include "services.frugal"`,
			expectedCount: 3,
		},
		{
			name:          "no includes",
			content:       `struct User { 1: i64 id }`,
			expectedCount: 0,
		},
		{
			name: "includes with other content",
			content: `include "common.frugal"

struct User {
    1: i64 id
}

include "types.frugal"

service UserService {
    User getUser()
}`,
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := createTestDocument("file:///test.frugal", tc.content)

			includes := resolver.extractIncludes(doc)

			if len(includes) != tc.expectedCount {
				t.Errorf("Expected %d includes, got %d", tc.expectedCount, len(includes))
			}

			if tc.expectedInclude != "" {
				found := false
				for _, include := range includes {
					if include == tc.expectedInclude {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find include %q in %v", tc.expectedInclude, includes)
				}
			}
		})
	}
}

func TestResolveIncludePathEdgeCases(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	testCases := []struct {
		name        string
		includePath string
		fromFile    string
		expectError bool
	}{
		{
			name:        "empty include path",
			includePath: "",
			fromFile:    "file:///test.frugal",
			expectError: false, // Resolver handles gracefully
		},
		{
			name:        "empty from file",
			includePath: "common.frugal",
			fromFile:    "",
			expectError: false, // Resolver handles gracefully
		},
		{
			name:        "malformed URI",
			includePath: "common.frugal",
			fromFile:    "not-a-uri",
			expectError: false, // Resolver handles gracefully
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolver.resolveIncludePath(tc.includePath, tc.fromFile)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result == "" && tc.includePath != "" {
					t.Error("Expected non-empty result for valid include path")
				}
			}
		})
	}
}

func TestDependencyGraphIntegrity(t *testing.T) {
	resolver := NewIncludeResolver([]string{"/workspace"})

	// Create a dependency chain: A includes B, B includes C
	contentA := `include "b.frugal"
struct A {}`

	contentB := `include "c.frugal"
struct B {}`

	contentC := `struct C {}`

	docA := createTestDocument("file:///a.frugal", contentA)
	docB := createTestDocument("file:///b.frugal", contentB)
	docC := createTestDocument("file:///c.frugal", contentC)

	// Update documents
	resolver.UpdateDocument(docA)
	resolver.UpdateDocument(docB)
	resolver.UpdateDocument(docC)

	// Check dependency relationships
	depsA := resolver.GetDependencies("file:///a.frugal")
	if len(depsA) == 0 {
		t.Error("Expected A to have dependencies")
	}

	depsB := resolver.GetDependencies("file:///b.frugal")
	if len(depsB) == 0 {
		t.Error("Expected B to have dependencies")
	}

	depsC := resolver.GetDependencies("file:///c.frugal")
	if len(depsC) != 0 {
		t.Errorf("Expected C to have no dependencies, got %d", len(depsC))
	}

	// Check dependent relationships
	dependentsC := resolver.GetDependents("file:///c.frugal")
	if len(dependentsC) == 0 {
		t.Error("Expected C to have dependents")
	}
}

// Helper function to create a test document
func createTestDocument(uri string, content string) *document.Document {
	// Create with parse result for realistic testing
	p, err := parser.NewParser()
	if err != nil {
		// Return document without parse result if parser creation fails
		return &document.Document{
			URI:         uri,
			Path:        uri[7:], // Remove file:// prefix
			Content:     []byte(content),
			Version:     1,
			ParseResult: nil,
		}
	}
	defer p.Close()

	parseResult, err := p.Parse([]byte(content))
	if err != nil {
		// Return document without parse result if parsing fails
		return &document.Document{
			URI:         uri,
			Path:        uri[7:], // Remove file:// prefix
			Content:     []byte(content),
			Version:     1,
			ParseResult: nil,
		}
	}

	return &document.Document{
		URI:         uri,
		Path:        uri[7:], // Remove file:// prefix
		Content:     []byte(content),
		Version:     1,
		ParseResult: parseResult,
	}
}

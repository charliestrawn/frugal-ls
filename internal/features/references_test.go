package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
)

func TestReferencesProvider(t *testing.T) {
	provider := NewReferencesProvider()

	tests := []struct {
		name                string
		content             string
		position            protocol.Position
		includeDeclaration  bool
		expectedRefCount    int
	}{
		{
			name: "find struct references",
			content: `struct User {
    1: string name
}

service UserService {
    User getUser(1: i64 id),
    void updateUser(1: User user)
}

const User DEFAULT_USER = {};`,
			position: protocol.Position{Line: 0, Character: 7}, // "User" in struct declaration
			includeDeclaration: true,
			expectedRefCount: 4, // Declaration + 3 references
		},
		{
			name: "find field references",
			content: `struct User {
    1: string name,
    2: string email
}

service UserService {
    string getName() {
        return user.name;
    }
}`,
			position: protocol.Position{Line: 1, Character: 14}, // "name" in field declaration
			includeDeclaration: true,
			expectedRefCount: 1, // Only declaration (user.name might not parse as identifier)
		},
		{
			name: "find service method references",
			content: `service UserService {
    User getUser(1: i64 id)
}

service AdminService extends UserService {
    void callGetUser() {
        this.getUser(123);
    }
}`,
			position: protocol.Position{Line: 1, Character: 9}, // "getUser" in method declaration
			includeDeclaration: true,
			expectedRefCount: 1, // Only declaration (method calls might not parse as simple identifiers)
		},
		{
			name: "no references found",
			content: `struct User {
    1: string name
}

struct Product {
    1: string title
}`,
			position: protocol.Position{Line: 4, Character: 7}, // "Product" 
			includeDeclaration: true,
			expectedRefCount: 1, // Only declaration
		},
		{
			name: "exclude declaration",
			content: `struct User {
    1: string name
}

service UserService {
    User getUser(1: i64 id)
}`,
			position: protocol.Position{Line: 0, Character: 7}, // "User" in struct declaration
			includeDeclaration: false,
			expectedRefCount: 1, // Only the reference in service method
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := createTestDocument("file:///test.frugal", tt.content)
			if err != nil {
				t.Fatalf("failed to create document: %v", err)
			}

			allDocuments := map[string]*document.Document{
				"file:///test.frugal": doc,
			}

			references, err := provider.ProvideReferences(doc, tt.position, tt.includeDeclaration, allDocuments)
			if err != nil {
				t.Fatalf("failed to provide references: %v", err)
			}

			if len(references) != tt.expectedRefCount {
				t.Errorf("expected %d references, got %d", tt.expectedRefCount, len(references))
				for i, ref := range references {
					t.Logf("Reference %d: %+v", i, ref.Range)
				}
			}

			// Verify all references have valid ranges
			for i, ref := range references {
				if ref.Range.Start.Line < 0 || ref.Range.End.Line < 0 {
					t.Errorf("Reference %d has invalid line numbers: %+v", i, ref.Range)
				}
				if ref.Range.Start.Character < 0 || ref.Range.End.Character < 0 {
					t.Errorf("Reference %d has invalid character positions: %+v", i, ref.Range)
				}
				if ref.URI == "" {
					t.Errorf("Reference %d has empty URI", i)
				}
			}
		})
	}
}

func TestReferencesProviderCrossFile(t *testing.T) {
	provider := NewReferencesProvider()

	// Create multiple documents with cross-references
	commonContent := `struct User {
    1: string name,
    2: string email
}`

	serviceContent := `include "common.frugal"

service UserService {
    User getUser(1: i64 id),
    void updateUser(1: User user)
}`

	clientContent := `include "common.frugal"
include "service.frugal"

struct UserRequest {
    1: User user
}`

	commonDoc, err := createTestDocument("file:///common.frugal", commonContent)
	if err != nil {
		t.Fatalf("failed to create common document: %v", err)
	}

	serviceDoc, err := createTestDocument("file:///service.frugal", serviceContent)
	if err != nil {
		t.Fatalf("failed to create service document: %v", err)
	}

	clientDoc, err := createTestDocument("file:///client.frugal", clientContent)
	if err != nil {
		t.Fatalf("failed to create client document: %v", err)
	}

	allDocuments := map[string]*document.Document{
		"file:///common.frugal":  commonDoc,
		"file:///service.frugal": serviceDoc,
		"file:///client.frugal":  clientDoc,
	}

	// Find references to "User" struct from common.frugal
	references, err := provider.ProvideReferences(
		commonDoc,
		protocol.Position{Line: 0, Character: 7}, // "User" in struct declaration
		true, // include declaration
		allDocuments,
	)

	if err != nil {
		t.Fatalf("failed to provide cross-file references: %v", err)
	}

	// Should find references in all three files
	// Declaration in common.frugal + references in service.frugal + reference in client.frugal
	expectedMinRefs := 3
	if len(references) < expectedMinRefs {
		t.Errorf("expected at least %d cross-file references, got %d", expectedMinRefs, len(references))
		for i, ref := range references {
			t.Logf("Reference %d: URI=%s, Range=%+v", i, ref.URI, ref.Range)
		}
	}

	// Verify we have references from different files
	uriCounts := make(map[string]int)
	for _, ref := range references {
		uriCounts[ref.URI]++
	}

	if len(uriCounts) < 2 {
		t.Errorf("expected references from multiple files, got references from %d files", len(uriCounts))
		t.Logf("URI counts: %+v", uriCounts)
	}
}

func TestGetSymbolAtPosition(t *testing.T) {
	provider := NewReferencesProvider()

	content := `struct User {
    1: string name,
    2: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	doc, err := createTestDocument("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	tests := []struct {
		name           string
		position       protocol.Position
		expectedSymbol string
		expectRange    bool
	}{
		{
			name:           "struct name",
			position:       protocol.Position{Line: 0, Character: 7},
			expectedSymbol: "User",
			expectRange:    true,
		},
		{
			name:           "field name",
			position:       protocol.Position{Line: 1, Character: 14},
			expectedSymbol: "name",
			expectRange:    true,
		},
		{
			name:           "field type",
			position:       protocol.Position{Line: 1, Character: 10},
			expectedSymbol: "string",
			expectRange:    true,
		},
		{
			name:           "service name",
			position:       protocol.Position{Line: 5, Character: 8},
			expectedSymbol: "UserService",
			expectRange:    true,
		},
		{
			name:           "method return type",
			position:       protocol.Position{Line: 6, Character: 4},
			expectedSymbol: "User",
			expectRange:    true,
		},
		{
			name:           "whitespace (no symbol)",
			position:       protocol.Position{Line: 3, Character: 0},
			expectedSymbol: "",
			expectRange:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol, symbolRange := provider.getSymbolAtPosition(doc, tt.position)

			if symbol != tt.expectedSymbol {
				t.Errorf("expected symbol '%s', got '%s'", tt.expectedSymbol, symbol)
			}

			if tt.expectRange && symbolRange == nil {
				t.Error("expected symbol range but got nil")
			} else if !tt.expectRange && symbolRange != nil {
				t.Error("expected no symbol range but got one")
			}

			if symbolRange != nil {
				// Verify range is valid
				if symbolRange.Start.Line < 0 || symbolRange.End.Line < 0 {
					t.Errorf("invalid line numbers in range: %+v", symbolRange)
				}
				if symbolRange.Start.Character < 0 || symbolRange.End.Character < 0 {
					t.Errorf("invalid character positions in range: %+v", symbolRange)
				}
			}
		})
	}
}

func TestFindReferencesInDocument(t *testing.T) {
	provider := NewReferencesProvider()

	content := `struct User {
    1: string name
}

struct UserProfile {
    1: User user,
    2: string displayName
}

service UserService {
    User getUser(1: i64 id),
    UserProfile getUserProfile(1: User user)
}`

	doc, err := createTestDocument("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	// Find all references to "User" in the document
	references := provider.findReferencesInDocument(doc, "User")

	// Should find: struct declaration, field type in UserProfile, 
	// return type in getUser, parameter type in getUserProfile
	expectedRefCount := 4
	if len(references) != expectedRefCount {
		t.Errorf("expected %d references to 'User', got %d", expectedRefCount, len(references))
		for i, ref := range references {
			t.Logf("Reference %d: %+v", i, ref)
		}
	}

	// Verify all references have valid positions
	for i, ref := range references {
		if ref.Start.Line < 0 || ref.End.Line < 0 {
			t.Errorf("Reference %d has invalid line numbers: %+v", i, ref)
		}
		if ref.Start.Character < 0 || ref.End.Character < 0 {
			t.Errorf("Reference %d has invalid character positions: %+v", i, ref)
		}
	}
}

func TestNodeToRange(t *testing.T) {
	provider := NewReferencesProvider()
	content := "struct User {\n    1: string name\n}"

	doc, err := createTestDocument("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	if doc.ParseResult == nil || doc.ParseResult.Tree == nil {
		t.Fatal("document has no parse tree")
	}

	root := doc.ParseResult.Tree.RootNode()
	
	// Find the first identifier (should be "User")
	var identifierNode *tree_sitter.Node
	provider.walkTreeForIdentifier(root, &identifierNode)

	if identifierNode == nil {
		t.Fatal("could not find identifier node")
	}

	nodeRange := provider.nodeToRange(identifierNode, content)
	if nodeRange == nil {
		t.Fatal("nodeToRange returned nil")
	}

	// The "User" identifier should be at line 0, around character 7
	if nodeRange.Start.Line != 0 {
		t.Errorf("expected start line 0, got %d", nodeRange.Start.Line)
	}

	if nodeRange.Start.Character < 6 || nodeRange.Start.Character > 8 {
		t.Errorf("expected start character around 7, got %d", nodeRange.Start.Character)
	}
}

// Helper function to find first identifier in tree
func (p *ReferencesProvider) walkTreeForIdentifier(node *tree_sitter.Node, result **tree_sitter.Node) {
	if *result != nil {
		return // Already found
	}

	if node.Kind() == "identifier" {
		*result = node
		return
	}

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		p.walkTreeForIdentifier(child, result)
	}
}

// Test helper to create a document with parsing
func createTestDocument(uri, content string) (*document.Document, error) {
	// Extract path from URI for proper validation
	path := strings.TrimPrefix(uri, "file://")
	
	doc := &document.Document{
		URI:     uri,
		Path:    path,
		Content: []byte(content),
		Version: 1,
	}

	// Parse the document
	p, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		return nil, err
	}

	doc.ParseResult = result
	return doc, nil
}
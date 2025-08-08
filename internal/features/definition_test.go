package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func createTestDocumentForDefinition(uri, content string) (*document.Document, error) {
	p, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		return nil, err
	}

	var symbols []ast.Symbol
	if result.GetRootNode() != nil {
		symbols = ast.ExtractSymbols(result.GetRootNode(), []byte(content))
	}

	// Extract path from URI for proper validation
	path := strings.TrimPrefix(uri, "file://")
	
	doc := &document.Document{
		URI:         uri,
		Path:        path,
		Content:     []byte(content),
		Version:     1,
		ParseResult: result,
		Symbols:     symbols,
	}

	return doc, nil
}

func TestDefinitionProvider(t *testing.T) {
	provider := NewDefinitionProvider()
	if provider == nil {
		t.Fatal("Definition provider should not be nil")
	}
}

func TestProvideDefinitionForStruct(t *testing.T) {
	provider := NewDefinitionProvider()

	content := `struct User {
    1: string name,
    2: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	doc, err := createTestDocumentForDefinition("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	allDocs := map[string]*document.Document{
		"file:///test.frugal": doc,
	}

	// Position at "User" in the service method return type (line 6, after indentation)
	position := protocol.Position{Line: 6, Character: 4}
	locations, err := provider.ProvideDefinition(doc, position, allDocs)
	if err != nil {
		t.Fatalf("ProvideDefinition failed: %v", err)
	}

	if len(locations) == 0 {
		t.Fatal("Expected at least one definition location")
	}

	// Should point to the struct definition at line 0
	location := locations[0]
	if location.URI != doc.URI {
		t.Errorf("Expected URI %s, got %s", doc.URI, location.URI)
	}

	if location.Range.Start.Line != 0 {
		t.Errorf("Expected definition at line 0, got line %d", location.Range.Start.Line)
	}
}

func TestProvideDefinitionForField(t *testing.T) {
	provider := NewDefinitionProvider()

	content := `struct User {
    1: string name,
    2: i64 id
}

struct UserProfile {
    1: User user,
    2: string bio
}`

	doc, err := createTestDocumentForDefinition("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	allDocs := map[string]*document.Document{
		"file:///test.frugal": doc,
	}

	// Position at "User" in UserProfile struct (line 6, after field number and colon)
	position := protocol.Position{Line: 6, Character: 8}
	locations, err := provider.ProvideDefinition(doc, position, allDocs)
	if err != nil {
		t.Fatalf("ProvideDefinition failed: %v", err)
	}

	if len(locations) == 0 {
		t.Fatal("Expected at least one definition location")
	}

	// Should point to the User struct definition at line 0
	location := locations[0]
	if location.Range.Start.Line != 0 {
		t.Errorf("Expected definition at line 0, got line %d", location.Range.Start.Line)
	}
}

func TestProvideDefinitionNoResult(t *testing.T) {
	provider := NewDefinitionProvider()

	content := `struct User {
    1: string name
}`

	doc, err := createTestDocumentForDefinition("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	allDocs := map[string]*document.Document{
		"file:///test.frugal": doc,
	}

	// Position at a primitive type "string" - should not have definition
	position := protocol.Position{Line: 1, Character: 10}
	locations, err := provider.ProvideDefinition(doc, position, allDocs)
	if err != nil {
		t.Fatalf("ProvideDefinition failed: %v", err)
	}

	// Primitive types don't have definitions
	if len(locations) > 0 {
		t.Error("Expected no definition for primitive type")
	}
}

func TestProvideDefinitionCrossFile(t *testing.T) {
	provider := NewDefinitionProvider()

	// First file with User definition
	userContent := `struct User {
    1: string name,
    2: i64 id
}`

	userDoc, err := createTestDocumentForDefinition("file:///user.frugal", userContent)
	if err != nil {
		t.Fatalf("Failed to create user document: %v", err)
	}
	defer userDoc.ParseResult.Close()

	// Second file using User
	serviceContent := `include "user.frugal"

service UserService {
    User getUser(1: i64 userId)
}`

	serviceDoc, err := createTestDocumentForDefinition("file:///service.frugal", serviceContent)
	if err != nil {
		t.Fatalf("Failed to create service document: %v", err)
	}
	defer serviceDoc.ParseResult.Close()

	allDocs := map[string]*document.Document{
		"file:///user.frugal":    userDoc,
		"file:///service.frugal": serviceDoc,
	}

	// Position at "User" in service method return type
	position := protocol.Position{Line: 3, Character: 4}
	locations, err := provider.ProvideDefinition(serviceDoc, position, allDocs)
	if err != nil {
		t.Fatalf("ProvideDefinition failed: %v", err)
	}

	if len(locations) == 0 {
		t.Fatal("Expected definition location in cross-file scenario")
	}

	// Should point to User struct in user.frugal
	location := locations[0]
	if location.URI != userDoc.URI {
		t.Errorf("Expected definition in %s, got %s", userDoc.URI, location.URI)
	}

	if location.Range.Start.Line != 0 {
		t.Errorf("Expected definition at line 0, got line %d", location.Range.Start.Line)
	}
}

func TestProvideDefinitionInvalidPosition(t *testing.T) {
	provider := NewDefinitionProvider()

	content := `struct User {
    1: string name
}`

	doc, err := createTestDocumentForDefinition("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	allDocs := map[string]*document.Document{
		"file:///test.frugal": doc,
	}

	testCases := []struct {
		name     string
		position protocol.Position
	}{
		{
			name:     "beyond last line",
			position: protocol.Position{Line: 10, Character: 0},
		},
		{
			name:     "beyond line end",
			position: protocol.Position{Line: 0, Character: 1000},
		},
		{
			name:     "whitespace",
			position: protocol.Position{Line: 2, Character: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			locations, err := provider.ProvideDefinition(doc, tc.position, allDocs)
			if err != nil {
				t.Errorf("ProvideDefinition should handle invalid position gracefully: %v", err)
			}

			// Should return empty result for invalid positions
			if locations != nil && len(locations) > 0 {
				t.Errorf("Expected no definitions for invalid position, got %d", len(locations))
			}
		})
	}
}

func TestProvideDefinitionEmptyDocument(t *testing.T) {
	provider := NewDefinitionProvider()

	doc, err := createTestDocumentForDefinition("file:///empty.frugal", "")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	allDocs := map[string]*document.Document{
		"file:///empty.frugal": doc,
	}

	position := protocol.Position{Line: 0, Character: 0}
	locations, err := provider.ProvideDefinition(doc, position, allDocs)
	if err != nil {
		t.Errorf("ProvideDefinition should handle empty document: %v", err)
	}

	if locations != nil && len(locations) > 0 {
		t.Error("Expected no definitions in empty document")
	}
}

func TestProvideDefinitionNilDocument(t *testing.T) {
	provider := NewDefinitionProvider()

	position := protocol.Position{Line: 0, Character: 0}
	allDocs := make(map[string]*document.Document)

	// Test with document that has no parse result
	doc := &document.Document{
		URI:         "file:///invalid.frugal",
		Content:     []byte("test"),
		Version:     1,
		ParseResult: nil, // No parse result
	}

	locations, err := provider.ProvideDefinition(doc, position, allDocs)
	if err != nil {
		t.Errorf("ProvideDefinition should handle nil parse result: %v", err)
	}

	if locations != nil && len(locations) > 0 {
		t.Error("Expected no definitions for document without parse result")
	}
}
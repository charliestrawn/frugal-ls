package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func createTestDocumentForCompletion(uri, content string) (*document.Document, error) {
	p, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		return nil, err
	}

	// Extract symbols like the document manager does
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

func TestCompletionProvider(t *testing.T) {
	provider := NewCompletionProvider()
	if provider == nil {
		t.Fatal("Completion provider should not be nil")
	}
}

func TestTopLevelCompletions(t *testing.T) {
	provider := NewCompletionProvider()

	// Empty file - should provide top-level completions
	doc, err := createTestDocumentForCompletion("file:///test.frugal", "")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	position := protocol.Position{Line: 0, Character: 0}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	// Should provide top-level keywords
	expectedKeywords := []string{"struct", "service", "enum", "const", "typedef", "scope", "namespace", "include"}
	foundKeywords := make(map[string]bool)

	for _, completion := range completions {
		foundKeywords[completion.Label] = true
	}

	for _, keyword := range expectedKeywords {
		if !foundKeywords[keyword] {
			t.Errorf("Expected top-level completion '%s' not found", keyword)
		}
	}
}

func TestStructCompletions(t *testing.T) {
	provider := NewCompletionProvider()

	content := `struct User {
    `
	doc, err := createTestDocumentForCompletion("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position inside struct
	position := protocol.Position{Line: 1, Character: 4}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	// Should suggest field number and basic types
	foundLabels := make(map[string]bool)
	for _, completion := range completions {
		foundLabels[completion.Label] = true
	}

	// Should suggest basic types
	basicTypes := []string{"string", "i32", "i64", "bool", "double"}
	for _, basicType := range basicTypes {
		if !foundLabels[basicType] {
			t.Errorf("Expected struct completion '%s' not found", basicType)
		}
	}
}

func TestServiceCompletions(t *testing.T) {
	provider := NewCompletionProvider()

	content := `service UserService {
    `
	doc, err := createTestDocumentForCompletion("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position inside service
	position := protocol.Position{Line: 1, Character: 4}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	// Should suggest return types and method keywords
	foundLabels := make(map[string]bool)
	for _, completion := range completions {
		foundLabels[completion.Label] = true
	}

	// Should suggest service-specific keywords and types
	serviceKeywords := []string{"void", "oneway", "string", "i32", "i64", "bool"}
	for _, keyword := range serviceKeywords {
		if !foundLabels[keyword] {
			t.Errorf("Expected service completion '%s' not found", keyword)
		}
	}
}

func TestTypeCompletions(t *testing.T) {
	provider := NewCompletionProvider()

	content := `struct User {
    1: `
	doc, err := createTestDocumentForCompletion("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position after field number where type is expected
	position := protocol.Position{Line: 1, Character: 7}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	// Should suggest types
	foundLabels := make(map[string]bool)
	for _, completion := range completions {
		foundLabels[completion.Label] = true
	}

	// Should suggest basic types and container types
	expectedTypes := []string{"string", "i32", "i64", "bool", "double", "binary", "list", "set", "map"}
	for _, expectedType := range expectedTypes {
		if !foundLabels[expectedType] {
			t.Errorf("Expected type completion '%s' not found", expectedType)
		}
	}
}

func TestScopeCompletions(t *testing.T) {
	provider := NewCompletionProvider()

	content := `scope EventScope `
	doc, err := createTestDocumentForCompletion("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position after scope name where "prefix" is expected
	position := protocol.Position{Line: 0, Character: 18}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	// Should suggest "prefix" keyword
	foundPrefix := false
	for _, completion := range completions {
		if completion.Label == "prefix" {
			foundPrefix = true
			break
		}
	}

	if !foundPrefix {
		t.Error("Expected 'prefix' completion in scope context")
	}
}

func TestEnumCompletions(t *testing.T) {
	provider := NewCompletionProvider()

	content := `enum Status {
    `
	doc, err := createTestDocumentForCompletion("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position inside enum
	position := protocol.Position{Line: 1, Character: 4}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	// Should have some completions for enum context
	if len(completions) == 0 {
		t.Error("Expected some enum completions")
	}

	// Enum completions should be identifiers/values
	for _, completion := range completions {
		if completion.Kind == nil {
			t.Error("Completion items should have a kind")
		}
	}
}

func TestCompletionWithExistingTypes(t *testing.T) {
	provider := NewCompletionProvider()

	// Document with existing user-defined types
	content := `struct User {
    1: string name
}

service UserService {
    `
	doc, err := createTestDocumentForCompletion("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position in service where return type is expected
	position := protocol.Position{Line: 4, Character: 4}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	// Should suggest the existing User type
	foundUser := false
	for _, completion := range completions {
		if completion.Label == "User" {
			foundUser = true
			break
		}
	}

	if !foundUser {
		t.Error("Expected 'User' type completion based on existing struct")
	}
}

func TestCompletionItemDetails(t *testing.T) {
	provider := NewCompletionProvider()

	doc, err := createTestDocumentForCompletion("file:///test.frugal", "")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	position := protocol.Position{Line: 0, Character: 0}
	completions, err := provider.ProvideCompletion(doc, position)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	if len(completions) == 0 {
		t.Fatal("Expected some completions")
	}

	// Check that completion items have proper structure
	for _, completion := range completions {
		if completion.Label == "" {
			t.Error("Completion label should not be empty")
		}

		if completion.Kind == nil {
			t.Error("Completion should have a kind")
		}

		// Detail and documentation are optional but should be consistent
		if completion.Detail != nil && *completion.Detail == "" {
			t.Error("Completion detail should not be empty string if provided")
		}
	}
}

func TestCompletionPositionHandling(t *testing.T) {
	provider := NewCompletionProvider()

	content := `struct User {
    1: string name
}`

	doc, err := createTestDocumentForCompletion("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	testCases := []struct {
		name          string
		position      protocol.Position
		shouldSucceed bool
	}{
		{
			name:          "valid position",
			position:      protocol.Position{Line: 0, Character: 0},
			shouldSucceed: true,
		},
		{
			name:          "end of line",
			position:      protocol.Position{Line: 0, Character: 12}, // End of "struct User {"
			shouldSucceed: true,
		},
		{
			name:          "beyond last line",
			position:      protocol.Position{Line: 10, Character: 0},
			shouldSucceed: true, // Should handle gracefully - return empty list
		},
		{
			name:          "beyond line end",
			position:      protocol.Position{Line: 0, Character: 1000},
			shouldSucceed: true, // Should handle gracefully
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			completions, err := provider.ProvideCompletion(doc, tc.position)

			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Completion should succeed but got error: %v", err)
				}
				// Completions can be empty for out-of-bounds positions, that's expected
				if completions == nil {
					t.Errorf("Completions should not be nil, even if empty. Got: %v", completions)
				}
			} else {
				if err == nil {
					t.Error("Expected completion to fail")
				}
			}
		})
	}
}

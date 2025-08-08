package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func TestSemanticTokensProvider(t *testing.T) {
	provider := NewSemanticTokensProvider()
	if provider == nil {
		t.Fatal("Semantic tokens provider should not be nil")
	}
}

func TestSemanticTokensLegend(t *testing.T) {
	provider := NewSemanticTokensProvider()
	legend := provider.GetLegend()
	
	// Verify we have expected token types
	expectedTypes := []string{"keyword", "string", "number", "comment", "type", "class", "function"}
	for _, expectedType := range expectedTypes {
		found := false
		for _, tokenType := range legend.TokenTypes {
			if tokenType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected token type '%s' not found in legend", expectedType)
		}
	}
	
	// Verify we have expected modifiers
	expectedModifiers := []string{"declaration", "definition", "readonly"}
	for _, expectedModifier := range expectedModifiers {
		found := false
		for _, modifier := range legend.TokenModifiers {
			if modifier == expectedModifier {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected token modifier '%s' not found in legend", expectedModifier)
		}
	}
}

func TestSemanticTokensForSimpleFrugal(t *testing.T) {
	provider := NewSemanticTokensProvider()
	
	content := `struct User {
    1: string name,
    2: i32 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	doc, err := createTestDocumentForSemanticTokens("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	tokens, err := provider.ProvideSemanticTokens(doc)
	if err != nil {
		t.Fatalf("ProvideSemanticTokens failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected semantic tokens but got nil")
	}

	if len(tokens.Data) == 0 {
		t.Error("Expected semantic token data but got empty array")
	}

	// Verify tokens are properly encoded (should be multiple of 5)
	if len(tokens.Data)%5 != 0 {
		t.Errorf("Token data length should be multiple of 5, got %d", len(tokens.Data))
	}

	t.Logf("Generated %d token data points", len(tokens.Data))
}

func TestSemanticTokensWithKeywords(t *testing.T) {
	provider := NewSemanticTokensProvider()
	
	content := `include "common.frugal"
namespace go example

const string VERSION = "1.0.0"
typedef string UserId

enum Status {
    ACTIVE = 1,
    INACTIVE = 2
}

struct User {
    1: required string name,
    2: optional i32 age
}

exception UserNotFound {
    1: string message
}

service UserService {
    User getUser(1: UserId id) throws (1: UserNotFound notFound),
    oneway void updateUser(1: User user)
}

scope Events prefix "user" {
    UserCreated: User,
    UserDeleted: UserId
}`

	doc, err := createTestDocumentForSemanticTokens("file:///complex.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	tokens, err := provider.ProvideSemanticTokens(doc)
	if err != nil {
		t.Fatalf("ProvideSemanticTokens failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected semantic tokens but got nil")
	}

	if len(tokens.Data) == 0 {
		t.Error("Expected semantic token data but got empty array")
	}

	// Should have many more tokens for this complex example
	if len(tokens.Data) < 50 {
		t.Errorf("Expected at least 50 token data points for complex file, got %d", len(tokens.Data))
	}

	t.Logf("Generated %d token data points for complex file", len(tokens.Data))
}

func TestSemanticTokensRange(t *testing.T) {
	provider := NewSemanticTokensProvider()
	
	content := `struct User {
    1: string name
}`

	doc, err := createTestDocumentForSemanticTokens("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Test range tokens (currently just returns full document tokens)
	rang := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 2, Character: 1},
	}
	
	tokens, err := provider.ProvideSemanticTokensRange(doc, rang)
	if err != nil {
		t.Fatalf("ProvideSemanticTokensRange failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected semantic tokens but got nil")
	}
}

// Test helper to create a document for semantic tokens testing
func createTestDocumentForSemanticTokens(uri, content string) (*document.Document, error) {
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
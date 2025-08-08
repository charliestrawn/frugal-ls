package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func createTestDocumentForHover(uri, content string) (*document.Document, error) {
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

	doc := &document.Document{
		URI:         uri,
		Content:     []byte(content),
		Version:     1,
		ParseResult: result,
		Symbols:     symbols,
	}

	return doc, nil
}

func TestHoverProvider(t *testing.T) {
	provider := NewHoverProvider()
	if provider == nil {
		t.Fatal("Hover provider should not be nil")
	}
}

func TestProvideHoverForStruct(t *testing.T) {
	provider := NewHoverProvider()

	content := `struct User {
    1: string name,
    2: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position at "User" in struct definition (line 0)
	position := protocol.Position{Line: 0, Character: 7}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Fatalf("ProvideHover failed: %v", err)
	}

	if hover == nil {
		t.Fatal("Expected hover information for struct")
	}

	if hover.Contents == nil {
		t.Fatal("Expected hover contents")
	}

	// Check that hover contains useful information
	content_str := ""
	if markupContent, ok := hover.Contents.(protocol.MarkupContent); ok {
		if markupContent.Kind == protocol.MarkupKindMarkdown {
			content_str = markupContent.Value
		}
	}

	if content_str == "" {
		t.Error("Expected non-empty hover content")
	}

	// Should mention it's a struct
	if !strings.Contains(strings.ToLower(content_str), "struct") {
		t.Error("Hover should mention that User is a struct")
	}
}

func TestProvideHoverForService(t *testing.T) {
	provider := NewHoverProvider()

	content := `service UserService {
    User getUser(1: i64 userId)
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position at "UserService" in service definition
	position := protocol.Position{Line: 0, Character: 12}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Fatalf("ProvideHover failed: %v", err)
	}

	if hover == nil {
		t.Fatal("Expected hover information for service")
	}

	if hover.Contents == nil {
		t.Fatal("Expected hover contents")
	}

	content_str := ""
	if markupContent, ok := hover.Contents.(protocol.MarkupContent); ok {
		content_str = markupContent.Value
	}
	if content_str == "" {
		t.Error("Expected non-empty hover content")
	}

	// Should mention it's a service
	if !strings.Contains(strings.ToLower(content_str), "service") {
		t.Error("Hover should mention that UserService is a service")
	}
}

func TestProvideHoverForMethod(t *testing.T) {
	provider := NewHoverProvider()

	content := `struct User {
    1: string name,
    2: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position at method name "getUser"
	position := protocol.Position{Line: 6, Character: 10}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Fatalf("ProvideHover failed: %v", err)
	}

	if hover == nil {
		t.Fatal("Expected hover information for method")
	}

	if hover.Contents == nil {
		t.Fatal("Expected hover contents")
	}

	content_str := ""
	if markupContent, ok := hover.Contents.(protocol.MarkupContent); ok {
		content_str = markupContent.Value
	}
	if content_str == "" {
		t.Error("Expected non-empty hover content")
	}

	// Should mention it's a method and show signature
	if !strings.Contains(strings.ToLower(content_str), "method") && !strings.Contains(strings.ToLower(content_str), "function") {
		t.Error("Hover should mention that getUser is a method/function")
	}
}

func TestProvideHoverForField(t *testing.T) {
	provider := NewHoverProvider()

	content := `struct User {
    1: string name,
    2: i64 id
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position at field name "name"
	position := protocol.Position{Line: 1, Character: 14}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Fatalf("ProvideHover failed: %v", err)
	}

	if hover == nil {
		t.Fatal("Expected hover information for field")
	}

	if hover.Contents == nil {
		t.Fatal("Expected hover contents")
	}

	content_str := ""
	if markupContent, ok := hover.Contents.(protocol.MarkupContent); ok {
		content_str = markupContent.Value
	}
	if content_str == "" {
		t.Error("Expected non-empty hover content")
	}

	// Should mention field information
	if !strings.Contains(strings.ToLower(content_str), "field") {
		t.Error("Hover should mention that name is a field")
	}
}

func TestProvideHoverNoResult(t *testing.T) {
	provider := NewHoverProvider()

	content := `struct User {
    1: string name
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position on whitespace - should not have hover
	position := protocol.Position{Line: 2, Character: 0}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Fatalf("ProvideHover failed: %v", err)
	}

	if hover != nil {
		t.Error("Expected no hover information for whitespace")
	}
}

func TestProvideHoverInvalidPosition(t *testing.T) {
	provider := NewHoverProvider()

	content := `struct User {
    1: string name
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hover, err := provider.ProvideHover(doc, tc.position)
			if err != nil {
				t.Errorf("ProvideHover should handle invalid position gracefully: %v", err)
			}

			// Should return no hover for invalid positions
			if hover != nil {
				t.Error("Expected no hover for invalid position")
			}
		})
	}
}

func TestProvideHoverEmptyDocument(t *testing.T) {
	provider := NewHoverProvider()

	doc, err := createTestDocumentForHover("file:///empty.frugal", "")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	position := protocol.Position{Line: 0, Character: 0}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Errorf("ProvideHover should handle empty document: %v", err)
	}

	if hover != nil {
		t.Error("Expected no hover in empty document")
	}
}

func TestProvideHoverNilDocument(t *testing.T) {
	provider := NewHoverProvider()

	position := protocol.Position{Line: 0, Character: 0}

	// Test with document that has no parse result
	doc := &document.Document{
		URI:         "file:///invalid.frugal",
		Content:     []byte("test"),
		Version:     1,
		ParseResult: nil, // No parse result
	}

	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Errorf("ProvideHover should handle nil parse result: %v", err)
	}

	if hover != nil {
		t.Error("Expected no hover for document without parse result")
	}
}

func TestHoverRange(t *testing.T) {
	provider := NewHoverProvider()

	content := `struct User {
    1: string name
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position at "User" in struct definition
	position := protocol.Position{Line: 0, Character: 7}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Fatalf("ProvideHover failed: %v", err)
	}

	if hover == nil {
		t.Fatal("Expected hover information")
	}

	// Check that hover has a proper range
	if hover.Range == nil {
		t.Error("Expected hover to have a range")
	} else {
		// Range should be reasonable (not negative, within document bounds)
		if hover.Range.Start.Line < 0 || hover.Range.End.Line < hover.Range.Start.Line {
			t.Error("Hover range should be valid")
		}
	}
}

func TestHoverContentFormat(t *testing.T) {
	provider := NewHoverProvider()

	content := `struct User {
    1: string name,
    2: i64 id
}`

	doc, err := createTestDocumentForHover("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	position := protocol.Position{Line: 0, Character: 7}
	hover, err := provider.ProvideHover(doc, position)
	if err != nil {
		t.Fatalf("ProvideHover failed: %v", err)
	}

	if hover == nil {
		t.Fatal("Expected hover information")
	}

	if hover.Contents == nil {
		t.Fatal("Expected hover contents")
	}

	// Check that content format is set appropriately
	if markupContent, ok := hover.Contents.(protocol.MarkupContent); ok {
		// Should be markdown formatted
		if markupContent.Kind != protocol.MarkupKindMarkdown {
			t.Error("Expected hover content to be markdown")
		}

		if markupContent.Value == "" {
			t.Error("Expected hover content to have a value")
		}
	} else {
		t.Error("Expected hover contents to be MarkupContent")
	}
}
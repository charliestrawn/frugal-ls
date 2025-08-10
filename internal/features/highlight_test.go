package features

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
)

func TestNewDocumentHighlightProvider(t *testing.T) {
	provider := NewDocumentHighlightProvider()
	if provider == nil {
		t.Fatal("Expected non-nil DocumentHighlightProvider")
	}
	if provider.referencesProvider == nil {
		t.Fatal("Expected non-nil ReferencesProvider")
	}
}

func TestProvideDocumentHighlightNoParseResult(t *testing.T) {
	doc := &document.Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte("struct User {}"),
		Version:     1,
		ParseResult: nil, // No parse result
	}

	provider := NewDocumentHighlightProvider()
	position := protocol.Position{Line: 0, Character: 7} // On "User"

	highlights, err := provider.ProvideDocumentHighlight(doc, position)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(highlights) != 0 {
		t.Errorf("Expected 0 highlights for document without parse result, got %d", len(highlights))
	}
}

func TestProvideDocumentHighlightBasic(t *testing.T) {
	content := `struct User {
    1: i64 id,
    2: string name
}

service UserService {
    User getUser(1: i64 userId)
}`

	// Create a document with parse result
	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	parseResult, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer parseResult.Close()

	doc := &document.Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte(content),
		Version:     1,
		ParseResult: parseResult,
	}

	provider := NewDocumentHighlightProvider()

	// Test highlighting "User" in struct definition
	position := protocol.Position{Line: 0, Character: 7} // On "User" in struct definition
	highlights, err := provider.ProvideDocumentHighlight(doc, position)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should find at least the definition
	if len(highlights) == 0 {
		t.Error("Expected at least one highlight for User symbol")
	}

	// Verify we have at least one read highlight (for the definition)
	hasReadHighlight := false
	for _, highlight := range highlights {
		if highlight.Kind != nil && *highlight.Kind == protocol.DocumentHighlightKindRead {
			hasReadHighlight = true
			break
		}
	}

	if !hasReadHighlight {
		t.Error("Expected at least one read highlight for User symbol")
	}
}

func TestProvideDocumentHighlightNoSymbol(t *testing.T) {
	content := `struct User {
    1: i64 id
}`

	// Create a document with parse result
	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	parseResult, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer parseResult.Close()

	doc := &document.Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte(content),
		Version:     1,
		ParseResult: parseResult,
	}

	provider := NewDocumentHighlightProvider()

	// Test position on whitespace - should return no highlights
	position := protocol.Position{Line: 0, Character: 0} // On whitespace
	highlights, err := provider.ProvideDocumentHighlight(doc, position)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(highlights) != 0 {
		t.Errorf("Expected 0 highlights for position with no symbol, got %d", len(highlights))
	}
}

func TestProvideDocumentHighlightMultipleOccurrences(t *testing.T) {
	content := `struct User {
    1: i64 id
}

service UserService {
    User getUser(1: i64 userId),
    User createUser(1: User userData)
}`

	// Create a document with parse result
	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	parseResult, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer parseResult.Close()

	doc := &document.Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte(content),
		Version:     1,
		ParseResult: parseResult,
	}

	provider := NewDocumentHighlightProvider()

	// Test highlighting "User" - should find definition and multiple usages
	position := protocol.Position{Line: 0, Character: 7} // On "User" in struct definition
	highlights, err := provider.ProvideDocumentHighlight(doc, position)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should find multiple occurrences of "User"
	if len(highlights) < 2 {
		t.Errorf("Expected at least 2 highlights for User symbol (found %d), content:\n%s", len(highlights), content)
		for i, highlight := range highlights {
			t.Logf("Highlight %d: line %d, char %d-%d", i, highlight.Range.Start.Line, highlight.Range.Start.Character, highlight.Range.End.Character)
		}
	}
}

func TestProvideDocumentHighlightEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		position protocol.Position
		expected int // Expected number of highlights
	}{
		{
			name:     "empty file",
			content:  "",
			position: protocol.Position{Line: 0, Character: 0},
			expected: 0,
		},
		{
			name:     "comment only",
			content:  "// Just a comment",
			position: protocol.Position{Line: 0, Character: 5},
			expected: 0,
		},
		{
			name:     "beyond line bounds",
			content:  "struct User {}",
			position: protocol.Position{Line: 10, Character: 0}, // Line doesn't exist
			expected: 0,
		},
		{
			name:     "beyond character bounds",
			content:  "struct User {}",
			position: protocol.Position{Line: 0, Character: 100}, // Character doesn't exist
			expected: 0,
		},
	}

	provider := NewDocumentHighlightProvider()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create document with parse result if content is not empty
			var doc *document.Document
			if tc.content != "" {
				p, err := parser.NewParser()
				if err != nil {
					t.Fatalf("Failed to create parser: %v", err)
				}
				defer p.Close()

				parseResult, err := p.Parse([]byte(tc.content))
				if err != nil {
					t.Fatalf("Failed to parse: %v", err)
				}
				defer parseResult.Close()

				doc = &document.Document{
					URI:         "file:///test.frugal",
					Path:        "/test.frugal",
					Content:     []byte(tc.content),
					Version:     1,
					ParseResult: parseResult,
				}
			} else {
				doc = &document.Document{
					URI:         "file:///test.frugal",
					Path:        "/test.frugal",
					Content:     []byte(tc.content),
					Version:     1,
					ParseResult: nil,
				}
			}

			highlights, err := provider.ProvideDocumentHighlight(doc, tc.position)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if len(highlights) != tc.expected {
				t.Errorf("Expected %d highlights, got %d", tc.expected, len(highlights))
			}
		})
	}
}

func TestProvideDocumentHighlightTypes(t *testing.T) {
	content := `struct User {
    1: i64 id
}`

	// Create a document with parse result
	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	parseResult, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer parseResult.Close()

	doc := &document.Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte(content),
		Version:     1,
		ParseResult: parseResult,
	}

	provider := NewDocumentHighlightProvider()

	// Test highlighting should return valid highlight types
	position := protocol.Position{Line: 0, Character: 7} // On "User"
	highlights, err := provider.ProvideDocumentHighlight(doc, position)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify all highlights have valid ranges
	for i, highlight := range highlights {
		if highlight.Range.Start.Line > highlight.Range.End.Line {
			t.Errorf("Highlight %d has invalid range: start line %d > end line %d", i, highlight.Range.Start.Line, highlight.Range.End.Line)
		}

		if highlight.Range.Start.Line == highlight.Range.End.Line &&
			highlight.Range.Start.Character >= highlight.Range.End.Character {
			t.Errorf("Highlight %d has invalid range: start char %d >= end char %d", i, highlight.Range.Start.Character, highlight.Range.End.Character)
		}

		// Kind should be Read, Write, or Text if present
		if highlight.Kind != nil {
			validKind := *highlight.Kind == protocol.DocumentHighlightKindRead ||
				*highlight.Kind == protocol.DocumentHighlightKindWrite ||
				*highlight.Kind == protocol.DocumentHighlightKindText
			if !validKind {
				t.Errorf("Highlight %d has invalid kind: %v", i, *highlight.Kind)
			}
		}
	}
}

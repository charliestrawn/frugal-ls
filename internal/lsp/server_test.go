package lsp

import (
	"testing"

	"frugal-ls/internal/document"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestNewServer(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	if server.docManager == nil {
		t.Fatal("Document manager should not be nil")
	}

	if server.includeResolver == nil {
		t.Fatal("Include resolver should not be nil")
	}

	// Check that all feature providers are initialized
	if server.completionProvider == nil {
		t.Error("Completion provider should not be nil")
	}
	if server.hoverProvider == nil {
		t.Error("Hover provider should not be nil")
	}
	if server.documentSymbolProvider == nil {
		t.Error("Document symbol provider should not be nil")
	}
	if server.definitionProvider == nil {
		t.Error("Definition provider should not be nil")
	}
	if server.referencesProvider == nil {
		t.Error("References provider should not be nil")
	}
	if server.documentHighlightProvider == nil {
		t.Error("Document highlight provider should not be nil")
	}
	if server.codeActionProvider == nil {
		t.Error("Code action provider should not be nil")
	}
	if server.formattingProvider == nil {
		t.Error("Formatting provider should not be nil")
	}
}

func TestServerInitialization(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test server capabilities directly
	caps := server.getServerCapabilities()

	if caps.CompletionProvider == nil {
		t.Error("Completion provider capability should be set")
	}
	if caps.HoverProvider == nil {
		t.Error("Hover provider capability should be set")
	}
	if caps.DefinitionProvider == nil {
		t.Error("Definition provider capability should be set")
	}
	if caps.ReferencesProvider == nil {
		t.Error("References provider capability should be set")
	}
	if caps.DocumentSymbolProvider == nil {
		t.Error("Document symbol provider capability should be set")
	}
}

func TestDocumentLifecycle(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	uri := "file:///test.frugal"
	content := `struct User {
    1: string name,
    2: i64 id
}`

	// Test document management directly through document manager
	// to avoid LSP handler complications in unit tests

	// Test document open
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "frugal",
			Version:    1,
			Text:       content,
		},
	}
	_, err = server.docManager.DidOpen(openParams)
	if err != nil {
		t.Fatalf("Document open failed: %v", err)
	}

	// Verify document was added
	doc, exists := server.docManager.GetDocument(uri)
	if !exists || doc == nil {
		t.Fatal("Document should be in manager after opening")
	}

	if string(doc.Content) != content {
		t.Errorf("Document content mismatch. Expected %q, got %q", content, string(doc.Content))
	}

	if doc.Version != 1 {
		t.Errorf("Document version should be 1, got %d", doc.Version)
	}

	// Test document change
	newContent := content + "\n\nservice UserService {\n    User getUser(1: i64 id)\n}"
	changeParams := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEvent{
				Text: newContent,
			},
		},
	}
	_, err = server.docManager.DidChange(changeParams)
	if err != nil {
		t.Fatalf("Document change failed: %v", err)
	}

	// Verify document was updated
	doc, exists = server.docManager.GetDocument(uri)
	if !exists || doc == nil {
		t.Fatal("Document should still exist after change")
	}

	if doc.Version != 2 {
		t.Errorf("Document version should be 2, got %d", doc.Version)
	}

	if string(doc.Content) != newContent {
		t.Errorf("Document content should be updated")
	}

	// Test document close
	closeParams := &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}
	err = server.docManager.DidClose(closeParams)
	if err != nil {
		t.Fatalf("Document close failed: %v", err)
	}

	// Verify document was removed
	doc, exists = server.docManager.GetDocument(uri)
	if exists || doc != nil {
		t.Fatal("Document should be removed after closing")
	}
}

func TestLanguageFeatures(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	uri := "file:///test.frugal"
	content := `struct User {
    1: string name,
    2: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	// Open document through document manager
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "frugal",
			Version:    1,
			Text:       content,
		},
	}
	_, err = server.docManager.DidOpen(openParams)
	if err != nil {
		t.Fatalf("Document open failed: %v", err)
	}

	// Test that document exists and is parsed
	doc, exists := server.docManager.GetDocument(uri)
	if !exists || doc == nil {
		t.Fatal("Document should exist")
	}

	if doc.ParseResult == nil {
		t.Fatal("Document should be parsed")
	}

	if doc.ParseResult.HasErrors() {
		t.Errorf("Document should parse without errors, got: %v", doc.ParseResult.Errors)
	}

	// Test that feature providers can be used (basic smoke tests)
	// More detailed tests should be in their respective feature test files

	// Test references provider
	position := protocol.Position{Line: 0, Character: 7} // On "User"
	allDocs := map[string]*document.Document{uri: doc}
	references, err := server.referencesProvider.ProvideReferences(doc, position, true, allDocs)
	if err != nil {
		t.Errorf("References failed: %v", err)
	}

	// Should find at least the declaration and usage
	if len(references) < 2 {
		t.Errorf("Expected at least 2 references, got %d", len(references))
	}

	// Test hover provider
	hover, err := server.hoverProvider.ProvideHover(doc, position)
	if err != nil {
		t.Errorf("Hover failed: %v", err)
	}
	if hover == nil {
		t.Error("Hover should return content for User struct")
	}

	// Test completion provider
	completionPos := protocol.Position{Line: 1, Character: 10}
	completions, err := server.completionProvider.ProvideCompletion(doc, completionPos)
	if err != nil {
		t.Errorf("Completion failed: %v", err)
	}
	if completions == nil {
		t.Error("Completion result should not be nil")
	}

	// Test document symbols
	symbols, err := server.documentSymbolProvider.ProvideDocumentSymbols(doc)
	if err != nil {
		t.Errorf("Document symbols failed: %v", err)
	}
	if len(symbols) == 0 {
		t.Error("Document should have symbols")
	}
}

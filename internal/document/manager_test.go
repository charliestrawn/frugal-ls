package document

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestNewManager(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	// Should start with no documents
	docs := manager.GetAllDocuments()
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

func TestDocumentDidOpen(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	uri := "file:///test.frugal"
	content := `struct User {
    1: string name,
    2: i64 id
}`

	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "frugal",
			Version:    1,
			Text:       content,
		},
	}

	doc, err := manager.DidOpen(params)
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document should not be nil")
	}

	if doc.URI != uri {
		t.Errorf("Expected URI %s, got %s", uri, doc.URI)
	}

	if string(doc.Content) != content {
		t.Errorf("Expected content %q, got %q", content, string(doc.Content))
	}

	if doc.Version != 1 {
		t.Errorf("Expected version 1, got %d", doc.Version)
	}

	// Verify document is in manager
	retrievedDoc, exists := manager.GetDocument(uri)
	if !exists {
		t.Fatal("Document should exist in manager")
	}

	if retrievedDoc != doc {
		t.Error("Retrieved document should be the same instance")
	}

	// Verify parsing was performed
	if doc.ParseResult == nil {
		t.Fatal("Document should be parsed")
	}

	if doc.ParseResult.HasErrors() {
		t.Errorf("Document should parse without errors, got: %v", doc.ParseResult.Errors)
	}
}

func TestDocumentDidChange(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	uri := "file:///test.frugal"
	initialContent := `struct User {
    1: string name
}`

	// Open document first
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "frugal",
			Version:    1,
			Text:       initialContent,
		},
	}

	doc, err := manager.DidOpen(openParams)
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	// Test full document change
	newContent := `struct User {
    1: string name,
    2: i64 id
}`

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

	updatedDoc, err := manager.DidChange(changeParams)
	if err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}

	if updatedDoc == nil {
		t.Fatal("Updated document should not be nil")
	}

	if string(updatedDoc.Content) != newContent {
		t.Errorf("Expected content %q, got %q", newContent, string(updatedDoc.Content))
	}

	if updatedDoc.Version != 2 {
		t.Errorf("Expected version 2, got %d", updatedDoc.Version)
	}

	// Verify document was re-parsed
	if updatedDoc.ParseResult == nil {
		t.Fatal("Document should be re-parsed after change")
	}

	if updatedDoc.ParseResult.HasErrors() {
		t.Errorf("Updated document should parse without errors, got: %v", updatedDoc.ParseResult.Errors)
	}

	// Verify it's the same document instance
	if updatedDoc != doc {
		t.Error("Should update the same document instance")
	}
}

func TestDocumentDidClose(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	uri := "file:///test.frugal"
	content := `struct User {
    1: string name,
    2: i64 id
}`

	// Open document first
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "frugal",
			Version:    1,
			Text:       content,
		},
	}

	_, err = manager.DidOpen(openParams)
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	// Verify document exists
	_, exists := manager.GetDocument(uri)
	if !exists {
		t.Fatal("Document should exist before close")
	}

	// Close document
	closeParams := &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}

	err = manager.DidClose(closeParams)
	if err != nil {
		t.Fatalf("DidClose failed: %v", err)
	}

	// Verify document was removed
	_, exists = manager.GetDocument(uri)
	if exists {
		t.Error("Document should not exist after close")
	}

	// Verify no documents remain
	allDocs := manager.GetAllDocuments()
	if len(allDocs) != 0 {
		t.Errorf("Expected 0 documents after close, got %d", len(allDocs))
	}
}

func TestDocumentParsingErrors(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	uri := "file:///test.frugal"
	// Invalid content with syntax errors
	invalidContent := `struct User {
    1: string name
    // Missing comma and closing brace
    2: i64 id`

	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "frugal",
			Version:    1,
			Text:       invalidContent,
		},
	}

	doc, err := manager.DidOpen(params)
	if err != nil {
		t.Fatalf("DidOpen should not fail even with invalid syntax: %v", err)
	}

	// Document should exist but have parsing errors
	if doc.ParseResult == nil {
		t.Fatal("Document should have parse result even with errors")
	}

	if !doc.ParseResult.HasErrors() {
		t.Error("Document should have parsing errors")
	}

	if len(doc.ParseResult.Errors) == 0 {
		t.Error("Document should report specific parsing errors")
	}
}

func TestMultipleDocuments(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Open multiple documents
	documents := []struct {
		uri     string
		content string
	}{
		{"file:///user.frugal", `struct User { 1: string name }`},
		{"file:///service.frugal", `service UserService { User getUser() }`},
		{"file:///types.frugal", `typedef string UserId`},
	}

	for i, docInfo := range documents {
		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        docInfo.uri,
				LanguageID: "frugal",
				Version:    int32(i + 1),
				Text:       docInfo.content,
			},
		}

		doc, err := manager.DidOpen(params)
		if err != nil {
			t.Fatalf("Failed to open document %s: %v", docInfo.uri, err)
		}

		if doc == nil {
			t.Fatalf("Document %s should not be nil", docInfo.uri)
		}
	}

	// Verify all documents exist
	allDocs := manager.GetAllDocuments()
	if len(allDocs) != len(documents) {
		t.Errorf("Expected %d documents, got %d", len(documents), len(allDocs))
	}

	for _, docInfo := range documents {
		doc, exists := manager.GetDocument(docInfo.uri)
		if !exists {
			t.Errorf("Document %s should exist", docInfo.uri)
			continue
		}

		if string(doc.Content) != docInfo.content {
			t.Errorf("Document %s content mismatch", docInfo.uri)
		}
	}

	// Close one document
	closeParams := &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: documents[0].uri},
	}

	err = manager.DidClose(closeParams)
	if err != nil {
		t.Fatalf("Failed to close document: %v", err)
	}

	// Verify only that document was removed
	allDocs = manager.GetAllDocuments()
	if len(allDocs) != len(documents)-1 {
		t.Errorf("Expected %d documents after close, got %d", len(documents)-1, len(allDocs))
	}

	_, exists := manager.GetDocument(documents[0].uri)
	if exists {
		t.Errorf("Document %s should not exist after close", documents[0].uri)
	}

	// Other documents should still exist
	for i := 1; i < len(documents); i++ {
		_, exists := manager.GetDocument(documents[i].uri)
		if !exists {
			t.Errorf("Document %s should still exist", documents[i].uri)
		}
	}
}

func TestDocumentURIPath(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	testCases := []struct {
		name        string
		uri         string
		expectedPath string
	}{
		{
			name:        "unix path",
			uri:         "file:///home/user/test.frugal",
			expectedPath: "/home/user/test.frugal",
		},
		{
			name:        "relative-like path",
			uri:         "file:///test.frugal",
			expectedPath: "/test.frugal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        tc.uri,
					LanguageID: "frugal",
					Version:    1,
					Text:       "struct Test {}",
				},
			}

			doc, err := manager.DidOpen(params)
			if err != nil {
				t.Fatalf("DidOpen failed: %v", err)
			}

			if doc.Path != tc.expectedPath {
				t.Errorf("Expected path %s, got %s", tc.expectedPath, doc.Path)
			}
		})
	}
}
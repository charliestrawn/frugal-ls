package document

import (
	"strings"
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

func TestDocumentDidChangeIncremental(t *testing.T) {
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

	_, err = manager.DidOpen(openParams)
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	// Test incremental change - add a comma after "name"
	// The content is: "struct User {\n    1: string name\n}"
	// Line 1 is "    1: string name" (0-indexed), so "name" ends at character 16 (0-indexed)
	changeParams := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEvent{
				Range: &protocol.Range{
					Start: protocol.Position{Line: 1, Character: 18}, // After "name"
					End:   protocol.Position{Line: 1, Character: 18},
				},
				Text: ",",
			},
		},
	}

	updatedDoc, err := manager.DidChange(changeParams)
	if err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}

	expectedContent := `struct User {
    1: string name,
}`

	if string(updatedDoc.Content) != expectedContent {
		t.Errorf("Incremental change failed.\nExpected:\n%q\nGot:\n%q", expectedContent, string(updatedDoc.Content))
	}

	// Test incremental change - add a new field
	// Current content is: "struct User {\n    1: string name,\n}"
	// Line 1 is "    1: string name," so after the comma is character 17
	changeParams2 := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                3,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEvent{
				Range: &protocol.Range{
					Start: protocol.Position{Line: 1, Character: 19}, // After "name,"
					End:   protocol.Position{Line: 1, Character: 19},
				},
				Text: "\n    2: i64 id",
			},
		},
	}

	updatedDoc2, err := manager.DidChange(changeParams2)
	if err != nil {
		t.Fatalf("Second DidChange failed: %v", err)
	}

	expectedContent2 := `struct User {
    1: string name,
    2: i64 id
}`

	if string(updatedDoc2.Content) != expectedContent2 {
		t.Errorf("Second incremental change failed.\nExpected:\n%q\nGot:\n%q", expectedContent2, string(updatedDoc2.Content))
	}

	// Verify document was re-parsed
	if updatedDoc2.ParseResult == nil {
		t.Fatal("Document should be re-parsed after incremental change")
	}
}

func TestDocumentDidChangeRealWorldTyping(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	uri := "file:///test.frugal"
	initialContent := `struct User {
    1: string name
}`

	// Open document
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "frugal",
			Version:    1,
			Text:       initialContent,
		},
	}

	_, err = manager.DidOpen(openParams)
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	// Simulate typing "struct NewType {" at the end of the document
	// This is a common scenario that was causing issues
	changeParams := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEvent{
				Range: &protocol.Range{
					Start: protocol.Position{Line: 2, Character: 1}, // After the closing brace
					End:   protocol.Position{Line: 2, Character: 1},
				},
				Text: "\n\nstruct NewType {\n    1: i32 id\n}",
			},
		},
	}

	updatedDoc, err := manager.DidChange(changeParams)
	if err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}

	expectedContent := `struct User {
    1: string name
}

struct NewType {
    1: i32 id
}`

	if string(updatedDoc.Content) != expectedContent {
		t.Errorf("Real-world typing simulation failed.\nExpected:\n%q\nGot:\n%q", expectedContent, string(updatedDoc.Content))
	}

	// Test typing character by character (more realistic)
	// Start with clean state
	openParams2 := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///test2.frugal",
			LanguageID: "frugal",
			Version:    1,
			Text:       "",
		},
	}

	doc2, err := manager.DidOpen(openParams2)
	if err != nil {
		t.Fatalf("DidOpen failed for test2: %v", err)
	}
	
	// Verify initial empty content
	if string(doc2.Content) != "" {
		t.Errorf("Expected empty initial content, got %q", string(doc2.Content))
	}

	// Type "struct " character by character
	chars := []string{"s", "t", "r", "u", "c", "t", " "}
	for i, char := range chars {
		changeParams := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test2.frugal"},
				Version:                int32(i + 2),
			},
			ContentChanges: []any{
				protocol.TextDocumentContentChangeEvent{
					Range: &protocol.Range{
						Start: protocol.Position{Line: 0, Character: uint32(i)},
						End:   protocol.Position{Line: 0, Character: uint32(i)},
					},
					Text: char,
				},
			},
		}

		doc2, err = manager.DidChange(changeParams)
		if err != nil {
			t.Fatalf("Character-by-character typing failed at char %d (%s): %v", i, char, err)
		}

		expectedPartial := strings.Join(chars[:i+1], "")
		if string(doc2.Content) != expectedPartial {
			t.Errorf("Character-by-character typing failed at step %d.\nExpected: %q\nGot: %q", i, expectedPartial, string(doc2.Content))
		}
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
		name         string
		uri          string
		expectedPath string
	}{
		{
			name:         "unix path",
			uri:          "file:///home/user/test.frugal",
			expectedPath: "/home/user/test.frugal",
		},
		{
			name:         "relative-like path",
			uri:          "file:///test.frugal",
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

func TestDocumentNilParseResult(t *testing.T) {
	// Create a document with no parse result
	doc := &Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte("test content"),
		Version:     1,
		ParseResult: nil, // No parse result
	}

	// Ensure we're using the fallback path by not setting global provider
	globalDiagnosticsProvider = nil

	diagnostics := doc.GetDiagnostics()

	// Should return empty array, not nil - this is critical for LSP protocol compliance
	if diagnostics == nil {
		t.Error("GetDiagnostics should return empty array, not nil - this violates LSP protocol")
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected empty diagnostics array, got %d diagnostics", len(diagnostics))
	}
}

func TestGetBasicParseErrorDiagnosticsNilParseResult(t *testing.T) {
	// Create a document with no parse result
	doc := &Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte("test content"),
		Version:     1,
		ParseResult: nil, // No parse result
	}

	diagnostics := doc.getBasicParseErrorDiagnostics()

	// Should return empty array, not nil
	if diagnostics == nil {
		t.Error("getBasicParseErrorDiagnostics should return empty array, not nil")
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected empty diagnostics array, got %d diagnostics", len(diagnostics))
	}
}

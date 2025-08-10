package features

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
)

func TestNewFormattingProvider(t *testing.T) {
	provider := NewFormattingProvider()
	if provider == nil {
		t.Fatal("Expected non-nil FormattingProvider")
	}
}

func TestProvideDocumentFormattingBasic(t *testing.T) {
	content := `struct User {
1:i64 id,
  2:   string name
}`

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("Expected 1 edit, got %d", len(edits))
	}

	expectedFormatted := `struct User {
    1: i64 id,
    2: string name
}`

	if edits[0].NewText != expectedFormatted {
		t.Errorf("Expected formatted content:\n%s\nGot:\n%s", expectedFormatted, edits[0].NewText)
	}
}

func TestProvideDocumentFormattingWithComments(t *testing.T) {
	content := `// Header comment
struct User {
1:i64 id,//User ID
  2:   string name // Full name
}`

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("Expected 1 edit, got %d", len(edits))
	}

	// Should preserve comments and format structure
	if !containsSubstring(edits[0].NewText, "// Header comment") {
		t.Error("Expected to preserve header comment")
	}
	if !containsSubstring(edits[0].NewText, "//User ID") {
		t.Error("Expected to preserve inline comment")
	}
	if !containsSubstring(edits[0].NewText, "// Full name") {
		t.Error("Expected to preserve field comment")
	}
}

func TestProvideDocumentFormattingParentheses(t *testing.T) {
	content := `service TestService {
User getUser( 1: i64 id ),
void updateUser(1:User user   )
}`

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("Expected 1 edit, got %d", len(edits))
	}

	// Check that spaces inside parentheses are trimmed
	formatted := edits[0].NewText
	if containsSubstring(formatted, "( ") {
		t.Error("Expected spaces after opening parentheses to be trimmed")
	}
	if containsSubstring(formatted, " )") {
		t.Error("Expected spaces before closing parentheses to be trimmed")
	}
	
	// Check proper indentation
	if !containsSubstring(formatted, "    User getUser(") {
		t.Error("Expected proper indentation for method")
	}
}

func TestProvideDocumentFormattingTabsToSpaces(t *testing.T) {
	content := "struct User {\n\t1:\ti64\tid\n}"

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      2,
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("Expected 1 edit, got %d", len(edits))
	}

	expectedFormatted := `struct User {
  1: i64 id
}`

	if edits[0].NewText != expectedFormatted {
		t.Errorf("Expected formatted content:\n%s\nGot:\n%s", expectedFormatted, edits[0].NewText)
	}
}

func TestProvideDocumentFormattingPreserveTabs(t *testing.T) {
	content := "struct User {\n1: i64 id\n}"

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": false, // Use tabs
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("Expected 1 edit, got %d", len(edits))
	}

	expectedFormatted := "struct User {\n\t1: i64 id\n}"

	if edits[0].NewText != expectedFormatted {
		t.Errorf("Expected formatted content:\n%s\nGot:\n%s", expectedFormatted, edits[0].NewText)
	}
}

func TestProvideDocumentFormattingNoChanges(t *testing.T) {
	content := `struct User {
    1: i64 id,
    2: string name
}`

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return no edits if content is already properly formatted
	if len(edits) != 0 {
		t.Errorf("Expected 0 edits for already formatted content, got %d", len(edits))
	}
}

func TestProvideDocumentFormattingInvalidFile(t *testing.T) {
	doc := &document.Document{
		URI:     "file:///test.txt", // Not a .frugal file
		Path:    "/test.txt",
		Content: []byte("not frugal content"),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return no edits for invalid Frugal files
	if edits != nil {
		t.Errorf("Expected nil edits for invalid file, got %v", edits)
	}
}

func TestProvideDocumentFormattingNestedStructures(t *testing.T) {
	content := `service TestService {
struct NestedStruct {
1: string field
}
NestedStruct getData()
}`

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("Expected 1 edit, got %d", len(edits))
	}

	expectedFormatted := `service TestService {
    struct NestedStruct {
        1: string field
    }
    NestedStruct getData()
}`

	if edits[0].NewText != expectedFormatted {
		t.Errorf("Expected formatted content:\n%s\nGot:\n%s", expectedFormatted, edits[0].NewText)
	}
}

func TestProvideDocumentRangeFormatting(t *testing.T) {
	content := `struct User {
1:i64 id,
  2:   string name
}`

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(content),
		Version: 1,
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	// Range formatting should fall back to document formatting
	rng := protocol.Range{
		Start: protocol.Position{Line: 1, Character: 0},
		End:   protocol.Position{Line: 2, Character: 0},
	}

	edits, err := provider.ProvideDocumentRangeFormatting(doc, rng, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("Expected 1 edit, got %d", len(edits))
	}

	// Should format entire document
	expectedFormatted := `struct User {
    1: i64 id,
    2: string name
}`

	if edits[0].NewText != expectedFormatted {
		t.Errorf("Expected formatted content:\n%s\nGot:\n%s", expectedFormatted, edits[0].NewText)
	}
}

func TestFormattingProviderEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty file",
			content:  "",
			expected: "",
		},
		{
			name:     "only comments",
			content:  "// Just a comment\n/* Another comment */",
			expected: "// Just a comment\n/* Another comment */",
		},
		{
			name:     "multiple empty lines",
			content:  "struct User {\n\n\n    1: i64 id\n\n}",
			expected: "struct User {\n\n\n    1: i64 id\n\n}",
		},
		{
			name:     "mixed spaces and tabs",
			content:  "struct User {\n\t 1: i64 id\n}",
			expected: "struct User {\n    1: i64 id\n}",
		},
	}

	provider := NewFormattingProvider()
	options := protocol.FormattingOptions{
		"tabSize":      4,
		"insertSpaces": true,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := &document.Document{
				URI:     "file:///test.frugal",
				Path:    "/test.frugal",
				Content: []byte(tc.content),
				Version: 1,
			}

			edits, err := provider.ProvideDocumentFormatting(doc, options)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			var result string
			if len(edits) == 0 {
				result = tc.content
			} else {
				result = edits[0].NewText
			}

			if result != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
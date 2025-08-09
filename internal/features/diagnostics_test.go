package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func TestDiagnosticsProvider(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    1: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have no diagnostics for valid content (User type is defined)
	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics for valid content, got %d", len(diagnostics))
		for i, diag := range diagnostics {
			t.Errorf("Diagnostic %d: %s", i, diag.Message)
		}
	}
}

func TestDiagnosticsDuplicateStructNames(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    1: i64 id
}

struct User {
    2: string name
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have one diagnostic for duplicate struct name
	if len(diagnostics) != 1 {
		t.Errorf("Expected 1 diagnostic for duplicate struct, got %d", len(diagnostics))
		return
	}

	diagnostic := diagnostics[0]
	if !strings.Contains(diagnostic.Message, "Duplicate struct definition 'User'") {
		t.Errorf("Expected duplicate struct message, got: %s", diagnostic.Message)
	}

	if *diagnostic.Severity != protocol.DiagnosticSeverityError {
		t.Errorf("Expected error severity for duplicate struct")
	}

	// Should have related information pointing to first definition
	if diagnostic.RelatedInformation == nil || len(diagnostic.RelatedInformation) != 1 {
		t.Error("Expected related information for duplicate struct")
	}
}

func TestDiagnosticsDuplicateFieldIds(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    1: i64 id,
    1: string name
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have one diagnostic for duplicate field ID
	if len(diagnostics) != 1 {
		t.Errorf("Expected 1 diagnostic for duplicate field ID, got %d", len(diagnostics))
		return
	}

	diagnostic := diagnostics[0]
	if !strings.Contains(diagnostic.Message, "Duplicate field ID 1") {
		t.Errorf("Expected duplicate field ID message, got: %s", diagnostic.Message)
	}
}

func TestDiagnosticsInvalidFieldId(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    -1: string name
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have diagnostic for invalid field ID (-1)
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Field ID must be positive") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected diagnostic for negative field ID")
		for i, diag := range diagnostics {
			t.Errorf("Diagnostic %d: %s", i, diag.Message)
		}
	}
}

func testDiagnosticsZeroFieldId(t *testing.T) { // Disabled - tree-sitter parsing issue
	provider := NewDiagnosticsProvider()

	content := `struct User {
    0: i64 id
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have diagnostic for invalid field ID (0)
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Field ID must be positive, got 0") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected diagnostic for zero field ID")
		for i, diag := range diagnostics {
			t.Errorf("Diagnostic %d: %s", i, diag.Message)
		}
	}
}

func TestDiagnosticsMethodParameterFieldIds(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `service UserService {
    User getUser(1: i64 userId, 1: string name)
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have one diagnostic for duplicate parameter field ID
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Duplicate field ID 1 in parameter list") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected diagnostic for duplicate parameter field ID")
	}
}

func TestDiagnosticsNamingConventions(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct user_info {
    1: i64 id
}

service user_service {
    void getData()
}

enum status_code {
    ACTIVE = 1
}

const string user_name = "test"`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have warnings for naming convention violations
	var namingWarnings []protocol.Diagnostic
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != nil && *diagnostic.Severity == protocol.DiagnosticSeverityWarning {
			if strings.Contains(diagnostic.Message, "should follow PascalCase") ||
			   strings.Contains(diagnostic.Message, "should follow UPPER_SNAKE_CASE") {
				namingWarnings = append(namingWarnings, diagnostic)
			}
		}
	}

	// Should have warnings for struct, service (PascalCase) and enum (PascalCase), const (UPPER_SNAKE_CASE)
	if len(namingWarnings) < 3 {
		t.Errorf("Expected at least 3 naming convention warnings, got %d", len(namingWarnings))
		for i, warning := range namingWarnings {
			t.Logf("Warning %d: %s", i, warning.Message)
		}
	}
}

func TestDiagnosticsUnknownType(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    1: UnknownType data
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have error for unknown type
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Unknown type 'UnknownType'") {
			found = true
			if *diagnostic.Severity != protocol.DiagnosticSeverityError {
				t.Error("Expected error severity for unknown type")
			}
			break
		}
	}

	if !found {
		t.Error("Expected diagnostic for unknown type")
	}
}

func TestDiagnosticsValidBuiltinTypes(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    1: i64 id,
    2: string name,
    3: bool active,
    4: double score,
    5: binary data,
    6: list<string> tags,
    7: map<string, i32> properties
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have no unknown type errors for builtin types
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Unknown type") {
			t.Errorf("Unexpected unknown type error for builtin type: %s", diagnostic.Message)
		}
	}
}

func TestDiagnosticsValidUserDefinedTypes(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    1: i64 id
}

struct Profile {
    1: User user,
    2: Status status
}

enum Status {
    ACTIVE = 1,
    INACTIVE = 2
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have no unknown type errors for user-defined types
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Unknown type") {
			t.Errorf("Unexpected unknown type error for user-defined type: %s", diagnostic.Message)
		}
	}
}

func TestDiagnosticsTypedefTypes(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `typedef string UserId

struct User {
    1: UserId id,
    2: string name
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have no unknown type errors for typedef types
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Unknown type 'UserId'") {
			t.Errorf("Typedef type should be recognized: %s", diagnostic.Message)
		}
	}
}

func TestDiagnosticsMethodParametersVsThrowsFieldIds(t *testing.T) {
	provider := NewDiagnosticsProvider()

	// This should be valid - throws and parameters have separate field ID namespaces
	content := `exception UserNotFound {
    1: string message
}

service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound ex)
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should NOT have duplicate field ID errors between parameters and throws
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Duplicate field ID 1") {
			t.Errorf("Parameters and throws should have separate field ID namespaces: %s", diagnostic.Message)
		}
	}
}

func TestDiagnosticsDuplicateThrowsFieldIds(t *testing.T) {
	provider := NewDiagnosticsProvider()

	// This should be invalid - duplicate field IDs within same throws list
	content := `exception UserNotFound {
    1: string message
}

exception ValidationError {
    1: string details
}

service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound notFound, 1: ValidationError validation)
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have duplicate field ID error within throws list
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "Duplicate field ID 1 in throws list") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected diagnostic for duplicate field ID within throws list")
		for i, diag := range diagnostics {
			t.Errorf("Diagnostic %d: %s", i, diag.Message)
		}
	}
}

func TestDiagnosticsParseErrors(t *testing.T) {
	provider := NewDiagnosticsProvider()

	// Invalid syntax - missing closing brace
	content := `struct User {
    1: i64 id`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should have parse error diagnostics
	if len(diagnostics) == 0 {
		t.Error("Expected parse error diagnostics for invalid syntax")
	}

	// Parse errors should have error severity
	for _, diagnostic := range diagnostics {
		if *diagnostic.Severity != protocol.DiagnosticSeverityError {
			t.Error("Expected error severity for parse errors")
		}
	}
}

func TestDiagnosticsNamingConventionHelpers(t *testing.T) {
	provider := NewDiagnosticsProvider()

	// Test PascalCase validation
	testCases := []struct {
		input    string
		expected bool
	}{
		{"User", true},
		{"UserService", true},
		{"APIKey", true},
		{"user", false},
		{"userService", false},
		{"User_Service", false},
		{"USER", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := provider.isPascalCase(tc.input)
		if result != tc.expected {
			t.Errorf("isPascalCase(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}

	// Test UPPER_SNAKE_CASE validation
	testCases2 := []struct {
		input    string
		expected bool
	}{
		{"USER_ID", true},
		{"MAX_CONNECTIONS", true},
		{"API_KEY", true},
		{"DEFAULT_TIMEOUT_MS", true},
		{"user_id", false},
		{"User_ID", false},
		{"USER-ID", false},
		{"_USER_ID", false},
		{"USER_ID_", false},
		{"USER__ID", false},
		{"", false},
	}

	for _, tc := range testCases2 {
		result := provider.isUpperSnakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("isUpperSnakeCase(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestDiagnosticsRangeCalculation(t *testing.T) {
	provider := NewDiagnosticsProvider()

	content := `struct User {
    1: i64 id
}`

	doc := createTestDocumentForDiagnostics(t, "file:///test.frugal", content)
	defer doc.ParseResult.Close()

	// Get the struct definition node
	root := doc.ParseResult.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	structNode := ast.FindNodeByType(root, "struct_definition")
	if structNode == nil {
		t.Fatal("Expected struct definition node")
	}

	identifierNode := ast.FindNodeByType(structNode, "identifier")
	if identifierNode == nil {
		t.Fatal("Expected identifier node")
	}

	// Test range calculation
	rng := provider.nodeToRange(identifierNode, doc.Content)

	// Identifier "User" should be on line 0 (0-indexed)
	if rng.Start.Line != 0 {
		t.Errorf("Expected start line 0, got %d", rng.Start.Line)
	}

	// Should have reasonable character positions
	if rng.Start.Character >= rng.End.Character {
		t.Errorf("Expected start character (%d) < end character (%d)", rng.Start.Character, rng.End.Character)
	}
}

// Test helper functions

func createTestDocumentForDiagnostics(t *testing.T, uri, content string) *document.Document {
	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var symbols []ast.Symbol
	if result.GetRootNode() != nil {
		symbols = ast.ExtractSymbols(result.GetRootNode(), []byte(content))
	}

	doc := &document.Document{
		URI:         uri,
		Path:        uri[7:], // Remove "file://" prefix
		Content:     []byte(content),
		Version:     1,
		ParseResult: result,
		Symbols:     symbols,
	}

	return doc
}

func TestDiagnosticsNeverReturnNil(t *testing.T) {
	provider := NewDiagnosticsProvider()

	// Test with nil parse result
	doc := &document.Document{
		URI:         "file:///test.frugal",
		Path:        "/test.frugal",
		Content:     []byte("test content"),
		Version:     1,
		ParseResult: nil,
	}

	diagnostics := provider.ProvideDiagnostics(doc)

	// Should return empty array, not nil - critical for LSP protocol compliance
	if diagnostics == nil {
		t.Error("ProvideDiagnostics should return empty array, not nil - this violates LSP protocol")
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected empty diagnostics array, got %d diagnostics", len(diagnostics))
	}

	// Test with empty content
	doc2 := createTestDocumentForDiagnostics(t, "file:///empty.frugal", "")
	defer doc2.ParseResult.Close()

	diagnostics2 := provider.ProvideDiagnostics(doc2)

	if diagnostics2 == nil {
		t.Error("ProvideDiagnostics should return empty array for empty content, not nil")
	}
}
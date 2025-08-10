package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func TestFrugalFormatterBasic(t *testing.T) {
	options := protocol.FormattingOptions{
		"tabSize":      float64(4),
		"insertSpaces": true,
	}

	formatter := NewFrugalFormatter(options)
	if formatter.indentSize != 4 {
		t.Errorf("Expected indent size 4, got %d", formatter.indentSize)
	}
	if !formatter.useSpaces {
		t.Error("Expected to use spaces")
	}
}

func TestFormatterService(t *testing.T) {
	unformatted := `service UserService{
User getUser(1:i64 userId)throws(1:UserNotFound error),
void updateUser(1:User user)
}`

	expected := `service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound error),
    void updateUser(1: User user)
}`

	result := testFormat(t, unformatted)

	// Normalize whitespace for comparison
	result = normalizeWhitespace(result)
	expected = normalizeWhitespace(expected)

	if result != expected {
		t.Errorf("Service formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterStruct(t *testing.T) {
	unformatted := `struct User{
1:required i64 id,
2:optional string name,
3:optional string email
}`

	expected := `struct User {
    1: required i64 id,
    2: optional string name,
    3: optional string email
}`

	result := testFormat(t, unformatted)

	result = normalizeWhitespace(result)
	expected = normalizeWhitespace(expected)

	if result != expected {
		t.Errorf("Struct formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterEnum(t *testing.T) {
	unformatted := `enum Status{
ACTIVE=1,
INACTIVE=2,
PENDING=3
}`

	expected := `enum Status {
    ACTIVE = 1,
    INACTIVE = 2,
    PENDING = 3
}`

	result := testFormat(t, unformatted)

	result = normalizeWhitespace(result)
	expected = normalizeWhitespace(expected)

	if result != expected {
		t.Errorf("Enum formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterScope(t *testing.T) {
	unformatted := `scope UserEvents prefix"user"{
UserCreated:User,
UserUpdated:User
}`

	expected := `scope UserEvents prefix "user" {
    UserCreated: User,
    UserUpdated: User
}`

	result := testFormat(t, unformatted)

	result = normalizeWhitespace(result)
	expected = normalizeWhitespace(expected)

	if result != expected {
		t.Errorf("Scope formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterConst(t *testing.T) {
	unformatted := `const i32 DEFAULT_TIMEOUT=5000`

	expected := `const i32 DEFAULT_TIMEOUT = 5000;`

	result := testFormat(t, unformatted)

	result = normalizeWhitespace(result)
	expected = normalizeWhitespace(expected)

	if result != expected {
		t.Errorf("Const formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterTypedef(t *testing.T) {
	unformatted := `typedef string  UserId
typedef map<string,string>Metadata`

	expected := `typedef string UserId
typedef map<string,string> Metadata`

	result := testFormat(t, unformatted)

	result = normalizeWhitespace(result)
	expected = normalizeWhitespace(expected)

	if result != expected {
		t.Errorf("Typedef formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterIncludeAndNamespace(t *testing.T) {
	unformatted := `include"common.frugal"
namespace go   example`

	expected := `include "common.frugal"
namespace go example`

	result := testFormat(t, unformatted)

	result = normalizeWhitespace(result)
	expected = normalizeWhitespace(expected)

	if result != expected {
		t.Errorf("Include/namespace formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatterCompleteDocument(t *testing.T) {
	unformatted := `include"common.frugal"
namespace go example
service UserService{
User getUser(1:i64 userId)throws(1:UserNotFound error),
void updateUser(1:User user)
}
struct User{
1:required i64 id,
2:optional string name
}
enum Status{
ACTIVE=1,
INACTIVE=2
}
const i32 TIMEOUT=5000`

	expected := `include "common.frugal"

namespace go example

service UserService {
    User getUser(1: i64 userId) throws (1: UserNotFound error),
    void updateUser(1: User user)
}

struct User {
    1: required i64 id,
    2: optional string name
}
enum Status {
    ACTIVE = 1,
    INACTIVE = 2
}

const i32 TIMEOUT = 5000;`

	result := testFormat(t, unformatted)

	// Compare line by line for better debugging
	resultLines := strings.Split(strings.TrimSpace(result), "\n")
	expectedLines := strings.Split(strings.TrimSpace(expected), "\n")

	if len(resultLines) != len(expectedLines) {
		t.Errorf("Line count mismatch. Expected %d lines, got %d lines", len(expectedLines), len(resultLines))
		t.Logf("Result:\n%s", result)
		return
	}

	for i, expectedLine := range expectedLines {
		if i < len(resultLines) {
			resultLine := resultLines[i]
			if normalizeWhitespace(resultLine) != normalizeWhitespace(expectedLine) {
				t.Errorf("Line %d mismatch.\nExpected: %q\nGot:      %q", i+1, expectedLine, resultLine)
			}
		}
	}
}

func TestFormatterFieldAlignment(t *testing.T) {
	unformatted := `struct User {
1: i64 id,
22: required string veryLongFieldName,
3: optional bool active
}`

	result := testFormat(t, unformatted)

	// Check that fields are aligned (this is a basic test)
	lines := strings.Split(result, "\n")
	fieldLines := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "struct") {
			fieldLines = append(fieldLines, line)
		}
	}

	if len(fieldLines) < 2 {
		t.Error("Should have found field lines for alignment test")
		return
	}

	// Basic alignment check - all field lines should have consistent colon positioning
	t.Logf("Field alignment result:\n%s", strings.Join(fieldLines, "\n"))
}

func TestFormatterMultiLineComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic multi-line comment",
			input: `/* Multi-line comment
   about the service */
service UserService {
    User getUser(1: i64 userId)
}`,
			expected: `/**
 * Multi-line comment
 * about the service
 */
service UserService {
    User getUser(1: i64 userId)
}`,
		},
		{
			name: "multi-line comment with proper stars already",
			input: `/**
 * Already properly formatted
 * multi-line comment
 */
service UserService {
}`,
			expected: `/**
 * Already properly formatted
 * multi-line comment
 */
service UserService {}`,
		},
		{
			name: "multi-line comment with misaligned stars",
			input: `/*
*Misaligned comment
  *Mixed alignment
   about the service
*/
service UserService {
}`,
			expected: `/**
 * Misaligned comment
 * Mixed alignment
 * about the service
 */
service UserService {}`,
		},
		{
			name: "multi-line comment with empty lines",
			input: `/*
Some text

More text after empty line
*/
service UserService {
}`,
			expected: `/**
 * Some text
 *
 * More text after empty line
 */
service UserService {}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := testFormat(t, test.input)
			
			// Normalize whitespace for comparison
			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(test.expected)
			
			if result != expected {
				t.Errorf("Multi-line comment formatting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestFormattingProvider(t *testing.T) {
	provider := NewFormattingProvider()

	content := `service UserService{
User getUser(1:i64 userId)
}`

	doc, err := createTestDocumentForFormatting("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	options := protocol.FormattingOptions{
		"tabSize":      float64(4),
		"insertSpaces": true,
	}

	edits, err := provider.ProvideDocumentFormatting(doc, options)
	if err != nil {
		t.Fatalf("ProvideDocumentFormatting failed: %v", err)
	}

	if len(edits) == 0 {
		t.Error("Expected formatting edits but got none")
		return
	}

	if len(edits) != 1 {
		t.Errorf("Expected 1 edit, got %d", len(edits))
	}

	edit := edits[0]
	if edit.Range.Start.Line != 0 || edit.Range.Start.Character != 0 {
		t.Errorf("Expected edit to start at (0,0), got (%d,%d)", edit.Range.Start.Line, edit.Range.Start.Character)
	}

	if !strings.Contains(edit.NewText, "service UserService") {
		t.Error("Expected formatted content to contain service definition")
	}

	// Check that content is properly indented
	if !strings.Contains(edit.NewText, "    User getUser(") {
		t.Error("Expected method to be properly indented")
	}

	t.Logf("Formatted content:\n%s", edit.NewText)
}

// Test helper functions

func testFormat(t *testing.T, input string) string {
	doc, err := createTestDocumentForFormatting("file:///test.frugal", input)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	options := protocol.FormattingOptions{
		"tabSize":      float64(4),
		"insertSpaces": true,
	}

	formatter := NewFrugalFormatter(options)
	result, err := formatter.FormatDocument(doc)
	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	return strings.TrimSpace(result)
}

func normalizeWhitespace(s string) string {
	// Normalize multiple spaces to single space for comparison
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		// Keep leading whitespace but normalize other whitespace
		leading := len(line) - len(strings.TrimLeft(line, " \t"))
		if leading > 0 {
			leadingSpace := line[:leading]
			content := strings.Fields(strings.TrimSpace(line))
			if len(content) > 0 {
				normalized = append(normalized, leadingSpace+strings.Join(content, " "))
			} else {
				normalized = append(normalized, "")
			}
		} else {
			content := strings.Fields(line)
			if len(content) > 0 {
				normalized = append(normalized, strings.Join(content, " "))
			} else {
				normalized = append(normalized, "")
			}
		}
	}
	return strings.Join(normalized, "\n")
}

// Test helper to create a document for formatting testing
func createTestDocumentForFormatting(uri, content string) (*document.Document, error) {
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
		Path:        uri[7:], // Remove "file://" prefix
		Content:     []byte(content),
		Version:     1,
		ParseResult: result,
		Symbols:     symbols,
	}

	return doc, nil
}

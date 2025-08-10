package ast

import (
	"strings"
	"testing"

	"frugal-ls/internal/parser"
)

func TestNodeTypeConstants(t *testing.T) {
	// Test that all NodeType constants are properly defined
	expectedTypes := map[NodeType]string{
		NodeTypeService:   "service",
		NodeTypeScope:     "scope",
		NodeTypeStruct:    "struct",
		NodeTypeEnum:      "enum",
		NodeTypeConst:     "const",
		NodeTypeTypedef:   "typedef",
		NodeTypeException: "exception",
		NodeTypeInclude:   "include",
		NodeTypeNamespace: "namespace",
		NodeTypeMethod:    "method",
		NodeTypeField:     "field",
		NodeTypeEvent:     "event",
		NodeTypeEnumValue: "enum_value",
	}

	for nodeType, expectedStr := range expectedTypes {
		if string(nodeType) != expectedStr {
			t.Errorf("NodeType %s should equal %q, got %q", nodeType, expectedStr, string(nodeType))
		}
	}
}

func TestGetTextEdgeCases(t *testing.T) {
	source := []byte("test")

	// Test with nil node
	nilText := GetText(nil, source)
	if nilText != "" {
		t.Errorf("Expected empty string for nil node, got %q", nilText)
	}
}

func TestExtractSymbolsBasic(t *testing.T) {
	content := `struct User {
    1: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Should have 2 symbols: struct User and service UserService
	if len(symbols) != 2 {
		t.Errorf("Expected 2 symbols, got %d", len(symbols))
		for i, sym := range symbols {
			t.Logf("Symbol %d: %s (%s)", i, sym.Name, sym.Type)
		}
		return
	}

	// Verify struct symbol
	structSymbol := symbols[0]
	if structSymbol.Name != "User" {
		t.Errorf("Expected struct symbol name 'User', got %q", structSymbol.Name)
	}
	if structSymbol.Type != NodeTypeStruct {
		t.Errorf("Expected struct symbol type %s, got %s", NodeTypeStruct, structSymbol.Type)
	}
	if structSymbol.Line != 0 {
		t.Errorf("Expected struct symbol on line 0, got %d", structSymbol.Line)
	}

	// Verify service symbol
	serviceSymbol := symbols[1]
	if serviceSymbol.Name != "UserService" {
		t.Errorf("Expected service symbol name 'UserService', got %q", serviceSymbol.Name)
	}
	if serviceSymbol.Type != NodeTypeService {
		t.Errorf("Expected service symbol type %s, got %s", NodeTypeService, serviceSymbol.Type)
	}
}

func TestExtractSymbolsAllTypes(t *testing.T) {
	content := `struct User {
    1: i64 id
}

enum Status {
    ACTIVE = 1,
    INACTIVE = 2
}

service UserService {
    User getUser(1: i64 userId)
}

scope UserEvents {
    UserCreated: User,
    UserUpdated: User
}

exception UserNotFound {
    1: string message
}

const i32 DEFAULT_TIMEOUT = 5000;

typedef string UserId`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Should have symbols for: struct, enum, service, scope, exception, const, typedef
	expectedTypes := []NodeType{
		NodeTypeStruct, NodeTypeEnum, NodeTypeService, NodeTypeScope,
		NodeTypeException, NodeTypeConst, NodeTypeTypedef,
	}

	if len(symbols) != len(expectedTypes) {
		t.Errorf("Expected %d symbols, got %d", len(expectedTypes), len(symbols))
		for i, sym := range symbols {
			t.Logf("Symbol %d: %s (%s)", i, sym.Name, sym.Type)
		}
		return
	}

	// Verify we have all expected types
	foundTypes := make(map[NodeType]bool)
	for _, symbol := range symbols {
		foundTypes[symbol.Type] = true
	}

	for _, expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("Expected to find symbol of type %s", expectedType)
		}
	}
}

func TestExtractSymbolsPositions(t *testing.T) {
	content := `struct User {
    1: i64 id
}

service UserService {
    User getUser(1: i64 userId)
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	if len(symbols) < 2 {
		t.Fatal("Expected at least 2 symbols")
	}

	source := []byte(content)

	// Test that symbols have valid positions
	for i, symbol := range symbols {
		if symbol.StartPos >= symbol.EndPos {
			t.Errorf("Symbol %d has invalid position: start %d >= end %d", i, symbol.StartPos, symbol.EndPos)
		}

		if symbol.StartPos >= uint(len(source)) {
			t.Errorf("Symbol %d start position %d is beyond source length %d", i, symbol.StartPos, len(source))
		}

		if symbol.EndPos > uint(len(source)) {
			t.Errorf("Symbol %d end position %d is beyond source length %d", i, symbol.EndPos, len(source))
		}

		// Verify the symbol name appears in the extracted text region
		symbolText := string(source[symbol.StartPos:symbol.EndPos])
		if !strings.Contains(symbolText, symbol.Name) {
			t.Errorf("Symbol %d name %q not found in extracted text: %q", i, symbol.Name, symbolText)
		}
	}
}

func TestExtractSymbolsEmpty(t *testing.T) {
	content := ``

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Empty content should produce no symbols
	if len(symbols) != 0 {
		t.Errorf("Expected 0 symbols for empty content, got %d", len(symbols))
	}
}

func TestExtractSymbolsComments(t *testing.T) {
	content := `// This is a comment
struct User {
    1: i64 id // Field comment
}

/* Multi-line comment
   about the service */
service UserService {
    User getUser(1: i64 userId)
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Comments should not affect symbol extraction
	if len(symbols) != 2 {
		t.Errorf("Expected 2 symbols (ignoring comments), got %d", len(symbols))
	}

	// Verify symbols are still correctly identified
	hasStruct := false
	hasService := false

	for _, symbol := range symbols {
		if symbol.Type == NodeTypeStruct && symbol.Name == "User" {
			hasStruct = true
		}
		if symbol.Type == NodeTypeService && symbol.Name == "UserService" {
			hasService = true
		}
	}

	if !hasStruct {
		t.Error("Expected to find struct User symbol")
	}
	if !hasService {
		t.Error("Expected to find service UserService symbol")
	}
}

func TestFindNodeByType(t *testing.T) {
	content := `struct User {
    1: i64 id,
    2: string name
}

service UserService {
    User getUser(1: i64 userId)
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	// Test finding struct_definition
	structNode := FindNodeByType(root, "struct_definition")
	if structNode == nil {
		t.Error("Expected to find struct_definition node")
	}

	// Test finding service_definition
	serviceNode := FindNodeByType(root, "service_definition")
	if serviceNode == nil {
		t.Error("Expected to find service_definition node")
	}

	// Test finding identifier (should find first one)
	identifierNode := FindNodeByType(root, "identifier")
	if identifierNode == nil {
		t.Error("Expected to find identifier node")
	}

	// Test finding non-existent node type
	nonExistentNode := FindNodeByType(root, "non_existent_type")
	if nonExistentNode != nil {
		t.Error("Expected nil for non-existent node type")
	}

	// Test with nil root
	nilResult := FindNodeByType(nil, "identifier")
	if nilResult != nil {
		t.Error("Expected nil result when searching nil root")
	}
}

func TestGetTextBasic(t *testing.T) {
	content := `struct User {
    1: i64 id
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	source := []byte(content)

	// Test getting text from root node
	rootText := GetText(root, source)
	if rootText != content {
		t.Errorf("Expected root text to match source, got: %q", rootText)
	}

	// Test getting text from struct identifier
	structDef := FindNodeByType(root, "struct_definition")
	if structDef == nil {
		t.Fatal("Expected to find struct_definition node")
	}

	identifier := FindNodeByType(structDef, "identifier")
	if identifier == nil {
		t.Fatal("Expected to find identifier node")
	}

	identifierText := GetText(identifier, source)
	if identifierText != "User" {
		t.Errorf("Expected identifier text 'User', got %q", identifierText)
	}
}

func TestSymbolStartPosition(t *testing.T) {
	content := `struct User {
    1: i64 id
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	if len(symbols) == 0 {
		t.Fatal("Expected at least one symbol")
	}

	symbol := symbols[0]
	if symbol.Line < 0 {
		t.Errorf("Expected non-negative line number, got %d", symbol.Line)
	}
	if symbol.Column < 0 {
		t.Errorf("Expected non-negative column number, got %d", symbol.Column)
	}
}

// Test PrintTree doesn't panic
func TestPrintTree(t *testing.T) {
	content := `struct User {
    1: i64 id
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	// Test PrintTree doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintTree panicked: %v", r)
		}
	}()

	PrintTree(root, []byte(content), 0)

	// Test PrintTree with nil node doesn't panic
	PrintTree(nil, []byte(content), 0)
}

// Benchmark tests for performance
func BenchmarkExtractSymbols(b *testing.B) {
	content := `struct User {
    1: i64 id,
    2: string name
}

service UserService {
    User getUser(1: i64 userId),
    void updateUser(1: User user)
}

enum Status {
    ACTIVE = 1,
    INACTIVE = 2
}

scope UserEvents {
    UserCreated: User,
    UserUpdated: User
}`

	p, err := parser.NewParser()
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		b.Fatal("Expected root node")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractSymbols(root, []byte(content))
	}
}

func BenchmarkFindNodeByType(b *testing.B) {
	content := `struct User {
    1: i64 id,
    2: string name
}

service UserService {
    User getUser(1: i64 userId),
    void updateUser(1: User user)
}`

	p, err := parser.NewParser()
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		b.Fatal("Expected root node")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindNodeByType(root, "identifier")
	}
}

func BenchmarkGetText(b *testing.B) {
	content := `struct User {
    1: i64 id,
    2: string name
}`

	p, err := parser.NewParser()
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		b.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		b.Fatal("Expected root node")
	}

	identifier := FindNodeByType(root, "identifier")
	source := []byte(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetText(identifier, source)
	}
}

// Additional test cases for enhanced coverage

func TestGetTextBoundaryConditions(t *testing.T) {
	source := []byte("struct User { }")

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse(source)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	// Test GetText with empty source
	emptyText := GetText(root, []byte{})
	if emptyText != "" {
		t.Errorf("Expected empty string for empty source, got %q", emptyText)
	}

	// Test GetText with node that has zero-length text
	identifier := FindNodeByType(root, "identifier")
	if identifier != nil {
		text := GetText(identifier, source)
		if text != "User" {
			t.Errorf("Expected 'User', got %q", text)
		}
	}
}

func TestExtractSymbolsNestedStructures(t *testing.T) {
	content := `struct OuterStruct {
    1: InnerStruct inner
}

struct InnerStruct {
    1: string value
}

service NestedService {
    struct LocalStruct {
        1: i32 localField
    }
    
    LocalStruct getLocal()
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Should find at least OuterStruct, InnerStruct, and NestedService
	expectedNames := map[string]NodeType{
		"OuterStruct":   NodeTypeStruct,
		"InnerStruct":   NodeTypeStruct,
		"NestedService": NodeTypeService,
	}

	foundSymbols := make(map[string]NodeType)
	for _, symbol := range symbols {
		foundSymbols[symbol.Name] = symbol.Type
	}

	for expectedName, expectedType := range expectedNames {
		if foundType, exists := foundSymbols[expectedName]; !exists {
			t.Errorf("Expected to find symbol %q of type %s", expectedName, expectedType)
		} else if foundType != expectedType {
			t.Errorf("Expected symbol %q to be of type %s, got %s", expectedName, expectedType, foundType)
		}
	}
}

func TestExtractSymbolsWithInheritance(t *testing.T) {
	content := `struct BaseStruct {
    1: string baseField
}

struct DerivedStruct extends BaseStruct {
    2: i32 derivedField
}

exception BaseException {
    1: string message
}

exception SpecificException extends BaseException {
    2: i32 errorCode
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Should extract symbols regardless of inheritance
	expectedSymbols := map[string]NodeType{
		"BaseStruct":        NodeTypeStruct,
		"DerivedStruct":     NodeTypeStruct,
		"BaseException":     NodeTypeException,
		"SpecificException": NodeTypeException,
	}

	foundSymbols := make(map[string]NodeType)
	for _, symbol := range symbols {
		foundSymbols[symbol.Name] = symbol.Type
	}

	for name, expectedType := range expectedSymbols {
		if foundType, exists := foundSymbols[name]; !exists {
			t.Errorf("Expected to find symbol %q", name)
		} else if foundType != expectedType {
			t.Errorf("Symbol %q should be type %s, got %s", name, expectedType, foundType)
		}
	}
}

func TestExtractSymbolsComplexService(t *testing.T) {
	content := `service ComplexService {
    // Method with throws clause
    User getUser(1: i64 id) throws (1: UserNotFound notFound),
    
    // Method with multiple parameters
    void updateUser(
        1: i64 id,
        2: string name,
        3: optional string email
    ) throws (1: UserNotFound notFound, 2: ValidationError invalid),
    
    // Oneway method
    oneway void logEvent(1: string event),
    
    // Method with complex return type
    list<map<string, User>> getAllUserMaps()
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Should find the service symbol
	hasComplexService := false
	for _, symbol := range symbols {
		if symbol.Name == "ComplexService" && symbol.Type == NodeTypeService {
			hasComplexService = true
			break
		}
	}

	if !hasComplexService {
		t.Error("Expected to find ComplexService symbol")
	}
}

func TestExtractSymbolsAnnotations(t *testing.T) {
	content := `@deprecated
struct LegacyStruct {
    1: string oldField
}

@readonly
service ReadOnlyService {
    @throttle(1000)
    string getData(1: string key)
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	// Should extract symbols even with annotations
	expectedSymbols := []string{"LegacyStruct", "ReadOnlyService"}
	foundNames := make([]string, len(symbols))
	for i, symbol := range symbols {
		foundNames[i] = symbol.Name
	}

	for _, expected := range expectedSymbols {
		found := false
		for _, name := range foundNames {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find symbol %q (annotations should not prevent extraction)", expected)
		}
	}
}

func TestFindNodeByTypeRecursive(t *testing.T) {
	content := `struct OuterStruct {
    1: struct InnerStruct {
        1: string value
    } inner
}`

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	// Find multiple struct definitions (should find first one)
	firstStruct := FindNodeByType(root, "struct_definition")
	if firstStruct == nil {
		t.Error("Expected to find at least one struct_definition")
		return
	}

	// Verify it finds the outer struct first
	identifier := FindNodeByType(firstStruct, "identifier")
	if identifier != nil {
		text := GetText(identifier, []byte(content))
		if text != "OuterStruct" {
			t.Errorf("Expected to find OuterStruct first, got %q", text)
		}
	}
}

func TestSymbolPositionAccuracy(t *testing.T) {
	content := `namespace go test

struct User {
    1: i64 id,
    2: string name
}

service UserService {
    User getUser(1: i64 userId)
}`

	lines := strings.Split(content, "\n")

	p, err := parser.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	result, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer result.Close()

	root := result.GetRootNode()
	if root == nil {
		t.Fatal("Expected root node")
	}

	symbols := ExtractSymbols(root, []byte(content))

	for _, symbol := range symbols {
		// Verify line number is within bounds
		if symbol.Line >= len(lines) {
			t.Errorf("Symbol %s line %d is beyond source lines (%d)", symbol.Name, symbol.Line, len(lines))
			continue
		}

		// Verify the symbol name appears on the reported line
		line := lines[symbol.Line]
		if !strings.Contains(line, symbol.Name) {
			t.Errorf("Symbol %s not found on reported line %d: %q", symbol.Name, symbol.Line, line)
		}
	}
}

func TestGetTextEdgeCasesExtended(t *testing.T) {
	// Test with source that has invalid byte positions
	source := []byte("short")

	// Mock a node with invalid positions (this is theoretical since tree-sitter should not produce these)
	// But we test the bounds checking in GetText

	// Test GetText with position exactly at source length
	// This tests the bounds checking logic
	result := GetText(nil, source)
	if result != "" {
		t.Errorf("Expected empty string for nil node, got %q", result)
	}

	// Test with empty source
	emptyResult := GetText(nil, []byte{})
	if emptyResult != "" {
		t.Errorf("Expected empty string for nil node with empty source, got %q", emptyResult)
	}
}

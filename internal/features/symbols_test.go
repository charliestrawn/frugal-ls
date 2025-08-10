package features

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

func TestNewDocumentSymbolProvider(t *testing.T) {
	provider := NewDocumentSymbolProvider()
	if provider == nil {
		t.Fatal("Expected non-nil DocumentSymbolProvider")
	}
}

func TestProvideDocumentSymbolsEmpty(t *testing.T) {
	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte(""),
		Version: 1,
		Symbols: []ast.Symbol{}, // Empty symbols
	}

	provider := NewDocumentSymbolProvider()
	symbols, err := provider.ProvideDocumentSymbols(doc)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if symbols != nil {
		t.Errorf("Expected nil symbols for empty document, got %v", symbols)
	}
}

func TestProvideDocumentSymbolsBasic(t *testing.T) {
	// Mock symbols for testing
	mockSymbols := []ast.Symbol{
		{
			Name:     "User",
			Type:     ast.NodeTypeStruct,
			Line:     0,
			Column:   0,
			StartPos: 0,
			EndPos:   30,
		},
		{
			Name:     "UserService",
			Type:     ast.NodeTypeService,
			Line:     3,
			Column:   0,
			StartPos: 31,
			EndPos:   60,
		},
	}

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte("struct User {}\nservice UserService {}"),
		Version: 1,
		Symbols: mockSymbols,
	}

	provider := NewDocumentSymbolProvider()
	symbols, err := provider.ProvideDocumentSymbols(doc)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(symbols))
	}

	// Check struct symbol
	structSymbol := symbols[0]
	if structSymbol.Name != "User" {
		t.Errorf("Expected struct symbol name 'User', got %q", structSymbol.Name)
	}
	if structSymbol.Kind != protocol.SymbolKindStruct {
		t.Errorf("Expected struct symbol kind, got %v", structSymbol.Kind)
	}

	// Check service symbol
	serviceSymbol := symbols[1]
	if serviceSymbol.Name != "UserService" {
		t.Errorf("Expected service symbol name 'UserService', got %q", serviceSymbol.Name)
	}
	if serviceSymbol.Kind != protocol.SymbolKindClass {
		t.Errorf("Expected service symbol kind (class), got %v", serviceSymbol.Kind)
	}
}

func TestProvideWorkspaceSymbolsEmpty(t *testing.T) {
	documents := make(map[string]*document.Document)

	provider := NewDocumentSymbolProvider()
	symbols, err := provider.ProvideWorkspaceSymbols("", documents)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(symbols) != 0 {
		t.Errorf("Expected 0 symbols for empty workspace, got %d", len(symbols))
	}
}

func TestProvideWorkspaceSymbolsBasic(t *testing.T) {
	// Mock symbols
	mockSymbols1 := []ast.Symbol{
		{
			Name:     "User",
			Type:     ast.NodeTypeStruct,
			Line:     0,
			Column:   0,
			StartPos: 0,
			EndPos:   30,
		},
	}

	mockSymbols2 := []ast.Symbol{
		{
			Name:     "UserService",
			Type:     ast.NodeTypeService,
			Line:     0,
			Column:   0,
			StartPos: 0,
			EndPos:   40,
		},
	}

	documents := map[string]*document.Document{
		"file:///user.frugal": {
			URI:     "file:///user.frugal",
			Path:    "/user.frugal",
			Content: []byte("struct User {}"),
			Version: 1,
			Symbols: mockSymbols1,
		},
		"file:///service.frugal": {
			URI:     "file:///service.frugal",
			Path:    "/service.frugal",
			Content: []byte("service UserService {}"),
			Version: 1,
			Symbols: mockSymbols2,
		},
	}

	provider := NewDocumentSymbolProvider()
	symbols, err := provider.ProvideWorkspaceSymbols("", documents)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(symbols))
	}

	// Check that both symbols are present
	foundNames := make(map[string]bool)
	for _, symbol := range symbols {
		foundNames[symbol.Name] = true
	}

	if !foundNames["User"] {
		t.Error("Expected to find User symbol in workspace")
	}
	if !foundNames["UserService"] {
		t.Error("Expected to find UserService symbol in workspace")
	}
}

func TestProvideWorkspaceSymbolsWithQuery(t *testing.T) {
	mockSymbols := []ast.Symbol{
		{
			Name:     "User",
			Type:     ast.NodeTypeStruct,
			Line:     0,
			Column:   0,
			StartPos: 0,
			EndPos:   30,
		},
		{
			Name:     "UserService",
			Type:     ast.NodeTypeService,
			Line:     1,
			Column:   0,
			StartPos: 31,
			EndPos:   60,
		},
		{
			Name:     "Order",
			Type:     ast.NodeTypeStruct,
			Line:     2,
			Column:   0,
			StartPos: 61,
			EndPos:   90,
		},
	}

	documents := map[string]*document.Document{
		"file:///test.frugal": {
			URI:     "file:///test.frugal",
			Path:    "/test.frugal",
			Content: []byte("struct User {}\nservice UserService {}\nstruct Order {}"),
			Version: 1,
			Symbols: mockSymbols,
		},
	}

	provider := NewDocumentSymbolProvider()

	// Query for "User" - should match "User" and "UserService"
	symbols, err := provider.ProvideWorkspaceSymbols("User", documents)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(symbols) != 2 {
		t.Fatalf("Expected 2 symbols matching 'User', got %d", len(symbols))
	}

	foundNames := make(map[string]bool)
	for _, symbol := range symbols {
		foundNames[symbol.Name] = true
	}

	if !foundNames["User"] {
		t.Error("Expected to find User symbol")
	}
	if !foundNames["UserService"] {
		t.Error("Expected to find UserService symbol")
	}
	if foundNames["Order"] {
		t.Error("Did not expect to find Order symbol for 'User' query")
	}
}

func TestProvideWorkspaceSymbolsSkipInvalidFiles(t *testing.T) {
	mockSymbols := []ast.Symbol{
		{
			Name:     "User",
			Type:     ast.NodeTypeStruct,
			Line:     0,
			Column:   0,
			StartPos: 0,
			EndPos:   30,
		},
	}

	documents := map[string]*document.Document{
		"file:///test.frugal": {
			URI:     "file:///test.frugal",
			Path:    "/test.frugal",
			Content: []byte("struct User {}"),
			Version: 1,
			Symbols: mockSymbols,
		},
		"file:///test.txt": { // Invalid file extension
			URI:     "file:///test.txt",
			Path:    "/test.txt",
			Content: []byte("not frugal content"),
			Version: 1,
			Symbols: mockSymbols, // Has symbols but should be skipped
		},
	}

	provider := NewDocumentSymbolProvider()
	symbols, err := provider.ProvideWorkspaceSymbols("", documents)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only find symbol from valid .frugal file
	if len(symbols) != 1 {
		t.Fatalf("Expected 1 symbol (skipping invalid file), got %d", len(symbols))
	}

	if symbols[0].Name != "User" {
		t.Errorf("Expected User symbol, got %q", symbols[0].Name)
	}
}

func TestConvertToDocumentSymbolAllTypes(t *testing.T) {
	provider := NewDocumentSymbolProvider()

	testCases := []struct {
		symbolType   ast.NodeType
		expectedKind protocol.SymbolKind
	}{
		{ast.NodeTypeStruct, protocol.SymbolKindStruct},
		{ast.NodeTypeService, protocol.SymbolKindClass},
		{ast.NodeTypeEnum, protocol.SymbolKindEnum},
		{ast.NodeTypeException, protocol.SymbolKindClass},
		{ast.NodeTypeConst, protocol.SymbolKindConstant},
		{ast.NodeTypeTypedef, protocol.SymbolKindTypeParameter},
		{ast.NodeTypeScope, protocol.SymbolKindClass},
		{ast.NodeTypeMethod, protocol.SymbolKindVariable}, // Falls back to default
		{ast.NodeTypeField, protocol.SymbolKindVariable},  // Falls back to default
		{ast.NodeTypeEvent, protocol.SymbolKindVariable},  // Falls back to default
	}

	content := []byte("test content for symbol conversion")
	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: content,
		Version: 1,
	}

	for _, tc := range testCases {
		t.Run(string(tc.symbolType), func(t *testing.T) {
			symbol := ast.Symbol{
				Name:     "TestSymbol",
				Type:     tc.symbolType,
				Line:     0,
				Column:   0,
				StartPos: 0,
				EndPos:   10,
			}

			docSymbol := provider.convertToDocumentSymbol(symbol, doc)

			if docSymbol.Kind != tc.expectedKind {
				t.Errorf("Expected symbol kind %v for type %s, got %v", tc.expectedKind, tc.symbolType, docSymbol.Kind)
			}

			if docSymbol.Name != "TestSymbol" {
				t.Errorf("Expected symbol name 'TestSymbol', got %q", docSymbol.Name)
			}
		})
	}
}

func TestSymbolRangeCalculation(t *testing.T) {
	provider := NewDocumentSymbolProvider()

	symbol := ast.Symbol{
		Name:     "TestStruct",
		Type:     ast.NodeTypeStruct,
		Line:     5,
		Column:   10,
		StartPos: 100,
		EndPos:   150,
	}

	doc := &document.Document{
		URI:     "file:///test.frugal",
		Path:    "/test.frugal",
		Content: []byte("test content"),
		Version: 1,
	}

	docSymbol := provider.convertToDocumentSymbol(symbol, doc)

	if docSymbol.Name != "TestStruct" {
		t.Errorf("Expected symbol name 'TestStruct', got %q", docSymbol.Name)
	}

	if docSymbol.Kind != protocol.SymbolKindStruct {
		t.Errorf("Expected struct symbol kind, got %v", docSymbol.Kind)
	}

	if docSymbol.Range.Start.Line != 5 {
		t.Errorf("Expected line 5, got %d", docSymbol.Range.Start.Line)
	}

	if docSymbol.Range.Start.Character != 10 {
		t.Errorf("Expected column 10, got %d", docSymbol.Range.Start.Character)
	}

	// Selection range should be the same as the symbol name range when no Node is available
	if docSymbol.SelectionRange.Start.Line != 5 {
		t.Errorf("Expected selection range line 5, got %d", docSymbol.SelectionRange.Start.Line)
	}
}

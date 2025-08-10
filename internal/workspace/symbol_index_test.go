package workspace

import (
	"testing"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func TestSymbolIndexBasic(t *testing.T) {
	index := NewSymbolIndex()

	// Test initial state
	stats := index.GetStatistics()
	if stats["documents"] != 0 {
		t.Errorf("Expected 0 documents, got %v", stats["documents"])
	}
	if stats["total_symbols"] != 0 {
		t.Errorf("Expected 0 symbols, got %v", stats["total_symbols"])
	}
}

func TestSymbolIndexUpdateDocument(t *testing.T) {
	index := NewSymbolIndex()

	content := `service UserService {
    User getUser(1: i64 userId),
    void updateUser(1: User user)
}

struct User {
    1: required i64 id,
    2: optional string name
}

enum Status {
    ACTIVE = 1,
    INACTIVE = 2
}`

	doc, err := createTestDocumentForIndex("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Update document in index
	index.UpdateDocument(doc)

	// Check statistics
	stats := index.GetStatistics()
	if stats["documents"] != 1 {
		t.Errorf("Expected 1 document, got %v", stats["documents"])
	}

	totalSymbols := stats["total_symbols"].(int)
	if totalSymbols < 3 { // At least service, struct, enum
		t.Errorf("Expected at least 3 symbols, got %v", totalSymbols)
	}

	t.Logf("Index statistics: %+v", stats)
}

func TestSymbolIndexSearch(t *testing.T) {
	index := NewSymbolIndex()

	content := `service UserService {
    User getUser(1: i64 userId),
    void createUser(1: string name, 2: string email)
}

struct User {
    1: required i64 id,
    2: optional string name,
    3: optional string email
}

struct UserProfile {
    1: User user,
    2: string bio
}

enum UserStatus {
    ACTIVE = 1,
    INACTIVE = 2,
    SUSPENDED = 3
}`

	doc, err := createTestDocumentForIndex("file:///users.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	index.UpdateDocument(doc)

	// Test empty query (should return all symbols with limit)
	allSymbols := index.Search("", 10)
	if len(allSymbols) == 0 {
		t.Error("Expected some symbols for empty query")
	}
	t.Logf("Found %d symbols for empty query", len(allSymbols))

	// Test exact match
	userSymbols := index.Search("User", 10)
	if len(userSymbols) == 0 {
		t.Error("Expected to find 'User' symbols")
	}

	foundUserStruct := false
	foundUserService := false
	for _, symbol := range userSymbols {
		t.Logf("Found symbol: %s (kind: %d)", symbol.Name, symbol.Kind)
		if symbol.Name == "User" {
			foundUserStruct = true
		}
		if symbol.Name == "UserService" {
			foundUserService = true
		}
	}

	if !foundUserStruct {
		t.Error("Expected to find User struct")
	}
	if !foundUserService {
		t.Error("Expected to find UserService")
	}

	// Test partial match
	serviceSymbols := index.Search("Service", 10)
	if len(serviceSymbols) == 0 {
		t.Error("Expected to find symbols containing 'Service'")
	}

	// Test method search (should find nested methods)
	methodSymbols := index.Search("getUser", 10)
	if len(methodSymbols) == 0 {
		t.Error("Expected to find 'getUser' method")
	}

	// Test case insensitive search
	lowerSymbols := index.Search("user", 10)
	if len(lowerSymbols) == 0 {
		t.Error("Expected case insensitive search to work")
	}
}

func TestSymbolIndexSearchByType(t *testing.T) {
	index := NewSymbolIndex()

	content := `service OrderService {
    Order getOrder(1: i64 orderId)
}

struct Order {
    1: i64 id,
    2: string name
}

enum OrderStatus {
    PENDING = 1,
    SHIPPED = 2
}

const string DEFAULT_STATUS = "pending"`

	doc, err := createTestDocumentForIndex("file:///orders.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	index.UpdateDocument(doc)

	// Search for services only
	services := index.SearchByType([]ast.NodeType{ast.NodeTypeService}, "", 10)
	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}
	if len(services) > 0 && services[0].Name != "OrderService" {
		t.Errorf("Expected OrderService, got %s", services[0].Name)
	}

	// Search for structs only
	structs := index.SearchByType([]ast.NodeType{ast.NodeTypeStruct}, "", 10)
	if len(structs) != 1 {
		t.Errorf("Expected 1 struct, got %d", len(structs))
	}

	// Search for enums only
	enums := index.SearchByType([]ast.NodeType{ast.NodeTypeEnum}, "", 10)
	if len(enums) != 1 {
		t.Errorf("Expected 1 enum, got %d", len(enums))
	}

	// Search for constants only
	constants := index.SearchByType([]ast.NodeType{ast.NodeTypeConst}, "", 10)
	if len(constants) != 1 {
		t.Errorf("Expected 1 constant, got %d", len(constants))
	}
}

func TestSymbolIndexRemoveDocument(t *testing.T) {
	index := NewSymbolIndex()

	content := `service TestService {
    void testMethod()
}`

	doc, err := createTestDocumentForIndex("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Add document
	index.UpdateDocument(doc)

	// Verify it was added
	initialStats := index.GetStatistics()
	if initialStats["documents"] != 1 {
		t.Errorf("Expected 1 document, got %v", initialStats["documents"])
	}

	symbols := index.Search("TestService", 10)
	if len(symbols) == 0 {
		t.Error("Expected to find TestService after adding")
	}

	// Remove document
	index.RemoveDocument("file:///test.frugal")

	// Verify it was removed
	finalStats := index.GetStatistics()
	if finalStats["documents"] != 0 {
		t.Errorf("Expected 0 documents after removal, got %v", finalStats["documents"])
	}

	symbolsAfterRemoval := index.Search("TestService", 10)
	if len(symbolsAfterRemoval) != 0 {
		t.Error("Expected no symbols after document removal")
	}
}

func TestSymbolIndexMultipleDocuments(t *testing.T) {
	index := NewSymbolIndex()

	// Add first document
	content1 := `service UserService {
    User getUser(1: i64 userId)
}

struct User {
    1: i64 id,
    2: string name
}`

	doc1, err := createTestDocumentForIndex("file:///users.frugal", content1)
	if err != nil {
		t.Fatalf("Failed to create document 1: %v", err)
	}
	defer doc1.ParseResult.Close()
	index.UpdateDocument(doc1)

	// Add second document
	content2 := `service OrderService {
    Order getOrder(1: i64 orderId)
}

struct Order {
    1: i64 id,
    2: User user
}`

	doc2, err := createTestDocumentForIndex("file:///orders.frugal", content2)
	if err != nil {
		t.Fatalf("Failed to create document 2: %v", err)
	}
	defer doc2.ParseResult.Close()
	index.UpdateDocument(doc2)

	// Check statistics
	stats := index.GetStatistics()
	if stats["documents"] != 2 {
		t.Errorf("Expected 2 documents, got %v", stats["documents"])
	}

	// Search should find symbols from both documents
	allSymbols := index.Search("", 20)
	if len(allSymbols) < 4 { // At least 2 services + 2 structs
		t.Errorf("Expected at least 4 symbols, got %d", len(allSymbols))
	}

	// Search for User should find references in both documents
	userSymbols := index.Search("User", 10)
	foundUserService := false
	foundUserStruct := false

	for _, symbol := range userSymbols {
		if symbol.Name == "UserService" {
			foundUserService = true
		}
		if symbol.Name == "User" {
			foundUserStruct = true
		}
	}

	if !foundUserService {
		t.Error("Expected to find UserService")
	}
	if !foundUserStruct {
		t.Error("Expected to find User struct")
	}
}

func TestSymbolIndexRelevanceScoring(t *testing.T) {
	index := NewSymbolIndex()

	content := `service UserService {
    User getUser(1: i64 userId)
}

struct User {
    1: i64 id
}

struct UserProfile {
    1: User user
}

struct UserSettings {
    1: string theme
}`

	doc, err := createTestDocumentForIndex("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	index.UpdateDocument(doc)

	// Search for "User" - should prioritize exact matches
	symbols := index.Search("User", 10)
	if len(symbols) == 0 {
		t.Fatal("Expected to find symbols")
	}

	// First result should be exact match "User"
	if symbols[0].Name != "User" {
		t.Errorf("Expected exact match 'User' first, got '%s'", symbols[0].Name)
	}

	t.Logf("Search results for 'User' (in relevance order):")
	for i, symbol := range symbols {
		t.Logf("  %d. %s", i+1, symbol.Name)
	}
}

// Test helper to create a document for symbol index testing
func createTestDocumentForIndex(uri, content string) (*document.Document, error) {
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

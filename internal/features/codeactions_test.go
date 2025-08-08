package features

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

func TestCodeActionProvider(t *testing.T) {
	provider := NewCodeActionProvider()
	if provider == nil {
		t.Fatal("Code action provider should not be nil")
	}
}

func TestExtractParametersToStruct(t *testing.T) {
	provider := NewCodeActionProvider()

	content := `service UserService {
    User getUser(1: i64 userId, 2: string name, 3: bool active) throws (1: UserNotFound error),
    void updateUser(1: User user)
}`

	doc, err := createTestDocumentForCodeActions("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position on the getUser method (line 1, around column 10)
	rng := protocol.Range{
		Start: protocol.Position{Line: 1, Character: 10},
		End:   protocol.Position{Line: 1, Character: 15},
	}

	context := protocol.CodeActionContext{
		Diagnostics: []protocol.Diagnostic{},
	}

	actions, err := provider.ProvideCodeActions(doc, rng, context)
	if err != nil {
		t.Fatalf("ProvideCodeActions failed: %v", err)
	}

	// Look for the extract parameters action
	var extractAction *protocol.CodeAction
	for _, action := range actions {
		if strings.Contains(action.Title, "Extract parameters to") {
			extractAction = &action
			break
		}
	}

	if extractAction == nil {
		t.Fatal("Expected extract parameters action but didn't find one")
	}

	if extractAction.Kind == nil || *extractAction.Kind != protocol.CodeActionKindRefactor {
		t.Errorf("Expected refactor kind, got %v", extractAction.Kind)
	}

	if !strings.Contains(extractAction.Title, "GetUserRequest") {
		t.Errorf("Expected title to contain 'GetUserRequest', got: %s", extractAction.Title)
	}

	// Check that we have workspace edits
	if extractAction.Edit == nil || extractAction.Edit.Changes == nil {
		t.Fatal("Expected workspace edits but got none")
	}

	edits := extractAction.Edit.Changes["file:///test.frugal"]
	if len(edits) != 2 {
		t.Errorf("Expected 2 edits (struct creation + method update), got %d", len(edits))
	}

	t.Logf("Extract action title: %s", extractAction.Title)
	for i, edit := range edits {
		t.Logf("Edit %d: %s", i, edit.NewText)
	}
}

func TestParameterExtraction(t *testing.T) {
	provider := NewCodeActionProvider()

	content := `service TestService {
    string testMethod(1: i64 id, 2: string name, 3: bool flag)
}`

	doc, err := createTestDocumentForCodeActions("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Find the function node manually for testing
	root := doc.ParseResult.GetRootNode()
	var functionNode *tree_sitter.Node

	// Helper to find function definition
	var findFunction func(*tree_sitter.Node)
	findFunction = func(node *tree_sitter.Node) {
		if node.Kind() == "function_definition" {
			functionNode = node
			return
		}
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			findFunction(child)
		}
	}

	findFunction(root)

	if functionNode == nil {
		t.Fatal("Could not find function definition")
	}

	// Test parameter extraction
	parameters := provider.extractParameters(functionNode, doc.Content)
	if len(parameters) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(parameters))
	}

	expectedParams := []struct {
		ID   string
		Type string
		Name string
	}{
		{"1", "i64", "id"},
		{"2", "string", "name"},
		{"3", "bool", "flag"},
	}

	for i, expected := range expectedParams {
		if i >= len(parameters) {
			t.Errorf("Missing parameter %d", i)
			continue
		}
		param := parameters[i]
		if param.ID != expected.ID {
			t.Errorf("Parameter %d ID: expected %s, got %s", i, expected.ID, param.ID)
		}
		if param.Type != expected.Type {
			t.Errorf("Parameter %d Type: expected %s, got %s", i, expected.Type, param.Type)
		}
		if param.Name != expected.Name {
			t.Errorf("Parameter %d Name: expected %s, got %s", i, expected.Name, param.Name)
		}
	}

	// Test method name extraction
	methodName := provider.extractMethodName(functionNode, doc.Content)
	if methodName != "testMethod" {
		t.Errorf("Expected method name 'testMethod', got %s", methodName)
	}

	// Test struct name generation
	structName := provider.generateStructName(methodName)
	if structName != "TestMethodRequest" {
		t.Errorf("Expected struct name 'TestMethodRequest', got %s", structName)
	}

	// Test struct generation
	structDef := provider.generateParameterStruct(structName, parameters)
	expectedStruct := `struct TestMethodRequest {
    1: i64 id,
    2: string name,
    3: bool flag,
}`
	if structDef != expectedStruct {
		t.Errorf("Expected struct:\n%s\nGot:\n%s", expectedStruct, structDef)
	}
}

func TestSingleParameterMethodShouldNotOffer(t *testing.T) {
	provider := NewCodeActionProvider()

	content := `service UserService {
    void updateUser(1: User user)
}`

	doc, err := createTestDocumentForCodeActions("file:///test.frugal", content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	defer doc.ParseResult.Close()

	// Position on the updateUser method
	rng := protocol.Range{
		Start: protocol.Position{Line: 1, Character: 10},
		End:   protocol.Position{Line: 1, Character: 15},
	}

	context := protocol.CodeActionContext{
		Diagnostics: []protocol.Diagnostic{},
	}

	actions, err := provider.ProvideCodeActions(doc, rng, context)
	if err != nil {
		t.Fatalf("ProvideCodeActions failed: %v", err)
	}

	// Should not find extract parameters action for single parameter method
	for _, action := range actions {
		if strings.Contains(action.Title, "Extract parameters to") {
			t.Error("Should not offer extract parameters action for single parameter method")
		}
	}
}

// Test helper to create a document for code action testing
func createTestDocumentForCodeActions(uri, content string) (*document.Document, error) {
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

	// Extract path from URI for proper validation
	path := strings.TrimPrefix(uri, "file://")

	doc := &document.Document{
		URI:         uri,
		Path:        path,
		Content:     []byte(content),
		Version:     1,
		ParseResult: result,
		Symbols:     symbols,
	}

	return doc, nil
}
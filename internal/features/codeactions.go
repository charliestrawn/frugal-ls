package features

import (
	"fmt"
	"strconv"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// FindNodeAtPosition finds the AST node at a specific line and character position
func FindNodeAtPosition(node *tree_sitter.Node, source []byte, line, character uint) *tree_sitter.Node {
	if node == nil {
		return nil
	}

	start := node.StartPosition()
	end := node.EndPosition()

	// Check if the position is within this node
	if line < start.Row || line > end.Row {
		return nil
	}
	if line == start.Row && character < start.Column {
		return nil
	}
	if line == end.Row && character > end.Column {
		return nil
	}

	// Look for the most specific child node at this position
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if childNode := FindNodeAtPosition(child, source, line, character); childNode != nil {
			return childNode
		}
	}

	// Return this node if no child contains the position
	return node
}

// CodeActionProvider handles code actions and quick fixes for Frugal files
type CodeActionProvider struct{}

// NewCodeActionProvider creates a new code action provider
func NewCodeActionProvider() *CodeActionProvider {
	return &CodeActionProvider{}
}

// ProvideCodeActions provides code actions for a given range and context
func (c *CodeActionProvider) ProvideCodeActions(doc *document.Document, rng protocol.Range, context protocol.CodeActionContext) ([]protocol.CodeAction, error) {
	var actions []protocol.CodeAction
	
	if !doc.IsValidFrugalFile() {
		return actions, nil
	}
	
	// Add quick fixes for diagnostics
	if len(context.Diagnostics) > 0 {
		quickFixes := c.getQuickFixes(doc, context.Diagnostics)
		actions = append(actions, quickFixes...)
	}
	
	// Add refactoring actions
	refactorActions := c.getRefactorActions(doc, rng)
	actions = append(actions, refactorActions...)
	
	// Add source actions
	sourceActions := c.getSourceActions(doc)
	actions = append(actions, sourceActions...)
	
	return actions, nil
}

// getQuickFixes provides quick fixes for diagnostics
func (c *CodeActionProvider) getQuickFixes(doc *document.Document, diagnostics []protocol.Diagnostic) []protocol.CodeAction {
	var actions []protocol.CodeAction
	
	for _, diagnostic := range diagnostics {
		if diagnostic.Source != nil && *diagnostic.Source == "frugal-ls" {
			// Handle missing closing parenthesis
			if strings.Contains(diagnostic.Message, "Missing )") {
				action := c.createFixMissingParenthesis(doc, diagnostic)
				if action != nil {
					actions = append(actions, *action)
				}
			}
			
			// Handle missing semicolon
			if strings.Contains(diagnostic.Message, "Missing ;") {
				action := c.createFixMissingSemicolon(doc, diagnostic)
				if action != nil {
					actions = append(actions, *action)
				}
			}
		}
	}
	
	return actions
}

// getRefactorActions provides refactoring actions
func (c *CodeActionProvider) getRefactorActions(doc *document.Document, rng protocol.Range) []protocol.CodeAction {
	var actions []protocol.CodeAction
	
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return actions
	}
	
	// Find node at the range
	node := FindNodeAtPosition(doc.ParseResult.GetRootNode(), doc.Content, uint(rng.Start.Line), uint(rng.Start.Character))
	if node == nil {
		return actions
	}
	
	// Extract method from service
	if c.isInServiceBody(node) {
		action := c.createExtractMethodAction(doc, rng, node)
		if action != nil {
			actions = append(actions, *action)
		}
	}
	
	// Add field to struct
	if c.isInStructBody(node) {
		action := c.createAddFieldAction(doc, rng, node)
		if action != nil {
			actions = append(actions, *action)
		}
	}
	
	// Generate constructor for struct
	if node.Kind() == "struct_definition" {
		action := c.createGenerateConstructorAction(doc, node)
		if action != nil {
			actions = append(actions, *action)
		}
	}
	
	return actions
}

// getSourceActions provides source-level actions
func (c *CodeActionProvider) getSourceActions(doc *document.Document) []protocol.CodeAction {
	var actions []protocol.CodeAction
	
	// Add missing include action
	missingIncludeAction := c.createAddMissingIncludeAction(doc)
	if missingIncludeAction != nil {
		actions = append(actions, *missingIncludeAction)
	}
	
	// Organize imports action
	organizeImportsAction := c.createOrganizeIncludesAction(doc)
	if organizeImportsAction != nil {
		actions = append(actions, *organizeImportsAction)
	}
	
	// Generate service template
	generateServiceAction := c.createGenerateServiceAction(doc)
	if generateServiceAction != nil {
		actions = append(actions, *generateServiceAction)
	}
	
	// Generate scope template
	generateScopeAction := c.createGenerateScopeAction(doc)
	if generateScopeAction != nil {
		actions = append(actions, *generateScopeAction)
	}
	
	return actions
}

// createFixMissingParenthesis creates a quick fix for missing parenthesis
func (c *CodeActionProvider) createFixMissingParenthesis(doc *document.Document, diagnostic protocol.Diagnostic) *protocol.CodeAction {
	// Insert closing parenthesis at the end of the line
	line := diagnostic.Range.Start.Line
	endOfLine := protocol.Position{
		Line:      line,
		Character: diagnostic.Range.End.Character,
	}
	
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: endOfLine,
			End:   endOfLine,
		},
		NewText: ")",
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindQuickFix
	return &protocol.CodeAction{
		Title: "Add missing closing parenthesis",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
		Diagnostics: []protocol.Diagnostic{diagnostic},
	}
}

// createFixMissingSemicolon creates a quick fix for missing semicolon
func (c *CodeActionProvider) createFixMissingSemicolon(doc *document.Document, diagnostic protocol.Diagnostic) *protocol.CodeAction {
	// Insert semicolon at the diagnostic location
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: diagnostic.Range.End,
			End:   diagnostic.Range.End,
		},
		NewText: ";",
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindQuickFix
	return &protocol.CodeAction{
		Title: "Add missing semicolon",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
		Diagnostics: []protocol.Diagnostic{diagnostic},
	}
}

// createExtractMethodAction creates an action to extract a method from service
func (c *CodeActionProvider) createExtractMethodAction(doc *document.Document, rng protocol.Range, node *tree_sitter.Node) *protocol.CodeAction {
	// This is a placeholder for method extraction logic
	// In a real implementation, this would analyze the selected code and extract it into a new method
	
	methodName := "newMethod"
	methodSignature := fmt.Sprintf("    void %s() {\n        // Extracted method\n    }", methodName)
	
	// Find insertion point (end of service body)
	serviceNode := c.findParentOfType(node, "service_definition")
	if serviceNode == nil {
		return nil
	}
	
	// Insert at the end of service body
	insertPosition := protocol.Position{
		Line:      uint32(serviceNode.EndPosition().Row),
		Character: 0,
	}
	
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: insertPosition,
			End:   insertPosition,
		},
		NewText: "\n" + methodSignature,
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindRefactor
	return &protocol.CodeAction{
		Title: fmt.Sprintf("Extract method '%s'", methodName),
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
	}
}

// createAddFieldAction creates an action to add a field to struct
func (c *CodeActionProvider) createAddFieldAction(doc *document.Document, rng protocol.Range, node *tree_sitter.Node) *protocol.CodeAction {
	structNode := c.findParentOfType(node, "struct_definition")
	if structNode == nil {
		return nil
	}
	
	// Get next field ID by counting existing fields
	fieldID := c.getNextFieldID(structNode, doc.Content)
	
	newField := fmt.Sprintf("\n    %d: optional string newField,", fieldID)
	
	// Find insertion point (end of struct body)
	insertPosition := protocol.Position{
		Line:      uint32(structNode.EndPosition().Row),
		Character: 0,
	}
	
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: insertPosition,
			End:   insertPosition,
		},
		NewText: newField,
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindRefactor
	return &protocol.CodeAction{
		Title: "Add field to struct",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
	}
}

// createGenerateConstructorAction creates an action to generate a constructor
func (c *CodeActionProvider) createGenerateConstructorAction(doc *document.Document, structNode *tree_sitter.Node) *protocol.CodeAction {
	structName := c.extractIdentifier(structNode, doc.Content)
	if structName == "" {
		return nil
	}
	
	constructor := fmt.Sprintf(`
    %s create%s() {
        %s result;
        return result;
    }`, structName, structName, structName)
	
	// Insert after the struct definition
	insertPosition := protocol.Position{
		Line:      uint32(structNode.EndPosition().Row + 1),
		Character: 0,
	}
	
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: insertPosition,
			End:   insertPosition,
		},
		NewText: constructor,
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindSource
	return &protocol.CodeAction{
		Title: fmt.Sprintf("Generate constructor for %s", structName),
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
	}
}

// createAddMissingIncludeAction creates an action to add missing include
func (c *CodeActionProvider) createAddMissingIncludeAction(doc *document.Document) *protocol.CodeAction {
	// This would analyze undefined symbols and suggest includes
	// For now, we'll provide a generic include template
	
	includeStatement := "include \"common.frugal\"\n"
	
	// Insert at the top of the file (after initial comments)
	insertPosition := protocol.Position{Line: 0, Character: 0}
	
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: insertPosition,
			End:   insertPosition,
		},
		NewText: includeStatement,
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindSource
	return &protocol.CodeAction{
		Title: "Add common include",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
	}
}

// createOrganizeIncludesAction creates an action to organize includes
func (c *CodeActionProvider) createOrganizeIncludesAction(doc *document.Document) *protocol.CodeAction {
	// This would sort and deduplicate include statements
	// For now, we'll provide a placeholder
	
	kind := protocol.CodeActionKindSourceOrganizeImports
	return &protocol.CodeAction{
		Title: "Organize includes",
		Kind:  &kind,
		// Implementation would involve parsing includes and reordering them
	}
}

// createGenerateServiceAction creates an action to generate a service template
func (c *CodeActionProvider) createGenerateServiceAction(doc *document.Document) *protocol.CodeAction {
	serviceTemplate := `
service ExampleService {
    string ping(),
    
    void doSomething(1: string param) throws (1: Exception error)
}
`
	
	// Insert at the end of the document
	lines := strings.Split(string(doc.Content), "\n")
	insertPosition := protocol.Position{
		Line:      uint32(len(lines)),
		Character: 0,
	}
	
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: insertPosition,
			End:   insertPosition,
		},
		NewText: serviceTemplate,
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindSource
	return &protocol.CodeAction{
		Title: "Generate service template",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
	}
}

// createGenerateScopeAction creates an action to generate a scope template
func (c *CodeActionProvider) createGenerateScopeAction(doc *document.Document) *protocol.CodeAction {
	scopeTemplate := `
scope ExampleEvents prefix "example" {
    EventOccurred: EventData,
    StateChanged: StateInfo
}
`
	
	// Insert at the end of the document
	lines := strings.Split(string(doc.Content), "\n")
	insertPosition := protocol.Position{
		Line:      uint32(len(lines)),
		Character: 0,
	}
	
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: insertPosition,
			End:   insertPosition,
		},
		NewText: scopeTemplate,
	}
	
	changes := map[string][]protocol.TextEdit{
		doc.URI: {edit},
	}
	
	kind := protocol.CodeActionKindSource
	return &protocol.CodeAction{
		Title: "Generate scope template",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
	}
}

// Helper methods

// isInServiceBody checks if a node is within a service body
func (c *CodeActionProvider) isInServiceBody(node *tree_sitter.Node) bool {
	current := node
	for current != nil {
		if current.Kind() == "service_body" {
			return true
		}
		current = current.Parent()
	}
	return false
}

// isInStructBody checks if a node is within a struct body
func (c *CodeActionProvider) isInStructBody(node *tree_sitter.Node) bool {
	current := node
	for current != nil {
		if current.Kind() == "struct_body" {
			return true
		}
		current = current.Parent()
	}
	return false
}

// findParentOfType finds the first parent node of the specified type
func (c *CodeActionProvider) findParentOfType(node *tree_sitter.Node, nodeType string) *tree_sitter.Node {
	current := node.Parent()
	for current != nil {
		if current.Kind() == nodeType {
			return current
		}
		current = current.Parent()
	}
	return nil
}

// getNextFieldID determines the next field ID for a struct
func (c *CodeActionProvider) getNextFieldID(structNode *tree_sitter.Node, source []byte) int {
	maxID := 0
	
	// Walk through struct body to find existing field IDs
	c.walkNode(structNode, func(node *tree_sitter.Node) bool {
		if node.Kind() == "field_id" {
			idText := ast.GetText(node, source)
			if id, err := strconv.Atoi(strings.TrimSuffix(idText, ":")); err == nil {
				if id > maxID {
					maxID = id
				}
			}
		}
		return true // Continue walking
	})
	
	return maxID + 1
}

// extractIdentifier extracts the identifier name from a definition node
func (c *CodeActionProvider) extractIdentifier(node *tree_sitter.Node, source []byte) string {
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		return ast.GetText(nameNode, source)
	}
	return ""
}

// walkNode walks through all nodes in a tree
func (c *CodeActionProvider) walkNode(node *tree_sitter.Node, visitor func(*tree_sitter.Node) bool) {
	if node == nil {
		return
	}
	
	if !visitor(node) {
		return
	}
	
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		c.walkNode(child, visitor)
	}
}
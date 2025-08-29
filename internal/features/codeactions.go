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

const (
	nodeTypeStructDefinition   = "struct_definition"
	nodeTypeFunctionDefinition = "function_definition"
	nodeTypeFieldList          = "field_list"
	nodeTypeFieldType          = "field_type"
	nodeTypeIdentifier         = "identifier"
	nodeTypeThrows             = "throws"
	nodeTypeBaseType           = "base_type"
	nodeTypeTypeIdentifier     = "type_identifier"
	nodeTypeParameter          = "parameter"
	nodeTypeField              = "field"
	nodeTypeFieldID            = "field_id"
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

	// Extract parameter list to struct
	if c.isMethodWithMultipleParameters(node) {
		action := c.createExtractParametersToStructAction(doc, rng, node)
		if action != nil {
			actions = append(actions, *action)
		}
	}

	// Generate constructor for struct
	if node.Kind() == nodeTypeStructDefinition {
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
func (c *CodeActionProvider) createExtractMethodAction(doc *document.Document, _ protocol.Range, node *tree_sitter.Node) *protocol.CodeAction {
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
func (c *CodeActionProvider) createAddFieldAction(doc *document.Document, _ protocol.Range, node *tree_sitter.Node) *protocol.CodeAction {
	structNode := c.findParentOfType(node, nodeTypeStructDefinition)
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
func (c *CodeActionProvider) createOrganizeIncludesAction(_ *document.Document) *protocol.CodeAction {
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
		if node.Kind() == nodeTypeFieldID {
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
	nameNode := ast.FindNodeByType(node, nodeTypeIdentifier)
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

// isMethodWithMultipleParameters checks if a node is a method definition with multiple parameters
func (c *CodeActionProvider) isMethodWithMultipleParameters(node *tree_sitter.Node) bool {
	// Find the function definition node
	functionNode := c.findParentOfType(node, nodeTypeFunctionDefinition)
	if functionNode == nil && node.Kind() == nodeTypeFunctionDefinition {
		functionNode = node
	}
	if functionNode == nil {
		return false
	}

	// Count parameters (they are represented as field nodes in the first field_list)
	parameterCount := 0
	childCount := functionNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := functionNode.Child(i)
		if child.Kind() == nodeTypeFieldList {
			// This should be the parameter list (comes before throws)
			fieldCount := child.ChildCount()
			for j := uint(0); j < fieldCount; j++ {
				field := child.Child(j)
				if field.Kind() == nodeTypeField {
					parameterCount++
				}
			}
			break // Stop after first field_list
		}
	}

	return parameterCount >= 2 // Only suggest extraction for 2+ parameters
}

// createExtractParametersToStructAction creates a refactoring to extract parameters into a struct
func (c *CodeActionProvider) createExtractParametersToStructAction(doc *document.Document, _ protocol.Range, node *tree_sitter.Node) *protocol.CodeAction {
	// Find the function definition node
	functionNode := c.findParentOfType(node, nodeTypeFunctionDefinition)
	if functionNode == nil && node.Kind() == nodeTypeFunctionDefinition {
		functionNode = node
	}
	if functionNode == nil {
		return nil
	}

	// Extract method name
	methodName := c.extractMethodName(functionNode, doc.Content)
	if methodName == "" {
		return nil
	}

	// Extract parameters
	parameters := c.extractParameters(functionNode, doc.Content)
	if len(parameters) < 2 {
		return nil
	}

	// Generate struct name (e.g., "GetUserRequest")
	structName := c.generateStructName(methodName)

	// Create struct definition
	structDef := c.generateParameterStruct(structName, parameters)

	// Create updated method signature
	updatedMethod := c.generateUpdatedMethodSignature(functionNode, doc.Content, structName)

	// Create edits
	var edits []protocol.TextEdit

	// Add struct definition before the service
	serviceNode := c.findParentOfType(functionNode, "service_definition")
	if serviceNode != nil {
		structInsertPos := protocol.Position{
			Line:      uint32(serviceNode.StartPosition().Row),
			Character: 0,
		}
		edits = append(edits, protocol.TextEdit{
			Range: protocol.Range{
				Start: structInsertPos,
				End:   structInsertPos,
			},
			NewText: structDef + "\n\n",
		})
	}

	// Replace method signature
	methodRange := protocol.Range{
		Start: protocol.Position{
			Line:      uint32(functionNode.StartPosition().Row),
			Character: uint32(functionNode.StartPosition().Column),
		},
		End: protocol.Position{
			Line:      uint32(functionNode.EndPosition().Row),
			Character: uint32(functionNode.EndPosition().Column),
		},
	}

	edits = append(edits, protocol.TextEdit{
		Range:   methodRange,
		NewText: updatedMethod,
	})

	changes := map[string][]protocol.TextEdit{
		doc.URI: edits,
	}

	kind := protocol.CodeActionKindRefactor
	return &protocol.CodeAction{
		Title: fmt.Sprintf("Extract parameters to %s struct", structName),
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: changes,
		},
	}
}

// Parameter represents a method parameter
type Parameter struct {
	ID   string
	Type string
	Name string
}

// extractMethodName extracts the method name from a function definition
func (c *CodeActionProvider) extractMethodName(functionNode *tree_sitter.Node, source []byte) string {
	var methodName string
	c.walkNode(functionNode, func(n *tree_sitter.Node) bool {
		if n.Kind() == nodeTypeIdentifier && n.Parent().Kind() == nodeTypeFunctionDefinition {
			methodName = ast.GetText(n, source)
			return false // Stop walking once we find it
		}
		return true
	})
	return methodName
}

// extractParameters extracts parameters from a function definition
func (c *CodeActionProvider) extractParameters(functionNode *tree_sitter.Node, source []byte) []Parameter {
	var parameters []Parameter

	// Find the first field_list (parameter list), not the throws field_list
	childCount := functionNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := functionNode.Child(i)
		if child.Kind() == nodeTypeFieldList {
			// This should be the parameter list (comes before throws)
			fieldCount := child.ChildCount()
			for j := uint(0); j < fieldCount; j++ {
				field := child.Child(j)
				if field.Kind() == nodeTypeField {
					param := c.parseParameter(field, source)
					if param.Name != "" {
						parameters = append(parameters, param)
					}
				}
			}
			break // Stop after first field_list
		}
	}

	return parameters
}

// parseParameter parses a single parameter node (field node in function signature)
func (c *CodeActionProvider) parseParameter(fieldNode *tree_sitter.Node, source []byte) Parameter {
	param := Parameter{}

	// Walk through field children to extract ID, type, and name
	childCount := fieldNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := fieldNode.Child(i)

		switch child.Kind() {
		case nodeTypeFieldID:
			// Extract the number from field_id (e.g., "1:" -> "1")
			idText := ast.GetText(child, source)
			param.ID = strings.TrimSuffix(idText, ":")
		case nodeTypeFieldType:
			// Extract type from field_type node
			param.Type = c.extractTypeFromFieldType(child, source)
		case nodeTypeIdentifier:
			// This should be the parameter name
			param.Name = ast.GetText(child, source)
		}
	}

	return param
}

// extractTypeFromFieldType extracts the type string from a field_type node
func (c *CodeActionProvider) extractTypeFromFieldType(fieldTypeNode *tree_sitter.Node, source []byte) string {
	// Look for base_type within field_type
	childCount := fieldTypeNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := fieldTypeNode.Child(i)
		if child.Kind() == nodeTypeBaseType {
			return c.extractTypeFromBaseType(child, source)
		}
	}
	return ast.GetText(fieldTypeNode, source)
}

// extractTypeFromBaseType extracts the actual type from a base_type node
func (c *CodeActionProvider) extractTypeFromBaseType(baseTypeNode *tree_sitter.Node, source []byte) string {
	// The base_type contains the actual type keyword (string, i64, bool, etc.)
	childCount := baseTypeNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := baseTypeNode.Child(i)
		childKind := child.Kind()
		// Look for primitive type nodes
		if childKind == "string" || childKind == "i64" || childKind == "bool" ||
			childKind == "i32" || childKind == "double" || childKind == "byte" {
			return ast.GetText(child, source)
		}
	}
	return ast.GetText(baseTypeNode, source)
}

// generateStructName generates a struct name from method name (e.g., "getUser" -> "GetUserRequest")
func (c *CodeActionProvider) generateStructName(methodName string) string {
	// Capitalize first letter
	if len(methodName) == 0 {
		return "Request"
	}

	capitalized := strings.ToUpper(methodName[:1]) + methodName[1:]
	return capitalized + "Request"
}

// generateParameterStruct generates the struct definition from parameters
func (c *CodeActionProvider) generateParameterStruct(structName string, parameters []Parameter) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("struct %s {\n", structName))

	for _, param := range parameters {
		builder.WriteString(fmt.Sprintf("    %s: %s %s,\n", param.ID, param.Type, param.Name))
	}

	builder.WriteString("}")

	return builder.String()
}

// generateUpdatedMethodSignature generates the updated method signature using the struct
func (c *CodeActionProvider) generateUpdatedMethodSignature(functionNode *tree_sitter.Node, source []byte, structName string) string {
	// Extract return type and method name
	methodName := c.extractMethodName(functionNode, source)
	returnType := c.extractReturnType(functionNode, source)
	throwsClause := c.extractThrowsClause(functionNode, source)

	// Generate new signature
	signature := fmt.Sprintf("    %s %s(1: %s request)", returnType, methodName, structName)

	if throwsClause != "" {
		signature += " " + throwsClause
	}

	return signature
}

// extractReturnType extracts the return type from a function definition
func (c *CodeActionProvider) extractReturnType(functionNode *tree_sitter.Node, source []byte) string {
	// Look for function_type node which contains the return type
	childCount := functionNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := functionNode.Child(i)
		if child.Kind() == "function_type" {
			return c.extractTypeFromFieldType(child, source)
		}
	}

	return "void" // default
}

// extractThrowsClause extracts the throws clause if present
func (c *CodeActionProvider) extractThrowsClause(functionNode *tree_sitter.Node, source []byte) string {
	var throwsClause strings.Builder
	foundThrows := false

	childCount := functionNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := functionNode.Child(i)

		if child.Kind() == nodeTypeThrows {
			foundThrows = true
			throwsClause.WriteString("throws (")

			// Look for the next field_list (throws specifications)
			for j := i + 1; j < childCount; j++ {
				nextChild := functionNode.Child(j)
				if nextChild.Kind() == nodeTypeFieldList {
					// Extract throw specifications
					throwSpecs := c.extractThrowSpecs(nextChild, source)
					throwsClause.WriteString(throwSpecs)
					break
				}
			}
			throwsClause.WriteString(")")
			break
		}
	}

	if !foundThrows {
		return ""
	}

	return throwsClause.String()
}

// extractThrowSpecs extracts throw specifications from a field_list
func (c *CodeActionProvider) extractThrowSpecs(fieldListNode *tree_sitter.Node, source []byte) string {
	var specs []string

	fieldCount := fieldListNode.ChildCount()
	for i := uint(0); i < fieldCount; i++ {
		field := fieldListNode.Child(i)
		if field.Kind() == nodeTypeField {
			spec := c.extractThrowSpec(field, source)
			if spec != "" {
				specs = append(specs, spec)
			}
		}
	}

	return strings.Join(specs, ", ")
}

// extractThrowSpec extracts a single throw specification
func (c *CodeActionProvider) extractThrowSpec(fieldNode *tree_sitter.Node, source []byte) string {
	var id, exceptionType, name string

	childCount := fieldNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := fieldNode.Child(i)

		switch child.Kind() {
		case nodeTypeFieldID:
			id = ast.GetText(child, source)
		case nodeTypeFieldType:
			exceptionType = c.extractTypeFromFieldType(child, source)
		case nodeTypeIdentifier:
			name = ast.GetText(child, source)
		}
	}

	if id != "" && exceptionType != "" && name != "" {
		return fmt.Sprintf("%s %s %s", id, exceptionType, name)
	}

	return ""
}

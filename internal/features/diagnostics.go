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

// DiagnosticsProvider provides comprehensive diagnostics for Frugal files
type DiagnosticsProvider struct{}

// NewDiagnosticsProvider creates a new diagnostics provider
func NewDiagnosticsProvider() *DiagnosticsProvider {
	return &DiagnosticsProvider{}
}

// ProvideDiagnostics analyzes a document and returns diagnostics
func (d *DiagnosticsProvider) ProvideDiagnostics(doc *document.Document) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	if doc.ParseResult == nil {
		return diagnostics
	}

	// Add parse error diagnostics
	diagnostics = append(diagnostics, d.getParseErrorDiagnostics(doc)...)

	// Add semantic validation diagnostics if parsing succeeded
	if doc.ParseResult.GetRootNode() != nil {
		diagnostics = append(diagnostics, d.getSemanticDiagnostics(doc)...)
	}

	return diagnostics
}

// getParseErrorDiagnostics converts parse errors to diagnostics
func (d *DiagnosticsProvider) getParseErrorDiagnostics(doc *document.Document) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	for _, err := range doc.ParseResult.Errors {
		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(err.Line),
					Character: uint32(err.Column),
				},
				End: protocol.Position{
					Line:      uint32(err.Line),
					Character: uint32(err.Column + 1),
				},
			},
			Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
			Source:   &[]string{"frugal-ls"}[0],
			Message:  err.Message,
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics
}

// getSemanticDiagnostics performs semantic validation
func (d *DiagnosticsProvider) getSemanticDiagnostics(doc *document.Document) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	root := doc.ParseResult.GetRootNode()
	if root == nil {
		return diagnostics
	}

	// Check for various semantic issues
	diagnostics = append(diagnostics, d.checkDuplicateDefinitions(doc, root)...)
	diagnostics = append(diagnostics, d.checkFieldIdValidation(doc, root)...)
	diagnostics = append(diagnostics, d.checkUnusedImports(doc, root)...)
	diagnostics = append(diagnostics, d.checkNamingConventions(doc, root)...)
	diagnostics = append(diagnostics, d.checkTypeReferences(doc, root)...)

	return diagnostics
}

// checkDuplicateDefinitions checks for duplicate struct, service, enum names
func (d *DiagnosticsProvider) checkDuplicateDefinitions(doc *document.Document, root *tree_sitter.Node) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)
	seenNames := make(map[string]*tree_sitter.Node)

	d.walkDefinitions(root, doc.Content, func(defType, name string, node *tree_sitter.Node) {
		if name == "" {
			return
		}

		key := defType + ":" + name
		if existingNode, exists := seenNames[key]; exists {
			// Create diagnostic for duplicate definition
			diagnostic := protocol.Diagnostic{
				Range:    d.nodeToRange(node, doc.Content),
				Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
				Source:   &[]string{"frugal-ls"}[0],
				Message:  fmt.Sprintf("Duplicate %s definition '%s'", defType, name),
				RelatedInformation: []protocol.DiagnosticRelatedInformation{{
					Location: protocol.Location{
						URI:   doc.URI,
						Range: d.nodeToRange(existingNode, doc.Content),
					},
					Message: fmt.Sprintf("First definition of '%s' here", name),
				}},
			}
			diagnostics = append(diagnostics, diagnostic)
		} else {
			seenNames[key] = node
		}
	})

	return diagnostics
}

// checkFieldIdValidation validates field IDs in structs and methods
func (d *DiagnosticsProvider) checkFieldIdValidation(doc *document.Document, root *tree_sitter.Node) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	// Find all structs, exceptions, and service methods
	d.walkNodes(root, func(node *tree_sitter.Node) {
		nodeType := node.Kind()
		
		if nodeType == "struct_definition" || nodeType == "exception_definition" {
			diagnostics = append(diagnostics, d.validateStructFields(doc, node)...)
		} else if nodeType == "function_definition" {
			diagnostics = append(diagnostics, d.validateMethodFields(doc, node)...)
		}
	})

	return diagnostics
}

// validateStructFields checks field IDs in struct/exception definitions
func (d *DiagnosticsProvider) validateStructFields(doc *document.Document, structNode *tree_sitter.Node) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)
	seenFieldIds := make(map[int]*tree_sitter.Node)

	// Find struct body
	structBody := ast.FindNodeByType(structNode, "struct_body")
	if structBody == nil {
		return diagnostics
	}

	// Check each field
	childCount := structBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := structBody.Child(i)
		if child.Kind() == "field" {
			fieldId, fieldIdNode := d.extractFieldId(child, doc.Content)
			if fieldId == 0 {
				continue // Skip if we couldn't extract field ID
			}

			if existingNode, exists := seenFieldIds[fieldId]; exists {
				diagnostic := protocol.Diagnostic{
					Range:    d.nodeToRange(fieldIdNode, doc.Content),
					Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
					Source:   &[]string{"frugal-ls"}[0],
					Message:  fmt.Sprintf("Duplicate field ID %d", fieldId),
					RelatedInformation: []protocol.DiagnosticRelatedInformation{{
						Location: protocol.Location{
							URI:   doc.URI,
							Range: d.nodeToRange(existingNode, doc.Content),
						},
						Message: fmt.Sprintf("Field ID %d first used here", fieldId),
					}},
				}
				diagnostics = append(diagnostics, diagnostic)
			} else {
				seenFieldIds[fieldId] = fieldIdNode
			}

			// Validate field ID is positive
			if fieldId < 1 {
				diagnostic := protocol.Diagnostic{
					Range:    d.nodeToRange(fieldIdNode, doc.Content),
					Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
					Source:   &[]string{"frugal-ls"}[0],
					Message:  fmt.Sprintf("Field ID must be positive, got %d", fieldId),
				}
				diagnostics = append(diagnostics, diagnostic)
			}
		}
	}

	return diagnostics
}

// validateMethodFields checks field IDs in method parameters and throws
func (d *DiagnosticsProvider) validateMethodFields(doc *document.Document, methodNode *tree_sitter.Node) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	// Find all field_list nodes and determine their context
	var foundFieldLists []*tree_sitter.Node
	d.walkNodes(methodNode, func(node *tree_sitter.Node) {
		if node.Kind() == "field_list" {
			foundFieldLists = append(foundFieldLists, node)
		}
	})

	// Process each field_list - determine context by position and preceding siblings
	for _, fieldList := range foundFieldLists {
		context := d.determineFieldListContext(fieldList)
		diagnostics = append(diagnostics, d.validateSingleFieldList(doc, fieldList, context)...)
	}

	return diagnostics
}

// determineFieldListContext determines if a field_list is parameters or throws
func (d *DiagnosticsProvider) determineFieldListContext(fieldList *tree_sitter.Node) string {
	parent := fieldList.Parent()
	if parent == nil || parent.Kind() != "function_definition" {
		return "parameter list" // Default
	}

	// Find the position of this field_list and check for throws before it
	childCount := parent.ChildCount()
	var fieldListPosition uint = childCount // Initialize to impossible value
	var throwsPosition uint = childCount    // Initialize to impossible value

	// First pass: find positions
	for i := uint(0); i < childCount; i++ {
		child := parent.Child(i)
		if child == fieldList {
			fieldListPosition = i
		}
		if child.Kind() == "throws" {
			throwsPosition = i
		}
	}

	// If throws comes before this field_list, it's a throws list
	if throwsPosition < fieldListPosition {
		return "throws list"
	}

	return "parameter list"
}

// validateFieldList validates field IDs in a field list (parameters, throws, etc.)
func (d *DiagnosticsProvider) validateFieldList(doc *document.Document, parentNode *tree_sitter.Node, listType string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)
	seenFieldIds := make(map[int]*tree_sitter.Node)

	// Find all field lists of the specified type
	d.walkNodes(parentNode, func(node *tree_sitter.Node) {
		if node.Kind() == listType {
			childCount := node.ChildCount()
			for i := uint(0); i < childCount; i++ {
				child := node.Child(i)
				if child.Kind() == "field" {
					fieldId, fieldIdNode := d.extractFieldId(child, doc.Content)
					if fieldId == 0 {
						continue
					}

					if existingNode, exists := seenFieldIds[fieldId]; exists {
						diagnostic := protocol.Diagnostic{
							Range:    d.nodeToRange(fieldIdNode, doc.Content),
							Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
							Source:   &[]string{"frugal-ls"}[0],
							Message:  fmt.Sprintf("Duplicate field ID %d in parameter list", fieldId),
							RelatedInformation: []protocol.DiagnosticRelatedInformation{{
								Location: protocol.Location{
									URI:   doc.URI,
									Range: d.nodeToRange(existingNode, doc.Content),
								},
								Message: fmt.Sprintf("Field ID %d first used here", fieldId),
							}},
						}
						diagnostics = append(diagnostics, diagnostic)
					} else {
						seenFieldIds[fieldId] = fieldIdNode
					}
				}
			}
		}
	})

	return diagnostics
}

// validateSingleFieldList validates field IDs within a single field list
func (d *DiagnosticsProvider) validateSingleFieldList(doc *document.Document, fieldListNode *tree_sitter.Node, contextName string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)
	seenFieldIds := make(map[int]*tree_sitter.Node)

	childCount := fieldListNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := fieldListNode.Child(i)
		if child.Kind() == "field" {
			fieldId, fieldIdNode := d.extractFieldId(child, doc.Content)
			if fieldId == 0 {
				continue
			}

			if existingNode, exists := seenFieldIds[fieldId]; exists {
				diagnostic := protocol.Diagnostic{
					Range:    d.nodeToRange(fieldIdNode, doc.Content),
					Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
					Source:   &[]string{"frugal-ls"}[0],
					Message:  fmt.Sprintf("Duplicate field ID %d in %s", fieldId, contextName),
					RelatedInformation: []protocol.DiagnosticRelatedInformation{{
						Location: protocol.Location{
							URI:   doc.URI,
							Range: d.nodeToRange(existingNode, doc.Content),
						},
						Message: fmt.Sprintf("Field ID %d first used here", fieldId),
					}},
				}
				diagnostics = append(diagnostics, diagnostic)
			} else {
				seenFieldIds[fieldId] = fieldIdNode
			}
		}
	}

	return diagnostics
}

// checkUnusedImports checks for unused include statements
func (d *DiagnosticsProvider) checkUnusedImports(doc *document.Document, root *tree_sitter.Node) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	// For now, just check if includes exist - full cross-file analysis would be more complex
	// This is a placeholder for future enhancement
	
	return diagnostics
}

// checkNamingConventions validates naming conventions
func (d *DiagnosticsProvider) checkNamingConventions(doc *document.Document, root *tree_sitter.Node) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	d.walkDefinitions(root, doc.Content, func(defType, name string, node *tree_sitter.Node) {
		if name == "" {
			return
		}

		var expectedPattern string
		var severity protocol.DiagnosticSeverity

		switch defType {
		case "service", "struct", "exception", "enum", "scope":
			// Should be PascalCase
			if !d.isPascalCase(name) {
				expectedPattern = "PascalCase"
				severity = protocol.DiagnosticSeverityWarning
			}
		case "const":
			// Should be UPPER_SNAKE_CASE
			if !d.isUpperSnakeCase(name) {
				expectedPattern = "UPPER_SNAKE_CASE"
				severity = protocol.DiagnosticSeverityWarning
			}
		}

		if expectedPattern != "" {
			diagnostic := protocol.Diagnostic{
				Range:    d.nodeToRange(node, doc.Content),
				Severity: &severity,
				Source:   &[]string{"frugal-ls"}[0],
				Message:  fmt.Sprintf("%s '%s' should follow %s naming convention", strings.Title(defType), name, expectedPattern),
			}
			diagnostics = append(diagnostics, diagnostic)
		}
	})

	return diagnostics
}

// checkTypeReferences validates that referenced types exist
func (d *DiagnosticsProvider) checkTypeReferences(doc *document.Document, root *tree_sitter.Node) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	// Collect all defined types
	definedTypes := d.collectDefinedTypes(root, doc.Content)

	// Check all type references
	d.walkNodes(root, func(node *tree_sitter.Node) {
		if node.Kind() == "field_type" {
			d.validateTypeReference(doc, node, definedTypes, &diagnostics)
		}
	})

	return diagnostics
}

// Helper methods

// walkDefinitions walks through all top-level definitions
func (d *DiagnosticsProvider) walkDefinitions(root *tree_sitter.Node, content []byte, callback func(defType, name string, node *tree_sitter.Node)) {
	d.walkNodes(root, func(node *tree_sitter.Node) {
		nodeType := node.Kind()
		var defType, name string
		var nameNode *tree_sitter.Node

		switch nodeType {
		case "service_definition":
			defType = "service"
			nameNode = ast.FindNodeByType(node, "identifier")
		case "struct_definition":
			defType = "struct"
			nameNode = ast.FindNodeByType(node, "identifier")
		case "exception_definition":
			defType = "exception"
			nameNode = ast.FindNodeByType(node, "identifier")
		case "enum_definition":
			defType = "enum"
			nameNode = ast.FindNodeByType(node, "identifier")
		case "scope_definition":
			defType = "scope"
			nameNode = ast.FindNodeByType(node, "identifier")
		case "const_definition":
			defType = "const"
			nameNode = ast.FindNodeByType(node, "identifier")
		case "typedef_definition":
			defType = "typedef"
			// For typedef, find the alias name (second identifier)
			identifiers := d.findAllNodes(node, "identifier")
			if len(identifiers) > 0 {
				nameNode = identifiers[len(identifiers)-1] // Last identifier is the alias
			}
		}

		if nameNode != nil {
			name = ast.GetText(nameNode, content)
			callback(defType, name, nameNode)
		}
	})
}

// walkNodes recursively walks all nodes in the tree
func (d *DiagnosticsProvider) walkNodes(node *tree_sitter.Node, callback func(*tree_sitter.Node)) {
	if node == nil {
		return
	}

	callback(node)

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		d.walkNodes(child, callback)
	}
}

// findAllNodes finds all nodes of a specific type
func (d *DiagnosticsProvider) findAllNodes(root *tree_sitter.Node, nodeType string) []*tree_sitter.Node {
	var nodes []*tree_sitter.Node
	
	d.walkNodes(root, func(node *tree_sitter.Node) {
		if node.Kind() == nodeType {
			nodes = append(nodes, node)
		}
	})
	
	return nodes
}

// extractFieldId extracts field ID from a field node
func (d *DiagnosticsProvider) extractFieldId(fieldNode *tree_sitter.Node, content []byte) (int, *tree_sitter.Node) {
	fieldIdNode := ast.FindNodeByType(fieldNode, "field_id")
	if fieldIdNode == nil {
		return 0, nil
	}

	integerNode := ast.FindNodeByType(fieldIdNode, "integer")
	if integerNode == nil {
		return 0, nil
	}

	idText := ast.GetText(integerNode, content)
	if id, err := strconv.Atoi(idText); err == nil {
		return id, integerNode
	}

	return 0, nil
}

// collectDefinedTypes collects all defined type names
func (d *DiagnosticsProvider) collectDefinedTypes(root *tree_sitter.Node, content []byte) map[string]bool {
	definedTypes := make(map[string]bool)

	// Add built-in types
	builtinTypes := []string{
		"void", "bool", "byte", "i8", "i16", "i32", "i64", "double", "string", "binary",
		"list", "set", "map",
	}
	for _, t := range builtinTypes {
		definedTypes[t] = true
	}

	// Add user-defined types
	d.walkDefinitions(root, content, func(defType, name string, node *tree_sitter.Node) {
		if defType == "struct" || defType == "exception" || defType == "enum" || defType == "typedef" {
			definedTypes[name] = true
		}
	})

	return definedTypes
}

// validateTypeReference validates a single type reference
func (d *DiagnosticsProvider) validateTypeReference(doc *document.Document, typeNode *tree_sitter.Node, definedTypes map[string]bool, diagnostics *[]protocol.Diagnostic) {
	// Extract the base type name
	typeName := d.extractTypeName(typeNode, doc.Content)
	if typeName == "" || definedTypes[typeName] {
		return // Valid type or couldn't extract name
	}

	// Check if it might be a container type
	if d.isContainerType(typeNode) {
		return // Container types have their own validation
	}

	diagnostic := protocol.Diagnostic{
		Range:    d.nodeToRange(typeNode, doc.Content),
		Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
		Source:   &[]string{"frugal-ls"}[0],
		Message:  fmt.Sprintf("Unknown type '%s'", typeName),
	}
	*diagnostics = append(*diagnostics, diagnostic)
}

// extractTypeName extracts the main type name from a field_type node
func (d *DiagnosticsProvider) extractTypeName(typeNode *tree_sitter.Node, content []byte) string {
	// Look for identifier or base_type
	if identifier := ast.FindNodeByType(typeNode, "identifier"); identifier != nil {
		return ast.GetText(identifier, content)
	}
	
	if baseType := ast.FindNodeByType(typeNode, "base_type"); baseType != nil {
		return ast.GetText(baseType, content)
	}

	return ""
}

// isContainerType checks if this is a container type (list, set, map)
func (d *DiagnosticsProvider) isContainerType(typeNode *tree_sitter.Node) bool {
	return ast.FindNodeByType(typeNode, "container_type") != nil
}

// nodeToRange converts a tree-sitter node to an LSP range
func (d *DiagnosticsProvider) nodeToRange(node *tree_sitter.Node, content []byte) protocol.Range {
	startByte := node.StartByte()
	endByte := node.EndByte()
	
	// Convert byte offsets to line/column positions
	startPos := d.byteOffsetToPosition(content, uint32(startByte))
	endPos := d.byteOffsetToPosition(content, uint32(endByte))

	return protocol.Range{
		Start: startPos,
		End:   endPos,
	}
}

// byteOffsetToPosition converts a byte offset to line/column position
func (d *DiagnosticsProvider) byteOffsetToPosition(content []byte, offset uint32) protocol.Position {
	line := uint32(0)
	col := uint32(0)
	
	for i := uint32(0); i < offset && i < uint32(len(content)); i++ {
		if content[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	
	return protocol.Position{
		Line:      line,
		Character: col,
	}
}

// Naming convention helpers

// isPascalCase checks if a string follows PascalCase convention
func (d *DiagnosticsProvider) isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	// First character must be uppercase
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	
	// No underscores or spaces allowed
	if strings.Contains(s, "_") || strings.Contains(s, " ") {
		return false
	}
	
	// Must contain at least one lowercase letter (not all uppercase)
	hasLowercase := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			hasLowercase = true
			break
		}
	}
	
	return hasLowercase
}

// isUpperSnakeCase checks if a string follows UPPER_SNAKE_CASE convention
func (d *DiagnosticsProvider) isUpperSnakeCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	prevWasUnderscore := false
	for i, r := range s {
		if r >= 'a' && r <= 'z' {
			return false // No lowercase letters
		}
		if r >= 'A' && r <= 'Z' {
			prevWasUnderscore = false
			continue // Uppercase is fine
		}
		if r >= '0' && r <= '9' {
			prevWasUnderscore = false
			continue // Numbers are fine
		}
		if r == '_' {
			// Underscores are fine, but not at start/end or consecutive
			if i == 0 || i == len(s)-1 || prevWasUnderscore {
				return false
			}
			prevWasUnderscore = true
			continue
		}
		return false // Other characters not allowed
	}
	
	return true
}
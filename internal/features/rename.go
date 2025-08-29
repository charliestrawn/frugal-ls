package features

import (
	"fmt"
	"regexp"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// RenameProvider handles rename operations for Frugal files
type RenameProvider struct {
	referencesProvider *ReferencesProvider
}

// NewRenameProvider creates a new rename provider
func NewRenameProvider() *RenameProvider {
	return &RenameProvider{
		referencesProvider: NewReferencesProvider(),
	}
}

// PrepareRename handles textDocument/prepareRename requests
func (r *RenameProvider) PrepareRename(doc *document.Document, position protocol.Position) (*protocol.Range, error) {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return nil, nil
	}

	// Find the symbol at the position
	symbolInfo := r.findSymbolAt(doc, position)
	if symbolInfo == nil {
		return nil, fmt.Errorf("no renameable symbol found at position")
	}

	// Check if the symbol is renameable
	if !r.isRenameable(symbolInfo) {
		return nil, fmt.Errorf("symbol %s cannot be renamed", symbolInfo.Name)
	}

	// Return the range of the symbol
	return &symbolInfo.Range, nil
}

// Rename handles textDocument/rename requests
func (r *RenameProvider) Rename(doc *document.Document, position protocol.Position, newName string, allDocuments map[string]*document.Document) (*protocol.WorkspaceEdit, error) {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return nil, nil
	}

	// Validate the new name
	if err := r.validateNewName(newName); err != nil {
		return nil, err
	}

	// Find the symbol at the position
	symbolInfo := r.findSymbolAt(doc, position)
	if symbolInfo == nil {
		return nil, fmt.Errorf("no renameable symbol found at position")
	}

	// Check if the symbol is renameable
	if !r.isRenameable(symbolInfo) {
		return nil, fmt.Errorf("symbol %s cannot be renamed", symbolInfo.Name)
	}

	// Find all references to this symbol
	references, err := r.referencesProvider.ProvideReferences(doc, position, true, allDocuments)
	if err != nil {
		return nil, fmt.Errorf("failed to find references: %w", err)
	}

	// Check for naming conflicts
	if err := r.checkConflicts(symbolInfo, newName, allDocuments); err != nil {
		return nil, err
	}

	// Create workspace edit with text changes
	changes := make(map[string][]protocol.TextEdit)

	for _, location := range references {
		uri := location.URI
		textEdit := protocol.TextEdit{
			Range:   location.Range,
			NewText: newName,
		}

		if _, exists := changes[uri]; !exists {
			changes[uri] = []protocol.TextEdit{}
		}
		changes[uri] = append(changes[uri], textEdit)
	}

	workspaceEdit := &protocol.WorkspaceEdit{
		Changes: changes,
	}

	return workspaceEdit, nil
}

// SymbolInfo represents information about a symbol found at a position
type SymbolInfo struct {
	Name    string
	Kind    string
	Range   protocol.Range
	Context string // Additional context like parent struct/service
}

// findSymbolAt finds the symbol at the given position
func (r *RenameProvider) findSymbolAt(doc *document.Document, position protocol.Position) *SymbolInfo {
	root := doc.ParseResult.GetRootNode()
	if root == nil {
		return nil
	}

	// Convert position to byte offset
	lines := strings.Split(string(doc.Content), "\n")
	if int(position.Line) >= len(lines) {
		return nil
	}

	line := lines[position.Line]
	if int(position.Character) > len(line) {
		return nil
	}

	byteOffset := 0
	for i := 0; i < int(position.Line); i++ {
		byteOffset += len(lines[i]) + 1 // +1 for newline
	}
	byteOffset += int(position.Character)

	// Find the smallest node containing this position
	node := r.findNodeAt(root, uint32(byteOffset))
	if node == nil {
		return nil
	}

	// Look for identifier or renameable nodes
	return r.extractSymbolInfo(node, doc.Content)
}

// findNodeAt finds the smallest node containing the byte offset
func (r *RenameProvider) findNodeAt(node *tree_sitter.Node, offset uint32) *tree_sitter.Node {
	if node == nil {
		return nil
	}

	// Check if offset is within this node
	if offset < uint32(node.StartByte()) || offset >= uint32(node.EndByte()) {
		return nil
	}

	// Check children first (find smallest containing node)
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if childNode := r.findNodeAt(child, offset); childNode != nil {
			return childNode
		}
	}

	// Return this node if no child contains the offset
	return node
}

// extractSymbolInfo extracts symbol information from a node
func (r *RenameProvider) extractSymbolInfo(node *tree_sitter.Node, source []byte) *SymbolInfo {
	if node == nil {
		return nil
	}

	nodeType := node.Kind()

	// Handle different node types that can be renamed
	switch nodeType {
	case "identifier":
		return r.extractIdentifierInfo(node, source)
	case "type_identifier":
		return r.extractTypeInfo(node, source)
	default:
		// Check if parent is a renameable construct
		parent := node.Parent()
		if parent != nil {
			return r.extractSymbolInfo(parent, source)
		}
	}

	return nil
}

// extractIdentifierInfo extracts information from identifier nodes
func (r *RenameProvider) extractIdentifierInfo(node *tree_sitter.Node, source []byte) *SymbolInfo {
	name := ast.GetText(node, source)
	parent := node.Parent()
	if parent == nil {
		return nil
	}

	startPos := node.StartPosition()
	endPos := node.EndPosition()
	symbolRange := protocol.Range{
		Start: protocol.Position{Line: uint32(startPos.Row), Character: uint32(startPos.Column)},
		End:   protocol.Position{Line: uint32(endPos.Row), Character: uint32(endPos.Column)},
	}

	parentType := parent.Kind()
	switch parentType {
	case "struct_definition":
		return &SymbolInfo{
			Name:    name,
			Kind:    "struct",
			Range:   symbolRange,
			Context: "definition",
		}
	case "service_definition":
		return &SymbolInfo{
			Name:    name,
			Kind:    "service",
			Range:   symbolRange,
			Context: "definition",
		}
	case "enum_definition":
		return &SymbolInfo{
			Name:    name,
			Kind:    "enum",
			Range:   symbolRange,
			Context: "definition",
		}
	case "exception_definition":
		return &SymbolInfo{
			Name:    name,
			Kind:    "exception",
			Range:   symbolRange,
			Context: "definition",
		}
	case "scope_definition":
		return &SymbolInfo{
			Name:    name,
			Kind:    "scope",
			Range:   symbolRange,
			Context: "definition",
		}
	case "const_definition":
		return &SymbolInfo{
			Name:    name,
			Kind:    "constant",
			Range:   symbolRange,
			Context: "definition",
		}
	case "typedef_definition":
		return &SymbolInfo{
			Name:    name,
			Kind:    "typedef",
			Range:   symbolRange,
			Context: "definition",
		}
	case nodeTypeFunctionDefinition:
		return &SymbolInfo{
			Name:    name,
			Kind:    "method",
			Range:   symbolRange,
			Context: "definition",
		}
	case nodeTypeField:
		return &SymbolInfo{
			Name:    name,
			Kind:    "field",
			Range:   symbolRange,
			Context: "field",
		}
	case "enum_field":
		return &SymbolInfo{
			Name:    name,
			Kind:    "enum_value",
			Range:   symbolRange,
			Context: "enum_field",
		}
	case nodeTypeParameter:
		return &SymbolInfo{
			Name:    name,
			Kind:    nodeTypeParameter,
			Range:   symbolRange,
			Context: nodeTypeParameter,
		}
	default:
		// Check if it's a type reference
		if r.isTypeReference(parent) {
			return &SymbolInfo{
				Name:    name,
				Kind:    "type_reference",
				Range:   symbolRange,
				Context: "reference",
			}
		}
	}

	return nil
}

// extractTypeInfo extracts information from type identifier nodes
func (r *RenameProvider) extractTypeInfo(node *tree_sitter.Node, source []byte) *SymbolInfo {
	name := ast.GetText(node, source)
	startPos := node.StartPosition()
	endPos := node.EndPosition()

	return &SymbolInfo{
		Name: name,
		Kind: "type_reference",
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(startPos.Row), Character: uint32(startPos.Column)},
			End:   protocol.Position{Line: uint32(endPos.Row), Character: uint32(endPos.Column)},
		},
		Context: "type",
	}
}

// isTypeReference checks if a node represents a type reference
func (r *RenameProvider) isTypeReference(node *tree_sitter.Node) bool {
	if node == nil {
		return false
	}

	nodeType := node.Kind()
	return nodeType == "field_type" || nodeType == "return_type" || nodeType == "type_reference"
}

// isRenameable checks if a symbol can be renamed
func (r *RenameProvider) isRenameable(symbol *SymbolInfo) bool {
	if symbol == nil {
		return false
	}

	// Built-in types cannot be renamed
	builtInTypes := map[string]bool{
		"bool": true, "byte": true, "i8": true, "i16": true, "i32": true, "i64": true,
		"double": true, "string": true, "binary": true, "uuid": true,
		"list": true, "set": true, "map": true,
	}

	if builtInTypes[symbol.Name] {
		return false
	}

	// Keywords cannot be renamed
	keywords := map[string]bool{
		"include": true, "namespace": true, "service": true, "scope": true,
		"struct": true, "enum": true, "exception": true, "const": true,
		"typedef": true, "throws": true, "extends": true, "oneway": true,
		"required": true, "optional": true, "prefix": true,
	}

	if keywords[symbol.Name] {
		return false
	}

	// All user-defined symbols can be renamed
	return true
}

// validateNewName checks if the new name is valid
func (r *RenameProvider) validateNewName(newName string) error {
	if strings.TrimSpace(newName) == "" {
		return fmt.Errorf("new name cannot be empty")
	}

	// Check if it's a valid identifier
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !validIdentifier.MatchString(newName) {
		return fmt.Errorf("'%s' is not a valid identifier", newName)
	}

	// Check if it's a reserved keyword
	keywords := map[string]bool{
		"include": true, "namespace": true, "service": true, "scope": true,
		"struct": true, "enum": true, "exception": true, "const": true,
		"typedef": true, "throws": true, "extends": true, "oneway": true,
		"required": true, "optional": true, "prefix": true,
		"bool": true, "byte": true, "i8": true, "i16": true, "i32": true, "i64": true,
		"double": true, "string": true, "binary": true, "uuid": true,
		"list": true, "set": true, "map": true,
	}

	if keywords[newName] {
		return fmt.Errorf("'%s' is a reserved keyword and cannot be used as an identifier", newName)
	}

	return nil
}

// checkConflicts checks for naming conflicts in the target scope
func (r *RenameProvider) checkConflicts(symbol *SymbolInfo, newName string, allDocuments map[string]*document.Document) error {
	// For now, do basic conflict detection
	// In a more sophisticated implementation, we would check:
	// 1. Same-scope naming conflicts (e.g., two structs with same name)
	// 2. Cross-file dependencies and conflicts
	// 3. Scope-specific rules (e.g., field names within a struct)

	// Basic check: ensure we're not renaming to the same name
	if symbol.Name == newName {
		return fmt.Errorf("new name '%s' is the same as current name", newName)
	}

	// TODO: Implement more sophisticated conflict detection
	// This would require analyzing the symbol's scope and checking for existing symbols
	// with the new name in the same scope.

	return nil
}

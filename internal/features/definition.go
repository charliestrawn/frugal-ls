package features

import (
	"path/filepath"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// DefinitionProvider handles go-to-definition functionality for Frugal symbols
type DefinitionProvider struct{}

// NewDefinitionProvider creates a new definition provider
func NewDefinitionProvider() *DefinitionProvider {
	return &DefinitionProvider{}
}

// ProvideDefinition provides definition locations for a symbol at the given position
func (d *DefinitionProvider) ProvideDefinition(doc *document.Document, position protocol.Position, allDocuments map[string]*document.Document) ([]protocol.Location, error) {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return nil, nil
	}

	// Validate position bounds
	lines := strings.Split(string(doc.Content), "\n")
	if int(position.Line) >= len(lines) {
		return nil, nil // Beyond last line
	}

	currentLine := lines[position.Line]
	if int(position.Character) > len(currentLine) {
		return nil, nil // Beyond line end
	}

	// Find the node at the position
	node := FindNodeAtPosition(doc.ParseResult.GetRootNode(), doc.Content, uint(position.Line), uint(position.Character))
	if node == nil {
		return nil, nil
	}

	// Get the symbol name from the node
	symbolName := d.extractSymbolName(node, doc.Content)
	if symbolName == "" {
		return nil, nil
	}

	// If it's a qualified name (e.g., common.Address), resolve it properly
	var unqualifiedName string
	var targetDocuments map[string]*document.Document

	if strings.Contains(symbolName, ".") {
		parts := strings.Split(symbolName, ".")
		if len(parts) == 2 {
			namespace := parts[0]      // e.g., "common"
			unqualifiedName = parts[1] // e.g., "Address"

			// Find documents that match this namespace (by filename stem)
			targetDocuments = d.filterDocumentsByNamespace(namespace, allDocuments)
		}
	} else {
		// Unqualified name - search everywhere
		unqualifiedName = symbolName
		targetDocuments = allDocuments
	}

	// Find definition in the current document first (if it's in our target set)
	if _, inTarget := targetDocuments[doc.URI]; inTarget {
		location := d.findDefinitionInDocument(unqualifiedName, doc)
		if location != nil {
			return []protocol.Location{*location}, nil
		}
	}

	// Search other target documents
	for docURI, otherDoc := range targetDocuments {
		if docURI == doc.URI || !otherDoc.IsValidFrugalFile() {
			continue
		}

		location := d.findDefinitionInDocument(unqualifiedName, otherDoc)
		if location != nil {
			return []protocol.Location{*location}, nil
		}
	}

	return nil, nil
}

// extractSymbolName extracts the symbol name from a node
func (d *DefinitionProvider) extractSymbolName(node *tree_sitter.Node, source []byte) string {
	nodeType := node.Kind()

	// For identifiers, check if parent is a qualified type (field_type)
	if nodeType == nodeTypeIdentifier {
		parent := node.Parent()
		if parent != nil && parent.Kind() == nodeTypeFieldType {
			// Get the full text of the parent to capture qualified names (e.g., common.Address)
			fullText := ast.GetText(parent, source)
			if strings.Contains(fullText, ".") {
				return fullText // Return qualified name
			}
		}
		return ast.GetText(node, source)
	}

	// For field_type nodes, get the full text to handle qualified names
	if nodeType == nodeTypeFieldType {
		fullText := ast.GetText(node, source)
		if strings.Contains(fullText, ".") {
			return fullText // Return qualified name
		}
		// Fall through to find identifier
	}

	// For other node types, try to find an identifier child
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		return ast.GetText(nameNode, source)
	}

	return ""
}

// findDefinitionInDocument finds the first definition of a symbol in a document
func (d *DefinitionProvider) findDefinitionInDocument(symbolName string, doc *document.Document) *protocol.Location {
	// Search through document symbols first - these are more accurate
	symbols := doc.GetSymbols()
	for _, symbol := range symbols {
		if symbol.Name == symbolName {
			return &protocol.Location{
				URI: doc.URI,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      uint32(symbol.Line),
						Character: uint32(symbol.Column),
					},
					End: protocol.Position{
						Line:      uint32(symbol.Line),
						Character: uint32(symbol.Column) + uint32(len(symbol.Name)),
					},
				},
			}
		}
	}

	// If no symbols found, fall back to AST search (less accurate but more comprehensive)
	if doc.ParseResult != nil && doc.ParseResult.GetRootNode() != nil {
		return d.searchASTForDefinition(symbolName, doc.ParseResult.GetRootNode(), doc)
	}

	return nil
}

// searchASTForDefinition searches the AST for the first symbol definition
func (d *DefinitionProvider) searchASTForDefinition(symbolName string, node *tree_sitter.Node, doc *document.Document) *protocol.Location {
	if node == nil {
		return nil
	}

	// Check if this node defines the symbol
	if d.isDefinitionNode(node, symbolName, doc.Content) {
		startPos := node.StartPosition()
		endPos := node.EndPosition()

		return &protocol.Location{
			URI: doc.URI,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(startPos.Row),
					Character: uint32(startPos.Column),
				},
				End: protocol.Position{
					Line:      uint32(endPos.Row),
					Character: uint32(endPos.Column),
				},
			},
		}
	}

	// Recursively search child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if location := d.searchASTForDefinition(symbolName, child, doc); location != nil {
			return location
		}
	}

	return nil
}

// isDefinitionNode checks if a node defines a particular symbol
func (d *DefinitionProvider) isDefinitionNode(node *tree_sitter.Node, symbolName string, source []byte) bool {
	nodeType := node.Kind()

	// Check for definition node types
	definitionTypes := map[string]bool{
		"service_definition":   true,
		"scope_definition":     true,
		"struct_definition":    true,
		"enum_definition":      true,
		"const_definition":     true,
		"typedef_definition":   true,
		"exception_definition": true,
		"method":               true,
		"field":                true,
		"enum_value":           true,
	}

	if !definitionTypes[nodeType] {
		return false
	}

	// Find the identifier in this definition
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode == nil {
		return false
	}

	definedName := ast.GetText(nameNode, source)
	return definedName == symbolName
}

// filterDocumentsByNamespace filters documents to those matching a namespace (filename stem)
func (d *DefinitionProvider) filterDocumentsByNamespace(namespace string, allDocuments map[string]*document.Document) map[string]*document.Document {
	filtered := make(map[string]*document.Document)

	for uri, doc := range allDocuments {
		// Extract the filename stem from the URI
		// e.g., "file:///path/to/common.frugal" -> "common"
		path := uri
		if strings.HasPrefix(path, "file://") {
			path = path[7:] // Remove "file://" prefix
		}

		filename := filepath.Base(path)
		stem := strings.TrimSuffix(filename, filepath.Ext(filename))

		// Match the namespace to the filename stem
		if stem == namespace {
			filtered[uri] = doc
		}
	}

	return filtered
}


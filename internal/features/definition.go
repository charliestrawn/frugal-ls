package features

import (
	"fmt"
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

	// Find definitions in the current document first
	locations := d.findDefinitionsInDocument(symbolName, doc)

	// If not found in current document, search other documents
	if len(locations) == 0 {
		for docURI, otherDoc := range allDocuments {
			if docURI == doc.URI || !otherDoc.IsValidFrugalFile() {
				continue
			}

			docLocations := d.findDefinitionsInDocument(symbolName, otherDoc)
			locations = append(locations, docLocations...)
		}
	}

	return locations, nil
}

// extractSymbolName extracts the symbol name from a node
func (d *DefinitionProvider) extractSymbolName(node *tree_sitter.Node, source []byte) string {
	nodeType := node.Kind()

	// For identifiers, use the text directly
	if nodeType == "identifier" {
		return ast.GetText(node, source)
	}

	// For other node types, try to find an identifier child
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		return ast.GetText(nameNode, source)
	}

	return ""
}

// findDefinitionsInDocument finds all definitions of a symbol in a document
func (d *DefinitionProvider) findDefinitionsInDocument(symbolName string, doc *document.Document) []protocol.Location {
	var locations []protocol.Location

	// Search through document symbols first - these are more accurate
	symbols := doc.GetSymbols()
	for _, symbol := range symbols {
		if symbol.Name == symbolName {
			location := protocol.Location{
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
			locations = append(locations, location)
		}
	}

	// If no symbols found, fall back to AST search (less accurate but more comprehensive)
	if len(locations) == 0 && doc.ParseResult != nil && doc.ParseResult.GetRootNode() != nil {
		locations = d.searchASTForDefinitions(symbolName, doc.ParseResult.GetRootNode(), doc)
	}

	// Deduplicate any remaining duplicates
	return d.deduplicateLocations(locations)
}

// searchASTForDefinitions searches the AST for symbol definitions
func (d *DefinitionProvider) searchASTForDefinitions(symbolName string, node *tree_sitter.Node, doc *document.Document) []protocol.Location {
	var locations []protocol.Location

	if node == nil {
		return locations
	}

	// Check if this node defines the symbol
	if d.isDefinitionNode(node, symbolName, doc.Content) {
		startPos := node.StartPosition()
		endPos := node.EndPosition()

		location := protocol.Location{
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
		locations = append(locations, location)
	}

	// Recursively search child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		childLocations := d.searchASTForDefinitions(symbolName, child, doc)
		locations = append(locations, childLocations...)
	}

	return locations
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

// ProvideReferences finds all references to a symbol (for future implementation)
func (d *DefinitionProvider) ProvideReferences(doc *document.Document, position protocol.Position, includeDeclaration bool, allDocuments map[string]*document.Document) ([]protocol.Location, error) {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return nil, nil
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

	var locations []protocol.Location

	// Search all documents for references
	for _, searchDoc := range allDocuments {
		if !searchDoc.IsValidFrugalFile() {
			continue
		}

		docLocations := d.findReferencesInDocument(symbolName, searchDoc, includeDeclaration)
		locations = append(locations, docLocations...)
	}

	// Also include the current document if not already included
	currentLocations := d.findReferencesInDocument(symbolName, doc, includeDeclaration)
	locations = append(locations, currentLocations...)

	return d.deduplicateLocations(locations), nil
}

// findReferencesInDocument finds all references to a symbol in a document
func (d *DefinitionProvider) findReferencesInDocument(symbolName string, doc *document.Document, includeDeclaration bool) []protocol.Location {
	var locations []protocol.Location

	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return locations
	}

	// Search the entire AST for identifier nodes matching the symbol name
	referenceLocs := d.searchASTForReferences(symbolName, doc.ParseResult.GetRootNode(), doc, includeDeclaration)
	locations = append(locations, referenceLocs...)

	return locations
}

// searchASTForReferences searches the AST for symbol references
func (d *DefinitionProvider) searchASTForReferences(symbolName string, node *tree_sitter.Node, doc *document.Document, includeDeclaration bool) []protocol.Location {
	var locations []protocol.Location

	if node == nil {
		return locations
	}

	nodeType := node.Kind()
	nodeText := ast.GetText(node, doc.Content)

	// Check if this is a reference to the symbol
	if nodeType == "identifier" && nodeText == symbolName {
		// Determine if this is a declaration or reference
		isDeclaration := d.isInDeclarationContext(node, doc.Content)

		if includeDeclaration || !isDeclaration {
			startPos := node.StartPosition()
			endPos := node.EndPosition()

			location := protocol.Location{
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
			locations = append(locations, location)
		}
	}

	// Recursively search child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		childLocations := d.searchASTForReferences(symbolName, child, doc, includeDeclaration)
		locations = append(locations, childLocations...)
	}

	return locations
}

// isInDeclarationContext determines if an identifier node is in a declaration context
func (d *DefinitionProvider) isInDeclarationContext(node *tree_sitter.Node, source []byte) bool {
	// Walk up the parent chain to find declaration contexts
	current := node
	for current != nil {
		parent := current.Parent()
		if parent == nil {
			break
		}

		parentType := parent.Kind()
		declarationTypes := map[string]bool{
			"service_definition":   true,
			"scope_definition":     true,
			"struct_definition":    true,
			"enum_definition":      true,
			"const_definition":     true,
			"typedef_definition":   true,
			"exception_definition": true,
		}

		if declarationTypes[parentType] {
			// Check if this identifier is the name being declared
			// (usually the first identifier in the definition)
			nameNode := ast.FindNodeByType(parent, "identifier")
			if nameNode == node {
				return true
			}
		}

		current = parent
	}

	return false
}

// deduplicateLocations removes duplicate locations from the list
func (d *DefinitionProvider) deduplicateLocations(locations []protocol.Location) []protocol.Location {
	seen := make(map[string]bool)
	var result []protocol.Location

	for _, loc := range locations {
		key := d.locationKey(loc)
		if !seen[key] {
			seen[key] = true
			result = append(result, loc)
		}
	}

	return result
}

// locationKey creates a unique key for a location
func (d *DefinitionProvider) locationKey(loc protocol.Location) string {
	return fmt.Sprintf("%s:%d:%d:%d:%d",
		loc.URI,
		loc.Range.Start.Line,
		loc.Range.Start.Character,
		loc.Range.End.Line,
		loc.Range.End.Character)
}

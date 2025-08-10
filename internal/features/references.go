package features

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
)

// ReferencesProvider provides find references functionality
type ReferencesProvider struct{}

// NewReferencesProvider creates a new ReferencesProvider
func NewReferencesProvider() *ReferencesProvider {
	return &ReferencesProvider{}
}

// ProvideReferences finds all references to the symbol at the given position
func (p *ReferencesProvider) ProvideReferences(doc *document.Document, position protocol.Position, includeDeclaration bool, allDocuments map[string]*document.Document) ([]protocol.Location, error) {
	if doc.ParseResult == nil || doc.ParseResult.Tree == nil {
		return []protocol.Location{}, nil
	}

	// Find the symbol at the cursor position
	symbol, symbolRange := p.getSymbolAtPosition(doc, position)
	if symbol == "" {
		return []protocol.Location{}, nil
	}

	var locations []protocol.Location
	seenLocations := make(map[string]bool) // Deduplicate locations

	// Search for references in all documents
	for uri, document := range allDocuments {
		if document.ParseResult == nil || document.ParseResult.Tree == nil {
			continue
		}

		refs := p.findReferencesInDocument(document, symbol)
		for _, ref := range refs {
			locationKey := p.locationKey(uri, ref)
			if seenLocations[locationKey] {
				continue // Skip duplicate
			}
			seenLocations[locationKey] = true

			// Skip declaration if not requested
			if !includeDeclaration && uri == doc.URI && symbolRange != nil &&
				p.rangesEqual(ref, *symbolRange) {
				continue
			}

			location := protocol.Location{
				URI:   uri,
				Range: ref,
			}
			locations = append(locations, location)
		}
	}

	return locations, nil
}

// getSymbolAtPosition finds the symbol (identifier) at the given position
func (p *ReferencesProvider) getSymbolAtPosition(doc *document.Document, position protocol.Position) (string, *protocol.Range) {
	root := doc.ParseResult.Tree.RootNode()

	// Convert position to byte offset
	lines := strings.Split(string(doc.Content), "\n")
	if int(position.Line) >= len(lines) {
		return "", nil
	}

	line := lines[position.Line]
	if int(position.Character) >= len(line) {
		return "", nil
	}

	byteOffset := 0
	for i := 0; i < int(position.Line); i++ {
		byteOffset += len(lines[i]) + 1 // +1 for newline
	}
	byteOffset += int(position.Character)

	// Find the node at this position
	node := p.findNodeAtOffset(root, uint(byteOffset))
	if node == nil {
		return "", nil
	}

	// Look for identifier node
	identifierNode := p.findIdentifierNode(node)
	if identifierNode == nil {
		return "", nil
	}

	// Extract the symbol text
	symbol := string(doc.Content[identifierNode.StartByte():identifierNode.EndByte()])

	// Convert node range back to LSP range
	symbolRange := p.nodeToRange(identifierNode, string(doc.Content))

	return symbol, symbolRange
}

// findNodeAtOffset finds the deepest node that contains the given byte offset
func (p *ReferencesProvider) findNodeAtOffset(node *tree_sitter.Node, offset uint) *tree_sitter.Node {
	if node.StartByte() > offset || node.EndByte() <= offset {
		return nil
	}

	// Check children first (deepest match)
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if result := p.findNodeAtOffset(child, offset); result != nil {
			return result
		}
	}

	return node
}

// findIdentifierNode finds an identifier node starting from the given node
func (p *ReferencesProvider) findIdentifierNode(node *tree_sitter.Node) *tree_sitter.Node {
	// Check if current node is an identifier or a type that should be treated as one
	if p.isSymbolNode(node) {
		return node
	}

	// Check immediate children for identifiers
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if p.isSymbolNode(child) {
			return child
		}
	}

	// Check parent for identifier, but only if we're directly inside it
	parent := node.Parent()
	if parent != nil && p.isSymbolNode(parent) {
		return parent
	}

	return nil
}

// isSymbolNode checks if a node represents a symbol that can be referenced
func (p *ReferencesProvider) isSymbolNode(node *tree_sitter.Node) bool {
	kind := node.Kind()
	return kind == "identifier" || kind == "base_type" || kind == "type_identifier"
}

// findReferencesInDocument finds all references to the given symbol in a document
func (p *ReferencesProvider) findReferencesInDocument(doc *document.Document, symbol string) []protocol.Range {
	var references []protocol.Range

	root := doc.ParseResult.Tree.RootNode()
	p.walkTreeForReferences(root, symbol, string(doc.Content), &references)

	return references
}

// walkTreeForReferences recursively walks the syntax tree looking for identifier references
func (p *ReferencesProvider) walkTreeForReferences(node *tree_sitter.Node, symbol string, content string, references *[]protocol.Range) {
	if node.Kind() == "identifier" {
		nodeText := content[node.StartByte():node.EndByte()]
		if nodeText == symbol {
			// Found a reference
			nodeRange := p.nodeToRange(node, content)
			if nodeRange != nil {
				*references = append(*references, *nodeRange)
			}
		}
	}

	// Recursively check child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		p.walkTreeForReferences(child, symbol, content, references)
	}
}

// nodeToRange converts a tree-sitter node to an LSP Range
func (p *ReferencesProvider) nodeToRange(node *tree_sitter.Node, content string) *protocol.Range {
	startPos := node.StartPosition()
	endPos := node.EndPosition()

	return &protocol.Range{
		Start: protocol.Position{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
		},
		End: protocol.Position{
			Line:      uint32(endPos.Row),
			Character: uint32(endPos.Column),
		},
	}
}

// getSymbolContext determines the context of a symbol (struct field, service method, etc.)
func (p *ReferencesProvider) getSymbolContext(node *tree_sitter.Node) string {
	parent := node.Parent()
	if parent == nil {
		return "unknown"
	}

	switch parent.Kind() {
	case "struct_definition":
		return "struct"
	case "service_definition":
		return "service"
	case "enum_definition":
		return "enum"
	case "exception_definition":
		return "exception"
	case "scope_definition":
		return "scope"
	case "function_definition":
		return "function"
	case "field":
		return "field"
	case "enum_field":
		return "enum_field"
	case "scope_operation":
		return "scope_operation"
	default:
		return "identifier"
	}
}

// locationKey creates a unique key for a location to avoid duplicates
func (p *ReferencesProvider) locationKey(uri string, r protocol.Range) string {
	return fmt.Sprintf("%s:%d:%d:%d:%d", uri, r.Start.Line, r.Start.Character, r.End.Line, r.End.Character)
}

// rangesEqual checks if two ranges are equal
func (p *ReferencesProvider) rangesEqual(r1, r2 protocol.Range) bool {
	return r1.Start.Line == r2.Start.Line &&
		r1.Start.Character == r2.Start.Character &&
		r1.End.Line == r2.End.Line &&
		r1.End.Character == r2.End.Character
}

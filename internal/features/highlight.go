package features

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
)

// DocumentHighlightProvider provides document highlight functionality
type DocumentHighlightProvider struct {
	referencesProvider *ReferencesProvider
}

// NewDocumentHighlightProvider creates a new DocumentHighlightProvider
func NewDocumentHighlightProvider() *DocumentHighlightProvider {
	return &DocumentHighlightProvider{
		referencesProvider: NewReferencesProvider(),
	}
}

// ProvideDocumentHighlight finds all occurrences of the symbol at the given position in the current document
func (p *DocumentHighlightProvider) ProvideDocumentHighlight(doc *document.Document, position protocol.Position) ([]protocol.DocumentHighlight, error) {
	if doc.ParseResult == nil || doc.ParseResult.Tree == nil {
		return []protocol.DocumentHighlight{}, nil
	}

	// Find the symbol at the cursor position
	symbol, _ := p.referencesProvider.getSymbolAtPosition(doc, position)
	if symbol == "" {
		return []protocol.DocumentHighlight{}, nil
	}

	var highlights []protocol.DocumentHighlight

	// Find all references in the current document only
	allDocuments := map[string]*document.Document{
		doc.URI: doc,
	}

	locations, err := p.referencesProvider.ProvideReferences(doc, position, true, allDocuments)
	if err != nil {
		return nil, err
	}

	// Convert locations to document highlights
	for _, location := range locations {
		if location.URI == doc.URI { // Only highlights in current document
			highlight := protocol.DocumentHighlight{
				Range: location.Range,
				Kind:  p.getHighlightKind(doc, location.Range, symbol),
			}
			highlights = append(highlights, highlight)
		}
	}

	return highlights, nil
}

// getHighlightKind determines the kind of highlight based on the symbol context
func (p *DocumentHighlightProvider) getHighlightKind(doc *document.Document, r protocol.Range, symbol string) *protocol.DocumentHighlightKind {
	// Convert range to byte offset to find the node
	content := string(doc.Content)
	lines := strings.Split(content, "\n")
	
	if int(r.Start.Line) >= len(lines) {
		return nil
	}

	byteOffset := 0
	for i := 0; i < int(r.Start.Line); i++ {
		byteOffset += len(lines[i]) + 1 // +1 for newline
	}
	byteOffset += int(r.Start.Character)

	// Find the node at this position
	root := doc.ParseResult.Tree.RootNode()
	node := p.findNodeAtOffset(root, uint(byteOffset))
	if node == nil {
		return nil
	}

	// Determine highlight kind based on context
	context := p.getSymbolContext(node)
	
	switch context {
	case "struct", "service", "enum", "exception", "scope":
		// Type definitions
		if p.isDeclaration(node) {
			kind := protocol.DocumentHighlightKindWrite
			return &kind
		}
		kind := protocol.DocumentHighlightKindRead
		return &kind
	case "function":
		// Method/function definitions
		if p.isDeclaration(node) {
			kind := protocol.DocumentHighlightKindWrite
			return &kind
		}
		kind := protocol.DocumentHighlightKindRead
		return &kind
	case "field", "enum_field":
		// Field definitions and references
		if p.isDeclaration(node) {
			kind := protocol.DocumentHighlightKindWrite
			return &kind
		}
		kind := protocol.DocumentHighlightKindRead
		return &kind
	default:
		// Generic text highlight
		kind := protocol.DocumentHighlightKindText
		return &kind
	}
}

// isDeclaration determines if a node represents a declaration rather than a reference
func (p *DocumentHighlightProvider) isDeclaration(node *tree_sitter.Node) bool {
	parent := node.Parent()
	if parent == nil {
		return false
	}

	// Check various declaration contexts
	switch parent.Kind() {
	case "struct_definition", "service_definition", "enum_definition", 
		 "exception_definition", "scope_definition", "function_definition":
		// If the identifier is the first child after the keyword, it's likely a declaration
		childCount := parent.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := parent.Child(i)
			if child.Kind() == "identifier" && child == node {
				return i == 1 // Usually keyword is at 0, name at 1
			}
		}
	case "field", "enum_field":
		// Field declarations
		return true
	}

	return false
}

// Helper functions (reuse from ReferencesProvider)
func (p *DocumentHighlightProvider) findNodeAtOffset(node *tree_sitter.Node, offset uint) *tree_sitter.Node {
	return p.referencesProvider.findNodeAtOffset(node, offset)
}

func (p *DocumentHighlightProvider) getSymbolContext(node *tree_sitter.Node) string {
	return p.referencesProvider.getSymbolContext(node)
}
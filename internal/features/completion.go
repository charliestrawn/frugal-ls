package features

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// CompletionProvider handles code completion for Frugal files
type CompletionProvider struct{}

// NewCompletionProvider creates a new completion provider
func NewCompletionProvider() *CompletionProvider {
	return &CompletionProvider{}
}

// ProvideCompletion provides completion items for a given position
func (c *CompletionProvider) ProvideCompletion(doc *document.Document, position protocol.Position) ([]protocol.CompletionItem, error) {
	completions := make([]protocol.CompletionItem, 0)
	
	// Get the current line content
	lines := strings.Split(string(doc.Content), "\n")
	if int(position.Line) >= len(lines) {
		return completions, nil // Return empty slice, not nil
	}
	
	currentLine := lines[position.Line]
	
	// Handle character position beyond line length
	prefixEnd := int(position.Character)
	if prefixEnd > len(currentLine) {
		prefixEnd = len(currentLine)
	}
	linePrefix := currentLine[:prefixEnd]
	
	// Determine the context and provide appropriate completions
	context := c.determineCompletionContext(linePrefix)
	
	switch context {
	case CompletionContextTopLevel:
		completions = append(completions, c.getTopLevelCompletions()...)
	case CompletionContextService:
		completions = append(completions, c.getServiceCompletions()...)
	case CompletionContextScope:
		completions = append(completions, c.getScopeCompletions()...)
	case CompletionContextStruct:
		completions = append(completions, c.getStructCompletions()...)
	case CompletionContextEnum:
		completions = append(completions, c.getEnumCompletions()...)
	case CompletionContextType:
		completions = append(completions, c.getTypeCompletions()...)
	case CompletionContextGeneral:
		completions = append(completions, c.getKeywordCompletions()...)
		completions = append(completions, c.getTypeCompletions()...)
	}
	
	// Add symbol-based completions (variables, methods, etc.)
	symbolCompletions := c.getSymbolCompletions(doc, position)
	completions = append(completions, symbolCompletions...)
	
	return completions, nil
}

// CompletionContext represents different completion contexts
type CompletionContext int

const (
	CompletionContextTopLevel CompletionContext = iota
	CompletionContextService
	CompletionContextScope
	CompletionContextStruct
	CompletionContextEnum
	CompletionContextType
	CompletionContextGeneral
)

// determineCompletionContext analyzes the line prefix to determine completion context
func (c *CompletionProvider) determineCompletionContext(linePrefix string) CompletionContext {
	trimmed := strings.TrimSpace(linePrefix)
	
	// Check for specific contexts
	if strings.Contains(linePrefix, "service ") && strings.Contains(linePrefix, "{") {
		return CompletionContextService
	}
	if strings.Contains(linePrefix, "scope ") && strings.Contains(linePrefix, "{") {
		return CompletionContextScope
	}
	if strings.Contains(linePrefix, "struct ") && strings.Contains(linePrefix, "{") {
		return CompletionContextStruct
	}
	if strings.Contains(linePrefix, "enum ") && strings.Contains(linePrefix, "{") {
		return CompletionContextEnum
	}
	
	// Check for type context (after ':' or parameter lists)
	if strings.Contains(trimmed, ":") && !strings.HasSuffix(trimmed, ":") {
		return CompletionContextType
	}
	
	// Top-level context (outside any blocks)
	if !strings.Contains(linePrefix, "{") || 
	   (strings.Count(linePrefix, "{") <= strings.Count(linePrefix, "}")) {
		return CompletionContextTopLevel
	}
	
	return CompletionContextGeneral
}

// getTopLevelCompletions returns completions available at the top level
func (c *CompletionProvider) getTopLevelCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:      "include",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Include directive"}[0],
			InsertText: &[]string{"include \"$1\""}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "namespace",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Namespace declaration"}[0],
			InsertText: &[]string{"namespace $1 $2"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "const",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Constant declaration"}[0],
			InsertText: &[]string{"const $1 $2 = $3"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "typedef",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Type alias declaration"}[0],
			InsertText: &[]string{"typedef $1 $2"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "struct",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Struct declaration"}[0],
			InsertText: &[]string{"struct $1 {\n\t$2\n}"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "enum",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Enum declaration"}[0],
			InsertText: &[]string{"enum $1 {\n\t$2\n}"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "exception",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Exception declaration"}[0],
			InsertText: &[]string{"exception $1 {\n\t$2\n}"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "service",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Service declaration"}[0],
			InsertText: &[]string{"service $1 {\n\t$2\n}"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "scope",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Scope declaration (Frugal pub/sub)"}[0],
			InsertText: &[]string{"scope $1 {\n\t$2\n}"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
	}
}

// getServiceCompletions returns completions available inside service blocks
func (c *CompletionProvider) getServiceCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:      "oneway",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"One-way method (no response)"}[0],
			InsertText: &[]string{"oneway void $1($2)"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "throws",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Exception specification"}[0],
			InsertText: &[]string{"throws ($1: $2)"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "extends",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Service inheritance"}[0],
			InsertText: &[]string{"extends $1"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
	}
}

// getScopeCompletions returns completions available inside scope blocks
func (c *CompletionProvider) getScopeCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:      "prefix",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Topic prefix for pub/sub"}[0],
			InsertText: &[]string{"prefix \"$1\""}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
	}
}

// getStructCompletions returns completions available inside struct blocks
func (c *CompletionProvider) getStructCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:      "required",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Required field"}[0],
			InsertText: &[]string{"required"}[0],
		},
		{
			Label:      "optional",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Optional field"}[0],
			InsertText: &[]string{"optional"}[0],
		},
	}
}

// getEnumCompletions returns completions available inside enum blocks
func (c *CompletionProvider) getEnumCompletions() []protocol.CompletionItem {
	// Enum values don't have specific keywords, but we can suggest common patterns
	return []protocol.CompletionItem{}
}

// getTypeCompletions returns completions for Frugal types
func (c *CompletionProvider) getTypeCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		// Primitive types
		{Label: "bool", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "byte", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "i8", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "i16", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "i32", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "i64", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "double", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "string", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "binary", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		
		// Container types
		{
			Label:      "list",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"List container"}[0],
			InsertText: &[]string{"list<$1>"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "set",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Set container"}[0],
			InsertText: &[]string{"set<$1>"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		{
			Label:      "map",
			Kind:       &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0],
			Detail:     &[]string{"Map container"}[0],
			InsertText: &[]string{"map<$1, $2>"}[0],
			InsertTextFormat: &[]protocol.InsertTextFormat{protocol.InsertTextFormatSnippet}[0],
		},
		
		// Common return type
		{Label: "void", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
	}
}

// getKeywordCompletions returns general keyword completions
func (c *CompletionProvider) getKeywordCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{Label: "true", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
		{Label: "false", Kind: &[]protocol.CompletionItemKind{protocol.CompletionItemKindKeyword}[0]},
	}
}

// getSymbolCompletions returns completions based on symbols in the document
func (c *CompletionProvider) getSymbolCompletions(doc *document.Document, position protocol.Position) []protocol.CompletionItem {
	var completions []protocol.CompletionItem
	
	symbols := doc.GetSymbols()
	for _, symbol := range symbols {
		// Don't suggest the symbol at the current position
		if symbol.Line == uint(position.Line) {
			continue
		}
		
		var kind protocol.CompletionItemKind
		var detail string
		
		switch symbol.Type {
		case ast.NodeTypeService:
			kind = protocol.CompletionItemKindClass
			detail = "Service"
		case ast.NodeTypeScope:
			kind = protocol.CompletionItemKindClass  
			detail = "Scope (pub/sub)"
		case ast.NodeTypeStruct:
			kind = protocol.CompletionItemKindStruct
			detail = "Struct"
		case ast.NodeTypeEnum:
			kind = protocol.CompletionItemKindEnum
			detail = "Enum"
		case ast.NodeTypeConst:
			kind = protocol.CompletionItemKindConstant
			detail = "Constant"
		case ast.NodeTypeTypedef:
			kind = protocol.CompletionItemKindTypeParameter
			detail = "Type alias"
		case ast.NodeTypeException:
			kind = protocol.CompletionItemKindClass
			detail = "Exception"
		default:
			kind = protocol.CompletionItemKindVariable
			detail = "Symbol"
		}
		
		completions = append(completions, protocol.CompletionItem{
			Label:  symbol.Name,
			Kind:   &kind,
			Detail: &detail,
		})
	}
	
	return completions
}


// findNodeContainingByte finds the deepest node containing the given byte offset
func findNodeContainingByte(node *tree_sitter.Node, byteOffset uint) *tree_sitter.Node {
	if node == nil {
		return nil
	}
	
	// Check if byte offset is within this node's range
	if byteOffset < node.StartByte() || byteOffset > node.EndByte() {
		return nil
	}
	
	// Check children for a more specific match
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if childNode := findNodeContainingByte(child, byteOffset); childNode != nil {
			return childNode
		}
	}
	
	// No child contains the position, so this node is the deepest match
	return node
}
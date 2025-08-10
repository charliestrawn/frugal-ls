package features

import (
	"sort"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// SemanticTokensProvider provides semantic token highlighting for Frugal files
type SemanticTokensProvider struct{}

// NewSemanticTokensProvider creates a new semantic tokens provider
func NewSemanticTokensProvider() *SemanticTokensProvider {
	return &SemanticTokensProvider{}
}

// Token represents a semantic token with position and classification
type Token struct {
	Line      uint32
	Character uint32
	Length    uint32
	TokenType uint32
	Modifiers uint32
}

// Frugal token types based on LSP semantic token types
const (
	TokenTypeKeyword   uint32 = 0
	TokenTypeString    uint32 = 1
	TokenTypeNumber    uint32 = 2
	TokenTypeComment   uint32 = 3
	TokenTypeOperator  uint32 = 4
	TokenTypeType      uint32 = 5
	TokenTypeClass     uint32 = 6
	TokenTypeInterface uint32 = 7
	TokenTypeFunction  uint32 = 8
	TokenTypeVariable  uint32 = 9
	TokenTypeProperty  uint32 = 10
	TokenTypeNamespace uint32 = 11
	TokenTypeParameter uint32 = 12
	TokenTypeDecorator uint32 = 13
)

// Token modifiers
const (
	TokenModifierDeclaration  uint32 = 0
	TokenModifierDefinition   uint32 = 1
	TokenModifierReadonly     uint32 = 2
	TokenModifierStatic       uint32 = 3
	TokenModifierDeprecated   uint32 = 4
	TokenModifierAbstract     uint32 = 5
	TokenModifierAsync        uint32 = 6
	TokenModifierModification uint32 = 7
)

// GetLegend returns the semantic token legend (types and modifiers)
func (s *SemanticTokensProvider) GetLegend() protocol.SemanticTokensLegend {
	return protocol.SemanticTokensLegend{
		TokenTypes: []string{
			"keyword",   // 0
			"string",    // 1
			"number",    // 2
			"comment",   // 3
			"operator",  // 4
			"type",      // 5
			"class",     // 6
			"interface", // 7
			"function",  // 8
			"variable",  // 9
			"property",  // 10
			"namespace", // 11
			"parameter", // 12
			"decorator", // 13
		},
		TokenModifiers: []string{
			"declaration",  // 0
			"definition",   // 1
			"readonly",     // 2
			"static",       // 3
			"deprecated",   // 4
			"abstract",     // 5
			"async",        // 6
			"modification", // 7
		},
	}
}

// ProvideSemanticTokens provides semantic tokens for the entire document
func (s *SemanticTokensProvider) ProvideSemanticTokens(doc *document.Document) (*protocol.SemanticTokens, error) {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}

	var tokens []Token
	root := doc.ParseResult.GetRootNode()

	// Walk the AST and collect tokens
	s.walkNodeForTokens(root, doc.Content, &tokens)

	// Sort tokens by position
	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].Line != tokens[j].Line {
			return tokens[i].Line < tokens[j].Line
		}
		return tokens[i].Character < tokens[j].Character
	})

	// Convert to LSP format (relative encoding)
	data := s.encodeTokens(tokens)

	return &protocol.SemanticTokens{Data: data}, nil
}

// ProvideSemanticTokensRange provides semantic tokens for a specific range
func (s *SemanticTokensProvider) ProvideSemanticTokensRange(doc *document.Document, rang protocol.Range) (*protocol.SemanticTokens, error) {
	// For now, just provide tokens for the entire document
	// In a more optimized implementation, we'd filter by range
	return s.ProvideSemanticTokens(doc)
}

// walkNodeForTokens recursively walks the AST and collects semantic tokens
func (s *SemanticTokensProvider) walkNodeForTokens(node *tree_sitter.Node, source []byte, tokens *[]Token) {
	if node == nil {
		return
	}

	nodeType := node.Kind()
	startPos := node.StartPosition()
	nodeText := ast.GetText(node, source)

	// Classify different node types
	switch nodeType {
	case "include", "namespace", "service", "scope", "struct", "enum", "exception",
		"const", "typedef", "throws", "extends", "oneway", "required", "optional", "prefix":
		// Keywords
		*tokens = append(*tokens, Token{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
			Length:    uint32(len(nodeText)),
			TokenType: TokenTypeKeyword,
			Modifiers: 0,
		})

	case "string_literal":
		// String literals
		*tokens = append(*tokens, Token{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
			Length:    uint32(len(nodeText)),
			TokenType: TokenTypeString,
			Modifiers: 0,
		})

	case "number", "integer", "float":
		// Numeric literals
		*tokens = append(*tokens, Token{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
			Length:    uint32(len(nodeText)),
			TokenType: TokenTypeNumber,
			Modifiers: 0,
		})

	case "comment", "line_comment", "block_comment":
		// Comments
		*tokens = append(*tokens, Token{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
			Length:    uint32(len(nodeText)),
			TokenType: TokenTypeComment,
			Modifiers: 0,
		})

	case "identifier":
		// Context-dependent identifier classification
		tokenType, modifiers := s.classifyIdentifier(node, source)
		*tokens = append(*tokens, Token{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
			Length:    uint32(len(nodeText)),
			TokenType: tokenType,
			Modifiers: modifiers,
		})

	case "base_type", "type_identifier":
		// Type references
		*tokens = append(*tokens, Token{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
			Length:    uint32(len(nodeText)),
			TokenType: TokenTypeType,
			Modifiers: 0,
		})

	case "operator", "=", ":", ";", ",", "{", "}", "(", ")", "[", "]", "<", ">":
		// Operators and punctuation
		*tokens = append(*tokens, Token{
			Line:      uint32(startPos.Row),
			Character: uint32(startPos.Column),
			Length:    uint32(len(nodeText)),
			TokenType: TokenTypeOperator,
			Modifiers: 0,
		})
	}

	// Recursively process child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		s.walkNodeForTokens(child, source, tokens)
	}
}

// classifyIdentifier determines the token type and modifiers for an identifier based on context
func (s *SemanticTokensProvider) classifyIdentifier(node *tree_sitter.Node, source []byte) (uint32, uint32) {
	parent := node.Parent()
	if parent == nil {
		return TokenTypeVariable, 0
	}

	parentType := parent.Kind()

	switch parentType {
	case "service_definition":
		// Service name declaration
		return TokenTypeClass, 1 << TokenModifierDeclaration

	case "struct_definition", "exception_definition":
		// Struct/exception name declaration
		return TokenTypeClass, 1 << TokenModifierDeclaration

	case "enum_definition":
		// Enum name declaration
		return TokenTypeType, 1 << TokenModifierDeclaration

	case "scope_definition":
		// Scope name declaration
		return TokenTypeNamespace, 1 << TokenModifierDeclaration

	case "function_definition":
		// Method name declaration
		return TokenTypeFunction, 1 << TokenModifierDeclaration

	case "field":
		// Field name in struct
		return TokenTypeProperty, 1 << TokenModifierDeclaration

	case "const_definition":
		// Constant name
		return TokenTypeVariable, (1 << TokenModifierDeclaration) | (1 << TokenModifierReadonly)

	case "typedef_definition":
		// Type alias name
		return TokenTypeType, 1 << TokenModifierDeclaration

	case "parameter":
		// Method parameter
		return TokenTypeParameter, 0

	case "namespace_declaration":
		// Namespace value
		return TokenTypeNamespace, 0

	default:
		// Check if it's a type reference
		if s.isTypeReference(node, parent) {
			return TokenTypeType, 0
		}

		// Default to variable
		return TokenTypeVariable, 0
	}
}

// isTypeReference checks if an identifier is being used as a type reference
func (s *SemanticTokensProvider) isTypeReference(node *tree_sitter.Node, parent *tree_sitter.Node) bool {
	// Look for patterns where identifiers are used as types
	parentType := parent.Kind()

	switch parentType {
	case "type_reference", "return_type", "field_type":
		return true
	case "parameter":
		// Check if this identifier is the type part of a parameter
		childCount := parent.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := parent.Child(i)
			if child == node && i == 0 { // First identifier in parameter is usually the type
				return true
			}
		}
	}

	return false
}

// encodeTokens converts tokens to LSP relative encoding format
func (s *SemanticTokensProvider) encodeTokens(tokens []Token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}

	data := make([]uint32, 0, len(tokens)*5)

	var lastLine, lastChar uint32

	for _, token := range tokens {
		// Calculate deltas
		deltaLine := token.Line - lastLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = token.Character - lastChar
		} else {
			deltaChar = token.Character
		}

		// Add token data: [deltaLine, deltaChar, length, tokenType, tokenModifiers]
		data = append(data, deltaLine, deltaChar, token.Length, token.TokenType, token.Modifiers)

		lastLine = token.Line
		lastChar = token.Character
	}

	return data
}

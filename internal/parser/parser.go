package parser

import (
	"fmt"

	tree_sitter_frugal "github.com/charliestrawn/tree-sitter-frugal/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// TreeSitterParser wraps the tree-sitter parser for Frugal files
type TreeSitterParser struct {
	parser *tree_sitter.Parser
	lang   *tree_sitter.Language
}

// ParseResult contains the result of parsing a Frugal file
type ParseResult struct {
	Tree   *tree_sitter.Tree
	Errors []ParseError
}

// ParseError represents a parsing error
type ParseError struct {
	Message string
	Line    uint
	Column  uint
	Offset  uint
}

// NewParser creates a new TreeSitterParser instance
func NewParser() (*TreeSitterParser, error) {
	parser := tree_sitter.NewParser()
	if parser == nil {
		return nil, fmt.Errorf("failed to create tree-sitter parser")
	}

	langPtr := tree_sitter_frugal.Language()
	lang := tree_sitter.NewLanguage(langPtr)
	if lang == nil {
		return nil, fmt.Errorf("failed to get Frugal language")
	}

	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	return &TreeSitterParser{
		parser: parser,
		lang:   lang,
	}, nil
}

// Parse parses the given Frugal source code and returns the syntax tree
func (p *TreeSitterParser) Parse(source []byte) (*ParseResult, error) {
	tree := p.parser.Parse(source, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse source code")
	}

	result := &ParseResult{
		Tree:   tree,
		Errors: []ParseError{},
	}

	// Check for syntax errors
	if tree.RootNode().HasError() {
		result.Errors = p.collectErrors(tree.RootNode(), source)
	}

	return result, nil
}

// collectErrors walks the syntax tree and collects parsing errors
func (p *TreeSitterParser) collectErrors(node *tree_sitter.Node, source []byte) []ParseError {
	var errors []ParseError

	if node.Kind() == "ERROR" {
		point := node.StartPosition()
		errors = append(errors, ParseError{
			Message: "Syntax error",
			Line:    point.Row,
			Column:  point.Column,
			Offset:  node.StartByte(),
		})
	}

	if node.IsMissing() {
		point := node.StartPosition()
		errors = append(errors, ParseError{
			Message: fmt.Sprintf("Missing %s", node.Kind()),
			Line:    point.Row,
			Column:  point.Column,
			Offset:  node.StartByte(),
		})
	}

	// Recursively check child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		childErrors := p.collectErrors(child, source)
		errors = append(errors, childErrors...)
	}

	return errors
}

// GetRootNode returns the root node of the last parsed tree
func (r *ParseResult) GetRootNode() *tree_sitter.Node {
	if r.Tree == nil {
		return nil
	}
	return r.Tree.RootNode()
}

// HasErrors returns true if there are parsing errors
func (r *ParseResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// Close releases resources held by the parser
func (p *TreeSitterParser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// Close releases resources held by the parse result
func (r *ParseResult) Close() {
	if r.Tree != nil {
		r.Tree.Close()
	}
}

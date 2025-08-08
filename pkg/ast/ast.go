package ast

import (
	"fmt"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// NodeType represents different types of nodes in the Frugal AST
type NodeType string

const (
	NodeTypeService   NodeType = "service"
	NodeTypeScope     NodeType = "scope"
	NodeTypeStruct    NodeType = "struct"
	NodeTypeEnum      NodeType = "enum"
	NodeTypeConst     NodeType = "const"
	NodeTypeTypedef   NodeType = "typedef"
	NodeTypeException NodeType = "exception"
	NodeTypeInclude   NodeType = "include"
	NodeTypeNamespace NodeType = "namespace"
	NodeTypeMethod    NodeType = "method"
	NodeTypeField     NodeType = "field"
	NodeTypeEvent     NodeType = "event"
	NodeTypeEnumValue NodeType = "enum_value"
)

// Symbol represents a symbol in the Frugal AST
type Symbol struct {
	Name     string
	Type     NodeType
	Line     int
	Column   int
	StartPos uint
	EndPos   uint
	Node     *tree_sitter.Node
}

// Document represents a parsed Frugal document
type Document struct {
	Source  []byte
	Symbols []Symbol
}

// GetText returns the text content of a node
func GetText(node *tree_sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	start := node.StartByte()
	end := node.EndByte()
	if start >= uint(len(source)) || end > uint(len(source)) || start > end {
		return ""
	}
	return string(source[start:end])
}

// FindNodeByType recursively searches for a node of the specified type
func FindNodeByType(node *tree_sitter.Node, nodeType string) *tree_sitter.Node {
	if node == nil {
		return nil
	}

	if node.Kind() == nodeType {
		return node
	}

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if result := FindNodeByType(child, nodeType); result != nil {
			return result
		}
	}

	return nil
}

// ExtractSymbols extracts symbols from the AST for LSP features
func ExtractSymbols(root *tree_sitter.Node, source []byte) []Symbol {
	var symbols []Symbol
	extractSymbolsRecursive(root, source, &symbols)
	return symbols
}

func extractSymbolsRecursive(node *tree_sitter.Node, source []byte, symbols *[]Symbol) {
	if node == nil {
		return
	}

	nodeType := node.Kind()
	
	// Check if this node represents a symbol we care about
	switch nodeType {
	case "service_definition":
		if symbol := extractServiceSymbol(node, source); symbol != nil {
			*symbols = append(*symbols, *symbol)
		}
	case "scope_definition":
		if symbol := extractScopeSymbol(node, source); symbol != nil {
			*symbols = append(*symbols, *symbol)
		}
	case "struct_definition":
		if symbol := extractStructSymbol(node, source); symbol != nil {
			*symbols = append(*symbols, *symbol)
		}
	case "enum_definition":
		if symbol := extractEnumSymbol(node, source); symbol != nil {
			*symbols = append(*symbols, *symbol)
		}
	case "const_definition":
		if symbol := extractConstSymbol(node, source); symbol != nil {
			*symbols = append(*symbols, *symbol)
		}
	case "typedef_definition":
		if symbol := extractTypedefSymbol(node, source); symbol != nil {
			*symbols = append(*symbols, *symbol)
		}
	case "exception_definition":
		if symbol := extractExceptionSymbol(node, source); symbol != nil {
			*symbols = append(*symbols, *symbol)
		}
	}

	// Recursively process child nodes
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		extractSymbolsRecursive(child, source, symbols)
	}
}

func extractServiceSymbol(node *tree_sitter.Node, source []byte) *Symbol {
	nameNode := FindNodeByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	name := GetText(nameNode, source)
	point := node.StartPosition()

	return &Symbol{
		Name:     name,
		Type:     NodeTypeService,
		Line:     int(point.Row),
		Column:   int(point.Column),
		StartPos: node.StartByte(),
		EndPos:   node.EndByte(),
		Node:     node,
	}
}

func extractScopeSymbol(node *tree_sitter.Node, source []byte) *Symbol {
	nameNode := FindNodeByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	name := GetText(nameNode, source)
	point := node.StartPosition()

	return &Symbol{
		Name:     name,
		Type:     NodeTypeScope,
		Line:     int(point.Row),
		Column:   int(point.Column),
		StartPos: node.StartByte(),
		EndPos:   node.EndByte(),
		Node:     node,
	}
}

func extractStructSymbol(node *tree_sitter.Node, source []byte) *Symbol {
	nameNode := FindNodeByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	name := GetText(nameNode, source)
	point := node.StartPosition()

	return &Symbol{
		Name:     name,
		Type:     NodeTypeStruct,
		Line:     int(point.Row),
		Column:   int(point.Column),
		StartPos: node.StartByte(),
		EndPos:   node.EndByte(),
		Node:     node,
	}
}

func extractEnumSymbol(node *tree_sitter.Node, source []byte) *Symbol {
	nameNode := FindNodeByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	name := GetText(nameNode, source)
	point := node.StartPosition()

	return &Symbol{
		Name:     name,
		Type:     NodeTypeEnum,
		Line:     int(point.Row),
		Column:   int(point.Column),
		StartPos: node.StartByte(),
		EndPos:   node.EndByte(),
		Node:     node,
	}
}

func extractConstSymbol(node *tree_sitter.Node, source []byte) *Symbol {
	nameNode := FindNodeByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	name := GetText(nameNode, source)
	point := node.StartPosition()

	return &Symbol{
		Name:     name,
		Type:     NodeTypeConst,
		Line:     int(point.Row),
		Column:   int(point.Column),
		StartPos: node.StartByte(),
		EndPos:   node.EndByte(),
		Node:     node,
	}
}

func extractTypedefSymbol(node *tree_sitter.Node, source []byte) *Symbol {
	nameNode := FindNodeByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	name := GetText(nameNode, source)
	point := node.StartPosition()

	return &Symbol{
		Name:     name,
		Type:     NodeTypeTypedef,
		Line:     int(point.Row),
		Column:   int(point.Column),
		StartPos: node.StartByte(),
		EndPos:   node.EndByte(),
		Node:     node,
	}
}

func extractExceptionSymbol(node *tree_sitter.Node, source []byte) *Symbol {
	nameNode := FindNodeByType(node, "identifier")
	if nameNode == nil {
		return nil
	}

	name := GetText(nameNode, source)
	point := node.StartPosition()

	return &Symbol{
		Name:     name,
		Type:     NodeTypeException,
		Line:     int(point.Row),
		Column:   int(point.Column),
		StartPos: node.StartByte(),
		EndPos:   node.EndByte(),
		Node:     node,
	}
}

// PrintTree prints the AST tree for debugging purposes
func PrintTree(node *tree_sitter.Node, source []byte, indent int) {
	if node == nil {
		return
	}

	indentStr := strings.Repeat("  ", indent)
	nodeText := GetText(node, source)
	if len(nodeText) > 50 {
		nodeText = nodeText[:50] + "..."
	}
	nodeText = strings.ReplaceAll(nodeText, "\n", "\\n")
	
	fmt.Printf("%s%s: %q\n", indentStr, node.Kind(), nodeText)

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		PrintTree(child, source, indent+1)
	}
}
package features

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// DocumentSymbolProvider handles document symbol outline for Frugal files
type DocumentSymbolProvider struct{}

// NewDocumentSymbolProvider creates a new document symbol provider
func NewDocumentSymbolProvider() *DocumentSymbolProvider {
	return &DocumentSymbolProvider{}
}

// ProvideDocumentSymbols provides a hierarchical outline of symbols in the document
func (d *DocumentSymbolProvider) ProvideDocumentSymbols(doc *document.Document) ([]protocol.DocumentSymbol, error) {
	symbols := doc.GetSymbols()
	if len(symbols) == 0 {
		return nil, nil
	}

	var documentSymbols []protocol.DocumentSymbol

	for _, symbol := range symbols {
		docSymbol := d.convertToDocumentSymbol(symbol, doc)
		documentSymbols = append(documentSymbols, docSymbol)
	}

	return documentSymbols, nil
}

// ProvideWorkspaceSymbols provides symbols across all documents in the workspace
func (d *DocumentSymbolProvider) ProvideWorkspaceSymbols(query string, documents map[string]*document.Document) ([]protocol.SymbolInformation, error) {
	var workspaceSymbols []protocol.SymbolInformation

	for uri, doc := range documents {
		if !doc.IsValidFrugalFile() {
			continue
		}

		symbols := doc.GetSymbols()
		for _, symbol := range symbols {
			// Filter symbols based on query if provided
			if query != "" && !containsIgnoreCase(symbol.Name, query) {
				continue
			}

			workspaceSymbol := protocol.SymbolInformation{
				Name: symbol.Name,
				Kind: d.getSymbolKind(symbol.Type),
				Location: protocol.Location{
					URI: uri,
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
				},
			}

			workspaceSymbols = append(workspaceSymbols, workspaceSymbol)
		}
	}

	return workspaceSymbols, nil
}

// convertToDocumentSymbol converts an AST symbol to LSP DocumentSymbol
func (d *DocumentSymbolProvider) convertToDocumentSymbol(symbol ast.Symbol, doc *document.Document) protocol.DocumentSymbol {
	symbolRange := protocol.Range{
		Start: protocol.Position{
			Line:      uint32(symbol.Line),
			Character: uint32(symbol.Column),
		},
		End: protocol.Position{
			Line:      uint32(symbol.Line),
			Character: uint32(symbol.Column) + uint32(len(symbol.Name)),
		},
	}

	// For complex symbols, try to get the full range including the body
	selectionRange := symbolRange
	if symbol.Node != nil {
		fullRange := protocol.Range{
			Start: protocol.Position{
				Line:      uint32(symbol.Node.StartPosition().Row),
				Character: uint32(symbol.Node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(symbol.Node.EndPosition().Row),
				Character: uint32(symbol.Node.EndPosition().Column),
			},
		}
		symbolRange = fullRange
	}

	docSymbol := protocol.DocumentSymbol{
		Name:           symbol.Name,
		Kind:           d.getSymbolKind(symbol.Type),
		Range:          symbolRange,
		SelectionRange: selectionRange,
	}

	// Add detail information
	detail := d.getSymbolDetail(symbol, doc)
	if detail != "" {
		docSymbol.Detail = &detail
	}

	// Add children for structured symbols
	children := d.getSymbolChildren(symbol, doc)
	if len(children) > 0 {
		docSymbol.Children = children
	}

	return docSymbol
}

// getSymbolKind converts AST node type to LSP symbol kind
func (d *DocumentSymbolProvider) getSymbolKind(nodeType ast.NodeType) protocol.SymbolKind {
	switch nodeType {
	case ast.NodeTypeService:
		return protocol.SymbolKindClass
	case ast.NodeTypeScope:
		return protocol.SymbolKindClass
	case ast.NodeTypeStruct:
		return protocol.SymbolKindStruct
	case ast.NodeTypeEnum:
		return protocol.SymbolKindEnum
	case ast.NodeTypeConst:
		return protocol.SymbolKindConstant
	case ast.NodeTypeTypedef:
		return protocol.SymbolKindTypeParameter
	case ast.NodeTypeException:
		return protocol.SymbolKindClass
	case ast.NodeTypeInclude:
		return protocol.SymbolKindModule
	case ast.NodeTypeNamespace:
		return protocol.SymbolKindNamespace
	default:
		return protocol.SymbolKindVariable
	}
}

// getSymbolDetail returns additional detail information for symbols
func (d *DocumentSymbolProvider) getSymbolDetail(symbol ast.Symbol, doc *document.Document) string {
	switch symbol.Type {
	case ast.NodeTypeService:
		return "Service"
	case ast.NodeTypeScope:
		return "Scope (pub/sub)"
	case ast.NodeTypeStruct:
		return "Struct"
	case ast.NodeTypeEnum:
		return "Enum"
	case ast.NodeTypeConst:
		return "Constant"
	case ast.NodeTypeTypedef:
		return "Type Alias"
	case ast.NodeTypeException:
		return "Exception"
	case ast.NodeTypeInclude:
		return "Include"
	case ast.NodeTypeNamespace:
		return "Namespace"
	default:
		return ""
	}
}

// getSymbolChildren extracts child symbols for structured types
func (d *DocumentSymbolProvider) getSymbolChildren(symbol ast.Symbol, doc *document.Document) []protocol.DocumentSymbol {
	var children []protocol.DocumentSymbol

	if symbol.Node == nil {
		return children
	}

	switch symbol.Type {
	case ast.NodeTypeService:
		children = d.extractServiceMethods(symbol.Node, doc)
	case ast.NodeTypeScope:
		children = d.extractScopeEvents(symbol.Node, doc)
	case ast.NodeTypeStruct, ast.NodeTypeException:
		children = d.extractStructFields(symbol.Node, doc)
	case ast.NodeTypeEnum:
		children = d.extractEnumValues(symbol.Node, doc)
	}

	return children
}

// extractServiceMethods extracts method symbols from a service
func (d *DocumentSymbolProvider) extractServiceMethods(serviceNode *tree_sitter.Node, doc *document.Document) []protocol.DocumentSymbol {
	var methods []protocol.DocumentSymbol

	// Find service body
	serviceBody := ast.FindNodeByType(serviceNode, "service_body")
	if serviceBody == nil {
		return methods
	}

	// Extract methods from service body
	childCount := serviceBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := serviceBody.Child(i)
		if d.isMethodNode(child) {
			method := d.extractMethodSymbol(child, doc)
			if method != nil {
				methods = append(methods, *method)
			}
		}
	}

	return methods
}

// extractScopeEvents extracts event symbols from a scope
func (d *DocumentSymbolProvider) extractScopeEvents(scopeNode *tree_sitter.Node, doc *document.Document) []protocol.DocumentSymbol {
	var events []protocol.DocumentSymbol

	// Find scope body
	scopeBody := ast.FindNodeByType(scopeNode, "scope_body")
	if scopeBody == nil {
		return events
	}

	// Extract events from scope body
	childCount := scopeBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := scopeBody.Child(i)
		if d.isEventNode(child) {
			event := d.extractEventSymbol(child, doc)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}

// extractStructFields extracts field symbols from a struct
func (d *DocumentSymbolProvider) extractStructFields(structNode *tree_sitter.Node, doc *document.Document) []protocol.DocumentSymbol {
	var fields []protocol.DocumentSymbol

	// Find struct body
	structBody := ast.FindNodeByType(structNode, "struct_body")
	if structBody == nil {
		return fields
	}

	// Extract fields from struct body
	childCount := structBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := structBody.Child(i)
		if d.isFieldNode(child) {
			field := d.extractFieldSymbol(child, doc)
			if field != nil {
				fields = append(fields, *field)
			}
		}
	}

	return fields
}

// extractEnumValues extracts value symbols from an enum
func (d *DocumentSymbolProvider) extractEnumValues(enumNode *tree_sitter.Node, doc *document.Document) []protocol.DocumentSymbol {
	var values []protocol.DocumentSymbol

	// Find enum body
	enumBody := ast.FindNodeByType(enumNode, "enum_body")
	if enumBody == nil {
		return values
	}

	// Extract values from enum body
	childCount := enumBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := enumBody.Child(i)
		if d.isEnumValueNode(child) {
			value := d.extractEnumValueSymbol(child, doc)
			if value != nil {
				values = append(values, *value)
			}
		}
	}

	return values
}

// Helper methods to check node types
func (d *DocumentSymbolProvider) isMethodNode(node *tree_sitter.Node) bool {
	kind := node.Kind()
	return kind == "method" || kind == "function" ||
		(kind == nodeTypeIdentifier && d.looksLikeMethod(node))
}

func (d *DocumentSymbolProvider) isEventNode(node *tree_sitter.Node) bool {
	kind := node.Kind()
	return kind == "event" || kind == nodeTypeIdentifier
}

func (d *DocumentSymbolProvider) isFieldNode(node *tree_sitter.Node) bool {
	kind := node.Kind()
	return kind == "field" || kind == "struct_field" ||
		(kind == "identifier" && d.looksLikeField(node))
}

func (d *DocumentSymbolProvider) isEnumValueNode(node *tree_sitter.Node) bool {
	kind := node.Kind()
	return kind == "enum_value" || kind == "identifier"
}

// Helper methods to extract specific symbol types
func (d *DocumentSymbolProvider) extractMethodSymbol(node *tree_sitter.Node, doc *document.Document) *protocol.DocumentSymbol {
	// Get method name and signature
	methodText := ast.GetText(node, doc.Content)
	if methodText == "" {
		return nil
	}

	// Try to extract method name
	var methodName string
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		methodName = ast.GetText(nameNode, doc.Content)
	}

	if methodName == "" {
		// Fallback: try to parse method name from text
		parts := strings.Fields(strings.TrimSpace(methodText))
		if len(parts) > 0 {
			methodName = parts[0]
		}
	}

	if methodName == "" {
		return nil
	}

	return &protocol.DocumentSymbol{
		Name: methodName,
		Kind: protocol.SymbolKindMethod,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.EndPosition().Row),
				Character: uint32(node.EndPosition().Column),
			},
		},
		SelectionRange: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column) + uint32(len(methodName)),
			},
		},
		Detail: &[]string{"Method"}[0],
	}
}

func (d *DocumentSymbolProvider) extractEventSymbol(node *tree_sitter.Node, doc *document.Document) *protocol.DocumentSymbol {
	eventText := ast.GetText(node, doc.Content)
	if eventText == "" {
		return nil
	}

	// Parse event name from "EventName: Type" format
	parts := strings.Split(strings.TrimSpace(eventText), ":")
	if len(parts) < 1 {
		return nil
	}

	eventName := strings.TrimSpace(parts[0])
	if eventName == "" {
		return nil
	}

	return &protocol.DocumentSymbol{
		Name: eventName,
		Kind: protocol.SymbolKindEvent,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.EndPosition().Row),
				Character: uint32(node.EndPosition().Column),
			},
		},
		SelectionRange: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column) + uint32(len(eventName)),
			},
		},
		Detail: &[]string{"Event"}[0],
	}
}

func (d *DocumentSymbolProvider) extractFieldSymbol(node *tree_sitter.Node, doc *document.Document) *protocol.DocumentSymbol {
	fieldText := ast.GetText(node, doc.Content)
	if fieldText == "" {
		return nil
	}

	// Try to extract field name from various formats
	// "1: required string name"
	// "2: optional i32 id"
	var fieldName string
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		fieldName = ast.GetText(nameNode, doc.Content)
	}

	if fieldName == "" {
		// Fallback: try to parse field name from text
		parts := strings.Fields(strings.TrimSpace(fieldText))
		if len(parts) > 0 {
			fieldName = parts[len(parts)-1] // Usually the last part
		}
	}

	if fieldName == "" {
		return nil
	}

	return &protocol.DocumentSymbol{
		Name: fieldName,
		Kind: protocol.SymbolKindField,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.EndPosition().Row),
				Character: uint32(node.EndPosition().Column),
			},
		},
		SelectionRange: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column) + uint32(len(fieldName)),
			},
		},
		Detail: &[]string{"Field"}[0],
	}
}

func (d *DocumentSymbolProvider) extractEnumValueSymbol(node *tree_sitter.Node, doc *document.Document) *protocol.DocumentSymbol {
	valueText := ast.GetText(node, doc.Content)
	if valueText == "" {
		return nil
	}

	// Parse enum value from "VALUE = 1" format
	parts := strings.Split(strings.TrimSpace(valueText), "=")
	valueName := strings.TrimSpace(parts[0])
	if valueName == "" {
		return nil
	}

	return &protocol.DocumentSymbol{
		Name: valueName,
		Kind: protocol.SymbolKindEnumMember,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.EndPosition().Row),
				Character: uint32(node.EndPosition().Column),
			},
		},
		SelectionRange: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column),
			},
			End: protocol.Position{
				Line:      uint32(node.StartPosition().Row),
				Character: uint32(node.StartPosition().Column) + uint32(len(valueName)),
			},
		},
		Detail: &[]string{"Enum Value"}[0],
	}
}

// Helper methods for heuristics
func (d *DocumentSymbolProvider) looksLikeMethod(node *tree_sitter.Node) bool {
	// Heuristic: check if the node text contains parentheses (method signature)
	return strings.Contains(ast.GetText(node, []byte{}), "(")
}

func (d *DocumentSymbolProvider) looksLikeField(node *tree_sitter.Node) bool {
	// Heuristic: check if the node is part of a field definition
	return strings.Contains(ast.GetText(node, []byte{}), ":")
}

// containsIgnoreCase checks if s contains substr (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

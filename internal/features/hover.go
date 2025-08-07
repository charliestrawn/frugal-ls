package features

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// HoverProvider handles hover information for Frugal symbols
type HoverProvider struct{}

// NewHoverProvider creates a new hover provider
func NewHoverProvider() *HoverProvider {
	return &HoverProvider{}
}

// ProvideHover provides hover information for a given position
func (h *HoverProvider) ProvideHover(doc *document.Document, position protocol.Position) (*protocol.Hover, error) {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return nil, nil
	}
	
	// Find the node at the position
	node := FindNodeAtPosition(doc.ParseResult.GetRootNode(), doc.Content, uint(position.Line), uint(position.Character))
	if node == nil {
		return nil, nil
	}
	
	// Get hover information based on the node
	hoverInfo := h.getHoverInfo(node, doc)
	if hoverInfo == nil {
		return nil, nil
	}
	
	// Create the range for the hover
	hoverRange := protocol.Range{
		Start: protocol.Position{
			Line:      uint32(node.StartPosition().Row),
			Character: uint32(node.StartPosition().Column),
		},
		End: protocol.Position{
			Line:      uint32(node.EndPosition().Row), 
			Character: uint32(node.EndPosition().Column),
		},
	}
	
	return &protocol.Hover{
		Contents: *hoverInfo,
		Range:    &hoverRange,
	}, nil
}

// getHoverInfo extracts hover information for a given node
func (h *HoverProvider) getHoverInfo(node *tree_sitter.Node, doc *document.Document) *protocol.MarkupContent {
	nodeType := node.Kind()
	nodeText := ast.GetText(node, doc.Content)
	
	var content strings.Builder
	var found bool
	
	// Handle different node types
	switch nodeType {
	case "identifier":
		// Find the symbol this identifier refers to
		if symbolInfo := h.findSymbolByName(nodeText, doc); symbolInfo != nil {
			content.WriteString(h.formatSymbolInfo(symbolInfo, doc))
			found = true
		}
		
	case "service_definition":
		if serviceName := h.extractServiceName(node, doc.Content); serviceName != "" {
			content.WriteString(fmt.Sprintf("**Service**: `%s`\n\n", serviceName))
			content.WriteString("Defines a service with RPC methods.\n\n")
			content.WriteString(h.getServiceMethods(node, doc.Content))
			found = true
		}
		
	case "scope_definition":
		if scopeName := h.extractScopeName(node, doc.Content); scopeName != "" {
			content.WriteString(fmt.Sprintf("**Scope**: `%s`\n\n", scopeName))
			content.WriteString("Defines a pub/sub scope for event messaging.\n\n")
			content.WriteString(h.getScopeEvents(node, doc.Content))
			found = true
		}
		
	case "struct_definition":
		if structName := h.extractStructName(node, doc.Content); structName != "" {
			content.WriteString(fmt.Sprintf("**Struct**: `%s`\n\n", structName))
			content.WriteString("Data structure definition.\n\n")
			content.WriteString(h.getStructFields(node, doc.Content))
			found = true
		}
		
	case "enum_definition":
		if enumName := h.extractEnumName(node, doc.Content); enumName != "" {
			content.WriteString(fmt.Sprintf("**Enum**: `%s`\n\n", enumName))
			content.WriteString("Enumeration type definition.\n\n")
			content.WriteString(h.getEnumValues(node, doc.Content))
			found = true
		}
		
	case "const_definition":
		if constInfo := h.extractConstInfo(node, doc.Content); constInfo != "" {
			content.WriteString(fmt.Sprintf("**Constant**: %s\n\n", constInfo))
			found = true
		}
		
	case "typedef_definition":
		if typedefInfo := h.extractTypedefInfo(node, doc.Content); typedefInfo != "" {
			content.WriteString(fmt.Sprintf("**Type Alias**: %s\n\n", typedefInfo))
			found = true
		}
		
	// Handle type information
	case "field_type":
		typeInfo := h.getTypeInfo(nodeText)
		if typeInfo != "" {
			content.WriteString(fmt.Sprintf("**Type**: `%s`\n\n%s", nodeText, typeInfo))
			found = true
		}
	}
	
	// Add syntax information for keywords
	if !found {
		if keywordInfo := h.getKeywordInfo(nodeText); keywordInfo != "" {
			content.WriteString(keywordInfo)
			found = true
		}
	}
	
	if !found {
		return nil
	}
	
	return &protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: content.String(),
	}
}

// findSymbolByName finds a symbol by name in the document
func (h *HoverProvider) findSymbolByName(name string, doc *document.Document) *ast.Symbol {
	symbols := doc.GetSymbols()
	for _, symbol := range symbols {
		if symbol.Name == name {
			return &symbol
		}
	}
	return nil
}

// formatSymbolInfo formats hover information for a symbol
func (h *HoverProvider) formatSymbolInfo(symbol *ast.Symbol, doc *document.Document) string {
	var content strings.Builder
	
	// Symbol type and name
	switch symbol.Type {
	case ast.NodeTypeService:
		content.WriteString(fmt.Sprintf("**Service**: `%s`\n\n", symbol.Name))
		content.WriteString("Service definition with RPC methods.")
	case ast.NodeTypeScope:
		content.WriteString(fmt.Sprintf("**Scope**: `%s`\n\n", symbol.Name))
		content.WriteString("Pub/sub scope for event messaging.")
	case ast.NodeTypeStruct:
		content.WriteString(fmt.Sprintf("**Struct**: `%s`\n\n", symbol.Name))
		content.WriteString("Data structure definition.")
	case ast.NodeTypeEnum:
		content.WriteString(fmt.Sprintf("**Enum**: `%s`\n\n", symbol.Name))
		content.WriteString("Enumeration type.")
	case ast.NodeTypeConst:
		content.WriteString(fmt.Sprintf("**Constant**: `%s`\n\n", symbol.Name))
		content.WriteString("Constant value.")
	case ast.NodeTypeTypedef:
		content.WriteString(fmt.Sprintf("**Type Alias**: `%s`\n\n", symbol.Name))
		content.WriteString("Type alias definition.")
	case ast.NodeTypeException:
		content.WriteString(fmt.Sprintf("**Exception**: `%s`\n\n", symbol.Name))
		content.WriteString("Exception type definition.")
	}
	
	// Add location information
	content.WriteString(fmt.Sprintf("\n\n*Defined at line %d, column %d*", symbol.Line+1, symbol.Column+1))
	
	return content.String()
}

// extractServiceName extracts the service name from a service definition node
func (h *HoverProvider) extractServiceName(node *tree_sitter.Node, source []byte) string {
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		return ast.GetText(nameNode, source)
	}
	return ""
}

// extractScopeName extracts the scope name from a scope definition node  
func (h *HoverProvider) extractScopeName(node *tree_sitter.Node, source []byte) string {
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		return ast.GetText(nameNode, source)
	}
	return ""
}

// extractStructName extracts the struct name from a struct definition node
func (h *HoverProvider) extractStructName(node *tree_sitter.Node, source []byte) string {
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		return ast.GetText(nameNode, source)
	}
	return ""
}

// extractEnumName extracts the enum name from an enum definition node
func (h *HoverProvider) extractEnumName(node *tree_sitter.Node, source []byte) string {
	nameNode := ast.FindNodeByType(node, "identifier")
	if nameNode != nil {
		return ast.GetText(nameNode, source)
	}
	return ""
}

// extractConstInfo extracts constant information
func (h *HoverProvider) extractConstInfo(node *tree_sitter.Node, source []byte) string {
	// Find type, name, and value
	var constType, constName, constValue string
	
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "field_type":
			constType = ast.GetText(child, source)
		case "identifier":
			if constName == "" { // First identifier is the name
				constName = ast.GetText(child, source)
			}
		case "const_value":
			constValue = ast.GetText(child, source)
		}
	}
	
	return fmt.Sprintf("`%s %s = %s`", constType, constName, constValue)
}

// extractTypedefInfo extracts typedef information
func (h *HoverProvider) extractTypedefInfo(node *tree_sitter.Node, source []byte) string {
	var baseType, aliasName string
	
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "field_type":
			baseType = ast.GetText(child, source)
		case "identifier":
			aliasName = ast.GetText(child, source)
		}
	}
	
	return fmt.Sprintf("`%s` → `%s`", aliasName, baseType)
}

// getServiceMethods gets method information for a service
func (h *HoverProvider) getServiceMethods(node *tree_sitter.Node, source []byte) string {
	var content strings.Builder
	content.WriteString("**Methods:**\n")
	
	// Find service body and extract methods
	serviceBody := ast.FindNodeByType(node, "service_body")
	if serviceBody == nil {
		return "No methods found."
	}
	
	methodCount := 0
	childCount := serviceBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := serviceBody.Child(i)
		if child.Kind() == "method" || strings.Contains(child.Kind(), "method") {
			methodText := ast.GetText(child, source)
			content.WriteString(fmt.Sprintf("- `%s`\n", strings.TrimSpace(methodText)))
			methodCount++
		}
	}
	
	if methodCount == 0 {
		return "No methods defined."
	}
	
	return content.String()
}

// getScopeEvents gets event information for a scope
func (h *HoverProvider) getScopeEvents(node *tree_sitter.Node, source []byte) string {
	var content strings.Builder
	content.WriteString("**Events:**\n")
	
	// Find scope body and extract events
	scopeBody := ast.FindNodeByType(node, "scope_body")
	if scopeBody == nil {
		return "No events found."
	}
	
	eventCount := 0
	childCount := scopeBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := scopeBody.Child(i)
		eventText := ast.GetText(child, source)
		if strings.TrimSpace(eventText) != "" && !strings.HasPrefix(strings.TrimSpace(eventText), "//") {
			content.WriteString(fmt.Sprintf("- `%s`\n", strings.TrimSpace(eventText)))
			eventCount++
		}
	}
	
	if eventCount == 0 {
		return "No events defined."
	}
	
	return content.String()
}

// getStructFields gets field information for a struct
func (h *HoverProvider) getStructFields(node *tree_sitter.Node, source []byte) string {
	var content strings.Builder
	content.WriteString("**Fields:**\n")
	
	// Find struct body and extract fields
	structBody := ast.FindNodeByType(node, "struct_body")
	if structBody == nil {
		return "No fields found."
	}
	
	fieldCount := 0
	childCount := structBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := structBody.Child(i)
		if child.Kind() == "field" || strings.Contains(child.Kind(), "field") {
			fieldText := ast.GetText(child, source)
			content.WriteString(fmt.Sprintf("- `%s`\n", strings.TrimSpace(fieldText)))
			fieldCount++
		}
	}
	
	if fieldCount == 0 {
		return "No fields defined."
	}
	
	return content.String()
}

// getEnumValues gets value information for an enum
func (h *HoverProvider) getEnumValues(node *tree_sitter.Node, source []byte) string {
	var content strings.Builder
	content.WriteString("**Values:**\n")
	
	// Find enum body and extract values
	enumBody := ast.FindNodeByType(node, "enum_body")
	if enumBody == nil {
		return "No values found."
	}
	
	valueCount := 0
	childCount := enumBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := enumBody.Child(i)
		valueText := ast.GetText(child, source)
		if strings.TrimSpace(valueText) != "" && !strings.HasPrefix(strings.TrimSpace(valueText), "//") {
			content.WriteString(fmt.Sprintf("- `%s`\n", strings.TrimSpace(valueText)))
			valueCount++
		}
	}
	
	if valueCount == 0 {
		return "No values defined."
	}
	
	return content.String()
}

// getTypeInfo returns information about Frugal types
func (h *HoverProvider) getTypeInfo(typeName string) string {
	typeMap := map[string]string{
		"bool":    "Boolean value (true/false)",
		"byte":    "8-bit signed integer (-128 to 127)",
		"i8":      "8-bit signed integer (-128 to 127)",
		"i16":     "16-bit signed integer (-32,768 to 32,767)",
		"i32":     "32-bit signed integer (-2^31 to 2^31-1)",
		"i64":     "64-bit signed integer (-2^63 to 2^63-1)",
		"double":  "64-bit floating point number",
		"string":  "UTF-8 encoded string",
		"binary":  "Byte array/binary data",
		"void":    "No return value",
		"list":    "Ordered collection of elements",
		"set":     "Unordered collection of unique elements", 
		"map":     "Key-value mapping",
	}
	
	if info, exists := typeMap[typeName]; exists {
		return info
	}
	return ""
}

// getKeywordInfo returns information about Frugal keywords
func (h *HoverProvider) getKeywordInfo(keyword string) string {
	keywordMap := map[string]string{
		"include":   "**Keyword**: `include`\n\nImports definitions from another Frugal file.",
		"namespace": "**Keyword**: `namespace`\n\nDefines the namespace/package for generated code.",
		"const":     "**Keyword**: `const`\n\nDefines a constant value.",
		"typedef":   "**Keyword**: `typedef`\n\nCreates a type alias.",
		"struct":    "**Keyword**: `struct`\n\nDefines a data structure with named fields.",
		"enum":      "**Keyword**: `enum`\n\nDefines an enumeration type with named values.",
		"exception": "**Keyword**: `exception`\n\nDefines an exception type that can be thrown.",
		"service":   "**Keyword**: `service`\n\nDefines an RPC service with methods.",
		"scope":     "**Keyword**: `scope`\n\nDefines a pub/sub scope for event messaging (Frugal extension).",
		"oneway":    "**Keyword**: `oneway`\n\nMethod modifier indicating no response is expected.",
		"throws":    "**Keyword**: `throws`\n\nSpecifies exceptions that a method can throw.",
		"extends":   "**Keyword**: `extends`\n\nIndicates inheritance from another service.",
		"required":  "**Keyword**: `required`\n\nField modifier indicating the field must be set.",
		"optional":  "**Keyword**: `optional`\n\nField modifier indicating the field is optional.",
		"prefix":    "**Keyword**: `prefix`\n\nDefines topic prefix for pub/sub messaging.",
	}
	
	if info, exists := keywordMap[keyword]; exists {
		return info
	}
	return ""
}
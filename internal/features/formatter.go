package features

import (
	"fmt"
	"regexp"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

const (
	formatterNodeTypeHeader               = "header"
	formatterNodeTypeInclude              = "include"
	formatterNodeTypeComment              = "comment"
	formatterNodeTypeWhitespace           = "whitespace"
	formatterNodeTypeNamespaceDeclaration = "namespace_declaration"
	formatterNodeTypeScopeOperation       = "scope_operation"
	formatterNodeTypeEnumField            = "enum_field"
)

// FrugalFormatter provides comprehensive AST-based formatting for Frugal files
type FrugalFormatter struct {
	indentSize    int
	useSpaces     bool
	insertSpaces  bool
	maxLineLength int
	alignFields   bool
	sortImports   bool
}

// fieldComponents represents the components of a field definition
type fieldComponents struct {
	id, qualifier, fieldType, name string
}

// NewFrugalFormatter creates a new formatter with the given options
func NewFrugalFormatter(options protocol.FormattingOptions) *FrugalFormatter {
	formatter := &FrugalFormatter{
		indentSize:    4,
		useSpaces:     true,
		insertSpaces:  true,
		maxLineLength: 100,
		alignFields:   true,
		sortImports:   true,
	}

	// Apply LSP formatting options
	if tabSize, ok := options["tabSize"]; ok {
		if ts, ok := tabSize.(float64); ok {
			formatter.indentSize = int(ts)
		} else if ts, ok := tabSize.(int); ok {
			formatter.indentSize = ts
		}
	}

	if insertSpaces, ok := options["insertSpaces"]; ok {
		if is, ok := insertSpaces.(bool); ok {
			formatter.insertSpaces = is
			formatter.useSpaces = is
		}
	}

	return formatter
}

// FormatDocument formats the entire Frugal document using AST
func (f *FrugalFormatter) FormatDocument(doc *document.Document) (string, error) {
	if doc.ParseResult == nil || doc.ParseResult.GetRootNode() == nil {
		return string(doc.Content), nil
	}

	root := doc.ParseResult.GetRootNode()
	return f.formatNode(root, doc.Content, 0), nil
}

// FormatRange formats a specific range of the document
func (f *FrugalFormatter) FormatRange(doc *document.Document, rng protocol.Range) (string, error) {
	// For now, format the entire document
	// TODO: Optimize to format only the affected nodes
	return f.FormatDocument(doc)
}

// formatNode recursively formats a node and its children
func (f *FrugalFormatter) formatNode(node *tree_sitter.Node, source []byte, indentLevel int) string {
	if node == nil {
		return ""
	}

	nodeType := node.Kind()

	switch nodeType {
	case "document":
		return f.formatDocument(node, source, indentLevel)
	case formatterNodeTypeComment:
		return f.formatComment(node, source, indentLevel)
	case "definition":
		// Handle the intermediate definition node
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			return f.formatNode(child, source, indentLevel) // Format the first child
		}
		return f.formatGenericNode(node, source, indentLevel)
	case formatterNodeTypeInclude:
		return f.formatInclude(node, source)
	case formatterNodeTypeHeader:
		// Handle the header wrapper for includes
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			if child.Kind() == formatterNodeTypeInclude {
				return f.formatInclude(child, source)
			}
		}
		return f.formatGenericNode(node, source, indentLevel)
	case formatterNodeTypeNamespaceDeclaration:
		return f.formatNamespace(node, source)
	case diagnosticsNodeTypeServiceDefinition:
		return f.formatService(node, source, indentLevel)
	case diagnosticsNodeTypeScopeDefinition:
		return f.formatScope(node, source, indentLevel)
	case nodeTypeStructDefinition:
		return f.formatStruct(node, source, indentLevel)
	case diagnosticsNodeTypeEnumDefinition:
		return f.formatEnum(node, source, indentLevel)
	case diagnosticsNodeTypeExceptionDefinition:
		return f.formatException(node, source, indentLevel)
	case diagnosticsNodeTypeConstDefinition:
		return f.formatConst(node, source, indentLevel)
	case diagnosticsNodeTypeTypedefDefinition:
		return f.formatTypedef(node, source, indentLevel)
	default:
		// Default formatting for unknown nodes
		return f.formatGenericNode(node, source, indentLevel)
	}
}

// formatDocument formats the root document node
//
//nolint:gocognit // Complex formatting logic is inherently complex
func (f *FrugalFormatter) formatDocument(node *tree_sitter.Node, source []byte, _ int) string {
	var sections []string
	var currentSection []string

	// Group related definitions together
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		childType := child.Kind()

		// Skip only whitespace, but preserve comments
		if childType == formatterNodeTypeWhitespace {
			continue
		}

		formatted := f.formatNode(child, source, 0)
		if strings.TrimSpace(formatted) != "" {
			currentSection = append(currentSection, formatted)

			// Comments don't trigger section breaks - they stay with the next definition
			if childType == formatterNodeTypeComment {
				continue
			}

			// Determine the actual child type for section breaks
			actualChildType := f.getActualDefinitionType(child)

			// Look ahead to next non-empty child to determine if we should break
			shouldBreak := f.shouldAddSectionBreak(actualChildType)
			if shouldBreak {
				// Special case: don't break between include and namespace if they are consecutive
				// and namespace is the only thing after include (simple include + namespace case)
				if actualChildType == formatterNodeTypeInclude {
					nextChildType := f.getNextDefinitionType(node, source, i)
					if nextChildType == formatterNodeTypeNamespaceDeclaration {
						// Count remaining definitions - if only namespace left, don't break
						remainingDefs := f.countRemainingDefinitions(node, source, i)
						if remainingDefs == 1 {
							shouldBreak = false // Don't break between include and namespace when they're alone
						}
					}
				}
			}

			if shouldBreak {
				if len(currentSection) > 0 {
					sections = append(sections, strings.Join(currentSection, "\n"))
					currentSection = nil
				}
			}
		}
	}

	// Add remaining section
	if len(currentSection) > 0 {
		sections = append(sections, strings.Join(currentSection, "\n"))
	}

	// Join sections with double newlines
	result := strings.Join(sections, "\n\n")

	return result
}

// getActualDefinitionType gets the actual definition type from a definition node
func (f *FrugalFormatter) getActualDefinitionType(node *tree_sitter.Node) string {
	if node.Kind() == "definition" {
		// Look for the actual definition child
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			childType := child.Kind()
			if strings.HasSuffix(childType, "_definition") ||
				childType == formatterNodeTypeInclude ||
				childType == formatterNodeTypeNamespaceDeclaration {
				return childType
			}
		}
	} else if node.Kind() == formatterNodeTypeHeader {
		// Look inside header for the actual type
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			childType := child.Kind()
			if childType == formatterNodeTypeInclude {
				return formatterNodeTypeInclude
			} else if childType == "namespace" {
				return formatterNodeTypeNamespaceDeclaration
			}
		}
	}
	return node.Kind()
}

// getNextDefinitionType looks ahead to find the next non-empty definition type
func (f *FrugalFormatter) getNextDefinitionType(parentNode *tree_sitter.Node, _ []byte, currentIndex uint) string {
	childCount := parentNode.ChildCount()
	for i := currentIndex + 1; i < childCount; i++ {
		child := parentNode.Child(i)
		childType := child.Kind()

		// Skip whitespace and comments
		if childType == formatterNodeTypeComment || childType == formatterNodeTypeWhitespace {
			continue
		}

		// Return the actual definition type for the next child
		return f.getActualDefinitionType(child)
	}
	return ""
}

// countRemainingDefinitions counts how many definitions are left after the current index
func (f *FrugalFormatter) countRemainingDefinitions(parentNode *tree_sitter.Node, _ []byte, currentIndex uint) int {
	count := 0
	childCount := parentNode.ChildCount()
	for i := currentIndex + 1; i < childCount; i++ {
		child := parentNode.Child(i)
		childType := child.Kind()

		// Skip whitespace and comments
		if childType == formatterNodeTypeComment || childType == formatterNodeTypeWhitespace {
			continue
		}

		count++
	}
	return count
}

// shouldAddSectionBreak determines if we should add a section break after this node type
func (f *FrugalFormatter) shouldAddSectionBreak(nodeType string) bool {
	switch nodeType {
	case formatterNodeTypeNamespaceDeclaration, formatterNodeTypeInclude, formatterNodeTypeHeader:
		return true
	case diagnosticsNodeTypeServiceDefinition, diagnosticsNodeTypeScopeDefinition:
		return true
	case nodeTypeStructDefinition, diagnosticsNodeTypeExceptionDefinition:
		return false // Group structs and exceptions together
	case diagnosticsNodeTypeEnumDefinition:
		return true // Separate enums from other definitions
	case diagnosticsNodeTypeConstDefinition, diagnosticsNodeTypeTypedefDefinition:
		return false // Group these together
	default:
		return false
	}
}

// formatInclude formats include statements
func (f *FrugalFormatter) formatInclude(node *tree_sitter.Node, source []byte) string {
	// Find the literal_string child to get the path
	var includePath string
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child.Kind() == "literal_string" {
			pathText := ast.GetText(child, source)
			// Remove quotes and extract just the path
			includePath = strings.Trim(pathText, "\"")
			break
		}
	}

	if includePath != "" {
		return fmt.Sprintf("include \"%s\"", includePath)
	}

	// Fallback to regex if AST parsing fails
	nodeText := ast.GetText(node, source)
	re := regexp.MustCompile(`include\s*"([^"]+)"`)
	if matches := re.FindStringSubmatch(nodeText); len(matches) > 1 {
		return fmt.Sprintf("include \"%s\"", matches[1])
	}

	return strings.TrimSpace(nodeText)
}

// formatNamespace formats namespace declarations
func (f *FrugalFormatter) formatNamespace(node *tree_sitter.Node, source []byte) string {
	nodeText := ast.GetText(node, source)

	// Extract namespace parts
	re := regexp.MustCompile(`namespace\s+(\w+)\s+(.+)`)
	if matches := re.FindStringSubmatch(strings.TrimSpace(nodeText)); len(matches) > 2 {
		return fmt.Sprintf("namespace %s %s", matches[1], matches[2])
	}

	return strings.TrimSpace(nodeText)
}

// formatService formats service definitions
func (f *FrugalFormatter) formatService(node *tree_sitter.Node, source []byte, indentLevel int) string {
	// Check if service contains comments - if so, use conservative formatting
	if f.nodeContainsComments(node, source) {
		return f.formatConservatively(node, source, indentLevel)
	}

	serviceName := f.extractIdentifier(node, source)
	if serviceName == "" {
		return f.formatGenericNode(node, source, indentLevel)
	}

	indent := f.getIndent(indentLevel)
	bodyIndent := f.getIndent(indentLevel + 1)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%sservice %s {", indent, serviceName))

	// Format service methods
	serviceBody := ast.FindNodeByType(node, "service_body")
	if serviceBody != nil {
		methods := f.extractServiceMethods(serviceBody, source)
		if len(methods) > 0 {
			result.WriteString("\n")
			for i, method := range methods {
				result.WriteString(bodyIndent + method)
				if i < len(methods)-1 {
					result.WriteString(",")
				}
				result.WriteString("\n")
			}
			result.WriteString(indent)
		}
	}

	result.WriteString("}")
	return result.String()
}

// formatComment formats comment nodes
func (f *FrugalFormatter) formatComment(node *tree_sitter.Node, source []byte, indentLevel int) string {
	indent := f.getIndent(indentLevel)
	commentText := strings.TrimSpace(ast.GetText(node, source))

	// Handle multi-line comments with proper star alignment
	if strings.HasPrefix(commentText, "/*") && strings.Contains(commentText, "\n") {
		return f.formatMultiLineComment(commentText, indent)
	}

	// Preserve the original comment format for single-line comments
	return indent + commentText
}

// formatMultiLineComment formats multi-line comments with proper star alignment
//
//nolint:gocognit // Complex comment formatting logic handles many edge cases
func (f *FrugalFormatter) formatMultiLineComment(commentText, indent string) string {
	lines := strings.Split(commentText, "\n")
	if len(lines) <= 1 {
		return indent + commentText
	}

	var formattedLines []string

	// First line: should be "/**"
	firstLine := strings.TrimSpace(lines[0])
	var firstLineContent string

	if firstLine == "/*" {
		// Just opening, no content
		formattedLines = append(formattedLines, indent+"/**")
	} else if strings.HasPrefix(firstLine, "/**") {
		// Already starts with /** - check if it has content
		if len(firstLine) > 3 {
			// Has content after /** - extract it
			firstLineContent = strings.TrimSpace(firstLine[3:])
			formattedLines = append(formattedLines, indent+"/**")
			if firstLineContent != "" {
				formattedLines = append(formattedLines, indent+" * "+firstLineContent)
			}
		} else {
			// Just /** with no content
			formattedLines = append(formattedLines, indent+"/**")
		}
	} else if strings.HasPrefix(firstLine, "/*") {
		// Has content on first line - extract it
		firstLineContent = strings.TrimSpace(firstLine[2:]) // Remove "/*"
		formattedLines = append(formattedLines, indent+"/**")
		if firstLineContent != "" {
			formattedLines = append(formattedLines, indent+" * "+firstLineContent)
		}
	} else {
		// This shouldn't happen in a well-formed comment, but handle it
		formattedLines = append(formattedLines, indent+"/**")
		if firstLine != "" {
			formattedLines = append(formattedLines, indent+" * "+firstLine)
		}
	}

	// Middle lines: align stars with space
	for i := 1; i < len(lines)-1; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			formattedLines = append(formattedLines, indent+" *")
		} else {
			// Ensure line starts with " * "
			if strings.HasPrefix(line, "*") {
				if len(line) == 1 || line[1] != ' ' {
					line = " * " + line[1:]
				} else {
					line = " " + line
				}
			} else if !strings.HasPrefix(line, " *") {
				line = " * " + line
			}
			// Ensure space after star if there's content
			if strings.HasPrefix(line, " *") && len(line) > 2 && line[2] != ' ' {
				line = " * " + line[2:]
			}
			formattedLines = append(formattedLines, indent+line)
		}
	}

	// Last line: should be " */"
	lastLine := strings.TrimSpace(lines[len(lines)-1])
	if lastLine == "*/" {
		formattedLines = append(formattedLines, indent+" */")
	} else if strings.HasSuffix(lastLine, "*/") {
		// Handle case where last line has content and */
		if len(lastLine) > 2 {
			content := strings.TrimSpace(lastLine[:len(lastLine)-2])
			if content != "" {
				// Remove leading star if present
				if strings.HasPrefix(content, "*") {
					content = strings.TrimSpace(content[1:])
				}
				if content != "" {
					formattedLines = append(formattedLines, indent+" * "+content)
				}
			}
		}
		formattedLines = append(formattedLines, indent+" */")
	} else {
		// Add the last line content
		if lastLine != "" {
			if strings.HasPrefix(lastLine, "*") {
				lastLine = " * " + strings.TrimSpace(lastLine[1:])
			} else if !strings.HasPrefix(lastLine, " *") {
				lastLine = " * " + lastLine
			}
			formattedLines = append(formattedLines, indent+lastLine)
		}
		formattedLines = append(formattedLines, indent+" */")
	}

	return strings.Join(formattedLines, "\n")
}

// formatScope formats scope definitions
func (f *FrugalFormatter) formatScope(node *tree_sitter.Node, source []byte, indentLevel int) string {
	indent := f.getIndent(indentLevel)
	bodyIndent := f.getIndent(indentLevel + 1)

	// Extract scope name and prefix
	scopeName := f.extractIdentifier(node, source)
	prefix := f.extractScopePrefix(node, source)

	var result strings.Builder
	if prefix != "" {
		result.WriteString(fmt.Sprintf("%sscope %s prefix \"%s\" {", indent, scopeName, prefix))
	} else {
		result.WriteString(fmt.Sprintf("%sscope %s {", indent, scopeName))
	}

	// Format scope events
	scopeBody := ast.FindNodeByType(node, "scope_body")
	if scopeBody != nil {
		events := f.extractScopeEvents(scopeBody, source)
		if len(events) > 0 {
			result.WriteString("\n")
			for i, event := range events {
				result.WriteString(bodyIndent + event)
				if i < len(events)-1 {
					result.WriteString(",")
				}
				result.WriteString("\n")
			}
			result.WriteString(indent)
		}
	}

	result.WriteString("}")
	return result.String()
}

// formatStruct formats struct definitions
func (f *FrugalFormatter) formatStruct(node *tree_sitter.Node, source []byte, indentLevel int) string {
	// Check if struct contains comments - if so, use conservative formatting
	if f.nodeContainsComments(node, source) {
		return f.formatConservatively(node, source, indentLevel)
	}

	structName := f.extractIdentifier(node, source)
	if structName == "" {
		return f.formatGenericNode(node, source, indentLevel)
	}

	indent := f.getIndent(indentLevel)
	bodyIndent := f.getIndent(indentLevel + 1)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%sstruct %s {", indent, structName))

	// Format struct fields
	structBody := ast.FindNodeByType(node, "struct_body")
	if structBody != nil {
		fields := f.extractStructFields(structBody, source)
		if len(fields) > 0 {
			result.WriteString("\n")

			// Align fields if requested
			if f.alignFields {
				fields = f.alignFieldDefinitions(fields)
			}

			for i, field := range fields {
				result.WriteString(bodyIndent + field)
				if i < len(fields)-1 {
					result.WriteString(",")
				}
				result.WriteString("\n")
			}
			result.WriteString(indent)
		}
	}

	result.WriteString("}")
	return result.String()
}

// formatEnum formats enum definitions
func (f *FrugalFormatter) formatEnum(node *tree_sitter.Node, source []byte, indentLevel int) string {
	enumName := f.extractIdentifier(node, source)
	if enumName == "" {
		return f.formatGenericNode(node, source, indentLevel)
	}

	indent := f.getIndent(indentLevel)
	bodyIndent := f.getIndent(indentLevel + 1)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%senum %s {", indent, enumName))

	// Format enum values
	enumBody := ast.FindNodeByType(node, "enum_body")
	if enumBody != nil {
		values := f.extractEnumValues(enumBody, source)
		if len(values) > 0 {
			result.WriteString("\n")

			// Align enum values if requested
			if f.alignFields {
				values = f.alignEnumValues(values)
			}

			for i, value := range values {
				result.WriteString(bodyIndent + value)
				if i < len(values)-1 {
					result.WriteString(",")
				}
				result.WriteString("\n")
			}
			result.WriteString(indent)
		}
	}

	result.WriteString("}")
	return result.String()
}

// formatException formats exception definitions
func (f *FrugalFormatter) formatException(node *tree_sitter.Node, source []byte, indentLevel int) string {
	// Exceptions are formatted the same as structs
	exceptionName := f.extractIdentifier(node, source)
	if exceptionName == "" {
		return f.formatGenericNode(node, source, indentLevel)
	}

	indent := f.getIndent(indentLevel)
	bodyIndent := f.getIndent(indentLevel + 1)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%sexception %s {", indent, exceptionName))

	// Format exception fields (same as struct fields)
	exceptionBody := ast.FindNodeByType(node, "exception_body")
	if exceptionBody == nil {
		// Try struct_body as fallback
		exceptionBody = ast.FindNodeByType(node, "struct_body")
	}

	if exceptionBody != nil {
		fields := f.extractStructFields(exceptionBody, source)
		if len(fields) > 0 {
			result.WriteString("\n")

			if f.alignFields {
				fields = f.alignFieldDefinitions(fields)
			}

			for i, field := range fields {
				result.WriteString(bodyIndent + field)
				if i < len(fields)-1 {
					result.WriteString(",")
				}
				result.WriteString("\n")
			}
			result.WriteString(indent)
		}
	}

	result.WriteString("}")
	return result.String()
}

// formatConst formats constant definitions
func (f *FrugalFormatter) formatConst(node *tree_sitter.Node, source []byte, indentLevel int) string {
	indent := f.getIndent(indentLevel)
	nodeText := strings.TrimSpace(ast.GetText(node, source))

	// Parse const definition: const type name = value
	re := regexp.MustCompile(`const\s+(\w+)\s+(\w+)\s*=\s*(.+?)(?:;)?$`)
	if matches := re.FindStringSubmatch(nodeText); len(matches) > 3 {
		constType := matches[1]
		constName := matches[2]
		constValue := strings.TrimSpace(matches[3])

		// Add semicolon if not present
		if !strings.HasSuffix(constValue, ";") {
			constValue += ";"
		}

		return fmt.Sprintf("%sconst %s %s = %s", indent, constType, constName, constValue)
	}

	return indent + nodeText
}

// formatTypedef formats typedef definitions
func (f *FrugalFormatter) formatTypedef(node *tree_sitter.Node, source []byte, indentLevel int) string {
	indent := f.getIndent(indentLevel)

	var baseType, aliasName string

	// Extract components from AST
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		switch child.Kind() {
		case nodeTypeFieldType:
			baseType = strings.TrimSpace(ast.GetText(child, source))
		case nodeTypeIdentifier:
			aliasName = strings.TrimSpace(ast.GetText(child, source))
		}
	}

	if baseType != "" && aliasName != "" {
		return fmt.Sprintf("%stypedef %s %s", indent, baseType, aliasName)
	}

	// Fallback to regex
	nodeText := strings.TrimSpace(ast.GetText(node, source))
	re := regexp.MustCompile(`typedef\s+(.+?)\s+(\w+)`)
	if matches := re.FindStringSubmatch(nodeText); len(matches) > 2 {
		baseType := strings.TrimSpace(matches[1])
		aliasName := matches[2]
		return fmt.Sprintf("%stypedef %s %s", indent, baseType, aliasName)
	}

	return indent + nodeText
}

// formatGenericNode provides default formatting for unrecognized nodes
func (f *FrugalFormatter) formatGenericNode(node *tree_sitter.Node, source []byte, indentLevel int) string {
	indent := f.getIndent(indentLevel)
	nodeText := strings.TrimSpace(ast.GetText(node, source))
	return indent + nodeText
}

// Helper methods for extracting information from nodes

// extractIdentifier finds the first identifier in a node
func (f *FrugalFormatter) extractIdentifier(node *tree_sitter.Node, source []byte) string {
	identifierNode := ast.FindNodeByType(node, "identifier")
	if identifierNode != nil {
		return ast.GetText(identifierNode, source)
	}
	return ""
}

// extractScopePrefix extracts the prefix from a scope definition
func (f *FrugalFormatter) extractScopePrefix(node *tree_sitter.Node, source []byte) string {
	// Look for scope_prefix child node first
	scopePrefixNode := ast.FindNodeByType(node, "scope_prefix")
	if scopePrefixNode != nil {
		// Find the literal_string within the scope_prefix
		childCount := scopePrefixNode.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := scopePrefixNode.Child(i)
			if child.Kind() == "literal_string" {
				prefixText := ast.GetText(child, source)
				// Remove quotes
				return strings.Trim(prefixText, "\"")
			}
		}
	}

	// Fallback to regex
	nodeText := ast.GetText(node, source)
	re := regexp.MustCompile(`prefix\s*"([^"]+)"`)
	if matches := re.FindStringSubmatch(nodeText); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractServiceMethods extracts and formats service methods
func (f *FrugalFormatter) extractServiceMethods(serviceBody *tree_sitter.Node, source []byte) []string {
	var methods []string

	childCount := serviceBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := serviceBody.Child(i)
		if child.Kind() == nodeTypeFunctionDefinition {
			method := f.formatServiceMethod(child, source)
			if method != "" {
				methods = append(methods, method)
			}
		}
	}

	return methods
}

// formatServiceMethod formats a single service method
func (f *FrugalFormatter) formatServiceMethod(methodNode *tree_sitter.Node, source []byte) string {
	methodText := strings.TrimSpace(ast.GetText(methodNode, source))

	// Remove trailing comma if present (it's handled separately)
	methodText = strings.TrimSuffix(methodText, ",")
	methodText = strings.TrimSpace(methodText)

	// Clean up spacing and formatting
	methodText = f.normalizeMethodSignature(methodText)

	return methodText
}

// normalizeMethodSignature normalizes spacing in method signatures
func (f *FrugalFormatter) normalizeMethodSignature(signature string) string {
	// Remove extra whitespace
	signature = regexp.MustCompile(`\s+`).ReplaceAllString(signature, " ")
	signature = strings.TrimSpace(signature)

	// Format throws clause - ensure proper spacing and parentheses
	signature = regexp.MustCompile(`\s*throws\s*\(\s*`).ReplaceAllString(signature, " throws (")
	signature = regexp.MustCompile(`\s*\)\s*$`).ReplaceAllString(signature, ")")

	// Format parameter parentheses - ensure proper spacing
	signature = regexp.MustCompile(`\(\s*`).ReplaceAllString(signature, "(")
	signature = regexp.MustCompile(`\s*\)`).ReplaceAllString(signature, ")")

	// Ensure colon formatting in parameters: "1: type param"
	signature = regexp.MustCompile(`(\d+)\s*:\s*`).ReplaceAllString(signature, "$1: ")

	// Format parameters - ensure space after commas but handle parentheses properly
	signature = regexp.MustCompile(`,\s*`).ReplaceAllString(signature, ", ")

	return signature
}

// extractScopeEvents extracts and formats scope events
func (f *FrugalFormatter) extractScopeEvents(scopeBody *tree_sitter.Node, source []byte) []string {
	var events []string

	childCount := scopeBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := scopeBody.Child(i)
		if child.Kind() == formatterNodeTypeScopeOperation {
			event := strings.TrimSpace(ast.GetText(child, source))
			// Remove trailing comma if present (it's handled separately)
			event = strings.TrimSuffix(event, ",")
			event = strings.TrimSpace(event)
			if event != "" {
				// Normalize event format "EventName: Type"
				event = f.normalizeEventDefinition(event)
				events = append(events, event)
			}
		}
	}

	return events
}

// normalizeEventDefinition normalizes event definitions
func (f *FrugalFormatter) normalizeEventDefinition(event string) string {
	// Format "EventName: Type"
	if parts := strings.Split(event, ":"); len(parts) == 2 {
		eventName := strings.TrimSpace(parts[0])
		eventType := strings.TrimSpace(parts[1])
		eventType = strings.TrimSuffix(eventType, ",")
		return fmt.Sprintf("%s: %s", eventName, eventType)
	}
	return strings.TrimSuffix(strings.TrimSpace(event), ",")
}

// extractStructFields extracts and formats struct fields
func (f *FrugalFormatter) extractStructFields(structBody *tree_sitter.Node, source []byte) []string {
	var fields []string

	childCount := structBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := structBody.Child(i)
		if child.Kind() == nodeTypeField {
			field := strings.TrimSpace(ast.GetText(child, source))
			if field != "" {
				field = f.normalizeFieldDefinition(field)
				fields = append(fields, field)
			}
		}
	}

	return fields
}

// normalizeFieldDefinition normalizes field definitions
func (f *FrugalFormatter) normalizeFieldDefinition(field string) string {
	// Remove trailing comma
	field = strings.TrimSuffix(strings.TrimSpace(field), ",")

	// Normalize spacing: "1: required string name"
	field = regexp.MustCompile(`(\d+)\s*:\s*`).ReplaceAllString(field, "$1: ")
	field = regexp.MustCompile(`\s+`).ReplaceAllString(field, " ")

	return field
}

// extractEnumValues extracts and formats enum values
func (f *FrugalFormatter) extractEnumValues(enumBody *tree_sitter.Node, source []byte) []string {
	var values []string

	childCount := enumBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := enumBody.Child(i)
		if child.Kind() == formatterNodeTypeEnumField {
			value := strings.TrimSpace(ast.GetText(child, source))
			// Remove trailing comma if present (it's handled separately)
			value = strings.TrimSuffix(value, ",")
			value = strings.TrimSpace(value)
			if value != "" {
				value = f.normalizeEnumValue(value)
				values = append(values, value)
			}
		}
	}

	return values
}

// normalizeEnumValue normalizes enum value definitions
func (f *FrugalFormatter) normalizeEnumValue(value string) string {
	// Remove trailing comma
	value = strings.TrimSuffix(strings.TrimSpace(value), ",")

	// Normalize "NAME = value" format
	if parts := strings.Split(value, "="); len(parts) == 2 {
		name := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		return fmt.Sprintf("%s = %s", name, val)
	}

	return value
}

// Alignment helpers

// alignFieldDefinitions aligns field definitions for better readability
func (f *FrugalFormatter) alignFieldDefinitions(fields []string) []string {
	if len(fields) <= 1 {
		return fields
	}

	// Find the maximum width of each component
	var maxIdWidth, maxTypeWidth int

	var components []fieldComponents

	for _, field := range fields {
		comp := f.parseFieldComponents(field)
		components = append(components, comp)

		if len(comp.id) > maxIdWidth {
			maxIdWidth = len(comp.id)
		}
		if len(comp.qualifier+" "+comp.fieldType) > maxTypeWidth {
			maxTypeWidth = len(comp.qualifier + " " + comp.fieldType)
		}
	}

	// Reconstruct with alignment
	var aligned []string
	for _, comp := range components {
		idPart := comp.id + strings.Repeat(" ", maxIdWidth-len(comp.id))

		var typePart string
		if comp.qualifier != "" {
			typePart = comp.qualifier + " " + comp.fieldType
		} else {
			typePart = comp.fieldType
		}
		typePart += strings.Repeat(" ", maxTypeWidth-len(typePart))

		aligned = append(aligned, fmt.Sprintf("%s: %s %s", idPart, typePart, comp.name))
	}

	return aligned
}

// parseFieldComponents parses a field definition into components
func (f *FrugalFormatter) parseFieldComponents(field string) fieldComponents {
	// Parse "1: required string name" format
	parts := strings.SplitN(field, ":", 2)
	if len(parts) != 2 {
		return fieldComponents{fieldType: field}
	}

	id := strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

	restParts := strings.Fields(rest)
	if len(restParts) == 0 {
		return fieldComponents{id: id}
	}

	var qualifier, fieldType, name string

	if len(restParts) >= 3 && (restParts[0] == "required" || restParts[0] == "optional") {
		qualifier = restParts[0]
		fieldType = restParts[1]
		name = strings.Join(restParts[2:], " ")
	} else if len(restParts) >= 2 {
		fieldType = restParts[0]
		name = strings.Join(restParts[1:], " ")
	} else {
		fieldType = restParts[0]
	}

	return fieldComponents{
		id:        id,
		qualifier: qualifier,
		fieldType: fieldType,
		name:      name,
	}
}

// alignEnumValues aligns enum value definitions
func (f *FrugalFormatter) alignEnumValues(values []string) []string {
	if len(values) <= 1 {
		return values
	}

	maxNameWidth := 0
	var names, vals []string

	for _, value := range values {
		if parts := strings.Split(value, "="); len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			names = append(names, name)
			vals = append(vals, val)
			if len(name) > maxNameWidth {
				maxNameWidth = len(name)
			}
		} else {
			names = append(names, value)
			vals = append(vals, "")
		}
	}

	var aligned []string
	for i, name := range names {
		if vals[i] != "" {
			paddedName := name + strings.Repeat(" ", maxNameWidth-len(name))
			aligned = append(aligned, fmt.Sprintf("%s = %s", paddedName, vals[i]))
		} else {
			aligned = append(aligned, name)
		}
	}

	return aligned
}

// getIndent returns the indentation string for the given level
func (f *FrugalFormatter) getIndent(level int) string {
	if f.useSpaces {
		return strings.Repeat(" ", level*f.indentSize)
	}
	return strings.Repeat("\t", level)
}

// nodeContainsComments checks if a node or its children contain comments
//
//nolint:unparam // source parameter maintained for API consistency
func (f *FrugalFormatter) nodeContainsComments(node *tree_sitter.Node, source []byte) bool {
	if node == nil {
		return false
	}

	// Check current node
	if node.Kind() == formatterNodeTypeComment {
		return true
	}

	// Check children recursively
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if f.nodeContainsComments(child, source) {
			return true
		}
	}

	return false
}

// formatConservatively provides conservative formatting that preserves comments and spacing
//
//nolint:gocognit // Complex conservative formatting preserves existing structure
func (f *FrugalFormatter) formatConservatively(node *tree_sitter.Node, source []byte, indentLevel int) string {
	indent := f.getIndent(indentLevel)
	nodeText := ast.GetText(node, source)

	// Preserve the original text but fix basic indentation
	lines := strings.Split(nodeText, "\n")
	var formattedLines []string

	inMultiLineComment := false
	commentStartIndent := 0

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			if inMultiLineComment {
				// Preserve proper spacing for empty lines in multi-line comments
				formattedLines = append(formattedLines, strings.Repeat(" ", commentStartIndent)+" *")
			} else {
				formattedLines = append(formattedLines, "")
			}
			continue
		}

		// Check if we're entering a multi-line comment
		if strings.HasPrefix(trimmedLine, "/*") && strings.Contains(trimmedLine, "*") {
			inMultiLineComment = true
			// Calculate the base indent for this comment
			originalIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if i == 0 {
				commentStartIndent = len(indent)
			} else {
				commentStartIndent = originalIndent
			}
		}

		// Handle multi-line comment formatting
		if inMultiLineComment {
			baseIndent := strings.Repeat(" ", commentStartIndent)

			if strings.HasPrefix(trimmedLine, "/**") {
				// Opening line
				formattedLines = append(formattedLines, baseIndent+"/**")
			} else if strings.HasPrefix(trimmedLine, "/*") && !strings.HasPrefix(trimmedLine, "/**") {
				// Convert /* to /**
				content := strings.TrimSpace(trimmedLine[2:])
				formattedLines = append(formattedLines, baseIndent+"/**")
				if content != "" && !strings.HasSuffix(content, "*/") {
					formattedLines = append(formattedLines, baseIndent+" * "+content)
				}
			} else if trimmedLine == "*/" {
				// Closing line
				formattedLines = append(formattedLines, baseIndent+" */")
				inMultiLineComment = false
			} else if strings.HasSuffix(trimmedLine, "*/") {
				// Content and closing on same line
				content := strings.TrimSpace(trimmedLine[:len(trimmedLine)-2])
				if strings.HasPrefix(content, "*") {
					content = strings.TrimSpace(content[1:])
				}
				if content != "" {
					formattedLines = append(formattedLines, baseIndent+" * "+content)
				}
				formattedLines = append(formattedLines, baseIndent+" */")
				inMultiLineComment = false
			} else {
				// Content line
				content := trimmedLine
				if strings.HasPrefix(content, "*") {
					content = strings.TrimSpace(content[1:])
				}
				formattedLines = append(formattedLines, baseIndent+" * "+content)
			}
		} else {
			// Regular line formatting
			// For the first line, use the specified indent level
			if i == 0 {
				formattedLines = append(formattedLines, indent+trimmedLine)
			} else {
				// For subsequent lines, preserve relative indentation
				originalIndent := len(line) - len(strings.TrimLeft(line, " \t"))
				if originalIndent > 0 {
					// Preserve relative indentation but start from base indent
					relativeIndent := f.getIndent(indentLevel + 1)
					formattedLines = append(formattedLines, relativeIndent+trimmedLine)
				} else {
					formattedLines = append(formattedLines, indent+trimmedLine)
				}
			}
		}
	}

	return strings.Join(formattedLines, "\n")
}

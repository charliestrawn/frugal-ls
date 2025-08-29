package workspace

import (
	"sort"
	"strings"
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"frugal-ls/internal/document"
	"frugal-ls/pkg/ast"
)

// IndexedSymbol represents a symbol with additional indexing information
type IndexedSymbol struct {
	ast.Symbol
	URI           string
	ContainerName string // Parent container (service, struct, etc.)
	FullName      string // Fully qualified name for better search
}

// SymbolIndex provides fast workspace-wide symbol search
type SymbolIndex struct {
	mu             sync.RWMutex
	symbols        map[string][]IndexedSymbol       // URI -> symbols
	nameIndex      map[string][]IndexedSymbol       // lowercased name -> symbols
	typeIndex      map[ast.NodeType][]IndexedSymbol // type -> symbols
	containerIndex map[string][]IndexedSymbol       // container name -> symbols
	version        int64
}

// NewSymbolIndex creates a new workspace symbol index
func NewSymbolIndex() *SymbolIndex {
	return &SymbolIndex{
		symbols:        make(map[string][]IndexedSymbol),
		nameIndex:      make(map[string][]IndexedSymbol),
		typeIndex:      make(map[ast.NodeType][]IndexedSymbol),
		containerIndex: make(map[string][]IndexedSymbol),
		version:        0,
	}
}

// UpdateDocument updates symbols for a specific document
func (si *SymbolIndex) UpdateDocument(doc *document.Document) {
	if !doc.IsValidFrugalFile() {
		return
	}

	si.mu.Lock()
	defer si.mu.Unlock()

	// Remove old symbols for this document
	si.removeDocumentSymbols(doc.URI)

	// Extract and index new symbols
	symbols := doc.GetSymbols()
	indexedSymbols := si.processSymbols(symbols, doc)

	// Update indices
	si.symbols[doc.URI] = indexedSymbols
	for _, symbol := range indexedSymbols {
		si.addToIndices(symbol)
	}

	si.version++
}

// RemoveDocument removes all symbols for a document
func (si *SymbolIndex) RemoveDocument(uri string) {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.removeDocumentSymbols(uri)
	si.version++
}

// Search finds symbols matching the query
func (si *SymbolIndex) Search(query string, limit int) []protocol.SymbolInformation {
	si.mu.RLock()
	defer si.mu.RUnlock()

	if query == "" {
		return si.getAllSymbols(limit)
	}

	// Find matching symbols using multiple strategies
	matches := si.findMatches(query)

	// Sort by relevance
	sort.Slice(matches, func(i, j int) bool {
		return si.calculateRelevance(matches[i], query) > si.calculateRelevance(matches[j], query)
	})

	// Convert to LSP format and apply limit
	return si.convertToSymbolInformation(matches, limit)
}

// SearchByType finds symbols of specific types
func (si *SymbolIndex) SearchByType(symbolTypes []ast.NodeType, query string, limit int) []protocol.SymbolInformation {
	si.mu.RLock()
	defer si.mu.RUnlock()

	var candidates []IndexedSymbol
	for _, symbolType := range symbolTypes {
		if symbols, exists := si.typeIndex[symbolType]; exists {
			candidates = append(candidates, symbols...)
		}
	}

	// Filter by query if provided
	if query != "" {
		candidates = si.filterByQuery(candidates, query)
	}

	// Sort by relevance
	sort.Slice(candidates, func(i, j int) bool {
		return si.calculateRelevance(candidates[i], query) > si.calculateRelevance(candidates[j], query)
	})

	return si.convertToSymbolInformation(candidates, limit)
}

// GetStatistics returns indexing statistics
func (si *SymbolIndex) GetStatistics() map[string]interface{} {
	si.mu.RLock()
	defer si.mu.RUnlock()

	totalSymbols := 0
	for _, symbols := range si.symbols {
		totalSymbols += len(symbols)
	}

	typeStats := make(map[string]int)
	for symbolType, symbols := range si.typeIndex {
		typeStats[string(symbolType)] = len(symbols)
	}

	return map[string]interface{}{
		"version":       si.version,
		"documents":     len(si.symbols),
		"total_symbols": totalSymbols,
		"by_type":       typeStats,
		"name_entries":  len(si.nameIndex),
		"containers":    len(si.containerIndex),
	}
}

// processSymbols converts AST symbols to indexed symbols with additional metadata
func (si *SymbolIndex) processSymbols(symbols []ast.Symbol, doc *document.Document) []IndexedSymbol {
	var indexed []IndexedSymbol

	// First pass: index top-level symbols
	for _, symbol := range symbols {
		indexedSymbol := IndexedSymbol{
			Symbol:        symbol,
			URI:           doc.URI,
			ContainerName: "",
			FullName:      symbol.Name,
		}
		indexed = append(indexed, indexedSymbol)
	}

	// Second pass: extract nested symbols (methods, fields, etc.)
	for _, symbol := range symbols {
		nested := si.extractNestedSymbols(symbol, doc, symbol.Name)
		indexed = append(indexed, nested...)
	}

	return indexed
}

// extractNestedSymbols recursively extracts nested symbols from containers
func (si *SymbolIndex) extractNestedSymbols(containerSymbol ast.Symbol, doc *document.Document, containerName string) []IndexedSymbol {
	var nested []IndexedSymbol

	if containerSymbol.Node == nil {
		return nested
	}

	switch containerSymbol.Type {
	case ast.NodeTypeService:
		nested = append(nested, si.extractServiceSymbols(containerSymbol, doc, containerName)...)
	case ast.NodeTypeScope:
		nested = append(nested, si.extractScopeSymbols(containerSymbol, doc, containerName)...)
	case ast.NodeTypeStruct, ast.NodeTypeException:
		nested = append(nested, si.extractStructSymbols(containerSymbol, doc, containerName)...)
	case ast.NodeTypeEnum:
		nested = append(nested, si.extractEnumSymbols(containerSymbol, doc, containerName)...)
	}

	return nested
}

// extractServiceSymbols extracts method symbols from a service
func (si *SymbolIndex) extractServiceSymbols(serviceSymbol ast.Symbol, doc *document.Document, containerName string) []IndexedSymbol {
	var methods []IndexedSymbol

	// Find service body and extract methods
	serviceBody := ast.FindNodeByType(serviceSymbol.Node, "service_body")
	if serviceBody == nil {
		return methods
	}

	childCount := serviceBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := serviceBody.Child(i)
		if child.Kind() == "function_definition" {
			methodSymbol := si.extractMethodSymbol(child, doc.Content, containerName, doc.URI)
			if methodSymbol != nil {
				methods = append(methods, *methodSymbol)
			}
		}
	}

	return methods
}

// extractScopeSymbols extracts event symbols from a scope
func (si *SymbolIndex) extractScopeSymbols(scopeSymbol ast.Symbol, doc *document.Document, containerName string) []IndexedSymbol {
	var events []IndexedSymbol

	// Find scope body and extract events
	scopeBody := ast.FindNodeByType(scopeSymbol.Node, "scope_body")
	if scopeBody == nil {
		return events
	}

	childCount := scopeBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := scopeBody.Child(i)
		eventSymbol := si.extractEventSymbol(child, doc.Content, containerName, doc.URI)
		if eventSymbol != nil {
			events = append(events, *eventSymbol)
		}
	}

	return events
}

// extractStructSymbols extracts field symbols from a struct
func (si *SymbolIndex) extractStructSymbols(structSymbol ast.Symbol, doc *document.Document, containerName string) []IndexedSymbol {
	var fields []IndexedSymbol

	// Find struct body and extract fields
	structBody := ast.FindNodeByType(structSymbol.Node, "struct_body")
	if structBody == nil {
		return fields
	}

	childCount := structBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := structBody.Child(i)
		if child.Kind() == "field" {
			fieldSymbol := si.extractFieldSymbol(child, doc.Content, containerName, doc.URI)
			if fieldSymbol != nil {
				fields = append(fields, *fieldSymbol)
			}
		}
	}

	return fields
}

// extractEnumSymbols extracts value symbols from an enum
func (si *SymbolIndex) extractEnumSymbols(enumSymbol ast.Symbol, doc *document.Document, containerName string) []IndexedSymbol {
	var values []IndexedSymbol

	// Find enum body and extract values
	enumBody := ast.FindNodeByType(enumSymbol.Node, "enum_body")
	if enumBody == nil {
		return values
	}

	childCount := enumBody.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := enumBody.Child(i)
		if child.Kind() == "enum_value" {
			valueSymbol := si.extractEnumValueSymbol(child, doc.Content, containerName, doc.URI)
			if valueSymbol != nil {
				values = append(values, *valueSymbol)
			}
		}
	}

	return values
}

// extractMethodSymbol creates a method symbol from a function definition node
func (si *SymbolIndex) extractMethodSymbol(methodNode *tree_sitter.Node, source []byte, containerName string, uri string) *IndexedSymbol {
	// Find method name
	var methodName string
	childCount := methodNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := methodNode.Child(i)
		if child.Kind() == "identifier" {
			methodName = ast.GetText(child, source)
			break
		}
	}

	if methodName == "" {
		return nil
	}

	return &IndexedSymbol{
		Symbol: ast.Symbol{
			Name:   methodName,
			Type:   ast.NodeTypeMethod,
			Line:   int(methodNode.StartPosition().Row),
			Column: int(methodNode.StartPosition().Column),
			Node:   methodNode,
		},
		URI:           uri,
		ContainerName: containerName,
		FullName:      containerName + "." + methodName,
	}
}

// extractEventSymbol creates an event symbol
func (si *SymbolIndex) extractEventSymbol(eventNode *tree_sitter.Node, source []byte, containerName string, uri string) *IndexedSymbol {
	eventText := ast.GetText(eventNode, source)
	if eventText == "" {
		return nil
	}

	// Parse "EventName: Type" format
	parts := strings.Split(strings.TrimSpace(eventText), ":")
	if len(parts) < 1 {
		return nil
	}

	eventName := strings.TrimSpace(parts[0])
	if eventName == "" {
		return nil
	}

	return &IndexedSymbol{
		Symbol: ast.Symbol{
			Name:   eventName,
			Type:   ast.NodeTypeEvent,
			Line:   int(eventNode.StartPosition().Row),
			Column: int(eventNode.StartPosition().Column),
			Node:   eventNode,
		},
		URI:           uri,
		ContainerName: containerName,
		FullName:      containerName + "." + eventName,
	}
}

// extractFieldSymbol creates a field symbol
func (si *SymbolIndex) extractFieldSymbol(fieldNode *tree_sitter.Node, source []byte, containerName string, uri string) *IndexedSymbol {
	// Find field name (last identifier in the field)
	var fieldName string
	childCount := fieldNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := fieldNode.Child(i)
		if child.Kind() == "identifier" {
			fieldName = ast.GetText(child, source)
		}
	}

	if fieldName == "" {
		return nil
	}

	return &IndexedSymbol{
		Symbol: ast.Symbol{
			Name:   fieldName,
			Type:   ast.NodeTypeField,
			Line:   int(fieldNode.StartPosition().Row),
			Column: int(fieldNode.StartPosition().Column),
			Node:   fieldNode,
		},
		URI:           uri,
		ContainerName: containerName,
		FullName:      containerName + "." + fieldName,
	}
}

// extractEnumValueSymbol creates an enum value symbol
func (si *SymbolIndex) extractEnumValueSymbol(valueNode *tree_sitter.Node, source []byte, containerName string, uri string) *IndexedSymbol {
	valueText := ast.GetText(valueNode, source)
	if valueText == "" {
		return nil
	}

	// Parse "VALUE = 1" format
	parts := strings.Split(strings.TrimSpace(valueText), "=")
	valueName := strings.TrimSpace(parts[0])
	if valueName == "" {
		return nil
	}

	return &IndexedSymbol{
		Symbol: ast.Symbol{
			Name:   valueName,
			Type:   ast.NodeTypeEnumValue,
			Line:   int(valueNode.StartPosition().Row),
			Column: int(valueNode.StartPosition().Column),
			Node:   valueNode,
		},
		URI:           uri,
		ContainerName: containerName,
		FullName:      containerName + "." + valueName,
	}
}

// removeDocumentSymbols removes all symbols for a document from all indices
func (si *SymbolIndex) removeDocumentSymbols(uri string) {
	if symbols, exists := si.symbols[uri]; exists {
		// Remove from name index
		for _, symbol := range symbols {
			si.removeFromNameIndex(symbol)
			si.removeFromTypeIndex(symbol)
			si.removeFromContainerIndex(symbol)
		}
		delete(si.symbols, uri)
	}
}

// addToIndices adds a symbol to all relevant indices
func (si *SymbolIndex) addToIndices(symbol IndexedSymbol) {
	// Name index (case-insensitive)
	lowerName := strings.ToLower(symbol.Name)
	si.nameIndex[lowerName] = append(si.nameIndex[lowerName], symbol)

	// Type index
	si.typeIndex[symbol.Type] = append(si.typeIndex[symbol.Type], symbol)

	// Container index
	if symbol.ContainerName != "" {
		lowerContainer := strings.ToLower(symbol.ContainerName)
		si.containerIndex[lowerContainer] = append(si.containerIndex[lowerContainer], symbol)
	}
}

// removeFromNameIndex removes a symbol from the name index
func (si *SymbolIndex) removeFromNameIndex(symbol IndexedSymbol) {
	lowerName := strings.ToLower(symbol.Name)
	if symbols, exists := si.nameIndex[lowerName]; exists {
		si.nameIndex[lowerName] = si.removeSymbolFromSlice(symbols, symbol)
		if len(si.nameIndex[lowerName]) == 0 {
			delete(si.nameIndex, lowerName)
		}
	}
}

// removeFromTypeIndex removes a symbol from the type index
func (si *SymbolIndex) removeFromTypeIndex(symbol IndexedSymbol) {
	if symbols, exists := si.typeIndex[symbol.Type]; exists {
		si.typeIndex[symbol.Type] = si.removeSymbolFromSlice(symbols, symbol)
		if len(si.typeIndex[symbol.Type]) == 0 {
			delete(si.typeIndex, symbol.Type)
		}
	}
}

// removeFromContainerIndex removes a symbol from the container index
func (si *SymbolIndex) removeFromContainerIndex(symbol IndexedSymbol) {
	if symbol.ContainerName != "" {
		lowerContainer := strings.ToLower(symbol.ContainerName)
		if symbols, exists := si.containerIndex[lowerContainer]; exists {
			si.containerIndex[lowerContainer] = si.removeSymbolFromSlice(symbols, symbol)
			if len(si.containerIndex[lowerContainer]) == 0 {
				delete(si.containerIndex, lowerContainer)
			}
		}
	}
}

// removeSymbolFromSlice removes a symbol from a slice
func (si *SymbolIndex) removeSymbolFromSlice(slice []IndexedSymbol, target IndexedSymbol) []IndexedSymbol {
	for i, symbol := range slice {
		if symbol.URI == target.URI &&
			symbol.Name == target.Name &&
			symbol.Line == target.Line &&
			symbol.Column == target.Column {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// findMatches finds symbols matching the query using multiple strategies
//
//nolint:gocognit // Symbol matching requires complex logic for fuzzy search and ranking
func (si *SymbolIndex) findMatches(query string) []IndexedSymbol {
	lowerQuery := strings.ToLower(query)
	var matches []IndexedSymbol
	seen := make(map[string]bool)

	// Strategy 1: Exact name match
	if symbols, exists := si.nameIndex[lowerQuery]; exists {
		for _, symbol := range symbols {
			key := si.symbolKey(symbol)
			if !seen[key] {
				matches = append(matches, symbol)
				seen[key] = true
			}
		}
	}

	// Strategy 2: Name prefix match
	for name, symbols := range si.nameIndex {
		if strings.HasPrefix(name, lowerQuery) && name != lowerQuery {
			for _, symbol := range symbols {
				key := si.symbolKey(symbol)
				if !seen[key] {
					matches = append(matches, symbol)
					seen[key] = true
				}
			}
		}
	}

	// Strategy 3: Name contains match
	for name, symbols := range si.nameIndex {
		if strings.Contains(name, lowerQuery) && !strings.HasPrefix(name, lowerQuery) {
			for _, symbol := range symbols {
				key := si.symbolKey(symbol)
				if !seen[key] {
					matches = append(matches, symbol)
					seen[key] = true
				}
			}
		}
	}

	// Strategy 4: Container name match
	for containerName, symbols := range si.containerIndex {
		if strings.Contains(containerName, lowerQuery) {
			for _, symbol := range symbols {
				key := si.symbolKey(symbol)
				if !seen[key] {
					matches = append(matches, symbol)
					seen[key] = true
				}
			}
		}
	}

	return matches
}

// filterByQuery filters symbols by query string
func (si *SymbolIndex) filterByQuery(symbols []IndexedSymbol, query string) []IndexedSymbol {
	if query == "" {
		return symbols
	}

	lowerQuery := strings.ToLower(query)
	var filtered []IndexedSymbol

	for _, symbol := range symbols {
		if si.symbolMatchesQuery(symbol, lowerQuery) {
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// symbolMatchesQuery checks if a symbol matches the query
func (si *SymbolIndex) symbolMatchesQuery(symbol IndexedSymbol, lowerQuery string) bool {
	lowerName := strings.ToLower(symbol.Name)
	lowerContainer := strings.ToLower(symbol.ContainerName)
	lowerFullName := strings.ToLower(symbol.FullName)

	return strings.Contains(lowerName, lowerQuery) ||
		strings.Contains(lowerContainer, lowerQuery) ||
		strings.Contains(lowerFullName, lowerQuery)
}

// calculateRelevance calculates search relevance score for a symbol
func (si *SymbolIndex) calculateRelevance(symbol IndexedSymbol, query string) int {
	if query == "" {
		return 1
	}

	lowerQuery := strings.ToLower(query)
	lowerName := strings.ToLower(symbol.Name)
	lowerContainer := strings.ToLower(symbol.ContainerName)

	score := 0

	// Exact match gets highest score (with case sensitivity bonus)
	if lowerName == lowerQuery {
		score += 1000
		// Case-sensitive exact match gets extra points
		if symbol.Name == query {
			score += 100
		}
	}

	// Prefix match gets high score
	if strings.HasPrefix(lowerName, lowerQuery) && lowerName != lowerQuery {
		score += 500
		// Case-sensitive prefix match gets extra points
		if strings.HasPrefix(symbol.Name, query) {
			score += 50
		}
	}

	// Contains match gets medium score
	if strings.Contains(lowerName, lowerQuery) && !strings.HasPrefix(lowerName, lowerQuery) {
		score += 250
	}

	// Container name match gets lower score
	if strings.Contains(lowerContainer, lowerQuery) {
		score += 100
	}

	// Prefer certain symbol types
	switch symbol.Type {
	case ast.NodeTypeService, ast.NodeTypeStruct, ast.NodeTypeEnum:
		score += 50
	case ast.NodeTypeMethod:
		score += 30
	case ast.NodeTypeConst, ast.NodeTypeTypedef:
		score += 20
	case ast.NodeTypeField, ast.NodeTypeEvent, ast.NodeTypeEnumValue:
		score += 10
	}

	return score
}

// getAllSymbols returns all symbols up to the limit
func (si *SymbolIndex) getAllSymbols(limit int) []protocol.SymbolInformation {
	var allSymbols []IndexedSymbol
	for _, symbols := range si.symbols {
		allSymbols = append(allSymbols, symbols...)
	}

	// Sort by type and name for consistent ordering
	sort.Slice(allSymbols, func(i, j int) bool {
		if allSymbols[i].Type != allSymbols[j].Type {
			return string(allSymbols[i].Type) < string(allSymbols[j].Type)
		}
		return allSymbols[i].Name < allSymbols[j].Name
	})

	return si.convertToSymbolInformation(allSymbols, limit)
}

// convertToSymbolInformation converts indexed symbols to LSP SymbolInformation
func (si *SymbolIndex) convertToSymbolInformation(symbols []IndexedSymbol, limit int) []protocol.SymbolInformation {
	var result []protocol.SymbolInformation

	count := 0
	for _, symbol := range symbols {
		if limit > 0 && count >= limit {
			break
		}

		symbolInfo := protocol.SymbolInformation{
			Name: symbol.Name,
			Kind: si.getSymbolKind(symbol.Type),
			Location: protocol.Location{
				URI: symbol.URI,
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

		// Add container information
		if symbol.ContainerName != "" {
			symbolInfo.ContainerName = &symbol.ContainerName
		}

		result = append(result, symbolInfo)
		count++
	}

	return result
}

// getSymbolKind converts AST node type to LSP symbol kind
func (si *SymbolIndex) getSymbolKind(nodeType ast.NodeType) protocol.SymbolKind {
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
	case ast.NodeTypeMethod:
		return protocol.SymbolKindMethod
	case ast.NodeTypeField:
		return protocol.SymbolKindField
	case ast.NodeTypeEvent:
		return protocol.SymbolKindEvent
	case ast.NodeTypeEnumValue:
		return protocol.SymbolKindEnumMember
	default:
		return protocol.SymbolKindVariable
	}
}

// symbolKey generates a unique key for a symbol
func (si *SymbolIndex) symbolKey(symbol IndexedSymbol) string {
	return symbol.URI + ":" + symbol.FullName + ":" + string(symbol.Type)
}

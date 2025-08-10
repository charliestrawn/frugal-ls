package document

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"frugal-ls/internal/parser"
	"frugal-ls/pkg/ast"
)

// DiagnosticsProvider is a minimal interface to avoid circular imports
type DiagnosticsProvider interface {
	ProvideDiagnostics(doc *Document) []protocol.Diagnostic
}

// Global diagnostics provider - initialized by the server
var globalDiagnosticsProvider DiagnosticsProvider

// SetDiagnosticsProvider sets the diagnostics provider (called from server initialization)
func SetDiagnosticsProvider(provider DiagnosticsProvider) {
	globalDiagnosticsProvider = provider
}

// Document represents an open document in the LSP server
type Document struct {
	URI     string
	Path    string
	Content []byte
	Version int32

	// Cached parsing results
	ParseResult *parser.ParseResult
	Symbols     []ast.Symbol

	// Mutex for thread-safe access
	mutex sync.RWMutex
}

// Manager handles document lifecycle and caching
type Manager struct {
	documents map[string]*Document
	parser    *parser.TreeSitterParser
	mutex     sync.RWMutex
}

// NewManager creates a new document manager
func NewManager() (*Manager, error) {
	p, err := parser.NewParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	return &Manager{
		documents: make(map[string]*Document),
		parser:    p,
	}, nil
}

// Close releases resources held by the manager
func (m *Manager) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Close all cached parse results
	for _, doc := range m.documents {
		doc.mutex.Lock()
		if doc.ParseResult != nil {
			doc.ParseResult.Close()
		}
		doc.mutex.Unlock()
	}

	if m.parser != nil {
		m.parser.Close()
	}
}

// DidOpen handles the textDocument/didOpen notification
func (m *Manager) DidOpen(params *protocol.DidOpenTextDocumentParams) (*Document, error) {
	uri := params.TextDocument.URI

	// Parse file path from URI
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %w", err)
	}

	path := parsedURI.Path
	content := []byte(params.TextDocument.Text)
	version := params.TextDocument.Version

	doc := &Document{
		URI:     uri,
		Path:    path,
		Content: content,
		Version: version,
	}

	// Parse the document
	if err := m.parseDocument(doc); err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	m.mutex.Lock()
	m.documents[uri] = doc
	m.mutex.Unlock()

	return doc, nil
}

// DidChange handles the textDocument/didChange notification
func (m *Manager) DidChange(params *protocol.DidChangeTextDocumentParams) (*Document, error) {
	uri := params.TextDocument.URI
	version := params.TextDocument.Version

	m.mutex.RLock()
	doc, exists := m.documents[uri]
	m.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	doc.mutex.Lock()
	defer doc.mutex.Unlock()

	// Apply content changes
	for _, change := range params.ContentChanges {
		// Cast the change to the proper type
		textChange, ok := change.(protocol.TextDocumentContentChangeEvent)
		if !ok {
			continue
		}

		if textChange.Range == nil {
			// Full document update
			doc.Content = []byte(textChange.Text)
		} else {
			// Incremental update - apply the change to the existing content
			if err := m.applyIncrementalChange(doc, &textChange); err != nil {
				// If incremental update fails, fallback to full content replacement
				// This shouldn't happen in normal cases, but provides safety
				doc.Content = []byte(textChange.Text)
			}
		}
	}

	doc.Version = version

	// Close old parse result
	if doc.ParseResult != nil {
		doc.ParseResult.Close()
		doc.ParseResult = nil
	}

	// Re-parse the document
	if err := m.parseDocument(doc); err != nil {
		return nil, fmt.Errorf("failed to re-parse document: %w", err)
	}

	return doc, nil
}

// DidClose handles the textDocument/didClose notification
func (m *Manager) DidClose(params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI

	m.mutex.Lock()
	defer m.mutex.Unlock()

	doc, exists := m.documents[uri]
	if !exists {
		return nil // Document wasn't tracked, nothing to do
	}

	// Clean up resources
	doc.mutex.Lock()
	if doc.ParseResult != nil {
		doc.ParseResult.Close()
	}
	doc.mutex.Unlock()

	delete(m.documents, uri)
	return nil
}

// GetDocument retrieves a document by URI
func (m *Manager) GetDocument(uri string) (*Document, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	doc, exists := m.documents[uri]
	return doc, exists
}

// GetAllDocuments returns a copy of all currently managed documents
func (m *Manager) GetAllDocuments() map[string]*Document {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to avoid concurrent access issues
	allDocuments := make(map[string]*Document, len(m.documents))
	for uri, doc := range m.documents {
		allDocuments[uri] = doc
	}

	return allDocuments
}

// applyIncrementalChange applies an incremental text change to a document
func (m *Manager) applyIncrementalChange(doc *Document, change *protocol.TextDocumentContentChangeEvent) error {
	if change.Range == nil {
		return fmt.Errorf("incremental change must have a range")
	}

	content := string(doc.Content)
	lines := strings.Split(content, "\n")

	startLine := int(change.Range.Start.Line)
	startChar := int(change.Range.Start.Character)
	endLine := int(change.Range.End.Line)
	endChar := int(change.Range.End.Character)

	// Validate range bounds
	if startLine < 0 || startLine >= len(lines) {
		return fmt.Errorf("start line %d out of bounds (0-%d)", startLine, len(lines)-1)
	}
	if endLine < 0 || endLine >= len(lines) {
		return fmt.Errorf("end line %d out of bounds (0-%d)", endLine, len(lines)-1)
	}
	if startChar < 0 || startChar > len(lines[startLine]) {
		return fmt.Errorf("start character %d out of bounds (0-%d)", startChar, len(lines[startLine]))
	}
	if endChar < 0 || endChar > len(lines[endLine]) {
		return fmt.Errorf("end character %d out of bounds (0-%d)", endChar, len(lines[endLine]))
	}

	var result strings.Builder

	// Add lines before the change
	for i := 0; i < startLine; i++ {
		result.WriteString(lines[i])
		result.WriteString("\n")
	}

	// Add the part of start line before the change
	result.WriteString(lines[startLine][:startChar])

	// Add the new text
	result.WriteString(change.Text)

	// Add the part of end line after the change
	result.WriteString(lines[endLine][endChar:])

	// Add lines after the change
	for i := endLine + 1; i < len(lines); i++ {
		result.WriteString("\n")
		result.WriteString(lines[i])
	}

	doc.Content = []byte(result.String())
	return nil
}

// parseDocument parses a document and updates its cached results
func (m *Manager) parseDocument(doc *Document) error {
	// Only parse .frugal files
	if !strings.HasSuffix(doc.Path, ".frugal") {
		return nil
	}

	result, err := m.parser.Parse(doc.Content)
	if err != nil {
		return err
	}

	// Extract symbols
	var symbols []ast.Symbol
	if result.GetRootNode() != nil {
		symbols = ast.ExtractSymbols(result.GetRootNode(), doc.Content)
	}

	doc.ParseResult = result
	doc.Symbols = symbols

	return nil
}

// GetDiagnostics provides comprehensive diagnostics including parse errors and semantic validation
func (d *Document) GetDiagnostics() []protocol.Diagnostic {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// If we have a comprehensive diagnostics provider, use it
	if globalDiagnosticsProvider != nil {
		return globalDiagnosticsProvider.ProvideDiagnostics(d)
	}

	// Fallback to basic parse error diagnostics
	return d.getBasicParseErrorDiagnostics()
}

// getBasicParseErrorDiagnostics provides basic parse error diagnostics as fallback
func (d *Document) getBasicParseErrorDiagnostics() []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	if d.ParseResult == nil {
		return diagnostics
	}

	for _, err := range d.ParseResult.Errors {
		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(err.Line),
					Character: uint32(err.Column),
				},
				End: protocol.Position{
					Line:      uint32(err.Line),
					Character: uint32(err.Column + 1), // Simple single-character range
				},
			},
			Severity: &[]protocol.DiagnosticSeverity{protocol.DiagnosticSeverityError}[0],
			Source:   &[]string{"frugal-ls"}[0],
			Message:  err.Message,
		}

		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics
}

// GetSymbols returns the cached symbols for the document
func (d *Document) GetSymbols() []ast.Symbol {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.Symbols
}

// IsValidFrugalFile checks if the document is a .frugal file
func (d *Document) IsValidFrugalFile() bool {
	return strings.HasSuffix(d.Path, ".frugal") ||
		strings.HasSuffix(filepath.Base(d.Path), ".frugal")
}

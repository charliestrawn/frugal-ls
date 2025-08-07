package lsp

import (
	"fmt"
	"log"
	"os"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	"frugal-lsp/internal/document"
)

const (
	LanguageServerName = "frugal-lsp"
	LanguageServerVersion = "0.1.0"
)

// Server represents the Frugal LSP server
type Server struct {
	server      *server.Server
	docManager  *document.Manager
	logger      *log.Logger
}

// NewServer creates a new Frugal LSP server
func NewServer() (*Server, error) {
	// Create document manager
	docManager, err := document.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create document manager: %w", err)
	}

	// Create logger
	logger := log.New(os.Stderr, "[frugal-lsp] ", log.LstdFlags)

	// Create the server
	lspServer := &Server{
		docManager: docManager,
		logger:     logger,
	}

	// Set up GLSP server
	handler := protocol.Handler{
		Initialize:             lspServer.initialize,
		Initialized:            lspServer.initialized,
		Shutdown:               lspServer.shutdown,
		TextDocumentDidOpen:    lspServer.textDocumentDidOpen,
		TextDocumentDidChange:  lspServer.textDocumentDidChange,
		TextDocumentDidClose:   lspServer.textDocumentDidClose,
		TextDocumentDidSave:    lspServer.textDocumentDidSave,
	}

	serverInstance := server.NewServer(&handler, LanguageServerName, false)
	lspServer.server = serverInstance

	return lspServer, nil
}

// Run starts the LSP server
func (s *Server) Run() error {
	s.logger.Println("Starting Frugal LSP server...")
	
	defer func() {
		s.logger.Println("Shutting down Frugal LSP server...")
		if s.docManager != nil {
			s.docManager.Close()
		}
	}()

	return s.server.RunStdio()
}

// initialize handles the initialize request
func (s *Server) initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	s.logger.Printf("Initialize request from client: %s", params.ClientInfo.Name)
	
	capabilities := s.getServerCapabilities()
	
	version := LanguageServerVersion
	serverInfo := protocol.InitializeResultServerInfo{
		Name:    LanguageServerName,
		Version: &version,
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo:   &serverInfo,
	}, nil
}

// initialized handles the initialized notification
func (s *Server) initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	s.logger.Println("Client initialized, server ready")
	return nil
}

// shutdown handles the shutdown request
func (s *Server) shutdown(context *glsp.Context) error {
	s.logger.Println("Shutdown request received")
	return nil
}

// textDocumentDidOpen handles textDocument/didOpen notifications
func (s *Server) textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	s.logger.Printf("Document opened: %s", params.TextDocument.URI)
	
	doc, err := s.docManager.DidOpen(params)
	if err != nil {
		s.logger.Printf("Error opening document: %v", err)
		return err
	}

	// Send diagnostics if this is a Frugal file
	if doc.IsValidFrugalFile() {
		s.publishDiagnostics(context, doc)
	}

	return nil
}

// textDocumentDidChange handles textDocument/didChange notifications
func (s *Server) textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	s.logger.Printf("Document changed: %s (version %d)", params.TextDocument.URI, params.TextDocument.Version)
	
	doc, err := s.docManager.DidChange(params)
	if err != nil {
		s.logger.Printf("Error processing document changes: %v", err)
		return err
	}

	// Send updated diagnostics if this is a Frugal file
	if doc.IsValidFrugalFile() {
		s.publishDiagnostics(context, doc)
	}

	return nil
}

// textDocumentDidClose handles textDocument/didClose notifications
func (s *Server) textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.logger.Printf("Document closed: %s", params.TextDocument.URI)
	
	err := s.docManager.DidClose(params)
	if err != nil {
		s.logger.Printf("Error closing document: %v", err)
	}

	// Clear diagnostics for closed document
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []protocol.Diagnostic{},
	})

	return err
}

// textDocumentDidSave handles textDocument/didSave notifications
func (s *Server) textDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	s.logger.Printf("Document saved: %s", params.TextDocument.URI)
	
	// For now, we don't need special handling for save events
	// The document content is already up-to-date from didChange events
	
	return nil
}

// publishDiagnostics sends diagnostics to the client
func (s *Server) publishDiagnostics(context *glsp.Context, doc *document.Document) {
	diagnostics := doc.GetDiagnostics()
	
	s.logger.Printf("Publishing %d diagnostics for %s", len(diagnostics), doc.URI)
	
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         doc.URI,
		Diagnostics: diagnostics,
	})
}

// getServerCapabilities returns the server's capabilities
func (s *Server) getServerCapabilities() protocol.ServerCapabilities {
	return protocol.ServerCapabilities{
		// Document synchronization
		TextDocumentSync: protocol.TextDocumentSyncOptions{
			OpenClose: &[]bool{true}[0],
			Change:    &[]protocol.TextDocumentSyncKind{protocol.TextDocumentSyncKindFull}[0],
			Save: &protocol.SaveOptions{
				IncludeText: &[]bool{false}[0],
			},
		},
		
		// Future capabilities to be implemented in Phase 3
		HoverProvider:      &[]bool{false}[0],
		CompletionProvider: nil,
		DocumentSymbolProvider: &[]bool{false}[0],
		DefinitionProvider: &[]bool{false}[0],
		WorkspaceSymbolProvider: &[]bool{false}[0],
	}
}
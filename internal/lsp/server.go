package lsp

import (
	"fmt"
	"log"
	"os"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	"frugal-ls/internal/document"
	"frugal-ls/internal/features"
	"frugal-ls/internal/workspace"
)

const (
	// LanguageServerName is the identifier for the Frugal language server
	LanguageServerName = "frugal-ls"
	// LanguageServerVersion is the current version of the Frugal language server
	LanguageServerVersion = "0.1.0"
)

// Server represents the Frugal LSP server
type Server struct {
	server     *server.Server
	docManager *document.Manager
	logger     *log.Logger

	// Workspace management
	includeResolver *workspace.IncludeResolver
	symbolIndex     *workspace.SymbolIndex
	workspaceRoots  []string

	// Language feature providers
	completionProvider        *features.CompletionProvider
	hoverProvider             *features.HoverProvider
	documentSymbolProvider    *features.DocumentSymbolProvider
	definitionProvider        *features.DefinitionProvider
	referencesProvider        *features.ReferencesProvider
	documentHighlightProvider *features.DocumentHighlightProvider
	codeActionProvider        *features.CodeActionProvider
	formattingProvider        *features.FormattingProvider
	semanticTokensProvider    *features.SemanticTokensProvider
	renameProvider            *features.RenameProvider
}

// NewServer creates a new Frugal LSP server
func NewServer() (*Server, error) {
	// Create document manager
	docManager, err := document.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create document manager: %w", err)
	}

	// Create logger
	logger := log.New(os.Stderr, "[frugal-ls] ", log.LstdFlags)

	// Initialize workspace roots (will be updated from InitializeParams)
	workspaceRoots := []string{"."}
	includeResolver := workspace.NewIncludeResolver(workspaceRoots)
	symbolIndex := workspace.NewSymbolIndex()

	// Initialize diagnostics provider
	diagnosticsProvider := features.NewDiagnosticsProvider()
	document.SetDiagnosticsProvider(diagnosticsProvider)

	// Create the server with language feature providers
	lspServer := &Server{
		docManager:                docManager,
		logger:                    logger,
		includeResolver:           includeResolver,
		symbolIndex:               symbolIndex,
		workspaceRoots:            workspaceRoots,
		completionProvider:        features.NewCompletionProvider(),
		hoverProvider:             features.NewHoverProvider(),
		documentSymbolProvider:    features.NewDocumentSymbolProvider(),
		definitionProvider:        features.NewDefinitionProvider(),
		referencesProvider:        features.NewReferencesProvider(),
		documentHighlightProvider: features.NewDocumentHighlightProvider(),
		codeActionProvider:        features.NewCodeActionProvider(),
		formattingProvider:        features.NewFormattingProvider(),
		semanticTokensProvider:    features.NewSemanticTokensProvider(),
		renameProvider:            features.NewRenameProvider(),
	}

	// Set up GLSP server
	handler := protocol.Handler{
		Initialize:                      lspServer.initialize,
		Initialized:                     lspServer.initialized,
		Shutdown:                        lspServer.shutdown,
		TextDocumentDidOpen:             lspServer.textDocumentDidOpen,
		TextDocumentDidChange:           lspServer.textDocumentDidChange,
		TextDocumentDidClose:            lspServer.textDocumentDidClose,
		TextDocumentDidSave:             lspServer.textDocumentDidSave,
		TextDocumentCompletion:          lspServer.textDocumentCompletion,
		TextDocumentHover:               lspServer.textDocumentHover,
		TextDocumentDocumentSymbol:      lspServer.textDocumentDocumentSymbol,
		TextDocumentDefinition:          lspServer.textDocumentDefinition,
		TextDocumentReferences:          lspServer.textDocumentReferences,
		TextDocumentDocumentHighlight:   lspServer.textDocumentDocumentHighlight,
		TextDocumentCodeAction:          lspServer.textDocumentCodeAction,
		TextDocumentFormatting:          lspServer.textDocumentFormatting,
		TextDocumentRangeFormatting:     lspServer.textDocumentRangeFormatting,
		TextDocumentSemanticTokensFull:  lspServer.textDocumentSemanticTokensFull,
		TextDocumentSemanticTokensRange: lspServer.textDocumentSemanticTokensRange,
		TextDocumentPrepareRename:       lspServer.textDocumentPrepareRename,
		TextDocumentRename:              lspServer.textDocumentRename,
		WorkspaceSymbol:                 lspServer.workspaceSymbol,
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

	// Update workspace roots from initialize params
	if params.WorkspaceFolders != nil && len(params.WorkspaceFolders) > 0 {
		s.workspaceRoots = nil
		for _, folder := range params.WorkspaceFolders {
			s.workspaceRoots = append(s.workspaceRoots, folder.URI)
		}
		s.includeResolver = workspace.NewIncludeResolver(s.workspaceRoots)
	} else if params.RootURI != nil {
		s.workspaceRoots = []string{*params.RootURI}
		s.includeResolver = workspace.NewIncludeResolver(s.workspaceRoots)
	}

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
		// Update include dependencies
		if err := s.includeResolver.UpdateDocument(doc); err != nil {
			s.logger.Printf("Error updating document dependencies: %v", err)
		}

		// Update symbol index
		s.symbolIndex.UpdateDocument(doc)

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
		// Update include dependencies
		if err := s.includeResolver.UpdateDocument(doc); err != nil {
			s.logger.Printf("Error updating document dependencies: %v", err)
		}

		// Update symbol index
		s.symbolIndex.UpdateDocument(doc)

		s.publishDiagnostics(context, doc)

		// Request semantic token refresh to update highlighting
		if err := s.refreshSemanticTokens(context); err != nil {
			s.logger.Printf("Error refreshing semantic tokens: %v", err)
		}
	}

	return nil
}

// textDocumentDidClose handles textDocument/didClose notifications
func (s *Server) textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.logger.Printf("Document closed: %s", params.TextDocument.URI)

	// Remove from include resolver and symbol index
	s.includeResolver.RemoveDocument(params.TextDocument.URI)
	s.symbolIndex.RemoveDocument(params.TextDocument.URI)

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
			Change:    &[]protocol.TextDocumentSyncKind{protocol.TextDocumentSyncKindIncremental}[0],
			Save: &protocol.SaveOptions{
				IncludeText: &[]bool{false}[0],
			},
		},

		// Language features
		HoverProvider: &[]bool{true}[0],
		CompletionProvider: &protocol.CompletionOptions{
			TriggerCharacters: []string{".", ":", " "},
		},
		DocumentSymbolProvider:    &[]bool{true}[0],
		DefinitionProvider:        &[]bool{true}[0],
		ReferencesProvider:        &[]bool{true}[0],
		DocumentHighlightProvider: &[]bool{true}[0],
		WorkspaceSymbolProvider:   &[]bool{true}[0],
		CodeActionProvider: &protocol.CodeActionOptions{
			CodeActionKinds: []protocol.CodeActionKind{
				protocol.CodeActionKindQuickFix,
				protocol.CodeActionKindRefactor,
				protocol.CodeActionKindSource,
				protocol.CodeActionKindSourceOrganizeImports,
			},
		},
		DocumentFormattingProvider:      &[]bool{true}[0],
		DocumentRangeFormattingProvider: &[]bool{true}[0],
		RenameProvider: &protocol.RenameOptions{
			PrepareProvider: &[]bool{true}[0],
		},
		SemanticTokensProvider: &protocol.SemanticTokensOptions{
			Legend: s.semanticTokensProvider.GetLegend(),
			Full:   &[]bool{true}[0],
			Range:  &[]bool{true}[0],
		},
	}
}

// textDocumentCompletion handles completion requests
func (s *Server) textDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	completions, err := s.completionProvider.ProvideCompletion(doc, params.Position)
	if err != nil {
		s.logger.Printf("Error providing completions: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d completions for %s", len(completions), params.TextDocument.URI)
	return completions, nil
}

// textDocumentHover handles hover requests
func (s *Server) textDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	hover, err := s.hoverProvider.ProvideHover(doc, params.Position)
	if err != nil {
		s.logger.Printf("Error providing hover: %v", err)
		return nil, err
	}

	if hover != nil {
		s.logger.Printf("Providing hover for %s", params.TextDocument.URI)
	}
	return hover, nil
}

// textDocumentDocumentSymbol handles document symbol requests
func (s *Server) textDocumentDocumentSymbol(context *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	symbols, err := s.documentSymbolProvider.ProvideDocumentSymbols(doc)
	if err != nil {
		s.logger.Printf("Error providing document symbols: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d document symbols for %s", len(symbols), params.TextDocument.URI)
	return symbols, nil
}

// textDocumentDefinition handles go-to-definition requests
func (s *Server) textDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	// Get all documents for cross-file navigation
	allDocuments := s.getAllDocuments()

	locations, err := s.definitionProvider.ProvideDefinition(doc, params.Position, allDocuments)
	if err != nil {
		s.logger.Printf("Error providing definition: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d definition locations for %s", len(locations), params.TextDocument.URI)
	return locations, nil
}

// textDocumentReferences handles find references requests
func (s *Server) textDocumentReferences(context *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	// Get all documents for cross-file reference search
	allDocuments := s.getAllDocuments()

	locations, err := s.referencesProvider.ProvideReferences(doc, params.Position, params.Context.IncludeDeclaration, allDocuments)
	if err != nil {
		s.logger.Printf("Error providing references: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d reference locations for %s", len(locations), params.TextDocument.URI)
	return locations, nil
}

// textDocumentDocumentHighlight handles document highlight requests
func (s *Server) textDocumentDocumentHighlight(context *glsp.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	highlights, err := s.documentHighlightProvider.ProvideDocumentHighlight(doc, params.Position)
	if err != nil {
		s.logger.Printf("Error providing document highlights: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d document highlights for %s", len(highlights), params.TextDocument.URI)
	return highlights, nil
}

// workspaceSymbol handles workspace symbol search requests
func (s *Server) workspaceSymbol(context *glsp.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	// Use the indexed search for better performance
	const maxResults = 100
	symbols := s.symbolIndex.Search(params.Query, maxResults)

	s.logger.Printf("Providing %d indexed workspace symbols for query '%s'", len(symbols), params.Query)
	return symbols, nil
}

// textDocumentCodeAction handles code action requests
func (s *Server) textDocumentCodeAction(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	actions, err := s.codeActionProvider.ProvideCodeActions(doc, params.Range, params.Context)
	if err != nil {
		s.logger.Printf("Error providing code actions: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d code actions for %s", len(actions), params.TextDocument.URI)
	return actions, nil
}

// textDocumentFormatting handles document formatting requests
func (s *Server) textDocumentFormatting(context *glsp.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	edits, err := s.formattingProvider.ProvideDocumentFormatting(doc, params.Options)
	if err != nil {
		s.logger.Printf("Error providing document formatting: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d formatting edits for %s", len(edits), params.TextDocument.URI)
	return edits, nil
}

// textDocumentRangeFormatting handles range formatting requests
func (s *Server) textDocumentRangeFormatting(context *glsp.Context, params *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	edits, err := s.formattingProvider.ProvideDocumentRangeFormatting(doc, params.Range, params.Options)
	if err != nil {
		s.logger.Printf("Error providing range formatting: %v", err)
		return nil, err
	}

	s.logger.Printf("Providing %d range formatting edits for %s", len(edits), params.TextDocument.URI)
	return edits, nil
}

// textDocumentSemanticTokensFull handles full document semantic tokens requests
func (s *Server) textDocumentSemanticTokensFull(context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	s.logger.Printf("Providing semantic tokens for %s", params.TextDocument.URI)

	tokens, err := s.semanticTokensProvider.ProvideSemanticTokens(doc)
	if err != nil {
		s.logger.Printf("Semantic tokens error for %s: %v", params.TextDocument.URI, err)
		return nil, err
	}

	s.logger.Printf("Provided %d semantic token data points for %s", len(tokens.Data), params.TextDocument.URI)
	return tokens, nil
}

// refreshSemanticTokens sends a workspace/semanticTokens/refresh request to the client
func (s *Server) refreshSemanticTokens(context *glsp.Context) error {
	// Send a workspace/semanticTokens/refresh notification to the client
	// This tells VS Code to re-request semantic tokens for all open documents
	context.Notify(protocol.MethodWorkspaceSemanticTokensRefresh, nil)
	return nil
}

// textDocumentSemanticTokensRange handles range semantic tokens requests
func (s *Server) textDocumentSemanticTokensRange(context *glsp.Context, params *protocol.SemanticTokensRangeParams) (any, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	s.logger.Printf("Providing semantic tokens range for %s", params.TextDocument.URI)

	tokens, err := s.semanticTokensProvider.ProvideSemanticTokensRange(doc, params.Range)
	if err != nil {
		s.logger.Printf("Semantic tokens range error for %s: %v", params.TextDocument.URI, err)
		return nil, err
	}

	return tokens, nil
}

// textDocumentPrepareRename handles prepare rename requests
func (s *Server) textDocumentPrepareRename(context *glsp.Context, params *protocol.PrepareRenameParams) (any, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	s.logger.Printf("Preparing rename for %s", params.TextDocument.URI)

	rangeResult, err := s.renameProvider.PrepareRename(doc, params.Position)
	if err != nil {
		s.logger.Printf("Error preparing rename: %v", err)
		return nil, err
	}

	return rangeResult, nil
}

// textDocumentRename handles rename requests
func (s *Server) textDocumentRename(context *glsp.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	doc, exists := s.docManager.GetDocument(params.TextDocument.URI)
	if !exists || !doc.IsValidFrugalFile() {
		return nil, nil
	}

	s.logger.Printf("Renaming symbol to '%s' in %s", params.NewName, params.TextDocument.URI)

	// Get all documents for cross-file rename
	allDocuments := s.getAllDocuments()

	workspaceEdit, err := s.renameProvider.Rename(doc, params.Position, params.NewName, allDocuments)
	if err != nil {
		s.logger.Printf("Error performing rename: %v", err)
		return nil, err
	}

	if workspaceEdit != nil {
		changeCount := 0
		for _, changes := range workspaceEdit.Changes {
			changeCount += len(changes)
		}
		s.logger.Printf("Rename successful: %d changes across %d files", changeCount, len(workspaceEdit.Changes))
	}

	return workspaceEdit, nil
}

// getAllDocuments returns all currently managed documents
func (s *Server) getAllDocuments() map[string]*document.Document {
	return s.docManager.GetAllDocuments()
}

package lsp

import (
	"context"
	"encoding/json"

	"github.com/aireilly/mdita-lsp/internal/codeaction"
	"github.com/aireilly/mdita-lsp/internal/codelens"
	"github.com/aireilly/mdita-lsp/internal/completion"
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/definition"
	"github.com/aireilly/mdita-lsp/internal/diagnostic"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/docsymbols"
	"github.com/aireilly/mdita-lsp/internal/hover"
	"github.com/aireilly/mdita-lsp/internal/references"
	"github.com/aireilly/mdita-lsp/internal/rename"
	"github.com/aireilly/mdita-lsp/internal/semantic"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Server struct {
	workspace *workspace.Workspace
	graph     *symbols.Graph
	notify    func(method string, params interface{})
}

func NewServer() *Server {
	return &Server{
		workspace: workspace.New(),
		graph:     symbols.NewGraph(),
		notify:    func(string, interface{}) {},
	}
}

func (s *Server) SetNotify(fn func(method string, params interface{})) {
	s.notify = fn
}

type InitializeParams struct {
	RootURI      string          `json:"rootUri"`
	Capabilities json.RawMessage `json:"capabilities"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	TextDocumentSync        int                    `json:"textDocumentSync"`
	CompletionProvider      *CompletionOptions     `json:"completionProvider,omitempty"`
	DefinitionProvider      bool                   `json:"definitionProvider"`
	HoverProvider           bool                   `json:"hoverProvider"`
	ReferencesProvider      bool                   `json:"referencesProvider"`
	RenameProvider          bool                   `json:"renameProvider"`
	CodeActionProvider      bool                   `json:"codeActionProvider"`
	CodeLensProvider        *CodeLensOptions        `json:"codeLensProvider,omitempty"`
	DocumentSymbolProvider  bool                   `json:"documentSymbolProvider"`
	WorkspaceSymbolProvider bool                   `json:"workspaceSymbolProvider"`
	SemanticTokensProvider  *SemanticTokensOptions `json:"semanticTokensProvider,omitempty"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type SemanticTokensOptions struct {
	Full   bool                  `json:"full"`
	Legend SemanticTokensLegend `json:"legend"`
}

type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type TextDocumentItem struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
	Text    string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     document.Position      `json:"position"`
}

type DidOpenParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeParams struct {
	TextDocument struct {
		URI     string `json:"uri"`
		Version int    `json:"version"`
	} `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type DidCloseParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type CompletionItemResult struct {
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
	Kind       int    `json:"kind"`
}

type LocationResult struct {
	URI   string         `json:"uri"`
	Range document.Range `json:"range"`
}

type HoverResult struct {
	Contents string         `json:"contents"`
	Range    document.Range `json:"range,omitempty"`
}

type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     document.Position      `json:"position"`
	NewName      string                 `json:"newName"`
}

type DiagnosticParams struct {
	URI         string             `json:"uri"`
	Diagnostics []DiagnosticResult `json:"diagnostics"`
}

type DiagnosticResult struct {
	Range    document.Range `json:"range"`
	Severity int            `json:"severity"`
	Code     string         `json:"code"`
	Source   string         `json:"source"`
	Message  string         `json:"message"`
}

func (s *Server) handleInitialize(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params InitializeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	if params.RootURI != "" {
		cfg := config.Default()
		folder := workspace.NewFolder(params.RootURI, cfg)
		folder.ScanFiles()
		s.workspace.AddFolder(folder)

		for _, doc := range folder.AllDocs() {
			s.graph.AddDefs(doc.URI, doc.Defs())
			s.graph.AddRefs(doc.URI, doc.Refs())
		}
	}

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: 1,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"[", "#", "("},
			},
			DefinitionProvider:      true,
			HoverProvider:           true,
			ReferencesProvider:      true,
			RenameProvider:          true,
			CodeActionProvider:      true,
			CodeLensProvider:        &CodeLensOptions{},
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			SemanticTokensProvider: &SemanticTokensOptions{
				Full: true,
				Legend: SemanticTokensLegend{
					TokenTypes:     semantic.TokenTypes,
					TokenModifiers: []string{},
				},
			},
		},
	}, nil
}

func (s *Server) handleDidOpen(_ context.Context, rawParams json.RawMessage) error {
	var params DidOpenParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	doc := document.New(params.TextDocument.URI, params.TextDocument.Version, params.TextDocument.Text)

	folder := s.workspace.FolderForURI(doc.URI)
	if folder == nil {
		cfg := config.Default()
		folder = workspace.NewFolder(parentURI(doc.URI), cfg)
		s.workspace.AddFolder(folder)
	}

	folder.AddDoc(doc)
	s.graph.AddDefs(doc.URI, doc.Defs())
	s.graph.AddRefs(doc.URI, doc.Refs())
	s.publishDiagnostics(doc, folder)

	return nil
}

func (s *Server) handleDidChange(_ context.Context, rawParams json.RawMessage) error {
	var params DidChangeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil
	}

	if len(params.ContentChanges) > 0 {
		newDoc := doc.ApplyChange(params.TextDocument.Version, params.ContentChanges[0].Text)
		folder.AddDoc(newDoc)
		s.graph.AddDefs(newDoc.URI, newDoc.Defs())
		s.graph.AddRefs(newDoc.URI, newDoc.Refs())
		s.publishDiagnostics(newDoc, folder)
	}
	return nil
}

func (s *Server) handleDidClose(_ context.Context, rawParams json.RawMessage) error {
	var params DidCloseParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleCompletion(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return []CompletionItemResult{}, nil
	}

	items := completion.Complete(doc, params.Position, folder, s.graph)
	var results []CompletionItemResult
	for _, item := range items {
		results = append(results, CompletionItemResult{
			Label:      item.Label,
			Detail:     item.Detail,
			InsertText: item.InsertText,
			Kind:       item.Kind,
		})
	}
	return results, nil
}

func (s *Server) handleDefinition(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	locs := definition.GotoDef(doc, params.Position, folder, s.graph)
	var results []LocationResult
	for _, loc := range locs {
		results = append(results, LocationResult{URI: loc.URI, Range: loc.Range})
	}
	return results, nil
}

func (s *Server) handleHover(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	content := hover.GetHover(doc, params.Position, folder, s.graph)
	if content == "" {
		return nil, nil
	}
	return HoverResult{Contents: content}, nil
}

func (s *Server) handleReferences(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	locs := references.FindRefs(doc, params.Position, folder, s.graph)
	var results []LocationResult
	for _, loc := range locs {
		results = append(results, LocationResult{URI: loc.URI, Range: loc.Range})
	}
	return results, nil
}

func (s *Server) handleRename(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params RenameParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	edits := rename.DoRename(doc, params.Position, params.NewName, folder, s.graph)
	result := make(map[string][]map[string]interface{})
	for _, edit := range edits {
		result[edit.URI] = append(result[edit.URI], map[string]interface{}{
			"range":   edit.Range,
			"newText": edit.NewText,
		})
	}
	return map[string]interface{}{"changes": result}, nil
}

func (s *Server) handleCodeAction(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Range        document.Range         `json:"range"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	actions := codeaction.GetActions(doc, params.Range, folder)
	var results []map[string]interface{}
	for _, a := range actions {
		results = append(results, map[string]interface{}{
			"title": a.Title,
			"kind":  a.Kind,
		})
	}
	return results, nil
}

func (s *Server) handleCodeLens(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	lenses := codelens.GetLenses(doc, s.graph)
	var results []map[string]interface{}
	for _, l := range lenses {
		results = append(results, map[string]interface{}{
			"range": l.Range,
			"command": map[string]interface{}{
				"title":   l.Title,
				"command": l.Command,
			},
		})
	}
	return results, nil
}

func (s *Server) handleDocumentSymbol(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	syms := docsymbols.GetSymbols(doc)
	return syms, nil
}

func (s *Server) handleSemanticTokensFull(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	data := semantic.Encode(doc)
	return map[string]interface{}{"data": data}, nil
}

func (s *Server) publishDiagnostics(doc *document.Document, folder *workspace.Folder) {
	diags := diagnostic.Check(doc, folder)
	var results []DiagnosticResult
	for _, d := range diags {
		results = append(results, DiagnosticResult{
			Range:    d.Range,
			Severity: int(d.Severity),
			Code:     d.Code,
			Source:   d.Source,
			Message:  d.Message,
		})
	}
	s.notify("textDocument/publishDiagnostics", DiagnosticParams{
		URI:         doc.URI,
		Diagnostics: results,
	})
}

func parentURI(uri string) string {
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == '/' {
			return uri[:i]
		}
	}
	return uri
}

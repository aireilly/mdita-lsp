package lsp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/aireilly/mdita-lsp/internal/codeaction"
	"github.com/aireilly/mdita-lsp/internal/codelens"
	"github.com/aireilly/mdita-lsp/internal/completion"
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/definition"
	"github.com/aireilly/mdita-lsp/internal/diagnostic"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/filerename"
	"github.com/aireilly/mdita-lsp/internal/docsymbols"
	"github.com/aireilly/mdita-lsp/internal/folding"
	"github.com/aireilly/mdita-lsp/internal/formatting"
	"github.com/aireilly/mdita-lsp/internal/hover"
	"github.com/aireilly/mdita-lsp/internal/inlayhint"
	"github.com/aireilly/mdita-lsp/internal/linkededit"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/references"
	"github.com/aireilly/mdita-lsp/internal/rename"
	"github.com/aireilly/mdita-lsp/internal/selection"
	"github.com/aireilly/mdita-lsp/internal/semantic"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Server struct {
	workspace  *workspace.Workspace
	graph      *symbols.Graph
	notify     func(method string, params interface{})
	diagBounce *debouncer
}

func NewServer() *Server {
	return &Server{
		workspace:  workspace.New(),
		graph:      symbols.NewGraph(),
		notify:     func(string, interface{}) {},
		diagBounce: newDebouncer(200 * time.Millisecond),
	}
}

func (s *Server) SetNotify(fn func(method string, params interface{})) {
	s.notify = fn
}

type InitializeParams struct {
	RootURI          string            `json:"rootUri"`
	Capabilities     json.RawMessage   `json:"capabilities"`
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
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
	RenameProvider          *RenameOptions         `json:"renameProvider,omitempty"`
	CodeActionProvider      bool                   `json:"codeActionProvider"`
	CodeLensProvider        *CodeLensOptions       `json:"codeLensProvider,omitempty"`
	DocumentLinkProvider    bool                   `json:"documentLinkProvider"`
	FoldingRangeProvider    bool                   `json:"foldingRangeProvider"`
	DocumentSymbolProvider  bool                   `json:"documentSymbolProvider"`
	WorkspaceSymbolProvider bool                   `json:"workspaceSymbolProvider"`
	SelectionRangeProvider       bool                   `json:"selectionRangeProvider"`
	LinkedEditingRangeProvider   bool                   `json:"linkedEditingRangeProvider"`
	DocumentFormattingProvider   bool                   `json:"documentFormattingProvider"`
	InlayHintProvider            bool                   `json:"inlayHintProvider"`
	SemanticTokensProvider       *SemanticTokensOptions `json:"semanticTokensProvider,omitempty"`
	Workspace               *WorkspaceCapabilities `json:"workspace,omitempty"`
}

type WorkspaceCapabilities struct {
	FileOperations *FileOperationCapabilities `json:"fileOperations,omitempty"`
}

type FileOperationCapabilities struct {
	DidCreate  *FileOperationRegistration `json:"didCreate,omitempty"`
	DidDelete  *FileOperationRegistration `json:"didDelete,omitempty"`
	WillRename *FileOperationRegistration `json:"willRename,omitempty"`
}

type FileOperationRegistration struct {
	Filters []FileOperationFilter `json:"filters"`
}

type FileOperationFilter struct {
	Pattern FileOperationPattern `json:"pattern"`
}

type FileOperationPattern struct {
	Glob string `json:"glob"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type RenameOptions struct {
	PrepareProvider bool `json:"prepareProvider"`
}

type SemanticTokensOptions struct {
	Full   bool                 `json:"full"`
	Range  bool                 `json:"range"`
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
	ContentChanges []ContentChange `json:"contentChanges"`
}

type ContentChange struct {
	Range *document.Range `json:"range,omitempty"`
	Text  string          `json:"text"`
}

type DidCloseParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type CompletionItemResult struct {
	Label         string            `json:"label"`
	Detail        string            `json:"detail,omitempty"`
	InsertText    string            `json:"insertText,omitempty"`
	Kind          int               `json:"kind"`
	Documentation *MarkupContent    `json:"documentation,omitempty"`
	Data          map[string]string `json:"data,omitempty"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
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

type TextEditResult struct {
	Range   document.Range `json:"range"`
	NewText string         `json:"newText"`
}

type WorkspaceEditResult struct {
	Changes map[string][]TextEditResult `json:"changes"`
}

func (s *Server) handleInitialize(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params InitializeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	roots := params.WorkspaceFolders
	if len(roots) == 0 && params.RootURI != "" {
		roots = []WorkspaceFolder{{URI: params.RootURI}}
	}
	for _, wf := range roots {
		s.addWorkspaceFolder(wf.URI)
	}

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: 2,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"[", "#", "("},
				ResolveProvider:   true,
			},
			DefinitionProvider:      true,
			HoverProvider:           true,
			ReferencesProvider:      true,
			RenameProvider:          &RenameOptions{PrepareProvider: true},
			CodeActionProvider:      true,
			CodeLensProvider:        &CodeLensOptions{},
			DocumentLinkProvider:    true,
			FoldingRangeProvider:    true,
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			SelectionRangeProvider:     true,
			LinkedEditingRangeProvider:  true,
			DocumentFormattingProvider: true,
			InlayHintProvider:          true,
			SemanticTokensProvider: &SemanticTokensOptions{
				Full:  true,
				Range: true,
				Legend: SemanticTokensLegend{
					TokenTypes:     semantic.TokenTypes,
					TokenModifiers: []string{},
				},
			},
			Workspace: &WorkspaceCapabilities{
				FileOperations: &FileOperationCapabilities{
					DidCreate: &FileOperationRegistration{
						Filters: []FileOperationFilter{
							{Pattern: FileOperationPattern{Glob: "**/*.md"}},
							{Pattern: FileOperationPattern{Glob: "**/*.mditamap"}},
						},
					},
					DidDelete: &FileOperationRegistration{
						Filters: []FileOperationFilter{
							{Pattern: FileOperationPattern{Glob: "**/*.md"}},
							{Pattern: FileOperationPattern{Glob: "**/*.mditamap"}},
						},
					},
					WillRename: &FileOperationRegistration{
						Filters: []FileOperationFilter{
							{Pattern: FileOperationPattern{Glob: "**/*.md"}},
							{Pattern: FileOperationPattern{Glob: "**/*.mditamap"}},
						},
					},
				},
			},
		},
	}, nil
}

func (s *Server) addWorkspaceFolder(uri string) {
	rootPath, _ := paths.URIToPath(uri)
	cfg := config.LoadMerged(rootPath)
	folder := workspace.NewFolder(uri, cfg)
	folder.ScanFiles()
	s.workspace.AddFolder(folder)

	for _, doc := range folder.AllDocs() {
		s.graph.AddDefs(doc.URI, doc.Defs())
		s.graph.AddRefs(doc.URI, doc.Refs())
	}
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
	s.publishDiagnosticsNow(doc, folder)

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

	text := doc.Text
	for _, change := range params.ContentChanges {
		if change.Range == nil {
			text = change.Text
		} else {
			text = applyIncrementalChange(text, doc.Lines, *change.Range, change.Text)
		}
	}

	newDoc := doc.ApplyChange(params.TextDocument.Version, text)
	folder.AddDoc(newDoc)
	s.graph.AddDefs(newDoc.URI, newDoc.Defs())
	s.graph.AddRefs(newDoc.URI, newDoc.Refs())
	s.scheduleDiagnostics(newDoc.URI, folder)
	return nil
}

func (s *Server) handleDidClose(_ context.Context, rawParams json.RawMessage) error {
	var params DidCloseParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	s.diagBounce.Flush(params.TextDocument.URI)

	s.notify("textDocument/publishDiagnostics", DiagnosticParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []DiagnosticResult{},
	})

	return nil
}

func (s *Server) handleDidSave(_ context.Context, rawParams json.RawMessage) error {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	s.diagBounce.Flush(params.TextDocument.URI)

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc != nil && folder != nil {
		s.publishDiagnosticsNow(doc, folder)
	}
	return nil
}

func (s *Server) handleDidChangeWorkspaceFolders(_ context.Context, rawParams json.RawMessage) error {
	var params struct {
		Event struct {
			Added   []WorkspaceFolder `json:"added"`
			Removed []WorkspaceFolder `json:"removed"`
		} `json:"event"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	for _, f := range params.Event.Removed {
		s.workspace.RemoveFolder(f.URI)
	}
	for _, f := range params.Event.Added {
		s.addWorkspaceFolder(f.URI)
	}
	return nil
}

func (s *Server) handleDidCreateFiles(_ context.Context, rawParams json.RawMessage) error {
	var params struct {
		Files []struct {
			URI string `json:"uri"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	for _, f := range params.Files {
		if !paths.IsMarkdownURI(f.URI) {
			continue
		}
		folder := s.workspace.FolderForURI(f.URI)
		if folder == nil {
			continue
		}
		filePath, err := paths.URIToPath(f.URI)
		if err != nil {
			continue
		}
		data, err := readFileBytes(filePath)
		if err != nil {
			continue
		}
		doc := document.New(f.URI, 0, string(data))
		folder.AddDoc(doc)
		s.graph.AddDefs(doc.URI, doc.Defs())
		s.graph.AddRefs(doc.URI, doc.Refs())
		s.publishDiagnosticsNow(doc, folder)
		s.refreshRelatedDiagnostics(folder)
	}
	return nil
}

func (s *Server) handleDidDeleteFiles(_ context.Context, rawParams json.RawMessage) error {
	var params struct {
		Files []struct {
			URI string `json:"uri"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	for _, f := range params.Files {
		folder := s.workspace.FolderForURI(f.URI)
		if folder == nil {
			continue
		}
		folder.RemoveDoc(f.URI)
		s.graph.RemoveDoc(f.URI)
		s.refreshRelatedDiagnostics(folder)
	}
	return nil
}

func (s *Server) handleWillRenameFiles(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		Files []struct {
			OldURI string `json:"oldUri"`
			NewURI string `json:"newUri"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	var renames []filerename.FileRename
	var folder *workspace.Folder
	for _, f := range params.Files {
		renames = append(renames, filerename.FileRename{
			OldURI: f.OldURI,
			NewURI: f.NewURI,
		})
		if folder == nil {
			folder = s.workspace.FolderForURI(f.OldURI)
		}
	}

	if folder == nil || len(renames) == 0 {
		return nil, nil
	}

	docEdits := filerename.ComputeEdits(renames, folder)
	if len(docEdits) == 0 {
		return nil, nil
	}

	changes := make(map[string][]TextEditResult)
	for _, de := range docEdits {
		for _, e := range de.Edits {
			changes[de.URI] = append(changes[de.URI], TextEditResult{
				Range:   e.Range,
				NewText: e.NewText,
			})
		}
	}

	return WorkspaceEditResult{Changes: changes}, nil
}

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
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
			Data:       item.Data,
		})
	}
	return results, nil
}

func (s *Server) handleCompletionResolve(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var item CompletionItemResult
	if err := json.Unmarshal(rawParams, &item); err != nil {
		return nil, err
	}

	var folder *workspace.Folder
	for _, f := range s.workspace.Folders() {
		folder = f
		break
	}
	if folder == nil {
		return item, nil
	}

	docs := completion.Resolve(item.Label, item.Data, folder)
	if docs != "" {
		item.Documentation = &MarkupContent{
			Kind:  "markdown",
			Value: docs,
		}
	}
	return item, nil
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

func (s *Server) handlePrepareRename(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	result := rename.Prepare(doc, params.Position)
	if result == nil {
		return nil, nil
	}
	return map[string]interface{}{
		"range":       result.Range,
		"placeholder": result.Text,
	}, nil
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
		entry := map[string]interface{}{
			"title": a.Title,
			"kind":  a.Kind,
		}
		if a.Edit != nil {
			entry["edit"] = map[string]interface{}{
				"changes": map[string][]map[string]interface{}{
					a.DocURI: {{
						"range":   a.Edit.Range,
						"newText": a.Edit.NewText,
					}},
				},
			}
		}
		if a.Command != nil {
			entry["command"] = map[string]interface{}{
				"title":     a.Command.Title,
				"command":   a.Command.Command,
				"arguments": a.Command.Arguments,
			}
		}
		results = append(results, entry)
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

func (s *Server) handleDocumentLink(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	var results []map[string]interface{}
	for _, wl := range doc.Index.WikiLinks() {
		if wl.Doc == "" {
			continue
		}
		targetSlug := paths.SlugOf(wl.Doc)
		target := folder.DocBySlug(targetSlug)
		if target != nil {
			results = append(results, map[string]interface{}{
				"range":  wl.Range,
				"target": target.URI,
			})
		}
	}
	for _, ml := range doc.Index.MdLinks() {
		if ml.URL != "" && !isExternalURL(ml.URL) {
			docPath, _ := paths.URIToPath(doc.URI)
			docDir := filepath.Dir(docPath)
			targetPath := filepath.Join(docDir, ml.URL)
			targetURI := paths.PathToURI(targetPath)
			if target := folder.DocByURI(targetURI); target != nil {
				results = append(results, map[string]interface{}{
					"range":  ml.Range,
					"target": target.URI,
				})
				continue
			}
			for _, d := range folder.AllDocs() {
				id := d.DocID(folder.RootURI)
				if id.RelPath == ml.URL || id.Stem+".md" == ml.URL {
					results = append(results, map[string]interface{}{
						"range":  ml.Range,
						"target": d.URI,
					})
					break
				}
			}
		}
	}
	return results, nil
}

func isExternalURL(url string) bool {
	return len(url) > 4 && (url[:4] == "http" || url[:2] == "//")
}

func (s *Server) handleFoldingRange(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
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

	ranges := folding.GetRanges(doc)
	var results []map[string]interface{}
	for _, r := range ranges {
		results = append(results, map[string]interface{}{
			"startLine":      r.StartLine,
			"startCharacter": r.StartCharacter,
			"endLine":        r.EndLine,
			"endCharacter":   r.EndCharacter,
			"kind":           r.Kind,
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

func (s *Server) handleWorkspaceSymbol(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	var allDocs []*document.Document
	for _, f := range s.workspace.Folders() {
		allDocs = append(allDocs, f.AllDocs()...)
	}

	syms := docsymbols.SearchWorkspace(allDocs, params.Query)
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

func (s *Server) handleSemanticTokensRange(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Range        document.Range         `json:"range"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	data := semantic.EncodeRange(doc, params.Range)
	return map[string]interface{}{"data": data}, nil
}

func (s *Server) handleSelectionRange(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Positions    []document.Position    `json:"positions"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	ranges := selection.GetRanges(doc, params.Positions)
	return ranges, nil
}

func (s *Server) handleLinkedEditingRange(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     document.Position      `json:"position"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return linkededit.GetLinkedRanges(doc, params.Position), nil
}

func (s *Server) handleFormatting(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Options      struct {
			TabSize      int  `json:"tabSize"`
			InsertSpaces bool `json:"insertSpaces"`
		} `json:"options"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	edits := formatting.Format(doc, formatting.Options{
		TabSize:      params.Options.TabSize,
		InsertSpaces: params.Options.InsertSpaces,
	})

	var results []TextEditResult
	for _, e := range edits {
		results = append(results, TextEditResult{
			Range:   e.Range,
			NewText: e.NewText,
		})
	}
	return results, nil
}

func (s *Server) handleInlayHint(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
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

	return inlayhint.GetHints(doc, params.Range, folder), nil
}

func (s *Server) scheduleDiagnostics(uri string, folder *workspace.Folder) {
	s.diagBounce.Schedule(uri, func() {
		doc, f := s.workspace.FindDoc(uri)
		if doc == nil {
			doc = folder.DocByURI(uri)
			f = folder
		}
		if doc != nil && f != nil {
			s.publishDiagnosticsNow(doc, f)
		}
	})
}

func (s *Server) publishDiagnosticsNow(doc *document.Document, folder *workspace.Folder) {
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

func (s *Server) refreshRelatedDiagnostics(folder *workspace.Folder) {
	for _, doc := range folder.AllDocs() {
		s.scheduleDiagnostics(doc.URI, folder)
	}
}

func applyIncrementalChange(text string, lineMap []int, rng document.Range, newText string) string {
	startOff := offsetFromPosition(lineMap, rng.Start)
	endOff := offsetFromPosition(lineMap, rng.End)
	if startOff < 0 {
		startOff = 0
	}
	if endOff > len(text) {
		endOff = len(text)
	}
	return text[:startOff] + newText + text[endOff:]
}

func offsetFromPosition(lineMap []int, pos document.Position) int {
	if pos.Line < 0 || pos.Line >= len(lineMap) {
		if pos.Line >= len(lineMap) && len(lineMap) > 0 {
			return lineMap[len(lineMap)-1] + pos.Character
		}
		return 0
	}
	return lineMap[pos.Line] + pos.Character
}

func parentURI(uri string) string {
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == '/' {
			return uri[:i]
		}
	}
	return uri
}

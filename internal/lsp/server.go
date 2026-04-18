package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aireilly/mdita-lsp/internal/codeaction"
	"github.com/aireilly/mdita-lsp/internal/codelens"
	"github.com/aireilly/mdita-lsp/internal/completion"
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/definition"
	"github.com/aireilly/mdita-lsp/internal/diagnostic"
	"github.com/aireilly/mdita-lsp/internal/ditaot"
	"github.com/aireilly/mdita-lsp/internal/docsymbols"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/filerename"
	"github.com/aireilly/mdita-lsp/internal/folding"
	"github.com/aireilly/mdita-lsp/internal/formatting"
	"github.com/aireilly/mdita-lsp/internal/highlight"
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
	workspace   *workspace.Workspace
	graph       *symbols.Graph
	notify      func(method string, params any)
	diagBounce  *debouncer
	version     string
	ditaBuilder *ditaot.Builder
}

func NewServer() *Server {
	return &Server{
		workspace:   workspace.New(),
		graph:       symbols.NewGraph(),
		notify:      func(string, any) {},
		diagBounce:  newDebouncer(200 * time.Millisecond),
		version:     "dev",
		ditaBuilder: &ditaot.Builder{},
	}
}

func (s *Server) SetVersion(v string) {
	s.version = v
}

func (s *Server) SetNotify(fn func(method string, params any)) {
	s.notify = fn
}

func (s *Server) logMessage(level int, msg string) {
	s.notify("window/logMessage", LogMessageParams{Type: level, Message: msg})
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
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync                int                    `json:"textDocumentSync"`
	CompletionProvider              *CompletionOptions     `json:"completionProvider,omitempty"`
	DefinitionProvider              bool                   `json:"definitionProvider"`
	HoverProvider                   bool                   `json:"hoverProvider"`
	ReferencesProvider              bool                   `json:"referencesProvider"`
	RenameProvider                  *RenameOptions         `json:"renameProvider,omitempty"`
	CodeActionProvider              bool                   `json:"codeActionProvider"`
	CodeLensProvider                *CodeLensOptions       `json:"codeLensProvider,omitempty"`
	DocumentHighlightProvider       bool                   `json:"documentHighlightProvider"`
	DocumentLinkProvider            bool                   `json:"documentLinkProvider"`
	FoldingRangeProvider            bool                   `json:"foldingRangeProvider"`
	DocumentSymbolProvider          bool                   `json:"documentSymbolProvider"`
	WorkspaceSymbolProvider         bool                   `json:"workspaceSymbolProvider"`
	SelectionRangeProvider          bool                   `json:"selectionRangeProvider"`
	LinkedEditingRangeProvider      bool                   `json:"linkedEditingRangeProvider"`
	DocumentFormattingProvider      bool                   `json:"documentFormattingProvider"`
	InlayHintProvider               bool                   `json:"inlayHintProvider"`
	DocumentRangeFormattingProvider bool                   `json:"documentRangeFormattingProvider"`
	DiagnosticProvider              *DiagnosticOptions     `json:"diagnosticProvider,omitempty"`
	ExecuteCommandProvider          *ExecuteCommandOptions `json:"executeCommandProvider,omitempty"`
	SemanticTokensProvider          *SemanticTokensOptions `json:"semanticTokensProvider,omitempty"`
	Workspace                       *WorkspaceCapabilities `json:"workspace,omitempty"`
}

type WorkspaceCapabilities struct {
	FileOperations *FileOperationCapabilities `json:"fileOperations,omitempty"`
}

type FileOperationCapabilities struct {
	WillCreate *FileOperationRegistration `json:"willCreate,omitempty"`
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

type DiagnosticOptions struct {
	InterFileDependencies bool `json:"interFileDependencies"`
	WorkspaceDiagnostics  bool `json:"workspaceDiagnostics"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
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

type CodeActionResult struct {
	Title       string               `json:"title"`
	Kind        string               `json:"kind"`
	Edit        *WorkspaceEditResult `json:"edit,omitempty"`
	Command     *CommandResult       `json:"command,omitempty"`
	Diagnostics []DiagnosticResult   `json:"diagnostics,omitempty"`
}

type CommandResult struct {
	Title     string   `json:"title"`
	Command   string   `json:"command"`
	Arguments []string `json:"arguments,omitempty"`
}

type CodeLensResult struct {
	Range   document.Range `json:"range"`
	Command CommandResult  `json:"command"`
}

type DocumentLinkResult struct {
	Range  document.Range `json:"range"`
	Target string         `json:"target"`
}

type FoldingRangeResult struct {
	StartLine      int    `json:"startLine"`
	StartCharacter int    `json:"startCharacter,omitempty"`
	EndLine        int    `json:"endLine"`
	EndCharacter   int    `json:"endCharacter,omitempty"`
	Kind           string `json:"kind,omitempty"`
}

type SemanticTokensResult struct {
	Data []uint32 `json:"data"`
}

type PrepareRenameResult struct {
	Range       document.Range `json:"range"`
	Placeholder string         `json:"placeholder"`
}

type ShowMessageParams struct {
	Type    int    `json:"type"`
	Message string `json:"message"`
}

type ApplyWorkspaceEditParams struct {
	Label string              `json:"label"`
	Edit  WorkspaceEditResult `json:"edit"`
}

type LogMessageParams struct {
	Type    int    `json:"type"`
	Message string `json:"message"`
}

const (
	LogError   = 1
	LogWarning = 2
	LogInfo    = 3
	LogDebug   = 4
)

type DocumentDiagnosticReport struct {
	Kind  string             `json:"kind"`
	Items []DiagnosticResult `json:"items"`
}

func (s *Server) handleInitialize(_ context.Context, rawParams json.RawMessage) (any, error) {
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
		ServerInfo: &ServerInfo{
			Name:    "mdita-lsp",
			Version: s.version,
		},
		Capabilities: ServerCapabilities{
			TextDocumentSync: 2,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"[", "#", "("},
				ResolveProvider:   true,
			},
			DefinitionProvider:              true,
			HoverProvider:                   true,
			ReferencesProvider:              true,
			RenameProvider:                  &RenameOptions{PrepareProvider: true},
			CodeActionProvider:              true,
			CodeLensProvider:                &CodeLensOptions{},
			DocumentHighlightProvider:       true,
			DocumentLinkProvider:            true,
			FoldingRangeProvider:            true,
			DocumentSymbolProvider:          true,
			WorkspaceSymbolProvider:         true,
			SelectionRangeProvider:          true,
			LinkedEditingRangeProvider:      true,
			DocumentFormattingProvider:      true,
			InlayHintProvider:               true,
			DocumentRangeFormattingProvider: true,
			DiagnosticProvider: &DiagnosticOptions{
				InterFileDependencies: true,
				WorkspaceDiagnostics:  false,
			},
			ExecuteCommandProvider: &ExecuteCommandOptions{
				Commands: []string{
					"mdita-lsp.createFile",
					"mdita-lsp.findReferences",
					"mdita-lsp.addToMap",
					"mdita-lsp.ditaOtBuild",
				},
			},
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
					WillCreate: &FileOperationRegistration{
						Filters: []FileOperationFilter{
							{Pattern: FileOperationPattern{Glob: "**/*.md"}},
						},
					},
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

	docs := folder.AllDocs()
	for _, doc := range docs {
		s.graph.AddDefs(doc.URI, doc.Defs())
		s.graph.AddRefs(doc.URI, doc.Refs())
	}
	s.logMessage(LogInfo, fmt.Sprintf("Indexed workspace %s: %d documents", rootPath, len(docs)))
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
	lineMap := doc.Lines
	for _, change := range params.ContentChanges {
		if change.Range == nil {
			text = change.Text
		} else {
			text = applyIncrementalChange(text, lineMap, *change.Range, change.Text)
		}
		lineMap = document.BuildLineMap(text)
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
		s.refreshRelatedDiagnostics(folder)
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

func (s *Server) handleWillCreateFiles(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params struct {
		Files []struct {
			URI string `json:"uri"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	changes := make(map[string][]TextEditResult)
	for _, f := range params.Files {
		if !paths.IsMarkdownURI(f.URI) {
			continue
		}
		filePath, err := paths.URIToPath(f.URI)
		if err != nil {
			continue
		}
		stem := filepath.Base(filePath)
		ext := filepath.Ext(stem)
		title := stem[:len(stem)-len(ext)]

		content := "---\n$schema: \"urn:oasis:names:tc:mdita:rng:topic.rng\"\n---\n\n# " + title + "\n\n"
		changes[f.URI] = []TextEditResult{{
			Range:   document.Rng(0, 0, 0, 0),
			NewText: content,
		}}
	}

	if len(changes) == 0 {
		return nil, nil
	}
	return WorkspaceEditResult{Changes: changes}, nil
}

func (s *Server) handleDidChangeConfiguration(_ context.Context, _ json.RawMessage) error {
	for _, folder := range s.workspace.Folders() {
		rootPath := folder.RootPath()
		folder.Config = config.LoadMerged(rootPath)
		s.refreshRelatedDiagnostics(folder)
	}
	s.logMessage(LogInfo, "Configuration reloaded")
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

func (s *Server) handleWillRenameFiles(_ context.Context, rawParams json.RawMessage) (any, error) {
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
	s.logMessage(LogInfo, fmt.Sprintf("File rename: updating %d documents", len(docEdits)))

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

func (s *Server) handleCompletion(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return []CompletionItemResult{}, nil
	}

	items := completion.Complete(doc, params.Position, folder)
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

func (s *Server) handleCompletionResolve(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handleDefinition(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	locs := definition.GotoDef(doc, params.Position, folder)
	var results []LocationResult
	for _, loc := range locs {
		results = append(results, LocationResult{URI: loc.URI, Range: loc.Range})
	}
	return results, nil
}

func (s *Server) handleHover(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	content := hover.GetHover(doc, params.Position, folder)
	if content == "" {
		return nil, nil
	}
	return HoverResult{Contents: content}, nil
}

func (s *Server) handleDocumentHighlight(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return highlight.GetHighlights(doc, params.Position), nil
}

func (s *Server) handleReferences(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handlePrepareRename(_ context.Context, rawParams json.RawMessage) (any, error) {
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
	return PrepareRenameResult{
		Range:       result.Range,
		Placeholder: result.Text,
	}, nil
}

func (s *Server) handleRename(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params RenameParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	edits := rename.DoRename(doc, params.Position, params.NewName, folder, s.graph)
	changes := make(map[string][]TextEditResult)
	for _, edit := range edits {
		changes[edit.URI] = append(changes[edit.URI], TextEditResult{
			Range:   edit.Range,
			NewText: edit.NewText,
		})
	}
	return WorkspaceEditResult{Changes: changes}, nil
}

func (s *Server) handleCodeAction(_ context.Context, rawParams json.RawMessage) (any, error) {
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
	var results []CodeActionResult
	for _, a := range actions {
		entry := CodeActionResult{
			Title: a.Title,
			Kind:  a.Kind,
		}
		if a.Edit != nil {
			entry.Edit = &WorkspaceEditResult{
				Changes: map[string][]TextEditResult{
					a.DocURI: {{
						Range:   a.Edit.Range,
						NewText: a.Edit.NewText,
					}},
				},
			}
		}
		if a.Command != nil {
			entry.Command = &CommandResult{
				Title:     a.Command.Title,
				Command:   a.Command.Command,
				Arguments: a.Command.Arguments,
			}
		}
		for _, d := range a.Diagnostics {
			entry.Diagnostics = append(entry.Diagnostics, DiagnosticResult{
				Range:    d.Range,
				Severity: d.Severity,
				Code:     d.Code,
				Source:   d.Source,
				Message:  d.Message,
			})
		}
		results = append(results, entry)
	}
	return results, nil
}

func (s *Server) handleCodeLens(_ context.Context, rawParams json.RawMessage) (any, error) {
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
	var results []CodeLensResult
	for _, l := range lenses {
		results = append(results, CodeLensResult{
			Range: l.Range,
			Command: CommandResult{
				Title:   l.Title,
				Command: l.Command,
			},
		})
	}
	return results, nil
}

func (s *Server) handleDocumentLink(_ context.Context, rawParams json.RawMessage) (any, error) {
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

	var results []DocumentLinkResult
	for _, wl := range doc.Index.WikiLinks() {
		if wl.Doc == "" {
			continue
		}
		targetSlug := paths.SlugOf(wl.Doc)
		target := folder.DocBySlug(targetSlug)
		if target != nil {
			results = append(results, DocumentLinkResult{
				Range:  wl.Range,
				Target: target.URI,
			})
		}
	}
	for _, ml := range doc.Index.MdLinks() {
		if ml.URL != "" && !isExternalURL(ml.URL) {
			if target := folder.ResolveLink(ml.URL, doc.URI); target != nil {
				results = append(results, DocumentLinkResult{
					Range:  ml.Range,
					Target: target.URI,
				})
			}
		}
	}
	return results, nil
}

func isExternalURL(url string) bool {
	return len(url) > 4 && (url[:4] == "http" || url[:2] == "//")
}

func (s *Server) handleFoldingRange(_ context.Context, rawParams json.RawMessage) (any, error) {
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
	var results []FoldingRangeResult
	for _, r := range ranges {
		results = append(results, FoldingRangeResult{
			StartLine:      r.StartLine,
			StartCharacter: r.StartCharacter,
			EndLine:        r.EndLine,
			EndCharacter:   r.EndCharacter,
			Kind:           r.Kind,
		})
	}
	return results, nil
}

func (s *Server) handleDocumentSymbol(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handleWorkspaceSymbol(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handleSemanticTokensFull(_ context.Context, rawParams json.RawMessage) (any, error) {
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
	return SemanticTokensResult{Data: data}, nil
}

func (s *Server) handleSemanticTokensRange(_ context.Context, rawParams json.RawMessage) (any, error) {
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
	return SemanticTokensResult{Data: data}, nil
}

func (s *Server) handleSelectionRange(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handleLinkedEditingRange(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handleFormatting(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handleInlayHint(_ context.Context, rawParams json.RawMessage) (any, error) {
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

func (s *Server) handlePullDiagnostics(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return DocumentDiagnosticReport{Kind: "full", Items: []DiagnosticResult{}}, nil
	}

	diags := diagnostic.Check(doc, folder)
	var items []DiagnosticResult
	for _, d := range diags {
		items = append(items, DiagnosticResult{
			Range:    d.Range,
			Severity: int(d.Severity),
			Code:     d.Code,
			Source:   d.Source,
			Message:  d.Message,
		})
	}
	return DocumentDiagnosticReport{Kind: "full", Items: items}, nil
}

func (s *Server) handleRangeFormatting(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Range        document.Range         `json:"range"`
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

	allEdits := formatting.Format(doc, formatting.Options{
		TabSize:      params.Options.TabSize,
		InsertSpaces: params.Options.InsertSpaces,
	})

	var results []TextEditResult
	for _, e := range allEdits {
		if e.Range.End.Line < params.Range.Start.Line || e.Range.Start.Line > params.Range.End.Line {
			continue
		}
		results = append(results, TextEditResult{
			Range:   e.Range,
			NewText: e.NewText,
		})
	}
	return results, nil
}

func (s *Server) handleExecuteCommand(_ context.Context, rawParams json.RawMessage) (any, error) {
	var params struct {
		Command   string   `json:"command"`
		Arguments []string `json:"arguments"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	switch params.Command {
	case "mdita-lsp.createFile":
		return s.executeCreateFile(params.Arguments)
	case "mdita-lsp.addToMap":
		return s.executeAddToMap(params.Arguments)
	case "mdita-lsp.ditaOtBuild":
		return s.executeDitaOtBuild(params.Arguments)
	}
	return nil, nil
}

func (s *Server) executeCreateFile(args []string) (any, error) {
	if len(args) < 1 {
		return nil, nil
	}
	filename := args[0]

	var folder *workspace.Folder
	for _, f := range s.workspace.Folders() {
		folder = f
		break
	}
	if folder == nil {
		return nil, nil
	}

	rootPath := folder.RootPath()
	filePath := filepath.Join(rootPath, filename)

	if _, err := os.Stat(filePath); err == nil {
		return nil, nil
	}

	stem := filepath.Base(filename)
	ext := filepath.Ext(stem)
	title := stem[:len(stem)-len(ext)]

	content := "# " + title + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	s.logMessage(LogInfo, "Created file: "+filename)

	uri := paths.PathToURI(filePath)
	doc := document.New(uri, 0, content)
	folder.AddDoc(doc)
	s.graph.AddDefs(uri, doc.Defs())
	s.graph.AddRefs(uri, doc.Refs())

	s.notify("window/showMessage", ShowMessageParams{
		Type:    3,
		Message: "Created " + filename,
	})
	return nil, nil
}

func (s *Server) executeAddToMap(args []string) (any, error) {
	if len(args) < 2 {
		return nil, nil
	}
	docURI := args[0]
	mapURI := args[1]

	doc, _ := s.workspace.FindDoc(docURI)
	mapDoc, _ := s.workspace.FindDoc(mapURI)
	if doc == nil || mapDoc == nil {
		return nil, nil
	}

	title := ""
	if t := doc.Index.Title(); t != nil {
		title = t.Text
	}
	docID := doc.DocID(s.workspace.FolderForURI(docURI).RootURI)

	newEntry := "- [" + title + "](" + docID.RelPath + ")\n"

	lastLine := len(mapDoc.Lines) - 1
	if lastLine < 0 {
		lastLine = 0
	}

	s.notify("workspace/applyEdit", ApplyWorkspaceEditParams{
		Label: "Add to map",
		Edit: WorkspaceEditResult{
			Changes: map[string][]TextEditResult{
				mapURI: {{
					Range:   document.Rng(lastLine, 0, lastLine, 0),
					NewText: newEntry,
				}},
			},
		},
	})
	return nil, nil
}

func (s *Server) executeDitaOtBuild(args []string) (any, error) {
	if len(args) < 2 {
		return nil, nil
	}
	mapURI := args[0]
	format := args[1]

	folder := s.workspace.FolderForURI(mapURI)
	if folder == nil {
		s.notify("window/showMessage", ShowMessageParams{
			Type:    LogError,
			Message: "DITA OT build: no workspace folder found",
		})
		return nil, nil
	}

	ditaPath, err := ditaot.ResolveDitaPath(folder.Config.Build.DitaOT.DitaPath)
	if err != nil {
		s.notify("window/showMessage", ShowMessageParams{
			Type:    LogError,
			Message: err.Error(),
		})
		return nil, nil
	}

	mapPath, err := paths.URIToPath(mapURI)
	if err != nil {
		return nil, nil
	}

	outputDir := folder.Config.Build.DitaOT.OutputDir
	if outputDir == "" {
		outputDir = "out"
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(folder.RootPath(), outputDir)
	}

	if !s.ditaBuilder.TryAcquire() {
		s.notify("window/showMessage", ShowMessageParams{
			Type:    LogWarning,
			Message: "DITA OT build already in progress",
		})
		return nil, nil
	}

	mapName := filepath.Base(mapPath)
	s.logMessage(LogInfo, fmt.Sprintf("DITA OT build started: %s (format: %s)", mapName, format))

	go func() {
		defer s.ditaBuilder.Release()

		result, err := s.ditaBuilder.Run(context.Background(), ditaPath, mapPath, format, outputDir)
		if err != nil {
			s.notify("window/showMessage", ShowMessageParams{
				Type:    LogError,
				Message: "DITA OT build error: " + err.Error(),
			})
			return
		}

		if result.Output != "" {
			s.logMessage(LogInfo, result.Output)
		}

		if result.Success {
			s.notify("window/showMessage", ShowMessageParams{
				Type:    LogInfo,
				Message: fmt.Sprintf("DITA OT build complete (%s). Output: %s", result.Elapsed.Round(time.Millisecond), outputDir),
			})
		} else {
			s.notify("window/showMessage", ShowMessageParams{
				Type:    LogError,
				Message: "DITA OT build failed. See output log for details.",
			})
		}
	}()

	return nil, nil
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

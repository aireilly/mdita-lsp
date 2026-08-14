package lsp

import (
	"context"
	"encoding/json"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/formatting"
)

type WillSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Reason       int                    `json:"reason"`
}

func (s *Server) handleWillSaveWaitUntil(ctx context.Context, params json.RawMessage) (any, error) {
	var p WillSaveTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	uri := p.TextDocument.URI
	folder := s.workspace.FolderForURI(uri)
	if folder == nil {
		return []any{}, nil
	}

	if !config.BoolVal(folder.Config.Core.Mdita.FormatTablesOnSave) {
		return []any{}, nil
	}

	doc := folder.DocByURI(uri)
	if doc == nil {
		return []any{}, nil
	}

	edits := formatting.AlignTables(doc.Text)
	if len(edits) == 0 {
		return []any{}, nil
	}

	var lspEdits []map[string]any
	for _, e := range edits {
		lspEdits = append(lspEdits, map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": e.Range.Start.Line, "character": e.Range.Start.Character},
				"end":   map[string]any{"line": e.Range.End.Line, "character": e.Range.End.Character},
			},
			"newText": e.NewText,
		})
	}
	return lspEdits, nil
}

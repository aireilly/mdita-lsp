package codeaction

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type CodeAction struct {
	Title   string
	Kind    string
	DocURI  string
	Edit    *TextEdit
	Command *Command
}

type TextEdit struct {
	Range   document.Range
	NewText string
}

type Command struct {
	Title     string
	Command   string
	Arguments []string
}

func GetActions(doc *document.Document, rng document.Range, folder *workspace.Folder) []CodeAction {
	var actions []CodeAction
	cfg := folder.Config

	if cfg.CodeActions.ToC.Enable {
		actions = append(actions, CodeAction{
			Title:  "Generate table of contents",
			Kind:   "source",
			DocURI: doc.URI,
		})
	}

	if cfg.CodeActions.CreateMissingFile.Enable {
		for _, wl := range doc.Index.WikiLinks() {
			if rangesOverlap(rng, wl.Range) && wl.Doc != "" {
				target := folder.DocBySlug(paths.SlugOf(wl.Doc))
				if target == nil {
					actions = append(actions, CodeAction{
						Title:  "Create '" + wl.Doc + ".md'",
						Kind:   "quickfix",
						DocURI: doc.URI,
						Command: &Command{
							Title:     "Create file",
							Command:   "mdita-lsp.createFile",
							Arguments: []string{wl.Doc + ".md"},
						},
					})
				}
			}
		}
	}

	return actions
}

func rangesOverlap(a, b document.Range) bool {
	if a.End.Line < b.Start.Line || b.End.Line < a.Start.Line {
		return false
	}
	return true
}

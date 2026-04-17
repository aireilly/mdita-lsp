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
		toc := GenerateToC(doc, cfg.CodeActions.ToC.IncludeLevels)
		if toc != "" {
			insertLine := 0
			title := doc.Index.Title()
			if title != nil {
				insertLine = title.Range.End.Line + 1
			}
			actions = append(actions, CodeAction{
				Title:  "Generate table of contents",
				Kind:   "source",
				DocURI: doc.URI,
				Edit: &TextEdit{
					Range:   document.Rng(insertLine, 0, insertLine, 0),
					NewText: "\n" + toc + "\n",
				},
			})
		}
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

	actions = append(actions, convertWikiLinkActions(doc, rng, folder)...)
	actions = append(actions, addFrontMatterAction(doc)...)
	actions = append(actions, addToMapActions(doc, folder)...)

	return actions
}

func convertWikiLinkActions(doc *document.Document, rng document.Range, folder *workspace.Folder) []CodeAction {
	var actions []CodeAction
	for _, wl := range doc.Index.WikiLinks() {
		if !rangesOverlap(rng, wl.Range) || wl.Doc == "" {
			continue
		}
		target := folder.DocBySlug(paths.SlugOf(wl.Doc))
		if target == nil {
			continue
		}
		targetID := target.DocID(folder.RootURI)
		url := targetID.RelPath
		if wl.Heading != "" {
			url += "#" + paths.Slugify(wl.Heading)
		}
		title := wl.Doc
		if wl.Title != "" {
			title = wl.Title
		}
		actions = append(actions, CodeAction{
			Title:  "Convert to markdown link",
			Kind:   "refactor",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   wl.Range,
				NewText: "[" + title + "](" + url + ")",
			},
		})
	}
	return actions
}

func addFrontMatterAction(doc *document.Document) []CodeAction {
	if doc.Meta != nil && doc.Meta.SchemaRaw != "" {
		return nil
	}
	if doc.Kind != document.Topic {
		return nil
	}
	fm := "---\n$schema: \"urn:oasis:names:tc:mdita:rng:topic.rng\"\n---\n\n"
	return []CodeAction{{
		Title:  "Add MDITA YAML front matter",
		Kind:   "source",
		DocURI: doc.URI,
		Edit: &TextEdit{
			Range:   document.Rng(0, 0, 0, 0),
			NewText: fm,
		},
	}}
}

func addToMapActions(doc *document.Document, folder *workspace.Folder) []CodeAction {
	if doc.Kind != document.Topic {
		return nil
	}
	var actions []CodeAction
	for _, d := range folder.AllDocs() {
		if d.Kind != document.Map {
			continue
		}
		actions = append(actions, CodeAction{
			Title:  "Add to " + mapTitle(d),
			Kind:   "source",
			DocURI: doc.URI,
			Command: &Command{
				Title:     "Add to map",
				Command:   "mdita-lsp.addToMap",
				Arguments: []string{doc.URI, d.URI},
			},
		})
	}
	return actions
}

func mapTitle(doc *document.Document) string {
	if t := doc.Index.Title(); t != nil {
		return t.Text
	}
	return "map"
}

func rangesOverlap(a, b document.Range) bool {
	if a.End.Line < b.Start.Line || b.End.Line < a.Start.Line {
		return false
	}
	return true
}

package codeaction

import (
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type DiagnosticInfo struct {
	Range    document.Range
	Severity int
	Code     string
	Source   string
	Message  string
}

type CodeAction struct {
	Title       string
	Kind        string
	DocURI      string
	Edit        *TextEdit
	Command     *Command
	Diagnostics []DiagnosticInfo
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

	if config.BoolVal(cfg.CodeActions.ToC.Enable) {
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

	if config.BoolVal(cfg.CodeActions.CreateMissingFile.Enable) {
		for _, ml := range doc.Index.MdLinks() {
			if rangesOverlap(rng, ml.Range) && ml.URL != "" {
				target := folder.ResolveLink(ml.URL, doc.URI)
				if target == nil {
					actions = append(actions, createMissingFileAction(ml.URL, doc.URI, folder))
				}
			}
		}
	}

	actions = append(actions, addFrontMatterAction(doc)...)
	actions = append(actions, addToMapActions(doc, folder)...)
	actions = append(actions, fixNBSPActions(doc, rng)...)
	actions = append(actions, fixFootnoteRefActions(doc, rng)...)
	actions = append(actions, fixHeadingHierarchyActions(doc, rng)...)
	actions = append(actions, buildDitaOTActions(doc, folder)...)

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

func fixNBSPActions(doc *document.Document, rng document.Range) []CodeAction {
	var actions []CodeAction
	for _, h := range doc.Index.Headings() {
		if !rangesOverlap(rng, h.Range) {
			continue
		}
		if !strings.ContainsRune(h.Text, '\u00A0') {
			continue
		}
		prefix := strings.Repeat("#", h.Level) + " "
		fixed := strings.ReplaceAll(h.Text, "\u00A0", " ")
		actions = append(actions, CodeAction{
			Title:  "Replace non-breaking whitespace",
			Kind:   "quickfix",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   h.Range,
				NewText: prefix + fixed,
			},
			Diagnostics: []DiagnosticInfo{{
				Range:    h.Range,
				Severity: 2,
				Code:     "3",
				Source:   "mdita-lsp",
				Message:  "Heading contains non-breaking whitespace",
			}},
		})
	}
	return actions
}

func fixFootnoteRefActions(doc *document.Document, rng document.Range) []CodeAction {
	var actions []CodeAction
	bf := doc.Index.Features
	defLabels := make(map[string]bool)
	for _, def := range bf.FootnoteDefLabels {
		defLabels[def.Label] = true
	}

	for _, ref := range bf.FootnoteRefLabels {
		if defLabels[ref.Label] {
			continue
		}
		if !rangesOverlap(rng, ref.Range) {
			continue
		}
		lastLine := len(doc.Lines) - 1
		if lastLine < 0 {
			lastLine = 0
		}
		newText := "\n[^" + ref.Label + "]: \n"
		actions = append(actions, CodeAction{
			Title:  "Add footnote definition for '" + ref.Label + "'",
			Kind:   "quickfix",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   document.Rng(lastLine, 0, lastLine, 0),
				NewText: newText,
			},
			Diagnostics: []DiagnosticInfo{{
				Range:    ref.Range,
				Severity: 2,
				Code:     "13",
				Source:   "mdita-lsp",
				Message:  "Footnote reference without definition: " + ref.Label,
			}},
		})
	}
	return actions
}

func fixHeadingHierarchyActions(doc *document.Document, rng document.Range) []CodeAction {
	var actions []CodeAction
	headings := doc.Index.Headings()
	for i := 1; i < len(headings); i++ {
		prev := headings[i-1].Level
		curr := headings[i].Level
		if curr <= prev+1 {
			continue
		}
		if !rangesOverlap(rng, headings[i].Range) {
			continue
		}
		fixedLevel := prev + 1
		prefix := strings.Repeat("#", fixedLevel) + " "
		actions = append(actions, CodeAction{
			Title:  "Fix heading level",
			Kind:   "quickfix",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   headings[i].Range,
				NewText: prefix + headings[i].Text,
			},
			Diagnostics: []DiagnosticInfo{{
				Range:    headings[i].Range,
				Severity: 2,
				Code:     "6",
				Source:   "mdita-lsp",
				Message:  "Invalid heading hierarchy: skipped heading level",
			}},
		})
	}
	return actions
}

func buildDitaOTActions(doc *document.Document, folder *workspace.Folder) []CodeAction {
	if doc.Kind != document.Map {
		return nil
	}
	if !config.BoolVal(folder.Config.Build.DitaOT.Enable) {
		return nil
	}
	return []CodeAction{
		{
			Title:  "Build XHTML with DITA OT",
			Kind:   "source",
			DocURI: doc.URI,
			Command: &Command{
				Title:     "Build XHTML",
				Command:   "mdita-lsp.ditaOtBuild",
				Arguments: []string{doc.URI, "xhtml"},
			},
		},
		{
			Title:  "Build DITA with DITA OT",
			Kind:   "source",
			DocURI: doc.URI,
			Command: &Command{
				Title:     "Build DITA",
				Command:   "mdita-lsp.ditaOtBuild",
				Arguments: []string{doc.URI, "dita"},
			},
		},
	}
}

func createMissingFileAction(relPath string, sourceURI string, folder *workspace.Folder) CodeAction {
	srcPath, _ := paths.URIToPath(sourceURI)
	srcDir := filepath.Dir(srcPath)
	fullPath := filepath.Clean(filepath.Join(srcDir, relPath))
	fileURI := paths.PathToURI(fullPath)

	return CodeAction{
		Title:  "Create '" + relPath + "'",
		Kind:   "quickfix",
		DocURI: sourceURI,
		Command: &Command{
			Title:     "Create file",
			Command:   "mdita-lsp.createFile",
			Arguments: []string{fileURI},
		},
	}
}

func rangesOverlap(a, b document.Range) bool {
	if a.End.Line < b.Start.Line || b.End.Line < a.Start.Line {
		return false
	}
	return true
}

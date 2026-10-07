package codeaction

import (
	"path/filepath"
	"strconv"
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
	Arguments []any
}

func GetActions(doc *document.Document, rng document.Range, folder *workspace.Folder) []CodeAction {
	var actions []CodeAction
	cfg := folder.Config

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
	actions = append(actions, addTaskSectionActions(doc)...)

	return actions
}

// mditaSchemaURN is the MDITA extended topic schema. The action used to
// insert the DITA topic URN, which selects the Markdown DITA parser, not an
// MDITA one.
const mditaSchemaURN = "urn:oasis:names:tc:mdita:xsd:topic.xsd"

// addFrontMatterAction offers to declare the MDITA schema.
//
// A file that already has front matter but no $schema gets the key added
// inside that block. Inserting a second block at the top of the file left the
// topic with two, and the plug-in reads only the first.
func addFrontMatterAction(doc *document.Document) []CodeAction {
	if doc.Kind != document.Topic {
		return nil
	}
	if doc.Meta != nil && doc.Meta.SchemaRaw != "" {
		return nil
	}

	if doc.Meta == nil {
		return []CodeAction{{
			Title:  "Add MDITA YAML front matter",
			Kind:   "source",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   document.Rng(0, 0, 0, 0),
				NewText: "---\n$schema: \"" + mditaSchemaURN + "\"\n---\n\n",
			},
		}}
	}

	// Insert the key just inside the opening "---".
	return []CodeAction{{
		Title:  "Add MDITA $schema to the front matter",
		Kind:   "source",
		DocURI: doc.URI,
		Edit: &TextEdit{
			Range:   document.Rng(1, 0, 1, 0),
			NewText: "$schema: \"" + mditaSchemaURN + "\"\n",
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
				Arguments: []any{doc.URI, d.URI},
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
				// LineRange, not Range: a heading's own range starts after
				// the hashes, so writing them back doubled them.
				Range:   h.LineRange,
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
				Code:     "8",
				Source:   "mdita-lsp",
				Message:  "Footnote reference without definition: " + ref.Label,
			}},
		})
	}
	return actions
}

func fixHeadingHierarchyActions(doc *document.Document, rng document.Range) []CodeAction {
	var actions []CodeAction
	level := 0
	for _, h := range doc.Index.Headings() {
		prev := level
		level = h.Level
		if h.Level <= prev+1 {
			continue
		}
		if !rangesOverlap(rng, h.Range) {
			continue
		}
		fixedLevel := prev + 1
		prefix := strings.Repeat("#", fixedLevel) + " "
		actions = append(actions, CodeAction{
			Title:  "Fix heading level",
			Kind:   "quickfix",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   h.LineRange,
				NewText: prefix + h.Text,
			},
			Diagnostics: []DiagnosticInfo{{
				Range:    h.Range,
				Severity: 1,
				Code:     "6",
				Source:   "mdita-lsp",
				Message: "Heading level raised from " + itoa(prev) + " to " + itoa(h.Level) +
					" without an intermediate heading level; the build fails on this",
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
				Arguments: []any{doc.URI, "xhtml"},
			},
		},
		{
			Title:  "Build DITA with DITA OT",
			Kind:   "source",
			DocURI: doc.URI,
			Command: &Command{
				Title:     "Build DITA",
				Command:   "mdita-lsp.ditaOtBuild",
				Arguments: []any{doc.URI, "dita"},
			},
		},
	}
}

func addTaskSectionActions(doc *document.Document) []CodeAction {
	if !document.IsTaskTopic(doc) {
		return nil
	}

	existing := make(map[document.TaskSectionKind]bool)
	for _, h := range doc.Index.Headings() {
		if h.TaskSection != document.TaskSectionNone {
			existing[h.TaskSection] = true
		}
	}

	type sectionTemplate struct {
		kind  document.TaskSectionKind
		title string
	}
	templates := []sectionTemplate{
		{document.TaskSectionPrereq, "Prerequisites"},
		{document.TaskSectionContext, "About this task"},
		{document.TaskSectionSteps, "Procedure"},
		{document.TaskSectionResult, "Verification"},
		{document.TaskSectionPostreq, "Next steps"},
	}

	var actions []CodeAction
	for _, tmpl := range templates {
		if existing[tmpl.kind] {
			continue
		}
		lastLine := len(strings.Split(doc.Text, "\n")) - 1
		actions = append(actions, CodeAction{
			Title:  "Add " + tmpl.title + " section",
			Kind:   "source",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   document.Rng(lastLine, 0, lastLine, 0),
				NewText: "\n## " + tmpl.title + "\n\n\n",
			},
		})
	}
	return actions
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
			Arguments: []any{fileURI},
		},
	}
}

func rangesOverlap(a, b document.Range) bool {
	if a.End.Line < b.Start.Line || b.End.Line < a.Start.Line {
		return false
	}
	return true
}

func itoa(i int) string { return strconv.Itoa(i) }

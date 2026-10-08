package completion

import (
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type TextEdit struct {
	Range   document.Range
	NewText string
}

type CompletionItem struct {
	Label         string
	Detail        string
	InsertText    string
	FilterText    string
	Kind          int // 1=text, 6=variable, 17=keyword, 18=snippet
	Documentation string
	Data          map[string]string
	TextEdit      *TextEdit
}

var yamlKeys = []string{
	"$schema", "id", "author", "source", "publisher", "permissions",
	"audience", "category", "keyword", "resourceid",
}

func Complete(doc *document.Document, pos document.Position, folder *workspace.Folder) []CompletionItem {
	return limit(complete(doc, pos, folder), folder.Config.Completion.MaxCandidates)
}

// limit caps the candidate list at completion.max_candidates, which was
// parsed from the config and never applied.
func limit(items []CompletionItem, max int) []CompletionItem {
	if max > 0 && len(items) > max {
		return items[:max]
	}
	return items
}

func complete(doc *document.Document, pos document.Position, folder *workspace.Folder) []CompletionItem {
	pe := DetectPartial(doc.Text, pos)
	if pe == nil {
		return nil
	}

	switch pe.Kind {
	case PartialInlineLink:
		return completeInlineDoc(pe.Input, doc, folder)
	case PartialInlineAnchor:
		return completeInlineAnchor(pe.DocPart, pe.Input, doc, folder)
	case PartialYamlKey:
		return completeYamlKey(pe.Input)
	case PartialKeyref:
		return completeKeyref(pe.Input, doc, folder, pe.Range)
	case PartialHeadingText:
		return completeTaskSectionHeading(pe.Input, doc)
	case PartialDataKeyref:
		return completeDataKeyref(pe.Input, folder, pe.Range)
	case PartialConref:
		return completeConref(pe.Input, doc, folder, pe.Range)
	case PartialConkeyref:
		return completeConkeyref(pe.Input, folder, pe.Range)
	}
	return nil
}

func completeInlineDoc(input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	srcPath, _ := paths.URIToPath(doc.URI)
	srcDir := filepath.Dir(srcPath)

	var items []CompletionItem
	for _, d := range folder.AllDocs() {
		if d.URI == doc.URI {
			continue
		}
		targetPath, _ := paths.URIToPath(d.URI)
		rel := paths.RelPath(srcDir, targetPath)
		rel = filepath.ToSlash(rel)

		if input == "" || strings.Contains(strings.ToLower(rel), strings.ToLower(input)) {
			title := ""
			if t := d.Index.Title(); t != nil {
				title = t.Text
			}
			items = append(items, CompletionItem{
				Label:      rel,
				Detail:     title,
				InsertText: rel,
				Kind:       17,
			})
		}
	}
	return items
}

func completeInlineAnchor(docPart, input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	var target *document.Document
	if docPart == "" {
		target = doc
	} else {
		target = folder.ResolveLink(docPart, doc.URI)
	}
	if target == nil {
		return nil
	}

	inputSlug := paths.SlugOf(input)
	// Tasks and generic topics are unchanged, so only a concept or a
	// reference gets the qualified form offered.
	sectionTopic := document.IsSectionTopic(target)
	var items []CompletionItem
	for _, h := range target.Index.Headings() {
		if inputSlug != "" && !h.Slug.Contains(inputSlug) {
			continue
		}
		// A section is not a topic, so the build resolves a link to one only
		// through the id of the topic that holds it.
		anchor := h.ID
		if h.Section && sectionTopic {
			anchor = target.SectionAddress(h)
		}
		items = append(items, CompletionItem{
			Label:      anchor,
			Detail:     h.Text,
			InsertText: anchor,
			Kind:       17,
		})
	}
	return items
}

func completeKeyref(input string, doc *document.Document, folder *workspace.Folder, editRange document.Range) []CompletionItem {
	table := keyref.BuildMergedTable(folder.MapTexts())

	var items []CompletionItem
	for _, key := range keyref.AllKeys(table) {
		if input == "" || strings.Contains(strings.ToLower(key), strings.ToLower(input)) {
			entry := table[key]
			detail := entry.Href
			if entry.Title != "" {
				detail = entry.Title + " (" + entry.Href + ")"
			}
			// A keyref is written as a reference-style link. The plug-in turns
			// it into <xref keyref="..."/> as long as the topic itself does not
			// define the label.
			newText := "[" + key + "]"
			items = append(items, CompletionItem{
				Label:  key,
				Detail: detail,
				Kind:   18,
				Data:   map[string]string{"kind": "keyref"},
				TextEdit: &TextEdit{
					Range:   editRange,
					NewText: newText,
				},
			})
		}
	}
	return items
}

func completeDataKeyref(input string, folder *workspace.Folder, editRange document.Range) []CompletionItem {
	table := keyref.BuildMergedTable(folder.MapTexts())
	var items []CompletionItem
	for _, key := range keyref.AllKeys(table) {
		if input == "" || strings.Contains(strings.ToLower(key), strings.ToLower(input)) {
			entry := table[key]
			detail := entry.Href
			if entry.Title != "" {
				detail = entry.Title
			}
			items = append(items, CompletionItem{
				Label:  key,
				Detail: detail,
				Kind:   18,
				Data:   map[string]string{"kind": "data-keyref"},
				TextEdit: &TextEdit{
					Range:   editRange,
					NewText: key,
				},
			})
		}
	}
	return items
}

func completeYamlKey(input string) []CompletionItem {
	var items []CompletionItem
	for _, key := range yamlKeys {
		if input == "" || strings.HasPrefix(key, input) || strings.HasPrefix("$"+key, input) {
			items = append(items, CompletionItem{
				Label:      key,
				InsertText: key + ": ",
				Kind:       6,
			})
		}
	}
	return items
}

// completeConref offers file paths for data-conref="..." attributes.
// After a '#', it switches to offering topic/element IDs from the matched file.
func completeConref(input string, doc *document.Document, folder *workspace.Folder, editRange document.Range) []CompletionItem {
	srcPath, _ := paths.URIToPath(doc.URI)
	srcDir := filepath.Dir(srcPath)

	// If input contains '#', offer heading IDs from the referenced file.
	if hashIdx := strings.Index(input, "#"); hashIdx >= 0 {
		filePart := input[:hashIdx]
		idPart := input[hashIdx+1:]
		targetPath := filepath.Join(srcDir, filePart)
		targetURI := paths.PathToURI(targetPath)
		targetDoc := folder.DocByURI(targetURI)
		if targetDoc == nil {
			return nil
		}
		var items []CompletionItem
		for _, el := range targetDoc.Elements {
			h, ok := el.(*document.Heading)
			if !ok || h.ID == "" {
				continue
			}
			conrefID := h.ID
			if idPart != "" && !strings.Contains(strings.ToLower(conrefID), strings.ToLower(idPart)) {
				continue
			}
			items = append(items, CompletionItem{
				Label:      filePart + "#" + conrefID,
				Detail:     h.Text,
				InsertText: filePart + "#" + conrefID,
				Kind:       17,
				Data:       map[string]string{"kind": "conref-id"},
				TextEdit: &TextEdit{
					Range:   editRange,
					NewText: filePart + "#" + conrefID,
				},
			})
		}
		return items
	}

	var items []CompletionItem
	for _, d := range folder.AllDocs() {
		if d.URI == doc.URI {
			continue
		}
		targetPath, _ := paths.URIToPath(d.URI)
		rel := paths.RelPath(srcDir, targetPath)
		rel = filepath.ToSlash(rel)
		if input != "" && !strings.Contains(strings.ToLower(rel), strings.ToLower(input)) {
			continue
		}
		title := ""
		if t := d.Index.Title(); t != nil {
			title = t.Text
		}
		items = append(items, CompletionItem{
			Label:      rel,
			Detail:     title,
			InsertText: rel,
			Kind:       17,
			Data:       map[string]string{"kind": "conref-file"},
			TextEdit: &TextEdit{
				Range:   editRange,
				NewText: rel,
			},
		})
	}
	return items
}

// completeConkeyref offers key names from the workspace key table for
// data-conkeyref="..." attributes.
func completeConkeyref(input string, folder *workspace.Folder, editRange document.Range) []CompletionItem {
	table := keyref.BuildMergedTable(folder.MapTexts())
	var items []CompletionItem
	for _, key := range keyref.AllKeys(table) {
		if input != "" && !strings.Contains(strings.ToLower(key), strings.ToLower(input)) {
			continue
		}
		entry := table[key]
		detail := entry.Href
		if entry.Title != "" {
			detail = entry.Title + " (" + entry.Href + ")"
		}
		items = append(items, CompletionItem{
			Label:  key,
			Detail: detail,
			Kind:   18,
			Data:   map[string]string{"kind": "conkeyref"},
			TextEdit: &TextEdit{
				Range:   editRange,
				NewText: key,
			},
		})
	}
	return items
}

func completeTaskSectionHeading(input string, doc *document.Document) []CompletionItem {
	if !document.IsTaskTopic(doc) {
		return nil
	}

	existing := make(map[string]bool)
	for _, h := range doc.Index.Headings() {
		existing[strings.ToLower(h.Text)] = true
	}

	titles := []struct{ title, detail string }{
		{"Prerequisites", "prereq — before the task"},
		{"About this task", "context — background info"},
		{"Procedure", "steps marker — the list below becomes <steps>"},
		{"Verification", "result — expected outcome"},
		{"Next steps", "postreq — follow-up actions"},
	}

	var items []CompletionItem
	for _, t := range titles {
		if existing[strings.ToLower(t.title)] {
			continue
		}
		if input != "" && !strings.Contains(strings.ToLower(t.title), strings.ToLower(input)) {
			continue
		}
		items = append(items, CompletionItem{
			Label:      t.title,
			Detail:     t.detail,
			InsertText: t.title,
			Kind:       17,
		})
	}
	return items
}

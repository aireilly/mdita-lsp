package completion

import (
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/vocabulary"
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
	Kind          int // 1=text, 6=variable, 17=keyword, 18=snippet
	Documentation string
	Data          map[string]string
	TextEdit      *TextEdit
}

var yamlKeys = []string{
	"author", "source", "publisher", "permissions", "audience",
	"category", "keyword", "resourceid", "$schema",
}

func Complete(doc *document.Document, pos document.Position, folder *workspace.Folder) []CompletionItem {
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
	case PartialAttrClass:
		return completeAttrClass(pe.Input, doc, pos, pe.Range, pe.HasCloseBrace)
	case PartialBlockAttr:
		return completeBlockAttr(pe.Input, pe.Range)
	case PartialAttrOpen:
		return completeAttrOpen(pe.Input, doc, pos, pe.Range, pe.HasCloseBrace)
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
	var items []CompletionItem
	for _, h := range target.Index.Headings() {
		if inputSlug == "" || h.Slug.Contains(inputSlug) {
			items = append(items, CompletionItem{
				Label:      h.ID,
				Detail:     h.Text,
				InsertText: h.ID,
				Kind:       17,
			})
		}
	}
	return items
}

func completeKeyref(input string, doc *document.Document, folder *workspace.Folder, editRange document.Range) []CompletionItem {
	table := keyref.BuildMergedTable(folder.MapTexts())
	srcPath, _ := paths.URIToPath(doc.URI)
	srcDir := filepath.Dir(srcPath)

	var items []CompletionItem
	for _, key := range keyref.AllKeys(table) {
		if input == "" || strings.Contains(strings.ToLower(key), strings.ToLower(input)) {
			entry := table[key]
			detail := entry.Href
			if entry.Title != "" {
				detail = entry.Title + " (" + entry.Href + ")"
			}
			linkText := key
			if entry.Title != "" {
				linkText = entry.Title
			}
			href := entry.Href
			target := folder.DocBySlug(paths.SlugOf(key))
			if target != nil {
				targetPath, _ := paths.URIToPath(target.URI)
				href = filepath.ToSlash(paths.RelPath(srcDir, targetPath))
			}
			newText := "[" + linkText + "](" + href + ")"
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

func completeTaskSectionHeading(input string, doc *document.Document) []CompletionItem {
	isTask := doc.Meta != nil && doc.Meta.Schema == document.SchemaTask
	if !isTask {
		for _, e := range doc.Elements {
			if h, ok := e.(*document.Heading); ok && h.IsTitle() && h.Attributes != nil {
				for _, c := range h.Attributes.Classes {
					if c == "task" {
						isTask = true
						break
					}
				}
			}
			if isTask {
				break
			}
		}
	}
	if !isTask {
		return nil
	}

	existing := make(map[string]bool)
	for _, h := range doc.Index.Headings() {
		existing[strings.ToLower(h.Text)] = true
	}

	titles := []struct{ title, detail string }{
		{"Prerequisites", "prereq — before the task"},
		{"About this task", "context — background info"},
		{"Verification", "result — expected outcome"},
		{"Next steps", "postreq — follow-up actions"},
		{"Related information", "related-links — links after body"},
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

func completeAttrClass(input string, doc *document.Document, pos document.Position, editRange document.Range, hasCloseBrace bool) []CompletionItem {
	var items []CompletionItem

	closeSuffix := "}"
	if hasCloseBrace {
		closeSuffix = ""
	}

	isHeading := false
	lines := strings.Split(doc.Text, "\n")
	line := ""
	if pos.Line < len(lines) {
		line = lines[pos.Line]
	}
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		isHeading = true
	}

	if isHeading {
		headingClasses := []struct{ class, detail string }{
			{"task", "Task topic type"},
			{"concept", "Concept topic type"},
			{"reference", "Reference topic type"},
			{"prereq", "Task prerequisite section"},
			{"context", "Task context section"},
			{"result", "Task result section"},
			{"postreq", "Task post-requisite section"},
			{"tasktroubleshooting", "Task troubleshooting section"},
			{"related-links", "Related links section"},
		}
		for _, c := range headingClasses {
			if input == "" || strings.HasPrefix(c.class, input) {
				items = append(items, CompletionItem{
					Label:  c.class,
					Detail: c.detail,
					Kind:   6,
					TextEdit: &TextEdit{
						Range:   editRange,
						NewText: c.class + closeSuffix,
					},
				})
			}
		}
		return items
	}

	parentKind := ""
	if pos.Character > 0 && pos.Character <= len(line) {
		before := line[:pos.Character]
		if strings.Contains(before, "**") || strings.Contains(before, "__") {
			parentKind = "bold"
		} else if strings.Contains(before, "`") {
			parentKind = "code"
		} else if strings.Contains(before, "*") {
			parentKind = "italic"
		}
	}

	for _, elem := range vocabulary.AllDomainElements() {
		if parentKind != "" && elem.ParentKind != parentKind {
			continue
		}
		if input == "" || strings.HasPrefix(elem.DITAElement, input) {
			items = append(items, CompletionItem{
				Label:  elem.DITAElement,
				Detail: "<" + elem.DITAElement + "> (" + elem.Domain + ")",
				Kind:   6,
				TextEdit: &TextEdit{
					Range:   editRange,
					NewText: elem.DITAElement + closeSuffix,
				},
			})
		}
	}
	return items
}

func completeBlockAttr(input string, editRange document.Range) []CompletionItem {
	var items []CompletionItem
	for _, ca := range vocabulary.AllConditionalAttributes() {
		if input == "" || strings.HasPrefix(ca.Name, input) {
			items = append(items, CompletionItem{
				Label:  ca.Name,
				Detail: ca.Description,
				Kind:   6,
				TextEdit: &TextEdit{
					Range:   editRange,
					NewText: ca.Name + "=\"\"",
				},
			})
		}
	}
	return items
}

func completeAttrOpen(input string, doc *document.Document, pos document.Position, editRange document.Range, hasCloseBrace bool) []CompletionItem {
	var items []CompletionItem

	closeSuffix := "}"
	if hasCloseBrace {
		closeSuffix = ""
	}

	lines := strings.Split(doc.Text, "\n")
	line := ""
	if pos.Line < len(lines) {
		line = lines[pos.Line]
	}

	isHeading := strings.HasPrefix(strings.TrimSpace(line), "#")

	if isHeading {
		headingClasses := []struct{ class, detail string }{
			{"task", "Task topic type"},
			{"concept", "Concept topic type"},
			{"reference", "Reference topic type"},
			{"prereq", "Task prerequisite section"},
			{"context", "Task context section"},
			{"result", "Task result section"},
			{"postreq", "Task post-requisite section"},
			{"tasktroubleshooting", "Task troubleshooting section"},
			{"related-links", "Related links section"},
		}
		for _, c := range headingClasses {
			label := "." + c.class
			if input == "" || strings.HasPrefix(label, input) {
				items = append(items, CompletionItem{
					Label:  label,
					Detail: c.detail,
					Kind:   6,
					TextEdit: &TextEdit{
						Range:   editRange,
						NewText: label + closeSuffix,
					},
				})
			}
		}
		for _, ca := range vocabulary.AllConditionalAttributes() {
			if input == "" || strings.HasPrefix(ca.Name, input) {
				items = append(items, CompletionItem{
					Label:  ca.Name,
					Detail: ca.Description,
					Kind:   6,
					TextEdit: &TextEdit{
						Range:   editRange,
						NewText: ca.Name + "=\"\"",
					},
				})
			}
		}
		return items
	}

	parentKind := ""
	if pos.Character > 0 && pos.Character <= len(line) {
		before := line[:pos.Character]
		if strings.Contains(before, "**") || strings.Contains(before, "__") {
			parentKind = "bold"
		} else if strings.Contains(before, "`") {
			parentKind = "code"
		} else if strings.Contains(before, "*") {
			parentKind = "italic"
		}
	}

	for _, elem := range vocabulary.AllDomainElements() {
		if parentKind != "" && elem.ParentKind != parentKind {
			continue
		}
		label := "." + elem.DITAElement
		if input == "" || strings.HasPrefix(label, input) {
			items = append(items, CompletionItem{
				Label:  label,
				Detail: "<" + elem.DITAElement + "> (" + elem.Domain + ")",
				Kind:   6,
				TextEdit: &TextEdit{
					Range:   editRange,
					NewText: label + closeSuffix,
				},
			})
		}
	}
	return items
}

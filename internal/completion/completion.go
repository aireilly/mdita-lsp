package completion

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type CompletionItem struct {
	Label      string
	Detail     string
	InsertText string
	Kind       int // 1=text, 6=variable, 17=keyword, 18=snippet
}

var yamlKeys = []string{
	"author", "source", "publisher", "permissions", "audience",
	"category", "keyword", "resourceid", "$schema",
}

func Complete(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) []CompletionItem {
	pe := DetectPartial(doc.Text, pos)
	if pe == nil {
		return nil
	}

	switch pe.Kind {
	case PartialWikiLink:
		return completeWikiDoc(pe.Input, doc, folder)
	case PartialWikiHeading:
		return completeWikiHeading(pe.DocPart, pe.Input, doc, folder)
	case PartialInlineLink:
		return completeInlineDoc(pe.Input, doc, folder)
	case PartialInlineAnchor:
		return completeInlineAnchor(pe.DocPart, pe.Input, doc, folder)
	case PartialYamlKey:
		return completeYamlKey(pe.Input)
	case PartialKeyref:
		return completeKeyref(pe.Input, folder)
	}
	return nil
}

func completeWikiDoc(input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	var items []CompletionItem
	inputSlug := paths.SlugOf(input)
	for _, d := range folder.AllDocs() {
		if d.URI == doc.URI {
			continue
		}
		id := d.DocID(folder.RootURI)
		if inputSlug == "" || id.Slug.Contains(inputSlug) {
			title := ""
			if t := d.Index.Title(); t != nil {
				title = t.Text
			}
			items = append(items, CompletionItem{
				Label:      id.Stem,
				Detail:     title,
				InsertText: id.Stem,
				Kind:       17,
			})
		}
	}
	return items
}

func completeWikiHeading(docPart, input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	var target *document.Document
	if docPart == "" {
		target = doc
	} else {
		target = folder.DocBySlug(paths.SlugOf(docPart))
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

func completeInlineDoc(input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	var items []CompletionItem
	for _, d := range folder.AllDocs() {
		if d.URI == doc.URI {
			continue
		}
		id := d.DocID(folder.RootURI)
		if input == "" || strings.Contains(strings.ToLower(id.RelPath), strings.ToLower(input)) {
			items = append(items, CompletionItem{
				Label:      id.RelPath,
				Detail:     id.Stem,
				InsertText: id.RelPath,
				Kind:       17,
			})
		}
	}
	return items
}

func completeInlineAnchor(docPart, input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	return completeWikiHeading(docPart, input, doc, folder)
}

func completeKeyref(input string, folder *workspace.Folder) []CompletionItem {
	var mapTexts []string
	for _, d := range folder.AllDocs() {
		if d.Kind == document.Map {
			mapTexts = append(mapTexts, d.Text)
		}
	}
	table := keyref.BuildMergedTable(mapTexts)
	var items []CompletionItem
	for _, key := range keyref.AllKeys(table) {
		if input == "" || strings.Contains(strings.ToLower(key), strings.ToLower(input)) {
			entry := table[key]
			detail := entry.Href
			if entry.Title != "" {
				detail = entry.Title + " (" + entry.Href + ")"
			}
			items = append(items, CompletionItem{
				Label:      key,
				Detail:     detail,
				InsertText: key + "]",
				Kind:       18,
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

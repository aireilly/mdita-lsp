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

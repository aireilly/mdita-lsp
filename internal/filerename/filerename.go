package filerename

import (
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type FileRename struct {
	OldURI string
	NewURI string
}

type TextEdit struct {
	Range   document.Range
	NewText string
}

type DocumentEdit struct {
	URI   string
	Edits []TextEdit
}

func ComputeEdits(renames []FileRename, folder *workspace.Folder) []DocumentEdit {
	var all []DocumentEdit
	for _, r := range renames {
		all = append(all, computeOneRename(r, folder)...)
	}
	return all
}

func computeOneRename(r FileRename, folder *workspace.Folder) []DocumentEdit {
	oldPath, _ := paths.URIToPath(r.OldURI)
	newPath, _ := paths.URIToPath(r.NewURI)
	oldStem := stemOf(oldPath)
	newStem := stemOf(newPath)
	oldSlug := paths.SlugOf(oldStem)
	editsByURI := make(map[string][]TextEdit)

	for _, doc := range folder.AllDocs() {
		if doc.URI == r.OldURI {
			continue
		}
		for _, elem := range doc.Elements {
			switch el := elem.(type) {
			case *document.WikiLink:
				if el.Doc != "" && paths.SlugOf(el.Doc) == oldSlug {
					editsByURI[doc.URI] = append(editsByURI[doc.URI], TextEdit{
						Range:   el.Range,
						NewText: buildWikiLink(newStem, el.Heading, el.Title),
					})
				}
			case *document.MdLink:
				if matchesMdLink(el, oldPath, doc.URI) {
					newRel := computeRelPath(doc.URI, newPath)
					newURL := newRel
					if el.Anchor != "" {
						newURL += "#" + el.Anchor
					}
					editsByURI[doc.URI] = append(editsByURI[doc.URI], TextEdit{
						Range:   el.Range,
						NewText: buildMdLink(el.Text, newURL),
					})
				}
			}
		}

	}

	var result []DocumentEdit
	for uri, edits := range editsByURI {
		result = append(result, DocumentEdit{URI: uri, Edits: edits})
	}
	return result
}

func matchesMdLink(link *document.MdLink, oldPath, docURI string) bool {
	if link.URL == "" || strings.HasPrefix(link.URL, "http://") || strings.HasPrefix(link.URL, "https://") {
		return false
	}
	docPath, _ := paths.URIToPath(docURI)
	docDir := filepath.Dir(docPath)
	resolved := filepath.Clean(filepath.Join(docDir, link.URL))
	return resolved == oldPath
}

func computeRelPath(docURI, targetPath string) string {
	docPath, _ := paths.URIToPath(docURI)
	docDir := filepath.Dir(docPath)
	rel, err := filepath.Rel(docDir, targetPath)
	if err != nil {
		return targetPath
	}
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

func stemOf(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func buildWikiLink(doc, heading, title string) string {
	s := "[["
	if doc != "" {
		s += doc
	}
	if heading != "" {
		s += "#" + heading
	}
	if title != "" {
		s += "|" + title
	}
	s += "]]"
	return s
}

func buildMdLink(text, url string) string {
	return "[" + text + "](" + url + ")"
}

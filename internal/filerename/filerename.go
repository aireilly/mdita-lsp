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
	editsByURI := make(map[string][]TextEdit)

	for _, doc := range folder.AllDocs() {
		if doc.URI == r.OldURI {
			// The moved file's own relative links are now resolved from a
			// different directory, so they need rewriting too. Only incoming
			// links were handled, which broke every outgoing one on a move
			// between directories.
			editsByURI[doc.URI] = append(editsByURI[doc.URI], outgoingEdits(doc, oldPath, newPath)...)
			continue
		}
		for _, elem := range doc.Elements {
			if el, ok := elem.(*document.MdLink); ok && matchesMdLink(el, oldPath, doc.URI) {
				newURL := computeRelPath(doc.URI, newPath)
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

	var result []DocumentEdit
	for uri, edits := range editsByURI {
		if len(edits) == 0 {
			continue
		}
		result = append(result, DocumentEdit{URI: uri, Edits: edits})
	}
	return result
}

// outgoingEdits rewrites the links in the file being moved so they still
// resolve to the same targets from the new directory.
func outgoingEdits(doc *document.Document, oldPath, newPath string) []TextEdit {
	oldDir := filepath.Dir(oldPath)
	newDir := filepath.Dir(newPath)
	if oldDir == newDir {
		return nil
	}

	var edits []TextEdit
	for _, elem := range doc.Elements {
		el, ok := elem.(*document.MdLink)
		if !ok || el.URL == "" || isExternalURL(el.URL) {
			continue
		}
		target := filepath.Clean(filepath.Join(oldDir, el.URL))
		rel, err := filepath.Rel(newDir, target)
		if err != nil {
			continue
		}
		newURL := filepath.ToSlash(rel)
		if !strings.HasPrefix(newURL, ".") {
			newURL = "./" + newURL
		}
		if el.Anchor != "" {
			newURL += "#" + el.Anchor
		}
		edits = append(edits, TextEdit{
			Range:   el.Range,
			NewText: buildMdLink(el.Text, newURL),
		})
	}
	return edits
}

func isExternalURL(rawURL string) bool {
	if strings.HasPrefix(rawURL, "/") {
		return true
	}
	colon := strings.Index(rawURL, ":")
	if colon < 2 {
		return false
	}
	for i := 0; i < colon; i++ {
		c := rawURL[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') &&
			c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	c := rawURL[0]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func matchesMdLink(link *document.MdLink, oldPath, docURI string) bool {
	if link.URL == "" || isExternalURL(link.URL) {
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

func buildMdLink(text, url string) string {
	return "[" + text + "](" + url + ")"
}

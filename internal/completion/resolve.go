package completion

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func Resolve(label string, data map[string]string, folder *workspace.Folder) string {
	if data == nil {
		return ""
	}

	switch data["kind"] {
	case "wiki-doc":
		return resolveDocumentation(label, folder)
	case "keyref":
		return resolveKeyrefDocs(label, folder)
	case "heading":
		return resolveHeadingDocs(data["doc"], label, folder)
	}
	return ""
}

func resolveDocumentation(stem string, folder *workspace.Folder) string {
	slug := paths.SlugOf(stem)
	doc := folder.DocBySlug(slug)
	if doc == nil {
		return ""
	}

	var parts []string
	if t := doc.Index.Title(); t != nil {
		parts = append(parts, "**"+t.Text+"**")
	}
	if doc.Index.ShortDesc != "" {
		parts = append(parts, doc.Index.ShortDesc)
	}
	id := doc.DocID(folder.RootURI)
	parts = append(parts, "_"+id.RelPath+"_")
	return strings.Join(parts, "\n\n")
}

func resolveKeyrefDocs(key string, folder *workspace.Folder) string {
	var mapTexts []string
	for _, d := range folder.AllDocs() {
		if d.Kind == document.Map {
			mapTexts = append(mapTexts, d.Text)
		}
	}
	table := keyref.BuildMergedTable(mapTexts)
	entry, ok := table[key]
	if !ok {
		return ""
	}

	var parts []string
	if entry.Title != "" {
		parts = append(parts, "**"+entry.Title+"**")
	}
	parts = append(parts, "href: `"+entry.Href+"`")
	return strings.Join(parts, "\n\n")
}

func resolveHeadingDocs(docRef, headingID string, folder *workspace.Folder) string {
	if docRef == "" {
		return ""
	}
	slug := paths.SlugOf(docRef)
	doc := folder.DocBySlug(slug)
	if doc == nil {
		return ""
	}
	for _, h := range doc.Index.Headings() {
		if h.ID == headingID {
			return "Heading in **" + docRef + "**: " + h.Text
		}
	}
	return ""
}

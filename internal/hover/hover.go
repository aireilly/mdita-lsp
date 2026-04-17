package hover

import (
	"fmt"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/keyref"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func GetHover(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) string {
	elem := doc.ElementAt(pos)
	if elem != nil {
		switch el := elem.(type) {
		case *document.WikiLink:
			return hoverWikiLink(el, doc, folder)
		case *document.MdLink:
			return hoverMdLink(el, doc, folder)
		case *document.Heading:
			return "**" + el.Text + "** (level " + itoa(el.Level) + ")"
		}
	}

	if kr := keyref.DetectAtPosition(doc.Text, pos); kr != nil {
		return hoverKeyref(kr, folder)
	}

	return ""
}

func hoverWikiLink(wl *document.WikiLink, doc *document.Document, folder *workspace.Folder) string {
	if wl.Doc == "" && wl.Heading != "" {
		slug := paths.SlugOf(wl.Heading)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return "**" + h.Text + "**"
		}
		return ""
	}

	targetSlug := paths.SlugOf(wl.Doc)
	target := folder.DocBySlug(targetSlug)
	if target == nil {
		return ""
	}

	title := target.Index.Title()
	if title != nil {
		result := "**" + title.Text + "**"
		if wl.Heading != "" {
			hslug := paths.SlugOf(wl.Heading)
			for _, h := range target.Index.HeadingsBySlug(hslug) {
				result += " > " + h.Text
			}
		}
		return result
	}
	return ""
}

func hoverKeyref(kr *keyref.KeyrefAtPos, folder *workspace.Folder) string {
	table := keyref.BuildMergedTable(collectMapTexts(folder))
	entry, ok := keyref.Resolve(table, kr.Label)
	if !ok {
		return ""
	}
	result := "**" + kr.Label + "** (keyref)"
	if entry.Title != "" {
		result += "\n\nTarget: " + entry.Title + " (" + entry.Href + ")"
	} else {
		result += "\n\nTarget: " + entry.Href
	}
	return result
}

func collectMapTexts(folder *workspace.Folder) []string {
	var texts []string
	for _, d := range folder.AllDocs() {
		if d.Kind == document.Map {
			texts = append(texts, d.Text)
		}
	}
	return texts
}

func hoverMdLink(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) string {
	if ml.URL == "" && ml.Anchor != "" {
		slug := paths.SlugOf(ml.Anchor)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return "**" + h.Text + "**"
		}
	}
	if ml.URL != "" {
		for _, d := range folder.AllDocs() {
			id := d.DocID(folder.RootURI)
			if id.RelPath == ml.URL || id.Stem+".md" == ml.URL {
				title := d.Index.Title()
				if title != nil {
					result := "**" + title.Text + "** (" + ml.URL + ")"
					if ml.Anchor != "" {
						hslug := paths.SlugOf(ml.Anchor)
						for _, h := range d.Index.HeadingsBySlug(hslug) {
							result += " > " + h.Text
						}
					}
					return result
				}
			}
		}
	}
	return ml.URL
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

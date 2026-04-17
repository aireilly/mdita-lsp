package definition

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Location struct {
	URI   string
	Range document.Range
}

func GotoDef(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) []Location {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}

	switch el := elem.(type) {
	case *document.WikiLink:
		return resolveWikiLink(el, doc, folder, graph)
	case *document.MdLink:
		return resolveMdLink(el, doc, folder)
	}

	return nil
}

func resolveWikiLink(wl *document.WikiLink, doc *document.Document, folder *workspace.Folder, graph *symbols.Graph) []Location {
	if wl.Doc == "" && wl.Heading != "" {
		slug := paths.SlugOf(wl.Heading)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return []Location{{URI: doc.URI, Range: h.Range}}
		}
		return nil
	}

	targetSlug := paths.SlugOf(wl.Doc)
	target := folder.DocBySlug(targetSlug)
	if target == nil {
		return nil
	}

	if wl.Heading != "" {
		hslug := paths.SlugOf(wl.Heading)
		for _, h := range target.Index.HeadingsBySlug(hslug) {
			return []Location{{URI: target.URI, Range: h.Range}}
		}
	}

	title := target.Index.Title()
	if title != nil {
		return []Location{{URI: target.URI, Range: title.Range}}
	}
	return []Location{{URI: target.URI, Range: document.Rng(0, 0, 0, 0)}}
}

func resolveMdLink(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) []Location {
	if ml.URL == "" && ml.Anchor != "" {
		slug := paths.SlugOf(ml.Anchor)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return []Location{{URI: doc.URI, Range: h.Range}}
		}
		return nil
	}

	if ml.URL != "" {
		for _, d := range folder.AllDocs() {
			id := d.DocID(folder.RootURI)
			if matchesURL(id, ml.URL) {
				if ml.Anchor != "" {
					hslug := paths.SlugOf(ml.Anchor)
					for _, h := range d.Index.HeadingsBySlug(hslug) {
						return []Location{{URI: d.URI, Range: h.Range}}
					}
				}
				title := d.Index.Title()
				if title != nil {
					return []Location{{URI: d.URI, Range: title.Range}}
				}
				return []Location{{URI: d.URI, Range: document.Rng(0, 0, 0, 0)}}
			}
		}
	}

	return nil
}

func matchesURL(id paths.DocID, url string) bool {
	return id.RelPath == url || id.Stem+".md" == url || id.Stem+".markdown" == url
}

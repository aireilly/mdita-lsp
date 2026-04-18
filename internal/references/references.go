package references

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

func FindRefs(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) []Location {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}

	switch el := elem.(type) {
	case *document.Heading:
		return findHeadingRefs(el, doc, folder, graph)
	case *document.MdLink:
		return findMdLinkRefs(el, doc, folder)
	case *document.WikiLink:
		return findWikiLinkRefs(el, doc, folder)
	}
	return nil
}

func findHeadingRefs(heading *document.Heading, doc *document.Document, folder *workspace.Folder, graph *symbols.Graph) []Location {
	target := document.Symbol{
		Kind: document.DefKind,
		Slug: heading.Slug,
	}
	refs := graph.FindRefs(target)
	var locs []Location
	for _, r := range refs {
		locs = append(locs, Location{URI: r.DocURI, Range: r.Range})
	}
	if heading.IsTitle() {
		docRefs := findDocRefs(doc.URI, folder)
		locs = append(locs, docRefs...)
	}
	return locs
}

func findMdLinkRefs(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) []Location {
	target := folder.ResolveLink(ml.URL, doc.URI)
	if target == nil {
		return nil
	}
	return findDocRefs(target.URI, folder)
}

func findWikiLinkRefs(wl *document.WikiLink, doc *document.Document, folder *workspace.Folder) []Location {
	if wl.Doc == "" {
		return nil
	}
	target := folder.DocBySlug(paths.SlugOf(wl.Doc))
	if target == nil {
		return nil
	}
	return findDocRefs(target.URI, folder)
}

func findDocRefs(targetURI string, folder *workspace.Folder) []Location {
	var locs []Location
	for _, d := range folder.AllDocs() {
		for _, ml := range d.Index.MdLinks() {
			if resolved := folder.ResolveLink(ml.URL, d.URI); resolved != nil && resolved.URI == targetURI {
				locs = append(locs, Location{URI: d.URI, Range: ml.Range})
			}
		}
		for _, wl := range d.Index.WikiLinks() {
			if wl.Doc == "" {
				continue
			}
			if resolved := folder.DocBySlug(paths.SlugOf(wl.Doc)); resolved != nil && resolved.URI == targetURI {
				locs = append(locs, Location{URI: d.URI, Range: wl.Range})
			}
		}
	}
	return locs
}

func CountRefs(heading *document.Heading, graph *symbols.Graph) int {
	target := document.Symbol{Slug: heading.Slug}
	return len(graph.FindRefs(target))
}

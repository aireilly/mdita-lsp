package codelens

import (
	"fmt"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Location struct {
	URI   string
	Range document.Range
}

type Lens struct {
	Range     document.Range
	Command   string
	Title     string
	URI       string
	Position  document.Position
	Locations []Location
}

func GetLenses(doc *document.Document, graph *symbols.Graph, folder *workspace.Folder) []Lens {
	var lenses []Lens
	for _, h := range doc.Index.Headings() {
		locs := findHeadingLocs(h, doc, graph, folder)
		lenses = append(lenses, Lens{
			Range:     h.Range,
			Command:   "editor.action.showReferences",
			Title:     fmt.Sprintf("%d references", len(locs)),
			URI:       doc.URI,
			Position:  h.Range.Start,
			Locations: locs,
		})
	}
	return lenses
}

func findHeadingLocs(h *document.Heading, doc *document.Document, graph *symbols.Graph, folder *workspace.Folder) []Location {
	target := document.Symbol{Slug: h.Slug}
	refs := graph.FindRefs(target)
	var locs []Location
	for _, r := range refs {
		locs = append(locs, Location{URI: r.DocURI, Range: r.Range})
	}
	if h.IsTitle() && folder != nil {
		for _, d := range folder.AllDocs() {
			for _, ml := range d.Index.MdLinks() {
				if resolved := folder.ResolveLink(ml.URL, d.URI); resolved != nil && resolved.URI == doc.URI {
					locs = append(locs, Location{URI: d.URI, Range: ml.Range})
				}
			}
		}
	}
	return locs
}

package codelens

import (
	"fmt"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/references"
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
	var locs []Location
	for _, l := range references.FindHeadingLocations(h, doc, folder) {
		locs = append(locs, Location{URI: l.URI, Range: l.Range})
	}
	return locs
}

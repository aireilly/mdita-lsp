package codelens

import (
	"fmt"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Lens struct {
	Range    document.Range
	Command  string
	Title    string
	Position document.Position
}

func GetLenses(doc *document.Document, graph *symbols.Graph, folder *workspace.Folder) []Lens {
	var lenses []Lens
	for _, h := range doc.Index.Headings() {
		count := countHeadingRefs(h, doc, graph, folder)
		lenses = append(lenses, Lens{
			Range:    h.Range,
			Command:  "mdita-lsp.findReferences",
			Title:    fmt.Sprintf("%d references", count),
			Position: h.Range.Start,
		})
	}
	return lenses
}

func countHeadingRefs(h *document.Heading, doc *document.Document, graph *symbols.Graph, folder *workspace.Folder) int {
	target := document.Symbol{Slug: h.Slug}
	count := len(graph.FindRefs(target))
	if h.IsTitle() && folder != nil {
		count += countDocRefs(doc.URI, folder)
	}
	return count
}

func countDocRefs(targetURI string, folder *workspace.Folder) int {
	count := 0
	for _, d := range folder.AllDocs() {
		for _, ml := range d.Index.MdLinks() {
			if resolved := folder.ResolveLink(ml.URL, d.URI); resolved != nil && resolved.URI == targetURI {
				count++
			}
		}
	}
	return count
}

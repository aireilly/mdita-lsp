package references

import (
	"github.com/aireilly/mdita-lsp/internal/document"
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

	heading, ok := elem.(*document.Heading)
	if !ok {
		return nil
	}

	target := document.Symbol{
		Kind: document.DefKind,
		Slug: heading.Slug,
	}

	refs := graph.FindRefs(target)
	var locs []Location
	for _, r := range refs {
		locs = append(locs, Location{URI: r.DocURI, Range: r.Range})
	}
	return locs
}

func CountRefs(heading *document.Heading, graph *symbols.Graph) int {
	target := document.Symbol{Slug: heading.Slug}
	return len(graph.FindRefs(target))
}

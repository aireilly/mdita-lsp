package codelens

import (
	"fmt"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
)

type Lens struct {
	Range   document.Range
	Command string
	Title   string
}

func GetLenses(doc *document.Document, graph *symbols.Graph) []Lens {
	var lenses []Lens
	for _, h := range doc.Index.Headings() {
		target := document.Symbol{Slug: h.Slug}
		refs := graph.FindRefs(target)
		lenses = append(lenses, Lens{
			Range:   h.Range,
			Command: "mdita-lsp.findReferences",
			Title:   fmt.Sprintf("%d references", len(refs)),
		})
	}
	return lenses
}

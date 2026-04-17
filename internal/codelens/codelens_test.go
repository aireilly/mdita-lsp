package codelens

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
)

func TestCodeLens(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\n## Section\n")
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	lenses := GetLenses(doc, g)
	if len(lenses) < 2 {
		t.Errorf("got %d lenses, want >= 2", len(lenses))
	}
}

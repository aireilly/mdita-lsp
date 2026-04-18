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

	lenses := GetLenses(doc, g, nil)
	if len(lenses) < 2 {
		t.Errorf("got %d lenses, want >= 2", len(lenses))
	}
}

func TestCodeLensRefCount(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\nContent.\n")
	doc2 := document.New("file:///project/a.md", 1, "# A\n\n[[intro#introduction]]\n")
	doc3 := document.New("file:///project/b.md", 1, "# B\n\n[[intro#introduction]]\n")

	g := symbols.NewGraph()
	for _, d := range []*document.Document{doc1, doc2, doc3} {
		g.AddDefs(d.URI, d.Defs())
		g.AddRefs(d.URI, d.Refs())
	}

	lenses := GetLenses(doc1, g, nil)
	if len(lenses) == 0 {
		t.Fatal("expected at least one lens")
	}
	if lenses[0].Title != "2 references" {
		t.Errorf("expected '2 references', got %q", lenses[0].Title)
	}
}

func TestCodeLensNoHeadings(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "Just text.\n")
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	lenses := GetLenses(doc, g, nil)
	if len(lenses) != 0 {
		t.Errorf("expected 0 lenses for no headings, got %d", len(lenses))
	}
}

func TestCodeLensCommand(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n")
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	lenses := GetLenses(doc, g, nil)
	if len(lenses) != 1 {
		t.Fatalf("expected 1 lens, got %d", len(lenses))
	}
	if lenses[0].Command != "mdita-lsp.findReferences" {
		t.Errorf("expected findReferences command, got %q", lenses[0].Command)
	}
}

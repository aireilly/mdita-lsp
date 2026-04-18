package codelens

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
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
	doc2 := document.New("file:///project/a.md", 1, "# A\n\n[link](intro.md)\n")
	doc3 := document.New("file:///project/b.md", 1, "# B\n\n[link](intro.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	f.AddDoc(doc3)

	g := symbols.NewGraph()
	for _, d := range f.AllDocs() {
		g.AddDefs(d.URI, d.Defs())
		g.AddRefs(d.URI, d.Refs())
	}

	lenses := GetLenses(doc1, g, f)
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

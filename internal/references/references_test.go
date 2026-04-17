package references

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestFindRefsToHeading(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\nContent.\n")
	doc2 := document.New("file:///project/a.md", 1, "# A\n\n[[intro#introduction]]\n")
	doc3 := document.New("file:///project/b.md", 1, "# B\n\n[[intro#introduction]]\n")

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

	heading := doc1.Index.Title()
	locs := FindRefs(doc1, heading.Range.Start, f, g)
	if len(locs) < 2 {
		t.Errorf("FindRefs returned %d, want >= 2", len(locs))
	}
}

func TestFindRefsNoRefs(t *testing.T) {
	doc := document.New("file:///project/lonely.md", 1, "# Lonely\n\nNo refs.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	heading := doc.Index.Title()
	locs := FindRefs(doc, heading.Range.Start, f, g)
	if len(locs) != 0 {
		t.Errorf("FindRefs returned %d, want 0", len(locs))
	}
}

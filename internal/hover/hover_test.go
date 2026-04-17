package hover

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestHoverWikiLink(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\nThis is the intro.\n")
	doc2 := document.New("file:///project/guide.md", 1, "# Guide\n\n[[intro]]\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	g := symbols.NewGraph()
	g.AddDefs(doc1.URI, doc1.Defs())
	g.AddDefs(doc2.URI, doc2.Defs())

	wl := doc2.Index.WikiLinks()[0]
	result := GetHover(doc2, wl.Rng().Start, f, g)
	if result == "" {
		t.Fatal("expected hover content")
	}
	if result != "**Introduction**" {
		t.Errorf("hover = %q, want %q", result, "**Introduction**")
	}
}

func TestHoverNoElement(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()

	result := GetHover(doc, document.Position{Line: 2, Character: 3}, f, g)
	if result != "" {
		t.Errorf("expected empty hover, got %q", result)
	}
}

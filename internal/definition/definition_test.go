package definition

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func setup() *workspace.Folder {
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)

	doc1 := document.New("file:///project/intro.md", 1,
		"# Introduction\n\n## Getting Started\n\nSome text.\n")
	doc2 := document.New("file:///project/guide.md", 1,
		"# Guide\n\n[[intro]]\n\n[[intro#getting-started]]\n\n[link](intro.md#getting-started)\n")

	f.AddDoc(doc1)
	f.AddDoc(doc2)

	return f
}

func TestGotoDefWikiLinkDoc(t *testing.T) {
	f := setup()
	doc := f.DocByURI("file:///project/guide.md")

	wl := doc.Index.WikiLinks()[0]
	locs := GotoDef(doc, wl.Rng().Start, f)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations")
	}
	if locs[0].URI != "file:///project/intro.md" {
		t.Errorf("URI = %q", locs[0].URI)
	}
}

func TestGotoDefWikiLinkHeading(t *testing.T) {
	f := setup()
	doc := f.DocByURI("file:///project/guide.md")

	wl := doc.Index.WikiLinks()[1]
	locs := GotoDef(doc, wl.Rng().Start, f)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for heading wiki link")
	}
}

func TestGotoDefIntraDocHeading(t *testing.T) {
	doc := document.New("file:///project/self.md", 1,
		"# Title\n\n## Section\n\n[[#section]]\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	wl := doc.Index.WikiLinks()[0]
	locs := GotoDef(doc, wl.Rng().Start, f)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for intra-doc heading")
	}
	if locs[0].URI != doc.URI {
		t.Errorf("expected same doc, got %q", locs[0].URI)
	}
}

func TestGotoDefNoResult(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	locs := GotoDef(doc, document.Position{Line: 2, Character: 3}, f)
	if len(locs) != 0 {
		t.Errorf("expected no locations, got %d", len(locs))
	}
}

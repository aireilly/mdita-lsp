package definition

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestGotoDefMdLinkRelativePath(t *testing.T) {
	target := document.New("file:///project/docs/install.md", 1,
		"# Installation\n\n## Prerequisites\n")
	source := document.New("file:///project/guide/index.md", 1,
		"# Guide\n\n[setup](../docs/install.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(target)
	f.AddDoc(source)

	mls := source.Index.MdLinks()
	if len(mls) == 0 {
		t.Fatal("no md links found")
	}
	locs := GotoDef(source, mls[0].Rng().Start, f)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for relative md link")
	}
	if locs[0].URI != "file:///project/docs/install.md" {
		t.Errorf("expected docs/install.md, got %q", locs[0].URI)
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

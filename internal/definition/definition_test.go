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

func TestGotoDefConref(t *testing.T) {
	target := document.New("file:///project/shared.md", 1,
		"# Shared\n\n## Warning {#warning-para}\n\nContent here.\n")
	source := document.New("file:///project/doc.md", 1,
		"# Doc\n\n<p data-conref=\"shared.md#topic/warning-para\">fallback</p>\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(target)
	f.AddDoc(source)

	// Find the ConrefElement in the source document
	elem := source.ElementAt(document.Position{Line: 2, Character: 10})
	if elem == nil {
		t.Fatal("no element at conref position")
	}
	if _, ok := elem.(*document.ConrefElement); !ok {
		t.Fatalf("expected ConrefElement, got %T", elem)
	}

	locs := GotoDef(source, document.Position{Line: 2, Character: 10}, f)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for conref")
	}
	if locs[0].URI != "file:///project/shared.md" {
		t.Errorf("expected shared.md URI, got %q", locs[0].URI)
	}
}

func TestGotoDefConrefMissingFile(t *testing.T) {
	source := document.New("file:///project/doc.md", 1,
		"# Doc\n\n<p data-conref=\"missing.md#topic/note\">fallback</p>\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(source)

	locs := GotoDef(source, document.Position{Line: 2, Character: 10}, f)
	if len(locs) != 0 {
		t.Errorf("expected no locations for missing conref target, got %d", len(locs))
	}
}

func TestGotoDefDitaFragment(t *testing.T) {
	target := document.New("file:///p/install.md", 1,
		"# Install the software\n\n## Prerequisites\n\nText.\n")
	src := document.New("file:///p/doc.md", 1,
		"# Doc\n\nSee [it](install.md#install-the-software/prerequisites).\n")
	f := workspace.NewFolder("file:///p", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	locs := GotoDef(src, document.Position{Line: 2, Character: 6}, f)
	if len(locs) != 1 {
		t.Fatalf("locations = %d, want 1", len(locs))
	}
	if locs[0].URI != target.URI {
		t.Errorf("URI = %q, want %q", locs[0].URI, target.URI)
	}
	if locs[0].Range.Start.Line != 2 {
		t.Errorf("line = %d, want the Prerequisites heading on line 2", locs[0].Range.Start.Line)
	}
}

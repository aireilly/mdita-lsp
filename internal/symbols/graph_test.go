package symbols

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestGraphResolveRef(t *testing.T) {
	g := NewGraph()

	def := document.Symbol{
		Kind:    document.DefKind,
		DefType: document.DefHeading,
		Name:    "Introduction",
		Slug:    paths.SlugOf("introduction"),
		DocURI:  "file:///project/intro.md",
		Range:   document.Rng(0, 0, 0, 14),
	}
	ref := document.Symbol{
		Kind:    document.RefKind,
		RefType: document.RefMdLink,
		Name:    "introduction",
		Slug:    paths.SlugOf("introduction"),
		DocURI:  "file:///project/other.md",
		Range:   document.Rng(2, 0, 2, 12),
	}

	g.AddDefs("file:///project/intro.md", []document.Symbol{def})
	g.AddRefs("file:///project/other.md", []document.Symbol{ref})

	defs := g.ResolveRef(ref)
	if len(defs) != 1 {
		t.Fatalf("ResolveRef returned %d, want 1", len(defs))
	}
	if defs[0].Name != "Introduction" {
		t.Errorf("resolved to %q, want %q", defs[0].Name, "Introduction")
	}
}

func TestGraphFindRefs(t *testing.T) {
	g := NewGraph()

	def := document.Symbol{
		Kind:    document.DefKind,
		DefType: document.DefHeading,
		Name:    "Setup",
		Slug:    paths.SlugOf("Setup"),
		DocURI:  "file:///project/setup.md",
	}
	ref1 := document.Symbol{
		Kind:   document.RefKind,
		Name:   "setup",
		Slug:   paths.SlugOf("setup"),
		DocURI: "file:///project/a.md",
	}
	ref2 := document.Symbol{
		Kind:   document.RefKind,
		Name:   "setup",
		Slug:   paths.SlugOf("setup"),
		DocURI: "file:///project/b.md",
	}

	g.AddDefs("file:///project/setup.md", []document.Symbol{def})
	g.AddRefs("file:///project/a.md", []document.Symbol{ref1})
	g.AddRefs("file:///project/b.md", []document.Symbol{ref2})

	refs := g.FindRefs(def)
	if len(refs) != 2 {
		t.Errorf("FindRefs returned %d, want 2", len(refs))
	}
}

func TestGraphUpdateDoc(t *testing.T) {
	g := NewGraph()

	g.AddDefs("file:///project/a.md", []document.Symbol{
		{Kind: document.DefKind, Name: "Old", Slug: paths.SlugOf("Old"), DocURI: "file:///project/a.md"},
	})

	g.AddDefs("file:///project/a.md", []document.Symbol{
		{Kind: document.DefKind, Name: "New", Slug: paths.SlugOf("New"), DocURI: "file:///project/a.md"},
	})

	allDefs := g.DefsByDoc("file:///project/a.md")
	if len(allDefs) != 1 || allDefs[0].Name != "New" {
		t.Errorf("after update, defs = %v", allDefs)
	}
}

func TestGraphRemoveDoc(t *testing.T) {
	g := NewGraph()
	g.AddDefs("file:///project/a.md", []document.Symbol{
		{Kind: document.DefKind, Name: "X", Slug: paths.SlugOf("X"), DocURI: "file:///project/a.md"},
	})
	g.RemoveDoc("file:///project/a.md")
	if len(g.DefsByDoc("file:///project/a.md")) != 0 {
		t.Error("expected 0 defs after remove")
	}
}

func TestGraphDocDef(t *testing.T) {
	g := NewGraph()
	g.AddDefs("file:///project/intro.md", []document.Symbol{
		{Kind: document.DefKind, DefType: document.DefDoc, Name: "file:///project/intro.md", DocURI: "file:///project/intro.md"},
		{Kind: document.DefKind, DefType: document.DefTitle, Name: "Introduction", Slug: paths.SlugOf("Introduction"), DocURI: "file:///project/intro.md"},
	})

	defs := g.ResolveDocRef(paths.SlugOf("intro"))
	if len(defs) != 1 {
		t.Fatalf("ResolveDocRef = %d, want 1", len(defs))
	}
}

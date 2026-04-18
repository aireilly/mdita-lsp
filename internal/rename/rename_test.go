package rename

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestPrepareRename(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# My Heading\n\nText.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	heading := doc.Index.Title()
	result := Prepare(doc, heading.Range.Start)
	if result == nil {
		t.Fatal("Prepare returned nil")
	}
	if result.Text != "My Heading" {
		t.Errorf("Text = %q", result.Text)
	}
}

func TestPrepareRenameNonHeading(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nText.\n")
	result := Prepare(doc, document.Position{Line: 2, Character: 0})
	if result != nil {
		t.Error("Prepare should return nil for non-heading")
	}
}

func TestPrepareRenameLink(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\n[link text](other.md)\n")
	result := Prepare(doc, document.Position{Line: 2, Character: 1})
	if result != nil {
		t.Error("Prepare should return nil for links")
	}
}

func TestRename(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\n## Details\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	g := symbols.NewGraph()
	for _, d := range f.AllDocs() {
		g.AddDefs(d.URI, d.Defs())
		g.AddRefs(d.URI, d.Refs())
	}

	heading := doc1.Index.Headings()[1]
	edits := DoRename(doc1, heading.Range.Start, "New Details", f, g)
	if len(edits) == 0 {
		t.Fatal("DoRename returned no edits")
	}

	foundHeading := false
	for _, e := range edits {
		if e.URI == doc1.URI {
			foundHeading = true
		}
	}
	if !foundHeading {
		t.Error("missing edit for heading document")
	}
}

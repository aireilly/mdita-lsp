package document

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestNewDocument(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "# Title\n\nSome text.\n")
	if doc.URI != "file:///test/doc.md" {
		t.Errorf("URI = %q", doc.URI)
	}
	if doc.Version != 1 {
		t.Errorf("Version = %d", doc.Version)
	}
	if doc.Kind != Topic {
		t.Errorf("Kind = %v, want Topic", doc.Kind)
	}
	title := doc.Index.Title()
	if title == nil || title.Text != "Title" {
		t.Errorf("Title = %v", title)
	}
}

func TestNewMapDocument(t *testing.T) {
	doc := New("file:///test/doc.mditamap", 1, "# My Map\n\n- [Topic](topic.md)\n")
	if doc.Kind != Map {
		t.Errorf("Kind = %v, want Map", doc.Kind)
	}
}

func TestDocumentLineMap(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "line 0\nline 1\nline 2\n")
	if len(doc.Lines) != 4 {
		t.Fatalf("Lines = %d, want 4", len(doc.Lines))
	}
	if doc.Lines[0] != 0 {
		t.Errorf("Lines[0] = %d, want 0", doc.Lines[0])
	}
	if doc.Lines[1] != 7 {
		t.Errorf("Lines[1] = %d, want 7", doc.Lines[1])
	}
}

func TestDocumentApplyFullChange(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "# Old\n")
	doc = doc.ApplyChange(2, "# New\n\nContent.\n")
	if doc.Version != 2 {
		t.Errorf("Version = %d, want 2", doc.Version)
	}
	title := doc.Index.Title()
	if title == nil || title.Text != "New" {
		t.Errorf("Title after change = %v", title)
	}
}

func TestDocumentSymbols(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "# Title\n\n## Section\n\n[[other]]\n\n[link](foo.md)\n")
	defs := doc.Defs()
	refs := doc.Refs()

	if len(defs) < 2 {
		t.Errorf("Defs = %d, want >= 2 (doc + headings)", len(defs))
	}
	if len(refs) < 2 {
		t.Errorf("Refs = %d, want >= 2 (wiki link + md link)", len(refs))
	}
}

func TestDocIDFromDocument(t *testing.T) {
	doc := New("file:///project/docs/intro.md", 1, "# Intro\n")
	id := doc.DocID("file:///project")
	if id.Stem != "intro" {
		t.Errorf("Stem = %q", id.Stem)
	}
	if id.Slug != paths.Slug("intro") {
		t.Errorf("Slug = %q", id.Slug)
	}
}

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
	doc := New("file:///test/doc.md", 1, "# Title\n\n## Section\n\n[link](foo.md)\n")
	defs := doc.Defs()
	refs := doc.Refs()

	if len(defs) < 2 {
		t.Errorf("Defs = %d, want >= 2 (doc + headings)", len(defs))
	}
	if len(refs) < 1 {
		t.Errorf("Refs = %d, want >= 1 (md link)", len(refs))
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

func TestImplicitContextSection(t *testing.T) {
	// The first paragraph becomes <shortdesc>; the second one is the implicit
	// <context>.
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\nShort description.\n\nSome context paragraph.\n\n1. Step one\n2. Step two\n"
	doc := New("file:///test.md", 1, text)
	found := false
	for _, s := range doc.ImplicitSections {
		if s.Kind == ImplicitContext {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected implicit context section before steps")
	}
}

func TestImplicitResultSection(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\n1. Step one\n2. Step two\n\nThe software is now installed.\n"
	doc := New("file:///test.md", 1, text)
	found := false
	for _, s := range doc.ImplicitSections {
		if s.Kind == ImplicitResult {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected implicit result section after steps")
	}
}

// IMPLICIT_CHOICES defaults to false and plugin.xml does not enable it, so a
// nested unordered list becomes <substeps>, the same as a nested ordered one.
func TestNestedUnorderedListIsSubsteps(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\n1. Choose an option:\n   - Option A\n   - Option B\n2. Continue.\n"
	doc := New("file:///test.md", 1, text)
	for _, s := range doc.ImplicitSections {
		if s.Kind == ImplicitChoices {
			t.Error("a nested unordered list must not be <choices> without {.choices}")
		}
	}
	if !hasSection(doc, ImplicitSubsteps) {
		t.Error("expected <substeps> for the nested unordered list")
	}
}

// The {.choices} outputclass is what makes it <choices>.
func TestChoicesOutputclassSection(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\n1. Choose an option:\n   - Option A\n   - Option B\n   {.choices}\n2. Continue.\n"
	doc := New("file:///test.md", 1, text)
	if !hasSection(doc, ImplicitChoices) {
		t.Errorf("expected <choices>, got %+v", doc.ImplicitSections)
	}
}

func hasSection(doc *Document, kind ImplicitSectionKind) bool {
	for _, s := range doc.ImplicitSections {
		if s.Kind == kind {
			return true
		}
	}
	return false
}

func TestImplicitSubstepsSection(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\n1. Do the following:\n   1. Sub-step A\n   2. Sub-step B\n2. Continue.\n"
	doc := New("file:///test.md", 1, text)
	found := false
	for _, s := range doc.ImplicitSections {
		if s.Kind == ImplicitSubsteps {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected implicit substeps section inside step")
	}
}

// IMPLICIT_CHOICETABLE defaults to false, so a table in a step stays a plain
// <table> until it carries {.choicetable}.
func TestNestedTableIsNotChoicetableByDefault(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\n1. Choose a plan:\n\n   | Plan | Price |\n   |------|-------|\n   | Basic | Free |\n\n2. Continue.\n"
	doc := New("file:///test.md", 1, text)
	if hasSection(doc, ImplicitChoicetable) {
		t.Error("a nested table must not be <choicetable> without {.choicetable}")
	}
}

func TestChoicetableOutputclassSection(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\n1. Choose a plan:\n\n   | Plan | Price |\n   |------|-------|\n   | Basic | Free |\n   {.choicetable}\n\n2. Continue.\n"
	doc := New("file:///test.md", 1, text)
	if !hasSection(doc, ImplicitChoicetable) {
		t.Errorf("expected <choicetable>, got %+v", doc.ImplicitSections)
	}
}

func TestImplicitSectionsNonTask(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:concept.xsd\n---\n# A concept\n\nSome paragraph.\n\n1. Item one\n"
	doc := New("file:///test.md", 1, text)
	if len(doc.ImplicitSections) > 0 {
		t.Errorf("expected no implicit sections for non-task topic, got %d", len(doc.ImplicitSections))
	}
}

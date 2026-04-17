package linkededit

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestLinkedEditingHeading(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n## Section\n\nSee [[#Section]].\n")

	headings := doc.Index.Headings()
	if len(headings) < 2 {
		t.Fatalf("expected at least 2 headings, got %d", len(headings))
	}

	result := GetLinkedRanges(doc, headings[1].Range.Start)
	if result == nil {
		t.Fatal("expected linked editing ranges for heading with intra-doc ref")
	}
	if len(result.Ranges) < 2 {
		t.Errorf("expected at least 2 ranges, got %d", len(result.Ranges))
	}
}

func TestLinkedEditingNoRefs(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n## Section\n\nPlain text.\n")

	headings := doc.Index.Headings()
	result := GetLinkedRanges(doc, headings[0].Range.Start)
	if result != nil {
		t.Error("expected nil for heading with no intra-doc refs")
	}
}

func TestLinkedEditingNonHeading(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\nPlain text.\n")

	result := GetLinkedRanges(doc, document.Position{Line: 2, Character: 3})
	if result != nil {
		t.Error("expected nil for non-heading position")
	}
}

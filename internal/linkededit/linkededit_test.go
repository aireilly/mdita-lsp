package linkededit

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestLinkedEditingHeading(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n## Section\n\nPlain text.\n")

	headings := doc.Index.Headings()
	if len(headings) < 2 {
		t.Fatalf("expected at least 2 headings, got %d", len(headings))
	}

	result := GetLinkedRanges(doc, headings[1].Range.Start)
	if result == nil {
		t.Fatal("expected linked editing ranges for heading")
	}
	if len(result.Ranges) != 1 {
		t.Errorf("expected 1 range, got %d", len(result.Ranges))
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

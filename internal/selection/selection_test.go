package selection

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestSelectionRangeInHeading(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\nParagraph.\n\n## Section\n\nMore text.\n")
	positions := []document.Position{{Line: 0, Character: 3}}
	ranges := GetRanges(doc, positions)

	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	sr := ranges[0]
	if sr.Range.Start.Line != 0 {
		t.Errorf("expected start line 0, got %d", sr.Range.Start.Line)
	}
}

func TestSelectionRangeInSection(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\nParagraph.\n\n## Section\n\nSection text.\n")
	positions := []document.Position{{Line: 6, Character: 3}}
	ranges := GetRanges(doc, positions)

	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	sr := ranges[0]
	if sr.Parent == nil {
		t.Fatal("expected parent selection range for section content")
	}
}

func TestSelectionRangeMultiplePositions(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\nLine one.\n\nLine two.\n")
	positions := []document.Position{
		{Line: 2, Character: 0},
		{Line: 4, Character: 0},
	}
	ranges := GetRanges(doc, positions)

	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}
}

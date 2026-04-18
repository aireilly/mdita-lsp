package highlight

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestHighlightHeading(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n## Section\n\nPlain text.\n")

	highlights := GetHighlights(doc, document.Position{Line: 2, Character: 3})
	if len(highlights) != 1 {
		t.Fatalf("expected 1 highlight (heading only), got %d", len(highlights))
	}
	if highlights[0].Kind != KindWrite {
		t.Errorf("heading should be KindWrite, got %d", highlights[0].Kind)
	}
}

func TestHighlightNonElement(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\nPlain text.\n")

	highlights := GetHighlights(doc, document.Position{Line: 2, Character: 3})
	if highlights != nil {
		t.Errorf("expected nil for plain text, got %d highlights", len(highlights))
	}
}

func TestHighlightHeadingNoRefs(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n## Section\n\nPlain text.\n")

	highlights := GetHighlights(doc, document.Position{Line: 2, Character: 3})
	if len(highlights) != 1 {
		t.Errorf("expected 1 highlight (heading only), got %d", len(highlights))
	}
}

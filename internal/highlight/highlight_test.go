package highlight

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestHighlightHeading(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n## Section\n\nSee [[#Section]].\n")

	highlights := GetHighlights(doc, document.Position{Line: 2, Character: 3})
	if len(highlights) < 2 {
		t.Fatalf("expected at least 2 highlights, got %d", len(highlights))
	}
	if highlights[0].Kind != KindWrite {
		t.Errorf("heading should be KindWrite, got %d", highlights[0].Kind)
	}
	if highlights[1].Kind != KindRead {
		t.Errorf("reference should be KindRead, got %d", highlights[1].Kind)
	}
}

func TestHighlightWikiLinkIntraDoc(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n## Section\n\nSee [[#Section]].\n")

	highlights := GetHighlights(doc, document.Position{Line: 4, Character: 6})
	if len(highlights) < 2 {
		t.Fatalf("expected at least 2 highlights (heading + ref), got %d", len(highlights))
	}
}

func TestHighlightWikiLinkCrossDoc(t *testing.T) {
	doc := document.New("file:///test/doc.md", 1,
		"# Title\n\n[[other]] and [[other]] again.\n")

	highlights := GetHighlights(doc, document.Position{Line: 2, Character: 3})
	if len(highlights) != 2 {
		t.Errorf("expected 2 highlights for same doc ref, got %d", len(highlights))
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

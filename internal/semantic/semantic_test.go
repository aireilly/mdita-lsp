package semantic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestSemanticTokensEmpty(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nNo links.\n")
	data := Encode(doc)
	if len(data) != 0 {
		t.Errorf("expected empty data for no links, got %d", len(data))
	}
}

func TestSemanticTokensNoWikiLinks(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\n[link](other.md)\n")
	data := Encode(doc)
	if len(data) != 0 {
		t.Errorf("expected empty data (no semantic tokens for md links), got %d", len(data))
	}
}

func TestEncodeRangeEmpty(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	rng := document.Rng(0, 0, 4, 0)
	data := EncodeRange(doc, rng)
	if len(data) != 0 {
		t.Errorf("expected empty range data, got %d", len(data))
	}
}

func TestTokenTypes(t *testing.T) {
	if len(TokenTypes) == 0 {
		t.Error("TokenTypes should not be empty")
	}
}

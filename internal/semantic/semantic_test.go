package semantic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestSemanticTokens(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\n[[other]]\n")
	data := Encode(doc)
	if len(data) == 0 {
		t.Error("expected semantic token data")
	}
	if len(data)%5 != 0 {
		t.Errorf("data length %d not a multiple of 5", len(data))
	}
}

func TestSemanticTokensEmpty(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nNo links.\n")
	data := Encode(doc)
	if len(data) != 0 {
		t.Errorf("expected empty data for no links, got %d", len(data))
	}
}

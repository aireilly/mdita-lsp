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

func TestSemanticTokensMultipleLinks(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n[[alpha]] and [[beta]]\n")
	data := Encode(doc)
	if len(data) != 10 {
		t.Errorf("expected 10 values (2 tokens * 5), got %d", len(data))
	}
}

func TestSemanticTokensDeltaEncoding(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n[[a]]\n\n[[b]]\n")
	data := Encode(doc)
	if len(data) < 10 {
		t.Fatalf("expected at least 10 values, got %d", len(data))
	}
	if data[5] != 2 {
		t.Errorf("expected delta line 2 for second token, got %d", data[5])
	}
}

func TestEncodeRange(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n[[a]]\n\n[[b]]\n\n[[c]]\n")
	rng := document.Rng(2, 0, 4, 0)
	data := EncodeRange(doc, rng)
	if len(data) != 10 {
		t.Errorf("expected 10 values (2 tokens in range), got %d", len(data))
	}
}

func TestTokenTypes(t *testing.T) {
	if len(TokenTypes) == 0 {
		t.Error("TokenTypes should not be empty")
	}
}

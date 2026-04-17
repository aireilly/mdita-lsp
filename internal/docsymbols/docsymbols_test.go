package docsymbols

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestDocumentSymbols(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n\n### Sub\n\n## Other\n")
	syms := GetSymbols(doc)

	if len(syms) == 0 {
		t.Fatal("expected symbols")
	}
	if syms[0].Name != "Title" {
		t.Errorf("syms[0].Name = %q", syms[0].Name)
	}
}

func TestDocumentSymbolsNesting(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n\n### Sub\n\n## Other\n")
	syms := GetSymbols(doc)

	if len(syms) != 1 {
		t.Fatalf("expected 1 root symbol, got %d", len(syms))
	}
	if syms[0].Kind != 5 {
		t.Errorf("title kind = %d, want 5 (class)", syms[0].Kind)
	}
	if len(syms[0].Children) != 2 {
		t.Errorf("expected 2 children of title, got %d", len(syms[0].Children))
	}
	if len(syms[0].Children) >= 1 && len(syms[0].Children[0].Children) != 1 {
		t.Errorf("Section should have 1 child (Sub), got %d", len(syms[0].Children[0].Children))
	}
}

func TestDocumentSymbolsNoHeadings(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "Just text.\n")
	syms := GetSymbols(doc)
	if syms != nil {
		t.Errorf("expected nil for no headings, got %v", syms)
	}
}

func TestWorkspaceSymbols(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/a.md", 1, "# Alpha\n"),
		document.New("file:///project/b.md", 1, "# Beta\n"),
	}
	syms := SearchWorkspace(docs, "alp")
	if len(syms) != 1 || syms[0].Name != "Alpha" {
		t.Errorf("SearchWorkspace = %v", syms)
	}
}

func TestWorkspaceSymbolsNoMatch(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/a.md", 1, "# Alpha\n"),
	}
	syms := SearchWorkspace(docs, "xyz")
	if len(syms) != 0 {
		t.Errorf("expected 0 matches, got %d", len(syms))
	}
}

func TestWorkspaceSymbolsCaseInsensitive(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/a.md", 1, "# Alpha\n"),
	}
	syms := SearchWorkspace(docs, "ALPHA")
	if len(syms) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(syms))
	}
}

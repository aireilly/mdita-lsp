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

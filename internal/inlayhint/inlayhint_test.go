package inlayhint

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func testFolder(docs ...*document.Document) *workspace.Folder {
	f := workspace.NewFolder("file:///project", config.Default())
	for _, d := range docs {
		f.AddDoc(d)
	}
	return f
}

func fullRange() document.Range {
	return document.Rng(0, 0, 100, 0)
}

func TestWikiLinkHint(t *testing.T) {
	target := document.New("file:///project/install.md", 1,
		"# Installation Guide\n\nContent.\n")
	source := document.New("file:///project/index.md", 1,
		"# Index\n\nSee [[install]] for details.\n")
	folder := testFolder(target, source)

	hints := GetHints(source, fullRange(), folder)
	found := false
	for _, h := range hints {
		if h.Label == " → Installation Guide" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected wiki link hint with title, got %v", hints)
	}
}

func TestKeyrefHint(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install Guide](install.md)\n")
	source := document.New("file:///project/doc.md", 1,
		"# Doc\n\nSee [install] for setup steps.\n")
	folder := testFolder(mapDoc, source)

	hints := GetHints(source, fullRange(), folder)
	found := false
	for _, h := range hints {
		if h.Label == " → Install Guide" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected keyref hint with title, got %v", hints)
	}
}

func TestNoHintsForPlainText(t *testing.T) {
	source := document.New("file:///project/doc.md", 1,
		"# Title\n\nJust plain text.\n")
	folder := testFolder(source)

	hints := GetHints(source, fullRange(), folder)
	if len(hints) != 0 {
		t.Errorf("expected no hints, got %d", len(hints))
	}
}

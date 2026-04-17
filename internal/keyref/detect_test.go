package keyref

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestDetectAtPosition(t *testing.T) {
	text := "See [install] for details."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 6})
	if kr == nil {
		t.Fatal("expected keyref at position")
	}
	if kr.Label != "install" {
		t.Errorf("expected label 'install', got %q", kr.Label)
	}
}

func TestDetectAtPositionOutside(t *testing.T) {
	text := "See [install] for details."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 0})
	if kr != nil {
		t.Error("expected no keyref outside brackets")
	}
}

func TestDetectIgnoresWikiLinks(t *testing.T) {
	text := "See [[install]] for details."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 6})
	if kr != nil {
		t.Error("expected no keyref for wiki link syntax")
	}
}

func TestDetectIgnoresFootnotes(t *testing.T) {
	text := "See [^note] for details."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 6})
	if kr != nil {
		t.Error("expected no keyref for footnote")
	}
}

func TestBuildMergedTable(t *testing.T) {
	mapTexts := []string{
		"# Map\n\n- [Install](install.md)\n- [Guide](guide.md)\n",
	}
	table := BuildMergedTable(mapTexts)
	if len(table) != 2 {
		t.Errorf("expected 2 keys, got %d", len(table))
	}
	if _, ok := table["install"]; !ok {
		t.Error("expected 'install' key")
	}
	if _, ok := table["guide"]; !ok {
		t.Error("expected 'guide' key")
	}
}

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

func TestDetectAllDoubleCurly(t *testing.T) {
	text := "# About {{product-name}}\n\nThe version is {{version}}.\n"
	locs := DetectAll(text)
	var dcKeys []string
	for _, l := range locs {
		dcKeys = append(dcKeys, l.Key)
	}
	if len(dcKeys) < 2 {
		t.Fatalf("expected at least 2 keyrefs, got %d: %v", len(dcKeys), dcKeys)
	}
	found := map[string]bool{}
	for _, k := range dcKeys {
		found[k] = true
	}
	if !found["product-name"] {
		t.Error("expected 'product-name' key")
	}
	if !found["version"] {
		t.Error("expected 'version' key")
	}
}

func TestDetectAllDoubleCurlySkipsCodeBlock(t *testing.T) {
	text := "# Title\n\n```\n{{template-var}}\n```\n\n{{real-key}} here.\n"
	locs := DetectAll(text)
	for _, l := range locs {
		if l.Key == "template-var" {
			t.Error("should not detect {{key}} inside fenced code block")
		}
	}
	found := false
	for _, l := range locs {
		if l.Key == "real-key" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'real-key' outside code block")
	}
}

func TestDetectAllDoubleCurlySkipsInlineCode(t *testing.T) {
	text := "Use `{{template}}` for templates.\n"
	locs := DetectAll(text)
	for _, l := range locs {
		if l.Key == "template" {
			t.Error("should not detect {{key}} inside inline code")
		}
	}
}

func TestDetectAtPositionDoubleCurly(t *testing.T) {
	text := "Install {{product-name}} now."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 12})
	if kr == nil {
		t.Fatal("expected keyref at position")
	}
	if kr.Label != "product-name" {
		t.Errorf("Label = %q, want %q", kr.Label, "product-name")
	}
	if kr.Range.Start.Character != 8 {
		t.Errorf("Range.Start.Character = %d, want 8", kr.Range.Start.Character)
	}
	if kr.Range.End.Character != 24 {
		t.Errorf("Range.End.Character = %d, want 24", kr.Range.End.Character)
	}
}

func TestDetectAtPositionDoubleCurlyOutside(t *testing.T) {
	text := "Install {{product-name}} now."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 2})
	if kr != nil {
		t.Error("expected no keyref outside {{}} range")
	}
}

func TestDetectDoubleCurlyInvalidKey(t *testing.T) {
	text := "See {{invalid key}} here.\n"
	locs := DetectAll(text)
	for _, l := range locs {
		if l.Key == "invalid key" {
			t.Error("should not detect key with spaces")
		}
	}
}

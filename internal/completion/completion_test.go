package completion

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestDetectPartialYamlKey(t *testing.T) {
	text := "---\naut"
	pe := DetectPartial(text, document.Position{Line: 1, Character: 3})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialYamlKey {
		t.Errorf("Kind = %v, want PartialYamlKey", pe.Kind)
	}
}

func TestDetectPartialKeyref(t *testing.T) {
	text := "# Title\n\nSee [inst"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 9})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialKeyref {
		t.Errorf("Kind = %v, want PartialKeyref", pe.Kind)
	}
	if pe.Input != "inst" {
		t.Errorf("Input = %q, want %q", pe.Input, "inst")
	}
}

func TestDetectNoPartial(t *testing.T) {
	text := "# Title\n\nPlain text."
	pe := DetectPartial(text, document.Position{Line: 2, Character: 5})
	if pe != nil {
		t.Errorf("expected nil partial, got %v", pe.Kind)
	}
}

func TestCompleteYamlKey(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "---\naut")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	items := Complete(doc, document.Position{Line: 1, Character: 3}, f)
	found := false
	for _, item := range items {
		if item.Label == "author" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'author' in YAML key completions")
	}
}

func TestCompleteInlineLinkRelativePath(t *testing.T) {
	doc1 := document.New("file:///project/docs/intro.md", 1, "# Introduction\n")
	doc2 := document.New("file:///project/guide/user.md", 1, "# User Guide\n\n[link](")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	items := Complete(doc2, document.Position{Line: 2, Character: 7}, f)
	if len(items) == 0 {
		t.Fatal("expected completion items for inline link")
	}
	found := false
	for _, item := range items {
		if item.InsertText == "../docs/intro.md" {
			found = true
		}
	}
	if !found {
		labels := make([]string, len(items))
		for i, item := range items {
			labels[i] = item.InsertText
		}
		t.Errorf("expected '../docs/intro.md' in completions, got %v", labels)
	}
}

func TestCompleteInlineLinkSameDir(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n")
	doc2 := document.New("file:///project/doc.md", 1, "# Doc\n\n[see](")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	items := Complete(doc2, document.Position{Line: 2, Character: 6}, f)
	found := false
	for _, item := range items {
		if item.InsertText == "intro.md" {
			found = true
		}
	}
	if !found {
		labels := make([]string, len(items))
		for i, item := range items {
			labels[i] = item.InsertText
		}
		t.Errorf("expected 'intro.md' in completions, got %v", labels)
	}
}

func TestCompleteKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install Guide](install.md)\n- [Config](config.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nSee [inst")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)
	items := Complete(topicDoc, document.Position{Line: 2, Character: 9}, f)
	found := false
	for _, item := range items {
		if item.Label == "install" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'install' keyref completion, got %v", items)
	}
}

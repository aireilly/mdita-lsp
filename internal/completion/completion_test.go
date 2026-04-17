package completion

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestDetectPartialWikiLink(t *testing.T) {
	text := "# Title\n\n[[int"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 5})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialWikiLink {
		t.Errorf("Kind = %v, want PartialWikiLink", pe.Kind)
	}
	if pe.Input != "int" {
		t.Errorf("Input = %q, want %q", pe.Input, "int")
	}
}

func TestDetectPartialWikiHeading(t *testing.T) {
	text := "# Title\n\n[[doc#sec"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 9})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialWikiHeading {
		t.Errorf("Kind = %v, want PartialWikiHeading", pe.Kind)
	}
	if pe.DocPart != "doc" {
		t.Errorf("DocPart = %q", pe.DocPart)
	}
	if pe.Input != "sec" {
		t.Errorf("Input = %q", pe.Input)
	}
}

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

func TestCompleteWikiDoc(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n")
	doc2 := document.New("file:///project/guide.md", 1, "# Guide\n\n[[int")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	g := symbols.NewGraph()
	g.AddDefs(doc1.URI, doc1.Defs())

	items := Complete(doc2, document.Position{Line: 2, Character: 5}, f, g)
	if len(items) == 0 {
		t.Fatal("expected completion items")
	}
	found := false
	for _, item := range items {
		if item.Label == "intro" || item.Label == "Introduction" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'intro' in completions, got %v", items)
	}
}

func TestCompleteYamlKey(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "---\naut")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()

	items := Complete(doc, document.Position{Line: 1, Character: 3}, f, g)
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

func TestCompleteKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install Guide](install.md)\n- [Config](config.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nSee [inst")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)
	g := symbols.NewGraph()

	items := Complete(topicDoc, document.Position{Line: 2, Character: 9}, f, g)
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

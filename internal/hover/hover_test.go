package hover

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestHoverWikiLink(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\nThis is the intro.\n")
	doc2 := document.New("file:///project/guide.md", 1, "# Guide\n\n[[intro]]\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
wl := doc2.Index.WikiLinks()[0]
	result := GetHover(doc2, wl.Rng().Start, f)
	if result == "" {
		t.Fatal("expected hover content")
	}
	if result != "**Introduction**" {
		t.Errorf("hover = %q, want %q", result, "**Introduction**")
	}
}

func TestHoverKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install Guide](install.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nSee [install] for details.\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)
result := GetHover(topicDoc, document.Position{Line: 2, Character: 6}, f)
	if result == "" {
		t.Fatal("expected hover content for keyref")
	}
	if !strings.Contains(result, "install") {
		t.Errorf("hover = %q, expected to contain 'install'", result)
	}
}

func TestHoverYAMLKey(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\nauthor: Jane Doe\n$schema: urn:oasis:names:tc:mdita:xsd:topic.xsd\n---\n# Title\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
result := GetHover(doc, document.Position{Line: 1, Character: 2}, f)
	if !strings.Contains(result, "author") {
		t.Errorf("expected hover for 'author', got %q", result)
	}
	if !strings.Contains(result, "Jane Doe") {
		t.Errorf("expected current value in hover, got %q", result)
	}
}

func TestHoverYAMLSchema(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\n$schema: urn:oasis:names:tc:mdita:xsd:topic.xsd\n---\n# Title\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
result := GetHover(doc, document.Position{Line: 1, Character: 3}, f)
	if !strings.Contains(result, "$schema") {
		t.Errorf("expected hover for '$schema', got %q", result)
	}
	if !strings.Contains(result, "DITA topic type") {
		t.Errorf("expected schema description, got %q", result)
	}
}

func TestHoverMdLinkRelativePath(t *testing.T) {
	target := document.New("file:///project/docs/install.md", 1,
		"# Installation Guide\n")
	source := document.New("file:///project/guide/index.md", 1,
		"# Guide\n\n[setup](../docs/install.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(target)
	f.AddDoc(source)

	mls := source.Index.MdLinks()
	if len(mls) == 0 {
		t.Fatal("no md links found")
	}
	result := GetHover(source, mls[0].Rng().Start, f)
	if !strings.Contains(result, "Installation Guide") {
		t.Errorf("expected hover with title, got %q", result)
	}
}

func TestHoverNoElement(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
result := GetHover(doc, document.Position{Line: 2, Character: 3}, f)
	if result != "" {
		t.Errorf("expected empty hover, got %q", result)
	}
}

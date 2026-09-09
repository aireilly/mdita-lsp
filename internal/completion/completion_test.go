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

func TestCompleteYamlKeyID(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "---\ni")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	items := Complete(doc, document.Position{Line: 1, Character: 1}, f)
	found := false
	for _, item := range items {
		if item.Label == "id" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'id' in YAML key completions")
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
		"# Map\n\n- [Install Guide](install.md)\n- [Config](config.md)\n\n[install]: install.md \"Install Guide\"\n[config]: config.md\n")
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

func TestDetectPartialConref(t *testing.T) {
	text := `# Title` + "\n\n" + `<p data-conref="shar`
	col := len(`<p data-conref="shar`)
	pe := DetectPartial(text, document.Position{Line: 2, Character: col})
	if pe == nil {
		t.Fatal("expected partial element for data-conref")
	}
	if pe.Kind != PartialConref {
		t.Errorf("Kind = %v, want PartialConref", pe.Kind)
	}
	if pe.Input != "shar" {
		t.Errorf("Input = %q, want %q", pe.Input, "shar")
	}
}

func TestDetectPartialConkeyref(t *testing.T) {
	text := `# Title` + "\n\n" + `<span data-conkeyref="warn`
	col := len(`<span data-conkeyref="warn`)
	pe := DetectPartial(text, document.Position{Line: 2, Character: col})
	if pe == nil {
		t.Fatal("expected partial element for data-conkeyref")
	}
	if pe.Kind != PartialConkeyref {
		t.Errorf("Kind = %v, want PartialConkeyref", pe.Kind)
	}
	if pe.Input != "warn" {
		t.Errorf("Input = %q, want %q", pe.Input, "warn")
	}
}

func TestCompleteConref(t *testing.T) {
	sharedDoc := document.New("file:///project/shared.md", 1, "# Shared\n\n## Warning\n")
	topicDoc := document.New("file:///project/doc.md", 1, `# Doc`+"\n\n"+`<p data-conref="shar`+"\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(sharedDoc)
	f.AddDoc(topicDoc)

	col := len(`<p data-conref="shar`)
	items := Complete(topicDoc, document.Position{Line: 2, Character: col}, f)
	found := false
	for _, item := range items {
		if item.Label == "shared.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'shared.md' conref completion, got %v", items)
	}
}

func TestCompleteConkeyref(t *testing.T) {
	// "warnings" is defined by a reference-style link definition, which the
	// plug-in renders as <keydef keys="warnings" href="warnings.md"/>.
	mapDoc := document.New("file:///project/map.mditamap", 1, "# Map\n\n- [Warnings](warnings.md)\n\n[warnings]: warnings.md\n")
	topicDoc := document.New("file:///project/doc.md", 1, `# Doc`+"\n\n"+`<span data-conkeyref="warn`+"\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)

	col := len(`<span data-conkeyref="warn`)
	items := Complete(topicDoc, document.Position{Line: 2, Character: col}, f)
	found := false
	for _, item := range items {
		if item.Label == "warnings" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'warnings' conkeyref completion, got %v", items)
	}
}

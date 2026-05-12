package completion

import (
	"strings"
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

func TestDetectPartialAttrContext(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      document.Position
		wantKind *PartialKind
		wantIn   string
	}{
		{
			name:     "bare { after bold",
			text:     "# Title\n\nClick **OK**{",
			pos:      document.Position{Line: 2, Character: 16},
			wantKind: ptrKind(PartialAttrOpen),
			wantIn:   "",
		},
		{
			name:     "bare { after code",
			text:     "# Title\n\nRun `cmd`{",
			pos:      document.Position{Line: 2, Character: 13},
			wantKind: ptrKind(PartialAttrOpen),
			wantIn:   "",
		},
		{
			name:     "bare { after italic",
			text:     "# Title\n\nSee *note*{",
			pos:      document.Position{Line: 2, Character: 14},
			wantKind: ptrKind(PartialAttrOpen),
			wantIn:   "",
		},
		{
			name:     "bare { in heading",
			text:     "# My Topic {",
			pos:      document.Position{Line: 0, Character: 12},
			wantKind: ptrKind(PartialAttrOpen),
			wantIn:   "",
		},
		{
			name:     "bare { with partial input triggers AttrClass",
			text:     "# Title\n\nClick **OK**{.ui",
			pos:      document.Position{Line: 2, Character: 19},
			wantKind: ptrKind(PartialAttrClass),
			wantIn:   "ui",
		},
		{
			name:     "bare { in plain text not detected",
			text:     "# Title\n\nUse {braces",
			pos:      document.Position{Line: 2, Character: 14},
			wantKind: nil,
			wantIn:   "",
		},
		{
			name:     "standalone { is block attr",
			text:     "# Title\n\n{aud",
			pos:      document.Position{Line: 2, Character: 4},
			wantKind: ptrKind(PartialBlockAttr),
			wantIn:   "aud",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := DetectPartial(tt.text, tt.pos)
			if tt.wantKind == nil {
				if pe != nil {
					t.Errorf("expected nil partial, got kind=%v", pe.Kind)
				}
				return
			}
			if pe == nil {
				t.Fatal("expected partial element, got nil")
			}
			if pe.Kind != *tt.wantKind {
				t.Errorf("Kind = %v, want %v", pe.Kind, *tt.wantKind)
			}
			if pe.Input != tt.wantIn {
				t.Errorf("Input = %q, want %q", pe.Input, tt.wantIn)
			}
		})
	}
}

func TestDetectPartialAttrRange(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		pos            document.Position
		wantStart      int
		wantEnd        int
		wantCloseBrace bool
	}{
		{
			name:           "AttrOpen range starts after {",
			text:           "# Title\n\nClick **OK**{",
			pos:            document.Position{Line: 2, Character: 13},
			wantStart:      13,
			wantEnd:        13,
			wantCloseBrace: false,
		},
		{
			name:           "AttrOpen range with auto-closed }",
			text:           "# Title\n\nClick **OK**{}",
			pos:            document.Position{Line: 2, Character: 13},
			wantStart:      13,
			wantEnd:        13,
			wantCloseBrace: true,
		},
		{
			name:           "AttrClass range starts after {.",
			text:           "# Title\n\nClick **OK**{.ui",
			pos:            document.Position{Line: 2, Character: 16},
			wantStart:      14,
			wantEnd:        16,
			wantCloseBrace: false,
		},
		{
			name:           "AttrClass range ends at cursor when auto-closed } present",
			text:           "# Title\n\nClick **OK**{.ui}",
			pos:            document.Position{Line: 2, Character: 16},
			wantStart:      14,
			wantEnd:        16,
			wantCloseBrace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := DetectPartial(tt.text, tt.pos)
			if pe == nil {
				t.Fatal("expected partial element, got nil")
			}
			if pe.Range.Start.Character != tt.wantStart {
				t.Errorf("Range.Start.Character = %d, want %d", pe.Range.Start.Character, tt.wantStart)
			}
			if pe.Range.End.Character != tt.wantEnd {
				t.Errorf("Range.End.Character = %d, want %d", pe.Range.End.Character, tt.wantEnd)
			}
			if pe.HasCloseBrace != tt.wantCloseBrace {
				t.Errorf("HasCloseBrace = %v, want %v", pe.HasCloseBrace, tt.wantCloseBrace)
			}
		})
	}
}

func TestCompleteAttrOpenInsertsWithoutBrace(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nClick **OK**{")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	items := Complete(doc, document.Position{Line: 2, Character: 13}, f)
	if len(items) == 0 {
		t.Fatal("expected completion items")
	}
	for _, item := range items {
		if item.TextEdit == nil {
			t.Errorf("item %q has nil TextEdit", item.Label)
			continue
		}
		if strings.HasPrefix(item.TextEdit.NewText, "{") {
			t.Errorf("item %q NewText = %q starts with { — should not include opening brace",
				item.Label, item.TextEdit.NewText)
		}
		if strings.Contains(item.TextEdit.NewText, "\n") {
			t.Errorf("item %q NewText = %q contains newline", item.Label, item.TextEdit.NewText)
		}
	}
	found := false
	for _, item := range items {
		if item.Label == ".shortcut" && item.TextEdit != nil {
			if item.TextEdit.NewText != ".shortcut}" {
				t.Errorf("NewText = %q, want %q", item.TextEdit.NewText, ".shortcut}")
			}
			if item.FilterText != "{.shortcut" {
				t.Errorf("FilterText = %q, want %q", item.FilterText, "{.shortcut")
			}
			if item.TextEdit.Range.Start.Character != 13 {
				t.Errorf("Range.Start.Character = %d, want 13 (after {)", item.TextEdit.Range.Start.Character)
			}
			found = true
		}
	}
	if !found {
		t.Error("expected '.shortcut' completion item")
	}
}

func TestCompleteAttrOpenAutoCloseOmitsCloseBrace(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nClick **OK**{}")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	items := Complete(doc, document.Position{Line: 2, Character: 13}, f)
	if len(items) == 0 {
		t.Fatal("expected completion items")
	}
	found := false
	for _, item := range items {
		if item.Label == ".shortcut" && item.TextEdit != nil {
			if item.TextEdit.NewText != ".shortcut" {
				t.Errorf("NewText = %q, want %q (no } since editor already has one)", item.TextEdit.NewText, ".shortcut")
			}
			found = true
		}
	}
	if !found {
		t.Error("expected '.shortcut' completion item")
	}
}

func TestCompleteAttrClassUsesTextEdit(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title {.ta")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	items := Complete(doc, document.Position{Line: 0, Character: 12}, f)
	if len(items) == 0 {
		t.Fatal("expected completion items")
	}
	found := false
	for _, item := range items {
		if item.Label == "task" && item.TextEdit != nil {
			if item.TextEdit.NewText != "task}" {
				t.Errorf("NewText = %q, want %q", item.TextEdit.NewText, "task}")
			}
			found = true
		}
	}
	if !found {
		t.Error("expected 'task' completion item with TextEdit")
	}
}

func TestCompleteAttrClassAutoCloseOmitsCloseBrace(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title {.ta}")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	items := Complete(doc, document.Position{Line: 0, Character: 12}, f)
	if len(items) == 0 {
		t.Fatal("expected completion items")
	}
	found := false
	for _, item := range items {
		if item.Label == "task" && item.TextEdit != nil {
			if item.TextEdit.NewText != "task" {
				t.Errorf("NewText = %q, want %q (no } since editor already has one)", item.TextEdit.NewText, "task")
			}
			if item.TextEdit.Range.End.Character != 12 {
				t.Errorf("Range.End.Character = %d, want 12 (range ends at cursor, not past })",
					item.TextEdit.Range.End.Character)
			}
			found = true
		}
	}
	if !found {
		t.Error("expected 'task' completion item")
	}
}

func ptrKind(k PartialKind) *PartialKind { return &k }

func TestDetectPartialDoubleCurlyKeyref(t *testing.T) {
	text := "# Title\n\nInstall {{prod"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 16})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialDoubleCurlyKeyref {
		t.Errorf("Kind = %v, want PartialDoubleCurlyKeyref", pe.Kind)
	}
	if pe.Input != "prod" {
		t.Errorf("Input = %q, want %q", pe.Input, "prod")
	}
}

func TestDetectPartialDoubleCurlyEmpty(t *testing.T) {
	text := "# Title\n\nInstall {{"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 12})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialDoubleCurlyKeyref {
		t.Errorf("Kind = %v, want PartialDoubleCurlyKeyref", pe.Kind)
	}
	if pe.Input != "" {
		t.Errorf("Input = %q, want empty", pe.Input)
	}
}

func TestCompleteDoubleCurlyKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n  version: \"4.15\"\n---\n# Map\n\n- [Install](install.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nInstall {{prod")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)
	items := Complete(topicDoc, document.Position{Line: 2, Character: 16}, f)
	found := false
	for _, item := range items {
		if item.Label == "product-name" {
			found = true
			if item.Kind != 6 {
				t.Errorf("Kind = %d, want 6 (variable)", item.Kind)
			}
		}
	}
	if !found {
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.Label
		}
		t.Errorf("expected 'product-name' completion, got %v", labels)
	}
}

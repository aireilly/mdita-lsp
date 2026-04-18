package codeaction

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestGenerateToC(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nIntro.\n\n## Section One\n\n### Subsection\n\n## Section Two\n")

	toc := GenerateToC(doc, []int{1, 2, 3, 4, 5, 6})
	if !strings.Contains(toc, "<!--toc:start-->") {
		t.Error("missing toc:start marker")
	}
	if !strings.Contains(toc, "<!--toc:end-->") {
		t.Error("missing toc:end marker")
	}
	if !strings.Contains(toc, "[Section One](#section-one)") {
		t.Errorf("missing Section One link, got:\n%s", toc)
	}
	if !strings.Contains(toc, "  - [Subsection](#subsection)") {
		t.Errorf("missing indented Subsection, got:\n%s", toc)
	}
}

func TestGetActions(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(0, 0, 10, 0), f)
	if len(actions) == 0 {
		t.Error("expected at least one code action (ToC)")
	}
	foundToC := false
	for _, a := range actions {
		if a.Title == "Generate table of contents" {
			foundToC = true
		}
	}
	if !foundToC {
		t.Error("missing ToC action")
	}
}

func TestAddFrontMatterAction(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nContent.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(0, 0, 5, 0), f)
	found := false
	for _, a := range actions {
		if a.Title == "Add MDITA YAML front matter" {
			found = true
			if a.Edit == nil || !strings.Contains(a.Edit.NewText, "$schema") {
				t.Error("expected front matter with $schema")
			}
		}
	}
	if !found {
		t.Error("missing add front matter action")
	}
}

func TestNoFrontMatterActionWhenPresent(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\n$schema: \"urn:oasis:names:tc:mdita:rng:topic.rng\"\n---\n\n# Title\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(0, 0, 5, 0), f)
	for _, a := range actions {
		if a.Title == "Add MDITA YAML front matter" {
			t.Error("should not offer front matter action when schema already present")
		}
	}
}

func TestFixNBSPAction(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\u00A0One\n\nContent.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(2, 0, 2, 20), f)
	found := false
	for _, a := range actions {
		if a.Title == "Replace non-breaking whitespace" {
			found = true
			if a.Kind != "quickfix" {
				t.Error("expected quickfix kind")
			}
			if a.Edit == nil {
				t.Fatal("expected edit")
			}
			if strings.ContainsRune(a.Edit.NewText, '\u00A0') {
				t.Error("edit should not contain NBSP")
			}
			if !strings.Contains(a.Edit.NewText, "## Section One") {
				t.Errorf("expected '## Section One', got %q", a.Edit.NewText)
			}
			if len(a.Diagnostics) != 1 || a.Diagnostics[0].Code != "3" {
				t.Error("expected diagnostic with code 3")
			}
		}
	}
	if !found {
		t.Error("missing NBSP quickfix action")
	}
}

func TestFixNBSPActionNotOffered(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Normal Heading\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(2, 0, 2, 20), f)
	for _, a := range actions {
		if a.Title == "Replace non-breaking whitespace" {
			t.Error("should not offer NBSP fix when no NBSP present")
		}
	}
}

func TestFixFootnoteRefAction(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nSee footnote[^1].\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(2, 0, 2, 20), f)
	found := false
	for _, a := range actions {
		if strings.HasPrefix(a.Title, "Add footnote definition") {
			found = true
			if a.Kind != "quickfix" {
				t.Error("expected quickfix kind")
			}
			if a.Edit == nil {
				t.Fatal("expected edit")
			}
			if !strings.Contains(a.Edit.NewText, "[^1]:") {
				t.Errorf("expected footnote def, got %q", a.Edit.NewText)
			}
			if len(a.Diagnostics) != 1 || a.Diagnostics[0].Code != "13" {
				t.Error("expected diagnostic with code 13")
			}
		}
	}
	if !found {
		t.Error("missing footnote ref quickfix action")
	}
}

func TestFixFootnoteRefNotOfferedWhenDefExists(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nSee footnote[^1].\n\n[^1]: The definition.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(2, 0, 2, 20), f)
	for _, a := range actions {
		if strings.HasPrefix(a.Title, "Add footnote definition") {
			t.Error("should not offer footnote def action when def exists")
		}
	}
}

func TestFixHeadingHierarchyAction(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n#### Skipped\n\nContent.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(2, 0, 2, 15), f)
	found := false
	for _, a := range actions {
		if a.Title == "Fix heading level" {
			found = true
			if a.Kind != "quickfix" {
				t.Error("expected quickfix kind")
			}
			if a.Edit == nil {
				t.Fatal("expected edit")
			}
			if !strings.HasPrefix(a.Edit.NewText, "## ") {
				t.Errorf("expected h2 fix, got %q", a.Edit.NewText)
			}
			if len(a.Diagnostics) != 1 || a.Diagnostics[0].Code != "6" {
				t.Error("expected diagnostic with code 6")
			}
		}
	}
	if !found {
		t.Error("missing heading hierarchy quickfix action")
	}
}

func TestFixHeadingHierarchyNoSkip(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n\n### Sub\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(0, 0, 5, 0), f)
	for _, a := range actions {
		if a.Title == "Fix heading level" {
			t.Error("should not offer fix when hierarchy is valid")
		}
	}
}

func boolPtr(v bool) *bool { return &v }

func TestBuildXHTMLActionOnMap(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Topic](topic.md)\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)

	actions := GetActions(mapDoc, document.Rng(0, 0, 3, 0), f)
	found := false
	for _, a := range actions {
		if a.Title == "Build XHTML with DITA OT" {
			found = true
			if a.Command == nil {
				t.Fatal("expected command")
			}
			if a.Command.Command != "mdita-lsp.ditaOtBuild" {
				t.Errorf("command = %q, want %q", a.Command.Command, "mdita-lsp.ditaOtBuild")
			}
			if len(a.Command.Arguments) != 2 || a.Command.Arguments[0] != mapDoc.URI || a.Command.Arguments[1] != "xhtml" {
				t.Errorf("arguments = %v, want [%s xhtml]", a.Command.Arguments, mapDoc.URI)
			}
		}
	}
	if !found {
		t.Error("missing 'Build XHTML with DITA OT' action for map document")
	}
}

func TestBuildDITAActionOnMap(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Topic](topic.md)\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)

	actions := GetActions(mapDoc, document.Rng(0, 0, 3, 0), f)
	found := false
	for _, a := range actions {
		if a.Title == "Build DITA with DITA OT" {
			found = true
			if a.Command == nil {
				t.Fatal("expected command")
			}
			if a.Command.Command != "mdita-lsp.ditaOtBuild" {
				t.Errorf("command = %q, want %q", a.Command.Command, "mdita-lsp.ditaOtBuild")
			}
			if len(a.Command.Arguments) != 2 || a.Command.Arguments[0] != mapDoc.URI || a.Command.Arguments[1] != "dita" {
				t.Errorf("arguments = %v, want [%s dita]", a.Command.Arguments, mapDoc.URI)
			}
		}
	}
	if !found {
		t.Error("missing 'Build DITA with DITA OT' action for map document")
	}
}

func TestBuildActionsNotOnTopic(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nContent.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(0, 0, 3, 0), f)
	for _, a := range actions {
		if a.Title == "Build XHTML with DITA OT" || a.Title == "Build DITA with DITA OT" {
			t.Errorf("should not offer build action %q for topic documents", a.Title)
		}
	}
}

func TestBuildActionsDisabled(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Topic](topic.md)\n")
	cfg := config.Default()
	cfg.Build.DitaOT.Enable = boolPtr(false)
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)

	actions := GetActions(mapDoc, document.Rng(0, 0, 3, 0), f)
	for _, a := range actions {
		if a.Title == "Build XHTML with DITA OT" || a.Title == "Build DITA with DITA OT" {
			t.Errorf("should not offer build action %q when disabled", a.Title)
		}
	}
}

func TestAddToMapAction(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# My Doc\n\nContent.\n")
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Other](other.md)\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	f.AddDoc(mapDoc)

	actions := GetActions(doc, document.Rng(0, 0, 3, 0), f)
	found := false
	for _, a := range actions {
		if strings.HasPrefix(a.Title, "Add to ") {
			found = true
			if a.Command == nil || a.Command.Command != "mdita-lsp.addToMap" {
				t.Error("expected addToMap command")
			}
		}
	}
	if !found {
		t.Error("missing add to map action")
	}
}

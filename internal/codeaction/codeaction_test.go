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
		"# Title\n\n## Section\n\n[[missing-doc]]\n")
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

func TestConvertWikiLinkAction(t *testing.T) {
	source := document.New("file:///project/index.md", 1,
		"# Index\n\nSee [[install]] for details.\n")
	target := document.New("file:///project/install.md", 1,
		"# Install\n\nContent.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(source)
	f.AddDoc(target)

	actions := GetActions(source, document.Rng(2, 4, 2, 15), f)
	found := false
	for _, a := range actions {
		if a.Title == "Convert to markdown link" {
			found = true
			if a.Edit == nil {
				t.Error("expected edit")
			} else if !strings.Contains(a.Edit.NewText, "install.md") {
				t.Errorf("expected markdown link, got %s", a.Edit.NewText)
			}
		}
	}
	if !found {
		t.Error("missing convert wiki link action")
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

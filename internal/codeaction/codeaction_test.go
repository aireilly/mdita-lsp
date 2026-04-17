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

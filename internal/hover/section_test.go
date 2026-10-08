package hover

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func hoverOn(t *testing.T, uri, text string, pos document.Position) string {
	t.Helper()
	doc := document.New(uri, 1, text)
	f := workspace.NewFolder("file:///project", config.Default())
	f.AddDoc(doc)
	return GetHover(doc, pos, f)
}

func TestHoverSectionInTypedTopic(t *testing.T) {
	got := hoverOn(t, "file:///project/c.md", "# C {.concept}\n\n## Background\n",
		document.Position{Line: 2, Character: 5})
	if !strings.Contains(got, "`<section>`") {
		t.Errorf("hover = %q, expected it to name a DITA section", got)
	}
	if !strings.Contains(got, "concept") {
		t.Errorf("hover = %q, expected it to name the topic type", got)
	}
	if !strings.Contains(got, "c/background") {
		t.Errorf("hover = %q, expected the qualified section address", got)
	}
}

func TestHoverSectionInReference(t *testing.T) {
	got := hoverOn(t, "file:///project/r.md", "# R {.reference}\n\n## Options\n",
		document.Position{Line: 2, Character: 5})
	if !strings.Contains(got, "reference") {
		t.Errorf("hover = %q, expected it to name the topic type", got)
	}
}

func TestHoverNestedTopicInGenericTopic(t *testing.T) {
	got := hoverOn(t, "file:///project/t.md", "# T\n\n## A\n",
		document.Position{Line: 2, Character: 4})
	if !strings.Contains(got, "nested `<topic>`") {
		t.Errorf("hover = %q, expected it to say the heading opens a nested topic", got)
	}
}

package completion

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

// A section is addressed through the topic that holds it, so completion
// offers the qualified form rather than the bare slug.
func TestAnchorCompletionQualifiesSections(t *testing.T) {
	target := document.New("file:///project/c.md", 1, "# C {.concept}\n\n## Background\n")
	src := document.New("file:///project/t.md", 1, "# T\n\nSee [x](c.md#)\n")
	f := workspace.NewFolder("file:///project", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	items := completeInlineAnchor("c.md", "", src, f)
	var found bool
	for _, it := range items {
		if it.InsertText == "c/background" {
			found = true
			if it.Detail != "Background" {
				t.Errorf("Detail = %q, want %q", it.Detail, "Background")
			}
		}
		if it.InsertText == "background" {
			t.Errorf("completion offered the unqualified section slug %q", it.InsertText)
		}
	}
	if !found {
		t.Errorf("no qualified section completion in %+v", items)
	}
}

func TestAnchorCompletionLeavesGenericTopicsAlone(t *testing.T) {
	target := document.New("file:///project/t2.md", 1, "# T2\n\n## A\n")
	src := document.New("file:///project/t.md", 1, "# T\n\nSee [x](t2.md#)\n")
	f := workspace.NewFolder("file:///project", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	var found bool
	for _, it := range completeInlineAnchor("t2.md", "", src, f) {
		if it.InsertText == "a" {
			found = true
		}
	}
	if !found {
		t.Error("expected the plain heading slug for a generic topic")
	}
}

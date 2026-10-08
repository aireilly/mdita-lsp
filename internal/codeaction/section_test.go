package codeaction

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func actionsFor(t *testing.T, text string, rng document.Range, others ...*document.Document) []CodeAction {
	t.Helper()
	doc := document.New("file:///project/a.md", 1, text)
	f := workspace.NewFolder("file:///project", config.Default())
	f.AddDoc(doc)
	for _, o := range others {
		f.AddDoc(o)
	}
	return GetActions(doc, rng, f)
}

func byTitle(actions []CodeAction, title string) *CodeAction {
	for i := range actions {
		if actions[i].Title == title {
			return &actions[i]
		}
	}
	return nil
}

func TestChangeToSectionLevelAction(t *testing.T) {
	actions := actionsFor(t, "# C {.concept}\n\n## Background\n\n### Detail\n", document.Rng(4, 0, 4, 10))
	a := byTitle(actions, "Change to ##")
	if a == nil {
		t.Fatalf("expected a 'Change to ##' action, got %+v", titles(actions))
	}
	if a.Edit == nil || a.Edit.NewText != "## Detail" {
		t.Errorf("edit = %+v, want '## Detail'", a.Edit)
	}
	if a.Edit.Range.Start.Line != 4 {
		t.Errorf("edit line = %d, want 4", a.Edit.Range.Start.Line)
	}
}

func TestAddSectionClassToEveryHeadingAtThisLevel(t *testing.T) {
	text := "# A\n\n## A.1\n\n## A.2 {.section}\n"
	actions := actionsFor(t, text, document.Rng(4, 0, 4, 10))
	a := byTitle(actions, "Add {.section} to every heading at this level")
	if a == nil {
		t.Fatalf("expected the add-class action, got %+v", titles(actions))
	}
	if a.Edit == nil || a.Edit.NewText != "## A.1 {.section}" {
		t.Errorf("edit = %+v, want '## A.1 {.section}'", a.Edit)
	}
	if a.Edit.Range.Start.Line != 2 {
		t.Errorf("edit line = %d, want 2", a.Edit.Range.Start.Line)
	}
}

func TestRemoveSectionClassFromThisHeading(t *testing.T) {
	text := "# A\n\n## A.1\n\n## A.2 {.section}\n"
	actions := actionsFor(t, text, document.Rng(4, 0, 4, 10))
	a := byTitle(actions, "Remove {.section} from this heading")
	if a == nil {
		t.Fatalf("expected the remove-class action, got %+v", titles(actions))
	}
	if a.Edit == nil || a.Edit.NewText != "## A.2" {
		t.Errorf("edit = %+v, want '## A.2'", a.Edit)
	}
}

func TestQualifySectionLinkAction(t *testing.T) {
	target := document.New("file:///project/c.md", 1, "# C {.concept}\n\n## Background\n")
	actions := actionsFor(t, "# T\n\nSee [x](c.md#background).\n", document.Rng(2, 4, 2, 24), target)
	a := byTitle(actions, "Add the topic id to the section link")
	if a == nil {
		t.Fatalf("expected the qualify-link action, got %+v", titles(actions))
	}
	if a.Edit == nil || a.Edit.NewText != "[x](c.md#c/background)" {
		t.Errorf("edit = %+v, want '[x](c.md#c/background)'", a.Edit)
	}
}

func titles(actions []CodeAction) []string {
	var out []string
	for _, a := range actions {
		out = append(out, a.Title)
	}
	return out
}

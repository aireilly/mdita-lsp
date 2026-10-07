package rename

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func folderWith(docs ...*document.Document) *workspace.Folder {
	f := workspace.NewFolder("file:///project", config.Default())
	for _, d := range docs {
		f.AddDoc(d)
	}
	return f
}

// The heading's range starts after the hashes, so a rename that wrote the
// hashes back produced "## ## Setup Steps".
func TestRenameReplacesTheWholeHeadingLine(t *testing.T) {
	doc := document.New("file:///project/guide.md", 1,
		"# Guide\n\n## Install Steps\n\ntext\n")
	f := folderWith(doc)

	edits := DoRename(doc, document.Position{Line: 2, Character: 5}, "Setup Steps", f, symbols.NewGraph())
	if len(edits) == 0 {
		t.Fatal("no edits")
	}
	e := edits[0]
	if e.NewText != "## Setup Steps" {
		t.Errorf("new text = %q, want %q", e.NewText, "## Setup Steps")
	}
	if e.Range.Start.Character != 0 {
		t.Errorf("edit starts at character %d, want 0", e.Range.Start.Character)
	}
	if e.Range.End.Character != len("## Install Steps") {
		t.Errorf("edit ends at character %d, want %d", e.Range.End.Character, len("## Install Steps"))
	}
}

func TestRenameUpdatesFragmentLinksInOtherFiles(t *testing.T) {
	target := document.New("file:///project/guide.md", 1,
		"# Guide\n\n## Install Steps\n\ntext\n")
	other := document.New("file:///project/intro.md", 1,
		"# Intro\n\nSee [the steps](guide.md#install-steps).\n")
	f := folderWith(target, other)

	edits := DoRename(target, document.Position{Line: 2, Character: 5}, "Setup Steps", f, symbols.NewGraph())

	var found bool
	for _, e := range edits {
		if e.URI == other.URI {
			found = true
			if e.NewText != "[the steps](guide.md#setup-steps)" {
				t.Errorf("link rewritten to %q", e.NewText)
			}
		}
	}
	if !found {
		t.Error("the link in intro.md was not updated")
	}
}

func TestRenameLeavesAnExplicitIDAlone(t *testing.T) {
	doc := document.New("file:///project/guide.md", 1,
		"# Guide\n\n## Install Steps {#install}\n")
	other := document.New("file:///project/intro.md", 1,
		"# Intro\n\n[go](guide.md#install)\n")
	f := folderWith(doc, other)

	edits := DoRename(doc, document.Position{Line: 2, Character: 5}, "Setup Steps", f, symbols.NewGraph())
	for _, e := range edits {
		if e.URI == other.URI {
			t.Errorf("link to an explicit id was rewritten: %q", e.NewText)
		}
	}
}

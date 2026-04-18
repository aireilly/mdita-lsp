package filerename

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func testFolder(docs ...*document.Document) *workspace.Folder {
	f := workspace.NewFolder("file:///project", config.Default())
	for _, d := range docs {
		f.AddDoc(d)
	}
	return f
}

func TestWikiLinkRename(t *testing.T) {
	referring := document.New("file:///project/index.md", 1,
		"# Index\n\nSee [[install]] for details.\n")
	target := document.New("file:///project/install.md", 1,
		"# Install\n\nContent.\n")

	folder := testFolder(referring, target)

	edits := ComputeEdits([]FileRename{
		{OldURI: "file:///project/install.md", NewURI: "file:///project/setup.md"},
	}, folder)

	if len(edits) != 1 {
		t.Fatalf("expected 1 document edit, got %d", len(edits))
	}
	if edits[0].URI != "file:///project/index.md" {
		t.Errorf("expected edit in index.md, got %s", edits[0].URI)
	}
	if len(edits[0].Edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits[0].Edits))
	}
	if edits[0].Edits[0].NewText != "[[setup]]" {
		t.Errorf("expected [[setup]], got %s", edits[0].Edits[0].NewText)
	}
}

func TestMdLinkRename(t *testing.T) {
	referring := document.New("file:///project/index.md", 1,
		"# Index\n\nSee [install guide](./install.md) for details.\n")
	target := document.New("file:///project/install.md", 1,
		"# Install\n\nContent.\n")

	folder := testFolder(referring, target)

	edits := ComputeEdits([]FileRename{
		{OldURI: "file:///project/install.md", NewURI: "file:///project/setup.md"},
	}, folder)

	if len(edits) != 1 {
		t.Fatalf("expected 1 document edit, got %d", len(edits))
	}
	found := false
	for _, e := range edits[0].Edits {
		if e.NewText == "[install guide](./setup.md)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected markdown link to be updated to setup.md")
	}
}

func TestNoSelfEdit(t *testing.T) {
	target := document.New("file:///project/install.md", 1,
		"# Install\n\nContent.\n")

	folder := testFolder(target)

	edits := ComputeEdits([]FileRename{
		{OldURI: "file:///project/install.md", NewURI: "file:///project/setup.md"},
	}, folder)

	if len(edits) != 0 {
		t.Errorf("expected no edits for document being renamed, got %d", len(edits))
	}
}

func TestMapHrefRename(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Install](install.md)\n- [Config](config.md)\n")
	target := document.New("file:///project/install.md", 1,
		"# Install\n\nContent.\n")

	folder := testFolder(mapDoc, target)

	edits := ComputeEdits([]FileRename{
		{OldURI: "file:///project/install.md", NewURI: "file:///project/setup.md"},
	}, folder)

	var mapEdits []TextEdit
	for _, de := range edits {
		if de.URI == "file:///project/map.mditamap" {
			mapEdits = de.Edits
		}
	}
	if len(mapEdits) == 0 {
		t.Fatal("expected map href to be updated")
	}
	if mapEdits[0].NewText != "[Install](./setup.md)" {
		t.Errorf("expected [Install](./setup.md), got %s", mapEdits[0].NewText)
	}
}

func TestMdLinkRenameSubdir(t *testing.T) {
	referring := document.New("file:///project/guide/index.md", 1,
		"# Guide\n\nSee [setup](../docs/install.md) for details.\n")
	target := document.New("file:///project/docs/install.md", 1,
		"# Install\n\nContent.\n")

	folder := testFolder(referring, target)

	edits := ComputeEdits([]FileRename{
		{OldURI: "file:///project/docs/install.md", NewURI: "file:///project/docs/setup.md"},
	}, folder)

	if len(edits) != 1 {
		t.Fatalf("expected 1 document edit, got %d", len(edits))
	}
	found := false
	for _, e := range edits[0].Edits {
		if e.NewText == "[setup](../docs/setup.md)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected cross-dir md link to be updated, got %v", edits[0].Edits)
	}
}

func TestMdLinkRenameMoveDir(t *testing.T) {
	referring := document.New("file:///project/index.md", 1,
		"# Index\n\nSee [guide](docs/intro.md) for help.\n")
	target := document.New("file:///project/docs/intro.md", 1,
		"# Intro\n")

	folder := testFolder(referring, target)

	edits := ComputeEdits([]FileRename{
		{OldURI: "file:///project/docs/intro.md", NewURI: "file:///project/guide/getting-started.md"},
	}, folder)

	if len(edits) != 1 {
		t.Fatalf("expected 1 document edit, got %d", len(edits))
	}
	found := false
	for _, e := range edits[0].Edits {
		if e.NewText == "[guide](./guide/getting-started.md)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected md link to be updated for dir move, got %v", edits[0].Edits)
	}
}

func TestWikiLinkWithHeading(t *testing.T) {
	referring := document.New("file:///project/index.md", 1,
		"# Index\n\nSee [[install#prerequisites]] for details.\n")
	target := document.New("file:///project/install.md", 1,
		"# Install\n\n## Prerequisites\n\nContent.\n")

	folder := testFolder(referring, target)

	edits := ComputeEdits([]FileRename{
		{OldURI: "file:///project/install.md", NewURI: "file:///project/setup.md"},
	}, folder)

	if len(edits) != 1 {
		t.Fatalf("expected 1 document edit, got %d", len(edits))
	}
	if edits[0].Edits[0].NewText != "[[setup#prerequisites]]" {
		t.Errorf("expected [[setup#prerequisites]], got %s", edits[0].Edits[0].NewText)
	}
}

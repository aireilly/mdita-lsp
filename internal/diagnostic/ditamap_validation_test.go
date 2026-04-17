package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestBrokenMapReference(t *testing.T) {
	mapDoc := makeDoc("file:///project/map.mditamap",
		"# My Map", "", "- [Topic](nonexistent.md)")
	f := makeFolder(mapDoc)
	diags := CheckDitamap(mapDoc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeBrokenMapReference {
			found = true
		}
	}
	if !found {
		t.Error("expected BrokenMapReference diagnostic")
	}
}

func TestValidMapReference(t *testing.T) {
	topic := makeDoc("file:///project/topic.md", "# Topic", "", "Content.")
	mapDoc := makeDoc("file:///project/map.mditamap",
		"# My Map", "", "- [Topic](topic.md)")
	f := makeFolder(topic, mapDoc)
	diags := CheckDitamap(mapDoc, f)

	for _, d := range diags {
		if d.Code == CodeBrokenMapReference {
			t.Error("should not report BrokenMapReference for valid ref")
		}
	}
}

func TestCircularMapReference(t *testing.T) {
	map1 := document.New("file:///project/a.mditamap", 1,
		"# Map A\n\n- [Map B](b.mditamap)\n")
	map2 := document.New("file:///project/b.mditamap", 1,
		"# Map B\n\n- [Map A](a.mditamap)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(map1)
	f.AddDoc(map2)

	diags := CheckDitamap(map1, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeCircularMapReference {
			found = true
		}
	}
	if !found {
		t.Error("expected CircularMapReference diagnostic")
	}
}

func TestInconsistentMapHeadingHierarchy(t *testing.T) {
	parent := document.New("file:///project/parent.md", 1,
		"# Parent Topic\n\nSome content.\n")
	child := document.New("file:///project/child.md", 1,
		"# Child Topic\n\nChild content.\n")

	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Parent](parent.md)\n  - [Child](child.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(parent)
	f.AddDoc(child)
	f.AddDoc(mapDoc)

	diags := CheckDitamap(mapDoc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeInconsistentMapHeadingHierarchy {
			found = true
		}
	}
	if !found {
		t.Error("expected InconsistentMapHeadingHierarchy diagnostic for nested H1")
	}
}

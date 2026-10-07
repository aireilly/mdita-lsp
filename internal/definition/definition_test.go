package definition

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestGotoDefMdLinkRelativePath(t *testing.T) {
	target := document.New("file:///project/docs/install.md", 1,
		"# Installation\n\n## Prerequisites\n")
	source := document.New("file:///project/guide/index.md", 1,
		"# Guide\n\n[setup](../docs/install.md)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(target)
	f.AddDoc(source)

	mls := source.Index.MdLinks()
	if len(mls) == 0 {
		t.Fatal("no md links found")
	}
	locs := GotoDef(source, mls[0].Rng().Start, f)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for relative md link")
	}
	if locs[0].URI != "file:///project/docs/install.md" {
		t.Errorf("expected docs/install.md, got %q", locs[0].URI)
	}
}

func TestGotoDefNoResult(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	locs := GotoDef(doc, document.Position{Line: 2, Character: 3}, f)
	if len(locs) != 0 {
		t.Errorf("expected no locations, got %d", len(locs))
	}
}

func TestGotoDefConref(t *testing.T) {
	target := document.New("file:///project/shared.md", 1,
		"# Shared\n\n## Warning {#warning-para}\n\nContent here.\n")
	source := document.New("file:///project/doc.md", 1,
		"# Doc\n\n<p data-conref=\"shared.md#topic/warning-para\">fallback</p>\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(target)
	f.AddDoc(source)

	// Find the ConrefElement in the source document
	elem := source.ElementAt(document.Position{Line: 2, Character: 10})
	if elem == nil {
		t.Fatal("no element at conref position")
	}
	if _, ok := elem.(*document.ConrefElement); !ok {
		t.Fatalf("expected ConrefElement, got %T", elem)
	}

	locs := GotoDef(source, document.Position{Line: 2, Character: 10}, f)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for conref")
	}
	if locs[0].URI != "file:///project/shared.md" {
		t.Errorf("expected shared.md URI, got %q", locs[0].URI)
	}
}

func TestGotoDefConrefMissingFile(t *testing.T) {
	source := document.New("file:///project/doc.md", 1,
		"# Doc\n\n<p data-conref=\"missing.md#topic/note\">fallback</p>\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(source)

	locs := GotoDef(source, document.Position{Line: 2, Character: 10}, f)
	if len(locs) != 0 {
		t.Errorf("expected no locations for missing conref target, got %d", len(locs))
	}
}

func TestGotoDefDitaFragment(t *testing.T) {
	target := document.New("file:///p/install.md", 1,
		"# Install the software\n\n## Prerequisites\n\nText.\n")
	src := document.New("file:///p/doc.md", 1,
		"# Doc\n\nSee [it](install.md#install-the-software/prerequisites).\n")
	f := workspace.NewFolder("file:///p", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	locs := GotoDef(src, document.Position{Line: 2, Character: 6}, f)
	if len(locs) != 1 {
		t.Fatalf("locations = %d, want 1", len(locs))
	}
	if locs[0].URI != target.URI {
		t.Errorf("URI = %q, want %q", locs[0].URI, target.URI)
	}
	if locs[0].Range.Start.Line != 2 {
		t.Errorf("line = %d, want the Prerequisites heading on line 2", locs[0].Range.Start.Line)
	}
}

func defFolder(docs ...*document.Document) *workspace.Folder {
	f := workspace.NewFolder("file:///project", config.Default())
	for _, d := range docs {
		f.AddDoc(d)
	}
	return f
}

func gotoFirstLink(t *testing.T, src *document.Document, f *workspace.Folder) []Location {
	t.Helper()
	mls := src.Index.MdLinks()
	if len(mls) == 0 {
		t.Fatal("no md links parsed")
	}
	return GotoDef(src, mls[0].Rng().Start, f)
}

func TestGotoDefFragmentInSameDocument(t *testing.T) {
	doc := document.New("file:///project/guide.md", 1,
		"# Guide\n\n[steps](#install-steps)\n\n## Install Steps\n")
	f := defFolder(doc)

	locs := gotoFirstLink(t, doc, f)
	if len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
	if locs[0].Range.Start.Line != 4 {
		t.Errorf("jumped to line %d, want 4", locs[0].Range.Start.Line)
	}
}

// A duplicate heading gets a -1 suffix in the id the plug-in generates, so a
// link to "#setup-1" has to find the second heading.
func TestGotoDefDuplicateHeadingAnchor(t *testing.T) {
	target := document.New("file:///project/guide.md", 1,
		"# Guide\n\n## Setup\n\nfirst\n\n## Setup\n\nsecond\n")
	src := document.New("file:///project/a.md", 1,
		"# A\n\n[second setup](guide.md#setup-1)\n")
	f := defFolder(target, src)

	locs := gotoFirstLink(t, src, f)
	if len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
	if locs[0].Range.Start.Line != 6 {
		t.Errorf("jumped to line %d, want 6 (the second Setup)", locs[0].Range.Start.Line)
	}
}

// '_' is a dash character in the plug-in's id generator, so "My_var config"
// becomes "my-var-config".
func TestGotoDefUnderscoreAnchor(t *testing.T) {
	target := document.New("file:///project/guide.md", 1,
		"# Guide\n\n## My_var config\n")
	src := document.New("file:///project/a.md", 1,
		"# A\n\n[cfg](guide.md#my-var-config)\n")
	f := defFolder(target, src)

	if locs := gotoFirstLink(t, src, f); len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
}

func TestGotoDefDitaFragmentAddressing(t *testing.T) {
	target := document.New("file:///project/guide.md", 1,
		"---\nid: guide\n---\n# Guide\n\n## Install Steps\n")
	src := document.New("file:///project/a.md", 1,
		"# A\n\n[x](guide.md#guide/install-steps)\n")
	f := defFolder(target, src)

	if locs := gotoFirstLink(t, src, f); len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
}

// An unresolvable fragment still navigates to the file. The diagnostic
// reports the broken fragment; go to definition gets as close as it can.
func TestGotoDefUnknownFragmentFallsBackToTheFile(t *testing.T) {
	target := document.New("file:///project/guide.md", 1, "# Guide\n")
	src := document.New("file:///project/a.md", 1, "# A\n\n[x](guide.md#nope)\n")
	f := defFolder(target, src)

	locs := gotoFirstLink(t, src, f)
	if len(locs) != 1 || locs[0].URI != target.URI {
		t.Errorf("got %v, want the target file's title", locs)
	}
}

func TestGotoDefMissingFile(t *testing.T) {
	src := document.New("file:///project/a.md", 1, "# A\n\n[x](missing.md)\n")
	f := defFolder(src)

	if locs := gotoFirstLink(t, src, f); len(locs) != 0 {
		t.Errorf("got %d locations for a missing file, want 0", len(locs))
	}
}

func TestGotoDefKeyrefToMapDefinition(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install](install.md)\n\n[install-guide]: install.md \"Installation Guide\"\n")
	target := document.New("file:///project/install.md", 1, "# Installation\n")
	src := document.New("file:///project/a.md", 1, "# A\n\nSee [install-guide] for details.\n")
	f := defFolder(mapDoc, target, src)

	locs := GotoDef(src, document.Position{Line: 2, Character: 10}, f)
	if len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
	if locs[0].URI != mapDoc.URI {
		t.Errorf("jumped to %q, want the map's keydef", locs[0].URI)
	}
}

func TestGotoDefConrefTarget(t *testing.T) {
	target := document.New("file:///project/shared.md", 1,
		"# Shared\n\n## Warning Text\n")
	src := document.New("file:///project/a.md", 1,
		"# A\n\n<p data-conref=\"shared.md#shared/warning-text\"></p>\n")
	f := defFolder(target, src)

	locs := GotoDef(src, document.Position{Line: 2, Character: 20}, f)
	if len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
	if locs[0].URI != target.URI {
		t.Errorf("jumped to %q, want %q", locs[0].URI, target.URI)
	}
}

func TestGotoDefConkeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Shared](shared.md)\n\n[warn]: shared.md \"Shared warnings\"\n")
	target := document.New("file:///project/shared.md", 1, "# Shared\n\n## Warning Text\n")
	src := document.New("file:///project/a.md", 1,
		"# A\n\n<p data-conkeyref=\"warn/warning-text\"></p>\n")
	f := defFolder(mapDoc, target, src)

	locs := GotoDef(src, document.Position{Line: 2, Character: 22}, f)
	if len(locs) != 1 || locs[0].URI != target.URI {
		t.Errorf("got %v, want shared.md", locs)
	}
}

func TestGotoDefConrefToMissingFile(t *testing.T) {
	src := document.New("file:///project/a.md", 1,
		"# A\n\n<p data-conref=\"gone.md#gone/x\"></p>\n")
	f := defFolder(src)

	if locs := GotoDef(src, document.Position{Line: 2, Character: 20}, f); len(locs) != 0 {
		t.Errorf("got %v, want no locations", locs)
	}
}

func TestGotoDefKeyrefWithNoKeydef(t *testing.T) {
	src := document.New("file:///project/a.md", 1, "# A\n\nSee [no-such-key] here.\n")
	f := defFolder(src)

	if locs := GotoDef(src, document.Position{Line: 2, Character: 8}, f); len(locs) != 0 {
		t.Errorf("got %v, want no locations", locs)
	}
}

// A link with no fragment lands on the target's title.
func TestGotoDefLinkWithoutFragmentLandsOnTheTitle(t *testing.T) {
	target := document.New("file:///project/guide.md", 1, "# Guide\n\ntext\n")
	src := document.New("file:///project/a.md", 1, "# A\n\n[g](guide.md)\n")
	f := defFolder(target, src)

	locs := gotoFirstLink(t, src, f)
	if len(locs) != 1 || locs[0].Range.Start.Line != 0 {
		t.Errorf("got %v, want the title on line 0", locs)
	}
}

// The plug-in resolves an href against the source file's directory. A link
// from a subdirectory to "target.md" must not find the root file.
func TestGotoDefDoesNotFallBackAcrossDirectories(t *testing.T) {
	root := document.New("file:///project/target.md", 1, "# Root target\n")
	src := document.New("file:///project/sub/stem.md", 1, "# Stem\n\n[t](target.md)\n")
	f := defFolder(root, src)

	if locs := gotoFirstLink(t, src, f); len(locs) != 0 {
		t.Errorf("got %v, want no locations; the build fails on sub/target.md", locs)
	}
}

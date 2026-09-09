package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestCheckConrefsTargetMissing(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n<p data-conref=\"missing.md#topic/note\">fallback</p>\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	diags := CheckConrefs(doc, f)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for missing conref target")
	}
	found := false
	for _, d := range diags {
		if d.Code == CodeConrefTargetMissing {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %s diagnostic, got %v", CodeConrefTargetMissing, diags)
	}
}

func TestCheckConrefsTargetPresent(t *testing.T) {
	target := document.New("file:///project/shared.md", 1,
		"# Shared\n\n## Warning {#warning-para}\n")
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n<p data-conref=\"shared.md#topic/warning-para\">fallback</p>\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(target)
	f.AddDoc(doc)

	diags := CheckConrefs(doc, f)
	for _, d := range diags {
		if d.Code == CodeConrefTargetMissing {
			t.Errorf("unexpected %s diagnostic when target exists", CodeConrefTargetMissing)
		}
	}
}

func TestCheckConrefsNoConrefs(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	diags := CheckConrefs(doc, f)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for doc without conrefs, got %d", len(diags))
	}
}

func TestCheckConkeyrefKeyMissing(t *testing.T) {
	// A map document is needed to populate the key table; without it, no
	// conkeyref diagnostics are emitted (same behaviour as CheckKeyrefs).
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Known Topic](known.md)\n\n[known]: known.md\n")
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n<span data-conkeyref=\"unknown-key/element\">fallback</span>\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(doc)

	diags := CheckConrefs(doc, f)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for missing conkeyref key")
	}
	found := false
	for _, d := range diags {
		if d.Code == CodeConkeyrefKeyMissing {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %s diagnostic, got %v", CodeConkeyrefKeyMissing, diags)
	}
}

func TestCheckConkeyrefNoMapFiles(t *testing.T) {
	// Without map files, conkeyref validation is skipped — no false positives.
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n<span data-conkeyref=\"unknown-key/element\">fallback</span>\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	diags := CheckConrefs(doc, f)
	for _, d := range diags {
		if d.Code == CodeConkeyrefKeyMissing {
			t.Errorf("unexpected conkeyref diagnostic when no map files present")
		}
	}
}

func TestCheckConkeyrefElementMissing(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Shared](shared.md)\n\n[warnings]: shared.md\n")
	shared := document.New("file:///project/shared.md", 1,
		"# Shared\n\n## Disk full\n\nText.\n")
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n<span data-conkeyref=\"warnings/no-such-id\">fallback</span>\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(shared)
	f.AddDoc(doc)

	diags := CheckConrefs(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeConkeyrefElementMissing {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected CodeConkeyrefElementMissing, got %+v", diags)
	}

	ok := document.New("file:///project/ok.md", 1,
		"# Title\n\n<span data-conkeyref=\"warnings/disk-full\">fallback</span>\n")
	f.AddDoc(ok)
	if d := CheckConrefs(ok, f); len(d) != 0 {
		t.Errorf("expected no diagnostics for a resolvable conkeyref, got %+v", d)
	}
}

func TestCheckAmbiguousLink(t *testing.T) {
	a := document.New("file:///project/a/install.md", 1, "# Install A\n")
	b := document.New("file:///project/b/install.md", 1, "# Install B\n")
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nSee [it](install.md).\n")
	f := workspace.NewFolder("file:///project", config.Default())
	f.AddDoc(a)
	f.AddDoc(b)
	f.AddDoc(doc)

	diags := checkLinks(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeAmbiguousLink {
			found = true
		}
	}
	if !found {
		t.Errorf("expected CodeAmbiguousLink, got %+v", diags)
	}
}

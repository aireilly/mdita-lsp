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
		"# Map\n\n- [Known Topic](known.md)\n")
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

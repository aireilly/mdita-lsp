package diagnostic

import (
	"strings"
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

// file.md#topic-id/element-id is DITA element addressing, which the plug-in
// passes through to @href unchanged.
func TestDitaFragmentLinkAccepted(t *testing.T) {
	target := document.New("file:///p/install.md", 1,
		"# Install the software\n\n## Prerequisites\n\nText.\n")
	src := document.New("file:///p/doc.md", 1,
		"# Doc\n\nSee [it](install.md#install-the-software/prerequisites).\n")
	f := workspace.NewFolder("file:///p", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	if d := checkLinks(src, f); len(d) != 0 {
		t.Errorf("expected no diagnostics for a valid fragment, got %+v", d)
	}
}

func TestDitaFragmentLinkWrongTopicID(t *testing.T) {
	target := document.New("file:///p/install.md", 1,
		"---\nid: install-sw\n---\n\n# Install the software\n\n## Prerequisites\n\nText.\n")
	src := document.New("file:///p/doc.md", 1,
		"# Doc\n\nSee [it](install.md#wrong-topic/prerequisites).\n")
	f := workspace.NewFolder("file:///p", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	diags := checkLinks(src, f)
	if len(diags) != 1 || diags[0].Code != CodeBrokenLink {
		t.Fatalf("expected one broken-link diagnostic, got %+v", diags)
	}
	if !strings.Contains(diags[0].Message, "install-sw") {
		t.Errorf("message should name the real topic id, got %q", diags[0].Message)
	}
}

func TestDitaFragmentLinkMissingElement(t *testing.T) {
	target := document.New("file:///p/install.md", 1, "# Install the software\n\nText.\n")
	src := document.New("file:///p/doc.md", 1,
		"# Doc\n\nSee [it](install.md#install-the-software/no-such-element).\n")
	f := workspace.NewFolder("file:///p", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	diags := checkLinks(src, f)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "no-such-element") {
		t.Fatalf("expected a missing-element diagnostic, got %+v", diags)
	}
}

// A bare topic id is a valid fragment too.
func TestFragmentTopicIDAccepted(t *testing.T) {
	target := document.New("file:///p/install.md", 1,
		"---\nid: install-sw\n---\n\n# Install the software\n\nText.\n")
	src := document.New("file:///p/doc.md", 1, "# Doc\n\nSee [it](install.md#install-sw).\n")
	f := workspace.NewFolder("file:///p", config.Default())
	f.AddDoc(target)
	f.AddDoc(src)

	if d := checkLinks(src, f); len(d) != 0 {
		t.Errorf("expected no diagnostics, got %+v", d)
	}
}

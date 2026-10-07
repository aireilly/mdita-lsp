package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
)

// goldmark's WithAutoHeadingID gives every heading an id attribute, so a check
// that reads node.Attributes() reports attributes on documents that contain
// none. That warned on every MDITA topic with a heading.

func TestNoAttributeWarningWithoutAuthoredAttributes(t *testing.T) {
	doc := document.New("file:///project/topic.md", 1,
		"---\n$schema: \"urn:oasis:names:tc:mdita:rng:topic.rng\"\nid: topic\n---\n\n# Title\n\nA short description.\n\n## Section\n\nBody text.\n")

	for _, d := range CheckProfile(doc, config.Default()) {
		if d.Code == CodeMditaProfileFeature {
			t.Errorf("unexpected profile warning on a document with no attributes: %s", d.Message)
		}
	}
}

func TestAttributeWarningWhenAuthorWritesAttributes(t *testing.T) {
	doc := document.New("file:///project/topic.md", 1,
		"---\n$schema: \"urn:oasis:names:tc:mdita:rng:topic.rng\"\nid: topic\n---\n\n# Title {.task}\n\nA short description.\n")

	found := false
	for _, d := range CheckProfile(doc, config.Default()) {
		if d.Code == CodeMditaProfileFeature {
			found = true
		}
	}
	if !found {
		t.Error("expected a profile warning for {.task} in an MDITA topic")
	}
}

// Only `profile: core` used to have any effect; `profile: extended` left the
// MDITA checks switched off entirely.
func TestProfileExtendedTurnsOnTheMditaChecks(t *testing.T) {
	doc := document.New("file:///project/a.md", 1, "# Title\n\n## Section\n\n### Too deep\n")

	cfg := config.Default()
	if diags := CheckProfile(doc, cfg); len(diags) != 0 {
		t.Errorf("no profile set, got %v; want no MDITA diagnostics for a .md file", diags)
	}

	extended := config.ProfileExtended
	cfg.Core.Mdita.Profile = &extended
	diags := CheckProfile(doc, cfg)
	if len(diags) == 0 {
		t.Error("profile: extended produced no MDITA diagnostics")
	}
}

func TestMditaEnableFalseTurnsTheChecksOff(t *testing.T) {
	doc := document.New("file:///project/a.mdita", 1, "# Title\n\n## Section\n\n### Too deep\n")
	cfg := config.Default()
	off := false
	cfg.Core.Mdita.Enable = &off

	if diags := CheckProfile(doc, cfg); len(diags) != 0 {
		t.Errorf("mdita.enable: false still produced %v", diags)
	}
}

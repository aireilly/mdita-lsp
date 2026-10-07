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

// Only `profile: core` used to have any effect: it was what switched the
// MDITA checks on at all, so `profile: extended` did nothing and could not
// even pick the profile for a .mdita file. Scope and profile are separate
// settings now.
func TestApplyToMarkdownTurnsOnTheMditaChecks(t *testing.T) {
	doc := document.New("file:///project/a.md", 1, "# Title\n\n## Section\n\n### Too deep\n")

	cfg := config.Default()
	extended := config.ProfileExtended
	cfg.Core.Mdita.Profile = &extended
	if diags := CheckProfile(doc, cfg); len(diags) != 0 {
		t.Errorf("a .md topic is Markdown DITA by default, got %v", diags)
	}

	on := true
	cfg.Core.Mdita.ApplyToMarkdown = &on
	if diags := CheckProfile(doc, cfg); len(diags) == 0 {
		t.Error("apply_to_markdown produced no MDITA diagnostics")
	}
}

// The profile setting picks which MDITA profile a .mdita file without a
// $schema gets. Before, it was read only when it equalled core.
func TestProfileSelectsTheMditaProfileForAMditaFile(t *testing.T) {
	doc := document.New("file:///project/a.mdita", 1,
		"# Title\n\nText with a footnote[^1].\n\n[^1]: the note\n")

	cfg := config.Default()
	if hasCode(codesOf(CheckProfile(doc, cfg)), CodeCoreProfileFeature) {
		t.Error("the extended profile has footnotes; none should be reported")
	}

	core := config.ProfileCore
	cfg.Core.Mdita.Profile = &core
	if !hasCode(codesOf(CheckProfile(doc, cfg)), CodeCoreProfileFeature) {
		t.Error("profile: core did not report the footnote")
	}
}

func codesOf(diags []Diagnostic) []string {
	var codes []string
	for _, d := range diags {
		codes = append(codes, d.Code)
	}
	return codes
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

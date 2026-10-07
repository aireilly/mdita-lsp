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

// A .md topic is Markdown DITA by default, and MDITA when the workspace says
// its maps give these files format="mdita".
func TestMarkdownTopicIsNotMditaByDefault(t *testing.T) {
	doc := document.New("file:///project/a.md", 1, "# Title\n\n## Section\n\n### Too deep\n")
	if diags := CheckProfile(doc, config.Default()); len(diags) != 0 {
		t.Errorf("a .md topic is Markdown DITA, got %v", diags)
	}
}

func TestApplyToMarkdownMakesAMarkdownTopicMdita(t *testing.T) {
	doc := document.New("file:///project/a.md", 1, "# Title\n\n## Section\n\n### Too deep\n")
	cfg := config.Default()
	on := true
	cfg.Core.Mdita.ApplyToMarkdown = &on

	if got := EffectiveProfile(doc, cfg); got != document.MditaExtended {
		t.Errorf("profile = %v, want MditaExtended", got)
	}
	if diags := CheckProfile(doc, cfg); len(diags) == 0 {
		t.Error("apply_to_markdown produced no MDITA diagnostics")
	}
}

// apply_to_markdown gives the extended profile, never core: MDitaReader's own
// default is extended, and a topic that needs core declares the $schema.
func TestApplyToMarkdownDoesNotSelectCore(t *testing.T) {
	doc := document.New("file:///project/a.md", 1,
		"# Title\n\nText with a footnote[^1].\n\n[^1]: the note\n")
	cfg := config.Default()
	on := true
	cfg.Core.Mdita.ApplyToMarkdown = &on

	for _, d := range CheckProfile(doc, cfg) {
		if d.Code == CodeCoreProfileFeature {
			t.Errorf("core-profile diagnostic under apply_to_markdown: %s", d.Message)
		}
	}
}

// A declared $schema still wins over the setting.
func TestSchemaWinsOverApplyToMarkdown(t *testing.T) {
	doc := document.New("file:///project/a.md", 1,
		"---\n$schema: urn:oasis:names:tc:mdita:core:xsd:topic.xsd\n---\n"+
			"# Title\n\nText with a footnote[^1].\n\n[^1]: the note\n")
	cfg := config.Default()
	on := true
	cfg.Core.Mdita.ApplyToMarkdown = &on

	if got := EffectiveProfile(doc, cfg); got != document.MditaCore {
		t.Errorf("profile = %v, want MditaCore", got)
	}
}

func TestMditaExtensionIsMditaExtended(t *testing.T) {
	doc := document.New("file:///project/a.mdita", 1, "# Title\n\n## Section\n\n### Too deep\n")
	if diags := CheckProfile(doc, config.Default()); len(diags) == 0 {
		t.Error("a .mdita topic produced no MDITA diagnostics")
	}
	if got := doc.DeclaredProfile(); got != document.MditaExtended {
		t.Errorf("profile = %v, want MditaExtended", got)
	}
}

// A $schema makes a .md topic MDITA, and picks the profile.
func TestSchemaSelectsTheProfile(t *testing.T) {
	cases := []struct {
		schema string
		want   document.MditaProfile
	}{
		{"urn:oasis:names:tc:mdita:core:xsd:topic.xsd", document.MditaCore},
		{"urn:oasis:names:tc:mdita:xsd:topic.xsd", document.MditaExtended},
		{"urn:oasis:names:tc:dita:xsd:topic.xsd", document.NotMdita},
	}
	for _, c := range cases {
		doc := document.New("file:///project/a.md", 1,
			"---\n$schema: "+c.schema+"\n---\n# Title\n")
		if got := doc.DeclaredProfile(); got != c.want {
			t.Errorf("%s gave profile %v, want %v", c.schema, got, c.want)
		}
	}
}

// A $schema overrides the extension, so a .mdita file can declare core.
func TestSchemaOverridesTheExtension(t *testing.T) {
	doc := document.New("file:///project/a.mdita", 1,
		"---\n$schema: urn:oasis:names:tc:mdita:core:xsd:topic.xsd\n---\n"+
			"# Title\n\nText with a footnote[^1].\n\n[^1]: the note\n")
	if got := doc.DeclaredProfile(); got != document.MditaCore {
		t.Fatalf("profile = %v, want MditaCore", got)
	}
	var found bool
	for _, d := range CheckProfile(doc, config.Default()) {
		if d.Code == CodeCoreProfileFeature {
			found = true
		}
	}
	if !found {
		t.Error("the core profile did not report the footnote")
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

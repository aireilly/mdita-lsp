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

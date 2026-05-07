package hover

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestHoverInlineAttrDomainElement(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\n$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng\n---\n# Topic\n\nShort desc.\n\nClick **Save**{.uicontrol} to save.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	// Position on the {.uicontrol} part
	result := GetHover(doc, document.Position{Line: 7, Character: 20}, f)
	if result == "" {
		t.Fatal("expected hover content for domain element")
	}
	if !strings.Contains(result, "uicontrol") {
		t.Errorf("expected 'uicontrol' in hover, got %q", result)
	}
	if !strings.Contains(result, "ui-d") {
		t.Errorf("expected domain 'ui-d' in hover, got %q", result)
	}
}

func TestHoverInlineAttrStepElement(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\n$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng\n---\n# Task\n\nShort desc.\n\n1. Do this\n   \n   *Result*{.stepresult}: Success.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	// Position on the {.stepresult} part
	result := GetHover(doc, document.Position{Line: 9, Character: 15}, f)
	if result == "" {
		t.Fatal("expected hover content for step element")
	}
	if !strings.Contains(result, "stepresult") {
		t.Errorf("expected 'stepresult' in hover, got %q", result)
	}
}

func TestHoverInlineAttrConditional(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\n$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng\n---\n# Topic\n\nShort desc.\n\nThis is *platform-specific*{platform=\"linux\"} text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	// Position on the {platform="linux"} part
	result := GetHover(doc, document.Position{Line: 7, Character: 27}, f)
	if result == "" {
		t.Fatal("expected hover content for conditional attribute")
	}
	if !strings.Contains(result, "conditional") {
		t.Errorf("expected 'conditional' in hover, got %q", result)
	}
	if !strings.Contains(result, "platform") {
		t.Errorf("expected 'platform' in hover, got %q", result)
	}
}

func TestHoverInlineAttrNotOnAttribute(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"---\n$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng\n---\n# Topic\n\nShort desc.\n\nClick **Save**{.uicontrol} to save.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	// Position on "Click" - not on attribute
	result := GetHover(doc, document.Position{Line: 7, Character: 2}, f)
	// Should return empty or other hover (not inline attr hover)
	if strings.Contains(result, "uicontrol") {
		t.Errorf("expected no inline attr hover away from attribute, got %q", result)
	}
}

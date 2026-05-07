package hover

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestHoverTaskSection(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		heading      string
		wantContains []string
	}{
		{
			name: "prereq section",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Prerequisites

You need admin access.
`,
			heading:      "Prerequisites",
			wantContains: []string{"prereq", "required before"},
		},
		{
			name: "context section",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## About this task

This task installs the software.
`,
			heading:      "About this task",
			wantContains: []string{"context", "Background"},
		},
		{
			name: "result section",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Verification

The software is installed.
`,
			heading:      "Verification",
			wantContains: []string{"result", "Expected result"},
		},
		{
			name: "postreq section",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Next steps

Configure the software.
`,
			heading:      "Next steps",
			wantContains: []string{"postreq", "Follow-up"},
		},
		{
			name: "tasktroubleshooting section",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Troubleshooting {.tasktroubleshooting}

If installation fails, check the logs.
`,
			heading:      "Troubleshooting",
			wantContains: []string{"tasktroubleshooting", "Troubleshooting"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := document.New("test.md", 1, tt.text)
			folder := workspace.NewFolder("/test", config.Default())

			var pos document.Position
			for _, h := range doc.Index.Headings() {
				if h.Text == tt.heading {
					pos = h.Range.Start
					break
				}
			}

			result := GetHover(doc, pos, folder)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("hover result missing %q\ngot: %s", want, result)
				}
			}
		})
	}
}

func TestHoverRelatedLinks(t *testing.T) {
	text := `---
---

# Topic title

Content.

## Related information

- [Link 1](doc1.md)
`

	doc := document.New("test.md", 1, text)
	folder := workspace.NewFolder("/test", config.Default())

	var pos document.Position
	for _, h := range doc.Index.Headings() {
		if h.Text == "Related information" {
			pos = h.Range.Start
			break
		}
	}

	result := GetHover(doc, pos, folder)
	if !strings.Contains(result, "related-links") {
		t.Errorf("expected related-links in hover, got: %s", result)
	}
}

func TestHoverHeadingClass(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		heading      string
		wantContains []string
	}{
		{
			name: "task topic type",
			text: `---
---

# Install the software {.task}

Brief description.
`,
			heading:      "Install the software",
			wantContains: []string{"task", "procedure"},
		},
		{
			name: "concept topic type",
			text: `---
---

# Understanding the software {.concept}

Brief description.
`,
			heading:      "Understanding the software",
			wantContains: []string{"concept", "explanatory"},
		},
		{
			name: "reference topic type",
			text: `---
---

# API reference {.reference}

Brief description.
`,
			heading:      "API reference",
			wantContains: []string{"reference", "lookup"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := document.New("test.md", 1, tt.text)
			folder := workspace.NewFolder("/test", config.Default())

			var pos document.Position
			for _, h := range doc.Index.Headings() {
				if h.Text == tt.heading {
					pos = h.Range.Start
					break
				}
			}

			result := GetHover(doc, pos, folder)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("hover result missing %q\ngot: %s", want, result)
				}
			}
		})
	}
}

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

func TestHoverImplicitContext(t *testing.T) {
	// Line 5 is the shortdesc paragraph; line 7 is the implicit <context>.
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\nShort description.\n\nSome context paragraph.\n\n1. Step one\n2. Step two\n"
	doc := document.New("test.md", 1, text)
	folder := workspace.NewFolder("/test", config.Default())
	result := GetHover(doc, document.Position{Line: 7, Character: 5}, folder)
	if !strings.Contains(result, "context") {
		t.Errorf("expected implicit context hover, got: %s", result)
	}
	if !strings.Contains(result, "Implicit") {
		t.Errorf("expected 'Implicit' in hover, got: %s", result)
	}
}

func TestHoverImplicitResult(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\n1. Step one\n2. Step two\n\nThe software is now installed.\n"
	doc := document.New("test.md", 1, text)
	folder := workspace.NewFolder("/test", config.Default())
	// Line 8 is "The software is now installed." — inside the implicit result range.
	result := GetHover(doc, document.Position{Line: 8, Character: 5}, folder)
	if !strings.Contains(result, "result") {
		t.Errorf("expected implicit result hover, got: %s", result)
	}
	if !strings.Contains(result, "Implicit") {
		t.Errorf("expected 'Implicit' in hover, got: %s", result)
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

// The enclosing steps range always matched first, so the hover for a nested
// region never showed.
func TestNestedRegionHoverWinsOverSteps(t *testing.T) {
	text := "# Task {.task}\n\nShort.\n\n1.  Command\n\n    1.  Sub A\n    2.  Sub B\n"
	doc := document.New("file:///t.md", 1, text)
	f := workspace.NewFolder("file:///", config.Default())
	f.AddDoc(doc)

	got := GetHover(doc, document.Position{Line: 6, Character: 8}, f)
	if !strings.Contains(got, "<substeps>") {
		t.Errorf("hover inside the nested list = %q, want the <substeps> text", got)
	}
}

func TestStepsHoverStillShowsOutsideTheNestedList(t *testing.T) {
	text := "# Task {.task}\n\nShort.\n\n1.  Command\n\n    1.  Sub A\n"
	doc := document.New("file:///t.md", 1, text)
	f := workspace.NewFolder("file:///", config.Default())
	f.AddDoc(doc)

	got := GetHover(doc, document.Position{Line: 4, Character: 5}, f)
	if !strings.Contains(got, "<steps>") {
		t.Errorf("hover on the step line = %q, want the <steps> text", got)
	}
}

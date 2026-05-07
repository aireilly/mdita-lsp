package document

import (
	"testing"
)

func TestResolveTaskSections(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected map[string]TaskSectionKind
	}{
		{
			name: "task by schema with standard sections",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Prerequisites

You need admin access.

## About this task

This task installs the software.

## Verification

The software is installed.

## Next steps

Configure the software.
`,
			expected: map[string]TaskSectionKind{
				"Prerequisites":   TaskSectionPrereq,
				"About this task": TaskSectionContext,
				"Verification":    TaskSectionResult,
				"Next steps":      TaskSectionPostreq,
			},
		},
		{
			name: "task by class attribute",
			text: `---
---

# Install the software {.task}

Brief description.

## Prerequisites

You need admin access.

## Verification

The software is installed.
`,
			expected: map[string]TaskSectionKind{
				"Prerequisites": TaskSectionPrereq,
				"Verification":  TaskSectionResult,
			},
		},
		{
			name: "task sections by class attributes",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Before you begin {.prereq}

You need admin access.

## Background {.context}

This task installs the software.

## What to do next {.postreq}

Configure the software.
`,
			expected: map[string]TaskSectionKind{
				"Before you begin": TaskSectionPrereq,
				"Background":       TaskSectionContext,
				"What to do next":  TaskSectionPostreq,
			},
		},
		{
			name: "troubleshooting section",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Prerequisites

You need admin access.

## Troubleshooting {.tasktroubleshooting}

If installation fails, check the logs.
`,
			expected: map[string]TaskSectionKind{
				"Prerequisites":   TaskSectionPrereq,
				"Troubleshooting": TaskSectionTroubleshooting,
			},
		},
		{
			name: "non-task topic has no task sections",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:concept.xsd
---

# Understanding the software

## Prerequisites

This heading is not recognized as a task section.
`,
			expected: map[string]TaskSectionKind{
				"Prerequisites": TaskSectionNone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := New("test.md", 1, tt.text)

			for headingText, expectedKind := range tt.expected {
				found := false
				for _, h := range doc.Index.Headings() {
					if h.Text == headingText {
						found = true
						if h.TaskSection != expectedKind {
							t.Errorf("heading %q: got TaskSection=%v, want %v",
								headingText, h.TaskSection, expectedKind)
						}
					}
				}
				if !found {
					t.Errorf("heading %q not found", headingText)
				}
			}
		})
	}
}

func TestRelatedLinksDetection(t *testing.T) {
	text := `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install {.task}

Short desc.

1. Do the thing.

## Related information

- [Concept](concept.md)
- [Reference](reference.md)
`
	doc := New("file:///test.md", 1, text)
	if doc.RelLinks == nil {
		t.Fatal("expected RelLinks to be set")
	}
	if doc.RelLinks.HeadingLine != 10 {
		t.Errorf("RelLinks.HeadingLine = %d, want 10", doc.RelLinks.HeadingLine)
	}
	if len(doc.RelLinks.Links) != 2 {
		t.Errorf("len(RelLinks.Links) = %d, want 2", len(doc.RelLinks.Links))
	}
	if len(doc.RelLinks.Links) >= 2 {
		if doc.RelLinks.Links[0].Text != "Concept" {
			t.Errorf("RelLinks.Links[0].Text = %q, want %q", doc.RelLinks.Links[0].Text, "Concept")
		}
		if doc.RelLinks.Links[0].URL != "concept.md" {
			t.Errorf("RelLinks.Links[0].URL = %q, want %q", doc.RelLinks.Links[0].URL, "concept.md")
		}
		if doc.RelLinks.Links[1].Text != "Reference" {
			t.Errorf("RelLinks.Links[1].Text = %q, want %q", doc.RelLinks.Links[1].Text, "Reference")
		}
		if doc.RelLinks.Links[1].URL != "reference.md" {
			t.Errorf("RelLinks.Links[1].URL = %q, want %q", doc.RelLinks.Links[1].URL, "reference.md")
		}
	}
}

func TestRelatedLinksStopsAtNextSection(t *testing.T) {
	text := `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install {.task}

Short desc.

1. Do the thing.

## Related information

- [Concept](concept.md)
- [Reference](reference.md)

## Next section

- [Should not be included](other.md)
`
	doc := New("file:///test.md", 1, text)
	if doc.RelLinks == nil {
		t.Fatal("expected RelLinks to be set")
	}
	if len(doc.RelLinks.Links) != 2 {
		t.Errorf("len(RelLinks.Links) = %d, want 2 (should not include links from next section)", len(doc.RelLinks.Links))
	}
}

func TestResolveRelatedLinks(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected map[string]bool
	}{
		{
			name: "related information heading",
			text: `---
---

# Topic title

Content.

## Related information

- [Link 1](doc1.md)
`,
			expected: map[string]bool{
				"Related information": true,
			},
		},
		{
			name: "related links heading",
			text: `---
---

# Topic title

Content.

## Related links

- [Link 1](doc1.md)
`,
			expected: map[string]bool{
				"Related links": true,
			},
		},
		{
			name: "related-links class attribute",
			text: `---
---

# Topic title

Content.

## See also {.related-links}

- [Link 1](doc1.md)
`,
			expected: map[string]bool{
				"See also": true,
			},
		},
		{
			name: "case insensitive detection",
			text: `---
---

# Topic title

Content.

## RELATED INFORMATION

- [Link 1](doc1.md)
`,
			expected: map[string]bool{
				"RELATED INFORMATION": true,
			},
		},
		{
			name: "non-related heading not marked",
			text: `---
---

# Topic title

Content.

## Other section

Some content.
`,
			expected: map[string]bool{
				"Other section": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := New("test.md", 1, tt.text)

			for headingText, expectedIsRelLinks := range tt.expected {
				found := false
				for _, h := range doc.Index.Headings() {
					if h.Text == headingText {
						found = true
						if h.IsRelLinks != expectedIsRelLinks {
							t.Errorf("heading %q: got IsRelLinks=%v, want %v",
								headingText, h.IsRelLinks, expectedIsRelLinks)
						}
					}
				}
				if !found {
					t.Errorf("heading %q not found", headingText)
				}
			}
		})
	}
}

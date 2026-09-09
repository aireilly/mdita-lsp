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

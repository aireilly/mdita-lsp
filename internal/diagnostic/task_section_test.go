package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestCheckTaskSections(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantCodes []string
	}{
		{
			name: "duplicate task section",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Prerequisites

You need admin access.

## Prerequisites

You also need network access.
`,
			wantCodes: []string{CodeDuplicateTaskSection},
		},
		{
			name: "task sections out of order",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Next steps

Configure the software.

## Prerequisites

You need admin access.
`,
			wantCodes: []string{CodeTaskSectionOutOfOrder},
		},
		{
			name: "task sections in correct order",
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
			wantCodes: []string{},
		},
		{
			name: "troubleshooting section at end",
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
			wantCodes: []string{},
		},
		{
			name: "non-task topic has no diagnostics",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:concept.xsd
---

# Understanding the software

## Next steps

This is not a task section.

## Prerequisites

Neither is this.
`,
			wantCodes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := document.New("test.md", 1, tt.text)
			diags := checkTaskSections(doc)

			if len(diags) != len(tt.wantCodes) {
				t.Fatalf("got %d diagnostics, want %d", len(diags), len(tt.wantCodes))
			}

			for i, want := range tt.wantCodes {
				if diags[i].Code != want {
					t.Errorf("diagnostic %d: got code=%q, want %q", i, diags[i].Code, want)
				}
			}
		})
	}
}

package completion

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestCompleteTaskSectionHeading(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		cursorPos document.Position
		wantItems []string
	}{
		{
			name: "complete task section heading in task topic",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Pre
`,
			cursorPos: document.Position{Line: 8, Character: 6},
			wantItems: []string{"Prerequisites"},
		},
		{
			name: "complete all task sections",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

##
`,
			cursorPos: document.Position{Line: 8, Character: 3},
			wantItems: []string{"Prerequisites", "About this task", "Procedure", "Verification", "Next steps"},
		},
		{
			name: "exclude existing sections",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software

Brief description.

## Prerequisites

You need admin access.

##
`,
			cursorPos: document.Position{Line: 12, Character: 3},
			wantItems: []string{"About this task", "Verification", "Next steps"},
		},
		{
			name: "no completion in non-task topic",
			text: `---
$schema: urn:oasis:names:tc:dita:xsd:concept.xsd
---

# Understanding the software

##
`,
			cursorPos: document.Position{Line: 6, Character: 3},
			wantItems: []string{},
		},
		{
			name: "completion in task by class",
			text: `---
---

# Install the software {.task}

Brief description.

## Ver
`,
			cursorPos: document.Position{Line: 7, Character: 6},
			wantItems: []string{"Verification"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := document.New("test.md", 1, tt.text)
			folder := workspace.NewFolder("/test", config.Default())
			items := Complete(doc, tt.cursorPos, folder)

			if len(tt.wantItems) == 0 {
				if len(items) > 0 {
					t.Errorf("expected no completions, got %d", len(items))
				}
				return
			}

			for _, want := range tt.wantItems {
				found := false
				for _, item := range items {
					if item.Label == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected completion item %q not found", want)
				}
			}
		})
	}
}

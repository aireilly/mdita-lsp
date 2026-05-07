package document

import (
	"testing"
)

func TestFullExtendedDocument(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n\n# Install the software {.task}\n\nShort description of the installation.\n\n## Prerequisites\n\nYou need administrator access.\n\n{platform=\"linux\"}\n\n- Ensure you have sudo privileges.\n\n## About this task\n\nThis procedure installs the base package.\n\n1. Click **File > Open**{.menucascade} to open the dialog.\n\n2. Edit `config.yaml`{.filepath} to set options.\n\n3. Run the `installer`{.cmdname} command.\n\n## Verification\n\nThe software is now installed.\n\n## Next steps\n\nConfigure the license key.\n\n## Related information\n\n- [Concept](concept.md)\n- [Reference](reference.md)\n"
	doc := New("file:///test.md", 1, text)

	// Heading attributes
	title := doc.Index.Title()
	if title == nil {
		t.Fatal("no title")
	}
	if title.Attributes == nil || len(title.Attributes.Classes) == 0 || title.Attributes.Classes[0] != "task" {
		t.Error("title should have .task class")
	}

	// Task sections
	headings := doc.Index.Headings()
	sectionCount := 0
	for _, h := range headings {
		if h.TaskSection != TaskSectionNone {
			sectionCount++
		}
	}
	if sectionCount != 4 {
		t.Errorf("task sections = %d, want 4", sectionCount)
	}

	// Related links
	relLinksFound := false
	for _, h := range headings {
		if h.IsRelLinks {
			relLinksFound = true
			break
		}
	}
	if !relLinksFound {
		t.Error("expected related links heading")
	}

	// Inline attributes
	if len(doc.InlineAttrs) < 3 {
		t.Errorf("inline attrs = %d, want >= 3", len(doc.InlineAttrs))
	}

	// Block attributes
	if len(doc.BlockAttrs) < 1 {
		t.Errorf("block attrs = %d, want >= 1", len(doc.BlockAttrs))
	}

	// HasAttributes
	if !doc.Index.Features.HasAttributes {
		t.Error("expected HasAttributes to be true")
	}
}

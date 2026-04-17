package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func makeDoc(uri string, lines ...string) *document.Document {
	text := ""
	for i, l := range lines {
		text += l
		if i < len(lines)-1 {
			text += "\n"
		}
	}
	return document.New(uri, 1, text)
}

func makeFolder(docs ...*document.Document) *workspace.Folder {
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	for _, d := range docs {
		f.AddDoc(d)
	}
	return f
}

func TestMissingYamlFrontMatter(t *testing.T) {
	doc := makeDoc("file:///project/doc.md", "# Title", "", "Some text.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeMissingYamlFrontMatter {
			found = true
		}
	}
	if !found {
		t.Error("expected MissingYamlFrontMatter diagnostic")
	}
}

func TestNoMissingYamlWhenPresent(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "Short desc.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeMissingYamlFrontMatter {
			t.Error("should not report MissingYamlFrontMatter when YAML is present")
		}
	}
}

func TestMissingShortDescription(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "## Next Section")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeMissingShortDescription {
			found = true
		}
	}
	if !found {
		t.Error("expected MissingShortDescription diagnostic")
	}
}

func TestInvalidHeadingHierarchy(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "Short desc.", "", "### Skipped H2")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeInvalidHeadingHierarchy {
			found = true
		}
	}
	if !found {
		t.Error("expected InvalidHeadingHierarchy diagnostic")
	}
}

func TestTaskMissingProcedure(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:task.xsd", "---",
		"# Task Title", "", "Some text.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeTaskMissingProcedure {
			found = true
		}
	}
	if !found {
		t.Error("expected TaskMissingProcedure diagnostic")
	}
}

func TestConceptHasProcedure(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:concept.xsd", "---",
		"# Concept Title", "", "Some text.", "", "1. step one", "2. step two")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeConceptHasProcedure {
			found = true
		}
	}
	if !found {
		t.Error("expected ConceptHasProcedure diagnostic")
	}
}

func TestBrokenLink(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"# Title", "", "[[nonexistent]]")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeBrokenLink {
			found = true
		}
	}
	if !found {
		t.Error("expected BrokenLink diagnostic")
	}
}

func TestValidLink(t *testing.T) {
	doc1 := makeDoc("file:///project/intro.md", "# Intro", "", "Some text.")
	doc2 := makeDoc("file:///project/doc.md", "# Doc", "", "[[intro]]")
	f := makeFolder(doc1, doc2)
	diags := Check(doc2, f)

	for _, d := range diags {
		if d.Code == CodeBrokenLink {
			t.Errorf("should not report BrokenLink for valid wiki link, got: %s", d.Message)
		}
	}
}

func TestUnknownAdmonitionType(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "Short desc.", "", "!!! invalid", "    content")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeUnknownAdmonitionType {
			found = true
		}
	}
	if !found {
		t.Error("expected UnknownAdmonitionType diagnostic")
	}
}

func TestFootnoteRefWithoutDef(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---",
		"# Title", "", "Short desc.", "",
		"See this[^missing] for details.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeFootnoteRefWithoutDef {
			found = true
		}
	}
	if !found {
		t.Error("expected FootnoteRefWithoutDef diagnostic")
	}
}

func TestFootnoteDefWithoutRef(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---",
		"# Title", "", "Short desc.", "",
		"Some text.", "",
		"[^orphan]: This definition is never referenced")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeFootnoteDefWithoutRef {
			found = true
		}
	}
	if !found {
		t.Error("expected FootnoteDefWithoutRef diagnostic")
	}
}

func TestMatchedFootnotesNoDiagnostic(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---",
		"# Title", "", "Short desc.", "",
		"See this[^note] for details.", "",
		"[^note]: A valid footnote")
	f := makeFolder(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeFootnoteRefWithoutDef || d.Code == CodeFootnoteDefWithoutRef {
			t.Errorf("should not report footnote diagnostics for matched pairs, got: %s", d.Message)
		}
	}
}

func TestMditaDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Core.Mdita.Enable = false
	cfg.Diagnostics.MditaCompliance = false

	doc := makeDoc("file:///project/doc.md", "# Title")
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeMissingYamlFrontMatter {
			t.Error("should not report MDITA diagnostics when disabled")
		}
	}
}

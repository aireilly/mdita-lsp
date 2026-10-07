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

func TestDiagnosticCodeValues(t *testing.T) {
	if CodeAmbiguousLink != "1" {
		t.Errorf("CodeAmbiguousLink = %q, want %q", CodeAmbiguousLink, "1")
	}
	if CodeFootnoteRefOrphan != "8" {
		t.Errorf("CodeFootnoteRefOrphan = %q, want %q", CodeFootnoteRefOrphan, "8")
	}
	if CodeUnresolvedKeyref != "10" {
		t.Errorf("CodeUnresolvedKeyref = %q, want %q", CodeUnresolvedKeyref, "10")
	}
	if CodeCoreProfileFeature != "13" {
		t.Errorf("CodeCoreProfileFeature = %q, want %q", CodeCoreProfileFeature, "13")
	}
	if CodeTaskTypeInNonTask != "18" {
		t.Errorf("CodeTaskTypeInNonTask = %q, want %q", CodeTaskTypeInNonTask, "18")
	}
}

func TestCoreProfileFootnoteWarning(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:topic.xsd\n---\n# Title\n\nText with footnote[^1].\n\n[^1]: Footnote text\n"
	doc := document.New("file:///test.md", 1, text)
	cfg := config.Default()
	core := config.ProfileCore
	cfg.Core.Mdita.Profile = &core
	diags := CheckProfile(doc, cfg)
	found := false
	for _, d := range diags {
		if d.Code == CodeCoreProfileFeature {
			found = true
		}
	}
	if !found {
		t.Error("expected core profile feature warning for footnote")
	}
}

func TestMissingYamlFrontMatter(t *testing.T) {
	doc := makeDoc("file:///project/doc.md", "# Title", "", "Some text.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeMissingFrontMatter {
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
		if d.Code == CodeMissingFrontMatter {
			t.Error("should not report MissingYamlFrontMatter when YAML is present")
		}
	}
}

func TestMissingShortDescription(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:topic.xsd", "---", "# Title", "", "## Next Section")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeMissingShortDesc {
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
		if d.Code == CodeHeadingHierarchy {
			found = true
		}
	}
	if !found {
		t.Error("expected InvalidHeadingHierarchy diagnostic")
	}
}

func TestBrokenLink(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"# Title", "", "[missing](nonexistent.md)")
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

func TestFootnoteRefWithoutDef(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---",
		"# Title", "", "Short desc.", "",
		"See this[^missing] for details.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeFootnoteRefOrphan {
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
		if d.Code == CodeFootnoteDefOrphan {
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
		if d.Code == CodeFootnoteRefOrphan || d.Code == CodeFootnoteDefOrphan {
			t.Errorf("should not report footnote diagnostics for matched pairs, got: %s", d.Message)
		}
	}
}

func TestLinkValidationDisabled(t *testing.T) {
	cfg := config.Default()
	no := false
	cfg.Diagnostics.LinkValidation = &no

	doc := makeDoc("file:///project/doc.md",
		"# Title\n\n[missing](nonexistent.md)\n")
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeBrokenLink || d.Code == CodeAmbiguousLink {
			t.Errorf("should not report link diagnostics when disabled, got code %s", d.Code)
		}
	}
}

func TestNbspDetectionDisabled(t *testing.T) {
	cfg := config.Default()
	no := false
	cfg.Diagnostics.NbspDetection = &no

	doc := makeDoc("file:///project/doc.md",
		"# Title\u00a0Here\n")
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeNBSP {
			t.Error("should not report NBSP diagnostics when disabled")
		}
	}
}

func TestMditaDisabled(t *testing.T) {
	cfg := config.Default()
	no := false
	cfg.Core.Mdita.Enable = &no
	cfg.Diagnostics.MditaCompliance = &no

	doc := makeDoc("file:///project/doc.md", "# Title")
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeMissingFrontMatter {
			t.Error("should not report MDITA diagnostics when disabled")
		}
	}
}

func TestBrokenMdLink(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"# Title", "", "[setup](nonexistent.md)")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeBrokenLink && d.Message == "Link to non-existent file 'nonexistent.md'" {
			found = true
		}
	}
	if !found {
		t.Error("expected broken link diagnostic for nonexistent.md")
	}
}

func TestValidMdLink(t *testing.T) {
	target := document.New("file:///project/install.md", 1, "# Install\n")
	source := makeDoc("file:///project/doc.md",
		"# Title", "", "[setup](install.md)")
	f := makeFolder(source, target)
	diags := Check(source, f)

	for _, d := range diags {
		if d.Code == CodeBrokenLink && d.Message == "Link to non-existent file 'install.md'" {
			t.Error("should not report broken link for existing file")
		}
	}
}

func TestBrokenMdLinkAnchor(t *testing.T) {
	target := document.New("file:///project/install.md", 1, "# Install\n")
	source := makeDoc("file:///project/doc.md",
		"# Title", "", "[setup](install.md#nonexistent)")
	f := makeFolder(source, target)
	diags := Check(source, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeBrokenLink && d.Message == "Link to non-existent heading '#nonexistent' in 'install.md'" {
			found = true
		}
	}
	if !found {
		t.Error("expected broken link diagnostic for nonexistent heading in install.md")
	}
}

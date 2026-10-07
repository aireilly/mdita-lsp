package diagnostic

import (
	"strings"
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
	// A .md topic is Markdown DITA unless the workspace says its maps give
	// these files format="mdita".
	on := true
	cfg.Core.Mdita.ApplyToMarkdown = &on
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

func linkFolder(docs ...*document.Document) *workspace.Folder {
	f := workspace.NewFolder("file:///project", config.Default())
	for _, d := range docs {
		f.AddDoc(d)
	}
	return f
}

func codesFor(doc *document.Document, f *workspace.Folder) []string {
	var codes []string
	for _, d := range Check(doc, f) {
		codes = append(codes, d.Code)
	}
	return codes
}

func hasCode(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

// The plug-in gives a root-relative href scope="external" and never resolves
// it against the source tree.
func TestRootRelativeLinkIsNotReported(t *testing.T) {
	doc := document.New("file:///project/a.md", 1, "# A\n\n[x](/target.md)\n")
	codes := codesFor(doc, linkFolder(doc))
	if hasCode(codes, CodeBrokenLink) {
		t.Errorf("root-relative link reported broken: %v", codes)
	}
}

func TestLinkWithASchemeIsNotReported(t *testing.T) {
	doc := document.New("file:///project/a.md", 1,
		"# A\n\n[x](ftp://example.com/f.md) and [y](mailto:a@example.com)\n")
	codes := codesFor(doc, linkFolder(doc))
	if hasCode(codes, CodeBrokenLink) {
		t.Errorf("a link with a scheme was reported broken: %v", codes)
	}
}

// The plug-in keeps "my%20file.md" as the href and the build resolves it to
// "my file.md" on disk.
func TestPercentEscapedLinkResolves(t *testing.T) {
	target := document.New("file:///project/my%20file.md", 1, "# My file\n")
	doc := document.New("file:///project/a.md", 1, "# A\n\n[x](my%20file.md)\n")
	codes := codesFor(doc, linkFolder(target, doc))
	if hasCode(codes, CodeBrokenLink) {
		t.Errorf("percent-escaped link reported broken: %v", codes)
	}
}

// A link to a file name that only exists elsewhere is a build failure, not a
// working link.
func TestLinkToASameNamedFileElsewhereIsReported(t *testing.T) {
	root := document.New("file:///project/target.md", 1, "# Root\n")
	doc := document.New("file:///project/sub/stem.md", 1, "# Stem\n\n[t](target.md)\n")
	codes := codesFor(doc, linkFolder(root, doc))
	if !hasCode(codes, CodeAmbiguousLink) {
		t.Errorf("expected the ambiguous-link diagnostic, got %v", codes)
	}
}

// Generated ids carry a -1 suffix for a duplicate heading and turn '_' into
// a dash; both were reported broken.
func TestFragmentsMatchingGeneratedIDs(t *testing.T) {
	target := document.New("file:///project/guide.md", 1,
		"# Guide\n\n## My_var config\n\n## Setup\n\n## Setup\n")
	doc := document.New("file:///project/a.md", 1,
		"# A\n\n[a](guide.md#my-var-config) [b](guide.md#setup-1)\n")
	codes := codesFor(doc, linkFolder(target, doc))
	if hasCode(codes, CodeBrokenLink) {
		t.Errorf("a working fragment was reported broken: %v", codes)
	}
}

// A document without front matter still gets the heading-skip check, and the
// skip is an error because the build throws.
func TestHeadingSkipWithoutFrontMatterIsAnError(t *testing.T) {
	doc := document.New("file:///project/a.md", 1, "# Title\n\n### Too deep\n")
	f := linkFolder(doc)
	var found bool
	for _, d := range Check(doc, f) {
		if d.Code == CodeHeadingHierarchy {
			found = true
			if d.Severity != SeverityError {
				t.Errorf("severity = %v, want error", d.Severity)
			}
		}
	}
	if !found {
		t.Error("no heading-hierarchy diagnostic for a file without front matter")
	}
}

func TestMissingFrontMatterIsInformation(t *testing.T) {
	doc := document.New("file:///project/a.md", 1, "# Title\n\ntext\n")
	for _, d := range Check(doc, linkFolder(doc)) {
		if d.Code == CodeMissingFrontMatter && d.Severity != SeverityInfo {
			t.Errorf("severity = %v, want information", d.Severity)
		}
	}
}

// A .mdita file is MDITA whether or not it declares a $schema, and a level 3
// heading there fails the build.
func TestMditaExtensionCapsHeadingsWithoutASchema(t *testing.T) {
	doc := document.New("file:///project/a.mdita", 1, "# Title\n\n## Section\n\n### Too deep\n")
	var found bool
	for _, d := range Check(doc, linkFolder(doc)) {
		if d.Code == CodeMditaProfileFeature {
			found = true
			if d.Severity != SeverityError {
				t.Errorf("severity = %v, want error", d.Severity)
			}
		}
	}
	if !found {
		t.Error("no MDITA profile diagnostic for a level 3 heading in a .mdita file")
	}
}

func TestFootnotesInsideAFencedBlockAreNotReported(t *testing.T) {
	doc := document.New("file:///project/a.md", 1,
		"# Title\n\n```\n[^1] sample\n```\n")
	codes := codesFor(doc, linkFolder(doc))
	if hasCode(codes, CodeFootnoteRefOrphan) {
		t.Errorf("a footnote inside a fenced block was reported: %v", codes)
	}
}

// TopicRenderer throws "Level 2 section title must be higher level than
// parent topic title 2" when a section heading sits at the level of the
// nested topic above it. Only skipped levels used to be checked.
func TestSectionAtTheSameLevelAsItsParentTopicIsAnError(t *testing.T) {
	doc := document.New("file:///project/a.md", 1,
		"# Task {.task}\n\nShort.\n\n## Details\n\ntext\n\n## Procedure\n\n1. Do it\n")
	var found bool
	for _, d := range Check(doc, linkFolder(doc)) {
		if d.Code == CodeHeadingHierarchy && strings.Contains(d.Message, "section title") {
			found = true
			if d.Severity != SeverityError {
				t.Errorf("severity = %v, want error", d.Severity)
			}
		}
	}
	if !found {
		t.Error("no diagnostic for a section at the level of its parent topic")
	}
}

func TestSectionBelowItsParentTopicIsFine(t *testing.T) {
	doc := document.New("file:///project/a.md", 1,
		"# Task {.task}\n\nShort.\n\n## Procedure\n\n1. Do it\n")
	for _, d := range Check(doc, linkFolder(doc)) {
		if d.Code == CodeHeadingHierarchy {
			t.Errorf("unexpected heading diagnostic: %s", d.Message)
		}
	}
}

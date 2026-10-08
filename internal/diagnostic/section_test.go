package diagnostic

import (
	"strings"
	"testing"
)

func findCode(diags []Diagnostic, code string) *Diagnostic {
	for i := range diags {
		if diags[i].Code == code {
			return &diags[i]
		}
	}
	return nil
}

// DITA sections do not nest, so a heading below a section in a concept or a
// reference fails the build.
func TestNestedSectionInTypedTopic(t *testing.T) {
	doc := makeDoc("file:///project/c.md", "# C {.concept}", "", "## Background", "", "### Detail", "")
	diags := Check(doc, makeFolder(doc))
	d := findCode(diags, CodeNestedSection)
	if d == nil {
		t.Fatalf("expected code %s, got %+v", CodeNestedSection, diags)
	}
	if d.Severity != SeverityError {
		t.Errorf("Severity = %d, want %d", d.Severity, SeverityError)
	}
	want := `"### Detail" can't go here: DITA sections don't nest. ` +
		`Make it a "##" section, or move it to its own topic.`
	if d.Message != want {
		t.Errorf("Message = %q, want %q", d.Message, want)
	}
	if d.Range.Start.Line != 4 {
		t.Errorf("Range.Start.Line = %d, want 4", d.Range.Start.Line)
	}
}

func TestNoNestedSectionDiagnosticInGenericTopic(t *testing.T) {
	doc := makeDoc("file:///project/t.md", "# T", "", "## A", "", "### B", "")
	diags := Check(doc, makeFolder(doc))
	if d := findCode(diags, CodeNestedSection); d != nil {
		t.Errorf("unexpected code %s on a generic topic: %+v", CodeNestedSection, d)
	}
}

func TestNoNestedSectionDiagnosticInTask(t *testing.T) {
	doc := makeDoc("file:///project/t.md", "# T {.task}", "", "## Details", "", "### More", "")
	diags := Check(doc, makeFolder(doc))
	if d := findCode(diags, CodeNestedSection); d != nil {
		t.Errorf("unexpected code %s on a task: %+v", CodeNestedSection, d)
	}
}

// A {.section} heading at the level of a heading that already opened a nested
// topic is the ordering trap, and it is an error in generic topics only.
func TestSectionAfterNestedTopic(t *testing.T) {
	doc := makeDoc("file:///project/t.md", "# A", "", "## A.1", "", "## A.2 {.section}", "")
	diags := Check(doc, makeFolder(doc))
	d := findCode(diags, CodeSectionAfterNestedTopic)
	if d == nil {
		t.Fatalf("expected code %s, got %+v", CodeSectionAfterNestedTopic, diags)
	}
	if d.Severity != SeverityError {
		t.Errorf("Severity = %d, want %d", d.Severity, SeverityError)
	}
	want := `"## A.2 {.section}" can't be a section here: "## A.1" above it opened a nested topic. ` +
		`Add {.section} to "## A.1", or remove it from "## A.2".`
	if d.Message != want {
		t.Errorf("Message = %q, want %q", d.Message, want)
	}
}

func TestSectionInTypedTopicIsNotTheOrderingTrap(t *testing.T) {
	doc := makeDoc("file:///project/c.md", "# C {.concept}", "", "## A", "", "## B", "")
	diags := Check(doc, makeFolder(doc))
	for _, d := range diags {
		if d.Code == CodeSectionAfterNestedTopic || d.Code == CodeNestedSection {
			t.Errorf("unexpected %s: %s", d.Code, d.Message)
		}
	}
}

// A link to a section needs the topic id, because a section is not a topic.
func TestSectionLinkNeedsTopicID(t *testing.T) {
	target := makeDoc("file:///project/c.md", "# C {.concept}", "", "## Background", "")
	src := makeDoc("file:///project/t.md", "# T", "", "See [x](c.md#background).", "")
	diags := Check(src, makeFolder(target, src))
	d := findCode(diags, CodeSectionLinkNeedsTopicID)
	if d == nil {
		t.Fatalf("expected code %s, got %+v", CodeSectionLinkNeedsTopicID, diags)
	}
	if d.Severity != SeverityWarning {
		t.Errorf("Severity = %d, want %d", d.Severity, SeverityWarning)
	}
	if !strings.Contains(d.Message, "c.md#c/background") {
		t.Errorf("Message = %q, expected the qualified form", d.Message)
	}
}

func TestQualifiedSectionLinkResolves(t *testing.T) {
	target := makeDoc("file:///project/c.md", "# C {.concept}", "", "## Background", "")
	src := makeDoc("file:///project/t.md", "# T", "", "See [x](c.md#c/background).", "")
	diags := Check(src, makeFolder(target, src))
	for _, d := range diags {
		if d.Code == CodeBrokenLink || d.Code == CodeSectionLinkNeedsTopicID {
			t.Errorf("unexpected %s: %s", d.Code, d.Message)
		}
	}
}

// A link to the topic itself is unaffected, whether or not it names the topic id.
func TestTopicLinkToTypedTopicIsUnchanged(t *testing.T) {
	target := makeDoc("file:///project/c.md", "# C {.concept}", "", "## Background", "")
	src := makeDoc("file:///project/t.md", "# T", "", "See [x](c.md) and [y](c.md#c).", "")
	diags := Check(src, makeFolder(target, src))
	for _, d := range diags {
		if d.Code == CodeSectionLinkNeedsTopicID || d.Code == CodeBrokenLink {
			t.Errorf("unexpected %s: %s", d.Code, d.Message)
		}
	}
}

func TestNoSectionLinkWarningForGenericTopic(t *testing.T) {
	target := makeDoc("file:///project/t2.md", "# T2", "", "## A", "")
	src := makeDoc("file:///project/t.md", "# T", "", "See [x](t2.md#a).", "")
	diags := Check(src, makeFolder(target, src))
	if d := findCode(diags, CodeSectionLinkNeedsTopicID); d != nil {
		t.Errorf("unexpected code %s: %s", d.Code, d.Message)
	}
}

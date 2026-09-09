package document

import "testing"

func kinds(doc *Document, want ImplicitSectionKind) []ImplicitSection {
	var out []ImplicitSection
	for _, s := range doc.ImplicitSections {
		if s.Kind == want {
			out = append(out, s)
		}
	}
	return out
}

func hasKind(doc *Document, want ImplicitSectionKind) bool {
	return len(kinds(doc, want)) > 0
}

// Mirrors org.lwdita src/test/resources/markdown/task/task_context_with_ul.md:
// the unordered list is followed by the steps list, so it stays in <context>.
func TestBodyUnorderedListBeforeStepsStaysContext(t *testing.T) {
	text := "# Deploy the app {.task}\n\nGather these items:\n\n-   SSH key\n-   Config file\n\n1.  Connect to the server.\n2.  Run the deploy script.\n"
	doc := New("file:///t.md", 1, text)

	if !hasKind(doc, ImplicitBodyList) {
		t.Error("expected the unordered list to be a body-ul inside <context>")
	}
	if !hasKind(doc, ImplicitSteps) {
		t.Error("expected the ordered list to be <steps>")
	}
	if hasKind(doc, ImplicitStepsUnordered) {
		t.Error("the unordered list must not be <steps-unordered> here")
	}
}

// A lone body-level unordered list becomes <steps-unordered>.
func TestBodyUnorderedListAloneIsStepsUnordered(t *testing.T) {
	text := "# Deploy {.task}\n\nShort description.\n\n-   Connect to the server.\n-   Run the deploy script.\n"
	doc := New("file:///t.md", 1, text)

	if !hasKind(doc, ImplicitStepsUnordered) {
		t.Errorf("expected <steps-unordered>, got %+v", doc.ImplicitSections)
	}
}

// Mirrors task_context_with_ol.md: the ordered list restarts its numbering, so
// the plug-in splits it into a body-ol and the real steps.
func TestOrderedListNumberingResetSplits(t *testing.T) {
	text := "# Install software {.task}\n\nPrerequisites:\n\n1.  Get admin access\n2.  Back up your data\n\n1.  Run the installer.\n2.  Follow the prompts.\n"
	doc := New("file:///t.md", 1, text)

	body := kinds(doc, ImplicitBodyList)
	steps := kinds(doc, ImplicitSteps)
	if len(body) != 1 || len(steps) != 1 {
		t.Fatalf("expected one body-ol and one steps block, got %+v", doc.ImplicitSections)
	}
	if body[0].Range.Start.Line != 4 || steps[0].Range.Start.Line != 7 {
		t.Errorf("split at the wrong line: body %+v steps %+v", body[0].Range, steps[0].Range)
	}
}

// Mirrors task_procedure_marker.md: the Procedure heading maps to no element
// and the list after it is the steps list, even though explicit sections
// precede it.
func TestStepsMarkerHeading(t *testing.T) {
	text := "# Install the software {.task}\n\nThis procedure installs the software.\n\n## Prerequisites\n\nYou have administrator access.\n\n## About this task\n\nThe installer applies defaults.\n\n## Procedure\n\n1.  Run the installer.\n2.  Accept the licence.\n\n## Verification\n\nConfirm the install.\n"
	doc := New("file:///t.md", 1, text)

	var marker *Heading
	for _, h := range doc.Index.Headings() {
		if h.Text == "Procedure" {
			marker = h
		}
	}
	if marker == nil || marker.TaskSection != TaskSectionSteps {
		t.Fatalf("Procedure heading not recognised as a steps marker: %+v", marker)
	}
	steps := kinds(doc, ImplicitSteps)
	if len(steps) != 1 {
		t.Fatalf("expected one steps block, got %+v", doc.ImplicitSections)
	}
	if steps[0].Range.Start.Line != 14 {
		t.Errorf("steps start line = %d, want 14", steps[0].Range.Start.Line)
	}
}

// The first paragraph under a typed title is the shortdesc, not <context>.
func TestShortdescParagraphIsNotContext(t *testing.T) {
	text := "# Task {.task}\n\nContext\n\n1.  Run the installer.\n"
	doc := New("file:///t.md", 1, text)

	if hasKind(doc, ImplicitContext) {
		t.Errorf("the shortdesc paragraph must not be <context>: %+v", doc.ImplicitSections)
	}
	if !hasKind(doc, ImplicitSteps) {
		t.Error("expected <steps>")
	}
}

func TestNestedListsInSteps(t *testing.T) {
	text := "# Task {.task}\n\nShort.\n\n1.  Command\n\n    1.  Sub A\n    2.  Sub B\n\n2.  Choose\n\n    *   First\n    *   Second\n"
	doc := New("file:///t.md", 1, text)

	if !hasKind(doc, ImplicitSubsteps) {
		t.Error("expected <substeps> for the nested ordered list")
	}
	if !hasKind(doc, ImplicitChoices) {
		t.Error("expected <choices> for the nested unordered list")
	}
}

func TestChoicetableInStep(t *testing.T) {
	text := "# Task {.task}\n\n1.  Select an option:\n\n    | Option | Description |\n    |--------|-------------|\n    | Fast   | Quick setup |\n"
	doc := New("file:///t.md", 1, text)

	if !hasKind(doc, ImplicitChoicetable) {
		t.Errorf("expected <choicetable>, got %+v", doc.ImplicitSections)
	}
}

func TestConfigurableTaskSectionTitles(t *testing.T) {
	defer SetImplicitTaskSectionTitles(nil)
	SetImplicitTaskSectionTitles(map[string][]string{
		"prereq": {"voraussetzungen"},
		"steps":  {"vorgehensweise"},
	})

	text := "# Task {.task}\n\nShort.\n\n## Voraussetzungen\n\nAdmin access.\n\n## Vorgehensweise\n\n1.  Run it.\n"
	doc := New("file:///t.md", 1, text)

	got := map[string]TaskSectionKind{}
	for _, h := range doc.Index.Headings() {
		got[h.Text] = h.TaskSection
	}
	if got["Voraussetzungen"] != TaskSectionPrereq {
		t.Errorf("Voraussetzungen = %v, want prereq", got["Voraussetzungen"])
	}
	if got["Vorgehensweise"] != TaskSectionSteps {
		t.Errorf("Vorgehensweise = %v, want steps marker", got["Vorgehensweise"])
	}
	if !hasKind(doc, ImplicitSteps) {
		t.Errorf("expected <steps> after the custom marker, got %+v", doc.ImplicitSections)
	}
}

// Defaults still apply for sections the override does not mention.
func TestConfigurableTaskSectionTitlesKeepDefaults(t *testing.T) {
	defer SetImplicitTaskSectionTitles(nil)
	SetImplicitTaskSectionTitles(map[string][]string{"prereq": {"voraussetzungen"}})

	doc := New("file:///t.md", 1, "# Task {.task}\n\nShort.\n\n## Verification\n\nDone.\n")
	for _, h := range doc.Index.Headings() {
		if h.Text == "Verification" && h.TaskSection != TaskSectionResult {
			t.Errorf("Verification = %v, want result", h.TaskSection)
		}
	}
}

// A non-task topic gets no implicit task structure.
func TestNoImplicitSectionsOutsideTask(t *testing.T) {
	doc := New("file:///t.md", 1, "# Topic\n\nText.\n\n1. One\n2. Two\n")
	if len(doc.ImplicitSections) != 0 {
		t.Errorf("expected no implicit sections, got %+v", doc.ImplicitSections)
	}
}

// A .md file that declares the DITA map schema is a map, the way the plug-in
// reads a Markdown DITA map.
func TestMarkdownMapSchemaIsMapKind(t *testing.T) {
	doc := New("file:///project/root.md", 1,
		"---\n$schema: urn:oasis:names:tc:dita:xsd:map.xsd\n---\n\n# Root\n\n- [A](a.md)\n")
	if doc.Kind != Map {
		t.Errorf("Kind = %v, want Map", doc.Kind)
	}
}

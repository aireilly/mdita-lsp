package diagnostic

import "testing"

func TestStepElementOutsideTask(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# My Topic", "", "Short desc.", "",
		"Some paragraph with `code`{.stepresult} annotation.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeStepElementOutsideStep {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeStepElementOutsideStep diagnostic for stepresult in non-task topic")
	}
}

func TestStepElementInsideTask(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:task.xsd", "---", "",
		"# My Task", "", "Short desc.", "",
		"1. Do something", "",
		"   Result: `success`{.stepresult}",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	for _, d := range diags {
		if d.Code == CodeStepElementOutsideStep {
			t.Errorf("unexpected CodeStepElementOutsideStep in task topic: %s", d.Message)
		}
	}
}

func TestStepElementInsideTaskWithTitleClass(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# My Task {.task}", "", "Short desc.", "",
		"1. Do something", "",
		"   Result: `success`{.stepresult}",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	for _, d := range diags {
		if d.Code == CodeStepElementOutsideStep {
			t.Errorf("unexpected CodeStepElementOutsideStep in task topic (title class): %s", d.Message)
		}
	}
}

func TestStepxmpOutsideTask(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# My Topic", "", "Short desc.", "",
		"Example: `docker run`{.stepxmp} shows usage.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeStepElementOutsideStep {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeStepElementOutsideStep diagnostic for stepxmp in non-task topic")
	}
}

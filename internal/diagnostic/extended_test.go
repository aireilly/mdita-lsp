package diagnostic

import "testing"

func TestUnknownOutputclass(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# Topic", "", "Short desc.", "",
		"Click **Save**{.notreal} to save.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeUnknownOutputclass {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeUnknownOutputclass diagnostic")
	}
}

func TestDomainClassWrongParent(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# Topic", "", "Short desc.", "",
		"Edit `Save`{.uicontrol} button.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeDomainClassWrongParent {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeDomainClassWrongParent diagnostic")
	}
}

func TestMenucascadeMissingSeparator(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# Topic", "", "Short desc.", "",
		"Click **File Open**{.menucascade} to open.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeMenucascadeMissingSeparator {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeMenucascadeMissingSeparator diagnostic")
	}
}

func TestValidDomainClass(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# Topic", "", "Short desc.", "",
		"Click **Save**{.uicontrol} to save.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	for _, d := range diags {
		if d.Code == CodeUnknownOutputclass || d.Code == CodeDomainClassWrongParent {
			t.Errorf("unexpected diagnostic: %s — %s", d.Code, d.Message)
		}
	}
}

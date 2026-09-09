package ditamap

import "testing"

func TestParseMapWithReltable(t *testing.T) {
	input := "# Map\n\n- [Overview](overview.md)\n- [Install](install.md)\n\n| [Overview](overview.md) | [Install](install.md) |\n|------------------------|----------------------|\n| [Config](config.md)    | [Troubleshoot](ts.md) |\n"
	m, err := ParseMap(input)
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if len(m.TopicRefs) != 2 {
		t.Fatalf("TopicRefs = %d, want 2", len(m.TopicRefs))
	}
	if len(m.RelTables) != 1 {
		t.Fatalf("RelTables = %d, want 1", len(m.RelTables))
	}
	rt := m.RelTables[0]
	if len(rt.Header) != 2 {
		t.Fatalf("Header = %d, want 2", len(rt.Header))
	}
	if len(rt.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(rt.Rows))
	}
	if len(rt.Rows[0].Cells) != 2 {
		t.Fatalf("Cells = %d, want 2", len(rt.Rows[0].Cells))
	}
}

// The plug-in renders every map list item as <topicref>, including references
// to other maps, which differ only by the @format value derived from the file
// extension.
func TestParseMapSubmapIsTopicref(t *testing.T) {
	input := "# Map\n\n- [Sub-map](submap.ditamap)\n- [MDITA sub](sub.mditamap)\n- [Topic](topic.md)\n"
	m, err := ParseMap(input)
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	want := []string{"submap.ditamap", "sub.mditamap", "topic.md"}
	if len(m.TopicRefs) != len(want) {
		t.Fatalf("TopicRefs = %d, want %d", len(m.TopicRefs), len(want))
	}
	for i, href := range want {
		if m.TopicRefs[i].Href != href {
			t.Errorf("TopicRefs[%d].Href = %q, want %q", i, m.TopicRefs[i].Href, href)
		}
	}
}

func TestParseMapNoReltable(t *testing.T) {
	input := "# Map\n\n- [A](a.md)\n"
	m, _ := ParseMap(input)
	if len(m.RelTables) != 0 {
		t.Errorf("RelTables = %d, want 0", len(m.RelTables))
	}
}

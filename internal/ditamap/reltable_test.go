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

func TestParseMapMapref(t *testing.T) {
	input := "# Map\n\n- [Sub-map](submap.ditamap)\n- [MDITA sub](sub.mditamap)\n- [Topic](topic.md)\n"
	m, err := ParseMap(input)
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if !m.TopicRefs[0].IsMapRef {
		t.Error("expected submap.ditamap to be mapref")
	}
	if !m.TopicRefs[1].IsMapRef {
		t.Error("expected sub.mditamap to be mapref")
	}
	if m.TopicRefs[2].IsMapRef {
		t.Error("expected topic.md to not be mapref")
	}
}

func TestParseMapNoReltable(t *testing.T) {
	input := "# Map\n\n- [A](a.md)\n"
	m, _ := ParseMap(input)
	if len(m.RelTables) != 0 {
		t.Errorf("RelTables = %d, want 0", len(m.RelTables))
	}
}

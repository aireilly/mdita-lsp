package ditamap

import "testing"

func TestParseMap(t *testing.T) {
	input := "# Product Documentation\n\n- [Getting Started](getting-started.md)\n  - [Installation](install.md)\n  - [Configuration](config.md)\n- [User Guide](user-guide.md)\n"
	m, err := ParseMap(input)
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if m.Title != "Product Documentation" {
		t.Errorf("Title = %q, want %q", m.Title, "Product Documentation")
	}
	if len(m.TopicRefs) != 2 {
		t.Fatalf("TopicRefs = %d, want 2", len(m.TopicRefs))
	}
	if m.TopicRefs[0].Title != "Getting Started" {
		t.Errorf("TopicRefs[0].Title = %q", m.TopicRefs[0].Title)
	}
	if m.TopicRefs[0].Href != "getting-started.md" {
		t.Errorf("TopicRefs[0].Href = %q", m.TopicRefs[0].Href)
	}
	if len(m.TopicRefs[0].Children) != 2 {
		t.Fatalf("TopicRefs[0].Children = %d, want 2", len(m.TopicRefs[0].Children))
	}
	if m.TopicRefs[0].Children[0].Title != "Installation" {
		t.Errorf("child[0].Title = %q", m.TopicRefs[0].Children[0].Title)
	}
}

func TestParseMapNoTitle(t *testing.T) {
	input := "- [Topic](topic.md)\n"
	m, err := ParseMap(input)
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if m.Title != "" {
		t.Errorf("Title = %q, want empty", m.Title)
	}
	if len(m.TopicRefs) != 1 {
		t.Errorf("TopicRefs = %d, want 1", len(m.TopicRefs))
	}
}

func TestParseMapEmpty(t *testing.T) {
	m, err := ParseMap("")
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if len(m.TopicRefs) != 0 {
		t.Errorf("TopicRefs = %d, want 0", len(m.TopicRefs))
	}
}

func TestAllHrefs(t *testing.T) {
	input := "# Map\n\n- [A](a.md)\n  - [B](b.md)\n- [C](c.md)\n"
	m, _ := ParseMap(input)
	hrefs := m.AllHrefs()
	if len(hrefs) != 3 {
		t.Errorf("AllHrefs = %d, want 3", len(hrefs))
	}
}

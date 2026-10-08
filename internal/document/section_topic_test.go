package document

import "testing"

func TestIsSectionTopic(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"concept by class", "# C {.concept}\n", true},
		{"reference by class", "# R {.reference}\n", true},
		{
			"concept by schema",
			"---\n$schema: urn:oasis:names:tc:dita:xsd:concept.xsd\n---\n\n# C\n",
			true,
		},
		{
			"reference by schema",
			"---\n$schema: urn:oasis:names:tc:dita:xsd:reference.xsd\n---\n\n# R\n",
			true,
		},
		{"task by class", "# T {.task}\n", false},
		{"generic topic", "# T\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSectionTopic(New("file:///a.md", 1, tt.text)); got != tt.want {
				t.Errorf("IsSectionTopic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveSections(t *testing.T) {
	tests := []struct {
		name string
		text string
		// want maps heading text to whether the plug-in builds it as a section.
		want map[string]bool
	}{
		{
			name: "unclassed H2 in a concept is a section",
			text: "# C {.concept}\n\nShortdesc.\n\n## Background\n\nText.\n\n## Limitations\n\nText.\n",
			want: map[string]bool{"Background": true, "Limitations": true},
		},
		{
			name: "unclassed H2 in a generic topic is a nested topic",
			text: "# T\n\n## A\n\n## B\n",
			want: map[string]bool{"A": false, "B": false},
		},
		{
			name: "unclassed H2 in a task is a nested topic",
			text: "# T {.task}\n\n## Details\n",
			want: map[string]bool{"Details": false},
		},
		{
			name: "a type class keeps the escape hatch, and its H3 is its section",
			text: "# C {.concept}\n\n## Nested {.concept}\n\n### Background\n",
			want: map[string]bool{"Nested": false, "Background": true},
		},
		{
			name: "a section class is still a section in a generic topic",
			text: "# T\n\n## A {.section}\n",
			want: map[string]bool{"A": true},
		},
		{
			// The build rejects it, so the model keeps it a section and the
			// diagnostic reports that sections do not nest.
			name: "H3 under a section in a concept is still a section",
			text: "# C {.concept}\n\n## A\n\n### B\n",
			want: map[string]bool{"A": true, "B": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := New("file:///a.md", 1, tt.text)
			for _, h := range doc.Index.Headings() {
				want, ok := tt.want[h.Text]
				if !ok {
					continue
				}
				if h.Section != want {
					t.Errorf("heading %q: Section = %v, want %v", h.Text, h.Section, want)
				}
			}
		})
	}
}

// A section belongs to the topic that encloses it, so its DITA address carries
// that topic's id.
func TestSectionAddress(t *testing.T) {
	doc := New("file:///c.md", 1, "# C {.concept}\n\n## Background\n")
	h := doc.Index.HeadingByID("background")
	if h == nil {
		t.Fatal("no background heading")
	}
	if got := doc.SectionAddress(h); got != "c/background" {
		t.Errorf("SectionAddress() = %q, want %q", got, "c/background")
	}
}

// The topic title is the topic, never a section of itself, which a $schema
// typed concept has to get right because its title carries no class.
func TestTitleIsNeverASection(t *testing.T) {
	for _, text := range []string{
		"---\n$schema: urn:oasis:names:tc:dita:xsd:concept.xsd\nid: c\n---\n\n# C\n\n## Background\n",
		"# C {.concept}\n\n## Background\n",
	} {
		doc := New("file:///c.md", 1, text)
		title := doc.Index.Title()
		if title == nil {
			t.Fatal("no title")
		}
		if title.Section {
			t.Errorf("title %q marked as a section in:\n%s", title.Text, text)
		}
		h := doc.Index.HeadingByID("background")
		if h == nil || !h.Section {
			t.Fatalf("Background is not a section: %+v", h)
		}
		if got := doc.SectionAddress(h); got != "c/background" {
			t.Errorf("SectionAddress() = %q, want %q", got, "c/background")
		}
	}
}

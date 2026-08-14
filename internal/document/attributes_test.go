package document

import (
	"reflect"
	"testing"
)

func TestParseAttrString(t *testing.T) {
	tests := []struct {
		name string
		attr string
		want ParsedAttribute
	}{
		{
			name: "class only",
			attr: ".uicontrol",
			want: ParsedAttribute{
				Classes:   []string{"uicontrol"},
				ID:        "",
				KeyValues: map[string]string{},
			},
		},
		{
			name: "class and id",
			attr: ".task #install",
			want: ParsedAttribute{
				Classes:   []string{"task"},
				ID:        "install",
				KeyValues: map[string]string{},
			},
		},
		{
			name: "key-value",
			attr: `audience="novice"`,
			want: ParsedAttribute{
				Classes:   nil,
				ID:        "",
				KeyValues: map[string]string{"audience": "novice"},
			},
		},
		{
			name: "class and key-value",
			attr: `.filepath platform="linux"`,
			want: ParsedAttribute{
				Classes:   []string{"filepath"},
				ID:        "",
				KeyValues: map[string]string{"platform": "linux"},
			},
		},
		{
			name: "multiple classes",
			attr: ".task .prereq",
			want: ParsedAttribute{
				Classes:   []string{"task", "prereq"},
				ID:        "",
				KeyValues: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAttrString(tt.attr)
			if !reflect.DeepEqual(got.Classes, tt.want.Classes) {
				t.Errorf("Classes = %v, want %v", got.Classes, tt.want.Classes)
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if !reflect.DeepEqual(got.KeyValues, tt.want.KeyValues) {
				t.Errorf("KeyValues = %v, want %v", got.KeyValues, tt.want.KeyValues)
			}
		})
	}
}

func TestScanBlockAttributes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
		check func([]BlockAttribute) error
	}{
		{
			name:  "single block attr",
			input: "{audience=\"novice\"}\n\nThis is a paragraph.",
			want:  1,
			check: func(attrs []BlockAttribute) error {
				if attrs[0].Attr.KeyValues["audience"] != "novice" {
					t.Errorf("audience = %v, want novice", attrs[0].Attr.KeyValues["audience"])
				}
				if attrs[0].Line != 0 {
					t.Errorf("Line = %d, want 0", attrs[0].Line)
				}
				return nil
			},
		},
		{
			name:  "multiple key-values",
			input: "{audience=\"expert\" platform=\"linux\"}\n\nAdvanced content.",
			want:  1,
			check: func(attrs []BlockAttribute) error {
				if attrs[0].Attr.KeyValues["audience"] != "expert" {
					t.Errorf("audience = %v, want expert", attrs[0].Attr.KeyValues["audience"])
				}
				if attrs[0].Attr.KeyValues["platform"] != "linux" {
					t.Errorf("platform = %v, want linux", attrs[0].Attr.KeyValues["platform"])
				}
				return nil
			},
		},
		{
			name:  "no block attrs",
			input: "This is a paragraph without attributes.",
			want:  0,
			check: nil,
		},
		{
			name:  "class-only block",
			input: "{.note}\n\nThis is a note.",
			want:  1,
			check: func(attrs []BlockAttribute) error {
				if len(attrs[0].Attr.Classes) != 1 || attrs[0].Attr.Classes[0] != "note" {
					t.Errorf("Classes = %v, want [note]", attrs[0].Attr.Classes)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScanBlockAttributes(tt.input)
			if len(got) != tt.want {
				t.Fatalf("got %d attributes, want %d", len(got), tt.want)
			}
			if tt.check != nil {
				_ = tt.check(got)
			}
		})
	}
}

func TestHeadingAttributesParsed(t *testing.T) {
	text := "# Install the software {.task}\n\nShort desc.\n\n## Prerequisites {.prereq}\n\nYou need admin access.\n"
	elements, bf, _ := Parse(text)
	if !bf.HasAttributes {
		t.Error("expected HasAttributes to be true")
	}

	var headings []*Heading
	for _, e := range elements {
		if h, ok := e.(*Heading); ok {
			headings = append(headings, h)
		}
	}

	if len(headings) != 2 {
		t.Fatalf("got %d headings, want 2", len(headings))
	}
	h1 := headings[0]
	if h1.Attributes == nil {
		t.Fatal("expected H1 to have attributes")
	}
	if len(h1.Attributes.Classes) != 1 || h1.Attributes.Classes[0] != "task" {
		t.Errorf("H1 classes = %v, want [task]", h1.Attributes.Classes)
	}
	h2 := headings[1]
	if h2.Attributes == nil {
		t.Fatal("expected H2 to have attributes")
	}
	if len(h2.Attributes.Classes) != 1 || h2.Attributes.Classes[0] != "prereq" {
		t.Errorf("H2 classes = %v, want [prereq]", h2.Attributes.Classes)
	}
}

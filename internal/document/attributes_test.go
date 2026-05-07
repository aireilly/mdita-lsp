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

func TestScanInlineAttributes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // number of attributes
		check func([]InlineAttribute) error
	}{
		{
			name:  "bold with class",
			input: "Click **Save**{.uicontrol} to continue.",
			want:  1,
			check: func(attrs []InlineAttribute) error {
				if attrs[0].TargetKind != "bold" {
					t.Errorf("TargetKind = %v, want bold", attrs[0].TargetKind)
				}
				if attrs[0].TargetText != "Save" {
					t.Errorf("TargetText = %v, want Save", attrs[0].TargetText)
				}
				if len(attrs[0].Attr.Classes) != 1 || attrs[0].Attr.Classes[0] != "uicontrol" {
					t.Errorf("Classes = %v, want [uicontrol]", attrs[0].Attr.Classes)
				}
				return nil
			},
		},
		{
			name:  "code with class",
			input: "Edit the `config.yaml`{.filepath} file.",
			want:  1,
			check: func(attrs []InlineAttribute) error {
				if attrs[0].TargetKind != "code" {
					t.Errorf("TargetKind = %v, want code", attrs[0].TargetKind)
				}
				if attrs[0].TargetText != "config.yaml" {
					t.Errorf("TargetText = %v, want config.yaml", attrs[0].TargetText)
				}
				if len(attrs[0].Attr.Classes) != 1 || attrs[0].Attr.Classes[0] != "filepath" {
					t.Errorf("Classes = %v, want [filepath]", attrs[0].Attr.Classes)
				}
				return nil
			},
		},
		{
			name:  "italic with class",
			input: "Read *War and Peace*{.cite} for context.",
			want:  1,
			check: func(attrs []InlineAttribute) error {
				if attrs[0].TargetKind != "italic" {
					t.Errorf("TargetKind = %v, want italic", attrs[0].TargetKind)
				}
				if attrs[0].TargetText != "War and Peace" {
					t.Errorf("TargetText = %v, want War and Peace", attrs[0].TargetText)
				}
				if len(attrs[0].Attr.Classes) != 1 || attrs[0].Attr.Classes[0] != "cite" {
					t.Errorf("Classes = %v, want [cite]", attrs[0].Attr.Classes)
				}
				return nil
			},
		},
		{
			name:  "underscore bold with class",
			input: "Press __Enter__{.uicontrol} to submit.",
			want:  1,
			check: func(attrs []InlineAttribute) error {
				if attrs[0].TargetKind != "bold" {
					t.Errorf("TargetKind = %v, want bold", attrs[0].TargetKind)
				}
				if attrs[0].TargetText != "Enter" {
					t.Errorf("TargetText = %v, want Enter", attrs[0].TargetText)
				}
				if len(attrs[0].Attr.Classes) != 1 || attrs[0].Attr.Classes[0] != "uicontrol" {
					t.Errorf("Classes = %v, want [uicontrol]", attrs[0].Attr.Classes)
				}
				return nil
			},
		},
		{
			name:  "multiple on one line",
			input: "Click **Save**{.uicontrol} or **Cancel**{.uicontrol}.",
			want:  2,
			check: func(attrs []InlineAttribute) error {
				if attrs[0].TargetText != "Save" {
					t.Errorf("First TargetText = %v, want Save", attrs[0].TargetText)
				}
				if attrs[1].TargetText != "Cancel" {
					t.Errorf("Second TargetText = %v, want Cancel", attrs[1].TargetText)
				}
				return nil
			},
		},
		{
			name:  "no attributes",
			input: "This is **bold** text without attributes.",
			want:  0,
			check: nil,
		},
		{
			name:  "key-value attribute",
			input: `The **command**{audience="expert"} is advanced.`,
			want:  1,
			check: func(attrs []InlineAttribute) error {
				if attrs[0].Attr.KeyValues["audience"] != "expert" {
					t.Errorf("audience = %v, want expert", attrs[0].Attr.KeyValues["audience"])
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScanInlineAttributes(tt.input)
			if len(got) != tt.want {
				t.Fatalf("got %d attributes, want %d", len(got), tt.want)
			}
			if tt.check != nil {
				_ = tt.check(got)
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

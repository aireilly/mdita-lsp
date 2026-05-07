package vocabulary

import (
	"strings"
	"testing"
)

func TestLookupDomainElement(t *testing.T) {
	tests := []struct {
		class       string
		wantFound   bool
		wantElement string
		wantDomain  string
		wantParent  string
	}{
		{"uicontrol", true, "uicontrol", "ui-d", "bold"},
		{"wintitle", true, "wintitle", "ui-d", "bold"},
		{"menucascade", true, "menucascade", "ui-d", "bold"},
		{"shortcut", true, "shortcut", "ui-d", "bold"},
		{"filepath", true, "filepath", "sw-d", "code"},
		{"cmdname", true, "cmdname", "sw-d", "code"},
		{"userinput", true, "userinput", "sw-d", "code"},
		{"systemoutput", true, "systemoutput", "sw-d", "code"},
		{"varname", true, "varname", "sw-d", "code"},
		{"msgph", true, "msgph", "sw-d", "code"},
		{"codeph", true, "codeph", "pr-d", "code"},
		{"option", true, "option", "pr-d", "code"},
		{"parmname", true, "parmname", "pr-d", "code"},
		{"apiname", true, "apiname", "pr-d", "code"},
		{"kwd", true, "kwd", "pr-d", "code"},
		{"cite", true, "cite", "topic", "italic"},
		{"draft-comment", true, "draft-comment", "topic", "paragraph"},
		{"notreal", false, "", "", ""},
		{"invalid", false, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.class, func(t *testing.T) {
			elem, found := LookupDomainElement(tt.class)
			if found != tt.wantFound {
				t.Errorf("LookupDomainElement(%q) found = %v, want %v", tt.class, found, tt.wantFound)
			}
			if found {
				if elem.DITAElement != tt.wantElement {
					t.Errorf("DITAElement = %q, want %q", elem.DITAElement, tt.wantElement)
				}
				if elem.Domain != tt.wantDomain {
					t.Errorf("Domain = %q, want %q", elem.Domain, tt.wantDomain)
				}
				if elem.ParentKind != tt.wantParent {
					t.Errorf("ParentKind = %q, want %q", elem.ParentKind, tt.wantParent)
				}
			}
		})
	}
}

func TestLookupTaskSection(t *testing.T) {
	tests := []struct {
		title       string
		wantFound   bool
		wantClass   string
		wantElement string
		wantOrder   int
	}{
		{"Prerequisites", true, "prereq", "prereq", 1},
		{"About this task", true, "context", "context", 2},
		{"Verification", true, "result", "result", 4},
		{"Next steps", true, "postreq", "postreq", 5},
		{"prerequisites", true, "prereq", "prereq", 1},
		{"ABOUT THIS TASK", true, "context", "context", 2},
		{"Unknown section", false, "", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			section, found := LookupTaskSection(tt.title)
			if found != tt.wantFound {
				t.Errorf("LookupTaskSection(%q) found = %v, want %v", tt.title, found, tt.wantFound)
			}
			if found {
				if section.Class != tt.wantClass {
					t.Errorf("Class = %q, want %q", section.Class, tt.wantClass)
				}
				if section.DITAElement != tt.wantElement {
					t.Errorf("DITAElement = %q, want %q", section.DITAElement, tt.wantElement)
				}
				if section.Order != tt.wantOrder {
					t.Errorf("Order = %d, want %d", section.Order, tt.wantOrder)
				}
			}
		})
	}
}

func TestLookupTaskSectionByClass(t *testing.T) {
	tests := []struct {
		class       string
		wantFound   bool
		wantTitle   string
		wantElement string
	}{
		{"prereq", true, "Prerequisites", "prereq"},
		{"context", true, "About this task", "context"},
		{"result", true, "Verification", "result"},
		{"postreq", true, "Next steps", "postreq"},
		{"tasktroubleshooting", true, "", "tasktroubleshooting"},
		{"unknown", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.class, func(t *testing.T) {
			section, found := LookupTaskSectionByClass(tt.class)
			if found != tt.wantFound {
				t.Errorf("LookupTaskSectionByClass(%q) found = %v, want %v", tt.class, found, tt.wantFound)
			}
			if found {
				if section.DefaultTitle != tt.wantTitle {
					t.Errorf("DefaultTitle = %q, want %q", section.DefaultTitle, tt.wantTitle)
				}
				if section.DITAElement != tt.wantElement {
					t.Errorf("DITAElement = %q, want %q", section.DITAElement, tt.wantElement)
				}
			}
		})
	}
}

func TestLookupStepElement(t *testing.T) {
	tests := []struct {
		class       string
		wantFound   bool
		wantElement string
	}{
		{"stepresult", true, "stepresult"},
		{"stepxmp", true, "stepxmp"},
		{"unknown", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.class, func(t *testing.T) {
			elem, found := LookupStepElement(tt.class)
			if found != tt.wantFound {
				t.Errorf("LookupStepElement(%q) found = %v, want %v", tt.class, found, tt.wantFound)
			}
			if found {
				if elem.DITAElement != tt.wantElement {
					t.Errorf("DITAElement = %q, want %q", elem.DITAElement, tt.wantElement)
				}
			}
		})
	}
}

func TestIsConditionalAttribute(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"audience", true},
		{"platform", true},
		{"product", true},
		{"otherprops", true},
		{"deliveryTarget", true},
		{"props", true},
		{"rev", true},
		{"unknown", false},
		{"class", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConditionalAttribute(tt.name)
			if got != tt.want {
				t.Errorf("IsConditionalAttribute(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestAllDomainElements(t *testing.T) {
	elements := AllDomainElements()
	if len(elements) != 17 {
		t.Errorf("AllDomainElements() returned %d, want 17", len(elements))
	}
}

func TestAllTaskSections(t *testing.T) {
	sections := AllTaskSections()
	if len(sections) != 5 {
		t.Errorf("AllTaskSections() returned %d, want 5", len(sections))
	}
}

func TestAllConditionalAttributes(t *testing.T) {
	attrs := AllConditionalAttributes()
	if len(attrs) != 7 {
		t.Errorf("AllConditionalAttributes() returned %d, want 7", len(attrs))
	}
}

func TestDomainElementsByParentKind(t *testing.T) {
	bold := DomainElementsByParentKind("bold")
	if len(bold) != 4 {
		t.Errorf("bold elements = %d, want 4", len(bold))
	}
	code := DomainElementsByParentKind("code")
	if len(code) != 11 {
		t.Errorf("code elements = %d, want 11", len(code))
	}
	italic := DomainElementsByParentKind("italic")
	if len(italic) != 1 {
		t.Errorf("italic elements = %d, want 1", len(italic))
	}
}

func TestDescriptionsPopulated(t *testing.T) {
	for _, elem := range AllDomainElements() {
		if strings.TrimSpace(elem.Description) == "" {
			t.Errorf("Domain element %q missing description", elem.DITAElement)
		}
	}
	for _, section := range AllTaskSections() {
		if strings.TrimSpace(section.Description) == "" {
			t.Errorf("Task section %q missing description", section.DITAElement)
		}
	}
	elem, found := LookupStepElement("stepresult")
	if !found {
		t.Error("stepresult not found")
	} else if strings.TrimSpace(elem.Description) == "" {
		t.Error("stepresult missing description")
	}
}

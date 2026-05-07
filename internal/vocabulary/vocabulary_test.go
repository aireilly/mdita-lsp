package vocabulary

import (
	"strings"
	"testing"
)

// Test domain element lookups
func TestLookupDomainElement(t *testing.T) {
	tests := []struct {
		class       string
		wantFound   bool
		wantElement string
		wantDomain  string
		wantParent  string
	}{
		// UI domain (bold)
		{"+ topic/ph ui-d/uicontrol", true, "uicontrol", "ui-d", "bold"},
		{"+ topic/keyword ui-d/wintitle", true, "wintitle", "ui-d", "bold"},
		{"+ topic/ph ui-d/menucascade", true, "menucascade", "ui-d", "bold"},
		{"+ topic/keyword ui-d/shortcut", true, "shortcut", "ui-d", "bold"},

		// Software domain (code)
		{"+ topic/ph sw-d/filepath", true, "filepath", "sw-d", "code"},
		{"+ topic/keyword sw-d/cmdname", true, "cmdname", "sw-d", "code"},
		{"+ topic/ph sw-d/userinput", true, "userinput", "sw-d", "code"},
		{"+ topic/ph sw-d/systemoutput", true, "systemoutput", "sw-d", "code"},
		{"+ topic/keyword sw-d/varname", true, "varname", "sw-d", "code"},
		{"+ topic/keyword sw-d/msgph", true, "msgph", "sw-d", "code"},

		// Programming domain (code)
		{"+ topic/ph pr-d/codeph", true, "codeph", "pr-d", "code"},
		{"+ topic/keyword pr-d/option", true, "option", "pr-d", "code"},
		{"+ topic/keyword pr-d/parmname", true, "parmname", "pr-d", "code"},
		{"+ topic/keyword pr-d/apiname", true, "apiname", "pr-d", "code"},
		{"+ topic/keyword pr-d/kwd", true, "kwd", "pr-d", "code"},

		// Cross-domain elements
		{"+ topic/cite", true, "cite", "topic", "italic"},
		{"+ topic/draft-comment", true, "draft-comment", "topic", "paragraph"},

		// Not found
		{"+ topic/unknown", false, "", "", ""},
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
				if elem.Class != tt.class {
					t.Errorf("Class = %q, want %q", elem.Class, tt.class)
				}
			}
		})
	}
}

// Test task section lookups by title
func TestLookupTaskSection(t *testing.T) {
	tests := []struct {
		title       string
		wantFound   bool
		wantClass   string
		wantElement string
		wantOrder   int
	}{
		// Exact matches
		{"Prerequisites", true, "+ topic/section task/prereq", "prereq", 1},
		{"About this task", true, "+ topic/section task/context", "context", 2},
		{"Verification", true, "+ topic/section task/result", "result", 4},
		{"Next steps", true, "+ topic/section task/postreq", "postreq", 5},
		{"", true, "+ topic/section task/tasktroubleshooting", "tasktroubleshooting", 6},

		// Case-insensitive
		{"prerequisites", true, "+ topic/section task/prereq", "prereq", 1},
		{"ABOUT THIS TASK", true, "+ topic/section task/context", "context", 2},
		{"verification", true, "+ topic/section task/result", "result", 4},
		{"NEXT STEPS", true, "+ topic/section task/postreq", "postreq", 5},

		// Not found
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

// Test task section lookup by class
func TestLookupTaskSectionByClass(t *testing.T) {
	tests := []struct {
		class       string
		wantFound   bool
		wantTitle   string
		wantElement string
	}{
		{"+ topic/section task/prereq", true, "Prerequisites", "prereq"},
		{"+ topic/section task/context", true, "About this task", "context"},
		{"+ topic/section task/result", true, "Verification", "result"},
		{"+ topic/section task/postreq", true, "Next steps", "postreq"},
		{"+ topic/section task/tasktroubleshooting", true, "", "tasktroubleshooting"},
		{"+ topic/section task/unknown", false, "", ""},
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

// Test step element lookups
func TestLookupStepElement(t *testing.T) {
	tests := []struct {
		class       string
		wantFound   bool
		wantElement string
	}{
		{"+ topic/itemgroup task/stepresult", true, "stepresult"},
		{"+ topic/itemgroup task/stepxmp", true, "stepxmp"},
		{"+ topic/itemgroup task/unknown", false, ""},
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

// Test conditional attribute checks
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
		{"id", false},
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

// Test AllDomainElements returns all 17 elements
func TestAllDomainElements(t *testing.T) {
	elements := AllDomainElements()
	if len(elements) != 17 {
		t.Errorf("AllDomainElements() returned %d elements, want 17", len(elements))
	}

	// Verify we have all expected elements
	expectedElements := []string{
		"uicontrol", "wintitle", "menucascade", "shortcut", // UI domain (4)
		"filepath", "cmdname", "userinput", "systemoutput", "varname", "msgph", // Software domain (6)
		"codeph", "option", "parmname", "apiname", "kwd", // Programming domain (5)
		"cite", "draft-comment", // Cross-domain (2)
	}

	elementMap := make(map[string]bool)
	for _, elem := range elements {
		elementMap[elem.DITAElement] = true
	}

	for _, expected := range expectedElements {
		if !elementMap[expected] {
			t.Errorf("AllDomainElements() missing element %q", expected)
		}
	}

	// Verify defensive copy - modifying result should not affect internal state
	original := AllDomainElements()
	modified := AllDomainElements()
	modified[0].Description = "MODIFIED"
	next := AllDomainElements()
	if next[0].Description == "MODIFIED" {
		t.Error("AllDomainElements() did not return defensive copy")
	}
	if original[0].Description == "MODIFIED" {
		t.Error("AllDomainElements() shares internal state")
	}
}

// Test AllTaskSections returns all 5 sections
func TestAllTaskSections(t *testing.T) {
	sections := AllTaskSections()
	if len(sections) != 5 {
		t.Errorf("AllTaskSections() returned %d sections, want 5", len(sections))
	}

	// Verify we have all expected sections
	expectedSections := []string{"prereq", "context", "result", "postreq", "tasktroubleshooting"}
	sectionMap := make(map[string]bool)
	for _, section := range sections {
		sectionMap[section.DITAElement] = true
	}

	for _, expected := range expectedSections {
		if !sectionMap[expected] {
			t.Errorf("AllTaskSections() missing section %q", expected)
		}
	}

	// Verify defensive copy
	original := AllTaskSections()
	modified := AllTaskSections()
	modified[0].Description = "MODIFIED"
	next := AllTaskSections()
	if next[0].Description == "MODIFIED" {
		t.Error("AllTaskSections() did not return defensive copy")
	}
	if original[0].Description == "MODIFIED" {
		t.Error("AllTaskSections() shares internal state")
	}
}

// Test AllConditionalAttributes returns all 7 attributes
func TestAllConditionalAttributes(t *testing.T) {
	attrs := AllConditionalAttributes()
	if len(attrs) != 7 {
		t.Errorf("AllConditionalAttributes() returned %d attributes, want 7", len(attrs))
	}

	// Verify we have all expected attributes
	expectedAttrs := []string{"audience", "platform", "product", "otherprops", "deliveryTarget", "props", "rev"}
	attrMap := make(map[string]bool)
	for _, attr := range attrs {
		attrMap[attr.Name] = true
	}

	for _, expected := range expectedAttrs {
		if !attrMap[expected] {
			t.Errorf("AllConditionalAttributes() missing attribute %q", expected)
		}
	}

	// Verify defensive copy
	original := AllConditionalAttributes()
	modified := AllConditionalAttributes()
	modified[0].Description = "MODIFIED"
	next := AllConditionalAttributes()
	if next[0].Description == "MODIFIED" {
		t.Error("AllConditionalAttributes() did not return defensive copy")
	}
	if original[0].Description == "MODIFIED" {
		t.Error("AllConditionalAttributes() shares internal state")
	}
}

// Test DomainElementsByParentKind filters correctly
func TestDomainElementsByParentKind(t *testing.T) {
	tests := []struct {
		parentKind   string
		wantCount    int
		wantElements []string
	}{
		{
			"bold",
			4,
			[]string{"uicontrol", "wintitle", "menucascade", "shortcut"},
		},
		{
			"code",
			11,
			[]string{"filepath", "cmdname", "userinput", "systemoutput", "varname", "msgph",
				"codeph", "option", "parmname", "apiname", "kwd"},
		},
		{
			"italic",
			1,
			[]string{"cite"},
		},
		{
			"paragraph",
			1,
			[]string{"draft-comment"},
		},
		{
			"unknown",
			0,
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.parentKind, func(t *testing.T) {
			elements := DomainElementsByParentKind(tt.parentKind)
			if len(elements) != tt.wantCount {
				t.Errorf("DomainElementsByParentKind(%q) returned %d elements, want %d",
					tt.parentKind, len(elements), tt.wantCount)
			}

			elementMap := make(map[string]bool)
			for _, elem := range elements {
				elementMap[elem.DITAElement] = true
				if elem.ParentKind != tt.parentKind {
					t.Errorf("Element %q has ParentKind %q, want %q",
						elem.DITAElement, elem.ParentKind, tt.parentKind)
				}
			}

			for _, expected := range tt.wantElements {
				if !elementMap[expected] {
					t.Errorf("DomainElementsByParentKind(%q) missing element %q",
						tt.parentKind, expected)
				}
			}
		})
	}
}

// Test that Description fields are populated
func TestDescriptionsPopulated(t *testing.T) {
	// Domain elements should have descriptions
	for _, elem := range AllDomainElements() {
		if strings.TrimSpace(elem.Description) == "" {
			t.Errorf("Domain element %q missing description", elem.DITAElement)
		}
	}

	// Task sections should have descriptions
	for _, section := range AllTaskSections() {
		if strings.TrimSpace(section.Description) == "" {
			t.Errorf("Task section %q missing description", section.DITAElement)
		}
	}

	// Step elements should have descriptions
	stepElements := []string{"+ topic/itemgroup task/stepresult", "+ topic/itemgroup task/stepxmp"}
	for _, class := range stepElements {
		elem, found := LookupStepElement(class)
		if !found {
			t.Errorf("Step element %q not found", class)
			continue
		}
		if strings.TrimSpace(elem.Description) == "" {
			t.Errorf("Step element %q missing description", elem.DITAElement)
		}
	}

	// Conditional attributes should have descriptions
	for _, attr := range AllConditionalAttributes() {
		if strings.TrimSpace(attr.Description) == "" {
			t.Errorf("Conditional attribute %q missing description", attr.Name)
		}
	}
}

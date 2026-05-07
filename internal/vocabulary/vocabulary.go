package vocabulary

import "strings"

// DomainElement represents a DITA domain specialization element.
type DomainElement struct {
	Class       string
	DITAElement string
	Domain      string
	ParentKind  string
	Description string
}

// TaskSection represents a DITA task section element.
type TaskSection struct {
	DefaultTitle string
	Class        string
	DITAElement  string
	Description  string
	Order        int
}

// StepElement represents a DITA task step child element.
type StepElement struct {
	Class       string
	DITAElement string
	Description string
}

// ConditionalAttribute represents a DITA conditional processing attribute.
type ConditionalAttribute struct {
	Name        string
	Description string
}

var (
	domainElementsByClass   map[string]DomainElement
	taskSectionsByTitle     map[string]TaskSection
	taskSectionsByClass     map[string]TaskSection
	stepElementsByClass     map[string]StepElement
	conditionalAttributeSet map[string]bool

	allDomainElements        []DomainElement
	allTaskSections          []TaskSection
	allConditionalAttributes []ConditionalAttribute
)

func init() {
	// Initialize domain elements
	domainElements := []DomainElement{
		// UI domain (4) - bold parent
		{
			Class:       "+ topic/ph ui-d/uicontrol",
			DITAElement: "uicontrol",
			Domain:      "ui-d",
			ParentKind:  "bold",
			Description: "A user interface control such as a button name, menu item, or dialog label.",
		},
		{
			Class:       "+ topic/keyword ui-d/wintitle",
			DITAElement: "wintitle",
			Domain:      "ui-d",
			ParentKind:  "bold",
			Description: "The title text that appears in a window or dialog box title bar.",
		},
		{
			Class:       "+ topic/ph ui-d/menucascade",
			DITAElement: "menucascade",
			Domain:      "ui-d",
			ParentKind:  "bold",
			Description: "A sequence of menu choices, typically separated by > or arrows.",
		},
		{
			Class:       "+ topic/keyword ui-d/shortcut",
			DITAElement: "shortcut",
			Domain:      "ui-d",
			ParentKind:  "bold",
			Description: "A keyboard shortcut or accelerator key combination.",
		},

		// Software domain (6) - code parent
		{
			Class:       "+ topic/ph sw-d/filepath",
			DITAElement: "filepath",
			Domain:      "sw-d",
			ParentKind:  "code",
			Description: "A file path, directory path, or file name.",
		},
		{
			Class:       "+ topic/keyword sw-d/cmdname",
			DITAElement: "cmdname",
			Domain:      "sw-d",
			ParentKind:  "code",
			Description: "The name of a command or executable program.",
		},
		{
			Class:       "+ topic/ph sw-d/userinput",
			DITAElement: "userinput",
			Domain:      "sw-d",
			ParentKind:  "code",
			Description: "Text or commands that a user enters into a computer system.",
		},
		{
			Class:       "+ topic/ph sw-d/systemoutput",
			DITAElement: "systemoutput",
			Domain:      "sw-d",
			ParentKind:  "code",
			Description: "Output produced by a computer system, such as console messages or log entries.",
		},
		{
			Class:       "+ topic/keyword sw-d/varname",
			DITAElement: "varname",
			Domain:      "sw-d",
			ParentKind:  "code",
			Description: "The name of a variable in programming or configuration contexts.",
		},
		{
			Class:       "+ topic/keyword sw-d/msgph",
			DITAElement: "msgph",
			Domain:      "sw-d",
			ParentKind:  "code",
			Description: "A message phrase, typically a system message or error text.",
		},

		// Programming domain (5) - code parent
		{
			Class:       "+ topic/ph pr-d/codeph",
			DITAElement: "codeph",
			Domain:      "pr-d",
			ParentKind:  "code",
			Description: "A snippet of code within a sentence or paragraph.",
		},
		{
			Class:       "+ topic/keyword pr-d/option",
			DITAElement: "option",
			Domain:      "pr-d",
			ParentKind:  "code",
			Description: "A command-line option or parameter flag.",
		},
		{
			Class:       "+ topic/keyword pr-d/parmname",
			DITAElement: "parmname",
			Domain:      "pr-d",
			ParentKind:  "code",
			Description: "The name of a function or method parameter.",
		},
		{
			Class:       "+ topic/keyword pr-d/apiname",
			DITAElement: "apiname",
			Domain:      "pr-d",
			ParentKind:  "code",
			Description: "The name of an API, function, method, or class.",
		},
		{
			Class:       "+ topic/keyword pr-d/kwd",
			DITAElement: "kwd",
			Domain:      "pr-d",
			ParentKind:  "code",
			Description: "A programming language keyword or reserved word.",
		},

		// Cross-domain (2)
		{
			Class:       "+ topic/cite",
			DITAElement: "cite",
			Domain:      "topic",
			ParentKind:  "italic",
			Description: "A citation to a book, article, or other published work.",
		},
		{
			Class:       "+ topic/draft-comment",
			DITAElement: "draft-comment",
			Domain:      "topic",
			ParentKind:  "paragraph",
			Description: "A comment or note for reviewers that should be removed before publication.",
		},
	}

	// Initialize task sections
	taskSections := []TaskSection{
		{
			DefaultTitle: "Prerequisites",
			Class:        "+ topic/section task/prereq",
			DITAElement:  "prereq",
			Description:  "Prerequisites or conditions that must be met before starting the task.",
			Order:        1,
		},
		{
			DefaultTitle: "About this task",
			Class:        "+ topic/section task/context",
			DITAElement:  "context",
			Description:  "Background information or context for the task.",
			Order:        2,
		},
		{
			DefaultTitle: "Verification",
			Class:        "+ topic/section task/result",
			DITAElement:  "result",
			Description:  "The expected outcome or result of completing the task.",
			Order:        4,
		},
		{
			DefaultTitle: "Next steps",
			Class:        "+ topic/section task/postreq",
			DITAElement:  "postreq",
			Description:  "Optional steps to perform after completing the task.",
			Order:        5,
		},
		{
			DefaultTitle: "",
			Class:        "+ topic/section task/tasktroubleshooting",
			DITAElement:  "tasktroubleshooting",
			Description:  "Troubleshooting information for common problems encountered during the task.",
			Order:        6,
		},
	}

	// Initialize step elements
	stepElements := []StepElement{
		{
			Class:       "+ topic/itemgroup task/stepresult",
			DITAElement: "stepresult",
			Description: "The result or outcome of performing a step.",
		},
		{
			Class:       "+ topic/itemgroup task/stepxmp",
			DITAElement: "stepxmp",
			Description: "An example demonstrating how to perform a step.",
		},
	}

	// Initialize conditional attributes
	conditionalAttributes := []ConditionalAttribute{
		{
			Name:        "audience",
			Description: "The intended audience for the content (e.g., administrator, developer, end-user).",
		},
		{
			Name:        "platform",
			Description: "The platform or operating system to which the content applies (e.g., linux, windows, macos).",
		},
		{
			Name:        "product",
			Description: "The product or product version to which the content applies.",
		},
		{
			Name:        "otherprops",
			Description: "A generic conditional attribute for custom filtering criteria.",
		},
		{
			Name:        "deliveryTarget",
			Description: "The delivery format or channel for the content (e.g., pdf, html, mobile).",
		},
		{
			Name:        "props",
			Description: "A generic property attribute for conditional processing.",
		},
		{
			Name:        "rev",
			Description: "The revision or version identifier for content tracking.",
		},
	}

	// Build lookup maps
	domainElementsByClass = make(map[string]DomainElement, len(domainElements))
	for _, elem := range domainElements {
		domainElementsByClass[elem.Class] = elem
	}

	taskSectionsByTitle = make(map[string]TaskSection, len(taskSections))
	taskSectionsByClass = make(map[string]TaskSection, len(taskSections))
	for _, section := range taskSections {
		// Case-insensitive title lookup
		taskSectionsByTitle[strings.ToLower(section.DefaultTitle)] = section
		taskSectionsByClass[section.Class] = section
	}

	stepElementsByClass = make(map[string]StepElement, len(stepElements))
	for _, elem := range stepElements {
		stepElementsByClass[elem.Class] = elem
	}

	conditionalAttributeSet = make(map[string]bool, len(conditionalAttributes))
	for _, attr := range conditionalAttributes {
		conditionalAttributeSet[attr.Name] = true
	}

	// Store slices for All* functions
	allDomainElements = domainElements
	allTaskSections = taskSections
	allConditionalAttributes = conditionalAttributes
}

// LookupDomainElement returns the domain element with the given class attribute value.
// The class parameter must be an exact match.
func LookupDomainElement(class string) (DomainElement, bool) {
	elem, found := domainElementsByClass[class]
	return elem, found
}

// LookupTaskSection returns the task section with the given title.
// The lookup is case-insensitive.
func LookupTaskSection(title string) (TaskSection, bool) {
	section, found := taskSectionsByTitle[strings.ToLower(title)]
	return section, found
}

// LookupTaskSectionByClass returns the task section with the given class attribute value.
func LookupTaskSectionByClass(class string) (TaskSection, bool) {
	section, found := taskSectionsByClass[class]
	return section, found
}

// LookupStepElement returns the step element with the given class attribute value.
func LookupStepElement(class string) (StepElement, bool) {
	elem, found := stepElementsByClass[class]
	return elem, found
}

// IsConditionalAttribute returns true if the given attribute name is a DITA conditional processing attribute.
func IsConditionalAttribute(name string) bool {
	return conditionalAttributeSet[name]
}

// AllDomainElements returns a defensive copy of all domain elements.
func AllDomainElements() []DomainElement {
	result := make([]DomainElement, len(allDomainElements))
	copy(result, allDomainElements)
	return result
}

// AllTaskSections returns a defensive copy of all task sections.
func AllTaskSections() []TaskSection {
	result := make([]TaskSection, len(allTaskSections))
	copy(result, allTaskSections)
	return result
}

// AllConditionalAttributes returns a defensive copy of all conditional attributes.
func AllConditionalAttributes() []ConditionalAttribute {
	result := make([]ConditionalAttribute, len(allConditionalAttributes))
	copy(result, allConditionalAttributes)
	return result
}

// DomainElementsByParentKind returns all domain elements with the given parent kind.
// Returns a new slice containing matching elements.
func DomainElementsByParentKind(kind string) []DomainElement {
	var result []DomainElement
	for _, elem := range allDomainElements {
		if elem.ParentKind == kind {
			result = append(result, elem)
		}
	}
	return result
}

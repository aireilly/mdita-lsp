# MDITA Extended Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full LSP authoring support for MDITA extended profile features from `redhat.mdita.extended`: domain specializations, conditional processing, task sections, related links, reltable, mapref.

**Architecture:** Central vocabulary registry (`internal/vocabulary/`) defines all extended MDITA elements as Go data structures. A regex-based attribute scanner parses `{.class key="value"}` syntax at heading, block, and inline levels. All LSP features (diagnostics, completion, hover, inlay hints, semantic tokens, code actions) query the vocabulary registry. No new dependencies — goldmark's built-in `parser.WithAttribute()` handles heading attributes; a custom scanner handles inline and block attributes.

**Tech Stack:** Go, goldmark (existing), goldmark `parser.WithAttribute()` (built-in, currently disabled)

**Design spec:** `docs/superpowers/specs/2026-05-07-mdita-extended-parity-design.md`

---

### Task 1: Vocabulary Registry

**Files:**
- Create: `internal/vocabulary/vocabulary.go`
- Create: `internal/vocabulary/vocabulary_test.go`

- [ ] **Step 1: Write failing tests for vocabulary lookups**

```go
// internal/vocabulary/vocabulary_test.go
package vocabulary

import (
	"testing"
)

func TestLookupDomainElement(t *testing.T) {
	elem, ok := LookupDomainElement("uicontrol")
	if !ok {
		t.Fatal("expected to find uicontrol")
	}
	if elem.DITAElement != "uicontrol" {
		t.Errorf("DITAElement = %q, want %q", elem.DITAElement, "uicontrol")
	}
	if elem.Domain != "ui-d" {
		t.Errorf("Domain = %q, want %q", elem.Domain, "ui-d")
	}
	if elem.ParentKind != "bold" {
		t.Errorf("ParentKind = %q, want %q", elem.ParentKind, "bold")
	}
}

func TestLookupDomainElementUnknown(t *testing.T) {
	_, ok := LookupDomainElement("notreal")
	if ok {
		t.Error("expected notreal to not be found")
	}
}

func TestLookupDomainElementAllEntries(t *testing.T) {
	cases := []struct {
		class      string
		dita       string
		domain     string
		parentKind string
	}{
		{"uicontrol", "uicontrol", "ui-d", "bold"},
		{"wintitle", "wintitle", "ui-d", "bold"},
		{"menucascade", "menucascade", "ui-d", "bold"},
		{"shortcut", "shortcut", "ui-d", "bold"},
		{"filepath", "filepath", "sw-d", "code"},
		{"cmdname", "cmdname", "sw-d", "code"},
		{"userinput", "userinput", "sw-d", "code"},
		{"systemoutput", "systemoutput", "sw-d", "code"},
		{"varname", "varname", "sw-d", "code"},
		{"msgph", "msgph", "sw-d", "code"},
		{"codeph", "codeph", "pr-d", "code"},
		{"option", "option", "pr-d", "code"},
		{"parmname", "parmname", "pr-d", "code"},
		{"apiname", "apiname", "pr-d", "code"},
		{"kwd", "kwd", "pr-d", "code"},
		{"cite", "cite", "topic", "italic"},
		{"draft-comment", "draft-comment", "topic", "paragraph"},
	}
	for _, tt := range cases {
		t.Run(tt.class, func(t *testing.T) {
			elem, ok := LookupDomainElement(tt.class)
			if !ok {
				t.Fatalf("not found: %s", tt.class)
			}
			if elem.DITAElement != tt.dita {
				t.Errorf("DITAElement = %q, want %q", elem.DITAElement, tt.dita)
			}
			if elem.Domain != tt.domain {
				t.Errorf("Domain = %q, want %q", elem.Domain, tt.domain)
			}
			if elem.ParentKind != tt.parentKind {
				t.Errorf("ParentKind = %q, want %q", elem.ParentKind, tt.parentKind)
			}
		})
	}
}

func TestLookupTaskSection(t *testing.T) {
	sec, ok := LookupTaskSection("Prerequisites")
	if !ok {
		t.Fatal("expected to find Prerequisites")
	}
	if sec.Class != "prereq" {
		t.Errorf("Class = %q, want %q", sec.Class, "prereq")
	}
	if sec.Order != 1 {
		t.Errorf("Order = %d, want 1", sec.Order)
	}
}

func TestLookupTaskSectionCaseInsensitive(t *testing.T) {
	_, ok := LookupTaskSection("prerequisites")
	if !ok {
		t.Fatal("expected case-insensitive match")
	}
}

func TestLookupTaskSectionByClass(t *testing.T) {
	sec, ok := LookupTaskSectionByClass("postreq")
	if !ok {
		t.Fatal("expected to find postreq")
	}
	if sec.DefaultTitle != "Next steps" {
		t.Errorf("DefaultTitle = %q, want %q", sec.DefaultTitle, "Next steps")
	}
}

func TestLookupStepElement(t *testing.T) {
	elem, ok := LookupStepElement("stepresult")
	if !ok {
		t.Fatal("expected to find stepresult")
	}
	if elem.DITAElement != "stepresult" {
		t.Errorf("DITAElement = %q, want %q", elem.DITAElement, "stepresult")
	}
}

func TestIsConditionalAttribute(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"platform", true},
		{"audience", true},
		{"product", true},
		{"otherprops", true},
		{"deliveryTarget", true},
		{"props", true},
		{"rev", true},
		{"foo", false},
		{"class", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConditionalAttribute(tt.name)
			if got != tt.want {
				t.Errorf("IsConditionalAttribute(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestAllDomainElements(t *testing.T) {
	all := AllDomainElements()
	if len(all) != 17 {
		t.Errorf("AllDomainElements() returned %d, want 17", len(all))
	}
}

func TestAllTaskSections(t *testing.T) {
	all := AllTaskSections()
	if len(all) != 5 {
		t.Errorf("AllTaskSections() returned %d, want 5", len(all))
	}
}

func TestAllConditionalAttributes(t *testing.T) {
	all := AllConditionalAttributes()
	if len(all) != 7 {
		t.Errorf("AllConditionalAttributes() returned %d, want 7", len(all))
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/vocabulary/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Implement the vocabulary registry**

```go
// internal/vocabulary/vocabulary.go
package vocabulary

import "strings"

type DomainElement struct {
	Class       string
	DITAElement string
	Domain      string
	ParentKind  string
	Description string
}

type TaskSection struct {
	DefaultTitle string
	Class        string
	DITAElement  string
	Description  string
	Order        int
}

type StepElement struct {
	Class       string
	DITAElement string
	Description string
}

type ConditionalAttribute struct {
	Name        string
	Description string
}

var domainElements = []DomainElement{
	// UI domain
	{"uicontrol", "uicontrol", "ui-d", "bold", "User interface control label"},
	{"wintitle", "wintitle", "ui-d", "bold", "Window or dialog title"},
	{"menucascade", "menucascade", "ui-d", "bold", "Menu path (items separated by ' > ')"},
	{"shortcut", "shortcut", "ui-d", "bold", "Keyboard shortcut"},
	// Software domain
	{"filepath", "filepath", "sw-d", "code", "File path or directory"},
	{"cmdname", "cmdname", "sw-d", "code", "Command name"},
	{"userinput", "userinput", "sw-d", "code", "User-entered text"},
	{"systemoutput", "systemoutput", "sw-d", "code", "System output text"},
	{"varname", "varname", "sw-d", "code", "Variable name"},
	{"msgph", "msgph", "sw-d", "code", "Message phrase"},
	// Programming domain
	{"codeph", "codeph", "pr-d", "code", "Code phrase"},
	{"option", "option", "pr-d", "code", "Command option"},
	{"parmname", "parmname", "pr-d", "code", "Parameter name"},
	{"apiname", "apiname", "pr-d", "code", "API name"},
	{"kwd", "kwd", "pr-d", "code", "Keyword"},
	// Cross-domain
	{"cite", "cite", "topic", "italic", "Citation or title of a work"},
	{"draft-comment", "draft-comment", "topic", "paragraph", "Draft review comment (not published)"},
}

var domainMap = func() map[string]DomainElement {
	m := make(map[string]DomainElement, len(domainElements))
	for _, e := range domainElements {
		m[e.Class] = e
	}
	return m
}()

var taskSections = []TaskSection{
	{"Prerequisites", "prereq", "prereq", "Content required before performing the task", 1},
	{"About this task", "context", "context", "Background information for the task", 2},
	{"Verification", "result", "result", "Expected result after completing the task", 4},
	{"Next steps", "postreq", "postreq", "Follow-up actions after completing the task", 5},
	{"", "tasktroubleshooting", "tasktroubleshooting", "Troubleshooting information for the task", 6},
}

var taskSectionByTitle = func() map[string]TaskSection {
	m := make(map[string]TaskSection, len(taskSections))
	for _, s := range taskSections {
		if s.DefaultTitle != "" {
			m[strings.ToLower(s.DefaultTitle)] = s
		}
	}
	return m
}()

var taskSectionByClass = func() map[string]TaskSection {
	m := make(map[string]TaskSection, len(taskSections))
	for _, s := range taskSections {
		m[s.Class] = s
	}
	return m
}()

var stepElements = []StepElement{
	{"stepresult", "stepresult", "Expected result after performing the step"},
	{"stepxmp", "stepxmp", "Example for the step"},
}

var stepElementMap = func() map[string]StepElement {
	m := make(map[string]StepElement, len(stepElements))
	for _, e := range stepElements {
		m[e.Class] = e
	}
	return m
}()

var conditionalAttributes = []ConditionalAttribute{
	{"audience", "Target audience (e.g., novice, expert)"},
	{"platform", "Target platform (e.g., linux, macos)"},
	{"product", "Product name or variant"},
	{"otherprops", "Custom profiling values"},
	{"deliveryTarget", "Output format (e.g., html, pdf)"},
	{"props", "Generic profiling attribute"},
	{"rev", "Revision identifier for flagging"},
}

var conditionalAttrSet = func() map[string]bool {
	m := make(map[string]bool, len(conditionalAttributes))
	for _, a := range conditionalAttributes {
		m[a.Name] = true
	}
	return m
}()

func LookupDomainElement(class string) (DomainElement, bool) {
	e, ok := domainMap[class]
	return e, ok
}

func LookupTaskSection(title string) (TaskSection, bool) {
	s, ok := taskSectionByTitle[strings.ToLower(title)]
	return s, ok
}

func LookupTaskSectionByClass(class string) (TaskSection, bool) {
	s, ok := taskSectionByClass[class]
	return s, ok
}

func LookupStepElement(class string) (StepElement, bool) {
	e, ok := stepElementMap[class]
	return e, ok
}

func IsConditionalAttribute(name string) bool {
	return conditionalAttrSet[name]
}

func AllDomainElements() []DomainElement {
	result := make([]DomainElement, len(domainElements))
	copy(result, domainElements)
	return result
}

func AllTaskSections() []TaskSection {
	result := make([]TaskSection, len(taskSections))
	copy(result, taskSections)
	return result
}

func AllConditionalAttributes() []ConditionalAttribute {
	result := make([]ConditionalAttribute, len(conditionalAttributes))
	copy(result, conditionalAttributes)
	return result
}

func DomainElementsByParentKind(kind string) []DomainElement {
	var result []DomainElement
	for _, e := range domainElements {
		if e.ParentKind == kind {
			result = append(result, e)
		}
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/vocabulary/ -v`
Expected: PASS — all tests green

- [ ] **Step 5: Run full test suite and lint**

Run: `make test && make lint`
Expected: All 202+ tests pass, zero lint issues

- [ ] **Step 6: Commit**

```bash
git add internal/vocabulary/
git commit -m "feat: add vocabulary registry for MDITA extended elements"
```

---

### Task 2: Attribute Parsing Foundation

**Files:**
- Create: `internal/document/attributes.go`
- Create: `internal/document/attributes_test.go`
- Modify: `internal/document/types.go` — add ParsedAttribute, extend Heading
- Modify: `internal/document/parser.go:59-69` — enable parser.WithAttribute(), extract heading attrs
- Modify: `internal/document/document.go:9-19` — add InlineAttrs/BlockAttrs fields

- [ ] **Step 1: Add ParsedAttribute type and extend Heading in types.go**

Add after the `Admonition` struct (line 156 of `internal/document/types.go`):

```go
type ParsedAttribute struct {
	Classes   []string
	ID        string
	KeyValues map[string]string
	Range     Range
}

type InlineAttribute struct {
	Attr       ParsedAttribute
	TargetKind string // "bold", "italic", "code", "paragraph"
	TargetText string
	Line       int
	Col        int
}

type BlockAttribute struct {
	Attr ParsedAttribute
	Line int
}
```

Extend `Heading` struct (line 87) — add `Attributes` field:

```go
type Heading struct {
	Level      int
	Text       string
	ID         string
	Slug       paths.Slug
	Range      Range
	Attributes *ParsedAttribute
}
```

Extend `Document` struct in `document.go` (line 9) — add new fields:

```go
type Document struct {
	URI         string
	Version     int
	Text        string
	Lines       []int
	Elements    []Element
	Symbols     []Symbol
	Index       *Index
	Meta        *YAMLMetadata
	Kind        DocKind
	InlineAttrs []InlineAttribute
	BlockAttrs  []BlockAttribute
}
```

- [ ] **Step 2: Write failing tests for the inline attribute scanner**

```go
// internal/document/attributes_test.go
package document

import (
	"testing"
)

func TestScanInlineAttributes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantCount  int
		wantClass  string
		wantTarget string
		wantText   string
	}{
		{
			name:       "bold with uicontrol",
			input:      "Click **Save**{.uicontrol} to save.",
			wantCount:  1,
			wantClass:  "uicontrol",
			wantTarget: "bold",
			wantText:   "Save",
		},
		{
			name:       "code with filepath",
			input:      "Edit `config.yaml`{.filepath} now.",
			wantCount:  1,
			wantClass:  "filepath",
			wantTarget: "code",
			wantText:   "config.yaml",
		},
		{
			name:       "italic with cite",
			input:      "Read *War and Peace*{.cite} today.",
			wantCount:  1,
			wantClass:  "cite",
			wantTarget: "italic",
			wantText:   "War and Peace",
		},
		{
			name:       "multiple on one line",
			input:      "Click **Save**{.uicontrol} and edit `file`{.filepath}.",
			wantCount:  2,
			wantClass:  "uicontrol",
			wantTarget: "bold",
			wantText:   "Save",
		},
		{
			name:      "no attributes",
			input:     "Normal **bold** and `code` text.",
			wantCount: 0,
		},
		{
			name:       "key-value attribute",
			input:      "For **experts**{audience=\"expert\"} only.",
			wantCount:  1,
			wantTarget: "bold",
			wantText:   "experts",
		},
		{
			name:       "underscore bold",
			input:      "Click __Save__{.uicontrol} to save.",
			wantCount:  1,
			wantClass:  "uicontrol",
			wantTarget: "bold",
			wantText:   "Save",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ScanInlineAttributes(tt.input)
			if len(attrs) != tt.wantCount {
				t.Fatalf("got %d attrs, want %d", len(attrs), tt.wantCount)
			}
			if tt.wantCount == 0 {
				return
			}
			if tt.wantClass != "" && len(attrs[0].Attr.Classes) > 0 && attrs[0].Attr.Classes[0] != tt.wantClass {
				t.Errorf("class = %q, want %q", attrs[0].Attr.Classes[0], tt.wantClass)
			}
			if tt.wantTarget != "" && attrs[0].TargetKind != tt.wantTarget {
				t.Errorf("target = %q, want %q", attrs[0].TargetKind, tt.wantTarget)
			}
			if tt.wantText != "" && attrs[0].TargetText != tt.wantText {
				t.Errorf("text = %q, want %q", attrs[0].TargetText, tt.wantText)
			}
		})
	}
}

func TestScanBlockAttributes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantKey   string
		wantVal   string
	}{
		{
			name:      "single block attr",
			input:     "{audience=\"novice\"}\n\n- Step one",
			wantCount: 1,
			wantKey:   "audience",
			wantVal:   "novice",
		},
		{
			name:      "multiple key-values",
			input:     "{platform=\"linux\" audience=\"expert\"}\n\n- Step one",
			wantCount: 1,
			wantKey:   "platform",
			wantVal:   "linux",
		},
		{
			name:      "no block attrs",
			input:     "Normal paragraph.\n\n- List item",
			wantCount: 0,
		},
		{
			name:      "class-only block attr",
			input:     "{.highlight}\n\nSome text.",
			wantCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := ScanBlockAttributes(tt.input)
			if len(attrs) != tt.wantCount {
				t.Fatalf("got %d attrs, want %d", len(attrs), tt.wantCount)
			}
			if tt.wantCount > 0 && tt.wantKey != "" {
				if v, ok := attrs[0].Attr.KeyValues[tt.wantKey]; !ok || v != tt.wantVal {
					t.Errorf("key %q = %q, want %q", tt.wantKey, v, tt.wantVal)
				}
			}
		})
	}
}

func TestParseAttrString(t *testing.T) {
	tests := []struct {
		input   string
		classes []string
		id      string
		kvCount int
	}{
		{".uicontrol", []string{"uicontrol"}, "", 0},
		{"#my-id .cls", []string{"cls"}, "my-id", 0},
		{`platform="linux"`, nil, "", 1},
		{`.filepath platform="linux"`, []string{"filepath"}, "", 1},
		{`.menucascade`, []string{"menucascade"}, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			attr := ParseAttrString(tt.input)
			if tt.classes != nil && len(attr.Classes) != len(tt.classes) {
				t.Fatalf("classes = %v, want %v", attr.Classes, tt.classes)
			}
			for i, c := range tt.classes {
				if attr.Classes[i] != c {
					t.Errorf("classes[%d] = %q, want %q", i, attr.Classes[i], c)
				}
			}
			if attr.ID != tt.id {
				t.Errorf("ID = %q, want %q", attr.ID, tt.id)
			}
			if len(attr.KeyValues) != tt.kvCount {
				t.Errorf("kvCount = %d, want %d", len(attr.KeyValues), tt.kvCount)
			}
		})
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/document/ -run TestScan -v`
Expected: FAIL — `ScanInlineAttributes` undefined

- [ ] **Step 4: Implement the attribute scanner**

```go
// internal/document/attributes.go
package document

import (
	"regexp"
	"strings"
)

var (
	inlineBoldAttrRegex      = regexp.MustCompile(`\*\*([^*]+)\*\*\{([^}]+)\}`)
	inlineBoldUnderAttrRegex = regexp.MustCompile(`__([^_]+)__\{([^}]+)\}`)
	inlineCodeAttrRegex      = regexp.MustCompile("` + "`" + `([^` + "`" + `]+)` + "`" + `\\{([^}]+)\\}`)
	inlineItalicAttrRegex    = regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*\{([^}]+)\}`)
	blockAttrRegex           = regexp.MustCompile(`(?m)^\{([^}]+)\}\s*$`)
	attrClassRegex           = regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9_-]*)`)
	attrIDRegex              = regexp.MustCompile(`#([a-zA-Z][a-zA-Z0-9_-]*)`)
	attrKVRegex              = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_-]*)="([^"]*)"`)
)

func ParseAttrString(s string) ParsedAttribute {
	attr := ParsedAttribute{KeyValues: make(map[string]string)}
	for _, m := range attrClassRegex.FindAllStringSubmatch(s, -1) {
		attr.Classes = append(attr.Classes, m[1])
	}
	if m := attrIDRegex.FindStringSubmatch(s); m != nil {
		attr.ID = m[1]
	}
	for _, m := range attrKVRegex.FindAllStringSubmatch(s, -1) {
		attr.KeyValues[m[1]] = m[2]
	}
	return attr
}

func ScanInlineAttributes(source string) []InlineAttribute {
	var attrs []InlineAttribute
	lines := strings.Split(source, "\n")

	for lineNum, line := range lines {
		attrs = append(attrs, scanLineInline(line, lineNum, inlineBoldAttrRegex, "bold")...)
		attrs = append(attrs, scanLineInline(line, lineNum, inlineBoldUnderAttrRegex, "bold")...)
		attrs = append(attrs, scanLineInline(line, lineNum, inlineCodeAttrRegex, "code")...)
		attrs = append(attrs, scanLineInline(line, lineNum, inlineItalicAttrRegex, "italic")...)
	}
	return attrs
}

func scanLineInline(line string, lineNum int, re *regexp.Regexp, kind string) []InlineAttribute {
	var attrs []InlineAttribute
	matches := re.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		text := line[m[2]:m[3]]
		attrStr := line[m[4]:m[5]]
		braceStart := strings.LastIndex(line[m[0]:m[1]], "{")
		col := m[0] + braceStart

		parsed := ParseAttrString(attrStr)
		parsed.Range = Rng(lineNum, col, lineNum, m[1])

		attrs = append(attrs, InlineAttribute{
			Attr:       parsed,
			TargetKind: kind,
			TargetText: text,
			Line:       lineNum,
			Col:        col,
		})
	}
	return attrs
}

func ScanBlockAttributes(source string) []BlockAttribute {
	var attrs []BlockAttribute
	matches := blockAttrRegex.FindAllStringSubmatchIndex(source, -1)
	for _, m := range matches {
		attrStr := source[m[2]:m[3]]
		line := strings.Count(source[:m[0]], "\n")
		parsed := ParseAttrString(attrStr)
		parsed.Range = rangeFromOffset(source, m[0], m[1])
		attrs = append(attrs, BlockAttribute{
			Attr: parsed,
			Line: line,
		})
	}
	return attrs
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/document/ -run TestScan -v && go test ./internal/document/ -run TestParseAttr -v`
Expected: PASS

- [ ] **Step 6: Enable parser.WithAttribute() and extract heading attributes**

Modify `internal/document/parser.go` — add `parser.WithAttribute()` to goldmark options (line 66-68):

```go
goldmark.WithParserOptions(
	parser.WithAutoHeadingID(),
	parser.WithAttribute(),
),
```

In the `case *ast.Heading:` block (lines 83-100), extract attributes from the AST node:

```go
case *ast.Heading:
	headingText := extractText(node, src)
	id := ""
	if idAttr, ok := node.AttributeString("id"); ok {
		if idBytes, ok := idAttr.([]byte); ok {
			id = string(idBytes)
		}
	}
	if id == "" {
		id = paths.Slugify(headingText)
	}
	var headingAttrs *ParsedAttribute
	if nodeAttrs := node.Attributes(); len(nodeAttrs) > 0 {
		pa := ParsedAttribute{KeyValues: make(map[string]string)}
		for _, attr := range nodeAttrs {
			key := string(attr.Name)
			val := ""
			switch v := attr.Value.(type) {
			case []byte:
				val = string(v)
			}
			if key == "class" {
				pa.Classes = strings.Fields(val)
			} else if key == "id" {
				// already handled above
			} else {
				pa.KeyValues[key] = val
			}
		}
		pa.Range = nodeRange(node, src)
		headingAttrs = &pa
		bf.HasAttributes = true
	}
	elements = append(elements, &Heading{
		Level:      node.Level,
		Text:       headingText,
		ID:         id,
		Slug:       paths.SlugOf(headingText),
		Range:      nodeRange(node, src),
		Attributes: headingAttrs,
	})
```

- [ ] **Step 7: Wire attribute scanning into Document construction**

In `internal/document/document.go`, after `Parse()` returns (line 22), call the scanners:

```go
func New(uri string, version int, text string) *Document {
	elements, bf, meta := Parse(text)
	idx := BuildIndex(elements, bf, meta)
	idx.Meta = meta

	kind := Topic
	if paths.IsMditaMapFile(uri, []string{"mditamap"}) {
		kind = Map
	}

	title := idx.Title()
	if title != nil && idx.ShortDesc == "" {
		idx.ShortDesc = findShortDesc(text, title)
	}

	inlineAttrs := ScanInlineAttributes(text)
	blockAttrs := ScanBlockAttributes(text)
	if len(inlineAttrs) > 0 || len(blockAttrs) > 0 {
		bf.HasAttributes = true
	}

	doc := &Document{
		URI:         uri,
		Version:     version,
		Text:        text,
		Lines:       buildLineMap(text),
		Elements:    elements,
		Index:       idx,
		Meta:        meta,
		Kind:        kind,
		InlineAttrs: inlineAttrs,
		BlockAttrs:  blockAttrs,
	}
	doc.Symbols = extractSymbols(doc)
	return doc
}
```

- [ ] **Step 8: Write integration test for heading attribute parsing**

Add to `internal/document/attributes_test.go`:

```go
func TestHeadingAttributesParsed(t *testing.T) {
	text := "# Install the software {.task}\n\nShort desc.\n\n## Prerequisites {.prereq}\n\nYou need admin access.\n"
	elements, bf, _ := Parse(text)
	if !bf.HasAttributes {
		t.Error("expected HasAttributes to be true")
	}
	headings := filterHeadings(elements)
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

func filterHeadings(elements []Element) []*Heading {
	var headings []*Heading
	for _, e := range elements {
		if h, ok := e.(*Heading); ok {
			headings = append(headings, h)
		}
	}
	return headings
}
```

- [ ] **Step 9: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 10: Commit**

```bash
git add internal/document/attributes.go internal/document/attributes_test.go internal/document/types.go internal/document/parser.go internal/document/document.go
git commit -m "feat: add attribute parsing for headings, inline elements, and blocks"
```

---

### Task 3: Task Section Detection + Diagnostics + Hover + Completion

**Files:**
- Modify: `internal/document/types.go` — add TaskSectionKind enum
- Modify: `internal/document/document.go` — add task section resolution
- Modify: `internal/diagnostic/diagnostic.go` — add codes 24, 25
- Modify: `internal/diagnostic/mdita.go` — add task section checks
- Modify: `internal/hover/hover.go` — add task section and heading class hover
- Modify: `internal/completion/partial.go` — add PartialHeadingClass kind
- Modify: `internal/completion/completion.go` — add heading class and task section completion

- [ ] **Step 1: Add TaskSectionKind enum to types.go**

Add after the `BlockAttribute` type:

```go
type TaskSectionKind int

const (
	TaskSectionNone TaskSectionKind = iota
	TaskSectionPrereq
	TaskSectionContext
	TaskSectionResult
	TaskSectionPostreq
	TaskSectionTroubleshooting
)
```

Extend the `Heading` struct — add `TaskSection` and `IsRelLinks` fields:

```go
type Heading struct {
	Level       int
	Text        string
	ID          string
	Slug        paths.Slug
	Range       Range
	Attributes  *ParsedAttribute
	TaskSection TaskSectionKind
	IsRelLinks  bool
}
```

- [ ] **Step 2: Write failing test for task section resolution**

Add to `internal/document/attributes_test.go`:

```go
func TestTaskSectionResolution(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n\n# Install {.task}\n\nShort desc.\n\n## Prerequisites\n\nNeed admin.\n\n## About this task\n\nContext info.\n\n1. Do the thing.\n\n## Verification\n\nCheck it worked.\n\n## Next steps\n\nConfigure.\n"
	doc := New("file:///test.md", 1, text)
	headings := doc.Index.Headings()
	// headings: Install, Prerequisites, About this task, Verification, Next steps
	if len(headings) < 5 {
		t.Fatalf("got %d headings, want >= 5", len(headings))
	}
	if headings[1].TaskSection != TaskSectionPrereq {
		t.Errorf("Prerequisites section = %d, want TaskSectionPrereq", headings[1].TaskSection)
	}
	if headings[2].TaskSection != TaskSectionContext {
		t.Errorf("About this task section = %d, want TaskSectionContext", headings[2].TaskSection)
	}
	if headings[3].TaskSection != TaskSectionResult {
		t.Errorf("Verification section = %d, want TaskSectionResult", headings[3].TaskSection)
	}
	if headings[4].TaskSection != TaskSectionPostreq {
		t.Errorf("Next steps section = %d, want TaskSectionPostreq", headings[4].TaskSection)
	}
}

func TestTaskSectionByClassAttr(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n\n# Install {.task}\n\nShort desc.\n\n## Before you begin {.prereq}\n\nNeed admin.\n"
	doc := New("file:///test.md", 1, text)
	headings := doc.Index.Headings()
	if len(headings) < 2 {
		t.Fatalf("got %d headings, want >= 2", len(headings))
	}
	if headings[1].TaskSection != TaskSectionPrereq {
		t.Errorf("section = %d, want TaskSectionPrereq", headings[1].TaskSection)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/document/ -run TestTaskSection -v`
Expected: FAIL — TaskSectionPrereq is always 0

- [ ] **Step 4: Implement task section resolution in document.go**

Add a `resolveTaskSections` function and call it from `New()`:

```go
func resolveTaskSections(doc *Document) {
	if doc.Meta == nil || doc.Meta.Schema != SchemaTask {
		isTask := false
		for _, e := range doc.Elements {
			if h, ok := e.(*Heading); ok && h.IsTitle() && h.Attributes != nil {
				for _, c := range h.Attributes.Classes {
					if c == "task" {
						isTask = true
					}
				}
			}
		}
		if !isTask {
			return
		}
	}
	for _, e := range doc.Elements {
		h, ok := e.(*Heading)
		if !ok || h.Level < 2 {
			continue
		}
		if h.Attributes != nil {
			for _, c := range h.Attributes.Classes {
				h.TaskSection = taskSectionKindFromClass(c)
				if h.TaskSection != TaskSectionNone {
					break
				}
			}
		}
		if h.TaskSection == TaskSectionNone {
			h.TaskSection = taskSectionKindFromTitle(h.Text)
		}
		lowerText := strings.ToLower(h.Text)
		if lowerText == "related information" || lowerText == "related links" {
			h.IsRelLinks = true
		}
		if h.Attributes != nil {
			for _, c := range h.Attributes.Classes {
				if c == "related-links" {
					h.IsRelLinks = true
				}
			}
		}
	}
}

func taskSectionKindFromTitle(title string) TaskSectionKind {
	switch strings.ToLower(title) {
	case "prerequisites":
		return TaskSectionPrereq
	case "about this task":
		return TaskSectionContext
	case "verification":
		return TaskSectionResult
	case "next steps":
		return TaskSectionPostreq
	default:
		return TaskSectionNone
	}
}

func taskSectionKindFromClass(class string) TaskSectionKind {
	switch class {
	case "prereq":
		return TaskSectionPrereq
	case "context":
		return TaskSectionContext
	case "result":
		return TaskSectionResult
	case "postreq":
		return TaskSectionPostreq
	case "tasktroubleshooting":
		return TaskSectionTroubleshooting
	default:
		return TaskSectionNone
	}
}
```

Call `resolveTaskSections(doc)` in `New()` before returning, after `extractSymbols`.

Also resolve `IsRelLinks` for non-task topics — the `resolveTaskSections` loop can also handle this. Extract the `IsRelLinks` logic into the general heading walk, or make `resolveTaskSections` always run the related-links check regardless of task status.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/document/ -run TestTaskSection -v`
Expected: PASS

- [ ] **Step 6: Write failing tests for task section diagnostics**

```go
// Add to internal/diagnostic/diagnostic_test.go or create extended_test.go
func TestDuplicateTaskSection(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:task.xsd", "---", "",
		"# Install {.task}", "", "Short desc.", "",
		"## Prerequisites", "", "First.", "",
		"## Prerequisites", "", "Second.", "",
		"1. Do the thing.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeDuplicateTaskSection {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeDuplicateTaskSection diagnostic")
	}
}

func TestTaskSectionOutOfOrder(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:task.xsd", "---", "",
		"# Install {.task}", "", "Short desc.", "",
		"## Verification", "", "Check it.", "",
		"## Prerequisites", "", "Need admin.", "",
		"1. Do the thing.",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeTaskSectionOutOfOrder {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeTaskSectionOutOfOrder diagnostic")
	}
}
```

- [ ] **Step 7: Add diagnostic codes and implement task section checks**

In `internal/diagnostic/diagnostic.go`, add codes after line 37:

```go
CodeUnknownOutputclass              = "20"
CodeDomainClassWrongParent          = "21"
CodeExtendedProfileRequired         = "22"
CodeUnknownConditionalAttribute     = "23"
CodeTaskSectionOutOfOrder           = "24"
CodeDuplicateTaskSection            = "25"
CodeRelLinksNonLinkContent          = "26"
CodeMenucascadeMissingSeparator     = "27"
CodeStepElementOutsideStep          = "28"
CodeReltableInconsistentColumns     = "29"
```

In `internal/diagnostic/mdita.go`, add a `checkTaskSections` function:

```go
func checkTaskSections(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	seen := make(map[document.TaskSectionKind]bool)
	lastOrder := 0

	for _, h := range doc.Index.Headings() {
		if h.TaskSection == document.TaskSectionNone {
			continue
		}
		if seen[h.TaskSection] {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityError,
				Code:     CodeDuplicateTaskSection,
				Source:   source,
				Message:  "Duplicate task section: \"" + h.Text + "\"",
			})
			continue
		}
		seen[h.TaskSection] = true

		order := taskSectionOrder(h.TaskSection)
		if order > 0 && order < lastOrder {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeTaskSectionOutOfOrder,
				Source:   source,
				Message:  "Task section \"" + h.Text + "\" should appear earlier",
			})
		}
		if order > lastOrder {
			lastOrder = order
		}
	}
	return diags
}

func taskSectionOrder(kind document.TaskSectionKind) int {
	switch kind {
	case document.TaskSectionPrereq:
		return 1
	case document.TaskSectionContext:
		return 2
	case document.TaskSectionResult:
		return 4
	case document.TaskSectionPostreq:
		return 5
	case document.TaskSectionTroubleshooting:
		return 6
	default:
		return 0
	}
}
```

Call `checkTaskSections` from `checkSchemaSpecific` inside the `SchemaTask` case, and also when the H1 heading has `{.task}` class.

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/diagnostic/ -run TestDuplicate -v && go test ./internal/diagnostic/ -run TestTaskSection -v`
Expected: PASS

- [ ] **Step 9: Add task section hover**

In `internal/hover/hover.go`, extend the heading hover case (line 25):

```go
case *document.Heading:
	if el.TaskSection != document.TaskSectionNone {
		return hoverTaskSection(el)
	}
	if el.Attributes != nil && len(el.Attributes.Classes) > 0 {
		return hoverHeadingClass(el)
	}
	return "**" + el.Text + "** (level " + itoa(el.Level) + ")"
```

Add the helper functions:

```go
func hoverTaskSection(h *document.Heading) string {
	desc := map[document.TaskSectionKind]struct{ elem, text string }{
		document.TaskSectionPrereq:           {"prereq", "Content required before performing the task"},
		document.TaskSectionContext:          {"context", "Background information for the task"},
		document.TaskSectionResult:           {"result", "Expected result after completing the task"},
		document.TaskSectionPostreq:          {"postreq", "Follow-up actions after completing the task"},
		document.TaskSectionTroubleshooting: {"tasktroubleshooting", "Troubleshooting information for the task"},
	}
	if d, ok := desc[h.TaskSection]; ok {
		return "DITA `<" + d.elem + ">` — " + d.text
	}
	return "**" + h.Text + "** (level " + itoa(h.Level) + ")"
}

func hoverHeadingClass(h *document.Heading) string {
	class := h.Attributes.Classes[0]
	topicTypes := map[string]string{
		"task":      "Task topic — procedure-oriented content with steps",
		"concept":   "Concept topic — explanatory or overview content",
		"reference": "Reference topic — lookup-oriented content (tables, lists)",
	}
	if desc, ok := topicTypes[class]; ok {
		return "DITA `<" + class + ">` — " + desc
	}
	return "**" + h.Text + "** (level " + itoa(h.Level) + ", class: " + class + ")"
}
```

- [ ] **Step 10: Add task section heading completion**

In `internal/completion/partial.go`, add a new `PartialKind`:

```go
const (
	PartialInlineLink PartialKind = iota
	PartialInlineAnchor
	PartialRefLink
	PartialYamlKey
	PartialKeyref
	PartialHeadingText
)
```

In `DetectPartial`, add detection for `## ` prefix (before the `](` check):

```go
if strings.HasPrefix(strings.TrimSpace(prefix), "## ") && !strings.Contains(prefix, "[") {
	input := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(prefix), "## "))
	return &PartialElement{
		Kind:  PartialHeadingText,
		Input: input,
	}
}
```

In `internal/completion/completion.go`, add the completion case and function:

```go
case PartialHeadingText:
	return completeTaskSectionHeading(pe.Input, doc)
```

```go
func completeTaskSectionHeading(input string, doc *document.Document) []CompletionItem {
	isTask := false
	if doc.Meta != nil && doc.Meta.Schema == document.SchemaTask {
		isTask = true
	}
	for _, e := range doc.Elements {
		if h, ok := e.(*document.Heading); ok && h.IsTitle() && h.Attributes != nil {
			for _, c := range h.Attributes.Classes {
				if c == "task" {
					isTask = true
				}
			}
		}
	}
	if !isTask {
		return nil
	}

	existing := make(map[string]bool)
	for _, h := range doc.Index.Headings() {
		existing[strings.ToLower(h.Text)] = true
	}

	titles := []struct{ title, detail string }{
		{"Prerequisites", "prereq — before the task"},
		{"About this task", "context — background info"},
		{"Verification", "result — expected outcome"},
		{"Next steps", "postreq — follow-up actions"},
		{"Related information", "related-links — links after body"},
	}

	var items []CompletionItem
	for _, t := range titles {
		if existing[strings.ToLower(t.title)] {
			continue
		}
		if input != "" && !strings.Contains(strings.ToLower(t.title), strings.ToLower(input)) {
			continue
		}
		items = append(items, CompletionItem{
			Label:      t.title,
			Detail:     t.detail,
			InsertText: t.title,
			Kind:       17,
		})
	}
	return items
}
```

- [ ] **Step 11: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 12: Commit**

```bash
git add internal/document/ internal/diagnostic/ internal/hover/ internal/completion/
git commit -m "feat: add task section detection, diagnostics, hover, and completion"
```

---

### Task 4: Domain Specialization Diagnostics + Hover + Completion

**Files:**
- Modify: `internal/diagnostic/mdita.go` — add outputclass and parent-kind checks
- Modify: `internal/hover/hover.go` — add inline attribute hover
- Modify: `internal/completion/partial.go` — add PartialAttrClass kind
- Modify: `internal/completion/completion.go` — add domain class completion

- [ ] **Step 1: Write failing tests for domain specialization diagnostics**

```go
// internal/diagnostic/extended_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/diagnostic/ -run TestUnknown -v`
Expected: FAIL

- [ ] **Step 3: Implement domain specialization diagnostics**

Add to `internal/diagnostic/mdita.go`:

```go
func checkInlineAttributes(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	for _, ia := range doc.InlineAttrs {
		for _, class := range ia.Attr.Classes {
			elem, ok := vocabulary.LookupDomainElement(class)
			if !ok {
				_, isStep := vocabulary.LookupStepElement(class)
				if !isStep {
					diags = append(diags, Diagnostic{
						Range:    ia.Attr.Range,
						Severity: SeverityWarning,
						Code:     CodeUnknownOutputclass,
						Source:   source,
						Message:  "Unknown outputclass \"{." + class + "}\"",
					})
				}
				continue
			}
			if elem.ParentKind != ia.TargetKind {
				diags = append(diags, Diagnostic{
					Range:    ia.Attr.Range,
					Severity: SeverityWarning,
					Code:     CodeDomainClassWrongParent,
					Source:   source,
					Message:  "Domain class \"" + class + "\" expects " + elem.ParentKind + ", not " + ia.TargetKind,
				})
			}
			if class == "menucascade" && !strings.Contains(ia.TargetText, " > ") {
				diags = append(diags, Diagnostic{
					Range:    ia.Attr.Range,
					Severity: SeverityWarning,
					Code:     CodeMenucascadeMissingSeparator,
					Source:   source,
					Message:  "menucascade requires \" > \" separator between menu items",
				})
			}
		}
	}
	return diags
}
```

Add the import of `vocabulary` package. Call `checkInlineAttributes` from `checkMditaCompliance`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/diagnostic/ -run TestUnknown -v && go test ./internal/diagnostic/ -run TestDomainClass -v && go test ./internal/diagnostic/ -run TestMenucascade -v && go test ./internal/diagnostic/ -run TestValidDomain -v`
Expected: PASS

- [ ] **Step 5: Add inline attribute hover**

In `internal/hover/hover.go`, add a check before the `elem` dispatch (after the YAML check, before `ElementAt`):

```go
if h := hoverInlineAttribute(doc, pos); h != "" {
	return h
}
```

```go
func hoverInlineAttribute(doc *document.Document, pos document.Position) string {
	for _, ia := range doc.InlineAttrs {
		if ia.Line != pos.Line {
			continue
		}
		if pos.Character < ia.Col || pos.Character > ia.Attr.Range.End.Character {
			continue
		}
		for _, class := range ia.Attr.Classes {
			if elem, ok := vocabulary.LookupDomainElement(class); ok {
				return "DITA `<" + elem.DITAElement + ">` (" + elem.Domain + ") — " + elem.Description
			}
			if se, ok := vocabulary.LookupStepElement(class); ok {
				return "DITA `<" + se.DITAElement + ">` — " + se.Description
			}
		}
		for key := range ia.Attr.KeyValues {
			if vocabulary.IsConditionalAttribute(key) {
				return "DITA conditional processing attribute `" + key + "`"
			}
		}
	}
	return ""
}
```

- [ ] **Step 6: Add domain class completion**

In `internal/completion/partial.go`, add `PartialAttrClass`:

```go
PartialAttrClass
```

Add detection in `DetectPartial` — look for `{.` pattern:

```go
if idx := strings.LastIndex(prefix, "{."); idx >= 0 {
	if !strings.Contains(prefix[idx:], "}") {
		input := prefix[idx+2:]
		return &PartialElement{
			Kind:  PartialAttrClass,
			Input: input,
		}
	}
}
```

In `completion.go`, add the case and function:

```go
case PartialAttrClass:
	return completeAttrClass(pe.Input, doc, pos)
```

```go
func completeAttrClass(input string, doc *document.Document, pos document.Position) []CompletionItem {
	var items []CompletionItem

	// Determine context (heading vs inline)
	isHeading := false
	line := ""
	lines := strings.Split(doc.Text, "\n")
	if pos.Line < len(lines) {
		line = lines[pos.Line]
	}
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		isHeading = true
	}

	if isHeading {
		headingClasses := []struct{ class, detail string }{
			{"task", "Task topic type"},
			{"concept", "Concept topic type"},
			{"reference", "Reference topic type"},
			{"prereq", "Task prerequisite section"},
			{"context", "Task context section"},
			{"result", "Task result section"},
			{"postreq", "Task post-requisite section"},
			{"tasktroubleshooting", "Task troubleshooting section"},
			{"related-links", "Related links section"},
		}
		for _, c := range headingClasses {
			if input == "" || strings.HasPrefix(c.class, input) {
				items = append(items, CompletionItem{
					Label:      c.class,
					Detail:     c.detail,
					InsertText: c.class + "}",
					Kind:       6,
				})
			}
		}
		return items
	}

	// Determine inline context
	parentKind := ""
	if strings.Contains(line[:pos.Character], "**") || strings.Contains(line[:pos.Character], "__") {
		parentKind = "bold"
	} else if strings.Contains(line[:pos.Character], "`") {
		parentKind = "code"
	} else if strings.Contains(line[:pos.Character], "*") {
		parentKind = "italic"
	}

	for _, elem := range vocabulary.AllDomainElements() {
		if parentKind != "" && elem.ParentKind != parentKind {
			continue
		}
		if input == "" || strings.HasPrefix(elem.Class, input) {
			items = append(items, CompletionItem{
				Label:      elem.Class,
				Detail:     "<" + elem.DITAElement + "> (" + elem.Domain + ")",
				InsertText: elem.Class + "}",
				Kind:       6,
			})
		}
	}
	return items
}
```

Add import for `vocabulary` package.

- [ ] **Step 7: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 8: Commit**

```bash
git add internal/diagnostic/ internal/hover/ internal/completion/
git commit -m "feat: add domain specialization diagnostics, hover, and completion"
```

---

### Task 5: Conditional Processing Attributes

**Files:**
- Modify: `internal/diagnostic/mdita.go` — add conditional attr checks
- Modify: `internal/hover/hover.go` — add block attribute hover
- Modify: `internal/completion/partial.go` — add PartialBlockAttr kind
- Modify: `internal/completion/completion.go` — add conditional attribute completion

- [ ] **Step 1: Write failing tests for conditional attribute diagnostics**

```go
func TestUnknownConditionalAttribute(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# Topic", "", "Short desc.", "",
		"{badattr=\"value\"}", "", "- Step one",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeUnknownConditionalAttribute {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeUnknownConditionalAttribute diagnostic")
	}
}

func TestValidConditionalAttribute(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:mdita:extended:rng:topic.rng", "---", "",
		"# Topic", "", "Short desc.", "",
		"{platform=\"linux\"}", "", "- Step one",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	for _, d := range diags {
		if d.Code == CodeUnknownConditionalAttribute {
			t.Errorf("unexpected diagnostic for valid attr: %s", d.Message)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/diagnostic/ -run TestUnknownConditional -v`
Expected: FAIL

- [ ] **Step 3: Implement conditional attribute diagnostics**

Add to `internal/diagnostic/mdita.go`:

```go
func checkBlockAttributes(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	for _, ba := range doc.BlockAttrs {
		for key := range ba.Attr.KeyValues {
			if !vocabulary.IsConditionalAttribute(key) {
				diags = append(diags, Diagnostic{
					Range:    ba.Attr.Range,
					Severity: SeverityWarning,
					Code:     CodeUnknownConditionalAttribute,
					Source:   source,
					Message:  "Unknown conditional attribute \"" + key + "\"",
				})
			}
		}
	}
	return diags
}
```

Call from `checkMditaCompliance`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/diagnostic/ -run TestUnknownConditional -v && go test ./internal/diagnostic/ -run TestValidConditional -v`
Expected: PASS

- [ ] **Step 5: Add block attribute hover**

In `internal/hover/hover.go`, add block attribute hover check (before `ElementAt`):

```go
if h := hoverBlockAttribute(doc, pos); h != "" {
	return h
}
```

```go
func hoverBlockAttribute(doc *document.Document, pos document.Position) string {
	for _, ba := range doc.BlockAttrs {
		if ba.Line != pos.Line {
			continue
		}
		for key, val := range ba.Attr.KeyValues {
			if vocabulary.IsConditionalAttribute(key) {
				for _, ca := range vocabulary.AllConditionalAttributes() {
					if ca.Name == key {
						return "DITA conditional processing attribute `" + key + "` — " + ca.Description + "\n\nCurrent value: `" + val + "`"
					}
				}
			}
		}
	}
	return ""
}
```

- [ ] **Step 6: Add conditional attribute completion**

In `internal/completion/partial.go`, add `PartialBlockAttr`:

```go
PartialBlockAttr
```

Add detection for `{` on standalone line:

```go
trimmed := strings.TrimSpace(prefix)
if strings.HasPrefix(trimmed, "{") && !strings.Contains(trimmed, "}") && !strings.Contains(line, "#") {
	if col == len(line) || strings.TrimSpace(line[col:]) == "" {
		input := strings.TrimPrefix(trimmed, "{")
		return &PartialElement{
			Kind:  PartialBlockAttr,
			Input: input,
		}
	}
}
```

In `completion.go`, add:

```go
case PartialBlockAttr:
	return completeBlockAttr(pe.Input)
```

```go
func completeBlockAttr(input string) []CompletionItem {
	var items []CompletionItem
	for _, ca := range vocabulary.AllConditionalAttributes() {
		snippet := ca.Name + "=\"\""
		if input == "" || strings.HasPrefix(ca.Name, input) {
			items = append(items, CompletionItem{
				Label:      ca.Name,
				Detail:     ca.Description,
				InsertText: snippet,
				Kind:       6,
			})
		}
	}
	return items
}
```

- [ ] **Step 7: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 8: Commit**

```bash
git add internal/diagnostic/ internal/hover/ internal/completion/
git commit -m "feat: add conditional processing attribute diagnostics, hover, and completion"
```

---

### Task 6: Related Links Section

**Files:**
- Modify: `internal/document/types.go` — add RelatedLinksInfo
- Modify: `internal/document/document.go` — detect and store related links
- Modify: `internal/diagnostic/mdita.go` — add non-link content check
- Modify: `internal/hover/hover.go` — add related links hover

- [ ] **Step 1: Add RelatedLinksInfo type and document field**

In `types.go`, add:

```go
type RelatedLinksInfo struct {
	HeadingLine int
	Links       []*MdLink
}
```

In `Document` struct, add:

```go
RelLinks *RelatedLinksInfo
```

- [ ] **Step 2: Write failing test for related links detection**

```go
func TestRelatedLinksDetection(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n\n# Install {.task}\n\nShort desc.\n\n1. Do the thing.\n\n## Related information\n\n- [Concept](concept.md)\n- [Reference](reference.md)\n"
	doc := New("file:///test.md", 1, text)
	if doc.RelLinks == nil {
		t.Fatal("expected RelLinks to be set")
	}
	if len(doc.RelLinks.Links) != 2 {
		t.Errorf("RelLinks.Links = %d, want 2", len(doc.RelLinks.Links))
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/document/ -run TestRelatedLinks -v`
Expected: FAIL

- [ ] **Step 4: Implement related links detection in document.go**

Add a `resolveRelatedLinks` function called from `New()`:

```go
func resolveRelatedLinks(doc *Document) {
	for _, e := range doc.Elements {
		h, ok := e.(*Heading)
		if !ok || !h.IsRelLinks {
			continue
		}
		rl := &RelatedLinksInfo{HeadingLine: h.Range.Start.Line}
		for _, el := range doc.Elements {
			ml, ok := el.(*MdLink)
			if !ok {
				continue
			}
			if ml.Range.Start.Line > h.Range.Start.Line {
				rl.Links = append(rl.Links, ml)
			}
		}
		doc.RelLinks = rl
		return
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/document/ -run TestRelatedLinks -v`
Expected: PASS

- [ ] **Step 6: Add related links diagnostic and hover**

Diagnostic in `mdita.go`:

```go
func checkRelatedLinks(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	for _, h := range doc.Index.Headings() {
		if !h.IsRelLinks {
			continue
		}
		lines := strings.Split(doc.Text, "\n")
		for i := h.Range.Start.Line + 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "#") {
				break
			}
			if !strings.HasPrefix(line, "- [") && !strings.HasPrefix(line, "* [") {
				diags = append(diags, Diagnostic{
					Range:    document.Rng(i, 0, i, len(lines[i])),
					Severity: SeverityWarning,
					Code:     CodeRelLinksNonLinkContent,
					Source:   source,
					Message:  "Related links section should contain only links",
				})
			}
		}
	}
	return diags
}
```

Hover in `hover.go` — extend the heading case:

```go
if el.IsRelLinks {
	return "DITA `<related-links>` — Links placed after the topic body in DITA output"
}
```

- [ ] **Step 7: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 8: Commit**

```bash
git add internal/document/ internal/diagnostic/ internal/hover/
git commit -m "feat: add related links section detection, diagnostics, and hover"
```

---

### Task 7: Step-Level Elements

**Files:**
- Modify: `internal/diagnostic/mdita.go` — add step element context check
- Modify: `internal/hover/hover.go` — already handled by inline attribute hover
- Modify: `internal/completion/completion.go` — step element class completion already covered

- [ ] **Step 1: Write failing test for stepresult outside step**

```go
func TestStepElementOutsideStep(t *testing.T) {
	doc := makeDoc("file:///test.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:concept.xsd", "---", "",
		"# Concept Topic", "", "Short desc.", "",
		"Some text.{.stepresult}",
	)
	f := makeFolder(doc)
	diags := Check(doc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeStepElementOutsideStep {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeStepElementOutsideStep diagnostic")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/diagnostic/ -run TestStepElement -v`
Expected: FAIL

- [ ] **Step 3: Implement step element context check**

Add to `internal/diagnostic/mdita.go`:

```go
func checkStepElements(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	isTask := doc.Meta != nil && doc.Meta.Schema == document.SchemaTask
	if !isTask {
		for _, e := range doc.Elements {
			if h, ok := e.(*document.Heading); ok && h.IsTitle() && h.Attributes != nil {
				for _, c := range h.Attributes.Classes {
					if c == "task" {
						isTask = true
					}
				}
			}
		}
	}

	for _, ia := range doc.InlineAttrs {
		for _, class := range ia.Attr.Classes {
			if _, ok := vocabulary.LookupStepElement(class); ok && !isTask {
				diags = append(diags, Diagnostic{
					Range:    ia.Attr.Range,
					Severity: SeverityWarning,
					Code:     CodeStepElementOutsideStep,
					Source:   source,
					Message:  class + " is only valid inside a task topic",
				})
			}
		}
	}
	return diags
}
```

Call from `checkMditaCompliance`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/diagnostic/ -run TestStepElement -v`
Expected: PASS

- [ ] **Step 5: Run full test suite and lint, commit**

Run: `make test && make lint`

```bash
git add internal/diagnostic/
git commit -m "feat: add step-level element diagnostics"
```

---

### Task 8: Ditamap Extensions (Reltable + Mapref)

**Files:**
- Modify: `internal/ditamap/ditamap.go` — add reltable parsing, mapref detection
- Create: `internal/ditamap/reltable_test.go`
- Modify: `internal/diagnostic/ditamap_validation.go` — add reltable checks
- Modify: `internal/hover/hover.go` — add mapref hover

- [ ] **Step 1: Add RelTable types to ditamap.go**

```go
type RelCell struct {
	TopicRefs []TopicRef
}

type RelRow struct {
	Cells []RelCell
}

type RelTable struct {
	Header []RelCell
	Rows   []RelRow
	Line   int
}
```

Extend `TopicRef`:

```go
type TopicRef struct {
	Href     string
	Title    string
	Children []TopicRef
	IsMapRef bool
}
```

Extend `MapStructure`:

```go
type MapStructure struct {
	Title     string
	TopicRefs []TopicRef
	RelTables []RelTable
}
```

- [ ] **Step 2: Write failing tests for reltable parsing and mapref**

```go
// internal/ditamap/reltable_test.go
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/ditamap/ -run TestParseMap -v`
Expected: FAIL

- [ ] **Step 4: Implement reltable parsing and mapref detection**

In `internal/ditamap/ditamap.go`, import `extension` and `gmast`:

```go
import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)
```

Update `ParseMap` to use tables extension and detect tables and mapref:

```go
func ParseMap(source string) (*MapStructure, error) {
	src := []byte(source)
	md := goldmark.New(
		goldmark.WithExtensions(extension.NewTable()),
	)
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	m := &MapStructure{}

	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Heading:
			if n.Level == 1 && m.Title == "" {
				m.Title = extractText(n, src)
			}
		case *ast.List:
			m.TopicRefs = append(m.TopicRefs, parseListItems(n, src)...)
		case *gmast.Table:
			rt := parseRelTable(n, src)
			m.RelTables = append(m.RelTables, rt)
		}
	}

	return m, nil
}
```

Add `parseRelTable` function:

```go
func parseRelTable(table *gmast.Table, src []byte) RelTable {
	rt := RelTable{}

	for c := table.FirstChild(); c != nil; c = c.NextSibling() {
		switch row := c.(type) {
		case *gmast.TableHeader:
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*gmast.TableCell); ok {
					rt.Header = append(rt.Header, parseCellRefs(tc, src))
				}
			}
		case *gmast.TableRow:
			rr := RelRow{}
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*gmast.TableCell); ok {
					rr.Cells = append(rr.Cells, parseCellRefs(tc, src))
				}
			}
			rt.Rows = append(rt.Rows, rr)
		}
	}
	return rt
}

func parseCellRefs(cell *gmast.TableCell, src []byte) RelCell {
	rc := RelCell{}
	_ = ast.Walk(cell, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok {
			ref := TopicRef{
				Href:  string(link.Destination),
				Title: extractText(link, src),
			}
			ref.IsMapRef = isMapRefHref(ref.Href)
			rc.TopicRefs = append(rc.TopicRefs, ref)
		}
		return ast.WalkContinue, nil
	})
	return rc
}
```

Add mapref detection to `parseListItems` — after setting `ref.Href`:

```go
ref.IsMapRef = isMapRefHref(ref.Href)
```

Add the helper:

```go
func isMapRefHref(href string) bool {
	return strings.HasSuffix(href, ".ditamap") || strings.HasSuffix(href, ".mditamap")
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ditamap/ -v`
Expected: PASS

- [ ] **Step 6: Add reltable diagnostic**

In `internal/diagnostic/ditamap_validation.go`, add:

```go
func checkReltable(doc *document.Document, mapStruct *ditamap.MapStructure) []Diagnostic {
	var diags []Diagnostic
	for _, rt := range mapStruct.RelTables {
		headerCols := len(rt.Header)
		if headerCols == 0 {
			continue
		}
		for i, row := range rt.Rows {
			if len(row.Cells) != headerCols {
				diags = append(diags, Diagnostic{
					Range:    document.Rng(rt.Line+i+2, 0, rt.Line+i+2, 0),
					Severity: SeverityWarning,
					Code:     CodeReltableInconsistentColumns,
					Source:   source,
					Message:  fmt.Sprintf("Relationship table row has %d columns, header has %d", len(row.Cells), headerCols),
				})
			}
		}
	}
	return diags
}
```

- [ ] **Step 7: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 8: Commit**

```bash
git add internal/ditamap/ internal/diagnostic/
git commit -m "feat: add relationship table parsing and mapref detection in ditamaps"
```

---

### Task 9: Semantic Tokens + Inlay Hints + Code Actions

**Files:**
- Modify: `internal/semantic/semantic.go` — add attribute tokens
- Modify: `internal/inlayhint/inlayhint.go` — add domain element hints
- Modify: `internal/codeaction/codeaction.go` — add related links and task section actions

- [ ] **Step 1: Add attribute semantic tokens**

In `internal/semantic/semantic.go`, update `TokenTypes` and `collectTokens`:

```go
const (
	TokenTypeRefLink   = 0
	TokenTypeDecorator = 1
)

var TokenTypes = []string{"class", "decorator"}
```

```go
func collectTokens(doc *document.Document) []token {
	var tokens []token

	for _, ia := range doc.InlineAttrs {
		tokens = append(tokens, token{
			line:   ia.Line,
			char:   ia.Col,
			length: ia.Attr.Range.End.Character - ia.Col,
			typ:    TokenTypeDecorator,
		})
	}
	for _, ba := range doc.BlockAttrs {
		tokens = append(tokens, token{
			line:   ba.Line,
			char:   ba.Attr.Range.Start.Character,
			length: ba.Attr.Range.End.Character - ba.Attr.Range.Start.Character,
			typ:    TokenTypeDecorator,
		})
	}

	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].line != tokens[j].line {
			return tokens[i].line < tokens[j].line
		}
		return tokens[i].char < tokens[j].char
	})

	return tokens
}
```

- [ ] **Step 2: Add domain element inlay hints**

In `internal/inlayhint/inlayhint.go`, add domain element hints:

```go
import "github.com/aireilly/mdita-lsp/internal/vocabulary"
```

Add after the `keyrefHints` call:

```go
hints = append(hints, domainHints(doc, rng)...)
```

```go
func domainHints(doc *document.Document, rng document.Range) []InlayHint {
	var hints []InlayHint
	for _, ia := range doc.InlineAttrs {
		if ia.Line < rng.Start.Line || ia.Line > rng.End.Line {
			continue
		}
		for _, class := range ia.Attr.Classes {
			if elem, ok := vocabulary.LookupDomainElement(class); ok {
				hints = append(hints, InlayHint{
					Position: document.Position{
						Line:      ia.Line,
						Character: ia.Attr.Range.End.Character,
					},
					Label: " → <" + elem.DITAElement + ">",
					Kind:  KindType,
				})
			}
		}
	}
	return hints
}
```

- [ ] **Step 3: Add related links and task section code actions**

In `internal/codeaction/codeaction.go`, add:

```go
actions = append(actions, addRelatedLinksAction(doc)...)
actions = append(actions, addTaskSectionActions(doc)...)
```

```go
func addRelatedLinksAction(doc *document.Document) []CodeAction {
	if doc.Kind != document.Topic {
		return nil
	}
	if doc.RelLinks != nil {
		return nil
	}
	for _, h := range doc.Index.Headings() {
		if h.IsRelLinks {
			return nil
		}
	}
	lastLine := len(strings.Split(doc.Text, "\n")) - 1
	return []CodeAction{{
		Title:  "Add related links section",
		Kind:   "source",
		DocURI: doc.URI,
		Edit: &TextEdit{
			Range:   document.Rng(lastLine, 0, lastLine, 0),
			NewText: "\n## Related information\n\n- []()\n",
		},
	}}
}

func addTaskSectionActions(doc *document.Document) []CodeAction {
	isTask := doc.Meta != nil && doc.Meta.Schema == document.SchemaTask
	if !isTask {
		for _, e := range doc.Elements {
			if h, ok := e.(*document.Heading); ok && h.IsTitle() && h.Attributes != nil {
				for _, c := range h.Attributes.Classes {
					if c == "task" {
						isTask = true
					}
				}
			}
		}
	}
	if !isTask {
		return nil
	}

	existing := make(map[document.TaskSectionKind]bool)
	for _, h := range doc.Index.Headings() {
		if h.TaskSection != document.TaskSectionNone {
			existing[h.TaskSection] = true
		}
	}

	type sectionTemplate struct {
		kind  document.TaskSectionKind
		title string
	}
	templates := []sectionTemplate{
		{document.TaskSectionPrereq, "Prerequisites"},
		{document.TaskSectionContext, "About this task"},
		{document.TaskSectionResult, "Verification"},
		{document.TaskSectionPostreq, "Next steps"},
	}

	var actions []CodeAction
	for _, tmpl := range templates {
		if existing[tmpl.kind] {
			continue
		}
		lastLine := len(strings.Split(doc.Text, "\n")) - 1
		actions = append(actions, CodeAction{
			Title:  "Add " + tmpl.title + " section",
			Kind:   "source",
			DocURI: doc.URI,
			Edit: &TextEdit{
				Range:   document.Rng(lastLine, 0, lastLine, 0),
				NewText: "\n## " + tmpl.title + "\n\n\n",
			},
		})
	}
	return actions
}
```

- [ ] **Step 4: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 5: Commit**

```bash
git add internal/semantic/ internal/inlayhint/ internal/codeaction/
git commit -m "feat: add semantic tokens, inlay hints, and code actions for extended MDITA"
```

---

### Task 10: Update Map Body Content Diagnostic + Final Integration

**Files:**
- Modify: `internal/diagnostic/mdita.go:120-131` — stop flagging tables in maps (reltables are valid)
- Modify: `internal/diagnostic/diagnostic.go` — add extended diagnostics config gate
- Create: `internal/document/attributes_integration_test.go` — end-to-end tests

- [ ] **Step 1: Fix map body content diagnostic for reltables**

In `internal/diagnostic/mdita.go`, update `checkSchemaSpecific` (line 120-131). Tables in map files are valid when they represent relationship tables. Remove `bf.HasTable` from the map body content check:

```go
if doc.Kind == document.Map {
	hasNonLinkContent := bf.HasOrderedList || bf.HasDefinitionList
	if hasNonLinkContent {
		// ...existing diagnostic...
	}
}
```

- [ ] **Step 2: Write end-to-end integration test**

```go
// internal/document/attributes_integration_test.go
package document

import (
	"testing"
)

func TestFullExtendedDocument(t *testing.T) {
	text := `---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---

# Install the software {.task}

Short description of the installation.

## Prerequisites

You need administrator access.

{platform="linux"}

- Ensure you have sudo privileges.

## About this task

This procedure installs the base package.

1. Click **File > Open**{.menucascade} to open the dialog.

2. Edit ` + "`config.yaml`{.filepath}" + ` to set options.

3. Run the ` + "`installer`{.cmdname}" + ` command.

   The installer runs.{.stepresult}

## Verification

The software is now installed.

## Next steps

Configure the license key.

## Related information

- [Concept](concept.md)
- [Reference](reference.md)
`
	doc := New("file:///test.md", 1, text)

	// Heading attributes
	title := doc.Index.Title()
	if title == nil {
		t.Fatal("no title")
	}
	if title.Attributes == nil || len(title.Attributes.Classes) == 0 || title.Attributes.Classes[0] != "task" {
		t.Error("title should have .task class")
	}

	// Task sections
	headings := doc.Index.Headings()
	sectionCount := 0
	for _, h := range headings {
		if h.TaskSection != TaskSectionNone {
			sectionCount++
		}
	}
	if sectionCount != 4 {
		t.Errorf("task sections = %d, want 4", sectionCount)
	}

	// Related links
	if doc.RelLinks == nil {
		t.Error("expected related links")
	}

	// Inline attributes
	if len(doc.InlineAttrs) < 3 {
		t.Errorf("inline attrs = %d, want >= 3", len(doc.InlineAttrs))
	}

	// Block attributes
	if len(doc.BlockAttrs) < 1 {
		t.Errorf("block attrs = %d, want >= 1", len(doc.BlockAttrs))
	}

	// HasAttributes
	if !doc.Index.Features.HasAttributes {
		t.Error("expected HasAttributes to be true")
	}
}
```

- [ ] **Step 3: Run full test suite and lint**

Run: `make test && make lint`
Expected: All tests pass, zero lint issues

- [ ] **Step 4: Commit**

```bash
git add internal/
git commit -m "feat: integration tests and diagnostic fix for MDITA extended parity"
```

- [ ] **Step 5: Update CLAUDE.md with new package and capability info**

In `CLAUDE.md`, add `vocabulary/` to the project structure and update the LSP capabilities list to include the new features. Add the new diagnostic codes to the documentation.

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with MDITA extended features"
```

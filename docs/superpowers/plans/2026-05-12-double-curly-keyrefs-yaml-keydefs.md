# Double-Curly Keyrefs and YAML Keydefs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `{{key}}` inline keyref syntax and YAML `keys:` keydef support to the mdita-lsp, matching the redhat.mdita.extended DITA-OT plugin v0.0.8.

**Architecture:** Two stacked features — Feature A (YAML `keys:` keydefs) enriches the key table with text/URL values from map front matter; Feature B (`{{key}}` syntax) adds a second keyref surface syntax alongside `[key]`. Feature A provides the data layer; Feature B consumes it through detection, completion, hover, definition, diagnostics, and inlay hints.

**Tech Stack:** Go, goldmark (existing), yaml.v3 (existing)

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/document/types.go` | Modify | Add `Keys` field to `YAMLMetadata` |
| `internal/document/parser.go` | Modify | Parse `keys:` block; export `ParseYAMLMeta` |
| `internal/document/parser_test.go` | Modify | Tests for YAML keys parsing |
| `internal/keyref/keyref.go` | Modify | Add `Value` to `KeyEntry`; `isURLValue`; YAML key extraction in `BuildMergedTable` |
| `internal/keyref/keyref_test.go` | Modify | Tests for YAML keydef extraction |
| `internal/keyref/coderegion.go` | Create | Fenced code block + inline code region detection |
| `internal/keyref/coderegion_test.go` | Create | Tests for code region detection |
| `internal/keyref/detect.go` | Modify | Add `{{key}}` detection to `DetectAll` and `DetectAtPosition` |
| `internal/keyref/detect_test.go` | Modify | Tests for `{{key}}` detection |
| `internal/completion/partial.go` | Modify | Add `PartialDoubleCurlyKeyref` detection |
| `internal/completion/completion.go` | Modify | Add `completeDoubleCurlyKeyref`; `"keys"` to `yamlKeys` |
| `internal/completion/resolve.go` | Modify | Handle `Value` in keyref docs |
| `internal/completion/completion_test.go` | Modify | Tests for `{{` completion |
| `internal/hover/hover.go` | Modify | Handle `Value` in hover; YAML `keys:` hover |
| `internal/hover/hover_test.go` | Modify | Tests for `{{key}}` and YAML keys hover |
| `internal/definition/definition.go` | Modify | Handle YAML-defined keys in `resolveKeyref` |
| `internal/definition/definition_test.go` | Modify | Tests for `{{key}}` go-to-def |
| `internal/diagnostic/keyref.go` | Modify | Add `{{key}}` scanning to `CheckKeyrefs` |
| `internal/diagnostic/keyref_test.go` | Modify | Tests for `{{key}}` diagnostics |
| `internal/inlayhint/inlayhint.go` | Modify | Show `Value` in keyref hints; detect `{{key}}` |
| `internal/inlayhint/inlayhint_test.go` | Modify | Tests for `{{key}}` inlay hints |

---

### Task 1: Add `Value` to `KeyEntry` and `Keys` to `YAMLMetadata`

**Files:**
- Modify: `internal/keyref/keyref.go:10-13`
- Modify: `internal/document/types.go:122-135`

- [ ] **Step 1: Add `Value` field to `KeyEntry`**

In `internal/keyref/keyref.go`, add the `Value` field:

```go
type KeyEntry struct {
	Href  string
	Title string
	Value string
}
```

- [ ] **Step 2: Add `Keys` field to `YAMLMetadata`**

In `internal/document/types.go`, add `Keys` to the struct:

```go
type YAMLMetadata struct {
	Author      string
	Source      string
	Publisher   string
	Permissions string
	Audience    string
	Category    string
	Keywords    []string
	ResourceID  string
	Schema      DitaSchema
	SchemaRaw   string
	OtherMeta   map[string]string
	Keys        map[string]string
	Range       Range
}
```

- [ ] **Step 3: Run tests to verify no regressions**

Run: `make test`
Expected: All 210+ tests pass — the new fields have zero values and don't affect existing behavior.

- [ ] **Step 4: Commit**

```bash
git add internal/keyref/keyref.go internal/document/types.go
git commit -m "feat: add Value to KeyEntry and Keys to YAMLMetadata"
```

---

### Task 2: Parse `keys:` block in YAML front matter

**Files:**
- Modify: `internal/document/parser.go:181-223`
- Modify: `internal/document/parser_test.go` (or create if not present)

- [ ] **Step 1: Write the failing test**

Create or add to `internal/document/parser_test.go`:

```go
package document

import "testing"

func TestParseYAMLKeys(t *testing.T) {
	source := "---\n$schema: urn:oasis:names:tc:dita:xsd:map.xsd\nkeys:\n  product-name: \"Red Hat OpenShift\"\n  version: \"4.15\"\n---\n# Map\n"
	_, _, meta := Parse(source)
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.Keys == nil {
		t.Fatal("expected Keys to be parsed")
	}
	if meta.Keys["product-name"] != "Red Hat OpenShift" {
		t.Errorf("product-name = %q, want %q", meta.Keys["product-name"], "Red Hat OpenShift")
	}
	if meta.Keys["version"] != "4.15" {
		t.Errorf("version = %q, want %q", meta.Keys["version"], "4.15")
	}
}

func TestParseYAMLKeysEmpty(t *testing.T) {
	source := "---\nauthor: Jane\n---\n# Doc\n"
	_, _, meta := Parse(source)
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.Keys != nil {
		t.Errorf("expected nil Keys when no keys: block, got %v", meta.Keys)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/document/ -run TestParseYAMLKeys -v`
Expected: FAIL — `Keys` is nil because `parseYAMLMeta` doesn't handle `"keys"` yet.

- [ ] **Step 3: Implement `keys:` parsing in `parseYAMLMeta`**

In `internal/document/parser.go`, add a case in the `parseYAMLMeta` switch (after the `"keyword"` case):

```go
		case "keys":
			if m, ok := val.(map[string]any); ok {
				meta.Keys = make(map[string]string, len(m))
				for k, v := range m {
					if s, ok := v.(string); ok {
						meta.Keys[k] = s
					}
				}
			}
```

- [ ] **Step 4: Export `ParseYAMLMeta` for external callers**

Add this exported function after `parseYAMLMeta` in `internal/document/parser.go`:

```go
func ParseYAMLMeta(text string) *YAMLMetadata {
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return nil
	}
	closeIdx := strings.Index(text[4:], "\n---")
	if closeIdx < 0 {
		closeIdx = strings.Index(text[4:], "\n...")
	}
	if closeIdx < 0 {
		return nil
	}
	yamlBlock := text[4 : 4+closeIdx]
	return parseYAMLMeta(yamlBlock)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/document/ -run TestParseYAMLKeys -v`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/document/parser.go internal/document/parser_test.go
git commit -m "feat: parse YAML keys: block in map front matter"
```

---

### Task 3: Extract YAML keydefs in `BuildMergedTable`

**Files:**
- Modify: `internal/keyref/keyref.go:55-70`
- Modify: `internal/keyref/keyref_test.go`

- [ ] **Step 1: Write the failing test for YAML keydefs**

Add to `internal/keyref/keyref_test.go`:

```go
func TestBuildMergedTableYAMLKeys(t *testing.T) {
	mapText := "---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n  version: \"4.15\"\n  docs-url: \"https://docs.example.com\"\n---\n# Map\n\n- [Install](install.md)\n"
	table := BuildMergedTable([]string{mapText})

	// stem-based key from TopicRef
	if _, ok := table["install"]; !ok {
		t.Error("expected 'install' key from TopicRef")
	}

	// YAML text keydef
	entry, ok := table["product-name"]
	if !ok {
		t.Fatal("expected 'product-name' key from YAML keys:")
	}
	if entry.Value != "Red Hat OpenShift" {
		t.Errorf("Value = %q, want %q", entry.Value, "Red Hat OpenShift")
	}
	if entry.Title != "Red Hat OpenShift" {
		t.Errorf("Title = %q, want %q", entry.Title, "Red Hat OpenShift")
	}
	if entry.Href != "" {
		t.Errorf("Href = %q, want empty for text keydef", entry.Href)
	}

	// YAML URL keydef
	urlEntry, ok := table["docs-url"]
	if !ok {
		t.Fatal("expected 'docs-url' key from YAML keys:")
	}
	if urlEntry.Href != "https://docs.example.com" {
		t.Errorf("Href = %q, want %q", urlEntry.Href, "https://docs.example.com")
	}
	if urlEntry.Value != "" {
		t.Errorf("Value = %q, want empty for URL keydef", urlEntry.Value)
	}
}

func TestBuildMergedTableYAMLKeysPrecedence(t *testing.T) {
	mapText := "---\nkeys:\n  install: \"Installation Guide\"\n---\n# Map\n\n- [Install](install.md)\n"
	table := BuildMergedTable([]string{mapText})

	entry := table["install"]
	// YAML key takes precedence over stem-based
	if entry.Value != "Installation Guide" {
		t.Errorf("YAML key should take precedence: Value = %q", entry.Value)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/keyref/ -run TestBuildMergedTableYAML -v`
Expected: FAIL — `product-name` key not found.

- [ ] **Step 3: Add `isURLValue` helper and update `BuildMergedTable`**

In `internal/keyref/keyref.go`, add the helper and update `BuildMergedTable`:

```go
func isURLValue(s string) bool {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return true
	}
	for _, ext := range []string{".md", ".dita", ".html", ".xml"} {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}

func BuildMergedTable(mapTexts []string) KeyTable {
	merged := make(KeyTable)
	for _, text := range mapTexts {
		m, err := ditamap.ParseMap(text)
		if err != nil {
			continue
		}
		table := ExtractKeys(m)
		for k, v := range table {
			if _, exists := merged[k]; !exists {
				merged[k] = v
			}
		}

		meta := document.ParseYAMLMeta(text)
		if meta != nil && meta.Keys != nil {
			for k, v := range meta.Keys {
				if isURLValue(v) {
					merged[k] = KeyEntry{Href: v}
				} else {
					merged[k] = KeyEntry{Value: v, Title: v}
				}
			}
		}
	}
	return merged
}
```

Add `"github.com/aireilly/mdita-lsp/internal/document"` to the imports at the top of the file. Also add `"strings"` to imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/keyref/ -run TestBuildMergedTableYAML -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/keyref/keyref.go internal/keyref/keyref_test.go
git commit -m "feat: extract YAML keys: keydefs into merged key table"
```

---

### Task 4: Code-region detection helper

**Files:**
- Create: `internal/keyref/coderegion.go`
- Create: `internal/keyref/coderegion_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/keyref/coderegion_test.go`:

```go
package keyref

import "testing"

func TestFencedCodeLines(t *testing.T) {
	text := "# Title\n\nSome text.\n\n```yaml\napiVersion: v1\nkind: Pod\n```\n\nMore text.\n"
	fenced := fencedCodeLines(text)
	// lines 4-7 (0-indexed) are inside the fenced block
	if !fenced[4] || !fenced[5] || !fenced[6] || !fenced[7] {
		t.Errorf("expected lines 4-7 to be fenced, got %v", fenced)
	}
	if fenced[0] || fenced[2] || fenced[8] || fenced[9] {
		t.Error("non-fenced lines should not be marked")
	}
}

func TestFencedCodeLinesMultiple(t *testing.T) {
	text := "Text\n\n```\ncode1\n```\n\nMiddle\n\n```\ncode2\n```\n"
	fenced := fencedCodeLines(text)
	if !fenced[2] || !fenced[3] || !fenced[4] {
		t.Error("first block should be fenced")
	}
	if fenced[5] || fenced[6] {
		t.Error("middle text should not be fenced")
	}
	if !fenced[8] || !fenced[9] || !fenced[10] {
		t.Error("second block should be fenced")
	}
}

func TestIsInInlineCode(t *testing.T) {
	tests := []struct {
		line   string
		col    int
		expect bool
	}{
		{"Use `{{key}}` here", 6, true},   // inside backticks
		{"Use `{{key}}` here", 0, false},  // before backticks
		{"Use `{{key}}` here", 14, false}, // after backticks
		{"No code here {{key}}", 14, false},
		{"Double ``{{key}}`` tick", 10, true},
	}
	for _, tt := range tests {
		got := isInInlineCode(tt.line, tt.col)
		if got != tt.expect {
			t.Errorf("isInInlineCode(%q, %d) = %v, want %v", tt.line, tt.col, got, tt.expect)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/keyref/ -run "TestFencedCode|TestIsInInlineCode" -v`
Expected: FAIL — functions not defined.

- [ ] **Step 3: Implement code-region helpers**

Create `internal/keyref/coderegion.go`:

```go
package keyref

import "strings"

func fencedCodeLines(text string) map[int]bool {
	lines := strings.Split(text, "\n")
	fenced := make(map[int]bool)
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced[i] = true
			inFence = !inFence
			continue
		}
		if inFence {
			fenced[i] = true
		}
	}
	return fenced
}

func isInInlineCode(line string, col int) bool {
	inCode := false
	i := 0
	for i < len(line) {
		if line[i] == '`' {
			ticks := 0
			for i < len(line) && line[i] == '`' {
				ticks++
				i++
			}
			if inCode {
				inCode = false
			} else {
				inCode = true
			}
			continue
		}
		if i == col {
			return inCode
		}
		i++
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/keyref/ -run "TestFencedCode|TestIsInInlineCode" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/keyref/coderegion.go internal/keyref/coderegion_test.go
git commit -m "feat: add code-region detection helpers for keyref parsing"
```

---

### Task 5: `{{key}}` detection in `DetectAll` and `DetectAtPosition`

**Files:**
- Modify: `internal/keyref/detect.go`
- Modify: `internal/keyref/detect_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/keyref/detect_test.go`:

```go
func TestDetectAllDoubleCurly(t *testing.T) {
	text := "# About {{product-name}}\n\nThe version is {{version}}.\n"
	locs := DetectAll(text)
	var dcKeys []string
	for _, l := range locs {
		dcKeys = append(dcKeys, l.Key)
	}
	if len(dcKeys) < 2 {
		t.Fatalf("expected at least 2 keyrefs, got %d: %v", len(dcKeys), dcKeys)
	}
	found := map[string]bool{}
	for _, k := range dcKeys {
		found[k] = true
	}
	if !found["product-name"] {
		t.Error("expected 'product-name' key")
	}
	if !found["version"] {
		t.Error("expected 'version' key")
	}
}

func TestDetectAllDoubleCurlySkipsCodeBlock(t *testing.T) {
	text := "# Title\n\n```\n{{template-var}}\n```\n\n{{real-key}} here.\n"
	locs := DetectAll(text)
	for _, l := range locs {
		if l.Key == "template-var" {
			t.Error("should not detect {{key}} inside fenced code block")
		}
	}
	found := false
	for _, l := range locs {
		if l.Key == "real-key" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'real-key' outside code block")
	}
}

func TestDetectAllDoubleCurlySkipsInlineCode(t *testing.T) {
	text := "Use `{{template}}` for templates.\n"
	locs := DetectAll(text)
	for _, l := range locs {
		if l.Key == "template" {
			t.Error("should not detect {{key}} inside inline code")
		}
	}
}

func TestDetectAtPositionDoubleCurly(t *testing.T) {
	text := "Install {{product-name}} now."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 12})
	if kr == nil {
		t.Fatal("expected keyref at position")
	}
	if kr.Label != "product-name" {
		t.Errorf("Label = %q, want %q", kr.Label, "product-name")
	}
	if kr.Range.Start.Character != 8 {
		t.Errorf("Range.Start.Character = %d, want 8", kr.Range.Start.Character)
	}
	if kr.Range.End.Character != 24 {
		t.Errorf("Range.End.Character = %d, want 24", kr.Range.End.Character)
	}
}

func TestDetectAtPositionDoubleCurlyOutside(t *testing.T) {
	text := "Install {{product-name}} now."
	kr := DetectAtPosition(text, document.Position{Line: 0, Character: 2})
	if kr != nil {
		t.Error("expected no keyref outside {{}} range")
	}
}

func TestDetectDoubleCurlyInvalidKey(t *testing.T) {
	text := "See {{invalid key}} here.\n"
	locs := DetectAll(text)
	for _, l := range locs {
		if l.Key == "invalid key" {
			t.Error("should not detect key with spaces")
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/keyref/ -run "TestDetect.*DoubleCurly|TestDetect.*Invalid" -v`
Expected: FAIL — `{{key}}` not detected.

- [ ] **Step 3: Implement `{{key}}` detection**

In `internal/keyref/detect.go`, add the regex and update both functions:

```go
var doubleCurlyRe = regexp.MustCompile(`\{\{([a-zA-Z0-9_.\-]+)\}\}`)

func DetectAll(text string) []KeyrefLocation {
	lines := strings.Split(text, "\n")
	fenced := fencedCodeLines(text)
	var locs []KeyrefLocation
	for i, line := range lines {
		// existing [key] detection
		matches := shortcutRefRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			bracketStart := m[0]
			labelStart := m[2]
			labelEnd := m[3]
			if bracketStart > 0 && line[bracketStart-1] == '[' {
				continue
			}
			label := line[labelStart:labelEnd]
			if strings.HasPrefix(label, "^") {
				continue
			}
			locs = append(locs, KeyrefLocation{
				Key:     label,
				Line:    i,
				EndChar: labelEnd + 1,
			})
		}

		// {{key}} detection — skip fenced code blocks
		if fenced[i] {
			continue
		}
		dcMatches := doubleCurlyRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range dcMatches {
			matchStart := m[0]
			keyStart := m[2]
			keyEnd := m[3]
			if isInInlineCode(line, matchStart) {
				continue
			}
			locs = append(locs, KeyrefLocation{
				Key:     line[keyStart:keyEnd],
				Line:    i,
				EndChar: m[1],
			})
		}
	}
	return locs
}

func DetectAtPosition(text string, pos document.Position) *KeyrefAtPos {
	lines := strings.Split(text, "\n")
	if pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]

	// existing [key] detection
	matches := shortcutRefRe.FindAllStringSubmatchIndex(line, -1)
	for _, m := range matches {
		bracketStart := m[0]
		labelStart := m[2]
		labelEnd := m[3]

		if bracketStart > 0 && line[bracketStart-1] == '[' {
			continue
		}

		label := line[labelStart:labelEnd]
		if strings.HasPrefix(label, "^") {
			continue
		}

		if pos.Character >= labelStart && pos.Character <= labelEnd {
			return &KeyrefAtPos{
				Label: label,
				Range: document.Rng(pos.Line, labelStart, pos.Line, labelEnd),
			}
		}
	}

	// {{key}} detection
	fenced := fencedCodeLines(text)
	if !fenced[pos.Line] {
		dcMatches := doubleCurlyRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range dcMatches {
			matchStart := m[0]
			matchEnd := m[1]
			keyStart := m[2]
			keyEnd := m[3]
			if isInInlineCode(line, matchStart) {
				continue
			}
			if pos.Character >= matchStart && pos.Character < matchEnd {
				return &KeyrefAtPos{
					Label: line[keyStart:keyEnd],
					Range: document.Rng(pos.Line, matchStart, pos.Line, matchEnd),
				}
			}
		}
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/keyref/ -v`
Expected: All keyref tests pass.

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/keyref/detect.go internal/keyref/detect_test.go
git commit -m "feat: detect {{key}} double-curly keyref syntax"
```

---

### Task 6: `{{key}}` completion

**Files:**
- Modify: `internal/completion/partial.go:9-21`
- Modify: `internal/completion/completion.go:30-33,35-59`
- Modify: `internal/completion/completion_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/completion/completion_test.go`:

```go
func TestDetectPartialDoubleCurlyKeyref(t *testing.T) {
	text := "# Title\n\nInstall {{prod"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 16})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialDoubleCurlyKeyref {
		t.Errorf("Kind = %v, want PartialDoubleCurlyKeyref", pe.Kind)
	}
	if pe.Input != "prod" {
		t.Errorf("Input = %q, want %q", pe.Input, "prod")
	}
}

func TestDetectPartialDoubleCurlyEmpty(t *testing.T) {
	text := "# Title\n\nInstall {{"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 12})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialDoubleCurlyKeyref {
		t.Errorf("Kind = %v, want PartialDoubleCurlyKeyref", pe.Kind)
	}
	if pe.Input != "" {
		t.Errorf("Input = %q, want empty", pe.Input)
	}
}

func TestCompleteDoubleCurlyKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n  version: \"4.15\"\n---\n# Map\n\n- [Install](install.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nInstall {{prod")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)
	items := Complete(topicDoc, document.Position{Line: 2, Character: 16}, f)
	found := false
	for _, item := range items {
		if item.Label == "product-name" {
			found = true
			if item.Kind != 6 {
				t.Errorf("Kind = %d, want 6 (variable)", item.Kind)
			}
		}
	}
	if !found {
		labels := make([]string, len(items))
		for i, it := range items {
			labels[i] = it.Label
		}
		t.Errorf("expected 'product-name' completion, got %v", labels)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/completion/ -run "TestDetectPartialDoubleCurly|TestCompleteDoubleCurly" -v`
Expected: FAIL — `PartialDoubleCurlyKeyref` not defined.

- [ ] **Step 3: Add `PartialDoubleCurlyKeyref` to `partial.go`**

In `internal/completion/partial.go`, add to the const block:

```go
const (
	PartialInlineLink PartialKind = iota
	PartialInlineAnchor
	PartialRefLink
	PartialYamlKey
	PartialKeyref
	PartialHeadingText
	PartialAttrClass
	PartialBlockAttr
	PartialAttrOpen
	PartialDoubleCurlyKeyref
)
```

In `DetectPartial()`, add `{{` detection after the block attribute check and before the `{.` attribute class check. Insert this block after the `inYamlBlock` check and before the block attribute `{` check:

```go
	// Detect {{key}} double-curly keyref
	if idx := strings.LastIndex(prefix, "{{"); idx >= 0 {
		if !strings.Contains(prefix[idx:], "}}") {
			input := prefix[idx+2:]
			startChar := idx
			endChar := col
			hasSuffix := false
			if col < len(line) {
				suffix := line[col:]
				if ci := strings.Index(suffix, "}}"); ci >= 0 {
					endChar = col + ci + 2
					hasSuffix = true
				}
			}
			if !hasSuffix {
				endChar = col
			}
			return &PartialElement{
				Kind:  PartialDoubleCurlyKeyref,
				Input: input,
				Range: document.Range{
					Start: document.Position{Line: pos.Line, Character: startChar},
					End:   document.Position{Line: pos.Line, Character: endChar},
				},
			}
		}
	}
```

- [ ] **Step 4: Add `completeDoubleCurlyKeyref` to `completion.go`**

Add `"keys"` to `yamlKeys` slice:

```go
var yamlKeys = []string{
	"$schema", "id", "author", "source", "publisher", "permissions",
	"audience", "category", "keyword", "resourceid", "keys",
}
```

Add the dispatch case in `Complete()`:

```go
	case PartialDoubleCurlyKeyref:
		return completeDoubleCurlyKeyref(pe.Input, doc, folder, pe.Range)
```

Add the handler function:

```go
func completeDoubleCurlyKeyref(input string, doc *document.Document, folder *workspace.Folder, editRange document.Range) []CompletionItem {
	table := keyref.BuildMergedTable(folder.MapTexts())

	var items []CompletionItem
	for _, key := range keyref.AllKeys(table) {
		if input == "" || strings.Contains(strings.ToLower(key), strings.ToLower(input)) {
			entry := table[key]
			detail := entry.Href
			if entry.Value != "" {
				detail = entry.Value
			} else if entry.Title != "" {
				detail = entry.Title + " (" + entry.Href + ")"
			}
			items = append(items, CompletionItem{
				Label:  key,
				Detail: detail,
				Kind:   6,
				Data:   map[string]string{"kind": "keyref"},
				TextEdit: &TextEdit{
					Range:   editRange,
					NewText: "{{" + key + "}}",
				},
			})
		}
	}
	return items
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/completion/ -run "TestDetectPartialDoubleCurly|TestCompleteDoubleCurly" -v`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/completion/partial.go internal/completion/completion.go internal/completion/completion_test.go
git commit -m "feat: add {{key}} double-curly keyref completion"
```

---

### Task 7: `{{key}}` hover and YAML `keys:` hover

**Files:**
- Modify: `internal/hover/hover.go:58-68,110-123`
- Modify: `internal/hover/hover_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/hover/hover_test.go`:

```go
func TestHoverDoubleCurlyKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n---\n# Map\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nInstall {{product-name}} now.\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)
	result := GetHover(topicDoc, document.Position{Line: 2, Character: 14}, f)
	if result == "" {
		t.Fatal("expected hover content for {{keyref}}")
	}
	if !strings.Contains(result, "product-name") {
		t.Errorf("hover = %q, expected to contain 'product-name'", result)
	}
	if !strings.Contains(result, "Red Hat OpenShift") {
		t.Errorf("hover = %q, expected to contain resolved value", result)
	}
}

func TestHoverDoubleCurlyKeyrefUnresolved(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install](install.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nInstall {{nonexistent-key}} now.\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)
	result := GetHover(topicDoc, document.Position{Line: 2, Character: 14}, f)
	if result != "" {
		t.Errorf("expected empty hover for unresolved {{keyref}}, got %q", result)
	}
}

func TestHoverYAMLKeysEntry(t *testing.T) {
	doc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n---\n# Map\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	result := GetHover(doc, document.Position{Line: 1, Character: 2}, f)
	if !strings.Contains(result, "keys") {
		t.Errorf("expected hover for 'keys', got %q", result)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/hover/ -run "TestHoverDoubleCurly|TestHoverYAMLKeys" -v`
Expected: FAIL — no hover for `{{key}}`.

- [ ] **Step 3: Update hover for `{{key}}` and `Value` display**

In `internal/hover/hover.go`, update `hoverKeyref` to handle `Value`:

```go
func hoverKeyref(kr *keyref.KeyrefAtPos, folder *workspace.Folder) string {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := keyref.Resolve(table, kr.Label)
	if !ok {
		return ""
	}
	result := "**" + kr.Label + "** (keyref)"
	if entry.Value != "" {
		result += "\n\nValue: " + entry.Value
	} else if entry.Title != "" {
		result += "\n\nTarget: " + entry.Title + " (" + entry.Href + ")"
	} else if entry.Href != "" {
		result += "\n\nTarget: " + entry.Href
	}
	return result
}
```

Add `"keys"` to `yamlKeyDocs`:

```go
var yamlKeyDocs = map[string]string{
	"$schema":     "DITA topic type schema. Values: `urn:oasis:names:tc:mdita:xsd:topic.xsd` (core), `urn:oasis:names:tc:mdita:extended:rng:topic.rng` (extended)",
	"author":      "Topic author name",
	"source":      "Original source of the content",
	"publisher":   "Publisher of the content",
	"permissions": "Access permissions for this topic",
	"audience":    "Intended audience for this topic",
	"category":    "Topic category for classification",
	"keyword":     "Keywords for indexing and search (comma-separated or YAML list)",
	"resourceid":  "Unique resource identifier for cross-references",
	"keys":        "Key definitions for keyword keyrefs (map files only)",
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/hover/ -run "TestHoverDoubleCurly|TestHoverYAMLKeys" -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/hover/hover.go internal/hover/hover_test.go
git commit -m "feat: add hover for {{key}} keyrefs and YAML keys: entries"
```

---

### Task 8: `{{key}}` go-to-definition

**Files:**
- Modify: `internal/definition/definition.go:62-87`
- Modify: `internal/definition/definition_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/definition/definition_test.go`:

```go
func TestGotoDefDoubleCurlyKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n---\n# Map\n\n- [Install](install.md)\n")
	install := document.New("file:///project/install.md", 1,
		"# Installation\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nInstall {{product-name}} now.\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(install)
	f.AddDoc(topicDoc)

	locs := GotoDef(topicDoc, document.Position{Line: 2, Character: 14}, f)
	if len(locs) == 0 {
		t.Fatal("expected definition location for {{product-name}}")
	}
	if locs[0].URI != "file:///project/map.mditamap" {
		t.Errorf("expected map URI, got %q", locs[0].URI)
	}
}

func TestGotoDefDoubleCurlyKeyrefHref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install Guide](install.md)\n")
	install := document.New("file:///project/install.md", 1,
		"# Installation\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nSee {{install}} for details.\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(install)
	f.AddDoc(topicDoc)

	locs := GotoDef(topicDoc, document.Position{Line: 2, Character: 8}, f)
	if len(locs) == 0 {
		t.Fatal("expected definition location for {{install}}")
	}
	if locs[0].URI != "file:///project/install.md" {
		t.Errorf("expected install.md URI, got %q", locs[0].URI)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/definition/ -run "TestGotoDefDoubleCurly" -v`
Expected: FAIL — no location returned.

- [ ] **Step 3: Update `resolveKeyref` to handle YAML-defined keys**

In `internal/definition/definition.go`, update `resolveKeyref`:

```go
func resolveKeyref(kr *keyref.KeyrefAtPos, doc *document.Document, folder *workspace.Folder) []Location {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := keyref.Resolve(table, kr.Label)
	if !ok {
		return nil
	}

	// YAML-defined text keydef: navigate to the map file's keys: block
	if entry.Value != "" && entry.Href == "" {
		for _, mapDoc := range folder.AllDocs() {
			if mapDoc.Kind != document.Map {
				continue
			}
			if mapDoc.Meta != nil && mapDoc.Meta.Keys != nil {
				if _, ok := mapDoc.Meta.Keys[kr.Label]; ok {
					defRange := findYAMLKeyLine(mapDoc.Text, kr.Label)
					return []Location{{URI: mapDoc.URI, Range: defRange}}
				}
			}
		}
		return nil
	}

	// Href-based keydef: navigate to target topic file
	for _, mapDoc := range folder.AllDocs() {
		if mapDoc.Kind != document.Map {
			continue
		}
		mapPath, _ := paths.URIToPath(mapDoc.URI)
		mapDir := filepath.Dir(mapPath)
		targetPath := filepath.Join(mapDir, entry.Href)
		targetURI := paths.PathToURI(targetPath)
		target := folder.DocByURI(targetURI)
		if target != nil {
			title := target.Index.Title()
			if title != nil {
				return []Location{{URI: target.URI, Range: title.Range}}
			}
			return []Location{{URI: target.URI, Range: document.Rng(0, 0, 0, 0)}}
		}
	}
	return nil
}
```

Add the `findYAMLKeyLine` helper in the same file:

```go
func findYAMLKeyLine(text string, key string) document.Range {
	lines := strings.Split(text, "\n")
	target := "  " + key + ":"
	for i, line := range lines {
		if strings.HasPrefix(line, target) {
			return document.Rng(i, 2, i, 2+len(key))
		}
	}
	return document.Rng(0, 0, 0, 0)
}
```

Add `"strings"` to the imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/definition/ -run "TestGotoDefDoubleCurly" -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/definition/definition.go internal/definition/definition_test.go
git commit -m "feat: add go-to-definition for {{key}} keyrefs"
```

---

### Task 9: `{{key}}` diagnostics

**Files:**
- Modify: `internal/diagnostic/keyref.go`
- Modify: `internal/diagnostic/keyref_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/diagnostic/keyref_test.go`:

```go
func TestUnresolvedDoubleCurlyKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install](install.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nInstall {{nonexistent-key}} now.\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)

	diags := CheckKeyrefs(topicDoc, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeUnresolvedKeyref && strings.Contains(d.Message, "nonexistent-key") {
			found = true
		}
	}
	if !found {
		t.Error("expected UnresolvedKeyref diagnostic for {{nonexistent-key}}")
	}
}

func TestResolvedDoubleCurlyKeyref(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  product-name: \"OpenShift\"\n---\n# Map\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\nInstall {{product-name}} now.\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)

	diags := CheckKeyrefs(topicDoc, f)
	for _, d := range diags {
		if d.Code == CodeUnresolvedKeyref && strings.Contains(d.Message, "product-name") {
			t.Errorf("should not report UnresolvedKeyref for resolved YAML key: %s", d.Message)
		}
	}
}

func TestDoubleCurlyKeyrefInCodeBlockNotDiagnosed(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# Map\n\n- [Install](install.md)\n")
	topicDoc := document.New("file:///project/topic.md", 1,
		"# Topic\n\n```\n{{template-var}}\n```\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)
	f.AddDoc(topicDoc)

	diags := CheckKeyrefs(topicDoc, f)
	for _, d := range diags {
		if d.Code == CodeUnresolvedKeyref && strings.Contains(d.Message, "template-var") {
			t.Error("should not diagnose {{key}} inside code block")
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/diagnostic/ -run "TestUnresolvedDoubleCurly|TestResolvedDoubleCurly|TestDoubleCurly.*Code" -v`
Expected: FAIL — no `import "strings"` may also need adding. The `{{nonexistent-key}}` is not detected.

- [ ] **Step 3: Update `CheckKeyrefs` to scan for `{{key}}`**

In `internal/diagnostic/keyref.go`, add `"strings"` to imports and use the keyref package's `DetectAll` which now handles both syntaxes:

```go
func CheckKeyrefs(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	var mapTexts []string
	for _, d := range folder.AllDocs() {
		if d.Kind == document.Map {
			mapTexts = append(mapTexts, d.Text)
		}
	}
	merged := keyref.BuildMergedTable(mapTexts)
	if len(merged) == 0 {
		return nil
	}

	// [key] shortcut keyrefs (existing)
	lines := strings.Split(doc.Text, "\n")
	for lineNum, line := range lines {
		matches := shortcutRefRe.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			label := line[match[2]:match[3]]
			if label == "" {
				continue
			}
			if doc.Index.LinkDefByLabel(label) != nil {
				continue
			}
			if _, ok := keyref.Resolve(merged, label); !ok {
				diags = append(diags, Diagnostic{
					Range:    document.Rng(lineNum, match[2], lineNum, match[3]),
					Severity: SeverityWarning,
					Code:     CodeUnresolvedKeyref,
					Source:   source,
					Message:  "Unresolved keyref: " + label,
				})
			}
		}
	}

	// {{key}} double-curly keyrefs
	dcLocs := keyref.DetectAllDoubleCurly(doc.Text)
	for _, loc := range dcLocs {
		if _, ok := keyref.Resolve(merged, loc.Key); !ok {
			startChar := loc.EndChar - len(loc.Key) - 4 // account for {{ and }}
			diags = append(diags, Diagnostic{
				Range:    document.Rng(loc.Line, startChar, loc.Line, loc.EndChar),
				Severity: SeverityWarning,
				Code:     CodeUnresolvedKeyref,
				Source:   source,
				Message:  "Unresolved keyref: " + loc.Key,
			})
		}
	}

	return diags
}
```

To support this, add a `DetectAllDoubleCurly` exported function in `internal/keyref/detect.go` that returns only `{{key}}` matches (not `[key]` matches). This avoids diagnostics needing to distinguish between the two types in the combined `DetectAll` output:

```go
func DetectAllDoubleCurly(text string) []KeyrefLocation {
	lines := strings.Split(text, "\n")
	fenced := fencedCodeLines(text)
	var locs []KeyrefLocation
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		matches := doubleCurlyRe.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			matchStart := m[0]
			keyStart := m[2]
			keyEnd := m[3]
			if isInInlineCode(line, matchStart) {
				continue
			}
			locs = append(locs, KeyrefLocation{
				Key:     line[keyStart:keyEnd],
				Line:    i,
				EndChar: m[1],
			})
		}
	}
	return locs
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/diagnostic/ -run "TestUnresolvedDoubleCurly|TestResolvedDoubleCurly|TestDoubleCurly.*Code" -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/keyref/detect.go internal/diagnostic/keyref.go internal/diagnostic/keyref_test.go
git commit -m "feat: diagnose unresolved {{key}} keyrefs"
```

---

### Task 10: `{{key}}` inlay hints

**Files:**
- Modify: `internal/inlayhint/inlayhint.go:65-90`
- Modify: `internal/inlayhint/inlayhint_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/inlayhint/inlayhint_test.go`:

```go
func TestDoubleCurlyKeyrefHint(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  product-name: \"Red Hat OpenShift\"\n---\n# Map\n")
	source := document.New("file:///project/doc.md", 1,
		"# Doc\n\nInstall {{product-name}} now.\n")
	folder := testFolder(mapDoc, source)

	hints := GetHints(source, fullRange(), folder)
	found := false
	for _, h := range hints {
		if h.Label == " → Red Hat OpenShift" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected inlay hint with resolved value, got %v", hints)
	}
}

func TestDoubleCurlyKeyrefHintURL(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"---\nkeys:\n  docs-url: \"https://docs.example.com\"\n---\n# Map\n")
	source := document.New("file:///project/doc.md", 1,
		"# Doc\n\nVisit {{docs-url}} for help.\n")
	folder := testFolder(mapDoc, source)

	hints := GetHints(source, fullRange(), folder)
	found := false
	for _, h := range hints {
		if h.Label == " → https://docs.example.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected inlay hint with href, got %v", hints)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/inlayhint/ -run "TestDoubleCurlyKeyref" -v`
Expected: FAIL — no hint for `{{key}}`.

- [ ] **Step 3: Update `keyrefHints` to handle `{{key}}` and `Value`**

In `internal/inlayhint/inlayhint.go`, update `keyrefHints`:

```go
func keyrefHints(doc *document.Document, rng document.Range, table keyref.KeyTable) []InlayHint {
	defs := keyref.DetectAll(doc.Text)
	var hints []InlayHint
	for _, d := range defs {
		if d.Line < rng.Start.Line || d.Line > rng.End.Line {
			continue
		}
		entry, ok := table[d.Key]
		if !ok {
			continue
		}
		label := entry.Href
		if entry.Value != "" {
			label = entry.Value
		} else if entry.Title != "" {
			label = entry.Title
		}
		hints = append(hints, InlayHint{
			Position: document.Position{
				Line:      d.Line,
				Character: d.EndChar,
			},
			Label: " → " + label,
			Kind:  KindType,
		})
	}
	return hints
}
```

Since `DetectAll` now returns both `[key]` and `{{key}}` matches, this handles both syntaxes automatically.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/aireilly/mdita-lsp && go test ./internal/inlayhint/ -v`
Expected: All inlay hint tests pass.

- [ ] **Step 5: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/inlayhint/inlayhint.go internal/inlayhint/inlayhint_test.go
git commit -m "feat: add inlay hints for {{key}} keyrefs with resolved values"
```

---

### Task 11: Update completion resolve for `Value`

**Files:**
- Modify: `internal/completion/resolve.go:22-35`

- [ ] **Step 1: Update `resolveKeyrefDocs` to show `Value`**

In `internal/completion/resolve.go`, update the function:

```go
func resolveKeyrefDocs(key string, folder *workspace.Folder) string {
	table := keyref.BuildMergedTable(folder.MapTexts())
	entry, ok := table[key]
	if !ok {
		return ""
	}

	var parts []string
	if entry.Value != "" {
		parts = append(parts, "**"+entry.Value+"**")
		parts = append(parts, "type: keyword keydef")
	} else {
		if entry.Title != "" {
			parts = append(parts, "**"+entry.Title+"**")
		}
		if entry.Href != "" {
			parts = append(parts, "href: `"+entry.Href+"`")
		}
	}
	return strings.Join(parts, "\n\n")
}
```

- [ ] **Step 2: Run full test suite**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/completion/resolve.go
git commit -m "feat: show keyword value in completion resolve docs"
```

---

### Task 12: Final lint and integration verification

- [ ] **Step 1: Run linter**

Run: `make lint`
Expected: Zero lint issues.

- [ ] **Step 2: Run full test suite with race detection**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 3: Build binary**

Run: `make build`
Expected: Clean build, no errors.

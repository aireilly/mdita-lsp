# org.lwdita Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor mdita-lsp main branch to align exclusively with org.lwdita's MDITA feature set, removing Red Hat extended profile features and adding conref support and auto-format-tables-on-save.

**Architecture:** Archive current main as `redhat.mdita.extended`, then strip domain elements, YAML keydefs, `{{keyref}}` syntax, and related-links from main. Replace with org.lwdita's reference-style keydefs, conref/conkeyref via `data-conref`/`data-conkeyref` HTML attributes, implicit task section detection, and `$schema`-based profile switching. Wire existing `alignTables` formatter to a `textDocument/willSaveWaitUntil` handler.

**Tech Stack:** Go, goldmark, yaml.v3, LSP 3.17

**Spec:** `docs/superpowers/specs/2026-08-14-org-lwdita-alignment-design.md`

## Global Constraints

- Go 1.21+, only two dependencies (goldmark, yaml.v3)
- All packages under `internal/` — not externally importable
- Binary name: `mdita-lsp`
- Config filename: `.mdita-lsp.yaml`
- `make lint` must pass before every commit
- `make test` must pass (race detector enabled)
- Tests colocated with source (`*_test.go` in each package)

---

### Task 1: Branch Creation

**Files:**
- No file changes — git operations only

**Interfaces:**
- Consumes: nothing
- Produces: `redhat.mdita.extended` branch at current HEAD

- [ ] **Step 1: Create the archival branch**

```bash
git branch redhat.mdita.extended main
```

- [ ] **Step 2: Push the archival branch**

```bash
git push origin redhat.mdita.extended
```

- [ ] **Step 3: Verify branch exists**

```bash
git log --oneline -1 redhat.mdita.extended
# Should show: acebd93 feat: align LSP with DITA-OT plugin...
```

---

### Task 2: Config & Schema Foundation

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/document/types.go`
- Modify: `internal/document/types_test.go` (create if absent)

**Interfaces:**
- Consumes: nothing
- Produces:
  - `config.Profile` type (`Core`, `Extended`) and `Config.Core.Mdita.Profile` field
  - `config.Config.Core.Mdita.FormatTablesOnSave` field (`*bool`, default `true`)
  - Updated `DitaSchema` constants: remove `SchemaMditaTopic`, `SchemaMditaCoreTopic`, `SchemaMditaExtendedTopic`
  - `DitaSchemaFromString` mapping only org.lwdita URNs

- [ ] **Step 1: Write failing test for Profile config parsing**

In `internal/config/config_test.go`, add:

```go
func TestParseProfileConfig(t *testing.T) {
	data := []byte("core:\n  mdita:\n    profile: core\n    formatTablesOnSave: false\n")
	cfg, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Core.Mdita.Profile != ProfileCore {
		t.Errorf("got profile %v, want ProfileCore", cfg.Core.Mdita.Profile)
	}
	if BoolVal(cfg.Core.Mdita.FormatTablesOnSave) {
		t.Error("formatTablesOnSave should be false")
	}
}

func TestDefaultProfileIsExtended(t *testing.T) {
	cfg := Default()
	if cfg.Core.Mdita.Profile != ProfileExtended {
		t.Errorf("default profile should be Extended, got %v", cfg.Core.Mdita.Profile)
	}
	if !BoolVal(cfg.Core.Mdita.FormatTablesOnSave) {
		t.Error("formatTablesOnSave default should be true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/aireilly/mdita-lsp && go test ./internal/config/ -run TestParseProfileConfig -v
```

Expected: compile error — `ProfileCore`, `FormatTablesOnSave` undefined.

- [ ] **Step 3: Add Profile type and config fields**

In `internal/config/config.go`, add the `Profile` type and update `MditaConfig`:

```go
type Profile int

const (
	ProfileExtended Profile = iota
	ProfileCore
)

type MditaConfig struct {
	Enable             *bool    `yaml:"enable"`
	MapExtensions      []string `yaml:"mapExtensions"`
	Profile            Profile  `yaml:"profile"`
	FormatTablesOnSave *bool    `yaml:"formatTablesOnSave"`
}
```

Update `Default()` to set `FormatTablesOnSave` to `boolPtr(true)` and `Profile` to `ProfileExtended`.

Update `Merge()` to merge the new fields:
```go
if overlay.Core.Mdita.Profile != 0 {
	result.Core.Mdita.Profile = overlay.Core.Mdita.Profile
}
if overlay.Core.Mdita.FormatTablesOnSave != nil {
	result.Core.Mdita.FormatTablesOnSave = overlay.Core.Mdita.FormatTablesOnSave
}
```

Add a custom YAML unmarshaler for `Profile`:
```go
func (p *Profile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	switch s {
	case "core":
		*p = ProfileCore
	case "extended", "":
		*p = ProfileExtended
	default:
		*p = ProfileExtended
	}
	return nil
}
```

- [ ] **Step 4: Run config tests**

```bash
go test ./internal/config/ -v
```

Expected: PASS.

- [ ] **Step 5: Write failing test for updated DitaSchemaFromString**

In `internal/document/`, create or update test file:

```go
func TestDitaSchemaFromStringOrgLwdita(t *testing.T) {
	tests := []struct {
		input string
		want  DitaSchema
	}{
		{"urn:oasis:names:tc:dita:xsd:topic.xsd", SchemaTopic},
		{"urn:oasis:names:tc:dita:rng:topic.rng", SchemaTopic},
		{"urn:oasis:names:tc:dita:xsd:task.xsd", SchemaTask},
		{"urn:oasis:names:tc:dita:xsd:concept.xsd", SchemaConcept},
		{"urn:oasis:names:tc:dita:xsd:reference.xsd", SchemaReference},
		{"urn:oasis:names:tc:dita:xsd:map.xsd", SchemaMap},
		// Removed MDITA-specific URNs should return SchemaUnknown
		{"urn:oasis:names:tc:mdita:xsd:topic.xsd", SchemaUnknown},
		{"urn:oasis:names:tc:mdita:core:xsd:topic.xsd", SchemaUnknown},
		{"urn:oasis:names:tc:mdita:extended:xsd:topic.xsd", SchemaUnknown},
	}
	for _, tt := range tests {
		got := DitaSchemaFromString(tt.input)
		if got != tt.want {
			t.Errorf("DitaSchemaFromString(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

```bash
go test ./internal/document/ -run TestDitaSchemaFromStringOrgLwdita -v
```

Expected: FAIL on the MDITA URNs (currently they return `SchemaMditaTopic` etc., not `SchemaUnknown`).

- [ ] **Step 7: Update DitaSchema constants and DitaSchemaFromString**

In `internal/document/types.go`:

Remove `SchemaMditaTopic`, `SchemaMditaCoreTopic`, `SchemaMditaExtendedTopic` from the `DitaSchema` const block. Keep only:

```go
const (
	SchemaTopic DitaSchema = iota
	SchemaConcept
	SchemaTask
	SchemaReference
	SchemaMap
	SchemaUnknown
)
```

Update `DitaSchemaFromString` to remove the MDITA-specific cases (lines 68-76).

- [ ] **Step 8: Run all document tests**

```bash
go test ./internal/document/ -v
```

Expected: PASS. Fix any tests that referenced the removed schema constants.

- [ ] **Step 9: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/document/types.go internal/document/*_test.go
git commit -m "feat: add Profile config, formatTablesOnSave, remove MDITA-specific schemas"
```

---

### Task 3: Strip Vocabulary & Domain Elements

**Files:**
- Delete: `internal/vocabulary/` (entire package)
- Modify: `internal/document/types.go` (remove `InlineAttribute`, `RelatedLinksInfo`)
- Modify: `internal/document/attributes.go` (remove `ScanInlineAttributes`)
- Modify: `internal/document/document.go` (remove `InlineAttrs`, `RelLinks` fields, `resolveRelatedLinks`)
- Modify: `internal/completion/partial.go` (remove `PartialAttrClass`, `PartialDoubleCurlyKeyref`, `PartialAttrOpen`)
- Modify: `internal/completion/completion.go` (remove `completeAttrClass`, `completeBlockAttr`, `completeAttrOpen`, `completeDoubleCurlyKeyref`, vocabulary import, remove `"keys"` from `yamlKeys`)
- Modify: `internal/hover/hover.go` (remove `hoverInlineAttribute`, `hoverBlockAttribute`, `hoverTaskList`, vocabulary import)
- Modify: `internal/inlayhint/inlayhint.go` (remove `domainHints`, vocabulary import)
- Modify: `internal/semantic/semantic.go` (remove `InlineAttrs` iteration from `collectTokens`)
- Modify: `internal/codeaction/codeaction.go` (remove `addRelatedLinksAction`, update `addFrontMatterAction` schema URN)
- Modify: test files in each modified package

**Interfaces:**
- Consumes: updated `DitaSchema` from Task 2
- Produces:
  - `Document` struct without `InlineAttrs` or `RelLinks` fields
  - `PartialKind` enum without domain-specific values
  - All packages compile without `vocabulary` import

- [ ] **Step 1: Delete vocabulary package**

```bash
rm -rf internal/vocabulary/
```

- [ ] **Step 2: Remove InlineAttribute type and RelatedLinksInfo from types.go**

In `internal/document/types.go`, delete:
- `InlineAttribute` struct (lines 169-175)
- `RelatedLinksInfo` struct (lines 193-196)

Keep `BlockAttribute` (lines 177-180) and `ParsedAttribute` (lines 162-167).

- [ ] **Step 3: Remove ScanInlineAttributes from attributes.go**

In `internal/document/attributes.go`, delete:
- `ScanInlineAttributes` function (line 33)
- `scanLineInline` helper (line 45)
- `inlineBoldAttrRegex`, `inlineBoldUnderAttrRegex`, `inlineCodeAttrRegex`, `inlineItalicAttrRegex` regex vars (lines 9-14)

Keep `ParseAttrString`, `ScanBlockAttributes`, `blockAttrRegex`, `attrClassRegex`, `attrIDRegex`, `attrKVRegex`.

- [ ] **Step 4: Update Document struct and New() in document.go**

In `internal/document/document.go`:

Remove `InlineAttrs` and `RelLinks` fields from the `Document` struct (lines 19, 21).

In `New()` (line 24), remove:
- `inlineAttrs := ScanInlineAttributes(text)` (line 39)
- `InlineAttrs: inlineAttrs,` from the struct literal (line 54)
- `RelLinks` field from struct literal
- `resolveRelatedLinks(doc)` call (line 59)
- The `if len(inlineAttrs) > 0 || ...` block — update to just check `blockAttrs`:
```go
if len(blockAttrs) > 0 {
    bf.HasAttributes = true
}
```

Delete the `resolveRelatedLinks` function (lines 274-306).

- [ ] **Step 5: Strip vocabulary imports from completion package**

In `internal/completion/partial.go`:
- Remove `PartialAttrClass`, `PartialBlockAttr`, `PartialAttrOpen`, `PartialDoubleCurlyKeyref` from the `PartialKind` const block (lines 17-21)
- Update `DetectPartial` to remove the code paths that return these kinds

In `internal/completion/completion.go`:
- Remove `vocabulary` import
- Remove `"keys"` from the `yamlKeys` slice (line 30)
- Remove `completeAttrClass` function (line 224)
- Remove `completeBlockAttr` function (line 296)
- Remove `completeAttrOpen` function (line 314)
- Remove `completeDoubleCurlyKeyref` function (line 402)
- Update `Complete` to remove the `case` branches for removed partial kinds

- [ ] **Step 6: Strip vocabulary imports from hover package**

In `internal/hover/hover.go`:
- Remove `vocabulary` import
- Remove `hoverInlineAttribute` function (line 238)
- Remove `hoverBlockAttribute` function (line 312)
- Remove `hoverTaskList` function (line 266) and its regex vars (lines 263-264)
- Keep `hoverHeadingClass` (line 225) — it shows topic type for `{.task}`, `{.concept}`, `{.reference}` which are valid org.lwdita
- Update `GetHover` (line 15) to remove calls to the deleted functions but keep the `hoverHeadingClass` call

- [ ] **Step 7: Strip vocabulary imports from inlayhint package**

In `internal/inlayhint/inlayhint.go`:
- Remove `vocabulary` import
- Remove `domainHints` function (line 94)
- Update `GetHints` (line 21) to remove the `domainHints` call

- [ ] **Step 8: Strip InlineAttrs from semantic package**

In `internal/semantic/semantic.go`:
- In `collectTokens` (line 38), remove the loop over `doc.InlineAttrs`
- Keep the loop over `doc.BlockAttrs` for conditional processing attribute tokens

- [ ] **Step 9: Update codeaction package**

In `internal/codeaction/codeaction.go`:
- Remove `addRelatedLinksAction` function (line 253)
- Update `GetActions` (line 41) to remove the `addRelatedLinksAction` call
- In `addFrontMatterAction` (line 68), change the schema URN from `"urn:oasis:names:tc:mdita:rng:topic.rng"` to `"urn:oasis:names:tc:dita:xsd:topic.xsd"`

- [ ] **Step 10: Update and remove tests**

For each modified package, update `*_test.go` files:
- Remove tests that reference `InlineAttribute`, domain elements, vocabulary lookups, `{{keyref}}` completions, domain hover, related links
- Keep tests for standard functionality (links, headings, keyrefs via `[key]` syntax, block attributes)

- [ ] **Step 11: Verify compilation and tests**

```bash
go build ./...
go test ./... -count=1
```

Fix any remaining compilation errors from removed types or functions.

- [ ] **Step 12: Run lint**

```bash
make lint
```

- [ ] **Step 13: Commit**

```bash
git add -A
git commit -m "refactor: strip vocabulary package, domain elements, and related links"
```

---

### Task 4: Strip Remaining Red Hat Extensions

**Files:**
- Modify: `internal/keyref/keyref.go` (remove YAML keydefs extraction)
- Modify: `internal/keyref/detect.go` (remove `{{keyref}}` detection)
- Modify: `internal/diagnostic/mdita.go` (remove domain-specific checks)
- Modify: `internal/diagnostic/diagnostic.go` (remove codes 8-12, 15, 20-29)
- Modify: `internal/diagnostic/keyref.go` (remove `{{keyref}}` detection)
- Modify: `internal/diagnostic/links.go` (remove `{{keyref}}` URL skip)
- Modify: `internal/document/parser.go` (remove `keys:` YAML parsing)
- Modify: `internal/document/types.go` (remove `Keys` field from `YAMLMetadata`)
- Modify: `internal/definition/definition.go` (remove YAML keydef resolution path)
- Modify: test files in each modified package

**Interfaces:**
- Consumes: cleaned document types from Task 3
- Produces:
  - `keyref.DetectAll` only detects `[key]` reference links (no `{{key}}`)
  - `keyref.BuildMergedTable` builds from LinkDefs only (no YAML keys)
  - `YAMLMetadata` without `Keys` field
  - Diagnostic codes 1-7 + footnote codes only (pre-renumber)

- [ ] **Step 1: Remove {{keyref}} detection from keyref/detect.go**

Delete:
- `doubleCurlyRe` regex var (line 11)
- `DetectAllDoubleCurly` function (line 126)
- `IsDoubleCurly` field handling in `DetectAll` (any branches that set it)

In `DetectAtPosition` (line 71), remove the `{{key}}` matching branch.

- [ ] **Step 2: Remove YAML keydefs from keyref/keyref.go**

In `BuildMergedTable` (line 69), remove the code path that reads from `meta.Keys`.

In `ExtractKeys` (line 19), remove the code path that reads from `m.Meta.Keys` (if it reads map metadata for YAML keys).

- [ ] **Step 3: Remove Keys from YAMLMetadata and parser**

In `internal/document/types.go`, remove `Keys map[string]string` from `YAMLMetadata` (line 134).

In `internal/document/parser.go`, remove the `case "keys":` block from `parseYAMLMeta` (lines 216-224).

- [ ] **Step 4: Remove YAML keydef resolution from definition.go**

In `internal/definition/definition.go`, simplify `resolveKeyref` (line 63) to remove the branch that navigates to a map's `keys:` YAML block. Remove `findYAMLKeyLine` helper (line 107).

The keyref definition should resolve to the `[key]: url` LinkDef line in the map file.

- [ ] **Step 5: Strip domain-specific diagnostics**

In `internal/diagnostic/diagnostic.go`, remove code constants 8-12, 15, 20-29. Keep codes 1-7, 13-14, 16-19.

In `internal/diagnostic/mdita.go`, remove calls to:
- `checkSchemaSpecific` (extended-in-core checks)
- `checkExtendedFeatures`
- `checkAdmonitions` (admonition type validation)
- `checkInlineAttributes`
- `checkBlockAttributes` (conditional attr validation)
- `checkStepElements`
- `checkRelatedLinks`
- `checkTaskSections` (task section order/duplication)

Keep: `checkHeadingHierarchy`, `checkFootnotes`.

In `internal/diagnostic/keyref.go`, remove `{{keyref}}` pattern matching.

In `internal/diagnostic/links.go`, remove the `{{...}}` URL skip pattern (line 28).

- [ ] **Step 6: Update tests**

Remove tests for `{{keyref}}` detection, YAML keydefs, domain diagnostics. Update remaining keyref tests to use `[key]` syntax only.

- [ ] **Step 7: Verify**

```bash
go build ./...
go test ./... -count=1
make lint
```

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor: strip {{keyref}}, YAML keydefs, and domain-specific diagnostics"
```

---

### Task 5: Keyref Model Refactor

**Files:**
- Modify: `internal/keyref/keyref.go` (reference-style keydef extraction, data-keyref detection)
- Modify: `internal/keyref/detect.go` (add `[text][key]` and `<span data-keyref>` detection)
- Create: `internal/keyref/htmlkeyref.go` (data-keyref HTML parsing)
- Modify: `internal/keyref/keyref_test.go`
- Modify: `internal/completion/completion.go` (keyref completion for new syntax)
- Modify: `internal/hover/hover.go` (data-keyref hover)
- Modify: `internal/definition/definition.go` (keyref definition via LinkDef)
- Modify: `internal/inlayhint/inlayhint.go` (data-keyref hints)

**Interfaces:**
- Consumes: `keyref.KeyTable`, `document.LinkDef` elements in map files
- Produces:
  - `keyref.ExtractKeys(m *ditamap.MapStructure) KeyTable` — builds from LinkDefs in map
  - `keyref.DetectAll(text string) []KeyrefLocation` — detects `[key]`, `[text][key]`, `<span data-keyref="key">`
  - `keyref.DetectAtPosition(text string, pos document.Position) *KeyrefAtPos` — cursor in any keyref form
  - `keyref.DetectDataKeyrefs(text string) []DataKeyref` — HTML keyref positions

- [ ] **Step 1: Write failing test for [text][key] detection**

In `internal/keyref/detect_test.go`:

```go
func TestDetectFullRefLink(t *testing.T) {
	text := "See [the guide][install-guide] for details."
	locs := DetectAll(text)
	if len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
	if locs[0].Key != "install-guide" {
		t.Errorf("got key %q, want %q", locs[0].Key, "install-guide")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/keyref/ -run TestDetectFullRefLink -v
```

Expected: FAIL — current regex only matches `[key]` not followed by `(` or `[`.

- [ ] **Step 3: Add [text][key] regex to detect.go**

Add a new regex for full reference links:

```go
var fullRefLinkRe = regexp.MustCompile(`\[[^\]]*\]\[([^\]]+)\]`)
```

Update `DetectAll` to scan with both `shortcutRefRe` and `fullRefLinkRe`, deduplicating by position.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/keyref/ -run TestDetectFullRefLink -v
```

- [ ] **Step 5: Write failing test for data-keyref detection**

```go
func TestDetectDataKeyref(t *testing.T) {
	text := `The <span data-keyref="product-name">Product</span> is ready.`
	locs := DetectAll(text)
	if len(locs) != 1 {
		t.Fatalf("got %d locations, want 1", len(locs))
	}
	if locs[0].Key != "product-name" {
		t.Errorf("got key %q, want %q", locs[0].Key, "product-name")
	}
}
```

- [ ] **Step 6: Create htmlkeyref.go with data-keyref parsing**

Create `internal/keyref/htmlkeyref.go`:

```go
package keyref

import (
	"regexp"

	"github.com/aireilly/mdita-lsp/internal/document"
)

var dataKeyrefRe = regexp.MustCompile(`<\w+\s[^>]*data-keyref="([^"]+)"[^>]*>`)

type DataKeyref struct {
	Key   string
	Range document.Range
}

func DetectDataKeyrefs(text string) []DataKeyref {
	var results []DataKeyref
	lines := splitLines(text)
	for lineNum, line := range lines {
		for _, m := range dataKeyrefRe.FindAllStringSubmatchIndex(line, -1) {
			key := line[m[2]:m[3]]
			results = append(results, DataKeyref{
				Key: key,
				Range: document.Rng(lineNum, m[0], lineNum, m[1]),
			})
		}
	}
	return results
}
```

Update `DetectAll` to also scan for data-keyrefs and merge results.

- [ ] **Step 7: Run all keyref tests**

```bash
go test ./internal/keyref/ -v
```

- [ ] **Step 8: Update DetectAtPosition for new syntax**

Update `DetectAtPosition` in `detect.go` to handle cursor inside:
- `[text][key]` — return key when cursor is in the `[key]` portion
- `<span data-keyref="key">` — return key when cursor is in the attribute value

```go
func DetectAtPosition(text string, pos document.Position) *KeyrefAtPos {
	lines := splitLines(text)
	if pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	col := pos.Character

	// Check [text][key] pattern
	for _, m := range fullRefLinkRe.FindAllStringSubmatchIndex(line, -1) {
		keyStart, keyEnd := m[2], m[3]
		if col >= keyStart && col <= keyEnd {
			key := line[keyStart:keyEnd]
			return &KeyrefAtPos{
				Key:   key,
				Range: document.Rng(pos.Line, m[0], pos.Line, m[1]),
			}
		}
	}

	// Check [key] shortcut
	for _, m := range shortcutRefRe.FindAllStringSubmatchIndex(line, -1) {
		keyStart, keyEnd := m[2], m[3]
		if col >= keyStart && col <= keyEnd {
			key := line[keyStart:keyEnd]
			return &KeyrefAtPos{
				Key:   key,
				Range: document.Rng(pos.Line, m[0], pos.Line, m[1]),
			}
		}
	}

	// Check data-keyref="key"
	for _, m := range dataKeyrefRe.FindAllStringSubmatchIndex(line, -1) {
		keyStart, keyEnd := m[2], m[3]
		if col >= keyStart && col <= keyEnd {
			key := line[keyStart:keyEnd]
			return &KeyrefAtPos{
				Key:   key,
				Range: document.Rng(pos.Line, m[0], pos.Line, m[1]),
			}
		}
	}

	return nil
}
```

- [ ] **Step 9: Update keyref extraction to use LinkDefs only**

In `keyref.go`, ensure `ExtractKeys` builds the `KeyTable` from `m.LinkDefs` (reference-style link definitions in map files). Each `[key]: url "title"` in a map becomes a `KeyEntry{Key: key, Href: url, Title: title}`.

Remove any code path that reads from YAML `meta.Keys` (already removed in Task 4, verify it's gone).

- [ ] **Step 10: Update definition.go for keyref → LinkDef navigation**

In `internal/definition/definition.go`, update `resolveKeyref` to find the `[key]: url` LinkDef line in the map file and return its location. The `LinkDef` element already has the correct `Range`.

```go
func resolveKeyref(kr *keyref.KeyrefAtPos, doc *document.Document, folder *workspace.Folder) []Location {
	for _, mapDoc := range folder.MapDocs() {
		for _, el := range mapDoc.Elements {
			if ld, ok := el.(*document.LinkDef); ok && ld.Label == kr.Key {
				return []Location{{URI: mapDoc.URI, Range: ld.Range}}
			}
		}
	}
	return nil
}
```

- [ ] **Step 11: Update completion for data-keyref**

In `internal/completion/completion.go`, add a completion context for `data-keyref="` that completes with available keys from the keyspace.

Add a new `PartialDataKeyref` kind to `partial.go` and detection in `DetectPartial` for cursor inside `data-keyref="..."`.

- [ ] **Step 12: Update inlayhint for data-keyref**

In `internal/inlayhint/inlayhint.go`, update `keyrefHints` to also show hints for `<span data-keyref="key">` elements, displaying the resolved href.

- [ ] **Step 13: Run all tests**

```bash
go test ./... -count=1
make lint
```

- [ ] **Step 14: Commit**

```bash
git add -A
git commit -m "feat: refactor keyrefs to org.lwdita reference-style model with data-keyref support"
```

---

### Task 6: Conref/Conkeyref System

**Files:**
- Create: `internal/conref/conref.go`
- Create: `internal/conref/conref_test.go`
- Modify: `internal/document/types.go` (add `ConrefElement` type, `DefElementID` DefType)
- Modify: `internal/document/document.go` (add `Conrefs` field, `ElementIDs` index)
- Modify: `internal/document/parser.go` (scan RawHTML for data-conref)
- Modify: `internal/definition/definition.go` (conref go-to-def)
- Modify: `internal/hover/hover.go` (conref hover)
- Modify: `internal/completion/completion.go` (conref completion)
- Modify: `internal/completion/partial.go` (add `PartialConref`)
- Modify: `internal/diagnostic/diagnostic.go` (conref diagnostic codes)
- Modify: `internal/inlayhint/inlayhint.go` (conref hints)

**Interfaces:**
- Consumes: `document.Document`, `keyref.KeyTable` (for conkeyref resolution)
- Produces:
  - `document.ConrefElement` struct implementing `Element`: `{Tag, FilePath, TopicID, ElementID string; IsKeyref bool; KeyName string; Range document.Range}`
  - `conref.Parse(text string) []*document.ConrefElement` — extracts data-conref/data-conkeyref from HTML, returns document elements directly
  - `conref.Resolve(ce *document.ConrefElement, folder *workspace.Folder, keyTable keyref.KeyTable) (string, error)` — returns resolved file URI

- [ ] **Step 1: Write failing test for conref parsing**

Create `internal/conref/conref_test.go`:

```go
package conref

import (
	"testing"
)

func TestParseConref(t *testing.T) {
	text := `<p data-conref="shared.md#topic/warning-para">fallback</p>`
	refs := Parse(text)
	if len(refs) != 1 {
		t.Fatalf("got %d conrefs, want 1", len(refs))
	}
	if refs[0].FilePath != "shared.md" {
		t.Errorf("FilePath = %q, want %q", refs[0].FilePath, "shared.md")
	}
	if refs[0].TopicID != "topic" {
		t.Errorf("TopicID = %q, want %q", refs[0].TopicID, "topic")
	}
	if refs[0].ElementID != "warning-para" {
		t.Errorf("ElementID = %q, want %q", refs[0].ElementID, "warning-para")
	}
	if refs[0].IsKeyref {
		t.Error("should not be a keyref")
	}
}

func TestParseConkeyref(t *testing.T) {
	text := `<span data-conkeyref="warnings/disk-full">fallback</span>`
	refs := Parse(text)
	if len(refs) != 1 {
		t.Fatalf("got %d conrefs, want 1", len(refs))
	}
	if refs[0].KeyName != "warnings" {
		t.Errorf("KeyName = %q, want %q", refs[0].KeyName, "warnings")
	}
	if refs[0].ElementID != "disk-full" {
		t.Errorf("ElementID = %q, want %q", refs[0].ElementID, "disk-full")
	}
	if !refs[0].IsKeyref {
		t.Error("should be a keyref")
	}
}

func TestParseNoConref(t *testing.T) {
	text := `<p>Regular paragraph with no conref.</p>`
	refs := Parse(text)
	if len(refs) != 0 {
		t.Errorf("got %d conrefs, want 0", len(refs))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/conref/ -v
```

Expected: compile error — package doesn't exist.

- [ ] **Step 3: Create conref package**

Create `internal/conref/conref.go`:

```go
package conref

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

var conrefRe = regexp.MustCompile(`<(\w+)\s[^>]*data-conref="([^"]+)"[^>]*>`)
var conkeyrefRe = regexp.MustCompile(`<(\w+)\s[^>]*data-conkeyref="([^"]+)"[^>]*>`)

func Parse(text string) []*document.ConrefElement {
	var refs []*document.ConrefElement
	lines := strings.Split(text, "\n")

	for lineNum, line := range lines {
		for _, m := range conrefRe.FindAllStringSubmatchIndex(line, -1) {
			value := line[m[4]:m[5]]
			ce := parseConrefValue(value)
			ce.Tag = line[m[2]:m[3]]
			ce.Range = document.Rng(lineNum, m[0], lineNum, m[1])
			refs = append(refs, ce)
		}

		for _, m := range conkeyrefRe.FindAllStringSubmatchIndex(line, -1) {
			value := line[m[4]:m[5]]
			ce := parseConkeyrefValue(value)
			ce.Tag = line[m[2]:m[3]]
			ce.IsKeyref = true
			ce.Range = document.Rng(lineNum, m[0], lineNum, m[1])
			refs = append(refs, ce)
		}
	}
	return refs
}

func parseConrefValue(value string) *document.ConrefElement {
	ce := &document.ConrefElement{}
	hashIdx := strings.Index(value, "#")
	if hashIdx < 0 {
		ce.FilePath = value
		return ce
	}
	ce.FilePath = value[:hashIdx]
	rest := value[hashIdx+1:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		ce.TopicID = rest
		return ce
	}
	ce.TopicID = rest[:slashIdx]
	ce.ElementID = rest[slashIdx+1:]
	return ce
}

func parseConkeyrefValue(value string) *document.ConrefElement {
	ce := &document.ConrefElement{}
	slashIdx := strings.Index(value, "/")
	if slashIdx < 0 {
		ce.KeyName = value
		return ce
	}
	ce.KeyName = value[:slashIdx]
	ce.ElementID = value[slashIdx+1:]
	return ce
}
```

- [ ] **Step 4: Run conref tests**

```bash
go test ./internal/conref/ -v
```

Expected: PASS.

- [ ] **Step 5: Add ConrefElement to document types**

In `internal/document/types.go`, add:

```go
type ConrefElement struct {
	Tag       string
	FilePath  string
	TopicID   string
	ElementID string
	IsKeyref  bool
	KeyName   string
	Range     Range
}

func (c *ConrefElement) Rng() Range { return c.Range }
func (c *ConrefElement) element()   {}
```

Add `DefElementID` to the `DefType` enum:

```go
const (
	DefDoc DefType = iota
	DefTitle
	DefHeading
	DefLinkDef
	DefElementID
)
```

- [ ] **Step 6: Add conref scanning to document parser**

In `internal/document/parser.go`, add a `conref` import and call `conref.Parse` from `Parse()`. After the goldmark walk (after line 171), add:

```go
for _, ce := range conref.Parse(source) {
	elements = append(elements, ce)
}
```

This uses the `conref.Parse` function from Step 3 which returns `[]*document.ConrefElement` — each implements `Element`, so they integrate directly into the elements slice.

- [ ] **Step 7: Index element IDs for conref targets**

In `internal/document/document.go`, add element ID extraction to `extractSymbols`:

```go
case *Heading:
	// existing heading symbol code...
	// Add element ID as a def symbol if heading has an explicit ID from {#id}
	if el.ID != "" && el.Attributes != nil && el.Attributes.ID != "" {
		syms = append(syms, Symbol{
			Kind:    DefKind,
			DefType: DefElementID,
			Name:    el.ID,
			DocURI:  doc.URI,
			Range:   el.Range,
		})
	}
```

- [ ] **Step 8: Write failing test for conref go-to-definition**

In `internal/definition/definition_test.go`:

```go
func TestConrefGotoDef(t *testing.T) {
	// Source doc with a conref element
	srcText := "---\n$schema: urn:oasis:names:tc:dita:xsd:topic.xsd\n---\n# Source\n\n<p data-conref=\"target.md#topic/warning\">fallback</p>\n"
	srcDoc := document.New("file:///src.md", 1, srcText)

	// Position cursor on the conref element (line 5)
	pos := document.Position{Line: 5, Character: 10}
	el := srcDoc.ElementAt(pos)
	if el == nil {
		t.Fatal("no element at conref position")
	}
	ce, ok := el.(*document.ConrefElement)
	if !ok {
		t.Fatalf("expected ConrefElement, got %T", el)
	}
	if ce.FilePath != "target.md" {
		t.Errorf("FilePath = %q, want %q", ce.FilePath, "target.md")
	}
}
```

- [ ] **Step 9: Implement conref go-to-definition**

In `internal/definition/definition.go`, update `GotoDef` to handle `ConrefElement`:

```go
case *document.ConrefElement:
	return resolveConref(el, doc, folder)
```

```go
func resolveConref(ce *document.ConrefElement, doc *document.Document, folder *workspace.Folder) []Location {
	if ce.IsKeyref {
		return resolveConkeyref(ce, folder)
	}
	targetURI := folder.ResolveRelativePath(doc.URI, ce.FilePath)
	targetDoc := folder.GetDoc(targetURI)
	if targetDoc == nil {
		return nil
	}
	if ce.ElementID == "" {
		return []Location{{URI: targetURI, Range: document.Rng(0, 0, 0, 0)}}
	}
	for _, el := range targetDoc.Elements {
		if h, ok := el.(*document.Heading); ok && h.ID == ce.ElementID {
			return []Location{{URI: targetURI, Range: h.Range}}
		}
	}
	return nil
}
```

- [ ] **Step 10: Implement conref hover**

In `internal/hover/hover.go`, add conref hover to `GetHover`:

```go
case *document.ConrefElement:
	return hoverConref(el, doc, folder)
```

```go
func hoverConref(ce *document.ConrefElement, doc *document.Document, folder *workspace.Folder) string {
	if ce.IsKeyref {
		return fmt.Sprintf("**conkeyref** `%s/%s`", ce.KeyName, ce.ElementID)
	}
	target := ce.FilePath
	if ce.TopicID != "" {
		target += "#" + ce.TopicID
	}
	if ce.ElementID != "" {
		target += "/" + ce.ElementID
	}
	return fmt.Sprintf("**conref** `%s`", target)
}
```

- [ ] **Step 11: Implement conref diagnostics**

In `internal/diagnostic/diagnostic.go`, add new codes:

```go
const (
	CodeConrefTargetMissing   = "14"
	CodeConrefElementMissing  = "15"
	CodeConkeyrefKeyMissing   = "16"
	CodeConkeyrefElementMissing = "17"
)
```

Create `internal/diagnostic/conref.go`:

```go
func CheckConrefs(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic
	for _, el := range doc.Elements {
		ce, ok := el.(*document.ConrefElement)
		if !ok {
			continue
		}
		if ce.IsKeyref {
			diags = append(diags, checkConkeyref(ce, folder)...)
		} else {
			diags = append(diags, checkConref(ce, doc, folder)...)
		}
	}
	return diags
}
```

Wire `CheckConrefs` into the `Check` function.

- [ ] **Step 12: Add conref completion**

In `internal/completion/partial.go`, add `PartialConref` kind and detection for cursor inside `data-conref="..."` or `data-conkeyref="..."`.

In `internal/completion/completion.go`, add `completeConref` that offers file paths after `data-conref="`, topic/element IDs after `#`/`/`, and key names for `data-conkeyref="`.

- [ ] **Step 13: Add conref inlay hints**

In `internal/inlayhint/inlayhint.go`, add `conrefHints` that shows the resolved target path for conref elements.

- [ ] **Step 14: Run all tests**

```bash
go test ./... -count=1
make lint
```

- [ ] **Step 15: Commit**

```bash
git add -A
git commit -m "feat: add conref/conkeyref support with definition, hover, diagnostics, completion"
```

---

### Task 7: Task Section Alignment

**Files:**
- Modify: `internal/document/document.go` (implicit context/result detection)
- Modify: `internal/document/types.go` (add `ImplicitSection` type)
- Modify: `internal/hover/hover.go` (implicit section hover)
- Modify: `internal/document/document_test.go`

**Interfaces:**
- Consumes: `document.Document` with `BlockFeatures` (HasOrderedList, HasUnorderedList, HasTable)
- Produces:
  - `Document.ImplicitSections []ImplicitSection` — detected implicit context/result/choices/choicetable regions
  - Updated `hoverTaskSection` showing implicit section info

- [ ] **Step 1: Write failing test for implicit context detection**

In `internal/document/document_test.go`:

```go
func TestImplicitContextSection(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\n---\n# Install the app\n\nSome context paragraph.\n\n1. Step one\n2. Step two\n"
	doc := New("file:///test.md", 1, text)
	found := false
	for _, s := range doc.ImplicitSections {
		if s.Kind == ImplicitContext {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected implicit context section before steps")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/document/ -run TestImplicitContextSection -v
```

Expected: compile error — `ImplicitSections`, `ImplicitContext` undefined.

- [ ] **Step 3: Add ImplicitSection types**

In `internal/document/types.go`:

```go
type ImplicitSectionKind int

const (
	ImplicitContext ImplicitSectionKind = iota
	ImplicitResult
	ImplicitChoices
	ImplicitSubsteps
	ImplicitChoicetable
)

type ImplicitSection struct {
	Kind  ImplicitSectionKind
	Range Range
}
```

Add `ImplicitSections []ImplicitSection` field to the `Document` struct.

- [ ] **Step 4: Implement implicit section detection**

In `internal/document/document.go`, add `resolveImplicitSections(doc *Document)` called from `New()` after `resolveTaskSections`:

```go
func resolveImplicitSections(doc *Document) {
	isTask := doc.Meta != nil && doc.Meta.Schema == SchemaTask
	if !isTask {
		for _, e := range doc.Elements {
			if h, ok := e.(*Heading); ok && h.IsTitle() && h.Attributes != nil {
				for _, c := range h.Attributes.Classes {
					if c == "task" {
						isTask = true
					}
				}
			}
		}
	}
	if !isTask {
		return
	}

	lines := strings.Split(doc.Text, "\n")
	titleLine := -1
	firstListLine := -1

	for _, e := range doc.Elements {
		if h, ok := e.(*Heading); ok && h.IsTitle() {
			titleLine = h.Range.Start.Line
		}
	}

	// Find first ordered/unordered list at body level
	for i := titleLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "1.") || strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			firstListLine = i
			break
		}
	}

	if firstListLine > 0 && titleLine >= 0 {
		// Content between title and first list is implicit context
		contextStart := titleLine + 1
		hasContent := false
		for i := contextStart; i < firstListLine; i++ {
			if strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "#") {
				hasContent = true
				break
			}
		}
		if hasContent {
			doc.ImplicitSections = append(doc.ImplicitSections, ImplicitSection{
				Kind:  ImplicitContext,
				Range: Rng(contextStart, 0, firstListLine-1, len(lines[firstListLine-1])),
			})
		}
	}
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/document/ -run TestImplicitContextSection -v
```

Expected: PASS.

- [ ] **Step 6: Write and implement implicit result detection**

Add a test for content after the last list being detected as implicit result. Implement in `resolveImplicitSections` by scanning for content after the last `ol`/`ul` block.

- [ ] **Step 7: Write and implement implicit choicetable detection**

Add a test: a table inside a list item in a task topic is detected as `ImplicitChoicetable`. Implement by scanning the goldmark AST for Table nodes nested inside ListItem nodes.

- [ ] **Step 8: Write and implement implicit choices/substeps detection**

Add tests: nested `ul` in a step is `ImplicitChoices`, nested `ol` is `ImplicitSubsteps`. These build on the AST walk in the parser.

- [ ] **Step 9: Update hover for implicit sections**

In `internal/hover/hover.go`, update `GetHover` to check if cursor position falls within an `ImplicitSection` range and return appropriate hover text:

```go
for _, s := range doc.ImplicitSections {
	if posInRange(pos, s.Range) {
		switch s.Kind {
		case document.ImplicitContext:
			return "**Implicit `<context>`** — content before steps wraps in `<context>` element"
		case document.ImplicitResult:
			return "**Implicit `<result>`** — content after steps wraps in `<result>` element"
		}
	}
}
```

- [ ] **Step 10: Run all tests**

```bash
go test ./... -count=1
make lint
```

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "feat: add implicit task section detection (context, result, choices, choicetable)"
```

---

### Task 8: Auto-format Tables on Save

**Files:**
- Modify: `internal/lsp/server.go` (add `TextDocumentSyncOptions`, `willSaveWaitUntil` capability)
- Modify: `internal/lsp/handler.go` (add `textDocument/willSaveWaitUntil` dispatch)
- Create: `internal/lsp/willsave.go` (handler implementation)
- Modify: `internal/formatting/formatting.go` (export `AlignTables`)
- Modify: `internal/formatting/formatting_test.go`
- Modify: `internal/config/config.go` (already done in Task 2)

**Interfaces:**
- Consumes: `formatting.AlignTables(text string) []formatting.TextEdit`, `config.BoolVal(cfg.Core.Mdita.FormatTablesOnSave)`
- Produces:
  - `textDocument/willSaveWaitUntil` handler returning `[]TextEdit` for table alignment
  - `TextDocumentSyncOptions` struct replacing the bare `int` in `ServerCapabilities`

- [ ] **Step 1: Write failing test for exported AlignTables**

In `internal/formatting/formatting_test.go`:

```go
func TestAlignTablesExported(t *testing.T) {
	text := "| a | bb |\n|---|---|\n| ccc | d |\n"
	edits := AlignTables(text)
	if len(edits) == 0 {
		t.Error("expected table alignment edits")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/formatting/ -run TestAlignTablesExported -v
```

Expected: compile error — `AlignTables` is unexported (currently `alignTables`).

- [ ] **Step 3: Export AlignTables function**

In `internal/formatting/formatting.go`, rename `alignTables` (line 112) to `AlignTables` and update the call in `Format` (line 20).

Add a wrapper that takes text instead of `[]string`:

```go
func AlignTables(text string) []TextEdit {
	lines := strings.Split(text, "\n")
	return alignTableLines(lines)
}
```

Rename the existing `alignTables(lines []string)` to `alignTableLines`.

- [ ] **Step 4: Run formatting tests**

```bash
go test ./internal/formatting/ -v
```

- [ ] **Step 5: Add TextDocumentSyncOptions struct**

In `internal/lsp/server.go`, replace the bare `int` in `ServerCapabilities`:

```go
type TextDocumentSyncOptions struct {
	OpenClose         bool `json:"openClose"`
	Change            int  `json:"change"`
	WillSaveWaitUntil bool `json:"willSaveWaitUntil"`
}

type ServerCapabilities struct {
	TextDocumentSync TextDocumentSyncOptions `json:"textDocumentSync"`
	// ... rest unchanged
}
```

In `handleInitialize`, update the capability registration:

```go
TextDocumentSync: TextDocumentSyncOptions{
	OpenClose:         true,
	Change:            2,
	WillSaveWaitUntil: true,
},
```

- [ ] **Step 6: Create willsave.go handler**

Create `internal/lsp/willsave.go`:

```go
package lsp

import (
	"context"
	"encoding/json"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/formatting"
)

type WillSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Reason       int                    `json:"reason"`
}

func (s *Server) handleWillSaveWaitUntil(ctx context.Context, params json.RawMessage) (any, error) {
	var p WillSaveTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	uri := p.TextDocument.URI
	folder := s.workspace.FolderForURI(uri)
	if folder == nil {
		return []any{}, nil
	}

	cfg := folder.Config()
	if !config.BoolVal(cfg.Core.Mdita.FormatTablesOnSave) {
		return []any{}, nil
	}

	doc := folder.GetDoc(uri)
	if doc == nil {
		return []any{}, nil
	}

	edits := formatting.AlignTables(doc.Text)
	if len(edits) == 0 {
		return []any{}, nil
	}

	var lspEdits []map[string]any
	for _, e := range edits {
		lspEdits = append(lspEdits, map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": e.Range.Start.Line, "character": e.Range.Start.Character},
				"end":   map[string]any{"line": e.Range.End.Line, "character": e.Range.End.Character},
			},
			"newText": e.NewText,
		})
	}
	return lspEdits, nil
}
```

- [ ] **Step 7: Add dispatch for willSaveWaitUntil**

In `internal/lsp/handler.go`, add to the `dispatch` function (in the `switch method` block):

```go
case "textDocument/willSaveWaitUntil":
	return s.handleWillSaveWaitUntil(ctx, params)
```

- [ ] **Step 8: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 9: Write integration test**

Test that the server returns table alignment edits for a `textDocument/willSaveWaitUntil` request with a document containing a misaligned table.

- [ ] **Step 10: Run all tests**

```bash
go test ./... -count=1
make lint
```

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "feat: auto-format markdown tables on save via willSaveWaitUntil"
```

---

### Task 9: Diagnostic Refactor

**Files:**
- Modify: `internal/diagnostic/diagnostic.go` (renumber codes, add new codes)
- Modify: `internal/diagnostic/mdita.go` (profile-aware diagnostics)
- Create: `internal/diagnostic/profile.go` (core profile feature warnings)
- Modify: `internal/diagnostic/diagnostic_test.go`
- Modify: `internal/lsp/server.go` (publish diagnostic code changes)

**Interfaces:**
- Consumes: `config.Profile`, `document.Document`, conref elements
- Produces:
  - Renumbered diagnostic codes 1-18
  - `CheckProfile(doc, cfg)` — warns on extended features in core profile documents
  - `CheckTaskType(doc)` — warns on task features in non-task topics

- [ ] **Step 1: Write failing test for renumbered diagnostics**

In `internal/diagnostic/diagnostic_test.go`:

```go
func TestDiagnosticCodeValues(t *testing.T) {
	if CodeAmbiguousLink != "1" {
		t.Errorf("CodeAmbiguousLink = %q, want %q", CodeAmbiguousLink, "1")
	}
	if CodeFootnoteRefOrphan != "8" {
		t.Errorf("CodeFootnoteRefOrphan = %q, want %q", CodeFootnoteRefOrphan, "8")
	}
	if CodeUnresolvedKeyref != "10" {
		t.Errorf("CodeUnresolvedKeyref = %q, want %q", CodeUnresolvedKeyref, "10")
	}
	if CodeCoreProfileFeature != "13" {
		t.Errorf("CodeCoreProfileFeature = %q, want %q", CodeCoreProfileFeature, "13")
	}
	if CodeTaskTypeInNonTask != "18" {
		t.Errorf("CodeTaskTypeInNonTask = %q, want %q", CodeTaskTypeInNonTask, "18")
	}
}
```

- [ ] **Step 2: Renumber diagnostic codes**

In `internal/diagnostic/diagnostic.go`, replace the code constants with the renumbered set:

```go
const (
	CodeAmbiguousLink       = "1"
	CodeBrokenLink          = "2"
	CodeNBSP                = "3"
	CodeMissingFrontMatter  = "4"
	CodeMissingShortDesc    = "5"
	CodeHeadingHierarchy    = "6"
	CodeUnrecognizedSchema  = "7"
	CodeFootnoteRefOrphan   = "8"
	CodeFootnoteDefOrphan   = "9"
	CodeUnresolvedKeyref    = "10"
	CodeBrokenMapTopicref   = "11"
	CodeMapHeadingHierarchy = "12"
	CodeCoreProfileFeature  = "13"
	CodeConrefTargetMissing = "14"
	CodeConrefElementMissing = "15"
	CodeConkeyrefKeyMissing = "16"
	CodeConkeyrefElementMissing = "17"
	CodeTaskTypeInNonTask   = "18"
)
```

- [ ] **Step 3: Run test to verify codes**

```bash
go test ./internal/diagnostic/ -run TestDiagnosticCodeValues -v
```

- [ ] **Step 4: Write failing test for core profile warning**

```go
func TestCoreProfileFootnoteWarning(t *testing.T) {
	text := "---\n$schema: urn:oasis:names:tc:dita:xsd:topic.xsd\n---\n# Title\n\nText with footnote[^1].\n\n[^1]: Footnote text\n"
	doc := document.New("file:///test.md", 1, text)
	cfg := config.Default()
	cfg.Core.Mdita.Profile = config.ProfileCore
	diags := CheckProfile(doc, cfg)
	found := false
	for _, d := range diags {
		if d.Code == CodeCoreProfileFeature {
			found = true
		}
	}
	if !found {
		t.Error("expected core profile feature warning for footnote")
	}
}
```

- [ ] **Step 5: Create profile.go**

Create `internal/diagnostic/profile.go`:

```go
package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
)

func CheckProfile(doc *document.Document, cfg *config.Config) []Diagnostic {
	if cfg.Core.Mdita.Profile != config.ProfileCore {
		return nil
	}

	var diags []Diagnostic

	idx := doc.Index
	if idx.BlockFeatures.HasFootnoteRefs || idx.BlockFeatures.HasFootnoteDefs {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeCoreProfileFeature,
			Source:   "mdita-lsp",
			Message:  "Footnotes are not available in MDITA core profile",
		})
	}

	if idx.BlockFeatures.HasDefinitionList {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeCoreProfileFeature,
			Source:   "mdita-lsp",
			Message:  "Definition lists are not available in MDITA core profile",
		})
	}

	if idx.BlockFeatures.HasAttributes {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeCoreProfileFeature,
			Source:   "mdita-lsp",
			Message:  "Attributes are not available in MDITA core profile",
		})
	}

	return diags
}
```

- [ ] **Step 6: Implement task type mismatch diagnostic**

Add to `internal/diagnostic/mdita.go`:

```go
func checkTaskTypeInNonTask(doc *document.Document) []Diagnostic {
	isTask := doc.Meta != nil && doc.Meta.Schema == document.SchemaTask
	if isTask {
		return nil
	}
	for _, e := range doc.Elements {
		if h, ok := e.(*document.Heading); ok && h.IsTitle() && h.Attributes != nil {
			for _, c := range h.Attributes.Classes {
				if c == "task" {
					return nil
				}
			}
		}
	}

	var diags []Diagnostic
	for _, e := range doc.Elements {
		if h, ok := e.(*document.Heading); ok && h.TaskSection != document.TaskSectionNone {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeTaskTypeInNonTask,
				Source:   "mdita-lsp",
				Message:  "Task section heading in a non-task topic (add {.task} to H1 or set $schema to task.xsd)",
			})
		}
	}
	return diags
}
```

- [ ] **Step 7: Wire new checks into Check()**

In `internal/diagnostic/diagnostic.go`, update `Check` to call `CheckProfile`, `CheckConrefs` (from Task 6), and `checkTaskTypeInNonTask`:

```go
func Check(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic
	cfg := folder.Config()

	if config.BoolVal(cfg.Diagnostics.MditaCompliance) {
		diags = append(diags, checkMditaCompliance(doc, cfg)...)
		diags = append(diags, CheckProfile(doc, cfg)...)
		diags = append(diags, checkTaskTypeInNonTask(doc)...)
	}
	if config.BoolVal(cfg.Diagnostics.LinkValidation) {
		diags = append(diags, checkLinks(doc, folder)...)
	}
	if config.BoolVal(cfg.Diagnostics.NbspDetection) {
		diags = append(diags, checkNonBreakingWhitespace(doc)...)
	}
	if doc.Kind == document.Map && config.BoolVal(cfg.Diagnostics.DitamapValidation) {
		diags = append(diags, CheckDitamap(doc, folder)...)
	}
	if config.BoolVal(cfg.Diagnostics.KeyrefResolution) {
		diags = append(diags, CheckKeyrefs(doc, folder)...)
	}
	diags = append(diags, CheckConrefs(doc, folder)...)
	return diags
}
```

- [ ] **Step 8: Update existing diagnostic tests**

Update test files to use the new code constants. Remove tests for deleted diagnostics (domain classes, related links, admonition types, etc.).

- [ ] **Step 9: Run all tests**

```bash
go test ./... -count=1
make lint
```

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "refactor: renumber diagnostics, add profile and task-type checks"
```

---

### Task 10: Final Verification & Cleanup

**Files:**
- Modify: `CLAUDE.md` (update LSP capabilities list)
- Modify: `.mdita-lsp.yaml` (update default config to reflect new features)
- Modify: `internal/lsp/server.go` (verify all capabilities registered)

**Interfaces:**
- Consumes: all prior tasks
- Produces: clean build, all tests passing, updated docs

- [ ] **Step 1: Full test suite**

```bash
make test
```

Verify 0 failures. Fix any remaining issues.

- [ ] **Step 2: Lint check**

```bash
make lint
```

Fix any lint issues.

- [ ] **Step 3: Cross-compile verification**

```bash
make publish
```

Verify all 5 platform builds succeed.

- [ ] **Step 4: Update CLAUDE.md**

Update the LSP capabilities list in `CLAUDE.md` to reflect:
- Removed: domain elements, related links, `{{keyref}}`, YAML keydefs
- Added: conref/conkeyref support, `$schema` profile switching, auto-format tables on save, implicit task sections
- Changed: keyref model (reference-style keydefs, `data-keyref`)

- [ ] **Step 5: Update .mdita-lsp.yaml default config**

Add `profile` and `formatTablesOnSave` to the example config.

- [ ] **Step 6: Commit**

```bash
git add CLAUDE.md .mdita-lsp.yaml
git commit -m "docs: update project docs for org.lwdita alignment"
```

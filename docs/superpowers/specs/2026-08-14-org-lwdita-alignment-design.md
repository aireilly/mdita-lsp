# org.lwdita Alignment Design

**Date:** 2026-08-14
**Status:** Draft
**Scope:** Refactor mdita-lsp main branch to align exclusively with [org.lwdita](https://github.com/jelovirt/org.lwdita) feature set. Archive current Red Hat extended features to `redhat.mdita.extended` branch.

## Branch Strategy

Create `redhat.mdita.extended` from current main at HEAD (`acebd93`). This is an archival fork preserving all Red Hat-specific features. The refactor happens on main. No ongoing divergence management; the extended branch can cherry-pick from main as the org.lwdita-aligned core evolves.

## Feature Removal

Strip these Red Hat-specific systems from main:

### vocabulary/ package
Delete entirely. The 17 domain elements (uicontrol, wintitle, menucascade, shortcut, filepath, cmdname, userinput, systemoutput, varname, msgph, codeph, option, parmname, apiname, kwd, cite, draft-comment), task list element registry, step element registry, and conditional attribute registry all go. Task section kinds (`prereq`, `context`, `result`, `postreq`, `tasktroubleshooting`) move to `document/types.go` as standalone constants since org.lwdita supports them.

### document/types.go
Remove `InlineAttribute` domain element types and `ParsedAttribute` for domain classes. Keep `BlockAttribute` for conditional processing attributes, `TaskSectionKind`, and heading attribute types.

### document/attributes.go
Remove domain class parsing (`{.uicontrol}`, `{.filepath}`, etc.). Keep header attribute parsing (`{#id .class key=val}`) and block attribute parsing for conditional attrs.

### keyref/ package
Remove `{{keyref}}` double-curly detection and YAML keydef loading. Rewrite for reference-style keyrefs (see Keyref Model section).

### completion/ package
Remove `PartialAttrClass`, `PartialDoubleCurlyKeyref`, and domain class completions. Keep inline link, YAML key, keyref, and task section completions.

### hover/ package
Remove domain element hover, related links hover, and heading class hover. Keep link, keyref, heading, and YAML key hover.

### diagnostic/ package
Remove codes 8-12 (schema-specific extended-in-core), 15 (admonition types), 20-29 (domain/attr specific: outputclass, domain class wrong parent, extended profile required, conditional attr validation, task section order/duplication, related links content, menucascade separator, step element outside step, reltable columns). Keep and renumber codes 1-7, 13-14, 16-19.

### codeaction/ package
Remove add-related-links and domain-specific fixes. Keep create file, front matter, add to map, DITA-OT build, task sections, NBSP fix, heading hierarchy fix.

### inlayhint/ package
Remove domain element mapping hints. Keep link target hints and keyref target hints.

### semantic/ package
Remove attribute decorator tokens for domain elements. Keep semantic token infrastructure for header attributes.

## Schema-based Profile System

### $schema front matter key

org.lwdita uses `$schema` in YAML front matter for profile/type selection:

```yaml
---
$schema: urn:oasis:names:tc:dita:xsd:topic.xsd
---
```

Supported schemas:

| URN | Type |
|-----|------|
| `urn:oasis:names:tc:dita:xsd:topic.xsd` | Generic topic |
| `urn:oasis:names:tc:dita:xsd:concept.xsd` | Concept |
| `urn:oasis:names:tc:dita:xsd:task.xsd` | Task |
| `urn:oasis:names:tc:dita:xsd:reference.xsd` | Reference |
| `urn:oasis:names:tc:dita:xsd:map.xsd` | Map |

The outputclass alternative (`{.task}` on H1) remains supported for type detection.

### Profile behavior

The `$schema` value also implies a profile level. org.lwdita distinguishes core and extended:

- **Core profile**: No attributes, no footnotes, no definition lists. Diagnostics warn when extended features appear.
- **Extended profile**: Attributes, footnotes, definition lists, specialization classes.
- **Default** (no `$schema`): Extended profile behavior.

Profile detection uses a config key in `.mdita-lsp.yaml`:

```yaml
profile: extended  # or "core"; default is "extended"
```

The document-level `$schema` sets the topic type. The project-level `profile` config sets the feature level. This matches org.lwdita's behavior where the schema sets specialization and the plugin configuration controls which markdown extensions load.

### Implementation

Refactor `DitaSchema` constants in `document/types.go` to use org.lwdita URN strings. Add a `Profile` type (`Core`, `Extended`) to `config/`. Thread profile through diagnostic, completion, and parser paths. Core profile disables footnote parsing in goldmark and suppresses attribute extension loading.

## Keyref Model Refactor

### Current model (remove)

- `(key)` inline syntax for keyref consumption
- YAML-based keydefs in separate `.yaml` files
- `{{key}}` double-curly inline keyrefs

### New model (org.lwdita)

**Keydef** (in map files): Reference-style link definitions.

```markdown
[key]: https://example.com "Title text"
```

Goldmark parses these as `LinkDefinition` nodes. When they appear in a map document (`$schema: ...map.xsd` or `.mditamap`), the LSP indexes them as DITA keydefs.

**Keyref consumption** (in topic files):

```markdown
See [link text][key] for details.
See [key] for the collapsed form.
```

Standard markdown reference links. The LSP resolves `[key]` against the map's keyspace built from reference-style definitions.

**Inline keyword keyref** (HDITA syntax):

```html
<span data-keyref="product-name">fallback</span>
```

Parsed from goldmark's raw HTML inline nodes. Resolves against the same keyspace.

### Implementation

Rewrite `internal/keyref/` package:

1. **Extraction**: Scan map documents for `LinkDefinition` nodes, index as keydefs with key name, href, and title.
2. **Resolution**: Build a keyspace per workspace from all map files. Topic files resolve reference links against this keyspace.
3. **Cursor detection**: Detect cursor inside `[text][key]`, `[key]`, or `<span data-keyref="...">` constructs.
4. **Completion**: When typing `][`, complete with available keys. When typing `data-keyref="`, complete with keys.
5. **Hover**: Show key definition (href + title) on hover over keyref.
6. **Definition**: Go-to-definition navigates to the `[key]: url` line in the map file.
7. **Diagnostics**: Warn on unresolved keyrefs (reference link with no matching keydef in any map).

The symbol graph gets a new edge type `KeyDef` linking map documents to their key definitions.

## Conref/Conkeyref System

New `internal/conref/` package.

### Syntax

```html
<p data-conref="file.md#topic/elementid">fallback</p>
<span data-conref="file.md#topic/elementid">fallback</span>
<p data-conkeyref="keyname/elementid">fallback</p>
```

The `data-conref` attribute value format: `filepath#topicid/elementid`. The `data-conkeyref` attribute value format: `keyname/elementid`.

### Parsing

During document indexing, scan goldmark's `RawHTML` nodes for elements with `data-conref` or `data-conkeyref` attributes. Extract:

- Element tag name
- Attribute name (`data-conref` or `data-conkeyref`)
- File path (for conref)
- Topic ID and element ID
- Key name (for conkeyref)
- Source position (line, column)

Store as `ConrefSymbol` entries in the document index.

### Element ID indexing

For conref targets to resolve, the LSP needs to index element IDs within documents. Header attributes (`{#myid}`) already produce IDs on headings. Extend to index IDs on any block element that has a header attribute with an ID component.

### LSP features

**Go-to-definition**: From a `data-conref` attribute, navigate to the target element in the target file. For `data-conkeyref`, resolve the key first (via keyspace), then navigate to the element within the resolved file.

**Hover**: Preview the referenced content. Show the target element's text content (first paragraph or heading text) in a hover popup.

**Diagnostics**:
- `conref-target-missing`: File referenced by conref does not exist
- `conref-element-missing`: Element ID not found in target file
- `conkeyref-key-missing`: Key not defined in any map
- `conkeyref-element-missing`: Key resolves but element ID not found

**Completion**: When the cursor is inside `data-conref="`, complete with workspace file paths. After `#`, complete with topic IDs from that file. After `/`, complete with element IDs. For `data-conkeyref="`, complete with defined keys, then `/` triggers element ID completion.

### Inlay hints

Show resolved conref target path as an inlay hint after the closing `>` of a conref element.

## Task Section Alignment

org.lwdita's task model uses both implicit and explicit section detection.

### Type detection

A topic is a task when either:
- `$schema: urn:oasis:names:tc:dita:xsd:task.xsd` in front matter
- `{.task}` outputclass on the H1 heading

Same pattern for concept (`{.concept}`, `...concept.xsd`) and reference (`{.reference}`, `...reference.xsd`).

### Implicit sections

In a task topic:
- Content before the first ordered/unordered list at body level wraps in `<context>`
- An `ol` at body level becomes `<steps>`; a `ul` becomes `<steps-unordered>`
- Content after the last steps block wraps in `<result>`

### Explicit sections

Headings with outputclass attributes name sections:
- `## Prerequisites {.prereq}`
- `## Context {.context}`
- `## Result {.result}`
- `## What to do next {.postreq}`
- `## Troubleshooting {.tasktroubleshooting}`

### Step structure (new: implicit nested detection)

Within a step (list item):
- First paragraph becomes `<cmd>`
- Subsequent content becomes `<info>`
- Nested `ol` becomes `<substeps>`
- Nested `ul` becomes `<choices>` (new: implicit detection, currently requires explicit `{.choices}`)
- Table within a step becomes `<choicetable>` (new: implicit detection)

### Implementation

The existing task section code handles explicit sections and step structure. Changes:
1. Add implicit `<context>` and `<result>` wrapping detection for hover/diagnostics
2. Add implicit `<choices>` detection (nested `ul` in step without requiring `{.choices}`)
3. Add implicit `<choicetable>` detection (table in step)
4. Remove vocabulary package dependency; move task section constants to `document/types.go`

## Auto-format Tables on Save

### Trigger mechanism

Register `textDocument/willSaveWaitUntil` capability during initialization. When the client sends `textDocument/willSave` with `reason: Manual` (user-initiated save), compute and return table formatting edits. The save applies the edits atomically before writing to disk.

Fallback for clients that don't support `willSaveWaitUntil`: listen for `textDocument/didSave`, compute edits, send `workspace/applyEdit` request. The file gets saved twice (once unformatted, once formatted) but the result is correct.

### Algorithm

The `alignTables` function in `internal/formatting/` already implements table alignment. It:
1. Identifies GFM table blocks in the document
2. Computes max column width per column
3. Pads cells with trailing spaces
4. Aligns separator row dashes
5. Preserves alignment markers (`:---`, `:---:`, `---:`)

For the on-save path, call `alignTables` on the full document and return the resulting `TextEdit` array.

### Configuration

Add to `.mdita-lsp.yaml`:

```yaml
formatTablesOnSave: true  # default: true
```

When `false`, tables only format on explicit format request (`textDocument/formatting`). The config merges at all three levels (global, project, workspace) following existing config merge behavior.

### Capability registration

In the `initialize` response, add `textDocumentSync.willSaveWaitUntil: true` to signal support. The existing `TextDocumentSyncOptions` struct in `internal/lsp/` needs this field added.

## Diagnostic Refactor

### Retained diagnostics (renumbered)

| New Code | Description |
|----------|-------------|
| 1 | Ambiguous link target |
| 2 | Broken link target |
| 3 | Non-breaking space detected |
| 4 | Missing YAML front matter |
| 5 | Missing short description |
| 6 | Heading hierarchy violation |
| 7 | Unrecognized schema (updated for org.lwdita URNs) |
| 8 | Footnote ref without definition |
| 9 | Footnote def without ref |
| 10 | Unresolved keyref |
| 11 | Broken map topicref |
| 12 | Map heading hierarchy |

### New diagnostics

| Code | Description |
|------|-------------|
| 13 | Core profile feature used (footnote, attribute, definition list in core-profile doc) |
| 14 | Broken conref target (file missing) |
| 15 | Broken conref element (ID not found) |
| 16 | Broken conkeyref (key not defined) |
| 17 | Broken conkeyref element (key resolves, element ID not found) |
| 18 | Task features in non-task topic |

### Removed diagnostics

All domain element diagnostics, related links diagnostics, menucascade separator, step-element-outside-step, reltable column validation, admonition type validation, core-vs-extended schema mismatch.

## Files Affected

### Deleted
- `internal/vocabulary/` (entire package)

### New
- `internal/conref/` (new package: conref.go, conref_test.go)

### Major rewrites
- `internal/keyref/` (new keyref model)
- `internal/diagnostic/` (renumbered, new codes, removed codes)

### Significant modifications
- `internal/document/types.go` (schema URNs, remove domain types)
- `internal/document/attributes.go` (remove domain class parsing)
- `internal/document/parser.go` (conref detection, profile-aware parsing)
- `internal/document/index.go` (element ID indexing, keydef indexing)
- `internal/completion/` (remove domain completions, add conref/keyref completions)
- `internal/hover/` (remove domain hover, add conref hover)
- `internal/definition/` (add conref go-to-definition)
- `internal/codeaction/` (remove related links, domain fixes)
- `internal/inlayhint/` (remove domain hints, add conref hints)
- `internal/semantic/` (remove domain attribute tokens)
- `internal/formatting/` (wire table formatting to save trigger)
- `internal/lsp/` (willSaveWaitUntil handler, capability registration)
- `internal/config/` (profile setting, formatTablesOnSave setting)

### Minor modifications
- `internal/symbols/` (new edge types for keydefs, conrefs)
- `internal/docsymbols/` (remove domain element symbols)
- `internal/codelens/` (no domain-specific lenses)
- `cmd/mdita-lsp/main.go` (capability registration)

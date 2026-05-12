# Double-Curly Keyrefs and YAML Keydefs

**Date:** 2026-05-12
**Status:** Proposed

## Context

The `redhat.mdita.extended` DITA-OT plugin (v0.0.8) added two related features:

1. **`{{key}}` inline keyref syntax** — parses `{{product-name}}` into `<keyword keyref="product-name"/>` for product name and version substitution.
2. **YAML `keys:` keydef block in maps** — defines keyword keydefs in map YAML front matter (`keys: { product-name: "OpenShift" }`), producing `<keydef>` elements in DITA output.

The mdita-lsp already supports `[keyname]` shortcut-style keyrefs with detection, completion, hover, definition, diagnostics, and inlay hints. It also parses YAML front matter and extracts keys from map `TopicRef` hrefs. This spec covers extending both systems.

Other plugin features from the same release (Troubleshooting task section, profiling attributes, leading attribute paragraphs) are already fully covered by the LSP.

## Feature A: YAML `keys:` Keydef Block

### Overview

Map files can define keyword keydefs in YAML front matter:

```yaml
---
$schema: urn:oasis:names:tc:dita:xsd:map.xsd
keys:
  product-name: "Red Hat OpenShift Container Platform"
  product-short: "OpenShift"
  version: "4.15"
  product-url: "https://www.redhat.com/openshift"
---
```

Text values produce keyword-content keydefs. URL values (starting with `http://`, `https://`, or ending with `.md`, `.dita`, `.html`, `.xml`) produce href-based keydefs.

### Changes

#### 1. `KeyEntry` — add `Value` field (`internal/keyref/keyref.go`)

Add a `Value string` field to `KeyEntry`. For YAML text keys, `Value` holds the keyword text (e.g., `"Red Hat OpenShift Container Platform"`). For URL keys, `Href` is set as before. `Title` is set to the value for display purposes on text keys.

```go
type KeyEntry struct {
    Href     string
    Title    string
    Value    string         // keyword text for YAML-defined text keydefs
    DefURI   string         // URI of the map file where this key is defined
    DefRange document.Range // range of the key definition (for go-to-definition)
}
```

`DefURI` and `DefRange` are populated during key extraction so that go-to-definition can navigate to the defining map file and line.

#### 2. `YAMLMetadata` — add `Keys` field (`internal/document/types.go`)

```go
type YAMLMetadata struct {
    // ... existing fields ...
    Keys map[string]string // YAML keys: block (map files only)
}
```

#### 3. Parse `keys:` in `parseYAMLMeta` (`internal/document/parser.go`)

Handle the `keys` case in the YAML parsing switch. The value is a `map[string]any` where each entry is a string key-value pair.

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

#### 4. Extract YAML keys in `BuildMergedTable` (`internal/keyref/keyref.go`)

After extracting TopicRef-based keys from `ditamap.ParseMap()`, also extract YAML front matter from the map text using a lightweight YAML-only parse (reuse `parseYAMLMeta` or extract the YAML block and unmarshal `keys:` directly — no need to run the full `document.Parse()` goldmark pipeline). YAML `keys:` entries take precedence over stem-based keys.

Add a helper `isURLValue(s string) bool` that checks for `http://`, `https://` prefixes or `.md`, `.dita`, `.html`, `.xml` suffixes — matching the plugin's `isUrl()` logic.

For text values: set `Value` and `Title` to the string.
For URL values: set `Href` to the string, `Title` empty.

#### 5. Completion for `keys:` YAML key (`internal/completion/completion.go`)

Add `"keys"` to the `yamlKeys` slice so it appears in YAML front matter completion for map files.

#### 6. Hover for `keys:` entries (`internal/hover/hover.go`)

When hovering on a key name inside a `keys:` block in map YAML, show the key name and its value. This is a natural extension of the existing `hoverYAMLKey()` function.

## Feature B: `{{key}}` Double-Curly Keyref Syntax

### Overview

The `{{key-name}}` syntax is an inline keyref that coexists with the `[keyname]` shortcut syntax. Valid key names match `[a-zA-Z0-9_.\-]+`. Invalid patterns (empty, spaces, special characters) pass through as literal text.

| Syntax | DITA output | Use case |
|--------|-------------|----------|
| `{{key}}` | `<keyword keyref="key"/>` | Product names, versions |
| `[text][key]` | `<xref keyref="key">text</xref>` | Link-style keyrefs |

### Changes

#### 1. Detection (`internal/keyref/detect.go`)

Add a `doubleCurlyRe` regex:

```go
var doubleCurlyRe = regexp.MustCompile(`\{\{([a-zA-Z0-9_.\-]+)\}\}`)
```

Extend `DetectAll()` to also scan for `{{key}}` matches and append `KeyrefLocation` entries. The `EndChar` should point past the closing `}}`.

Extend `DetectAtPosition()` to also check `{{key}}` matches at the cursor position. Return `KeyrefAtPos` with label = key name and range covering `{{key}}`.

Both functions must skip matches inside fenced code blocks (``` regions) and inline code (`` ` `` regions). Add a helper to identify code regions by line number.

#### 2. Completion (`internal/completion/partial.go`, `internal/completion/completion.go`)

Add `PartialDoubleCurlyKeyref` to the `PartialKind` enum.

In `DetectPartial()`, detect the `{{` prefix before the cursor (after existing checks but before `[` keyref detection):

```go
if idx := strings.LastIndex(prefix, "{{"); idx >= 0 {
    if !strings.Contains(prefix[idx:], "}}") {
        input := prefix[idx+2:]
        // return PartialDoubleCurlyKeyref with range
    }
}
```

The `Complete()` function dispatches `PartialDoubleCurlyKeyref` to a new `completeDoubleCurlyKeyref()` handler that:
- Builds the merged key table
- Returns `CompletionItem` entries with `InsertText = "{{key}}"` (or just `key}}` if the `{{` is already typed)
- Uses `TextEdit` to replace the partial range
- Sets `Kind = 6` (variable) to distinguish from link-style keyrefs

#### 3. Hover (`internal/hover/hover.go`)

Extend the hover function to check for `{{key}}` at the cursor position using `keyref.DetectAtPosition()` (which now handles both syntaxes). The hover content shows:

- **Key name** as heading
- **Resolved value** (from `KeyEntry.Value`) for text keydefs
- **Resolved href** (from `KeyEntry.Href`) for URL keydefs
- **"Unresolved keyref"** if not found in any map

#### 4. Go-to-definition (`internal/definition/definition.go`)

Extend `resolveKeyref()` to handle `{{key}}` cursor positions. For YAML-defined keys, navigate to the `keys:` line in the map file. For stem-based keys, navigate to the target topic file (existing behavior).

This uses the `DefURI` and `DefRange` fields on `KeyEntry` (added in Feature A) to navigate to the definition location in the map file.

#### 5. Diagnostics (`internal/diagnostic/keyref.go`)

Extend `CheckKeyrefs()` to also scan for `{{key}}` occurrences via `keyref.DetectAll()` (which now returns both syntaxes). Unresolved `{{key}}` references produce the same warning (code 16: `CodeUnresolvedKeyref`).

Skip `{{key}}` occurrences inside fenced code blocks and inline code spans.

#### 6. Inlay hints (`internal/inlayhint/inlayhint.go`)

Extend `keyrefHints()` to also process `{{key}}` occurrences from `keyref.DetectAll()`. Display:

- `Value` for text keydefs (e.g., ` -> Red Hat OpenShift Container Platform`)
- `Href` for URL keydefs (e.g., ` -> https://www.redhat.com/openshift`)

#### 7. Semantic tokens (`internal/semantic/semantic.go`) — optional

Add `{{key}}` ranges as semantic tokens. The `{{` and `}}` delimiters could use an operator token type, and the key name could use a variable token type. This is lower priority and can be deferred.

## Code region detection

Both detection and diagnostics need to skip `{{key}}` patterns inside code. Use a regex-based approach: scan for ``` fences to build a set of "code line ranges", and check inline code spans per-line with a backtick scanner. This avoids re-parsing the goldmark AST and is sufficient for detection purposes. Place the utility in `internal/keyref/` alongside the detection code.

Note: the existing `[keyname]` detection does not have this protection, but it's less ambiguous since `[` is a markdown construct. `{{` could appear literally in code blocks (e.g., Jinja templates, Helm charts), so code-region skipping is important for `{{key}}` specifically.

## Test plan

- `keyref/detect_test.go`: test `{{key}}` detection alongside `[key]` detection, including edge cases (inside code blocks, invalid key names, adjacent to text)
- `keyref/keyref_test.go`: test YAML `keys:` extraction, URL vs text value classification, precedence over stem keys
- `completion/`: test `{{` trigger, completion items with correct insert text and ranges
- `hover/`: test hover on `{{key}}` showing resolved value
- `diagnostic/`: test unresolved `{{key}}` warning, skip inside code blocks
- `inlayhint/`: test hints for `{{key}}` with resolved values
- Integration: test with map containing `keys:` block and topics using `{{key}}` syntax

## Out of scope

- Profiling attributes: already fully supported
- Troubleshooting task section: already in vocabulary
- Leading attribute paragraph rendering semantics: DITA-OT concern
- `{{key}}` semantic token highlighting: deferred
- Completion inside `keys:` sub-block for key names: deferred
- `conkeyref` / `conref` support: separate feature

# MDITA Extended Parity Design

Incorporate all features from `redhat.mdita.extended` (DITA-OT plugin) into `mdita-lsp` (Go LSP server), achieving full authoring support for the MDITA extended profile.

## Background

`redhat.mdita.extended` is a fork of `org.lwdita` that adds 15+ inline domain specializations, conditional processing attributes, extended task sections, related links, relationship tables, mapref, and DITA-to-Markdown roundtrip support. `mdita-lsp` currently has zero support for these features beyond schema name recognition.

## Architecture: Vocabulary-Driven with Vertical Slice Delivery

A central vocabulary registry (`internal/vocabulary/`) defines the complete extended MDITA vocabulary as Go data structures. All LSP features (diagnostics, completion, hover, code actions, semantic tokens, inlay hints) query this registry. Features are delivered in vertical slices — each slice adds parser support, diagnostics, completion, and hover for one feature area.

## 1. Vocabulary Registry (`internal/vocabulary/`)

### DomainElement

Defines each inline specialization that maps a Markdown `{.class}` attribute to a DITA element.

```go
type DomainElement struct {
    Class       string // outputclass name, e.g. "uicontrol"
    DITAElement string // DITA element name, e.g. "uicontrol"
    Domain      string // DITA domain, e.g. "ui-d"
    ParentKind  string // applicable Markdown parent: "bold", "code", "italic"
    Description string // hover text
}
```

**UI domain (ui-d):**

| Class | DITA Element | Parent | Description |
|-------|-------------|--------|-------------|
| `uicontrol` | `<uicontrol>` | bold | User interface control label |
| `wintitle` | `<wintitle>` | bold | Window or dialog title |
| `menucascade` | `<menucascade>` | bold | Menu path (split on ` > `) |
| `shortcut` | `<shortcut>` | bold | Keyboard shortcut |

**Software domain (sw-d):**

| Class | DITA Element | Parent | Description |
|-------|-------------|--------|-------------|
| `filepath` | `<filepath>` | code | File path or directory |
| `cmdname` | `<cmdname>` | code | Command name |
| `userinput` | `<userinput>` | code | User-entered text |
| `systemoutput` | `<systemoutput>` | code | System output text |
| `varname` | `<varname>` | code | Variable name |
| `msgph` | `<msgph>` | code | Message phrase |

**Programming domain (pr-d):**

| Class | DITA Element | Parent | Description |
|-------|-------------|--------|-------------|
| `codeph` | `<codeph>` | code | Code phrase |
| `option` | `<option>` | code | Command option |
| `parmname` | `<parmname>` | code | Parameter name |
| `apiname` | `<apiname>` | code | API name |
| `kwd` | `<kwd>` | code | Keyword |

**Cross-domain:**

| Class | DITA Element | Parent | Description |
|-------|-------------|--------|-------------|
| `cite` | `<cite>` | italic | Citation or title of a work |
| `draft-comment` | `<draft-comment>` | paragraph | Draft review comment (not published) |

### TaskSection

Defines recognized task heading patterns.

```go
type TaskSection struct {
    DefaultTitle string // case-insensitive match, e.g. "Prerequisites"
    Class        string // explicit class attribute, e.g. "prereq"
    DITAElement  string // DITA element name, e.g. "prereq"
    Description  string // hover text
    Order        int    // expected position in task (1=prereq, 2=context, 3=steps, 4=result, 5=postreq)
}
```

| Default Title | Class | DITA Element | Order |
|--------------|-------|-------------|-------|
| Prerequisites | `prereq` | `<prereq>` | 1 |
| About this task | `context` | `<context>` | 2 |
| *(steps)* | — | `<steps>` | 3 |
| Verification | `result` | `<result>` | 4 |
| Next steps | `postreq` | `<postreq>` | 5 |
| *(none)* | `tasktroubleshooting` | `<tasktroubleshooting>` | 6 |

### StepElement

Defines step-level specializations.

```go
type StepElement struct {
    Class       string // e.g. "stepresult"
    DITAElement string // e.g. "stepresult"
    Description string // hover text
}
```

| Class | DITA Element | Description |
|-------|-------------|-------------|
| `stepresult` | `<stepresult>` | Result of performing the step |
| `stepxmp` | `<stepxmp>` | Example for the step |

### ConditionalAttribute

Known DITA profiling attributes.

```go
type ConditionalAttribute struct {
    Name        string // e.g. "platform"
    Description string // hover text
}
```

| Name | Description |
|------|-------------|
| `audience` | Target audience (e.g., novice, expert) |
| `platform` | Target platform (e.g., linux, macos) |
| `product` | Product name or variant |
| `otherprops` | Custom profiling values |
| `deliveryTarget` | Output format (e.g., html, pdf) |
| `props` | Generic profiling attribute |
| `rev` | Revision identifier for flagging |

### Lookup Functions

```go
func LookupDomainElement(class string) (DomainElement, bool)
func LookupTaskSection(title string) (TaskSection, bool)      // case-insensitive
func LookupTaskSectionByClass(class string) (TaskSection, bool)
func LookupStepElement(class string) (StepElement, bool)
func IsConditionalAttribute(name string) bool
func AllDomainElements() []DomainElement
func AllTaskSections() []TaskSection
func AllConditionalAttributes() []ConditionalAttribute
```

## 2. Attribute Parsing

Three levels of attribute parsing, each targeting a different syntactic context.

### 2.1 Heading Attributes

Use goldmark's built-in `parser.WithAttribute()` option. Currently not enabled in `internal/document/parser.go`. Goldmark stores heading attributes as `ast.Attribute` on the heading node.

```go
goldmark.New(
    goldmark.WithParserOptions(
        parser.WithAttribute(), // enable {.class key="value"} on headings
    ),
    // ...existing extensions...
)
```

### 2.2 Block Attributes

Add `github.com/mdigger/goldmark-attributes` as a dependency. Handles `{key="value"}` on standalone lines before block elements.

```go
goldmark.New(
    goldmark.WithExtensions(
        attributes.Extension,
        // ...existing extensions...
    ),
)
```

New dependency count: 3 (goldmark, yaml.v3, goldmark-attributes).

### 2.3 Inline Attributes

Custom regex-based scanner in `internal/document/`. Runs on raw source text after goldmark parsing. Finds `{...}` immediately after closing bold/italic/code markers.

**Patterns matched:**

| Pattern | Example | Captures |
|---------|---------|----------|
| Bold + attr | `**Save**{.uicontrol}` | target="bold", text="Save", attrs=".uicontrol" |
| Code + attr | `` `config`{.filepath} `` | target="code", text="config", attrs=".filepath" |
| Italic + attr | `*Title*{.cite}` | target="italic", text="Title", attrs=".cite" |
| Paragraph + attr | `Text.{.stepresult}` | target="paragraph", attrs=".stepresult" |
| Paragraph + kv | `Text.{.draft-comment author="jsmith"}` | target="paragraph", attrs with key-values |

Paragraph-level attributes (`{.stepresult}`, `{.stepxmp}`, `{.draft-comment}`) are detected when `{...}` appears at the end of a paragraph's text content (after the last non-whitespace character). Block-level attributes (conditional processing) are detected when `{...}` appears on a standalone line with no other content.

The scanner extracts attributes into a `ParsedAttribute` struct:

```go
type ParsedAttribute struct {
    Classes   []string          // e.g. ["uicontrol"]
    ID        string            // e.g. "intro-section"
    KeyValues map[string]string // e.g. {"platform": "linux"}
    Range     Range             // byte offset range of the {…} block
}
```

### 2.4 Attribute Storage

Parsed attributes are stored on the document model:

```go
type InlineAttribute struct {
    Attr       ParsedAttribute
    TargetKind string // "bold", "italic", "code", "paragraph"
    TargetText string // text content of the inline element
    Line       int    // 0-based line number
    Col        int    // 0-based column of the opening {
}
```

## 3. Document Model Extensions

### HeadingInfo Extensions

```go
type HeadingInfo struct {
    // ...existing fields (Text, Level, Line, Slug, etc.)...
    Attributes  *ParsedAttribute // parsed {.class key="value"} on the heading
    TaskSection TaskSectionKind  // resolved task section type, or TaskSectionNone
    IsRelLinks  bool             // true if "Related information" / "Related links" / {.related-links}
}
```

`TaskSectionKind` is an enum: `TaskSectionNone`, `TaskSectionPrereq`, `TaskSectionContext`, `TaskSectionResult`, `TaskSectionPostreq`, `TaskSectionTroubleshooting`.

### Document Extensions

```go
type Document struct {
    // ...existing fields...
    InlineAttrs []InlineAttribute // all inline attributes found
    BlockAttrs  []BlockAttribute  // all block-level attributes found
    RelLinks    *RelatedLinksInfo // related links section, if present
}
```

### RelatedLinksInfo

```go
type RelatedLinksInfo struct {
    HeadingLine int    // line of the "Related information" heading
    Links       []Link // links within the section
}
```

### BlockAttribute

```go
type BlockAttribute struct {
    Attr ParsedAttribute
    Line int // line of the {…} block
}
```

## 4. Diagnostics

New diagnostic codes (extending the existing 19):

| Code | Severity | Condition | Message |
|------|----------|-----------|---------|
| 20 | Warning | `{.foo}` where foo is not a recognized domain element | Unknown outputclass "{.foo}" |
| 21 | Warning | `{.uicontrol}` on code span (should be bold) | Domain class "uicontrol" expects bold, not code |
| 22 | Info | Any `{key="value"}` attribute on non-extended-profile doc | Conditional attributes require MDITA extended profile |
| 23 | Warning | `{foo="bar"}` where foo is not a known conditional attr | Unknown conditional attribute "foo" |
| 24 | Warning | Task section heading in wrong order | Task section "Verification" should appear after steps |
| 25 | Error | Same task section heading appears twice | Duplicate task section: "Prerequisites" |
| 26 | Warning | Non-link content in related links section | Related links section should contain only links |
| 27 | Warning | `{.menucascade}` without ` > ` separator in text | menucascade requires " > " separator between menu items |
| 28 | Warning | `{.stepresult}` outside a task step | stepresult is only valid inside a task step |
| 29 | Warning | Reltable row has different column count than header | Relationship table row has N columns, header has M |

Existing diagnostic adjustments:

- Code for "generic attributes are extended profile" becomes code 22 (was previously a stub).
- Task schema validation gains task section ordering checks.

## 5. Completion

### 5.1 Domain Class Completion

**Trigger:** Typing `{.` inside or after an inline element.

**Behavior:** Show all domain element classes filtered by context:
- Inside/after bold → suggest uicontrol, wintitle, menucascade, shortcut
- Inside/after code → suggest filepath, cmdname, userinput, systemoutput, varname, msgph, codeph, option, parmname, apiname, kwd
- Inside/after italic → suggest cite

Each completion item shows the DITA element and domain in the detail field.

### 5.2 Heading Class Completion

**Trigger:** Typing `{.` after a heading.

**Behavior:** Suggest topic type classes (task, concept, reference) for H1, and task section classes (prereq, context, result, postreq, related-links, tasktroubleshooting) for H2 in task topics.

### 5.3 Conditional Attribute Completion

**Trigger:** Typing `{` on a standalone line (block attribute context).

**Behavior:** Suggest all conditional attributes as `attrname=""` snippets: `audience=""`, `platform=""`, `product=""`, etc.

### 5.4 Task Section Heading Completion

**Trigger:** Typing `## ` in a task topic.

**Behavior:** Suggest default task section titles: "Prerequisites", "About this task", "Verification", "Next steps", "Related information". Only suggest sections not already present.

### 5.5 DITAVAL Value Completion

**Trigger:** Inside `{platform="` with a cursor between quotes.

**Behavior:** If `.ditaval` files exist in the workspace, parse `<prop>` entries and suggest matching values for the current attribute name.

## 6. Hover

### 6.1 Domain Element Hover

Over `{.uicontrol}`: "DITA `<uicontrol>` (UI domain) -- User interface control label"

### 6.2 Conditional Attribute Hover

Over `{platform="linux"}`: "DITA conditional processing attribute `platform` -- Target platform. Content filtered by DITAVAL rules."

### 6.3 Task Section Hover

Over `## Prerequisites` in a task: "DITA `<prereq>` -- Content required before performing the task"

### 6.4 Related Links Hover

Over `## Related information`: "DITA `<related-links>` -- Links placed after the topic body in DITA output"

### 6.5 Step Element Hover

Over `{.stepresult}`: "DITA `<stepresult>` -- Expected result after performing the step"

## 7. Code Actions

| Action | Trigger | Effect |
|--------|---------|--------|
| Add related links section | Cursor in a topic without one | Insert `## Related information\n\n- []()\n` template |
| Add task section | Cursor in a task topic | Insert selected section heading (prereq, etc.) |
| Add domain class | Cursor on bold/code/italic text without attribute | Insert `{.class}` with class picker |
| Add conditional attribute | Cursor on a heading or block | Insert `{attr=""}` |

## 8. Ditamap Extensions

### 8.1 Relationship Tables

The mditamap parser (`internal/ditamap/`) gains table parsing:

```go
type RelTable struct {
    Header []RelCell // header row (relcolspec)
    Rows   []RelRow  // body rows
    Line   int       // source line
}

type RelRow struct {
    Cells []RelCell
}

type RelCell struct {
    TopicRefs []TopicRef // links in the cell
}
```

**Detection:** Markdown tables after the topic list in an `.mditamap` file. The parser currently stops after list parsing; extend to also parse subsequent tables.

**Diagnostics:** Inconsistent column counts, missing links, unresolvable topic refs.

**Hover:** Over a reltable cell link, show "Relationship table entry -- related to topics in other columns of this row."

### 8.2 Mapref

Links in mditamap to `.ditamap` or `.mditamap` files produce mapref semantics.

```go
type TopicRef struct {
    // ...existing fields...
    IsMapRef bool // true if href ends in .ditamap or .mditamap
}
```

**Detection:** Check link target extension in the map parser.

**Hover:** "Sub-map reference (mapref) -- includes topics from the referenced map."

**Go-to-definition:** Navigate to the referenced map file.

**Diagnostics:** Mapref target file does not exist.

## 9. Other LSP Features

### 9.1 Semantic Tokens

Attribute syntax `{...}` gets token type `decorator` (or `macro`). Domain class names within attributes get `type` modifier. Conditional attribute names get `variable` modifier.

### 9.2 Inlay Hints

After `**Save**{.uicontrol}`, show inline hint: `-> <uicontrol>`. After `{platform="linux"}`, show: `-> @platform`. These use the same rendering pattern as existing resolved-link inlay hints.

### 9.3 Document Symbols

Task sections appear in the document outline with DITA element type annotation. Related links section appears. In mditamap, reltable appears as a top-level symbol.

### 9.4 Formatting

Preserve attribute syntax during formatting. The formatter must not break `**text**{.class}` across lines or insert whitespace between the closing marker and `{`.

## 10. Delivery Slices

| Slice | Scope | Dependencies | Estimated Test Count |
|-------|-------|-------------|---------------------|
| 1 | Vocabulary registry + attribute parsing foundation | goldmark-attributes | ~30 |
| 2 | Heading attributes + task section detection + diagnostics + completion + hover | Slice 1 | ~40 |
| 3 | Inline attributes + domain specializations + diagnostics + completion + hover | Slice 1 | ~50 |
| 4 | Conditional processing attributes + diagnostics + completion + hover | Slice 1 | ~25 |
| 5 | Related links section detection + diagnostics + hover + symbols | Slice 1 | ~20 |
| 6 | Step-level elements (stepresult, stepxmp) + diagnostics + completion | Slice 1 | ~15 |
| 7 | Ditamap extensions (reltable, mapref) + diagnostics + hover | Slice 1 | ~30 |
| 8 | Semantic tokens + inlay hints + code actions + formatting | Slices 1-7 | ~25 |

Total estimated new tests: ~235. Total tests after: ~437.

## Non-Goals

- **DITA-to-Markdown roundtrip:** The LSP is an authoring tool for Markdown source. Roundtrip conversion is the DITA-OT plugin's responsibility.
- **DITAVAL file editing:** The LSP supports MDITA files, not DITAVAL XML. DITAVAL values are only read for completion suggestions.
- **HTML/PDF output:** Build output is handled by DITA-OT via the existing `ditaOtBuild` execute command.
- **`linklist` and `data-about`:** These elements have minimal coverage (44 occurrences in the AAP corpus) and are deferred.

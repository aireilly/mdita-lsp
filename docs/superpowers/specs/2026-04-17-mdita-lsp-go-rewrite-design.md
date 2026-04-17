# mdita-lsp: Go Rewrite Design Spec

## Overview

Rewrite the mdita-marksman LSP server from F# to Go. The new server is called `mdita-lsp` and lives in its own repository at `~/mdita-lsp`. Motivations: single static binary distribution (no runtime dependency) and easier contributor onboarding via Go's ecosystem.

## Scope

**Ported from F#:** All core LSP features — completion, goto definition, hover, references, rename, code actions (ToC, create missing file), code lenses (reference counts), semantic tokens, document symbols, workspace symbols. All MDITA-specific diagnostics (15 diagnostic codes). YAML front matter completion. `.mditamap` parsing. Wiki link syntax. Single-file mode. Gitignore support.

**New features:**
- **Keyref resolution:** Parse `[keyname]` shortcut references as keyrefs. Keys defined in `.mditamap` or keydef files. Provides: resolution, diagnostics for unresolved keyrefs, completion for available keys, hover showing resolved target.
- **Full ditamap validation:** All referenced files exist, no circular references in nested maps, heading hierarchy consistency across the map.

**Dropped from F# version:** `paranoid` config option, `incremental_references` option, `glfm_heading_ids` option.

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Language | Go | Static binaries, contributor accessibility |
| LSP framework | `go.lsp.dev/protocol` + `go.lsp.dev/jsonrpc2` | Same approach as gopls, well-maintained, auto-generated from LSP spec |
| Markdown parser | `github.com/yuin/goldmark` | Dominant Go parser, extensible, CommonMark compliant |
| Config format | YAML (`.mdita-lsp.yaml`) | Natural fit alongside MDITA's YAML front matter |
| Architecture | Feature-oriented packages under `internal/` | Independently testable, mirrors F# module structure |
| Testing | Unit + integration | Per-package unit tests, full JSON-RPC integration tests |

## Architecture

### Project Structure

```
mdita-lsp/
├── cmd/mdita-lsp/main.go       # Entry point, CLI flags, logging setup
├── internal/
│   ├── lsp/                    # Server lifecycle, handler dispatch, diagnostics manager
│   ├── document/               # Document, Element types, parser (goldmark), Index, Slug
│   ├── workspace/              # Workspace, Folder, file discovery, gitignore, doc sync
│   ├── symbols/                # Symbol types (Def/Ref), SymbolGraph, resolution
│   ├── completion/             # Completion candidates, ranking, partial element detection
│   ├── diagnostic/             # Link checks, MDITA compliance, ditamap validation
│   ├── definition/             # Goto definition logic
│   ├── hover/                  # Hover content generation
│   ├── references/             # Find references
│   ├── rename/                 # Rename + prepare rename
│   ├── codeaction/             # ToC generation, create missing file
│   ├── codelens/               # Reference count lenses
│   ├── docsymbols/             # Document + workspace symbol providers
│   ├── semantic/               # Semantic token encoding
│   ├── keyref/                 # Key definition parsing, keyref resolution
│   ├── ditamap/                # Mditamap parsing, structural validation
│   ├── config/                 # YAML config loading, merging, defaults
│   └── paths/                  # URI/path utilities, slug generation
├── testdata/                   # Shared fixtures: .md, .mditamap, .yaml configs
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
└── .github/workflows/
    ├── ci.yml                  # Build + test on push/PR
    └── release.yml             # Tag-triggered multi-platform binary release
```

### Document Model

Two-level representation (simplified from F#'s three-level CST/AST/Symbols):

1. **Elements** — parsed items with source ranges: headings, wiki links, markdown links, link definitions, YAML front matter. Carry both range data and normalized content.
2. **Symbols** — `Def` (doc title, heading, link def) and `Ref` (wiki link, md link, keyref) extracted from elements for cross-document resolution.

```go
type Document struct {
    URI      protocol.DocumentURI
    Version  int
    Text     string
    Lines    []int          // line start offsets
    Elements []Element
    Symbols  []Symbol
    Index    *Index
    Meta     *YAMLMetadata
    Kind     DocKind        // Topic | Map
}

type Element interface {
    Range() protocol.Range
    element()  // sealed
}
// Concrete: Heading, WikiLink, MdLink, LinkDef, YAMLBlock

type Symbol struct {
    Kind  SymKind  // Def or Ref
    Name  string
    Slug  Slug
    Scope DocID
    Range protocol.Range
}
```

**Index** — per-document fast lookup: headings by slug, links by target, YAML metadata, short description, block features (ordered lists, tables, definition lists, footnotes, admonitions).

**Parsing:** goldmark parses Markdown. A custom AST walker extracts Elements. A goldmark extension handles `[[doc#heading|title]]` wiki link syntax during parsing.

### Workspace & State Management

```
State
└── Workspace
    └── Folder (one per workspace folder)
        ├── root path
        ├── config (*Config, merged)
        ├── docs map[DocID]*Document
        ├── conn *SymbolGraph  (ref↔def edges)
        └── gitignore *GitIgnore
```

- File discovery scans for `*.md`, `*.markdown`, `*.mditamap` (respecting `.gitignore`).
- Supports full and incremental text sync (negotiated at init).
- On document change: re-parse, rebuild index/symbols, update symbol graph edges for that doc only, trigger debounced diagnostics.
- Symbol graph: bidirectional edges between Ref and Def symbols. Only changed document's edges recomputed.
- Single-file mode: ephemeral folder scoped to file's parent directory when document opened outside any workspace folder.
- Debounced diagnostics: goroutine per workspace, 200ms timer reset on each change.

### LSP Features

**Completion:** Triggered by `[`, `#`, `(`. Sources: wiki links (doc names + headings), inline links (file paths + anchors), reference links (link def labels), YAML keys/values, keyrefs. Configurable style: title-slug, file-stem, file-path-stem. Max candidates configurable.

**Goto Definition:** Resolves wiki links, markdown links, and keyrefs to target heading or document via symbol graph.

**Hover:** Shows target heading text, document title, or YAML metadata for links. For keyrefs, shows resolved value.

**References:** Find all refs to a heading or document via reverse symbol graph lookup.

**Rename:** Rename heading → updates all wiki links and markdown links referencing it across workspace. Supports prepare-rename validation.

**Document Symbols:** Hierarchical heading tree for outline view.

**Workspace Symbols:** Fuzzy search across all document titles and headings.

**Code Actions:** Generate/update table of contents. Create missing linked file.

**Code Lenses:** Reference count above each heading.

**Semantic Tokens:** Highlight wiki link components (doc part, heading part, title part).

### Diagnostics

**Ported (15 codes):**

| Code | Name | Severity |
|---|---|---|
| 1 | AmbiguousLink | Warning |
| 2 | BrokenLink | Error |
| 3 | NonBreakingWhitespace | Warning |
| 4 | MissingYamlFrontMatter | Warning |
| 5 | MissingShortDescription | Warning |
| 6 | InvalidHeadingHierarchy | Warning |
| 7 | UnrecognizedSchema | Warning |
| 8 | TaskMissingProcedure | Warning |
| 9 | ConceptHasProcedure | Info |
| 10 | ReferenceMissingTable | Info |
| 11 | MapHasBodyContent | Info |
| 12 | ExtendedFeatureInCoreProfile | Warning |
| 13 | FootnoteRefWithoutDef | Warning |
| 14 | FootnoteDefWithoutRef | Warning |
| 15 | UnknownAdmonitionType | Warning |

**New diagnostics:**

| Code | Name | Severity |
|---|---|---|
| 16 | UnresolvedKeyref | Warning |
| 17 | BrokenMapReference | Error |
| 18 | CircularMapReference | Error |
| 19 | InconsistentMapHeadingHierarchy | Warning |

### New Feature: Keyref Resolution

Keys are defined in `.mditamap` files or dedicated keydef files. A shortcut reference link `[keyname]` is treated as a keyref when MDITA mode is enabled and no matching link definition exists in the document.

Resolution: look up key in the folder's key table → return href + title. Keyrefs are resolved through the symbol graph like other references.

Provides: goto definition (jump to key definition), hover (show resolved href + title), completion (suggest available keys), diagnostics (UnresolvedKeyref for unknown keys).

### New Feature: Full Ditamap Validation

Validates `.mditamap` files structurally:

- **BrokenMapReference:** All `href` targets in topic refs resolve to existing files in the workspace.
- **CircularMapReference:** Detect cycles when maps reference other maps (nested map inclusion).
- **InconsistentMapHeadingHierarchy:** Validate that the heading structure across referenced topics is consistent with the map's nesting depth.

### Configuration

**File:** `.mdita-lsp.yaml` at workspace root. User-level: `~/.config/mdita-lsp/config.yaml`.

**Merging:** folder → user → defaults.

```yaml
core:
  markdown:
    file_extensions: [md, markdown, mditamap]
    text_sync: full  # or incremental
    title_from_heading: true
  mdita:
    enable: true
    map_extensions: [mditamap]

completion:
  wiki_style: title-slug  # or file-stem, file-path-stem
  max_candidates: 50

code_actions:
  toc:
    enable: true
    include_levels: [1, 2, 3, 4, 5, 6]
  create_missing_file:
    enable: true

diagnostics:
  mdita_compliance: true
  ditamap_validation: true
  keyref_resolution: true
```

### Build & Release

**Makefile targets:**
- `build` — `go build ./cmd/mdita-lsp`
- `test` — `go test ./...`
- `lint` — `golangci-lint run`
- `install` — `go install ./cmd/mdita-lsp`
- `clean` — remove build artifacts

**CI** (`ci.yml`): Build + test + lint on push to main and PRs. Ubuntu runner.

**Release** (`release.yml`): Tag-triggered (`v*`). GOOS/GOARCH matrix: linux-amd64, linux-arm64, macos-amd64, macos-arm64, windows-amd64. Static binaries, no CGO. Creates GitHub Release with auto-generated notes.

**Dependencies:**
- `go.lsp.dev/protocol` — LSP types
- `go.lsp.dev/jsonrpc2` — JSON-RPC transport
- `github.com/yuin/goldmark` — Markdown parsing
- `gopkg.in/yaml.v3` — YAML config + front matter
- `github.com/sabhiram/go-gitignore` — gitignore matching

## Testing Strategy

### Unit Tests

Per-package `_test.go` files:

| Package | Coverage |
|---|---|
| `document` | Parsing, index building, slug generation, YAML extraction, incremental edits, wiki link parser |
| `workspace` | File discovery, doc sync, config merging, single-file mode, gitignore filtering |
| `symbols` | Symbol extraction, graph edges, ref→def and def→ref resolution |
| `completion` | Partial element detection, candidate generation per source, ranking |
| `diagnostic` | Each diagnostic code individually |
| `definition` | Cross-doc and intra-doc goto definition for all link types |
| `hover` | Hover content for each link type |
| `references` | Find all references via reverse graph |
| `rename` | Heading rename propagation |
| `codeaction` | ToC generation, create-missing-file |
| `codelens` | Reference count computation |
| `ditamap` | Map parsing, circular detection, file existence |
| `keyref` | Key definition parsing, resolution, completion |
| `config` | YAML parsing, merging, defaults, errors |
| `paths` | URI↔path conversion, relative path resolution |

### Test Helpers

`internal/testutil/` package:
- `MakeDoc(uri, lines ...string) *Document` — in-memory document
- `MakeFolder(docs ...*Document) *Folder` — test folder with symbol graph
- `MakeWorkspace(folders ...*Folder) *Workspace` — full workspace

### Integration Tests

In `internal/lsp/`:
- Full server over in-memory `io.ReadWriter` (no real stdio)
- JSON-RPC lifecycle: initialize → initialized → open → request → verify
- Cover: initialization handshake, document sync, each LSP method end-to-end, error handling

### Test Fixtures

In `testdata/`, organized by scenario:
- `testdata/completion/` — docs with various completion contexts
- `testdata/diagnostics/` — docs triggering each diagnostic code
- `testdata/ditamap/` — valid, circular, broken-ref maps
- `testdata/keyref/` — keydef files and keyref documents
- `testdata/config/` — valid, invalid, partial configs

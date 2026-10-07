# CLAUDE.md

## Project overview

mdita-lsp is an LSP server for the Markdown source formats of the [org.lwdita](https://github.com/jelovirt/org.lwdita) DITA-OT plug-in: Markdown DITA (`md`, `markdown`), MDITA (`mdita`), and MDITA maps (`mditamap`). It is written in Go.

Scope rule: the server handles exactly what the plug-in parses. Do not add editor features for markdown the plug-in does not read, and do not model DITA constructs the plug-in never emits (for example `<mapref>`, `<related-links>`, or keyword keydefs).

- **Language:** Go
- **Repository:** `git@github.com:aireilly/mdita-lsp.git`
- **Binary name:** `mdita-lsp`
- **Dependencies:** goldmark, yaml.v3 (2 total)

## Build and test

```bash
make build      # Build the binary
make test       # Run tests with race detection across 27 packages
make lint       # Run golangci-lint
make install    # Build and install to ~/.local/bin
make publish    # Cross-compile for 5 platforms (~3.7 MB binary)
make clean      # Clean build artifacts
```

Ensure `~/go/bin` is on your `PATH`:

```bash
export PATH=$PATH:~/go/bin  # add to ~/.bashrc for persistence
```

## Project structure

```
cmd/mdita-lsp/          # Entry point (stdio JSON-RPC server)
internal/
  paths/                # URI/path utilities, slug generation
  config/               # YAML config loading with 3-level merging, profile selection
  document/             # Document parsing, indexing, symbol extraction, conref elements
    types.go            # Element types, symbols, DITA/MDITA schemas, implicit sections, conref
    parser.go           # goldmark parser, footnote regex, conref HTML scanning
    index.go            # Heading/link index with slug-based lookups
    document.go         # Document type, task detection, implicit task structure, configurable section titles
    attributes.go       # Block attribute parsing ({#id .class key="value"})
  conref/               # Content reference resolution (data-conref, data-conkeyref)
  ditamap/              # .mditamap parsing (nested markdown lists, TopicRef tree, reltable, mapref)
  workspace/            # Folder/workspace management, file scanning
  symbols/              # Symbol graph with bidirectional ref/def resolution
  diagnostic/           # 19 diagnostic codes (12 retired), MDITA compliance, profile, conref, link/map/keyref validation
  keyref/               # Keys from reference-style link definitions only, [text][key] refs, data-keyref
  definition/           # Go-to-definition for markdown links, keyrefs, conrefs
  hover/                # Hover for links, keyrefs, headings, YAML keys, task sections, conrefs, implicit task structure
  references/           # Find references to headings via symbol graph
  completion/           # Completion: inline links, YAML keys, keyrefs, data-keyref, conrefs, task sections
  rename/               # Heading rename
  codeaction/           # Create file, front matter, add to map, DITA OT, task sections
  codelens/             # Reference count lenses on headings
  docsymbols/           # Hierarchical document symbol outline, workspace symbol search
  folding/              # Folding ranges for headings, YAML front matter
  selection/            # Progressive selection expansion (line, element, section)
  linkededit/           # Linked editing of heading text
  formatting/           # Table alignment (including auto-format on save), trailing whitespace, heading spacing
                        # table.go: GFM-correct cell splitting; one edit per line, never overlapping
  inlayhint/            # Inline hints for link targets, keyref targets, conref targets
  filerename/           # Cross-reference updates on file rename (md links, map refs)
  highlight/            # Document highlight for headings and their intra-doc references
  semantic/             # Semantic token encoding (full + range) with attribute decorator tokens
  ditaot/               # DITA OT binary resolution and build invocation (xhtml, dita formats)
  lsp/                  # LSP server, JSON-RPC handler, diagnostic debouncing, willSaveWaitUntil
testdata/               # Test fixtures
.github/workflows/      # CI and Release workflows
```

## LSP capabilities

- Position encoding: `utf-16`, advertised in `initialize`. Every column crossing the protocol is UTF-16 code units, not bytes
- TextDocumentSync: Incremental (mode 2) with 200ms diagnostic debouncing, or full with `core.markdown.text_sync: full`
- WillSaveWaitUntil: Auto-format tables on save
- Completion (inline links, YAML keys, keyrefs, data-keyref, conrefs, task sections) with resolve
- Definition (markdown links, keyrefs, conrefs)
- Hover (markdown links, keyrefs, headings, YAML keys, task sections, conrefs, implicit task structure)
- Document Highlight, References, Rename (with prepare), Code Actions, Code Lens
- Document Links, Folding Ranges, Document Symbols, Workspace Symbols
- Selection Ranges, Linked Editing Ranges
- Formatting (full + range), Inlay Hints
- Semantic Tokens (full + range)
- Pull Diagnostics (textDocument/diagnostic, LSP 3.17)
- File Operations (didCreate, didDelete, willCreate, willRename)
- Execute Command (createFile, addToMap, ditaOtBuild)
- Profile support (core vs extended, resolved from $schema then from config)
- Diagnostic quick-fixes (NBSP, footnotes, heading hierarchy)
- Ditamap parsing (topicrefs, nesting, keydefs, relationship tables)
- Server Info (name + version in initialize response)
- Configuration change notification (workspace/didChangeConfiguration)

## Key files

- `Makefile` — build, test, publish targets
- `.mdita-lsp.yaml` — project-level config (user config at `~/.config/mdita-lsp/config.yaml`)
- `go.mod` — dependencies: goldmark, yaml.v3

## Workflow

- Run `make lint` before every commit to ensure zero lint issues
- golangci-lint must be built with the same Go toolchain that compiles the project, or it fails with "could not load export data ... export data version 4 is greater than maximum supported version 2" on every stdlib import. Reinstall it with `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2` after a Go upgrade; `.github/workflows/ci.yml` pins the same version
- Run `make test` to verify no regressions

## org.lwdita alignment facts

Details that are easy to get wrong; all verified against the plug-in source.

- Keys come only from reference-style link definitions in a map (`[key]: file.md "Title"` → `<keydef>`). A plain topicref defines no key, and every keydef carries an href — there are no keyword keydefs.
- `[key]` resolves to `<xref keyref>` only when the current topic does not define that label locally; a local definition makes it a plain `<xref href>`.
- A link to a `.ditamap`/`.mditamap` is a `<topicref>` with `@format`, never a `<mapref>`.
- `<topichead>`, `collection-type="sequence"`, and `<reltable>` exist only in Markdown DITA maps (`$schema: …map.xsd`). The `mditamap` reader enables no tables extension, and its link-less list items become plain `<topicref>`.
- Neither MDITA profile enables the attributes extension, so `{.class}` never works in MDITA. MDITA also caps headings at level 2, and a deeper heading throws rather than degrading.
- A heading id comes from flexmark's `HeaderIdGenerator.generateId` with the plug-in's defaults: letters lowercased, digits kept, ` -_` to `-`, everything else dropped, runs not collapsed, and duplicates resolved with `-1`, `-2`. `paths.Slugify` and `paths.AnchorIDs` reproduce it.
- `plugin.xml` enables `implicit-task-sections` for `md` and `markdown` (6.3.0 onwards; `md` alone before that), never for MDITA. `IMPLICIT_SUBSTEPS` defaults to true, `IMPLICIT_CHOICES` and `IMPLICIT_CHOICETABLE` to false.
- A skipped heading level, and a section heading at or above its parent topic's level, are both fatal `ParseException`s in `MarkdownParserImpl.validate` and `TopicRenderer`. Diagnostics for them are errors.
- The plug-in resolves an href against the source file's directory only. There is no same-name fallback anywhere in the tree.
- The paragraph after the title becomes `<shortdesc>` only when a `$schema` is declared or the title carries `{.concept}`/`{.task}`/`{.reference}`.
- A body-level list that is followed by another body-level list stays in `<context>` as body-ol/body-ul; only the last one becomes `<steps>`/`<steps-unordered>`. An ordered list that restarts its numbering at 1 is split at the restart.
- "Procedure"/"Steps" is a marker heading: it maps to no element and the list after it becomes `<steps>`. Section titles are configurable via `core.mdita.implicit_task_sections`.

## Conventions

- Config filename: `.mdita-lsp.yaml`
- Version injected via `-ldflags "-X main.version=..."` at build time
- All packages under `internal/` — not importable externally
- Tests colocated with source (`*_test.go` in each package)

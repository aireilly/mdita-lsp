# CLAUDE.md

## Project overview

mdita-lsp is an LSP server for MDITA (Markdown DITA) documents, written in Go. It aligns with the [org.lwdita](https://github.com/jelovirt/org.lwdita) DITA-OT plug-in feature set.

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
    types.go            # Element types, symbols, DITA schemas, implicit sections, conref
    parser.go           # goldmark parser, footnote regex, conref HTML scanning
    index.go            # Heading/link index with slug-based lookups
    document.go         # Document type with incremental change support, implicit task sections
    attributes.go       # Block attribute parsing ({#id .class key="value"})
  conref/               # Content reference resolution (data-conref, data-conkeyref)
  ditamap/              # .mditamap parsing (nested markdown lists, TopicRef tree, reltable, mapref)
  workspace/            # Folder/workspace management, file scanning
  symbols/              # Symbol graph with bidirectional ref/def resolution
  diagnostic/           # 19 diagnostic codes, MDITA compliance, profile, conref, link/map/keyref validation
  keyref/               # Reference-style keydefs, [text][key] refs, data-keyref detection
  definition/           # Go-to-definition for markdown links, keyrefs, conrefs
  hover/                # Hover for links, keyrefs, headings, YAML keys, task sections, conrefs, implicit sections
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

- TextDocumentSync: Incremental (mode 2) with 200ms diagnostic debouncing
- WillSaveWaitUntil: Auto-format tables on save
- Completion (inline links, YAML keys, keyrefs, data-keyref, conrefs, task sections) with resolve
- Definition (markdown links, keyrefs, conrefs)
- Hover (markdown links, keyrefs, headings, YAML keys, task sections, conrefs, implicit sections)
- Document Highlight, References, Rename (with prepare), Code Actions, Code Lens
- Document Links, Folding Ranges, Document Symbols, Workspace Symbols
- Selection Ranges, Linked Editing Ranges
- Formatting (full + range), Inlay Hints
- Semantic Tokens (full + range)
- Pull Diagnostics (textDocument/diagnostic, LSP 3.17)
- File Operations (didCreate, didDelete, willCreate, willRename)
- Execute Command (createFile, addToMap, ditaOtBuild)
- Profile support (core vs extended, matching org.lwdita)
- Diagnostic quick-fixes (NBSP, footnotes, heading hierarchy)
- Ditamap extensions (relationship tables, mapref detection)
- Server Info (name + version in initialize response)
- Configuration change notification (workspace/didChangeConfiguration)

## Key files

- `Makefile` — build, test, publish targets
- `.mdita-lsp.yaml` — project-level config (user config at `~/.config/mdita-lsp/config.yaml`)
- `go.mod` — dependencies: goldmark, yaml.v3

## Workflow

- Run `make lint` before every commit to ensure zero lint issues
- Run `make test` to verify no regressions

## Conventions

- Config filename: `.mdita-lsp.yaml`
- Version injected via `-ldflags "-X main.version=..."` at build time
- All packages under `internal/` — not importable externally
- Tests colocated with source (`*_test.go` in each package)

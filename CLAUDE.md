# CLAUDE.md

## Project overview

mdita-lsp is an LSP server for MDITA (Markdown DITA) documents, written in Go. It is the Go rewrite of [mdita-marksman](https://github.com/aireilly/mdita-marksman) (F#), with full feature parity plus additional capabilities.

- **Language:** Go
- **Repository:** `git@github.com:aireilly/mdita-lsp.git`
- **Binary name:** `mdita-lsp`
- **Dependencies:** goldmark, yaml.v3 (2 total)

## Build and test

```bash
make build      # Build the binary
make test       # Run tests with race detection (210+ tests across 27 packages)
make lint       # Run golangci-lint
make install    # Build and install to ~/.local/bin
make publish    # Cross-compile for 5 platforms (3.5 MB binary)
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
  config/               # YAML config loading with 3-level merging
  document/             # Document parsing, indexing, symbol extraction
    types.go            # Element types, symbols, DITA schemas, footnote labels
    parser.go           # goldmark parser, footnote/admonition regex
    index.go            # Heading/link index with slug-based lookups
    document.go         # Document type with incremental change support
    attributes.go       # Inline/block attribute parsing ({.class}, {key="value"})
  vocabulary/           # DITA domain element registry (17 elements, 5 task sections, 7 conditional attrs)
  ditamap/              # .mditamap parsing (nested markdown lists → TopicRef tree, reltable, mapref)
  workspace/            # Folder/workspace management, file scanning
  symbols/              # Symbol graph with bidirectional ref/def resolution
  diagnostic/           # 29 diagnostic codes, MDITA compliance, link/map/keyref/attribute validation
  keyref/               # Key extraction, resolution, cursor detection for keyrefs
  definition/           # Go-to-definition for markdown links and keyrefs
  hover/                # Hover for markdown links, keyrefs, headings, YAML keys, domain elements, task sections
  references/           # Find references to headings via symbol graph
  completion/           # Completion: inline links, YAML keys, keyrefs, task sections, attribute classes
  rename/               # Heading rename
  codeaction/           # ToC, create file, front matter, add to map, DITA OT, related links, task sections
  codelens/             # Reference count lenses on headings
  docsymbols/           # Hierarchical document symbol outline, workspace symbol search
  folding/              # Folding ranges for headings, YAML front matter, ToC markers
  selection/            # Progressive selection expansion (line → element → section)
  linkededit/           # Linked editing of heading text
  formatting/           # Table alignment, trailing whitespace, heading spacing, trailing newline
  inlayhint/            # Inline hints for link targets, keyref targets, and domain element mappings
  filerename/           # Cross-reference updates on file rename (md links, map refs)
  highlight/            # Document highlight for headings and their intra-doc references
  semantic/             # Semantic token encoding (full + range) with attribute decorator tokens
  ditaot/               # DITA OT binary resolution and build invocation (xhtml, dita formats)
  lsp/                  # LSP server, JSON-RPC handler, diagnostic debouncing, execute command
testdata/               # Test fixtures
.github/workflows/      # CI and Release workflows
```

## LSP capabilities

- TextDocumentSync: Incremental (mode 2) with 200ms diagnostic debouncing
- Completion (inline links, YAML keys, keyrefs, task sections, attribute classes) with resolve
- Definition (markdown links, keyrefs)
- Hover (markdown links, keyrefs, headings, YAML keys, domain elements, task sections, conditional attrs)
- Document Highlight, References, Rename (with prepare), Code Actions, Code Lens
- Document Links, Folding Ranges, Document Symbols, Workspace Symbols
- Selection Ranges, Linked Editing Ranges
- Formatting (full + range), Inlay Hints
- Semantic Tokens (full + range)
- Pull Diagnostics (textDocument/diagnostic, LSP 3.17)
- File Operations (didCreate, didDelete, willCreate, willRename)
- Execute Command (createFile, addToMap, ditaOtBuild)
- MDITA Extended Profile (17 domain elements, task sections, conditional processing, related links)
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

# CLAUDE.md

## Project overview

mdita-lsp is an LSP server for MDITA (Markdown DITA) documents, written in Go.

- **Language:** Go
- **Repository:** `git@github.com:aireilly/mdita-lsp.git`
- **Binary name:** `mdita-lsp`

## Build and test

```bash
make build      # Build the binary
make test       # Run tests with race detection
make lint       # Run golangci-lint
make install    # Build and install to ~/.local/bin
make publish    # Cross-compile for all platforms
make clean      # Clean build artifacts
```

## Project structure

```
cmd/mdita-lsp/          # Entry point
internal/
  paths/                # URI/path utilities, slug generation
  config/               # YAML config loading and merging
  document/             # Document parsing, indexing, symbol extraction
    types.go            # Element types, symbols, DITA schemas
    parser.go           # goldmark-based parser with wiki link extension
    wikilink_ext.go     # Custom goldmark inline parser for [[wiki links]]
    index.go            # Heading/link index with lookup methods
    document.go         # Document type with parse/reparse/symbol extraction
  ditamap/              # .mditamap parsing (nested markdown lists)
  workspace/            # Folder/workspace management, file scanning
  symbols/              # Symbol graph with bidirectional ref/def resolution
  diagnostic/           # 19 diagnostic codes, MDITA compliance, link/map validation
  keyref/               # Key extraction and resolution from ditamaps
  definition/           # Go-to-definition for wiki links and markdown links
  hover/                # Hover content for links and headings
  references/           # Find references to headings
  completion/           # Completion for wiki links, inline links, YAML keys
  rename/               # Heading rename with cross-doc ref updates
  codeaction/           # ToC generation, create missing file
  codelens/             # Reference count lenses on headings
  docsymbols/           # Document symbol outline tree
  semantic/             # Semantic token encoding for wiki links
  lsp/                  # LSP server, JSON-RPC handler, all LSP method handlers
testdata/               # Test fixtures
.github/workflows/      # CI and Release workflows
```

## Key files

- `Makefile` — build, test, publish targets
- `.mdita-lsp.yaml` — project-level config (user config at `~/.config/mdita-lsp/config.yaml`)
- `go.mod` — dependencies: goldmark, yaml.v3

## Conventions

- Config filename: `.mdita-lsp.yaml`
- Version injected via `-ldflags "-X main.version=..."` at build time
- All packages under `internal/` — not importable externally
- Tests colocated with source (`*_test.go` in each package)

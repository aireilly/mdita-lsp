# mdita-lsp

An LSP server for [MDITA](https://www.oasis-open.org/committees/tc_home.php?wg_abbrev=dita) (Markdown DITA) documents.

Provides comprehensive language support for `.md` and `.mditamap` files with 19 diagnostic codes, keyref resolution, incremental text sync, and full IDE integration.

## Install

### From GitHub Releases

Download the binary for your platform from [Releases](https://github.com/aireilly/mdita-lsp/releases) and place it on your `PATH`.

### From source

```bash
go install github.com/aireilly/mdita-lsp/cmd/mdita-lsp@latest
```

Or clone and build:

```bash
git clone https://github.com/aireilly/mdita-lsp.git
cd mdita-lsp
make install   # installs to ~/.local/bin
```

## Editor setup

### VS Code

Use any generic LSP client extension with:

```json
{
  "lsp.server.command": "mdita-lsp",
  "lsp.server.filetypes": ["markdown"]
}
```

### Neovim (nvim-lspconfig)

```lua
vim.api.nvim_create_autocmd("FileType", {
  pattern = { "markdown" },
  callback = function()
    vim.lsp.start({
      name = "mdita-lsp",
      cmd = { "mdita-lsp" },
      root_dir = vim.fs.dirname(vim.fs.find({ ".mdita-lsp.yaml", ".git" }, { upward = true })[1]),
    })
  end,
})
```

### Helix

Add to `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "markdown"
language-servers = ["mdita-lsp"]

[language-server.mdita-lsp]
command = "mdita-lsp"
```

## Configuration

Create `.mdita-lsp.yaml` in your project root or `~/.config/mdita-lsp/config.yaml` for user-wide settings.

```yaml
core:
  markdown:
    file_extensions: [md, markdown, mditamap]
  mdita:
    enable: true
    map_extensions: [mditamap]

completion:
  wiki_style: title-slug

code_actions:
  toc:
    enable: true
    include_levels: [2, 3, 4]
  create_missing_file:
    enable: true

diagnostics:
  mdita_compliance: true
  ditamap_validation: true
  keyref_resolution: true
```

## Features

| Feature | Description |
|---------|-------------|
| Diagnostics | 19 codes: MDITA compliance, link validation, heading hierarchy, footnotes, keyrefs, ditamap validation, map heading consistency |
| Completion | Wiki links (`[[`), inline links (`](`), YAML keys, heading anchors (`#`), keyrefs (`[`) with lazy documentation resolve |
| Go to Definition | Wiki links, markdown links, and keyref shortcut references |
| Hover | Document titles, heading text, keyref targets with href/title |
| Find References | All references to a heading across the workspace |
| Rename | Heading rename with cross-document wiki link updates |
| Code Actions | Generate ToC, create missing files, convert wiki→markdown links, add YAML front matter, add to mditamap, quick-fix NBSP/footnotes/heading hierarchy |
| Code Lens | Reference counts on headings |
| Document Links | Clickable links for wiki links and markdown links |
| Document Symbols | Hierarchical heading outline tree |
| Workspace Symbols | Search headings across all documents |
| Folding Ranges | Fold headings, YAML front matter, and ToC markers |
| Selection Ranges | Progressive selection expansion by line/element/section |
| Linked Editing | Simultaneous rename of headings and their `[[#heading]]` references |
| Formatting | Trim trailing whitespace, normalize headings, align tables, ensure trailing newline (full + range) |
| Inlay Hints | Show resolved wiki link titles and keyref targets inline |
| Document Highlight | Highlight all same-document references to heading under cursor |
| Semantic Tokens | Syntax highlighting for wiki links (full + range) |
| File Rename | Auto-update wiki links, markdown links, and map references on file rename |
| File Create | Auto-populate new `.md` files with MDITA YAML front matter |
| Execute Command | Create files, add documents to map |
| Pull Diagnostics | On-demand diagnostics via `textDocument/diagnostic` (LSP 3.17) |
| Text Sync | Incremental (mode 2) with 200ms diagnostic debouncing |
| File Operations | Auto-index created/deleted files, cross-document diagnostic refresh |

### Diagnostic codes

| Code | Name | Severity |
|------|------|----------|
| 1 | Ambiguous link | Warning |
| 2 | Broken link | Error |
| 3 | Non-breaking whitespace | Warning |
| 4 | Missing YAML front matter | Warning |
| 5 | Missing short description | Warning |
| 6 | Invalid heading hierarchy | Warning |
| 7 | Unrecognized schema | Warning |
| 8 | Task missing procedure | Warning |
| 9 | Concept has procedure | Info |
| 10 | Reference missing table | Info |
| 11 | Map has body content | Info |
| 12 | Extended feature in core profile | Warning |
| 13 | Footnote ref without def | Warning |
| 14 | Footnote def without ref | Info |
| 15 | Unknown admonition type | Warning |
| 16 | Unresolved keyref | Warning |
| 17 | Broken map reference | Error |
| 18 | Circular map reference | Error |
| 19 | Inconsistent map heading hierarchy | Info |

## MDITA map format

`.mditamap` files define document structure using nested markdown lists:

```markdown
# Product Documentation

- [Getting Started](getting-started.md)
  - [Installation](install.md)
  - [Configuration](config.md)
- [User Guide](user-guide.md)
```

Keys are derived from filenames (e.g., `install.md` → key `install`). Use `[install]` in topic files to create keyref shortcut references.

## Development

```bash
make build     # Build binary
make test      # Run 220 tests with race detection
make lint      # Run golangci-lint
make publish   # Cross-compile for 5 platforms (~3.5 MB each)
make clean     # Remove build artifacts
```

## License

See [LICENSE](LICENSE).

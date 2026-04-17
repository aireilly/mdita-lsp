# mdita-lsp

An LSP server for [MDITA](https://www.oasis-open.org/committees/tc_home.php?wg_abbrev=dita) (Markdown DITA) documents.

Provides diagnostics, completion, go-to-definition, hover, references, rename, code actions, code lens, document symbols, and semantic tokens for `.md` and `.mditamap` files.

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

Add to your `settings.json`:

```json
{
  "mdita-lsp.path": "mdita-lsp"
}
```

Or use any generic LSP client extension (e.g., [vscode-langservers](https://github.com/AriPerkkio/vscode-lsp-sample)) with:

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
  wiki_style: file-stem

code_actions:
  toc:
    enable: true
    include_levels: [1, 2, 3]
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
| Diagnostics | 19 diagnostic codes: MDITA compliance, link validation, heading hierarchy, schema-specific checks, ditamap validation |
| Completion | Wiki links (`[[`), inline links (`](`), YAML front matter keys, heading anchors (`#`) |
| Go to Definition | Navigate to linked documents and headings via wiki links and markdown links |
| Hover | Preview target document titles and headings |
| Find References | Find all references to a heading across the workspace |
| Rename | Rename headings with cross-document reference updates |
| Code Actions | Generate table of contents, create missing files |
| Code Lens | Reference counts on headings |
| Document Symbols | Heading outline tree |
| Semantic Tokens | Syntax highlighting for wiki links |

## MDITA map format

`.mditamap` files define document structure using nested markdown lists:

```markdown
# Product Documentation

- [Getting Started](getting-started.md)
  - [Installation](install.md)
  - [Configuration](config.md)
- [User Guide](user-guide.md)
```

## Development

```bash
make build     # Build binary
make test      # Run tests with race detection
make lint      # Run golangci-lint
make publish   # Cross-compile for all platforms
make clean     # Remove build artifacts
```

## License

See [LICENSE](LICENSE).

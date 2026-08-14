# mdita-lsp

An LSP server for [MDITA](https://www.oasis-open.org/committees/tc_home.php?wg_abbrev=dita) (Markdown DITA) documents, designed as the companion editor tooling for the [org.lwdita](https://github.com/jelovirt/org.lwdita) DITA-OT plug-in.

Every LSP feature maps directly to a markdown construct that the DITA-OT plug-in converts to DITA XML. The server validates, completes, and navigates MDITA content so that problems are caught at authoring time rather than at build time.

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

Ensure the install location is on your `PATH`:

```bash
# For go install:
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc
# For make install:
echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc
source ~/.bashrc
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

Create `.mdita-lsp.yaml` in your project root or `~/.config/mdita-lsp/config.yaml` for user-wide settings. Project config overrides user config, which overrides built-in defaults.

```yaml
core:
  markdown:
    file_extensions: [md, markdown, mditamap]
  mdita:
    enable: true
    map_extensions: [mditamap]
    profile: extended          # "core" or "extended" (default: extended)
    formatTablesOnSave: true   # auto-format tables on save (default: true)

completion:
  max_candidates: 50

code_actions:
  create_missing_file:
    enable: true

build:
  dita_ot:
    enable: true
    dita_path: ""          # Path to dita binary (empty = search $PATH)
    output_dir: "out"      # Output directory relative to workspace root

diagnostics:
  mdita_compliance: true
  ditamap_validation: true
  keyref_resolution: true
  link_validation: true
  nbsp_detection: true
```

### Profile selection

The `profile` setting controls which MDITA features are available:

- **extended** (default): Attributes, footnotes, definition lists, specialization classes.
- **core**: No attributes, no footnotes, no definition lists. Diagnostics warn when extended features appear.

This matches the org.lwdita plug-in's core vs. extended profile distinction.

## Supported markdown features

The LSP supports the same markdown features as the org.lwdita DITA-OT plug-in. Each feature below corresponds to a DITA conversion the plug-in performs.

### YAML front matter

The plug-in uses YAML front matter for topic type detection and prolog metadata. The LSP provides:

- **Completion** of all supported YAML keys: `$schema`, `id`, `author`, `source`, `publisher`, `permissions`, `audience`, `category`, `keyword`, `resourceid`
- **Hover** documentation for each key explaining its DITA mapping
- **Diagnostics** for missing front matter and unrecognized `$schema` values
- **Code action** to scaffold MDITA YAML front matter with default schema

```markdown
---
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
id: install-software
author: Documentation Team
category: Installation
keyword:
  - install
  - setup
---
```

Supported `$schema` values:

| Schema URN | Topic type |
|------------|------------|
| `urn:oasis:names:tc:dita:xsd:concept.xsd` | Concept |
| `urn:oasis:names:tc:dita:xsd:reference.xsd` | Reference |
| `urn:oasis:names:tc:dita:xsd:task.xsd` | Task |
| `urn:oasis:names:tc:dita:xsd:topic.xsd` | Generic topic |

### Headings and document structure

Headings map to DITA topic titles (H1) and sections (H2+). The LSP provides:

- **Document symbols** showing a hierarchical heading outline
- **Workspace symbols** to search headings across all documents
- **Folding ranges** for heading sections and YAML front matter
- **Selection ranges** for progressive expansion (line, element, section)
- **Linked editing** of heading text
- **Rename** with cross-file reference updates via the symbol graph
- **Document highlight** for headings and their intra-document references
- **Code lens** showing reference counts on headings
- **Diagnostics** for invalid heading hierarchy and heading-level skips

### Task topics

When `$schema` declares a task (or H1 has `{.task}` outputclass), the plug-in maps markdown constructs to DITA task elements. The LSP provides:

- **Completion** of task section headings: Prerequisites, Context, Result, What to do next, Troubleshooting
- **Code actions** to insert missing task sections
- **Hover** on task section headings showing their DITA element mapping

| Heading outputclass | DITA element |
|---------------------|--------------|
| `{.prereq}` | `<prereq>` |
| `{.context}` | `<context>` |
| `{.result}` | `<result>` |
| `{.postreq}` | `<postreq>` |
| `{.tasktroubleshooting}` | `<tasktroubleshooting>` |

The plug-in also performs implicit section detection:

- Ordered lists at body level become `<steps>` (nested OL becomes `<substeps>`)
- Unordered lists at body level become `<steps-unordered>` (nested UL becomes `<choices>`)
- Content before the first list wraps in `<context>`
- Content after the last list wraps in `<result>`
- Tables within a step become `<choicetable>`

### Links

The plug-in auto-classifies links based on URL pattern. The LSP provides:

- **Completion** of file paths inside `](` and heading anchors after `#`
- **Go to definition** for markdown links to other documents and headings
- **Diagnostics** for broken links and ambiguous links
- **Document links** making external URLs clickable
- **Inlay hints** showing resolved link targets inline
- **File rename** support that auto-updates cross-references when files are renamed

### Key references

Keys are defined as reference-style link definitions in map files:

```markdown
[product-name]: https://example.com "Product Name"
```

Keys are consumed in topic files via standard markdown reference links:

```markdown
See [the guide][install-guide] for details.
See [install-guide] for the collapsed form.
```

Inline keyword keyrefs use HDITA syntax:

```html
<span data-keyref="product-name">fallback</span>
```

The LSP provides:

- **Completion** of keyref references (`[keyname]`, `[text][keyname]`, `data-keyref="..."`)
- **Go to definition** navigating to the key definition line in the map file
- **Hover** showing resolved key targets and titles
- **Inlay hints** showing keyref resolution inline
- **Diagnostics** for unresolved keyrefs

### Content references (conref)

Content reuse via HTML data attributes, following the org.lwdita HDITA content reference model:

```html
<p data-conref="shared.md#topic/warning-para">fallback</p>
<span data-conkeyref="warnings/disk-full">fallback</span>
```

The LSP provides:

- **Go to definition** navigating to the referenced element
- **Hover** showing the conref target path
- **Completion** of file paths, topic IDs, and element IDs within conref attributes
- **Inlay hints** showing resolved conref targets
- **Diagnostics** for broken conref targets and missing element IDs

### Definition lists

Definition lists are converted to DITA `<dl>/<dlentry>`. In core profile, definition lists produce a diagnostic since they require the extended profile.

### Fenced code blocks

Fenced code blocks become `<codeblock>` with the language mapped to `@outputclass`.

### Pipe tables

Tables are converted to DITA `<simpletable>`. The LSP provides:

- **Formatting** to align table columns (full document and range)
- **Auto-format on save** when `formatTablesOnSave` is enabled (default: true)

### Images

Images are converted to `<image>` elements, with title-bearing images wrapped in `<fig>`. Standalone images receive `placement="break"`.

### Blockquotes

Blockquotes are converted to DITA `<lq>` (long quote).

### Inline formatting

Bold converts to `<b>`, italic to `<i>`, and code to `<codeph>`. Superscript and subscript are supported in extended profile.

### Hard line breaks

Trailing backslash or two trailing spaces produce a `<?linebreak?>` processing instruction.

### Footnotes

Footnotes are supported in the extended MDITA profile. The LSP provides:

- **Diagnostics** for footnote references without definitions and orphaned definitions
- **Code actions** to create missing footnote definitions

### Inline HTML

Inline HTML elements with `data-conref`, `data-conkeyref`, and `data-keyref` attributes are parsed for content reference and keyword keyref resolution.

## MDITA map format

`.mditamap` files define document structure using nested markdown lists. The plug-in converts these to DITA map XML.

```markdown
---
$schema: urn:oasis:names:tc:dita:xsd:map.xsd
---

# Product Documentation

- [Getting Started](getting-started.md)
  - [Installation](install.md)
  - [Configuration](config.md)
- [User Guide](user-guide.md)
```

The LSP provides:

- **Diagnostics** for broken map references, circular maps, and inconsistent heading hierarchy
- **Code actions** to add topics to an existing map
- **Execute command** to build XHTML or DITA output via DITA OT

### Topic references and sub-maps

Links become `<topicref>` elements. Links to `.ditamap` or `.mditamap` files are emitted as `<mapref>`.

### Ordered lists

Ordered list items produce `<topicref collection-type="sequence">`.

### Topic heads

List items without links become `<topichead>` with `<navtitle>`.

### Key definitions

Reference-style link definitions in map files define DITA keys:

```markdown
[install-guide]: install.md "Installation Guide"
```

Use `[install-guide]` or `[link text][install-guide]` in topic files to create keyref references.

### Relationship tables

Tables in `.mditamap` files are parsed as DITA `<reltable>`:

```markdown
| [Overview](overview.md) | [Install](install.md) |
|-------------------------|----------------------|
| [Config](config.md)     | [Troubleshoot](ts.md) |
```

## LSP capabilities

| Capability | Detail |
|-----------|--------|
| Text sync | Incremental (mode 2) with 200ms diagnostic debouncing |
| Completion | Trigger characters: `[`, `#`, `(`, `{` with resolve support |
| Definition | Markdown links, keyrefs, conrefs |
| Hover | Links, keyrefs, headings, YAML keys, task sections, conrefs |
| References | Cross-workspace heading references via symbol graph |
| Rename | Heading rename with prepare support |
| Code actions | Create missing files, add front matter, add to map, add task sections, fix NBSP/footnotes/heading hierarchy, build DITA OT |
| Code lens | Reference counts on headings |
| Document links | External URL detection |
| Document symbols | Hierarchical heading outline |
| Workspace symbols | Cross-document heading search |
| Folding ranges | Headings, YAML front matter |
| Selection ranges | Progressive expansion (line, element, section) |
| Linked editing | Heading text |
| Formatting | Table alignment, trailing whitespace, heading spacing, trailing newline (full + range) |
| Inlay hints | Link targets, keyref targets, conref targets |
| Document highlight | Heading and intra-document reference highlighting |
| Semantic tokens | Full + range encoding with attribute decorator tokens |
| Pull diagnostics | `textDocument/diagnostic` (LSP 3.17) |
| File operations | didCreate, didDelete, willCreate, willRename |
| Execute commands | `createFile`, `addToMap`, `ditaOtBuild` |
| Will save | `textDocument/willSaveWaitUntil` for table auto-format |

## Development

```bash
make build     # Build binary
make test      # Run tests with race detection
make lint      # Run golangci-lint
make publish   # Cross-compile for 5 platforms (~3.5 MB each)
make clean     # Remove build artifacts
```

Dependencies: `goldmark`, `yaml.v3` (2 total).

## License

See [LICENSE](LICENSE).

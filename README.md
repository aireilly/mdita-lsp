# mdita-lsp

An LSP server for [MDITA](https://www.oasis-open.org/committees/tc_home.php?wg_abbrev=dita) (Markdown DITA) documents, designed as the companion editor tooling for the [redhat.mdita.extended](https://github.com/aireilly/redhat.mdita.extended) DITA-OT plug-in.

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

## Supported markdown features

The LSP supports the same markdown features as the redhat.mdita.extended DITA-OT plug-in. Each feature below corresponds to a DITA conversion the plug-in performs.

### YAML front matter

The plug-in uses YAML front matter for topic type detection and prolog metadata. The LSP provides:

- **Completion** of all supported YAML keys: `$schema`, `id`, `author`, `source`, `publisher`, `permissions`, `audience`, `category`, `keyword`, `resourceid`
- **Hover** documentation for each key explaining its DITA mapping
- **Diagnostics** for missing front matter (code 4) and unrecognized `$schema` values (code 7)
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
| `urn:oasis:names:tc:mdita:rng:topic.rng` | MDITA topic |

### Headings and document structure

Headings map to DITA topic titles (H1) and sections (H2). The LSP provides:

- **Document symbols** showing a hierarchical heading outline
- **Workspace symbols** to search headings across all documents
- **Folding ranges** for heading sections and YAML front matter
- **Selection ranges** for progressive expansion (line → element → section)
- **Linked editing** of heading text
- **Rename** with cross-file reference updates via the symbol graph
- **Document highlight** for headings and their intra-document references
- **Code lens** showing reference counts on headings
- **Diagnostics** for invalid heading hierarchy (code 6) and heading-level skips

### Task topics

When `$schema` declares a task, the plug-in maps markdown constructs to DITA task elements. The LSP provides:

- **Diagnostics** for task missing procedure (code 8), concept has procedure (code 9), task section order (code 24), and duplicate sections (code 25)
- **Completion** of task section headings: Prerequisites, About this task, Verification, Next steps
- **Code actions** to insert missing task sections
- **Hover** on task section headings showing their DITA element mapping

| Heading text | DITA element |
|-------------|--------------|
| Prerequisites | `<prereq>` |
| About this task | `<context>` |
| Verification | `<result>` |
| Next steps | `<postreq>` |

The plug-in also maps:

- Ordered lists → `<steps>` (nested OL → `<substeps>`)
- Unordered lists → `<steps-unordered>` (nested UL → `<choices>`)
- Content before the first list → `<context>`
- Content after the list → `<result>`

### Related links

H2 headings titled "Related information" or "Related links" are converted by the plug-in to `<related-links>`. The LSP provides:

- **Code action** to add a related links section
- **Diagnostics** for non-link content inside related links sections (code 26)
- **Hover** showing the DITA `<related-links>` mapping

### Links

The plug-in auto-classifies links based on URL pattern. The LSP provides:

- **Completion** of file paths inside `](` and heading anchors after `#`
- **Go to definition** for markdown links to other documents and headings
- **Diagnostics** for broken links (code 2) and ambiguous links (code 1)
- **Document links** making external URLs clickable
- **Inlay hints** showing resolved link targets inline
- **File rename** support that auto-updates cross-references when files are renamed

### Key references

The plug-in processes DITA key references. Keys are derived from MDITA map topicrefs. The LSP provides:

- **Completion** of keyref shortcut references (`[keyname]`)
- **Go to definition** for keyrefs
- **Hover** showing resolved key targets and titles
- **Inlay hints** showing keyref resolution inline
- **Diagnostics** for unresolved keyrefs (code 16)

### Admonitions

Fenced admonitions (`!!! type`) are converted to DITA `<note>` elements. The LSP provides:

- **Diagnostics** for unknown admonition types (code 15)
- **Semantic tokens** for admonition syntax highlighting

Supported types: `note`, `tip`, `fastpath`, `restriction`, `important`, `remember`, `attention`, `caution`, `notice`, `danger`, `warning`, `trouble`. Other qualifiers produce `type="other"`.

### Definition lists

Definition lists are converted to DITA `<dl>/<dlentry>`. The LSP detects definition lists for profile validation:

- **Diagnostics** when definition lists appear in MDITA core profile topics (code 12), since they require the extended profile

### Fenced code blocks

Fenced code blocks become `<codeblock>` with the language mapped to `@outputclass`. Extended metadata syntax (`{.class #id key=value}`) is supported. The LSP provides:

- **Semantic tokens** for attribute metadata highlighting
- **Hover** on attribute classes and key-value pairs

### Pipe tables

Tables are converted to DITA `<simpletable>`. The LSP provides:

- **Formatting** to align table columns
- **Diagnostics** for reference topics missing a table (code 10)

### Images

Images are converted to `<image>` elements, with title-bearing images wrapped in `<fig>`. Standalone images receive `placement="break"`.

### Blockquotes

Blockquotes are converted to DITA `<lq>` (long quote).

### Inline formatting

The plug-in converts bold to `<b>`, italic to `<i>`, and code to `<codeph>` (or `<tt>` in extended profile). Superscript and subscript are supported in extended profile.

### Hard line breaks

Trailing backslash or two trailing spaces produce a `<?linebreak?>` processing instruction.

### Footnotes

Footnotes are supported in the extended MDITA profile. The LSP provides:

- **Diagnostics** for footnote references without definitions (code 13) and orphaned definitions (code 14)
- **Code actions** to create missing footnote definitions

### Inline HTML

Inline HTML tags are transformed to DITA equivalents by the plug-in via XSLT.

## Domain element specializations

The LSP supports inline attribute syntax for DITA domain specializations. These map standard markdown formatting to specific DITA elements:

```markdown
Click **File > Open**{.menucascade} to open the dialog.
Edit `config.yaml`{.filepath} to set options.
See *RFC 7231*{.cite} for details.
```

- **Completion** of domain element classes after `{.` and `{`
- **Hover** showing the DITA element name, domain, and description
- **Inlay hints** showing DITA element mappings inline
- **Diagnostics** for unknown outputclass (code 20) and wrong parent element (code 21)

| Domain | Elements | Markdown parent |
|--------|----------|-----------------|
| UI (ui-d) | `uicontrol`, `wintitle`, `menucascade`, `shortcut` | **bold** |
| Software (sw-d) | `filepath`, `cmdname`, `userinput`, `systemoutput`, `varname`, `msgph` | `` `code` `` |
| Programming (pr-d) | `codeph`, `option`, `parmname`, `apiname`, `kwd` | `` `code` `` |
| Topic | `cite` | *italic* |
| Topic | `draft-comment` | paragraph |

### Step elements

Inside task topics, step-level elements can be applied:

| Element | Description |
|---------|-------------|
| `stepresult` | Expected result of a step |
| `stepxmp` | Example for a step |

### Conditional processing attributes

Block-level attributes for DITA profiling:

```markdown
{platform="linux" audience="admin"}
```

- **Completion** of attribute names after `{`
- **Hover** documentation for each conditional attribute
- **Diagnostics** for unknown conditional attributes (code 23)

Supported: `audience`, `platform`, `product`, `otherprops`, `deliveryTarget`, `props`, `rev`.

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

- **Diagnostics** for broken map references (code 17), circular maps (code 18), inconsistent heading hierarchy (code 19), body content in maps (code 11), and reltable column inconsistency (code 29)
- **Code actions** to add topics to an existing map
- **Execute command** to build XHTML or DITA output via DITA OT

### Topic references and sub-maps

Links become `<topicref>` elements. Links to `.ditamap` or `.mditamap` files are emitted as `<mapref>`.

### Ordered lists

Ordered list items produce `<topicref collection-type="sequence">`.

### Topic heads

List items without links become `<topichead>` with `<navtitle>`.

### Key definitions

Keys are derived from topic filenames (e.g., `install.md` → key `install`). Use `[install]` in topic files to create keyref shortcut references.

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
| Definition | Markdown links, keyrefs |
| Hover | Links, keyrefs, headings, YAML keys, domain elements, task sections, conditional attributes |
| References | Cross-workspace heading references via symbol graph |
| Rename | Heading rename with prepare support |
| Code actions | Create missing files, add front matter, add to map, add task sections, add related links, fix NBSP/footnotes/heading hierarchy, build DITA OT |
| Code lens | Reference counts on headings |
| Document links | External URL detection |
| Document symbols | Hierarchical heading outline |
| Workspace symbols | Cross-document heading search |
| Folding ranges | Headings, YAML front matter |
| Selection ranges | Progressive expansion (line → element → section) |
| Linked editing | Heading text |
| Formatting | Table alignment, trailing whitespace, heading spacing, trailing newline (full + range) |
| Inlay hints | Link targets, keyref targets, domain element mappings |
| Document highlight | Heading and intra-document reference highlighting |
| Semantic tokens | Full + range encoding with attribute decorator tokens |
| Pull diagnostics | `textDocument/diagnostic` (LSP 3.17) |
| File operations | didCreate, didDelete, willCreate, willRename |
| Execute commands | `createFile`, `addToMap`, `ditaOtBuild` |

## Diagnostic codes

| Code | Name | Severity | DITA plug-in feature |
|------|------|----------|---------------------|
| 1 | Ambiguous link | Warning | Link auto-classification |
| 2 | Broken link | Error | Link processing |
| 3 | Non-breaking whitespace | Warning | Heading/title processing |
| 4 | Missing YAML front matter | Warning | `$schema` topic type detection |
| 5 | Missing short description | Warning | `<shortdesc>` generation |
| 6 | Invalid heading hierarchy | Warning | Topic/section nesting |
| 7 | Unrecognized schema | Warning | `$schema` validation |
| 8 | Task missing procedure | Warning | Task step generation |
| 9 | Concept has procedure | Info | Topic type validation |
| 10 | Reference missing table | Info | `<simpletable>` generation |
| 11 | Map has body content | Info | Map structure validation |
| 12 | Extended feature in core profile | Warning | Core vs. extended profile |
| 13 | Footnote ref without def | Warning | Footnote processing |
| 14 | Footnote def without ref | Info | Footnote processing |
| 15 | Unknown admonition type | Warning | `<note>` type mapping |
| 16 | Unresolved keyref | Warning | Key reference resolution |
| 17 | Broken map reference | Error | `<topicref>` processing |
| 18 | Circular map reference | Error | `<mapref>` processing |
| 19 | Inconsistent map heading hierarchy | Info | Map nesting validation |
| 20 | Unknown outputclass | Warning | Domain element classes |
| 21 | Domain class wrong parent | Warning | Domain specialization rules |
| 22 | Extended profile required | Warning | Profile feature gating |
| 23 | Unknown conditional attribute | Warning | Conditional processing |
| 24 | Task section out of order | Warning | Task section ordering |
| 25 | Duplicate task section | Error | Task section uniqueness |
| 26 | Related links non-link content | Warning | `<related-links>` generation |
| 27 | Menucascade missing separator | Warning | `<menucascade>` formatting |
| 28 | Step element outside task | Warning | Task step validation |
| 29 | Reltable inconsistent columns | Warning | `<reltable>` validation |

## Development

```bash
make build     # Build binary
make test      # Run 210+ tests with race detection
make lint      # Run golangci-lint
make publish   # Cross-compile for 5 platforms (~3.5 MB each)
make clean     # Remove build artifacts
```

Dependencies: `goldmark`, `yaml.v3` (2 total).

## License

See [LICENSE](LICENSE).

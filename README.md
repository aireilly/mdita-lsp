# mdita-lsp

An LSP server for the Markdown source formats of the [org.lwdita](https://github.com/jelovirt/org.lwdita) DITA-OT plug-in: Markdown DITA (`md`, `markdown`), MDITA (`mdita`), and MDITA maps (`mditamap`).

Every feature in this server maps to a markdown construct that the plug-in converts to DITA XML. Nothing here invents syntax the plug-in does not read. The server validates, completes, and navigates that syntax so problems surface while authoring instead of during a DITA-OT build.

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
    file_extensions: [md, markdown, mdita, mditamap]
    text_sync: incremental     # "incremental" or "full" document sync
  mdita:
    enable: true
    map_extensions: [mditamap]
    profile: extended          # MDITA profile where MDITA applies (default: extended)
    apply_to_markdown: false   # also treat .md and .markdown topics as MDITA
    formatTablesOnSave: true   # auto-format tables on save (default: true)
    implicit_task_sections:    # heading titles that map to task sections
      prereq: [prerequisites]
      context: ["about this task"]
      steps: [procedure, steps]
      result: [verification]
      postreq: ["next steps"]

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

`implicit_task_sections` mirrors the plug-in's `http://lwdita.org/sax/properties/implicit-task-sections/*` SAX properties. Only the sections you list are overridden; the rest keep the defaults above. The setting is applied server-wide, so in a multi-root session the most recently loaded workspace config wins.

## Source formats and profiles

The plug-in registers one parser per DITA-OT `format` value, and each parser enables a different set of markdown extensions. The server follows the same split.

| Plug-in format | Reader | What the server treats it as |
|----------------|--------|------------------------------|
| `md`, `markdown` | `MarkdownReader` | Markdown DITA: the full extension set, `{.class}` attributes, specialization from heading classes |
| `mdita` | `MDitaReader` | MDITA extended profile by default |
| `mditamap` | `MDitamapReader` | MDITA map |
| `hdita` | `HDitaReader` | Out of scope — the server reads markdown, not standalone HTML files |
| `wikidocs` | `MarkdownReader` | Out of scope — a build-time variant of Markdown DITA |

DITA-OT chooses the format from the `format` attribute on the `topicref` that points at the file, not from the file extension. Because the server cannot see the map that will consume a file, it goes by the extension: a `.mdita` file is MDITA, a `.md` or `.markdown` file is Markdown DITA. A declared `$schema` overrides that.

Two settings control the MDITA checks, because scope and profile are separate questions. `core.mdita.apply_to_markdown` extends them to `.md` and `.markdown` topics, which is what a workspace whose map gives those files `format="mdita"` needs; off by default, so only `.mdita` and `$schema`-typed files are MDITA. `core.mdita.profile` picks which profile applies where MDITA applies, and a declared `$schema` overrides it. `core.mdita.enable: false` turns the MDITA checks off everywhere.

### Profiles

| Feature | Markdown DITA | MDITA extended | MDITA core |
|---------|---------------|----------------|------------|
| `{.class}`, `{#id}` attributes | yes | no | no |
| Footnotes | yes | yes | no |
| Definition lists | yes | yes | no |
| Superscript, subscript | yes | yes | no |
| Strikethrough | yes | no | no |
| Abbreviations, admonitions, autolinks | yes | no | no |
| Tables | CALS `<table>` | `<simpletable>` | `<simpletable>` |
| Fenced code | `<codeblock>` | `<pre><tt>` | `<codeblock>` |
| Blockquote | `<lq>` | inlined, no `<lq>` | inlined, no `<lq>` |
| Inline code | `<codeph>` | `<tt>` | `<codeph>` |
| Heading depth | any | `#` and `##` only | `#` and `##` only |
| H2 becomes | a nested `<topic>`, unless classed | `<section>` | `<section>` |
| Specialization from heading class | yes | no | no |

Diagnostics report attributes in either MDITA profile, footnotes and definition lists in core, and headings below level 2 in either MDITA profile. A heading below level 2 is an error rather than a warning: `MarkdownParserImpl.validate` throws `LwDITA does not support level 3 heading` and the build produces nothing.

### YAML front matter

Front matter drives topic ID, parser selection, and the `<prolog>`. The server offers completion of every key the plug-in reads, hover documentation for each, a code action that scaffolds front matter, and diagnostics for missing front matter and unrecognized `$schema` values.

| Key | DITA output |
|-----|-------------|
| `id` | `@id` on the generated `<topic>` or `<map>` |
| `author` | `<author>` |
| `source` | `<source>` |
| `publisher` | `<publisher>` |
| `permissions` | `<permissions view="…">` |
| `audience` | `<metadata><audience type="…">` |
| `category` | `<metadata><category>` |
| `keyword` | `<metadata><keywords><keyword>` |
| `resourceid` | `<resourceid appid="…">` |
| anything else | `<data name="…" value="…">` |

`$schema` selects the parser profile and, for the concept, task, and reference schemas, forces the specialization:

| Schema URN | Result |
|------------|--------|
| `urn:oasis:names:tc:dita:xsd:topic.xsd`, `…:rng:topic.rng` | Generic topic |
| `urn:oasis:names:tc:dita:xsd:concept.xsd`, `…:rng:concept.rng` | `<concept>` |
| `urn:oasis:names:tc:dita:xsd:task.xsd`, `…:rng:task.rng` | `<task>` |
| `urn:oasis:names:tc:dita:xsd:reference.xsd`, `…:rng:reference.rng` | `<reference>` |
| `urn:oasis:names:tc:dita:xsd:map.xsd`, `…:rng:map.rng` | `<map>` |
| `urn:oasis:names:tc:mdita:xsd:topic.xsd`, `…:rng:topic.rng` | MDITA extended profile |
| `urn:oasis:names:tc:mdita:extended:xsd:topic.xsd`, `…:rng:topic.rng` | MDITA extended profile |
| `urn:oasis:names:tc:mdita:core:xsd:topic.xsd`, `…:rng:topic.rng` | MDITA core profile |

Declaring any `$schema` also turns on the shortdesc rule below.

## Supported markdown features

### Headings and document structure

H1 becomes the topic title. In Markdown DITA a lower heading becomes a nested `<topic>` unless it carries a class the plug-in recognizes; in MDITA an H2 always becomes a `<section>`. The plug-in rejects a document whose heading levels skip a level, and rejects a section heading that is not deeper than its parent topic title.

| Heading class | DITA element |
|---------------|--------------|
| `{.concept}` | `<concept>` |
| `{.task}` | `<task>` |
| `{.reference}` | `<reference>` |
| `{.section}` | `<section>` |
| `{.example}` | `<example>` |

The paragraph directly after the title becomes `<shortdesc>` when the document declares a `$schema`, or when the title carries a `{.concept}`, `{.task}`, or `{.reference}` class. Otherwise it is an ordinary body paragraph.

A document with several H1 headings is wrapped in a `<dita>` root. A document with no H1 gets one generated from the YAML `title`, the YAML `id`, or the filename.

The server provides:

- **Document symbols** showing a hierarchical heading outline
- **Workspace symbols** to search headings across all documents
- **Folding ranges** for heading sections and YAML front matter
- **Selection ranges** for progressive expansion (line, element, section)
- **Linked editing** of heading text
- **Rename** with cross-file reference updates via the symbol graph
- **Document highlight** for headings and their intra-document references
- **Code lens** showing reference counts on headings
- **Hover** naming the DITA element a classed heading produces
- **Diagnostics** for heading-level skips, missing short descriptions, and non-breaking whitespace in headings, with quick fixes for the last two

### Task topics

A topic becomes a DITA task when `$schema` names the task schema or the H1 carries `{.task}`. Sections come from heading classes, and — for Markdown DITA files parsed with the plug-in's `implicit-task-sections` feature, which `plugin.xml` enables for `format="md"` and `format="markdown"` — from well-known heading titles. The feature is off in both MDITA profiles, so a `## Prerequisites` in a `.mdita` file is a plain `<section>` and the server reports it as one.

Plug-in version matters here. For a `$schema`-typed task, the server's model matches the build only from 6.1.0 onwards: earlier versions discarded every reader feature for a document that declared a `$schema`, so `implicit-task-sections` never applied and each section heading became a nested task. Admonitions in such a topic need 6.2.0. A topicref with `format="markdown"` gets implicit task sections only from 6.3.0. Before that, `plugin.xml` enabled the feature for `format="md"` alone, so the same file produced a nested `<task>` per H2 under the other name.

| Heading class | Default heading title | DITA element |
|---------------|-----------------------|--------------|
| `{.prereq}` | Prerequisites | `<prereq>` |
| `{.context}` | About this task | `<context>` |
| — | Procedure, Steps | none: a marker; the ordered list after it becomes `<steps>` |
| `{.result}` | Verification | `<result>` |
| `{.postreq}` | Next steps | `<postreq>` |
| `{.tasktroubleshooting}` | — | `<tasktroubleshooting>` |

The titles are configurable, both in the plug-in and through `implicit_task_sections` in the server config.

Task structure the plug-in derives from the body, with no markup at all:

- A body-level ordered list becomes `<steps>`, each item a `<step>` whose first paragraph is `<cmd>` and whose remainder is `<info>`
- A body-level unordered list becomes `<steps-unordered>`
- A body-level list that is followed by another body-level list stays in `<context>` (the plug-in marks it `body-ol` or `body-ul`)
- An ordered list whose numbering restarts at 1 is split: the part before the restart stays in `<context>`, the rest becomes `<steps>`
- Body content before the steps list wraps in `<context>`; body content after it wraps in `<result>`
- A list nested inside a step becomes `<substeps>`, ordered or unordered: `implicit-substeps` defaults to true
- `{.choices}` turns a nested list into `<choices>`, and `{.choicetable}` turns a nested table into `<choicetable>` with `<chhead>`, `<chrow>`, `<choption>`, and `<chdesc>` cells. Without them `implicit-choices` and `implicit-choicetable` stay off, so a nested unordered list is `<substeps>` and a table in a step stays a plain `<table>`

The server provides:

- **Completion** of task section headings, including the Procedure marker
- **Code actions** to insert missing task sections
- **Hover** on task section headings and on every implicitly derived region, naming the DITA element it becomes
- **Diagnostics** for a task section heading in a topic that is not a task

### Links

The plug-in turns a markdown link into `<xref>` and derives `@format` from the target's file extension, plus `@scope="external"` for absolute URLs and root-relative paths. An email autolink gets `format="email"`.

A fragment is passed through to `@href` untouched, so both DITA addressing forms work: `file.md#topic-id` targets the topic, and `file.md#topic-id/element-id` targets an element inside it. The topic ID is the YAML `id` when present, otherwise the ID derived from the title.

The server provides:

- **Completion** of file paths inside `](` and heading anchors after `#`
- **Go to definition** for links to other documents, to headings, and to elements addressed as `#topic-id/element-id`
- **Diagnostics** for broken links, ambiguous links, and fragments that name a missing topic or element
- **Document links** making external URLs clickable
- **Inlay hints** showing the resolved target title
- **File rename** support that updates cross-references when files are renamed

### Key references

Keys are defined only by reference-style link definitions in a map file. The plug-in renders each one as a `<keydef>`; a plain topicref in a map defines no key.

```markdown
[install-guide]: install.md "Installation Guide"
```

Keys are consumed in topic files through reference-style links:

```markdown
See [the guide][install-guide] for details.
See [install-guide] for the collapsed form.
```

A reference resolves to `<xref keyref="…"/>` only when the topic itself does not define that label. A label the topic defines locally becomes an ordinary `<xref href="…">` instead.

Inline keyword keyrefs use HDITA syntax, which the plug-in maps to `@keyref` on the element:

```html
<span data-keyref="product-name">fallback</span>
```

The server provides:

- **Completion** of keyrefs (`[key]` and `data-keyref="…"`)
- **Go to definition** navigating to the key definition line in the map file
- **Hover** showing the resolved href and title
- **Inlay hints** showing keyref resolution inline
- **Diagnostics** for unresolved keyrefs

### Content references (conref)

The plug-in maps the HDITA `data-conref` and `data-conkeyref` attributes to DITA `@conref` and `@conkeyref`:

```html
<p data-conref="shared.md#topic/warning-para">fallback</p>
<span data-conkeyref="warnings/disk-full">fallback</span>
```

The server provides:

- **Go to definition** navigating to the referenced element
- **Hover** showing the conref target
- **Completion** of file paths, topic IDs, and element IDs inside the attribute
- **Inlay hints** showing resolved conref targets
- **Diagnostics** for broken conref targets and missing keys

### Tables

Pipe tables become CALS `<table>` in Markdown DITA and `<simpletable>` in both MDITA profiles. A table caption becomes `<title>`; column spans become `@namest`/`@nameend` or `@colspan`.

The server provides **formatting** to normalize pipe tables (full document and range) and **auto-format on save** when `formatTablesOnSave` is enabled. Each cell is written with one space inside its pipes, and delimiter rows collapse to `---` while keeping any alignment colons. Columns are not padded to a common width, so long cells never push a row past the editor's wrap point.

### Fenced code blocks

A fenced code block becomes `<codeblock>` — `<pre><tt>` in MDITA extended — with the info string mapped to `@outputclass` as `language-<lang>`. An info string wrapped in braces is parsed as attributes instead, so ```` ```{#id .class key=value} ```` sets `@id`, `@outputclass`, and arbitrary attributes.

### Images

An image becomes `<image>`. With a title it is wrapped in `<fig><title>`; alone in a paragraph it gets `placement="break"`; with alt text it gets an `<alt>` child. A reference-style image with no matching definition becomes `<image keyref="…"/>`. Attributes such as `{height=50px width=100px}` are carried onto the element.

### Definition lists

Definition lists become `<dl>` with `<dlentry>`, `<dt>`, and `<dd>`. They need the extended profile; in core the server reports a diagnostic.

### Footnotes

A footnote becomes an inline `<fn callout="…">`. A callout used more than once emits the `<fn>` once and cross-references it with `<xref type="fn">`. Footnotes need the extended profile.

The server provides **diagnostics** for footnote references without definitions and orphaned definitions, and a **code action** to create a missing definition.

### Blockquotes

A blockquote becomes `<lq>` in Markdown DITA. Both MDITA profiles render the content without a wrapper.

### Inline formatting

Bold becomes `<b>`, italic `<i>`, and inline code `<codeph>` — `<tt>` in MDITA extended. Superscript and subscript are unavailable in core. Strikethrough becomes `<line-through>` in Markdown DITA only. HTML entities are resolved to characters.

### Hard line breaks

A trailing backslash or two trailing spaces produce a `<?linebreak?>` processing instruction.

### Inline and block HTML

HTML is parsed as HDITA. Simple inline tags map straight to DITA elements, and everything else goes through the plug-in's HDITA-to-DITA stylesheet.

| HTML | Markdown DITA | MDITA extended |
|------|---------------|----------------|
| `<span>` | `<ph>` | `<ph>` |
| `<code>` | `<codeph>` | `<ph>` |
| `<s>` | `<line-through>` | `<ph>` |
| `<tt>` | `<tt>` | `<tt>` |
| `<b>`, `<strong>` | `<b>` | `<b>` |
| `<i>`, `<em>` | `<i>` | `<i>` |
| `<sub>`, `<sup>` | `<sub>`, `<sup>` | `<sub>`, `<sup>` |
| `<u>` | `<u>` | `<u>` |

The server reads `data-conref`, `data-conkeyref`, and `data-keyref` on these elements; the rest of the HTML vocabulary is passed to the build untouched.

## MDITA map format

A map is a `.mditamap` file, or a `.md` file that declares `$schema: urn:oasis:names:tc:dita:xsd:map.xsd`. Both define document structure with nested markdown lists.

```markdown
---
$schema: urn:oasis:names:tc:dita:xsd:map.xsd
---

# Product Documentation

- [Getting Started](getting-started.md)
  - [Installation](install.md)
  - [Configuration](config.md)

[install-guide]: install.md "Installation Guide"
```

The H1 becomes `<title>`; front matter becomes `<topicmeta>`. Each list item with a link becomes `<topicref>` with `@href` and `@format`, and its link text becomes `<navtitle>`. A list item with a reference-style link becomes a `<topicref keyref="…">`. A reference-style link definition becomes a `<keydef>`. Nesting becomes nested `<topicref>`.

Two map behaviours differ by format, because the `mditamap` reader enables fewer extensions than a Markdown DITA map:

| Construct | Markdown DITA map (`$schema: …map.xsd`) | `.mditamap` |
|-----------|------------------------------------------|-------------|
| List item without a link | `<topichead>` with `<navtitle>` | `<topicref>` with `<navtitle>` |
| Ordered list item | `<topicref collection-type="sequence">` | plain `<topicref>` |
| Pipe table | `<reltable>` with `<relheader>`, `<relrow>`, `<relcell>` | not parsed as a table |

A link to another map is an ordinary `<topicref>` whose `@format` is `ditamap` or `mditamap`; the plug-in does not emit `<mapref>`.

The server provides:

- **Diagnostics** for broken map references, circular map references, and topic heading levels that disagree with the map nesting
- **Code actions** to add a topic to an existing map
- **Execute command** to build XHTML or DITA output through DITA-OT

## Diagnostics

| Code | Message |
|------|---------|
| 1 | Ambiguous link |
| 2 | Broken link |
| 3 | Non-breaking whitespace in heading |
| 4 | No YAML front matter (information: Markdown DITA does not need it) |
| 5 | Missing short description |
| 6 | Skipped heading level (error: the build fails) |
| 7 | Unrecognized `$schema` |
| 8 | Footnote reference without definition |
| 9 | Footnote definition without reference |
| 10 | Unresolved keyref |
| 11 | Map references a non-existent file |
| 12 | Retired. Nesting a topicref does not constrain the nested topic's heading level, and the plug-in builds it without a message |
| 13 | Feature unavailable in MDITA core profile |
| 14 | Conref target not found |
| 15 | Conref element not found |
| 16 | Conkeyref key not found |
| 17 | Conkeyref element not found |
| 18 | Task section heading in a non-task topic |
| 19 | Circular map reference |
| 20 | Feature unavailable in the MDITA profiles (an error for a heading below level 2) |

## LSP capabilities

| Capability | Detail |
|-----------|--------|
| Position encoding | `utf-16`, negotiated in `initialize` |
| Text sync | Incremental (mode 2) with 200ms diagnostic debouncing, or full with `text_sync: full` |
| Completion | Trigger characters: `[`, `#`, `(`, `{` with resolve support |
| Definition | Markdown links, keyrefs, conrefs |
| Hover | Links, keyrefs, headings, YAML keys, task sections, implicit task structure, conrefs |
| References | Cross-workspace references to a topic and to each of its sections |
| Rename | Heading rename with prepare support; fragment links to the heading are repointed |
| Code actions | Create missing files, add front matter, add to map, add task sections, fix NBSP/footnotes/heading hierarchy, build with DITA-OT |
| Code lens | Reference counts on headings |
| Document links | External URL detection |
| Document symbols | Hierarchical heading outline |
| Workspace symbols | Cross-document heading search |
| Folding ranges | Headings, YAML front matter |
| Selection ranges | Progressive expansion (line, element, section) |
| Linked editing | Heading text |
| Formatting | Table normalization, trailing whitespace, heading spacing, trailing newline (full + range). Hard line breaks are kept, escaped and code-span pipes are not cell boundaries, and a pipe block that is not a table is left alone |
| Inlay hints | Link targets, keyref targets, conref targets |
| Document highlight | Heading and intra-document reference highlighting |
| Semantic tokens | Full + range encoding with attribute decorator tokens |
| Pull diagnostics | `textDocument/diagnostic` (LSP 3.17) |
| File operations | didCreate, didDelete, willCreate, willRename |
| Execute commands | `createFile`, `addToMap`, `ditaOtBuild` |
| Will save | `textDocument/willSaveWaitUntil` for table auto-format; it touches table lines only |

## Plug-in features the server does not surface

These are real plug-in behaviours with no editor affordance yet. They are listed so the coverage gap is explicit.

- **Raw DITA passthrough** — DITA element markup written directly in Markdown DITA (`raw-dita`, on by default for `md` and `markdown`, off for MDITA)
- **Jekyll tags** — `{% include file.md %}` becomes `<required-cleanup conref="…">`
- **Admonitions** — `!!! note` becomes `<note type="note">`, in schema-less Markdown DITA only
- **Abbreviations** — `*[HTML]: HyperText Markup Language` becomes `<ph otherprops="…">`
- **Autolinks** — bare URLs and `<user@example.com>` become `<xref>`
- **HDITA source files** — the `hdita` format reads standalone HTML documents
- **`wikidocs` format** — a Markdown DITA variant that synthesizes a missing title
- **DITA-to-Markdown transtypes** — `markdown`, `markdown_github`, `markdown_gitbook`, and `mdx` output, which run in the opposite direction

## Logging

The server writes a log to `mdita-lsp.log` in the user cache directory (`~/.cache/mdita-lsp` on Linux), truncated on each launch. Set `MDITA_LSP_LOG` to a path to write somewhere else, or to `-` to log to stderr.

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

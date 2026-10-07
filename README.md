# mdita-lsp

An LSP server for the Markdown source formats of the forked [org.lwdita](https://github.com/aireilly/org.lwdita) DITA-OT plug-in: Markdown DITA (`md`, `markdown`), MDITA (`mdita`), and MDITA maps (`mditamap`).

It validates, completes and navigates exactly the syntax the plug-in reads, so problems surface while you write instead of during a DITA-OT build.

## Requirements

**This server targets [aireilly/org.lwdita](https://github.com/aireilly/org.lwdita), a fork of [jelovirt/org.lwdita](https://github.com/jelovirt/org.lwdita). Version 6.3.0 or newer.** Against upstream, parts of what the server reports will not match what the build produces.

Every behaviour claim here holds against the fork. The fork carries features upstream lacks, and implicit task sections are the big one: `implicit-task-sections` and its configurable titles exist only in the fork. Upstream turns every `## Prerequisites` into a nested topic instead of a `<prereq>`. Its latest release is 5.9.1.

Install it over the `org.lwdita` in your DITA-OT:

```bash
cd $DITA_HOME/plugins
rm -rf org.lwdita && mkdir org.lwdita && cd org.lwdita
curl -LO https://github.com/aireilly/org.lwdita/releases/latest/download/org.lwdita-6.3.0.zip
unzip org.lwdita-6.3.0.zip && rm org.lwdita-6.3.0.zip
cd $DITA_HOME && ./bin/dita install
```

Confirm it took:

```bash
grep -m1 'plugin id="org.lwdita"' $DITA_HOME/plugins/org.lwdita/plugin.xml
```

## Install

### From GitHub Releases

Download the binary for your platform from [Releases](https://github.com/aireilly/mdita-lsp/releases) and place it on your `PATH`.

### From source

```bash
go install github.com/aireilly/mdita-lsp/cmd/mdita-lsp@latest
```

Alternatively, clone and build:

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

`implicit_task_sections` mirrors the plug-in's `http://lwdita.org/sax/properties/implicit-task-sections/*` SAX properties. List only the sections you want to change; the rest keep the defaults above. The setting takes effect server-wide, so in a multi-root session the workspace config loaded last wins.

## Source formats and profiles

The plug-in registers one parser per DITA-OT `format` value, and each parser enables a different set of markdown extensions. The server follows the same split.

| Plug-in format | Reader | What the server treats it as |
|----------------|--------|------------------------------|
| `md`, `markdown` | `MarkdownReader` | Markdown DITA: the full extension set, `{.class}` attributes, specialization from heading classes |
| `mdita` | `MDitaReader` | MDITA extended profile by default |
| `mditamap` | `MDitamapReader` | MDITA map |
| `hdita` | `HDitaReader` | Out of scope: the server reads markdown, not standalone HTML files |
| `wikidocs` | `MarkdownReader` | Out of scope: a build-time variant of Markdown DITA |

DITA-OT takes the format from the `topicref`, not the extension, and the server cannot see the map that will consume a file. It resolves the profile in this order:

1. A declared `$schema`. Reach for this one: DITA-OT reads it too, so the editor and the build cannot disagree.
2. Otherwise the extension. `.mdita` is MDITA extended, `.md` and `.markdown` are Markdown DITA.
3. `core.mdita.apply_to_markdown` also makes `.md` and `.markdown` MDITA extended. Set it when your map gives those files `format="mdita"`. Off by default.

Nothing selects the profile on its own, so declare `$schema: urn:oasis:names:tc:mdita:core:xsd:topic.xsd` in a topic that needs core. `core.mdita.enable: false` turns the MDITA checks off everywhere.

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

A heading below level 2 in either MDITA profile is an error, not a warning: the parser throws `LwDITA does not support level 3 heading` and the build produces nothing.

### YAML front matter

Front matter drives topic ID, parser selection, and the `<prolog>`. Front matter is optional: a topic without it is ordinary Markdown DITA.

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

Declaring any `$schema` also pulls the paragraph after the title into `<shortdesc>`, as does a `{.concept}`, `{.task}` or `{.reference}` title class.

## Authoring reference

The server models what the plug-in parses and nothing else. For the full markdown to DITA mapping, read the [org.lwdita README](https://github.com/aireilly/org.lwdita#readme). What follows is only the part that changes what the server reports.

### Headings

H1 is the topic title. In Markdown DITA a lower heading opens a nested `<topic>` unless it carries a recognized class; in MDITA an H2 is always a `<section>`.

| Heading class | DITA element |
|---------------|--------------|
| `{.concept}` | `<concept>` |
| `{.task}` | `<task>` |
| `{.reference}` | `<reference>` |
| `{.section}` | `<section>` |
| `{.example}` | `<example>` |

Two heading mistakes fail the build, so the server reports both as errors: skipping a level, and a section heading that is not deeper than the topic title above it.

Anchors follow the plug-in's rule. Letters lowercase, digits stay, space, `-` and `_` become `-`, and everything else disappears. Runs of separators survive intact, and a repeated heading gains `-1`, `-2`. So `## My_var config` is `#my-var-config`, and a second `## Setup` is `#setup-1`.

### Task sections

A topic is a task when `$schema` names the task schema or the H1 carries `{.task}`.

| Heading class | Default title | DITA element |
|---------------|---------------|--------------|
| `{.prereq}` | Prerequisites | `<prereq>` |
| `{.context}` | About this task | `<context>` |
| | Procedure, Steps | a marker: the list after it becomes `<steps>` |
| `{.result}` | Verification | `<result>` |
| `{.postreq}` | Next steps | `<postreq>` |
| `{.tasktroubleshooting}` | | `<tasktroubleshooting>` |

Titles are configurable through `implicit_task_sections`. They apply to `.md` and `.markdown` only; in MDITA the feature is off, so `## Prerequisites` in a `.mdita` file is a plain `<section>`.

A list nested in a step is `<substeps>`, whether ordered or unordered. `{.choices}` and `{.choicetable}` select those elements; without them a table in a step stays a plain `<table>`.

Plug-in versions: a `$schema`-typed task needs 6.1.0, admonitions in one need 6.2.0, and `format="markdown"` gets the section titles from 6.3.0.

### Links and keys

A fragment passes through to `@href`, so `file.md#topic-id` and `file.md#topic-id/element-id` both work. The topic ID is the YAML `id`, or the one derived from the title.

An href resolves against the source file's directory only. There is no same-name fallback, because the build has none either.

Keys come from reference-style link definitions in a map, and nowhere else:

```markdown
[install-guide]: install.md "Installation Guide"
```

Topics use them as `[the guide][install-guide]`, or the collapsed `[install-guide]`. A label the topic defines locally wins, becoming a plain `<xref href>`.

Conrefs and inline keyrefs use the HDITA attributes:

```html
<p data-conref="shared.md#topic/warning-para">fallback</p>
<span data-conkeyref="warnings/disk-full">fallback</span>
<span data-keyref="product-name">fallback</span>
```

### Formatting

Pipe tables normalize to one space inside each cell, delimiter rows collapse to `---` keeping any alignment colons, and no column grows to a common width. An escaped pipe and a pipe in a code span stay cell text, and a pipe block that is not a table keeps its formatting.

Two trailing spaces are a hard line break, which the plug-in renders as `<?linebreak?>`, so the formatter keeps them.

## MDITA map format

A map is a `.mditamap` file, or a `.md` file declaring `$schema: urn:oasis:names:tc:dita:xsd:map.xsd`. Both use nested markdown lists.

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

H1 becomes `<title>` and front matter becomes `<topicmeta>`. A list item with a link becomes `<topicref>` with `@href`, `@format`, and a `<navtitle>` from the link text. Nesting becomes nested `<topicref>`. A reference-style definition becomes a `<keydef>`. A link to another map is an ordinary `<topicref>` with `@format`, never a `<mapref>`.

Three constructs differ by format, because the `mditamap` reader enables fewer extensions:

| Construct | `.md` map | `.mditamap` |
|-----------|-----------|-------------|
| List item without a link | `<topichead>` | `<topicref>` with `<navtitle>` |
| Ordered list item | `<topicref collection-type="sequence">` | plain `<topicref>` |
| Pipe table | `<reltable>` | not parsed as a table |

## Diagnostics

| Code | Message |
|------|---------|
| 1 | Ambiguous link |
| 2 | Broken link |
| 3 | Non-breaking whitespace in heading |
| 4 | Retired. Front matter is optional in every format the plug-in reads, so this fired on every markdown file and reported no problem |
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
| Rename | Heading rename with prepare support, repointing every fragment link to that heading |
| Code actions | Create missing files, add front matter, add to map, add task sections, fix NBSP/footnotes/heading hierarchy, build with DITA-OT |
| Code lens | Reference counts on headings |
| Document links | External URL detection |
| Document symbols | Hierarchical heading outline |
| Workspace symbols | Cross-document heading search |
| Folding ranges | Headings, YAML front matter |
| Selection ranges | Progressive expansion (line, element, section) |
| Linked editing | Heading text |
| Formatting | Table normalization, trailing whitespace, heading spacing, trailing newline (full + range). Keeps hard line breaks, treats escaped and code-span pipes as cell text, and skips a pipe block that is not a table |
| Inlay hints | Link targets, keyref targets, conref targets |
| Document highlight | Heading and intra-document reference highlighting |
| Semantic tokens | Full + range encoding with attribute decorator tokens |
| Pull diagnostics | `textDocument/diagnostic` (LSP 3.17) |
| File operations | didCreate, didDelete, willCreate, willRename |
| Execute commands | `createFile`, `addToMap`, `ditaOtBuild` |
| Will save | `textDocument/willSaveWaitUntil` for table auto-format; it touches table lines only |

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

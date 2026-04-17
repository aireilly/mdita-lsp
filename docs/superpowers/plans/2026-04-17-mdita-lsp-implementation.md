# mdita-lsp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go LSP server for MDITA documents with full editing support, MDITA compliance diagnostics, keyref resolution, and ditamap validation.

**Architecture:** Feature-oriented packages under `internal/`, wired together by an LSP server package. Two-level document model (Elements + Symbols) built on goldmark. Symbol graph for cross-document resolution.

**Tech Stack:** Go, goldmark, go.lsp.dev/protocol, go.lsp.dev/jsonrpc2, gopkg.in/yaml.v3

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.gitignore`
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`
- Create: `cmd/mdita-lsp/main.go` (stub)

- [ ] **Step 1: Initialize Go module**

```bash
cd ~/mdita-lsp
go mod init github.com/aireilly/mdita-lsp
```

- [ ] **Step 2: Create Makefile**

Create `Makefile`:

```makefile
BINARY := mdita-lsp
PKG := github.com/aireilly/mdita-lsp
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint install clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/mdita-lsp

test:
	go test -race ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/mdita-lsp

clean:
	rm -f $(BINARY)
```

- [ ] **Step 3: Create .gitignore**

Create `.gitignore`:

```
mdita-lsp
*.exe
/dist/
```

- [ ] **Step 4: Create stub entry point**

Create `cmd/mdita-lsp/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, "mdita-lsp", version)
	os.Exit(1)
}
```

- [ ] **Step 5: Create CI workflow**

Create `.github/workflows/ci.yml`:

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: make test
      - run: make build
```

- [ ] **Step 6: Create release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  release:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - goos: linux
            goarch: amd64
          - goos: linux
            goarch: arm64
          - goos: darwin
            goarch: amd64
          - goos: darwin
            goarch: arm64
          - goos: windows
            goarch: amd64
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: '0'
        run: |
          EXT=""
          if [ "$GOOS" = "windows" ]; then EXT=".exe"; fi
          go build -ldflags "-X main.version=${{ github.ref_name }}" \
            -o "mdita-lsp-${{ matrix.goos }}-${{ matrix.goarch }}${EXT}" \
            ./cmd/mdita-lsp
      - name: Upload
        uses: softprops/action-gh-release@v2
        with:
          files: mdita-lsp-*
          generate_release_notes: true
```

- [ ] **Step 7: Verify build**

```bash
cd ~/mdita-lsp && go build ./cmd/mdita-lsp && ./mdita-lsp --version
```

Expected: prints `dev`

- [ ] **Step 8: Commit**

```bash
cd ~/mdita-lsp
git add go.mod Makefile .gitignore cmd/ .github/
git commit -m "feat: project scaffolding with build, CI, and release"
```

---

### Task 2: Paths Package

**Files:**
- Create: `internal/paths/paths.go`
- Create: `internal/paths/slug.go`
- Create: `internal/paths/paths_test.go`
- Create: `internal/paths/slug_test.go`

- [ ] **Step 1: Write slug tests**

Create `internal/paths/slug_test.go`:

```go
package paths

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Hello  World", "hello-world"},
		{"Hello - World", "hello-world"},
		{"Hello's World!", "hellos-world"},
		{"  spaces  ", "spaces"},
		{"UPPER CASE", "upper-case"},
		{"already-slug", "already-slug"},
		{"", ""},
		{"a", "a"},
		{"Hello #1 World", "hello-1-world"},
		{"Héllo Wörld", "héllo-wörld"},
		{"foo---bar", "foo-bar"},
		{"!@#start", "start"},
		{"end!@#", "end"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSlugIsSubstring(t *testing.T) {
	tests := []struct {
		haystack string
		needle   string
		want     bool
	}{
		{"hello-world", "hello", true},
		{"hello-world", "world", true},
		{"hello-world", "hello-world", true},
		{"hello-world", "xyz", false},
		{"hello-world", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.haystack+"_"+tt.needle, func(t *testing.T) {
			h := Slug(tt.haystack)
			n := Slug(tt.needle)
			if got := h.Contains(n); got != tt.want {
				t.Errorf("Slug(%q).Contains(%q) = %v, want %v", tt.haystack, tt.needle, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run slug tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/paths/...
```

Expected: FAIL — `Slugify` not defined

- [ ] **Step 3: Implement slug**

Create `internal/paths/slug.go`:

```go
package paths

import (
	"strings"
	"unicode"
)

type Slug string

func Slugify(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	var sepSeen bool
	var chunkState int // 0=none, 1=in-progress, 2=finished

	for _, ch := range s {
		isPunct := unicode.IsPunct(ch) || unicode.IsSymbol(ch)
		isSep := unicode.IsSpace(ch) || ch == '-'
		isOut := !isPunct && !isSep

		if isSep {
			sepSeen = true
		}

		if isOut {
			if sepSeen && chunkState == 2 {
				b.WriteByte('-')
				sepSeen = false
			}
			chunkState = 1
			b.WriteRune(unicode.ToLower(ch))
		} else if chunkState == 1 {
			chunkState = 2
		}
	}
	return b.String()
}

func SlugOf(s string) Slug {
	return Slug(Slugify(s))
}

func (s Slug) String() string {
	return string(s)
}

func (s Slug) Contains(sub Slug) bool {
	return strings.Contains(string(s), string(sub))
}
```

- [ ] **Step 4: Run slug tests to verify they pass**

```bash
cd ~/mdita-lsp && go test ./internal/paths/...
```

Expected: PASS

- [ ] **Step 5: Write path utility tests**

Create `internal/paths/paths_test.go`:

```go
package paths

import (
	"net/url"
	"testing"
)

func TestURIToPath(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"file:///home/user/doc.md", "/home/user/doc.md"},
		{"file:///home/user/my%20doc.md", "/home/user/my doc.md"},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			got, err := URIToPath(tt.uri)
			if err != nil {
				t.Fatalf("URIToPath(%q) error: %v", tt.uri, err)
			}
			if got != tt.want {
				t.Errorf("URIToPath(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestPathToURI(t *testing.T) {
	got := PathToURI("/home/user/doc.md")
	want := "file:///home/user/doc.md"
	if got != want {
		t.Errorf("PathToURI = %q, want %q", got, want)
	}
}

func TestRelPath(t *testing.T) {
	got := RelPath("/home/user/project", "/home/user/project/docs/file.md")
	if got != "docs/file.md" {
		t.Errorf("RelPath = %q, want %q", got, "docs/file.md")
	}
}

func TestDocIDFromURI(t *testing.T) {
	id := DocIDFromURI("file:///home/user/project/docs/intro.md", "file:///home/user/project")
	if id.RelPath != "docs/intro.md" {
		t.Errorf("DocID.RelPath = %q, want %q", id.RelPath, "docs/intro.md")
	}
	if id.Stem != "intro" {
		t.Errorf("DocID.Stem = %q, want %q", id.Stem, "intro")
	}
}

func TestIsMditaMapFile(t *testing.T) {
	tests := []struct {
		path string
		exts []string
		want bool
	}{
		{"foo.mditamap", []string{"mditamap"}, true},
		{"foo.md", []string{"mditamap"}, false},
		{"foo.ditamap", []string{"mditamap", "ditamap"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsMditaMapFile(tt.path, tt.exts); got != tt.want {
				t.Errorf("IsMditaMapFile(%q, %v) = %v, want %v", tt.path, tt.exts, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 6: Implement path utilities**

Create `internal/paths/paths.go`:

```go
package paths

import (
	"net/url"
	"path/filepath"
	"strings"
)

type DocID struct {
	URI     string
	RelPath string
	Stem    string
	Slug    Slug
}

func URIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	decoded, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func PathToURI(path string) string {
	return "file://" + path
}

func RelPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

func DocIDFromURI(uri, rootURI string) DocID {
	path, _ := URIToPath(uri)
	rootPath, _ := URIToPath(rootURI)
	rel := RelPath(rootPath, path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return DocID{
		URI:     uri,
		RelPath: rel,
		Stem:    stem,
		Slug:    SlugOf(stem),
	}
}

func IsMditaMapFile(path string, mapExts []string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	for _, me := range mapExts {
		if strings.EqualFold(ext, me) {
			return true
		}
	}
	return false
}

func IsMarkdownFile(path string, mdExts []string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	for _, me := range mdExts {
		if strings.EqualFold(ext, me) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 7: Run all paths tests**

```bash
cd ~/mdita-lsp && go test ./internal/paths/...
```

Expected: PASS

- [ ] **Step 8: Commit**

```bash
cd ~/mdita-lsp
git add internal/paths/
git commit -m "feat: paths package with slug generation and URI utilities"
```

---

### Task 3: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

Create `internal/config/config_test.go`:

```go
package config

import (
	"testing"
)

func TestParseYAML(t *testing.T) {
	input := `
core:
  markdown:
    file_extensions: [md, markdown, mditamap]
    text_sync: incremental
    title_from_heading: false
  mdita:
    enable: true
    map_extensions: [mditamap, ditamap]
completion:
  wiki_style: file-stem
  max_candidates: 100
code_actions:
  toc:
    enable: false
    include_levels: [1, 2, 3]
  create_missing_file:
    enable: true
diagnostics:
  mdita_compliance: true
  ditamap_validation: false
  keyref_resolution: true
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if cfg.Core.Markdown.TextSync != "incremental" {
		t.Errorf("TextSync = %q, want %q", cfg.Core.Markdown.TextSync, "incremental")
	}
	if cfg.Core.Markdown.TitleFromHeading != false {
		t.Errorf("TitleFromHeading = %v, want false", cfg.Core.Markdown.TitleFromHeading)
	}
	if cfg.Completion.WikiStyle != "file-stem" {
		t.Errorf("WikiStyle = %q, want %q", cfg.Completion.WikiStyle, "file-stem")
	}
	if cfg.Completion.MaxCandidates != 100 {
		t.Errorf("MaxCandidates = %d, want 100", cfg.Completion.MaxCandidates)
	}
	if cfg.CodeActions.ToC.Enable != false {
		t.Errorf("ToC.Enable = %v, want false", cfg.CodeActions.ToC.Enable)
	}
	exts := cfg.Core.Mdita.MapExtensions
	if len(exts) != 2 || exts[0] != "mditamap" || exts[1] != "ditamap" {
		t.Errorf("MapExtensions = %v, want [mditamap ditamap]", exts)
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Core.Markdown.TextSync != "full" {
		t.Errorf("default TextSync = %q, want %q", cfg.Core.Markdown.TextSync, "full")
	}
	if cfg.Core.Mdita.Enable != true {
		t.Errorf("default Mdita.Enable = %v, want true", cfg.Core.Mdita.Enable)
	}
	if cfg.Completion.MaxCandidates != 50 {
		t.Errorf("default MaxCandidates = %d, want 50", cfg.Completion.MaxCandidates)
	}
	if len(cfg.CodeActions.ToC.IncludeLevels) != 6 {
		t.Errorf("default IncludeLevels len = %d, want 6", len(cfg.CodeActions.ToC.IncludeLevels))
	}
}

func TestMerge(t *testing.T) {
	base := Default()
	overlay, _ := Parse([]byte(`
completion:
  wiki_style: file-path-stem
  max_candidates: 25
`))
	merged := Merge(base, overlay)
	if merged.Completion.WikiStyle != "file-path-stem" {
		t.Errorf("merged WikiStyle = %q, want %q", merged.Completion.WikiStyle, "file-path-stem")
	}
	if merged.Completion.MaxCandidates != 25 {
		t.Errorf("merged MaxCandidates = %d, want 25", merged.Completion.MaxCandidates)
	}
	if merged.Core.Mdita.Enable != true {
		t.Errorf("merged Mdita.Enable = %v, want true (from default)", merged.Core.Mdita.Enable)
	}
}

func TestParseEmpty(t *testing.T) {
	cfg, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("Parse empty error: %v", err)
	}
	if cfg.Core.Markdown.TextSync != "" {
		t.Errorf("empty parse should have zero values, got TextSync = %q", cfg.Core.Markdown.TextSync)
	}
}
```

- [ ] **Step 2: Run config tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/config/...
```

Expected: FAIL — `Parse` not defined

- [ ] **Step 3: Implement config**

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Core        CoreConfig        `yaml:"core"`
	Completion  CompletionConfig  `yaml:"completion"`
	CodeActions CodeActionsConfig `yaml:"code_actions"`
	Diagnostics DiagnosticsConfig `yaml:"diagnostics"`
}

type CoreConfig struct {
	Markdown MarkdownConfig `yaml:"markdown"`
	Mdita    MditaConfig    `yaml:"mdita"`
}

type MarkdownConfig struct {
	FileExtensions   []string `yaml:"file_extensions"`
	TextSync         string   `yaml:"text_sync"`
	TitleFromHeading bool     `yaml:"title_from_heading"`
}

type MditaConfig struct {
	Enable        bool     `yaml:"enable"`
	MapExtensions []string `yaml:"map_extensions"`
}

type CompletionConfig struct {
	WikiStyle     string `yaml:"wiki_style"`
	MaxCandidates int    `yaml:"max_candidates"`
}

type CodeActionsConfig struct {
	ToC               ToCConfig               `yaml:"toc"`
	CreateMissingFile CreateMissingFileConfig `yaml:"create_missing_file"`
}

type ToCConfig struct {
	Enable        bool  `yaml:"enable"`
	IncludeLevels []int `yaml:"include_levels"`
}

type CreateMissingFileConfig struct {
	Enable bool `yaml:"enable"`
}

type DiagnosticsConfig struct {
	MditaCompliance   bool `yaml:"mdita_compliance"`
	DitamapValidation bool `yaml:"ditamap_validation"`
	KeyrefResolution  bool `yaml:"keyref_resolution"`
}

func Default() *Config {
	return &Config{
		Core: CoreConfig{
			Markdown: MarkdownConfig{
				FileExtensions:   []string{"md", "markdown", "mditamap"},
				TextSync:         "full",
				TitleFromHeading: true,
			},
			Mdita: MditaConfig{
				Enable:        true,
				MapExtensions: []string{"mditamap"},
			},
		},
		Completion: CompletionConfig{
			WikiStyle:     "title-slug",
			MaxCandidates: 50,
		},
		CodeActions: CodeActionsConfig{
			ToC: ToCConfig{
				Enable:        true,
				IncludeLevels: []int{1, 2, 3, 4, 5, 6},
			},
			CreateMissingFile: CreateMissingFileConfig{
				Enable: true,
			},
		},
		Diagnostics: DiagnosticsConfig{
			MditaCompliance:   true,
			DitamapValidation: true,
			KeyrefResolution:  true,
		},
	}
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	if len(data) == 0 {
		return &cfg, nil
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	return Parse(data)
}

func Merge(base, overlay *Config) *Config {
	merged := *base

	if overlay.Core.Markdown.TextSync != "" {
		merged.Core.Markdown.TextSync = overlay.Core.Markdown.TextSync
	}
	if overlay.Core.Markdown.FileExtensions != nil {
		merged.Core.Markdown.FileExtensions = overlay.Core.Markdown.FileExtensions
	}
	if overlay.Core.Markdown.TitleFromHeading != base.Core.Markdown.TitleFromHeading {
		merged.Core.Markdown.TitleFromHeading = overlay.Core.Markdown.TitleFromHeading
	}
	if overlay.Core.Mdita.MapExtensions != nil {
		merged.Core.Mdita.MapExtensions = overlay.Core.Mdita.MapExtensions
	}

	if overlay.Completion.WikiStyle != "" {
		merged.Completion.WikiStyle = overlay.Completion.WikiStyle
	}
	if overlay.Completion.MaxCandidates != 0 {
		merged.Completion.MaxCandidates = overlay.Completion.MaxCandidates
	}

	if overlay.CodeActions.ToC.IncludeLevels != nil {
		merged.CodeActions.ToC.IncludeLevels = overlay.CodeActions.ToC.IncludeLevels
	}

	return &merged
}

func LoadMerged(folderRoot string) *Config {
	cfg := Default()

	home, err := os.UserHomeDir()
	if err == nil {
		userCfg, err := Load(filepath.Join(home, ".config", "mdita-lsp", "config.yaml"))
		if err == nil {
			cfg = Merge(cfg, userCfg)
		}
	}

	folderCfg, err := Load(filepath.Join(folderRoot, ".mdita-lsp.yaml"))
	if err == nil {
		cfg = Merge(cfg, folderCfg)
	}

	return cfg
}
```

- [ ] **Step 4: Add yaml.v3 dependency and run tests**

```bash
cd ~/mdita-lsp && go get gopkg.in/yaml.v3 && go test ./internal/config/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/config/ go.mod go.sum
git commit -m "feat: config package with YAML loading, defaults, and merging"
```

---

### Task 4: Document Types

**Files:**
- Create: `internal/document/types.go`
- Create: `internal/document/types_test.go`

- [ ] **Step 1: Write element type tests**

Create `internal/document/types_test.go`:

```go
package document

import "testing"

func TestHeadingIsTitle(t *testing.T) {
	h := &Heading{Level: 1, Text: "My Title", ID: "my-title"}
	if !h.IsTitle() {
		t.Error("Level 1 heading should be a title")
	}
	h2 := &Heading{Level: 2, Text: "Section", ID: "section"}
	if h2.IsTitle() {
		t.Error("Level 2 heading should not be a title")
	}
}

func TestDocKind(t *testing.T) {
	if Topic.String() != "topic" {
		t.Errorf("Topic.String() = %q, want %q", Topic.String(), "topic")
	}
	if Map.String() != "map" {
		t.Errorf("Map.String() = %q, want %q", Map.String(), "map")
	}
}

func TestDitaSchemaFromString(t *testing.T) {
	tests := []struct {
		input string
		want  DitaSchema
	}{
		{"urn:oasis:names:tc:dita:xsd:task.xsd", SchemaTask},
		{"urn:oasis:names:tc:dita:rng:task.rng", SchemaTask},
		{"urn:oasis:names:tc:dita:xsd:concept.xsd", SchemaConcept},
		{"urn:oasis:names:tc:dita:rng:concept.rng", SchemaConcept},
		{"urn:oasis:names:tc:dita:xsd:reference.xsd", SchemaReference},
		{"urn:oasis:names:tc:dita:xsd:topic.xsd", SchemaTopic},
		{"urn:oasis:names:tc:dita:xsd:map.xsd", SchemaMap},
		{"urn:oasis:names:tc:mdita:xsd:topic.xsd", SchemaMditaTopic},
		{"urn:oasis:names:tc:mdita:core:xsd:topic.xsd", SchemaMditaCoreTopic},
		{"urn:oasis:names:tc:mdita:extended:xsd:topic.xsd", SchemaMditaExtendedTopic},
		{"something-unknown", SchemaUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DitaSchemaFromString(tt.input)
			if got != tt.want {
				t.Errorf("DitaSchemaFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSymKind(t *testing.T) {
	if DefKind.String() != "def" {
		t.Errorf("DefKind.String() = %q", DefKind.String())
	}
	if RefKind.String() != "ref" {
		t.Errorf("RefKind.String() = %q", RefKind.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/document/...
```

Expected: FAIL

- [ ] **Step 3: Implement document types**

Create `internal/document/types.go`:

```go
package document

import "github.com/aireilly/mdita-lsp/internal/paths"

type Range struct {
	Start Position
	End   Position
}

type Position struct {
	Line      int
	Character int
}

func Rng(sl, sc, el, ec int) Range {
	return Range{Start: Position{Line: sl, Character: sc}, End: Position{Line: el, Character: ec}}
}

type DocKind int

const (
	Topic DocKind = iota
	Map
)

func (k DocKind) String() string {
	switch k {
	case Topic:
		return "topic"
	case Map:
		return "map"
	default:
		return "unknown"
	}
}

type DitaSchema int

const (
	SchemaTopic DitaSchema = iota
	SchemaConcept
	SchemaTask
	SchemaReference
	SchemaMap
	SchemaMditaTopic
	SchemaMditaCoreTopic
	SchemaMditaExtendedTopic
	SchemaUnknown
)

func DitaSchemaFromString(s string) DitaSchema {
	switch s {
	case "urn:oasis:names:tc:dita:xsd:topic.xsd",
		"urn:oasis:names:tc:dita:rng:topic.rng":
		return SchemaTopic
	case "urn:oasis:names:tc:dita:xsd:concept.xsd",
		"urn:oasis:names:tc:dita:rng:concept.rng":
		return SchemaConcept
	case "urn:oasis:names:tc:dita:xsd:task.xsd",
		"urn:oasis:names:tc:dita:rng:task.rng":
		return SchemaTask
	case "urn:oasis:names:tc:dita:xsd:reference.xsd",
		"urn:oasis:names:tc:dita:rng:reference.rng":
		return SchemaReference
	case "urn:oasis:names:tc:dita:xsd:map.xsd",
		"urn:oasis:names:tc:dita:rng:map.rng":
		return SchemaMap
	case "urn:oasis:names:tc:mdita:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:rng:topic.rng":
		return SchemaMditaTopic
	case "urn:oasis:names:tc:mdita:core:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:core:rng:topic.rng":
		return SchemaMditaCoreTopic
	case "urn:oasis:names:tc:mdita:extended:xsd:topic.xsd",
		"urn:oasis:names:tc:mdita:extended:rng:topic.rng":
		return SchemaMditaExtendedTopic
	default:
		return SchemaUnknown
	}
}

type Element interface {
	Rng() Range
	element()
}

type Heading struct {
	Level int
	Text  string
	ID    string
	Slug  paths.Slug
	Range Range
}

func (h *Heading) Rng() Range { return h.Range }
func (h *Heading) element()   {}
func (h *Heading) IsTitle() bool { return h.Level == 1 }

type WikiLink struct {
	Doc     string
	Heading string
	Title   string
	Range   Range
}

func (w *WikiLink) Rng() Range { return w.Range }
func (w *WikiLink) element()   {}

type MdLink struct {
	Text   string
	URL    string
	Anchor string
	IsRef  bool
	Range  Range
}

func (m *MdLink) Rng() Range { return m.Range }
func (m *MdLink) element()   {}

type LinkDef struct {
	Label string
	URL   string
	Range Range
}

func (l *LinkDef) Rng() Range { return l.Range }
func (l *LinkDef) element()   {}

type YAMLMetadata struct {
	Author      string
	Source      string
	Publisher   string
	Permissions string
	Audience    string
	Category    string
	Keywords    []string
	ResourceID  string
	Schema      DitaSchema
	SchemaRaw   string
	OtherMeta   map[string]string
	Range       Range
}

type BlockFeatures struct {
	HasOrderedList    bool
	HasUnorderedList  bool
	HasTable          bool
	HasDefinitionList bool
	HasFootnoteRefs   bool
	HasFootnoteDefs   bool
	HasStrikethrough  bool
	HasAttributes     bool
	Admonitions       []Admonition
}

type Admonition struct {
	Type  string
	Range Range
}

type SymKind int

const (
	DefKind SymKind = iota
	RefKind
)

func (k SymKind) String() string {
	if k == DefKind {
		return "def"
	}
	return "ref"
}

type DefType int

const (
	DefDoc DefType = iota
	DefTitle
	DefHeading
	DefLinkDef
)

type RefType int

const (
	RefWikiLink RefType = iota
	RefMdLink
	RefKeyref
)

type Symbol struct {
	Kind    SymKind
	DefType DefType
	RefType RefType
	Name    string
	Slug    paths.Slug
	DocURI  string
	Range   Range
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/document/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/document/
git commit -m "feat: document types — elements, symbols, DITA schemas"
```

---

### Task 5: Document Parser (goldmark + wiki link extension)

**Files:**
- Create: `internal/document/wikilink_ext.go`
- Create: `internal/document/parser.go`
- Create: `internal/document/parser_test.go`

- [ ] **Step 1: Write parser tests**

Create `internal/document/parser_test.go`:

```go
package document

import (
	"testing"
)

func TestParseHeadings(t *testing.T) {
	text := "# Title\n\n## Section One\n\n### Subsection\n"
	elements, _, _ := Parse(text)

	headings := filterHeadings(elements)
	if len(headings) != 3 {
		t.Fatalf("got %d headings, want 3", len(headings))
	}
	if headings[0].Level != 1 || headings[0].Text != "Title" {
		t.Errorf("heading[0] = {%d, %q}, want {1, \"Title\"}", headings[0].Level, headings[0].Text)
	}
	if headings[1].Level != 2 || headings[1].Text != "Section One" {
		t.Errorf("heading[1] = {%d, %q}, want {2, \"Section One\"}", headings[1].Level, headings[1].Text)
	}
	if headings[2].Level != 3 || headings[2].Text != "Subsection" {
		t.Errorf("heading[2] = {%d, %q}, want {3, \"Subsection\"}", headings[2].Level, headings[2].Text)
	}
}

func TestParseWikiLinks(t *testing.T) {
	text := "# Title\n\n[[other-doc]]\n\n[[doc#heading]]\n\n[[doc#heading|display title]]\n\n[[#local-heading]]\n"
	elements, _, _ := Parse(text)

	wikiLinks := filterWikiLinks(elements)
	if len(wikiLinks) != 4 {
		t.Fatalf("got %d wiki links, want 4", len(wikiLinks))
	}
	if wikiLinks[0].Doc != "other-doc" || wikiLinks[0].Heading != "" {
		t.Errorf("wl[0] = {%q, %q}, want {\"other-doc\", \"\"}", wikiLinks[0].Doc, wikiLinks[0].Heading)
	}
	if wikiLinks[1].Doc != "doc" || wikiLinks[1].Heading != "heading" {
		t.Errorf("wl[1] = {%q, %q}", wikiLinks[1].Doc, wikiLinks[1].Heading)
	}
	if wikiLinks[2].Doc != "doc" || wikiLinks[2].Heading != "heading" || wikiLinks[2].Title != "display title" {
		t.Errorf("wl[2] = {%q, %q, %q}", wikiLinks[2].Doc, wikiLinks[2].Heading, wikiLinks[2].Title)
	}
	if wikiLinks[3].Doc != "" || wikiLinks[3].Heading != "local-heading" {
		t.Errorf("wl[3] = {%q, %q}", wikiLinks[3].Doc, wikiLinks[3].Heading)
	}
}

func TestParseMdLinks(t *testing.T) {
	text := "# Title\n\n[link text](other.md)\n\n[anchor](other.md#section)\n\n[ref][label]\n"
	elements, _, _ := Parse(text)

	mdLinks := filterMdLinks(elements)
	if len(mdLinks) < 2 {
		t.Fatalf("got %d md links, want at least 2", len(mdLinks))
	}
	if mdLinks[0].Text != "link text" || mdLinks[0].URL != "other.md" {
		t.Errorf("ml[0] = {%q, %q}", mdLinks[0].Text, mdLinks[0].URL)
	}
	if mdLinks[1].URL != "other.md" || mdLinks[1].Anchor != "section" {
		t.Errorf("ml[1] = {%q, %q}", mdLinks[1].URL, mdLinks[1].Anchor)
	}
}

func TestParseYAMLFrontMatter(t *testing.T) {
	text := "---\nauthor: John\n$schema: urn:oasis:names:tc:dita:xsd:task.xsd\nkeyword: [go, lsp]\n---\n# Title\n"
	elements, _, meta := Parse(text)

	if meta == nil {
		t.Fatal("expected YAML metadata, got nil")
	}
	if meta.Author != "John" {
		t.Errorf("Author = %q, want %q", meta.Author, "John")
	}
	if meta.Schema != SchemaTask {
		t.Errorf("Schema = %v, want SchemaTask", meta.Schema)
	}
	if len(meta.Keywords) != 2 || meta.Keywords[0] != "go" {
		t.Errorf("Keywords = %v, want [go lsp]", meta.Keywords)
	}

	_ = elements
}

func TestParseLinkDefs(t *testing.T) {
	text := "# Title\n\n[label]: https://example.com\n"
	elements, _, _ := Parse(text)

	linkDefs := filterLinkDefs(elements)
	if len(linkDefs) != 1 {
		t.Fatalf("got %d link defs, want 1", len(linkDefs))
	}
	if linkDefs[0].Label != "label" || linkDefs[0].URL != "https://example.com" {
		t.Errorf("ld[0] = {%q, %q}", linkDefs[0].Label, linkDefs[0].URL)
	}
}

func TestParseBlockFeatures(t *testing.T) {
	text := "# Title\n\n1. step one\n2. step two\n\n| col1 | col2 |\n|------|------|\n| a | b |\n"
	_, bf, _ := Parse(text)

	if !bf.HasOrderedList {
		t.Error("expected HasOrderedList = true")
	}
	if !bf.HasTable {
		t.Error("expected HasTable = true")
	}
}

func TestParseAdmonitions(t *testing.T) {
	text := "# Title\n\n!!! note\n    This is a note.\n"
	_, bf, _ := Parse(text)

	if len(bf.Admonitions) != 1 {
		t.Fatalf("got %d admonitions, want 1", len(bf.Admonitions))
	}
	if bf.Admonitions[0].Type != "note" {
		t.Errorf("admonition type = %q, want %q", bf.Admonitions[0].Type, "note")
	}
}

func filterHeadings(elems []Element) []*Heading {
	var result []*Heading
	for _, e := range elems {
		if h, ok := e.(*Heading); ok {
			result = append(result, h)
		}
	}
	return result
}

func filterWikiLinks(elems []Element) []*WikiLink {
	var result []*WikiLink
	for _, e := range elems {
		if w, ok := e.(*WikiLink); ok {
			result = append(result, w)
		}
	}
	return result
}

func filterMdLinks(elems []Element) []*MdLink {
	var result []*MdLink
	for _, e := range elems {
		if m, ok := e.(*MdLink); ok {
			result = append(result, m)
		}
	}
	return result
}

func filterLinkDefs(elems []Element) []*LinkDef {
	var result []*LinkDef
	for _, e := range elems {
		if l, ok := e.(*LinkDef); ok {
			result = append(result, l)
		}
	}
	return result
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/document/...
```

Expected: FAIL — `Parse` not defined

- [ ] **Step 3: Implement wiki link goldmark extension**

Create `internal/document/wikilink_ext.go`:

```go
package document

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type wikiLinkNode struct {
	ast.BaseInline
	Doc     string
	Heading string
	Title   string
}

var kindWikiLink = ast.NewNodeKind("WikiLink")

func (n *wikiLinkNode) Kind() ast.NodeKind { return kindWikiLink }

func (n *wikiLinkNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type wikiLinkParser struct{}

func (p *wikiLinkParser) Trigger() []byte {
	return []byte{'['}
}

func (p *wikiLinkParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, seg := block.PeekLine()
	if len(line) < 2 || line[0] != '[' || line[1] != '[' {
		return nil
	}

	end := -1
	for i := 2; i < len(line)-1; i++ {
		if line[i] == ']' && line[i+1] == ']' {
			end = i
			break
		}
		if line[i] == '\n' {
			return nil
		}
	}
	if end < 0 {
		return nil
	}

	content := string(line[2:end])
	node := &wikiLinkNode{}

	docPart := content
	if idx := indexOf(content, '|'); idx >= 0 {
		node.Title = content[idx+1:]
		docPart = content[:idx]
	}
	if idx := indexOf(docPart, '#'); idx >= 0 {
		node.Heading = docPart[idx+1:]
		node.Doc = docPart[:idx]
	} else {
		node.Doc = docPart
	}

	consumed := end + 2
	block.Advance(consumed)
	node.SetLines(text.NewSegments())
	s := seg
	s = s.WithStop(s.Start + consumed)
	return node
}

func indexOf(s string, ch byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			return i
		}
	}
	return -1
}

type wikiLinkExtension struct{}

func (e *wikiLinkExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			parser.NewInlineParser(&wikiLinkParser{}, 199),
		),
	)
}
```

- [ ] **Step 4: Implement parser**

Create `internal/document/parser.go`:

```go
package document

import (
	"regexp"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

var admonitionRegex = regexp.MustCompile(`(?m)^!!!\s+(\w+)`)

func Parse(source string) ([]Element, *BlockFeatures, *YAMLMetadata) {
	src := []byte(source)

	var meta *YAMLMetadata
	mdContent := source
	yamlEnd := 0

	if strings.HasPrefix(source, "---\n") || strings.HasPrefix(source, "---\r\n") {
		closeIdx := strings.Index(source[4:], "\n---")
		if closeIdx < 0 {
			closeIdx = strings.Index(source[4:], "\n...")
		}
		if closeIdx >= 0 {
			yamlBlock := source[4 : 4+closeIdx]
			meta = parseYAMLMeta(yamlBlock)
			yamlEnd = 4 + closeIdx + 4
			if yamlEnd < len(source) && source[yamlEnd-1] == '\r' {
				yamlEnd++
			}
		}
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			&wikiLinkExtension{},
			extension.NewTable(),
			extension.Strikethrough,
			extension.DefinitionList,
			extension.Footnote,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	var elements []Element
	bf := &BlockFeatures{}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			headingText := extractText(node, src)
			id := string(node.Attribute("id"))
			if id == "" {
				id = paths.Slugify(headingText)
			}
			elements = append(elements, &Heading{
				Level: node.Level,
				Text:  headingText,
				ID:    id,
				Slug:  paths.SlugOf(headingText),
				Range: nodeRange(node, src),
			})

		case *wikiLinkNode:
			elements = append(elements, &WikiLink{
				Doc:     node.Doc,
				Heading: node.Heading,
				Title:   node.Title,
				Range:   nodeRange(node, src),
			})

		case *ast.Link:
			url := string(node.Destination)
			linkText := extractText(node, src)
			anchor := ""
			if idx := strings.Index(url, "#"); idx >= 0 {
				anchor = url[idx+1:]
				url = url[:idx]
			}
			if !isExternalURL(url) {
				elements = append(elements, &MdLink{
					Text:   linkText,
					URL:    url,
					Anchor: anchor,
					Range:  nodeRange(node, src),
				})
			}

		case *ast.LinkReferenceDefinition:
			// goldmark doesn't expose this as a separate node type by default
			// we handle it below via paragraph scanning

		case *ast.List:
			if node.IsOrdered() {
				bf.HasOrderedList = true
			} else {
				bf.HasUnorderedList = true
			}

		case *ast.Table:
			bf.HasTable = true

		case *gmast.Strikethrough:
			bf.HasStrikethrough = true

		case *gmast.DefinitionList:
			bf.HasDefinitionList = true

		case *gmast.FootnoteLink:
			bf.HasFootnoteRefs = true

		case *gmast.FootnoteBacklink:
			bf.HasFootnoteDefs = true
		}

		return ast.WalkContinue, nil
	})

	elements = append(elements, parseLinkDefs(mdContent, yamlEnd)...)
	bf.Admonitions = parseAdmonitions(source)

	return elements, bf, meta
}

func parseYAMLMeta(yamlContent string) *YAMLMetadata {
	meta := &YAMLMetadata{OtherMeta: make(map[string]string)}

	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return meta
	}

	for key, val := range raw {
		sval := ""
		switch v := val.(type) {
		case string:
			sval = v
		}

		switch key {
		case "author":
			meta.Author = sval
		case "source":
			meta.Source = sval
		case "publisher":
			meta.Publisher = sval
		case "permissions":
			meta.Permissions = sval
		case "audience":
			meta.Audience = sval
		case "category":
			meta.Category = sval
		case "resourceid":
			meta.ResourceID = sval
		case "$schema":
			meta.SchemaRaw = sval
			meta.Schema = DitaSchemaFromString(sval)
		case "keyword":
			meta.Keywords = parseKeywords(val)
		default:
			if sval != "" {
				meta.OtherMeta[key] = sval
			}
		}
	}
	return meta
}

func parseKeywords(val interface{}) []string {
	switch v := val.(type) {
	case string:
		return []string{v}
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

var linkDefRegex = regexp.MustCompile(`(?m)^\[([^\]]+)\]:\s+(.+)$`)

func parseLinkDefs(source string, offset int) []Element {
	var elements []Element
	matches := linkDefRegex.FindAllStringSubmatchIndex(source[offset:], -1)
	for _, m := range matches {
		label := source[offset+m[2] : offset+m[3]]
		url := strings.TrimSpace(source[offset+m[4] : offset+m[5]])
		rng := rangeFromOffset(source, offset+m[0], offset+m[1])
		elements = append(elements, &LinkDef{
			Label: label,
			URL:   url,
			Range: rng,
		})
	}
	return elements
}

func parseAdmonitions(source string) []Admonition {
	var admonitions []Admonition
	matches := admonitionRegex.FindAllStringSubmatchIndex(source, -1)
	for _, m := range matches {
		adType := source[m[2]:m[3]]
		rng := rangeFromOffset(source, m[0], m[1])
		admonitions = append(admonitions, Admonition{
			Type:  strings.ToLower(adType),
			Range: rng,
		})
	}
	return admonitions
}

func extractText(n ast.Node, src []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			sb.Write(t.Segment.Value(src))
		} else if c.HasChildren() {
			sb.WriteString(extractText(c, src))
		}
	}
	return sb.String()
}

func nodeRange(n ast.Node, src []byte) Range {
	lines := n.Lines()
	if lines.Len() > 0 {
		first := lines.At(0)
		last := lines.At(lines.Len() - 1)
		return rangeFromOffset(string(src), first.Start, last.Stop)
	}
	if n.Type() == ast.TypeInline {
		parent := n.Parent()
		if parent != nil {
			return nodeRange(parent, src)
		}
	}
	return Range{}
}

func rangeFromOffset(source string, start, end int) Range {
	sl, sc := offsetToLineCol(source, start)
	el, ec := offsetToLineCol(source, end)
	return Rng(sl, sc, el, ec)
}

func offsetToLineCol(source string, offset int) (int, int) {
	line := 0
	col := 0
	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

func isExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "mailto:") ||
		strings.HasPrefix(url, "ftp://")
}
```

- [ ] **Step 5: Add goldmark dependency and run tests**

```bash
cd ~/mdita-lsp && go get github.com/yuin/goldmark && go test -v ./internal/document/...
```

Expected: PASS (or minor fixes needed for goldmark AST specifics)

- [ ] **Step 6: Fix any test failures and verify all pass**

```bash
cd ~/mdita-lsp && go test ./internal/document/...
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
cd ~/mdita-lsp
git add internal/document/ go.mod go.sum
git commit -m "feat: document parser with goldmark, wiki link extension, YAML front matter"
```

---

### Task 6: Document Index

**Files:**
- Create: `internal/document/index.go`
- Create: `internal/document/index_test.go`

- [ ] **Step 1: Write index tests**

Create `internal/document/index_test.go`:

```go
package document

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestIndexHeadingsBySlug(t *testing.T) {
	elements := []Element{
		&Heading{Level: 1, Text: "Title", Slug: paths.SlugOf("Title")},
		&Heading{Level: 2, Text: "Section One", Slug: paths.SlugOf("Section One")},
		&Heading{Level: 2, Text: "Section Two", Slug: paths.SlugOf("Section Two")},
	}
	idx := BuildIndex(elements, nil, nil)

	got := idx.HeadingsBySlug(paths.SlugOf("section-one"))
	if len(got) != 1 {
		t.Fatalf("HeadingsBySlug(section-one) returned %d, want 1", len(got))
	}
	if got[0].Text != "Section One" {
		t.Errorf("got heading %q", got[0].Text)
	}
}

func TestIndexTitle(t *testing.T) {
	elements := []Element{
		&Heading{Level: 1, Text: "My Doc Title", Slug: paths.SlugOf("My Doc Title")},
		&Heading{Level: 2, Text: "Section", Slug: paths.SlugOf("Section")},
	}
	idx := BuildIndex(elements, nil, nil)

	title := idx.Title()
	if title == nil || title.Text != "My Doc Title" {
		t.Errorf("Title() = %v, want heading 'My Doc Title'", title)
	}
}

func TestIndexNoTitle(t *testing.T) {
	elements := []Element{
		&Heading{Level: 2, Text: "Section", Slug: paths.SlugOf("Section")},
	}
	idx := BuildIndex(elements, nil, nil)

	if idx.Title() != nil {
		t.Error("expected nil title when no H1")
	}
}

func TestIndexWikiLinks(t *testing.T) {
	elements := []Element{
		&WikiLink{Doc: "other", Heading: "section"},
		&WikiLink{Doc: "another"},
	}
	idx := BuildIndex(elements, nil, nil)

	wl := idx.WikiLinks()
	if len(wl) != 2 {
		t.Errorf("WikiLinks() returned %d, want 2", len(wl))
	}
}

func TestIndexShortDescription(t *testing.T) {
	idx := BuildIndex(nil, nil, nil)
	idx.ShortDesc = "This is the short description."
	if idx.ShortDesc != "This is the short description." {
		t.Error("short description not set")
	}
}

func TestIndexAllHeadings(t *testing.T) {
	elements := []Element{
		&Heading{Level: 1, Text: "T", Slug: paths.SlugOf("T")},
		&Heading{Level: 2, Text: "S", Slug: paths.SlugOf("S")},
		&WikiLink{Doc: "x"},
	}
	idx := BuildIndex(elements, nil, nil)
	if len(idx.Headings()) != 2 {
		t.Errorf("Headings() returned %d, want 2", len(idx.Headings()))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/document/... -run TestIndex
```

Expected: FAIL

- [ ] **Step 3: Implement index**

Create `internal/document/index.go`:

```go
package document

import "github.com/aireilly/mdita-lsp/internal/paths"

type Index struct {
	headings     []*Heading
	headingSlug  map[paths.Slug][]*Heading
	wikiLinks    []*WikiLink
	mdLinks      []*MdLink
	linkDefs     []*LinkDef
	linkDefLabel map[string]*LinkDef
	Meta         *YAMLMetadata
	Features     *BlockFeatures
	ShortDesc    string
}

func BuildIndex(elements []Element, bf *BlockFeatures, meta *YAMLMetadata) *Index {
	idx := &Index{
		headingSlug:  make(map[paths.Slug][]*Heading),
		linkDefLabel: make(map[string]*LinkDef),
		Features:     bf,
		Meta:         meta,
	}
	if idx.Features == nil {
		idx.Features = &BlockFeatures{}
	}

	for _, e := range elements {
		switch el := e.(type) {
		case *Heading:
			idx.headings = append(idx.headings, el)
			idx.headingSlug[el.Slug] = append(idx.headingSlug[el.Slug], el)
		case *WikiLink:
			idx.wikiLinks = append(idx.wikiLinks, el)
		case *MdLink:
			idx.mdLinks = append(idx.mdLinks, el)
		case *LinkDef:
			idx.linkDefs = append(idx.linkDefs, el)
			idx.linkDefLabel[el.Label] = el
		}
	}
	return idx
}

func (idx *Index) Headings() []*Heading {
	return idx.headings
}

func (idx *Index) HeadingsBySlug(slug paths.Slug) []*Heading {
	return idx.headingSlug[slug]
}

func (idx *Index) Title() *Heading {
	for _, h := range idx.headings {
		if h.IsTitle() {
			return h
		}
	}
	return nil
}

func (idx *Index) WikiLinks() []*WikiLink {
	return idx.wikiLinks
}

func (idx *Index) MdLinks() []*MdLink {
	return idx.mdLinks
}

func (idx *Index) LinkDefs() []*LinkDef {
	return idx.linkDefs
}

func (idx *Index) LinkDefByLabel(label string) *LinkDef {
	return idx.linkDefLabel[label]
}

func (idx *Index) AllLinks() []Element {
	var links []Element
	for _, w := range idx.wikiLinks {
		links = append(links, w)
	}
	for _, m := range idx.mdLinks {
		links = append(links, m)
	}
	return links
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/document/... -run TestIndex
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/document/index.go internal/document/index_test.go
git commit -m "feat: document index with heading/link lookups"
```

---

### Task 7: Document Type and Text Operations

**Files:**
- Create: `internal/document/document.go`
- Create: `internal/document/document_test.go`

- [ ] **Step 1: Write document tests**

Create `internal/document/document_test.go`:

```go
package document

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestNewDocument(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "# Title\n\nSome text.\n")
	if doc.URI != "file:///test/doc.md" {
		t.Errorf("URI = %q", doc.URI)
	}
	if doc.Version != 1 {
		t.Errorf("Version = %d", doc.Version)
	}
	if doc.Kind != Topic {
		t.Errorf("Kind = %v, want Topic", doc.Kind)
	}
	title := doc.Index.Title()
	if title == nil || title.Text != "Title" {
		t.Errorf("Title = %v", title)
	}
}

func TestNewMapDocument(t *testing.T) {
	doc := New("file:///test/doc.mditamap", 1, "# My Map\n\n- [Topic](topic.md)\n")
	if doc.Kind != Map {
		t.Errorf("Kind = %v, want Map", doc.Kind)
	}
}

func TestDocumentLineMap(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "line 0\nline 1\nline 2\n")
	if len(doc.Lines) != 4 {
		t.Fatalf("Lines = %d, want 4", len(doc.Lines))
	}
	if doc.Lines[0] != 0 {
		t.Errorf("Lines[0] = %d, want 0", doc.Lines[0])
	}
	if doc.Lines[1] != 7 {
		t.Errorf("Lines[1] = %d, want 7", doc.Lines[1])
	}
}

func TestDocumentApplyFullChange(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "# Old\n")
	doc = doc.ApplyChange(2, "# New\n\nContent.\n")
	if doc.Version != 2 {
		t.Errorf("Version = %d, want 2", doc.Version)
	}
	title := doc.Index.Title()
	if title == nil || title.Text != "New" {
		t.Errorf("Title after change = %v", title)
	}
}

func TestDocumentSymbols(t *testing.T) {
	doc := New("file:///test/doc.md", 1, "# Title\n\n## Section\n\n[[other]]\n\n[link](foo.md)\n")
	defs := doc.Defs()
	refs := doc.Refs()

	if len(defs) < 2 {
		t.Errorf("Defs = %d, want >= 2 (doc + headings)", len(defs))
	}
	if len(refs) < 2 {
		t.Errorf("Refs = %d, want >= 2 (wiki link + md link)", len(refs))
	}
}

func TestDocIDFromDocument(t *testing.T) {
	doc := New("file:///project/docs/intro.md", 1, "# Intro\n")
	id := doc.DocID("file:///project")
	if id.Stem != "intro" {
		t.Errorf("Stem = %q", id.Stem)
	}
	if id.Slug != paths.Slug("intro") {
		t.Errorf("Slug = %q", id.Slug)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/document/... -run "TestNew|TestDoc"
```

Expected: FAIL

- [ ] **Step 3: Implement Document**

Create `internal/document/document.go`:

```go
package document

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/paths"
)

type Document struct {
	URI      string
	Version  int
	Text     string
	Lines    []int
	Elements []Element
	Symbols  []Symbol
	Index    *Index
	Meta     *YAMLMetadata
	Kind     DocKind
}

func New(uri string, version int, text string) *Document {
	elements, bf, meta := Parse(text)
	idx := BuildIndex(elements, bf, meta)
	idx.Meta = meta

	kind := Topic
	if paths.IsMditaMapFile(uri, []string{"mditamap"}) {
		kind = Map
	}

	title := idx.Title()
	if title != nil && idx.ShortDesc == "" {
		idx.ShortDesc = findShortDesc(text, title)
	}

	doc := &Document{
		URI:      uri,
		Version:  version,
		Text:     text,
		Lines:    buildLineMap(text),
		Elements: elements,
		Index:    idx,
		Meta:     meta,
		Kind:     kind,
	}
	doc.Symbols = extractSymbols(doc)
	return doc
}

func (d *Document) ApplyChange(version int, newText string) *Document {
	return New(d.URI, version, newText)
}

func (d *Document) DocID(rootURI string) paths.DocID {
	return paths.DocIDFromURI(d.URI, rootURI)
}

func (d *Document) Defs() []Symbol {
	var defs []Symbol
	for _, s := range d.Symbols {
		if s.Kind == DefKind {
			defs = append(defs, s)
		}
	}
	return defs
}

func (d *Document) Refs() []Symbol {
	var refs []Symbol
	for _, s := range d.Symbols {
		if s.Kind == RefKind {
			refs = append(refs, s)
		}
	}
	return refs
}

func (d *Document) ElementAt(pos Position) Element {
	for _, e := range d.Elements {
		r := e.Rng()
		if posInRange(pos, r) {
			return e
		}
	}
	return nil
}

func posInRange(pos Position, r Range) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}
	return true
}

func buildLineMap(text string) []int {
	lines := []int{0}
	for i, ch := range text {
		if ch == '\n' {
			lines = append(lines, i+1)
		}
	}
	return lines
}

func findShortDesc(text string, title *Heading) string {
	lines := strings.Split(text, "\n")
	titleLine := title.Range.Start.Line
	for i := titleLine + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			return ""
		}
		return line
	}
	return ""
}

func extractSymbols(doc *Document) []Symbol {
	var syms []Symbol

	syms = append(syms, Symbol{
		Kind:    DefKind,
		DefType: DefDoc,
		Name:    doc.URI,
		DocURI:  doc.URI,
	})

	for _, e := range doc.Elements {
		switch el := e.(type) {
		case *Heading:
			dt := DefHeading
			if el.IsTitle() {
				dt = DefTitle
			}
			syms = append(syms, Symbol{
				Kind:    DefKind,
				DefType: dt,
				Name:    el.Text,
				Slug:    el.Slug,
				DocURI:  doc.URI,
				Range:   el.Range,
			})

		case *WikiLink:
			syms = append(syms, Symbol{
				Kind:    RefKind,
				RefType: RefWikiLink,
				Name:    el.Doc,
				Slug:    paths.SlugOf(el.Doc),
				DocURI:  doc.URI,
				Range:   el.Range,
			})

		case *MdLink:
			syms = append(syms, Symbol{
				Kind:    RefKind,
				RefType: RefMdLink,
				Name:    el.URL,
				DocURI:  doc.URI,
				Range:   el.Range,
			})

		case *LinkDef:
			syms = append(syms, Symbol{
				Kind:    DefKind,
				DefType: DefLinkDef,
				Name:    el.Label,
				DocURI:  doc.URI,
				Range:   el.Range,
			})
		}
	}
	return syms
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/document/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/document/document.go internal/document/document_test.go
git commit -m "feat: Document type with parsing, indexing, symbol extraction"
```

---

### Task 8: Ditamap Package

**Files:**
- Create: `internal/ditamap/ditamap.go`
- Create: `internal/ditamap/ditamap_test.go`

- [ ] **Step 1: Write ditamap tests**

Create `internal/ditamap/ditamap_test.go`:

```go
package ditamap

import "testing"

func TestParseMap(t *testing.T) {
	input := `# Product Documentation

- [Getting Started](getting-started.md)
  - [Installation](install.md)
  - [Configuration](config.md)
- [User Guide](user-guide.md)
`
	m, err := ParseMap(input)
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if m.Title != "Product Documentation" {
		t.Errorf("Title = %q, want %q", m.Title, "Product Documentation")
	}
	if len(m.TopicRefs) != 2 {
		t.Fatalf("TopicRefs = %d, want 2", len(m.TopicRefs))
	}
	if m.TopicRefs[0].Title != "Getting Started" {
		t.Errorf("TopicRefs[0].Title = %q", m.TopicRefs[0].Title)
	}
	if m.TopicRefs[0].Href != "getting-started.md" {
		t.Errorf("TopicRefs[0].Href = %q", m.TopicRefs[0].Href)
	}
	if len(m.TopicRefs[0].Children) != 2 {
		t.Fatalf("TopicRefs[0].Children = %d, want 2", len(m.TopicRefs[0].Children))
	}
	if m.TopicRefs[0].Children[0].Title != "Installation" {
		t.Errorf("child[0].Title = %q", m.TopicRefs[0].Children[0].Title)
	}
}

func TestParseMapNoTitle(t *testing.T) {
	input := "- [Topic](topic.md)\n"
	m, err := ParseMap(input)
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if m.Title != "" {
		t.Errorf("Title = %q, want empty", m.Title)
	}
	if len(m.TopicRefs) != 1 {
		t.Errorf("TopicRefs = %d, want 1", len(m.TopicRefs))
	}
}

func TestParseMapEmpty(t *testing.T) {
	m, err := ParseMap("")
	if err != nil {
		t.Fatalf("ParseMap error: %v", err)
	}
	if len(m.TopicRefs) != 0 {
		t.Errorf("TopicRefs = %d, want 0", len(m.TopicRefs))
	}
}

func TestParseMapDeeplyNested(t *testing.T) {
	input := `# Map
- [L1](l1.md)
  - [L2](l2.md)
    - [L3](l3.md)
`
	m, _ := ParseMap(input)
	if len(m.TopicRefs) != 1 {
		t.Fatalf("TopicRefs = %d", len(m.TopicRefs))
	}
	l2 := m.TopicRefs[0].Children
	if len(l2) != 1 {
		t.Fatalf("L2 children = %d", len(l2))
	}
	l3 := l2[0].Children
	if len(l3) != 1 || l3[0].Href != "l3.md" {
		t.Errorf("L3 = %v", l3)
	}
}

func TestAllHrefs(t *testing.T) {
	input := `# Map
- [A](a.md)
  - [B](b.md)
- [C](c.md)
`
	m, _ := ParseMap(input)
	hrefs := m.AllHrefs()
	if len(hrefs) != 3 {
		t.Errorf("AllHrefs = %d, want 3", len(hrefs))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/ditamap/...
```

Expected: FAIL

- [ ] **Step 3: Implement ditamap parser**

Create `internal/ditamap/ditamap.go`:

```go
package ditamap

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type TopicRef struct {
	Href     string
	Title    string
	Children []TopicRef
}

type MapStructure struct {
	Title     string
	TopicRefs []TopicRef
}

func ParseMap(source string) (*MapStructure, error) {
	src := []byte(source)
	md := goldmark.New()
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	m := &MapStructure{}

	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Heading:
			if n.Level == 1 && m.Title == "" {
				m.Title = extractText(n, src)
			}
		case *ast.List:
			m.TopicRefs = append(m.TopicRefs, parseListItems(n, src)...)
		}
	}

	return m, nil
}

func parseListItems(list *ast.List, src []byte) []TopicRef {
	var refs []TopicRef
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		ref := TopicRef{}
		for ic := item.FirstChild(); ic != nil; ic = ic.NextSibling() {
			switch n := ic.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				link := findLink(n, src)
				if link != nil {
					ref.Href = string(link.Destination)
					ref.Title = extractText(link, src)
				}
			case *ast.List:
				ref.Children = append(ref.Children, parseListItems(n, src)...)
			}
		}
		if ref.Href != "" || ref.Title != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func findLink(n ast.Node, src []byte) *ast.Link {
	var link *ast.Link
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if l, ok := child.(*ast.Link); ok {
			link = l
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return link
}

func extractText(n ast.Node, src []byte) string {
	var result []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			result = append(result, t.Segment.Value(src)...)
		} else if c.HasChildren() {
			result = append(result, []byte(extractText(c, src))...)
		}
	}
	return string(result)
}

func (m *MapStructure) AllHrefs() []string {
	var hrefs []string
	collectHrefs(m.TopicRefs, &hrefs)
	return hrefs
}

func collectHrefs(refs []TopicRef, out *[]string) {
	for _, ref := range refs {
		if ref.Href != "" {
			*out = append(*out, ref.Href)
		}
		collectHrefs(ref.Children, out)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/ditamap/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/ditamap/
git commit -m "feat: ditamap parser for .mditamap files"
```

---

### Task 9: Workspace Package

**Files:**
- Create: `internal/workspace/folder.go`
- Create: `internal/workspace/workspace.go`
- Create: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write workspace tests**

Create `internal/workspace/workspace_test.go`:

```go
package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestFolderAddRemoveDoc(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	doc := document.New("file:///project/doc.md", 1, "# Test\n")
	f.AddDoc(doc)

	if f.DocCount() != 1 {
		t.Errorf("DocCount = %d, want 1", f.DocCount())
	}

	got := f.DocByURI("file:///project/doc.md")
	if got == nil {
		t.Fatal("DocByURI returned nil")
	}

	f.RemoveDoc("file:///project/doc.md")
	if f.DocCount() != 0 {
		t.Errorf("DocCount after remove = %d, want 0", f.DocCount())
	}
}

func TestFolderDocBySlug(t *testing.T) {
	f := NewFolder("file:///project", config.Default())
	doc := document.New("file:///project/my-topic.md", 1, "# My Topic\n")
	f.AddDoc(doc)

	got := f.DocBySlug(paths.SlugOf("my-topic"))
	if got == nil {
		t.Fatal("DocBySlug returned nil")
	}
}

func TestWorkspaceAddRemoveFolder(t *testing.T) {
	ws := New()
	f := NewFolder("file:///project", config.Default())
	ws.AddFolder(f)

	if len(ws.Folders()) != 1 {
		t.Errorf("Folders = %d, want 1", len(ws.Folders()))
	}

	ws.RemoveFolder("file:///project")
	if len(ws.Folders()) != 0 {
		t.Errorf("Folders after remove = %d", len(ws.Folders()))
	}
}

func TestWorkspaceFindDoc(t *testing.T) {
	ws := New()
	f := NewFolder("file:///project", config.Default())
	doc := document.New("file:///project/intro.md", 1, "# Intro\n")
	f.AddDoc(doc)
	ws.AddFolder(f)

	got, folder := ws.FindDoc("file:///project/intro.md")
	if got == nil || folder == nil {
		t.Fatal("FindDoc returned nil")
	}
}

func TestFolderScanFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Doc\n"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.markdown"), []byte("# Notes\n"), 0644)
	os.WriteFile(filepath.Join(dir, "map.mditamap"), []byte("# Map\n- [D](doc.md)\n"), 0644)
	os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("not md"), 0644)

	cfg := config.Default()
	f := NewFolder(paths.PathToURI(dir), cfg)
	err := f.ScanFiles()
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}
	if f.DocCount() != 3 {
		t.Errorf("DocCount = %d, want 3", f.DocCount())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/workspace/...
```

Expected: FAIL

- [ ] **Step 3: Implement Folder**

Create `internal/workspace/folder.go`:

```go
package workspace

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

type Folder struct {
	RootURI string
	Config  *config.Config

	mu      sync.RWMutex
	docs    map[string]*document.Document
	slugMap map[paths.Slug]*document.Document
}

func NewFolder(rootURI string, cfg *config.Config) *Folder {
	return &Folder{
		RootURI: rootURI,
		Config:  cfg,
		docs:    make(map[string]*document.Document),
		slugMap: make(map[paths.Slug]*document.Document),
	}
}

func (f *Folder) AddDoc(doc *document.Document) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[doc.URI] = doc
	id := doc.DocID(f.RootURI)
	f.slugMap[id.Slug] = doc
}

func (f *Folder) RemoveDoc(uri string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	doc, ok := f.docs[uri]
	if !ok {
		return
	}
	id := doc.DocID(f.RootURI)
	delete(f.slugMap, id.Slug)
	delete(f.docs, uri)
}

func (f *Folder) DocByURI(uri string) *document.Document {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.docs[uri]
}

func (f *Folder) DocBySlug(slug paths.Slug) *document.Document {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.slugMap[slug]
}

func (f *Folder) DocCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.docs)
}

func (f *Folder) AllDocs() []*document.Document {
	f.mu.RLock()
	defer f.mu.RUnlock()
	docs := make([]*document.Document, 0, len(f.docs))
	for _, d := range f.docs {
		docs = append(docs, d)
	}
	return docs
}

func (f *Folder) ScanFiles() error {
	rootPath, err := paths.URIToPath(f.RootURI)
	if err != nil {
		return err
	}
	exts := f.Config.Core.Markdown.FileExtensions
	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == ".hg" {
				return filepath.SkipDir
			}
			return nil
		}
		if !paths.IsMarkdownFile(path, exts) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		uri := paths.PathToURI(path)
		doc := document.New(uri, 0, string(data))
		f.AddDoc(doc)
		return nil
	})
}

func (f *Folder) RootPath() string {
	p, _ := paths.URIToPath(f.RootURI)
	return p
}
```

- [ ] **Step 4: Implement Workspace**

Create `internal/workspace/workspace.go`:

```go
package workspace

import (
	"strings"
	"sync"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type Workspace struct {
	mu      sync.RWMutex
	folders map[string]*Folder
}

func New() *Workspace {
	return &Workspace{
		folders: make(map[string]*Folder),
	}
}

func (ws *Workspace) AddFolder(f *Folder) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.folders[f.RootURI] = f
}

func (ws *Workspace) RemoveFolder(rootURI string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.folders, rootURI)
}

func (ws *Workspace) Folders() []*Folder {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	folders := make([]*Folder, 0, len(ws.folders))
	for _, f := range ws.folders {
		folders = append(folders, f)
	}
	return folders
}

func (ws *Workspace) FolderByURI(rootURI string) *Folder {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.folders[rootURI]
}

func (ws *Workspace) FindDoc(uri string) (*document.Document, *Folder) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	for _, f := range ws.folders {
		if doc := f.DocByURI(uri); doc != nil {
			return doc, f
		}
	}
	return nil, nil
}

func (ws *Workspace) FolderForURI(uri string) *Folder {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	for _, f := range ws.folders {
		if strings.HasPrefix(uri, f.RootURI) {
			return f
		}
	}
	return nil
}
```

- [ ] **Step 5: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/workspace/...
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/mdita-lsp
git add internal/workspace/
git commit -m "feat: workspace and folder with document management and file scanning"
```

---

### Task 10: Symbols Package (SymbolGraph)

**Files:**
- Create: `internal/symbols/graph.go`
- Create: `internal/symbols/graph_test.go`

- [ ] **Step 1: Write symbol graph tests**

Create `internal/symbols/graph_test.go`:

```go
package symbols

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

func TestGraphResolveRef(t *testing.T) {
	g := NewGraph()

	def := document.Symbol{
		Kind:    document.DefKind,
		DefType: document.DefHeading,
		Name:    "Introduction",
		Slug:    paths.SlugOf("Introduction"),
		DocURI:  "file:///project/intro.md",
		Range:   document.Rng(0, 0, 0, 14),
	}
	ref := document.Symbol{
		Kind:    document.RefKind,
		RefType: document.RefWikiLink,
		Name:    "intro",
		Slug:    paths.SlugOf("intro"),
		DocURI:  "file:///project/other.md",
		Range:   document.Rng(2, 0, 2, 12),
	}

	g.AddDefs("file:///project/intro.md", []document.Symbol{def})
	g.AddRefs("file:///project/other.md", []document.Symbol{ref})

	defs := g.ResolveRef(ref)
	if len(defs) != 1 {
		t.Fatalf("ResolveRef returned %d, want 1", len(defs))
	}
	if defs[0].Name != "Introduction" {
		t.Errorf("resolved to %q, want %q", defs[0].Name, "Introduction")
	}
}

func TestGraphFindRefs(t *testing.T) {
	g := NewGraph()

	def := document.Symbol{
		Kind:    document.DefKind,
		DefType: document.DefHeading,
		Name:    "Setup",
		Slug:    paths.SlugOf("Setup"),
		DocURI:  "file:///project/setup.md",
	}
	ref1 := document.Symbol{
		Kind:   document.RefKind,
		Name:   "setup",
		Slug:   paths.SlugOf("setup"),
		DocURI: "file:///project/a.md",
	}
	ref2 := document.Symbol{
		Kind:   document.RefKind,
		Name:   "setup",
		Slug:   paths.SlugOf("setup"),
		DocURI: "file:///project/b.md",
	}

	g.AddDefs("file:///project/setup.md", []document.Symbol{def})
	g.AddRefs("file:///project/a.md", []document.Symbol{ref1})
	g.AddRefs("file:///project/b.md", []document.Symbol{ref2})

	refs := g.FindRefs(def)
	if len(refs) != 2 {
		t.Errorf("FindRefs returned %d, want 2", len(refs))
	}
}

func TestGraphUpdateDoc(t *testing.T) {
	g := NewGraph()

	g.AddDefs("file:///project/a.md", []document.Symbol{
		{Kind: document.DefKind, Name: "Old", Slug: paths.SlugOf("Old"), DocURI: "file:///project/a.md"},
	})

	g.AddDefs("file:///project/a.md", []document.Symbol{
		{Kind: document.DefKind, Name: "New", Slug: paths.SlugOf("New"), DocURI: "file:///project/a.md"},
	})

	allDefs := g.DefsByDoc("file:///project/a.md")
	if len(allDefs) != 1 || allDefs[0].Name != "New" {
		t.Errorf("after update, defs = %v", allDefs)
	}
}

func TestGraphRemoveDoc(t *testing.T) {
	g := NewGraph()
	g.AddDefs("file:///project/a.md", []document.Symbol{
		{Kind: document.DefKind, Name: "X", Slug: paths.SlugOf("X"), DocURI: "file:///project/a.md"},
	})
	g.RemoveDoc("file:///project/a.md")
	if len(g.DefsByDoc("file:///project/a.md")) != 0 {
		t.Error("expected 0 defs after remove")
	}
}

func TestGraphDocDef(t *testing.T) {
	g := NewGraph()
	g.AddDefs("file:///project/intro.md", []document.Symbol{
		{Kind: document.DefKind, DefType: document.DefDoc, Name: "file:///project/intro.md", DocURI: "file:///project/intro.md"},
		{Kind: document.DefKind, DefType: document.DefTitle, Name: "Introduction", Slug: paths.SlugOf("Introduction"), DocURI: "file:///project/intro.md"},
	})

	defs := g.ResolveDocRef(paths.SlugOf("intro"))
	if len(defs) != 1 {
		t.Fatalf("ResolveDocRef = %d, want 1", len(defs))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/symbols/...
```

Expected: FAIL

- [ ] **Step 3: Implement SymbolGraph**

Create `internal/symbols/graph.go`:

```go
package symbols

import (
	"strings"
	"sync"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

type Graph struct {
	mu   sync.RWMutex
	defs map[string][]document.Symbol // docURI -> defs
	refs map[string][]document.Symbol // docURI -> refs

	defsBySlug map[paths.Slug][]document.Symbol
	docSlugs   map[paths.Slug]string // stem slug -> docURI
}

func NewGraph() *Graph {
	return &Graph{
		defs:       make(map[string][]document.Symbol),
		refs:       make(map[string][]document.Symbol),
		defsBySlug: make(map[paths.Slug][]document.Symbol),
		docSlugs:   make(map[paths.Slug]string),
	}
}

func (g *Graph) AddDefs(docURI string, defs []document.Symbol) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.removeDefs(docURI)
	g.defs[docURI] = defs

	for _, d := range defs {
		if d.Slug != "" {
			g.defsBySlug[d.Slug] = append(g.defsBySlug[d.Slug], d)
		}
		if d.DefType == document.DefDoc {
			stem := docStemSlug(docURI)
			g.docSlugs[stem] = docURI
		}
	}
}

func (g *Graph) AddRefs(docURI string, refs []document.Symbol) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.refs[docURI] = refs
}

func (g *Graph) RemoveDoc(docURI string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.removeDefs(docURI)
	delete(g.refs, docURI)
}

func (g *Graph) removeDefs(docURI string) {
	old := g.defs[docURI]
	for _, d := range old {
		if d.Slug != "" {
			filtered := filterByDoc(g.defsBySlug[d.Slug], docURI)
			if len(filtered) == 0 {
				delete(g.defsBySlug, d.Slug)
			} else {
				g.defsBySlug[d.Slug] = filtered
			}
		}
		if d.DefType == document.DefDoc {
			stem := docStemSlug(docURI)
			delete(g.docSlugs, stem)
		}
	}
	delete(g.defs, docURI)
}

func (g *Graph) ResolveRef(ref document.Symbol) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.defsBySlug[ref.Slug]
}

func (g *Graph) ResolveDocRef(slug paths.Slug) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()

	docURI, ok := g.docSlugs[slug]
	if !ok {
		return nil
	}
	for _, d := range g.defs[docURI] {
		if d.DefType == document.DefTitle || d.DefType == document.DefDoc {
			return []document.Symbol{d}
		}
	}
	return nil
}

func (g *Graph) FindRefs(def document.Symbol) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []document.Symbol
	for _, refs := range g.refs {
		for _, r := range refs {
			if r.Slug == def.Slug {
				result = append(result, r)
			}
		}
	}
	return result
}

func (g *Graph) DefsByDoc(docURI string) []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.defs[docURI]
}

func (g *Graph) AllDefs() []document.Symbol {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var all []document.Symbol
	for _, defs := range g.defs {
		all = append(all, defs...)
	}
	return all
}

func filterByDoc(syms []document.Symbol, docURI string) []document.Symbol {
	var result []document.Symbol
	for _, s := range syms {
		if s.DocURI != docURI {
			result = append(result, s)
		}
	}
	return result
}

func docStemSlug(uri string) paths.Slug {
	lastSlash := strings.LastIndex(uri, "/")
	name := uri[lastSlash+1:]
	dotIdx := strings.LastIndex(name, ".")
	if dotIdx > 0 {
		name = name[:dotIdx]
	}
	return paths.SlugOf(name)
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/symbols/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/symbols/
git commit -m "feat: symbol graph with bidirectional ref/def resolution"
```

---

### Task 11: Diagnostic Package

**Files:**
- Create: `internal/diagnostic/diagnostic.go`
- Create: `internal/diagnostic/mdita.go`
- Create: `internal/diagnostic/links.go`
- Create: `internal/diagnostic/diagnostic_test.go`

- [ ] **Step 1: Write diagnostic tests**

Create `internal/diagnostic/diagnostic_test.go`:

```go
package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func makeDoc(uri string, lines ...string) *document.Document {
	text := ""
	for i, l := range lines {
		text += l
		if i < len(lines)-1 {
			text += "\n"
		}
	}
	return document.New(uri, 1, text)
}

func makeFolder(docs ...*document.Document) *workspace.Folder {
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	for _, d := range docs {
		f.AddDoc(d)
	}
	return f
}

func TestMissingYamlFrontMatter(t *testing.T) {
	doc := makeDoc("file:///project/doc.md", "# Title", "", "Some text.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeMissingYamlFrontMatter {
			found = true
		}
	}
	if !found {
		t.Error("expected MissingYamlFrontMatter diagnostic")
	}
}

func TestNoMissingYamlWhenPresent(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "Short desc.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeMissingYamlFrontMatter {
			t.Error("should not report MissingYamlFrontMatter when YAML is present")
		}
	}
}

func TestMissingShortDescription(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "## Next Section")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeMissingShortDescription {
			found = true
		}
	}
	if !found {
		t.Error("expected MissingShortDescription diagnostic")
	}
}

func TestInvalidHeadingHierarchy(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "Short desc.", "", "### Skipped H2")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeInvalidHeadingHierarchy {
			found = true
		}
	}
	if !found {
		t.Error("expected InvalidHeadingHierarchy diagnostic")
	}
}

func TestTaskMissingProcedure(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:task.xsd", "---",
		"# Task Title", "", "Some text.")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeTaskMissingProcedure {
			found = true
		}
	}
	if !found {
		t.Error("expected TaskMissingProcedure diagnostic")
	}
}

func TestConceptHasProcedure(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "$schema: urn:oasis:names:tc:dita:xsd:concept.xsd", "---",
		"# Concept Title", "", "Some text.", "", "1. step one", "2. step two")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeConceptHasProcedure {
			found = true
		}
	}
	if !found {
		t.Error("expected ConceptHasProcedure diagnostic")
	}
}

func TestBrokenLink(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"# Title", "", "[[nonexistent]]")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeBrokenLink {
			found = true
		}
	}
	if !found {
		t.Error("expected BrokenLink diagnostic")
	}
}

func TestValidLink(t *testing.T) {
	doc1 := makeDoc("file:///project/intro.md", "# Intro", "", "Some text.")
	doc2 := makeDoc("file:///project/doc.md", "# Doc", "", "[[intro]]")
	f := makeFolder(doc1, doc2)
	diags := Check(doc2, f)

	for _, d := range diags {
		if d.Code == CodeBrokenLink {
			t.Errorf("should not report BrokenLink for valid wiki link, got: %s", d.Message)
		}
	}
}

func TestUnknownAdmonitionType(t *testing.T) {
	doc := makeDoc("file:///project/doc.md",
		"---", "author: Test", "---", "# Title", "", "Short desc.", "", "!!! invalid", "    content")
	f := makeFolder(doc)
	diags := Check(doc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeUnknownAdmonitionType {
			found = true
		}
	}
	if !found {
		t.Error("expected UnknownAdmonitionType diagnostic")
	}
}

func TestMditaDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Core.Mdita.Enable = false
	cfg.Diagnostics.MditaCompliance = false

	doc := makeDoc("file:///project/doc.md", "# Title")
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	diags := Check(doc, f)

	for _, d := range diags {
		if d.Code == CodeMissingYamlFrontMatter {
			t.Error("should not report MDITA diagnostics when disabled")
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/diagnostic/...
```

Expected: FAIL

- [ ] **Step 3: Implement diagnostic types**

Create `internal/diagnostic/diagnostic.go`:

```go
package diagnostic

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Severity int

const (
	SeverityError   Severity = 1
	SeverityWarning Severity = 2
	SeverityInfo    Severity = 3
)

const (
	CodeAmbiguousLink                  = "1"
	CodeBrokenLink                     = "2"
	CodeNonBreakingWhitespace          = "3"
	CodeMissingYamlFrontMatter         = "4"
	CodeMissingShortDescription        = "5"
	CodeInvalidHeadingHierarchy        = "6"
	CodeUnrecognizedSchema             = "7"
	CodeTaskMissingProcedure           = "8"
	CodeConceptHasProcedure            = "9"
	CodeReferenceMissingTable          = "10"
	CodeMapHasBodyContent              = "11"
	CodeExtendedFeatureInCoreProfile   = "12"
	CodeFootnoteRefWithoutDef          = "13"
	CodeFootnoteDefWithoutRef          = "14"
	CodeUnknownAdmonitionType          = "15"
	CodeUnresolvedKeyref               = "16"
	CodeBrokenMapReference             = "17"
	CodeCircularMapReference           = "18"
	CodeInconsistentMapHeadingHierarchy = "19"
)

type Diagnostic struct {
	Range    document.Range
	Severity Severity
	Code     string
	Source   string
	Message  string
}

const source = "mdita-lsp"

func Check(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	cfg := folder.Config
	if cfg.Diagnostics.MditaCompliance && cfg.Core.Mdita.Enable {
		diags = append(diags, checkMditaCompliance(doc)...)
	}

	diags = append(diags, checkLinks(doc, folder)...)
	diags = append(diags, checkNonBreakingWhitespace(doc)...)

	return diags
}
```

- [ ] **Step 4: Implement MDITA compliance checks**

Create `internal/diagnostic/mdita.go`:

```go
package diagnostic

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

var validAdmonitionTypes = map[string]bool{
	"note": true, "tip": true, "warning": true, "caution": true,
	"danger": true, "attention": true, "important": true, "notice": true,
	"fastpath": true, "remember": true, "restriction": true, "trouble": true,
}

func checkMditaCompliance(doc *document.Document) []Diagnostic {
	var diags []Diagnostic

	if doc.Meta == nil {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeMissingYamlFrontMatter,
			Source:   source,
			Message:  "Missing YAML front matter",
		})
		return diags
	}

	if doc.Meta.SchemaRaw != "" && doc.Meta.Schema == document.SchemaUnknown {
		diags = append(diags, Diagnostic{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityWarning,
			Code:     CodeUnrecognizedSchema,
			Source:   source,
			Message:  "Unrecognized $schema value: " + doc.Meta.SchemaRaw,
		})
	}

	title := doc.Index.Title()
	if title != nil && doc.Index.ShortDesc == "" {
		diags = append(diags, Diagnostic{
			Range:    title.Range,
			Severity: SeverityWarning,
			Code:     CodeMissingShortDescription,
			Source:   source,
			Message:  "Missing short description (paragraph after title)",
		})
	}

	diags = append(diags, checkHeadingHierarchy(doc)...)
	diags = append(diags, checkSchemaSpecific(doc)...)
	diags = append(diags, checkExtendedFeatures(doc)...)
	diags = append(diags, checkAdmonitions(doc)...)

	return diags
}

func checkHeadingHierarchy(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	headings := doc.Index.Headings()
	for i := 1; i < len(headings); i++ {
		prev := headings[i-1].Level
		curr := headings[i].Level
		if curr > prev+1 {
			diags = append(diags, Diagnostic{
				Range:    headings[i].Range,
				Severity: SeverityWarning,
				Code:     CodeInvalidHeadingHierarchy,
				Source:   source,
				Message:  "Invalid heading hierarchy: skipped heading level",
			})
		}
	}
	return diags
}

func checkSchemaSpecific(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	if doc.Meta == nil {
		return diags
	}
	bf := doc.Index.Features

	switch doc.Meta.Schema {
	case document.SchemaTask:
		if !bf.HasOrderedList {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityWarning,
				Code:     CodeTaskMissingProcedure,
				Source:   source,
				Message:  "Task topic is missing a procedure (ordered list)",
			})
		}

	case document.SchemaConcept:
		if bf.HasOrderedList {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityInfo,
				Code:     CodeConceptHasProcedure,
				Source:   source,
				Message:  "Concept topic contains an ordered list — consider using task schema",
			})
		}

	case document.SchemaReference:
		if !bf.HasTable && !bf.HasDefinitionList {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityInfo,
				Code:     CodeReferenceMissingTable,
				Source:   source,
				Message:  "Reference topic is missing a table or definition list",
			})
		}

	case document.SchemaMap:
		if bf.HasOrderedList || bf.HasUnorderedList || bf.HasTable {
			// map may have body content beyond links — already checked by ditamap validator
		}
	}

	if doc.Kind == document.Map {
		hasNonLinkContent := bf.HasOrderedList || bf.HasTable || bf.HasDefinitionList
		if hasNonLinkContent {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityInfo,
				Code:     CodeMapHasBodyContent,
				Source:   source,
				Message:  "Map contains body content beyond topic references",
			})
		}
	}

	return diags
}

func checkExtendedFeatures(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	if doc.Meta == nil || doc.Meta.Schema != document.SchemaMditaCoreTopic {
		return diags
	}
	bf := doc.Index.Features

	if bf.HasDefinitionList {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Definition lists are an extended profile feature",
		})
	}
	if bf.HasFootnoteRefs || bf.HasFootnoteDefs {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Footnotes are an extended profile feature",
		})
	}
	if bf.HasStrikethrough {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Strikethrough is an extended profile feature",
		})
	}
	if bf.HasAttributes {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Generic attributes are an extended profile feature",
		})
	}
	if len(bf.Admonitions) > 0 {
		diags = append(diags, Diagnostic{
			Range: document.Rng(0, 0, 0, 0), Severity: SeverityWarning,
			Code: CodeExtendedFeatureInCoreProfile, Source: source,
			Message: "Admonitions are an extended profile feature",
		})
	}

	return diags
}

func checkAdmonitions(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	bf := doc.Index.Features
	for _, adm := range bf.Admonitions {
		if !validAdmonitionTypes[strings.ToLower(adm.Type)] {
			diags = append(diags, Diagnostic{
				Range:    adm.Range,
				Severity: SeverityWarning,
				Code:     CodeUnknownAdmonitionType,
				Source:   source,
				Message:  "Unknown admonition type: " + adm.Type,
			})
		}
	}
	return diags
}
```

- [ ] **Step 5: Implement link checks**

Create `internal/diagnostic/links.go`:

```go
package diagnostic

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func checkLinks(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	for _, wl := range doc.Index.WikiLinks() {
		if wl.Doc == "" && wl.Heading != "" {
			slug := paths.SlugOf(wl.Heading)
			if len(doc.Index.HeadingsBySlug(slug)) == 0 {
				diags = append(diags, Diagnostic{
					Range:    wl.Range,
					Severity: SeverityError,
					Code:     CodeBrokenLink,
					Source:   source,
					Message:  "Link to non-existent heading '" + wl.Heading + "'",
				})
			}
			continue
		}

		if wl.Doc != "" {
			slug := paths.SlugOf(wl.Doc)
			target := folder.DocBySlug(slug)
			if target == nil {
				diags = append(diags, Diagnostic{
					Range:    wl.Range,
					Severity: SeverityError,
					Code:     CodeBrokenLink,
					Source:   source,
					Message:  "Link to non-existent document '" + wl.Doc + "'",
				})
			} else if wl.Heading != "" {
				hslug := paths.SlugOf(wl.Heading)
				if len(target.Index.HeadingsBySlug(hslug)) == 0 {
					diags = append(diags, Diagnostic{
						Range:    wl.Range,
						Severity: SeverityError,
						Code:     CodeBrokenLink,
						Source:   source,
						Message:  "Link to non-existent heading '" + wl.Heading + "' in '" + wl.Doc + "'",
					})
				}
			}
		}
	}

	for _, ml := range doc.Index.MdLinks() {
		if ml.URL == "" && ml.Anchor != "" {
			slug := paths.SlugOf(ml.Anchor)
			if len(doc.Index.HeadingsBySlug(slug)) == 0 {
				diags = append(diags, Diagnostic{
					Range:    ml.Range,
					Severity: SeverityError,
					Code:     CodeBrokenLink,
					Source:   source,
					Message:  "Link to non-existent heading '#" + ml.Anchor + "'",
				})
			}
		}
	}

	return diags
}

func checkNonBreakingWhitespace(doc *document.Document) []Diagnostic {
	var diags []Diagnostic
	for _, h := range doc.Index.Headings() {
		if containsNBSP(h.Text) {
			diags = append(diags, Diagnostic{
				Range:    h.Range,
				Severity: SeverityWarning,
				Code:     CodeNonBreakingWhitespace,
				Source:   source,
				Message:  "Heading contains non-breaking whitespace",
			})
		}
	}
	return diags
}

func containsNBSP(s string) bool {
	return strings.ContainsRune(s, '\u00A0')
}
```

- [ ] **Step 6: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/diagnostic/...
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
cd ~/mdita-lsp
git add internal/diagnostic/
git commit -m "feat: diagnostics — MDITA compliance, link validation, 15 diagnostic codes"
```

---

### Task 12: Ditamap Validation

**Files:**
- Create: `internal/diagnostic/ditamap_validation.go`
- Create: `internal/diagnostic/ditamap_validation_test.go`

- [ ] **Step 1: Write ditamap validation tests**

Create `internal/diagnostic/ditamap_validation_test.go`:

```go
package diagnostic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestBrokenMapReference(t *testing.T) {
	mapDoc := makeDoc("file:///project/map.mditamap",
		"# My Map", "", "- [Topic](nonexistent.md)")
	f := makeFolder(mapDoc)
	diags := CheckDitamap(mapDoc, f)

	found := false
	for _, d := range diags {
		if d.Code == CodeBrokenMapReference {
			found = true
		}
	}
	if !found {
		t.Error("expected BrokenMapReference diagnostic")
	}
}

func TestValidMapReference(t *testing.T) {
	topic := makeDoc("file:///project/topic.md", "# Topic", "", "Content.")
	mapDoc := makeDoc("file:///project/map.mditamap",
		"# My Map", "", "- [Topic](topic.md)")
	f := makeFolder(topic, mapDoc)
	diags := CheckDitamap(mapDoc, f)

	for _, d := range diags {
		if d.Code == CodeBrokenMapReference {
			t.Error("should not report BrokenMapReference for valid ref")
		}
	}
}

func TestCircularMapReference(t *testing.T) {
	map1 := document.New("file:///project/a.mditamap", 1,
		"# Map A\n\n- [Map B](b.mditamap)\n")
	map2 := document.New("file:///project/b.mditamap", 1,
		"# Map B\n\n- [Map A](a.mditamap)\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(map1)
	f.AddDoc(map2)

	diags := CheckDitamap(map1, f)
	found := false
	for _, d := range diags {
		if d.Code == CodeCircularMapReference {
			found = true
		}
	}
	if !found {
		t.Error("expected CircularMapReference diagnostic")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/diagnostic/... -run TestBroken
```

Expected: FAIL

- [ ] **Step 3: Implement ditamap validation**

Create `internal/diagnostic/ditamap_validation.go`:

```go
package diagnostic

import (
	"path/filepath"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func CheckDitamap(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	if doc.Kind != document.Map {
		return nil
	}

	m, err := ditamap.ParseMap(doc.Text)
	if err != nil {
		return nil
	}

	var diags []Diagnostic
	diags = append(diags, checkMapRefs(m, doc, folder)...)
	diags = append(diags, checkCircularRefs(doc, folder)...)
	return diags
}

func checkMapRefs(m *ditamap.MapStructure, doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic
	docPath, _ := paths.URIToPath(doc.URI)
	docDir := filepath.Dir(docPath)

	for _, href := range m.AllHrefs() {
		targetPath := filepath.Join(docDir, href)
		targetURI := paths.PathToURI(targetPath)
		if folder.DocByURI(targetURI) == nil {
			diags = append(diags, Diagnostic{
				Range:    document.Rng(0, 0, 0, 0),
				Severity: SeverityError,
				Code:     CodeBrokenMapReference,
				Source:   source,
				Message:  "Map references non-existent file: " + href,
			})
		}
	}
	return diags
}

func checkCircularRefs(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	visited := make(map[string]bool)
	if hasCycle(doc.URI, folder, visited) {
		return []Diagnostic{{
			Range:    document.Rng(0, 0, 0, 0),
			Severity: SeverityError,
			Code:     CodeCircularMapReference,
			Source:   source,
			Message:  "Circular map reference detected",
		}}
	}
	return nil
}

func hasCycle(uri string, folder *workspace.Folder, visited map[string]bool) bool {
	if visited[uri] {
		return true
	}
	visited[uri] = true

	doc := folder.DocByURI(uri)
	if doc == nil || doc.Kind != document.Map {
		delete(visited, uri)
		return false
	}

	m, err := ditamap.ParseMap(doc.Text)
	if err != nil {
		delete(visited, uri)
		return false
	}

	docPath, _ := paths.URIToPath(uri)
	docDir := filepath.Dir(docPath)
	mapExts := folder.Config.Core.Mdita.MapExtensions

	for _, href := range m.AllHrefs() {
		if !paths.IsMditaMapFile(href, mapExts) {
			continue
		}
		targetPath := filepath.Join(docDir, href)
		targetURI := paths.PathToURI(targetPath)
		if hasCycle(targetURI, folder, visited) {
			return true
		}
	}

	delete(visited, uri)
	return false
}
```

- [ ] **Step 4: Wire ditamap checks into main Check function**

Edit `internal/diagnostic/diagnostic.go` — add ditamap checking:

In the `Check` function, add after the link checks:

```go
func Check(doc *document.Document, folder *workspace.Folder) []Diagnostic {
	var diags []Diagnostic

	cfg := folder.Config
	if cfg.Diagnostics.MditaCompliance && cfg.Core.Mdita.Enable {
		diags = append(diags, checkMditaCompliance(doc)...)
	}

	diags = append(diags, checkLinks(doc, folder)...)
	diags = append(diags, checkNonBreakingWhitespace(doc)...)

	if cfg.Diagnostics.DitamapValidation {
		diags = append(diags, CheckDitamap(doc, folder)...)
	}

	return diags
}
```

- [ ] **Step 5: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/diagnostic/...
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/mdita-lsp
git add internal/diagnostic/
git commit -m "feat: ditamap validation — broken refs, circular refs"
```

---

### Task 13: Keyref Package

**Files:**
- Create: `internal/keyref/keyref.go`
- Create: `internal/keyref/keyref_test.go`

- [ ] **Step 1: Write keyref tests**

Create `internal/keyref/keyref_test.go`:

```go
package keyref

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
)

func TestExtractKeys(t *testing.T) {
	m := &ditamap.MapStructure{
		Title: "Product Docs",
		TopicRefs: []ditamap.TopicRef{
			{Href: "install.md", Title: "Installation"},
			{Href: "config.md", Title: "Configuration",
				Children: []ditamap.TopicRef{
					{Href: "config-advanced.md", Title: "Advanced Config"},
				}},
		},
	}

	table := ExtractKeys(m)
	if len(table) != 3 {
		t.Fatalf("ExtractKeys = %d keys, want 3", len(table))
	}

	entry, ok := table["install"]
	if !ok {
		t.Fatal("missing key 'install'")
	}
	if entry.Href != "install.md" || entry.Title != "Installation" {
		t.Errorf("install entry = %+v", entry)
	}
}

func TestResolveKeyref(t *testing.T) {
	table := KeyTable{
		"install": {Href: "install.md", Title: "Installation"},
	}

	entry, ok := Resolve(table, "install")
	if !ok {
		t.Fatal("Resolve returned false")
	}
	if entry.Href != "install.md" {
		t.Errorf("Href = %q", entry.Href)
	}

	_, ok = Resolve(table, "nonexistent")
	if ok {
		t.Error("Resolve should return false for unknown key")
	}
}

func TestExtractKeysFromSlug(t *testing.T) {
	m := &ditamap.MapStructure{
		TopicRefs: []ditamap.TopicRef{
			{Href: "getting-started.md", Title: "Getting Started"},
		},
	}
	table := ExtractKeys(m)

	_, ok := table["getting-started"]
	if !ok {
		t.Error("expected key 'getting-started' derived from href stem")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/keyref/...
```

Expected: FAIL

- [ ] **Step 3: Implement keyref**

Create `internal/keyref/keyref.go`:

```go
package keyref

import (
	"path/filepath"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/ditamap"
)

type KeyEntry struct {
	Href  string
	Title string
}

type KeyTable map[string]KeyEntry

func ExtractKeys(m *ditamap.MapStructure) KeyTable {
	table := make(KeyTable)
	extractKeysFromRefs(m.TopicRefs, table)
	return table
}

func extractKeysFromRefs(refs []ditamap.TopicRef, table KeyTable) {
	for _, ref := range refs {
		if ref.Href != "" {
			key := stemFromHref(ref.Href)
			table[key] = KeyEntry{
				Href:  ref.Href,
				Title: ref.Title,
			}
		}
		extractKeysFromRefs(ref.Children, table)
	}
}

func stemFromHref(href string) string {
	base := filepath.Base(href)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func Resolve(table KeyTable, key string) (KeyEntry, bool) {
	entry, ok := table[key]
	return entry, ok
}

func AllKeys(table KeyTable) []string {
	keys := make([]string, 0, len(table))
	for k := range table {
		keys = append(keys, k)
	}
	return keys
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/keyref/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/keyref/
git commit -m "feat: keyref package — key extraction from ditamaps and resolution"
```

---

### Task 14: Definition Package

**Files:**
- Create: `internal/definition/definition.go`
- Create: `internal/definition/definition_test.go`

- [ ] **Step 1: Write definition tests**

Create `internal/definition/definition_test.go`:

```go
package definition

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func setup() (*workspace.Folder, *symbols.Graph) {
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	g := symbols.NewGraph()

	doc1 := document.New("file:///project/intro.md", 1,
		"# Introduction\n\n## Getting Started\n\nSome text.\n")
	doc2 := document.New("file:///project/guide.md", 1,
		"# Guide\n\n[[intro]]\n\n[[intro#getting-started]]\n\n[link](intro.md#getting-started)\n")

	f.AddDoc(doc1)
	f.AddDoc(doc2)
	g.AddDefs(doc1.URI, doc1.Defs())
	g.AddRefs(doc1.URI, doc1.Refs())
	g.AddDefs(doc2.URI, doc2.Defs())
	g.AddRefs(doc2.URI, doc2.Refs())

	return f, g
}

func TestGotoDefWikiLinkDoc(t *testing.T) {
	f, g := setup()
	doc := f.DocByURI("file:///project/guide.md")

	wl := doc.Index.WikiLinks()[0]
	locs := GotoDef(doc, wl.Rng().Start, f, g)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations")
	}
	if locs[0].URI != "file:///project/intro.md" {
		t.Errorf("URI = %q", locs[0].URI)
	}
}

func TestGotoDefWikiLinkHeading(t *testing.T) {
	f, g := setup()
	doc := f.DocByURI("file:///project/guide.md")

	wl := doc.Index.WikiLinks()[1]
	locs := GotoDef(doc, wl.Rng().Start, f, g)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for heading wiki link")
	}
}

func TestGotoDefIntraDocHeading(t *testing.T) {
	doc := document.New("file:///project/self.md", 1,
		"# Title\n\n## Section\n\n[[#section]]\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())
	g.AddRefs(doc.URI, doc.Refs())

	wl := doc.Index.WikiLinks()[0]
	locs := GotoDef(doc, wl.Rng().Start, f, g)
	if len(locs) == 0 {
		t.Fatal("GotoDef returned no locations for intra-doc heading")
	}
	if locs[0].URI != doc.URI {
		t.Errorf("expected same doc, got %q", locs[0].URI)
	}
}

func TestGotoDefNoResult(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()

	locs := GotoDef(doc, document.Position{Line: 2, Character: 3}, f, g)
	if len(locs) != 0 {
		t.Errorf("expected no locations, got %d", len(locs))
	}
}

```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/definition/...
```

Expected: FAIL

- [ ] **Step 3: Implement definition**

Create `internal/definition/definition.go`:

```go
package definition

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Location struct {
	URI   string
	Range document.Range
}

func GotoDef(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) []Location {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}

	switch el := elem.(type) {
	case *document.WikiLink:
		return resolveWikiLink(el, doc, folder, graph)
	case *document.MdLink:
		return resolveMdLink(el, doc, folder)
	}

	return nil
}

func resolveWikiLink(wl *document.WikiLink, doc *document.Document, folder *workspace.Folder, graph *symbols.Graph) []Location {
	if wl.Doc == "" && wl.Heading != "" {
		slug := paths.SlugOf(wl.Heading)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return []Location{{URI: doc.URI, Range: h.Range}}
		}
		return nil
	}

	targetSlug := paths.SlugOf(wl.Doc)
	target := folder.DocBySlug(targetSlug)
	if target == nil {
		return nil
	}

	if wl.Heading != "" {
		hslug := paths.SlugOf(wl.Heading)
		for _, h := range target.Index.HeadingsBySlug(hslug) {
			return []Location{{URI: target.URI, Range: h.Range}}
		}
	}

	title := target.Index.Title()
	if title != nil {
		return []Location{{URI: target.URI, Range: title.Range}}
	}
	return []Location{{URI: target.URI, Range: document.Rng(0, 0, 0, 0)}}
}

func resolveMdLink(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) []Location {
	if ml.URL == "" && ml.Anchor != "" {
		slug := paths.SlugOf(ml.Anchor)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return []Location{{URI: doc.URI, Range: h.Range}}
		}
		return nil
	}

	if ml.URL != "" {
		for _, d := range folder.AllDocs() {
			id := d.DocID(folder.RootURI)
			if matchesURL(id, ml.URL) {
				if ml.Anchor != "" {
					hslug := paths.SlugOf(ml.Anchor)
					for _, h := range d.Index.HeadingsBySlug(hslug) {
						return []Location{{URI: d.URI, Range: h.Range}}
					}
				}
				title := d.Index.Title()
				if title != nil {
					return []Location{{URI: d.URI, Range: title.Range}}
				}
				return []Location{{URI: d.URI, Range: document.Rng(0, 0, 0, 0)}}
			}
		}
	}

	return nil
}

func matchesURL(id paths.DocID, url string) bool {
	return id.RelPath == url || id.Stem+".md" == url || id.Stem+".markdown" == url
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/definition/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/definition/
git commit -m "feat: goto definition for wiki links and markdown links"
```

---

### Task 15: Hover Package

**Files:**
- Create: `internal/hover/hover.go`
- Create: `internal/hover/hover_test.go`

- [ ] **Step 1: Write hover tests**

Create `internal/hover/hover_test.go`:

```go
package hover

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestHoverWikiLink(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\nThis is the intro.\n")
	doc2 := document.New("file:///project/guide.md", 1, "# Guide\n\n[[intro]]\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	g := symbols.NewGraph()
	g.AddDefs(doc1.URI, doc1.Defs())
	g.AddDefs(doc2.URI, doc2.Defs())

	wl := doc2.Index.WikiLinks()[0]
	result := GetHover(doc2, wl.Rng().Start, f, g)
	if result == "" {
		t.Fatal("expected hover content")
	}
	if result != "**Introduction**" {
		t.Errorf("hover = %q, want %q", result, "**Introduction**")
	}
}

func TestHoverNoElement(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nPlain text.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()

	result := GetHover(doc, document.Position{Line: 2, Character: 3}, f, g)
	if result != "" {
		t.Errorf("expected empty hover, got %q", result)
	}
}
```

- [ ] **Step 2: Implement hover**

Create `internal/hover/hover.go`:

```go
package hover

import (
	"fmt"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func GetHover(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) string {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return ""
	}

	switch el := elem.(type) {
	case *document.WikiLink:
		return hoverWikiLink(el, doc, folder)
	case *document.MdLink:
		return hoverMdLink(el, doc, folder)
	case *document.Heading:
		return "**" + el.Text + "** (level " + itoa(el.Level) + ")"
	}
	return ""
}

func hoverWikiLink(wl *document.WikiLink, doc *document.Document, folder *workspace.Folder) string {
	if wl.Doc == "" && wl.Heading != "" {
		slug := paths.SlugOf(wl.Heading)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return "**" + h.Text + "**"
		}
		return ""
	}

	targetSlug := paths.SlugOf(wl.Doc)
	target := folder.DocBySlug(targetSlug)
	if target == nil {
		return ""
	}

	title := target.Index.Title()
	if title != nil {
		result := "**" + title.Text + "**"
		if wl.Heading != "" {
			hslug := paths.SlugOf(wl.Heading)
			for _, h := range target.Index.HeadingsBySlug(hslug) {
				result += " > " + h.Text
			}
		}
		return result
	}
	return ""
}

func hoverMdLink(ml *document.MdLink, doc *document.Document, folder *workspace.Folder) string {
	if ml.URL == "" && ml.Anchor != "" {
		slug := paths.SlugOf(ml.Anchor)
		for _, h := range doc.Index.HeadingsBySlug(slug) {
			return "**" + h.Text + "**"
		}
	}
	return ml.URL
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
```

- [ ] **Step 3: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/hover/...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
cd ~/mdita-lsp
git add internal/hover/
git commit -m "feat: hover content for wiki links and markdown links"
```

---

### Task 16: References Package

**Files:**
- Create: `internal/references/references.go`
- Create: `internal/references/references_test.go`

- [ ] **Step 1: Write references tests**

Create `internal/references/references_test.go`:

```go
package references

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestFindRefsToHeading(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\nContent.\n")
	doc2 := document.New("file:///project/a.md", 1, "# A\n\n[[intro]]\n")
	doc3 := document.New("file:///project/b.md", 1, "# B\n\n[[intro]]\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	f.AddDoc(doc3)

	g := symbols.NewGraph()
	for _, d := range f.AllDocs() {
		g.AddDefs(d.URI, d.Defs())
		g.AddRefs(d.URI, d.Refs())
	}

	heading := doc1.Index.Title()
	locs := FindRefs(doc1, heading.Range.Start, f, g)
	if len(locs) < 2 {
		t.Errorf("FindRefs returned %d, want >= 2", len(locs))
	}
}

func TestFindRefsNoRefs(t *testing.T) {
	doc := document.New("file:///project/lonely.md", 1, "# Lonely\n\nNo refs.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	heading := doc.Index.Title()
	locs := FindRefs(doc, heading.Range.Start, f, g)
	if len(locs) != 0 {
		t.Errorf("FindRefs returned %d, want 0", len(locs))
	}
}
```

- [ ] **Step 2: Implement references**

Create `internal/references/references.go`:

```go
package references

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Location struct {
	URI   string
	Range document.Range
}

func FindRefs(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) []Location {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}

	heading, ok := elem.(*document.Heading)
	if !ok {
		return nil
	}

	target := document.Symbol{
		Kind: document.DefKind,
		Slug: heading.Slug,
	}

	refs := graph.FindRefs(target)
	var locs []Location
	for _, r := range refs {
		locs = append(locs, Location{URI: r.DocURI, Range: r.Range})
	}
	return locs
}

func CountRefs(heading *document.Heading, graph *symbols.Graph) int {
	target := document.Symbol{Slug: heading.Slug}
	return len(graph.FindRefs(target))
}
```

- [ ] **Step 3: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/references/...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
cd ~/mdita-lsp
git add internal/references/
git commit -m "feat: find references for headings"
```

---

### Task 17: Completion Package

**Files:**
- Create: `internal/completion/completion.go`
- Create: `internal/completion/partial.go`
- Create: `internal/completion/completion_test.go`

- [ ] **Step 1: Write completion tests**

Create `internal/completion/completion_test.go`:

```go
package completion

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestDetectPartialWikiLink(t *testing.T) {
	text := "# Title\n\n[[int"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 5})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialWikiLink {
		t.Errorf("Kind = %v, want PartialWikiLink", pe.Kind)
	}
	if pe.Input != "int" {
		t.Errorf("Input = %q, want %q", pe.Input, "int")
	}
}

func TestDetectPartialWikiHeading(t *testing.T) {
	text := "# Title\n\n[[doc#sec"
	pe := DetectPartial(text, document.Position{Line: 2, Character: 9})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialWikiHeading {
		t.Errorf("Kind = %v, want PartialWikiHeading", pe.Kind)
	}
	if pe.DocPart != "doc" {
		t.Errorf("DocPart = %q", pe.DocPart)
	}
	if pe.Input != "sec" {
		t.Errorf("Input = %q", pe.Input)
	}
}

func TestDetectPartialYamlKey(t *testing.T) {
	text := "---\naut"
	pe := DetectPartial(text, document.Position{Line: 1, Character: 3})
	if pe == nil {
		t.Fatal("expected partial element")
	}
	if pe.Kind != PartialYamlKey {
		t.Errorf("Kind = %v, want PartialYamlKey", pe.Kind)
	}
}

func TestDetectNoPartial(t *testing.T) {
	text := "# Title\n\nPlain text."
	pe := DetectPartial(text, document.Position{Line: 2, Character: 5})
	if pe != nil {
		t.Errorf("expected nil partial, got %v", pe.Kind)
	}
}

func TestCompleteWikiDoc(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n")
	doc2 := document.New("file:///project/guide.md", 1, "# Guide\n\n[[int")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	g := symbols.NewGraph()
	g.AddDefs(doc1.URI, doc1.Defs())

	items := Complete(doc2, document.Position{Line: 2, Character: 5}, f, g)
	if len(items) == 0 {
		t.Fatal("expected completion items")
	}
	found := false
	for _, item := range items {
		if item.Label == "intro" || item.Label == "Introduction" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'intro' in completions, got %v", items)
	}
}

func TestCompleteYamlKey(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "---\naut")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)
	g := symbols.NewGraph()

	items := Complete(doc, document.Position{Line: 1, Character: 3}, f, g)
	found := false
	for _, item := range items {
		if item.Label == "author" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'author' in YAML key completions")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/mdita-lsp && go test ./internal/completion/...
```

Expected: FAIL

- [ ] **Step 3: Implement partial element detection**

Create `internal/completion/partial.go`:

```go
package completion

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type PartialKind int

const (
	PartialWikiLink PartialKind = iota
	PartialWikiHeading
	PartialInlineLink
	PartialInlineAnchor
	PartialRefLink
	PartialYamlKey
)

type PartialElement struct {
	Kind    PartialKind
	Input   string
	DocPart string
	Range   document.Range
}

func DetectPartial(text string, pos document.Position) *PartialElement {
	lines := strings.Split(text, "\n")
	if pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	col := pos.Character
	if col > len(line) {
		col = len(line)
	}
	prefix := line[:col]

	if inYamlBlock(lines, pos.Line) {
		key := strings.TrimSpace(prefix)
		if !strings.Contains(key, ":") {
			return &PartialElement{
				Kind:  PartialYamlKey,
				Input: key,
			}
		}
		return nil
	}

	if idx := strings.LastIndex(prefix, "[["); idx >= 0 {
		if !strings.Contains(prefix[idx:], "]]") {
			content := prefix[idx+2:]
			if hashIdx := strings.Index(content, "#"); hashIdx >= 0 {
				return &PartialElement{
					Kind:    PartialWikiHeading,
					DocPart: content[:hashIdx],
					Input:   content[hashIdx+1:],
				}
			}
			return &PartialElement{
				Kind:  PartialWikiLink,
				Input: content,
			}
		}
	}

	if idx := strings.LastIndex(prefix, "]("); idx >= 0 {
		if !strings.Contains(prefix[idx:], ")") {
			content := prefix[idx+2:]
			if hashIdx := strings.Index(content, "#"); hashIdx >= 0 {
				return &PartialElement{
					Kind:    PartialInlineAnchor,
					DocPart: content[:hashIdx],
					Input:   content[hashIdx+1:],
				}
			}
			return &PartialElement{
				Kind:  PartialInlineLink,
				Input: content,
			}
		}
	}

	return nil
}

func inYamlBlock(lines []string, lineNum int) bool {
	if lineNum == 0 {
		return false
	}
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return false
	}
	for i := 1; i < lineNum; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" || trimmed == "..." {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Implement completion**

Create `internal/completion/completion.go`:

```go
package completion

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type CompletionItem struct {
	Label      string
	Detail     string
	InsertText string
	Kind       int // 1=text, 6=variable, 17=keyword, 18=snippet
}

var yamlKeys = []string{
	"author", "source", "publisher", "permissions", "audience",
	"category", "keyword", "resourceid", "$schema",
}

var schemaValues = []string{
	"urn:oasis:names:tc:dita:xsd:topic.xsd",
	"urn:oasis:names:tc:dita:xsd:concept.xsd",
	"urn:oasis:names:tc:dita:xsd:task.xsd",
	"urn:oasis:names:tc:dita:xsd:reference.xsd",
	"urn:oasis:names:tc:dita:xsd:map.xsd",
	"urn:oasis:names:tc:mdita:core:xsd:topic.xsd",
	"urn:oasis:names:tc:mdita:extended:xsd:topic.xsd",
}

func Complete(doc *document.Document, pos document.Position, folder *workspace.Folder, graph *symbols.Graph) []CompletionItem {
	pe := DetectPartial(doc.Text, pos)
	if pe == nil {
		return nil
	}

	switch pe.Kind {
	case PartialWikiLink:
		return completeWikiDoc(pe.Input, doc, folder)
	case PartialWikiHeading:
		return completeWikiHeading(pe.DocPart, pe.Input, doc, folder)
	case PartialInlineLink:
		return completeInlineDoc(pe.Input, doc, folder)
	case PartialInlineAnchor:
		return completeInlineAnchor(pe.DocPart, pe.Input, doc, folder)
	case PartialYamlKey:
		return completeYamlKey(pe.Input)
	}
	return nil
}

func completeWikiDoc(input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	var items []CompletionItem
	inputSlug := paths.SlugOf(input)
	for _, d := range folder.AllDocs() {
		if d.URI == doc.URI {
			continue
		}
		id := d.DocID(folder.RootURI)
		if inputSlug == "" || id.Slug.Contains(inputSlug) {
			title := ""
			if t := d.Index.Title(); t != nil {
				title = t.Text
			}
			items = append(items, CompletionItem{
				Label:      id.Stem,
				Detail:     title,
				InsertText: id.Stem,
				Kind:       17,
			})
		}
	}
	return items
}

func completeWikiHeading(docPart, input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	var target *document.Document
	if docPart == "" {
		target = doc
	} else {
		target = folder.DocBySlug(paths.SlugOf(docPart))
	}
	if target == nil {
		return nil
	}

	inputSlug := paths.SlugOf(input)
	var items []CompletionItem
	for _, h := range target.Index.Headings() {
		if inputSlug == "" || h.Slug.Contains(inputSlug) {
			items = append(items, CompletionItem{
				Label:      h.ID,
				Detail:     h.Text,
				InsertText: h.ID,
				Kind:       17,
			})
		}
	}
	return items
}

func completeInlineDoc(input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	var items []CompletionItem
	for _, d := range folder.AllDocs() {
		if d.URI == doc.URI {
			continue
		}
		id := d.DocID(folder.RootURI)
		if input == "" || strings.Contains(strings.ToLower(id.RelPath), strings.ToLower(input)) {
			items = append(items, CompletionItem{
				Label:      id.RelPath,
				Detail:     id.Stem,
				InsertText: id.RelPath,
				Kind:       17,
			})
		}
	}
	return items
}

func completeInlineAnchor(docPart, input string, doc *document.Document, folder *workspace.Folder) []CompletionItem {
	return completeWikiHeading(docPart, input, doc, folder)
}

func completeYamlKey(input string) []CompletionItem {
	var items []CompletionItem
	for _, key := range yamlKeys {
		if input == "" || strings.HasPrefix(key, input) || strings.HasPrefix("$"+key, input) {
			items = append(items, CompletionItem{
				Label:      key,
				InsertText: key + ": ",
				Kind:       6,
			})
		}
	}
	return items
}
```

- [ ] **Step 5: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/completion/...
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd ~/mdita-lsp
git add internal/completion/
git commit -m "feat: completion — wiki links, inline links, YAML keys"
```

---

### Task 18: Rename Package

**Files:**
- Create: `internal/rename/rename.go`
- Create: `internal/rename/rename_test.go`

- [ ] **Step 1: Write rename tests**

Create `internal/rename/rename_test.go`:

```go
package rename

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestPrepareRename(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# My Heading\n\nText.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	heading := doc.Index.Title()
	result := Prepare(doc, heading.Range.Start)
	if result == nil {
		t.Fatal("Prepare returned nil")
	}
	if result.Text != "My Heading" {
		t.Errorf("Text = %q", result.Text)
	}
}

func TestPrepareRenameNonHeading(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nText.\n")
	result := Prepare(doc, document.Position{Line: 2, Character: 0})
	if result != nil {
		t.Error("Prepare should return nil for non-heading")
	}
}

func TestRename(t *testing.T) {
	doc1 := document.New("file:///project/intro.md", 1, "# Introduction\n\n## Details\n")
	doc2 := document.New("file:///project/guide.md", 1, "# Guide\n\n[[intro#details]]\n")

	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc1)
	f.AddDoc(doc2)
	g := symbols.NewGraph()
	for _, d := range f.AllDocs() {
		g.AddDefs(d.URI, d.Defs())
		g.AddRefs(d.URI, d.Refs())
	}

	heading := doc1.Index.Headings()[1]
	edits := DoRename(doc1, heading.Range.Start, "New Details", f, g)
	if len(edits) == 0 {
		t.Fatal("DoRename returned no edits")
	}

	foundHeading := false
	foundRef := false
	for _, e := range edits {
		if e.URI == doc1.URI {
			foundHeading = true
		}
		if e.URI == doc2.URI {
			foundRef = true
		}
	}
	if !foundHeading {
		t.Error("missing edit for heading document")
	}
	if !foundRef {
		t.Error("missing edit for referencing document")
	}
}
```

- [ ] **Step 2: Implement rename**

Create `internal/rename/rename.go`:

```go
package rename

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type PrepareResult struct {
	Range document.Range
	Text  string
}

type TextEdit struct {
	URI     string
	Range   document.Range
	NewText string
}

func Prepare(doc *document.Document, pos document.Position) *PrepareResult {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}
	heading, ok := elem.(*document.Heading)
	if !ok {
		return nil
	}
	return &PrepareResult{
		Range: heading.Range,
		Text:  heading.Text,
	}
}

func DoRename(doc *document.Document, pos document.Position, newName string, folder *workspace.Folder, graph *symbols.Graph) []TextEdit {
	elem := doc.ElementAt(pos)
	if elem == nil {
		return nil
	}
	heading, ok := elem.(*document.Heading)
	if !ok {
		return nil
	}

	var edits []TextEdit

	edits = append(edits, TextEdit{
		URI:     doc.URI,
		Range:   heading.Range,
		NewText: headingPrefix(heading.Level) + newName,
	})

	oldSlug := heading.Slug
	refs := graph.FindRefs(document.Symbol{Slug: oldSlug})
	newSlug := paths.Slugify(newName)

	for _, ref := range refs {
		refDoc := folder.DocByURI(ref.DocURI)
		if refDoc == nil {
			continue
		}
		elem := refDoc.ElementAt(ref.Range.Start)
		if elem == nil {
			continue
		}
		switch el := elem.(type) {
		case *document.WikiLink:
			if el.Heading != "" && paths.SlugOf(el.Heading) == oldSlug {
				edits = append(edits, TextEdit{
					URI:     ref.DocURI,
					Range:   el.Range,
					NewText: buildWikiLink(el.Doc, newSlug, el.Title),
				})
			}
		}
	}

	return edits
}

func headingPrefix(level int) string {
	s := ""
	for i := 0; i < level; i++ {
		s += "#"
	}
	return s + " "
}

func buildWikiLink(doc, heading, title string) string {
	s := "[["
	if doc != "" {
		s += doc
	}
	if heading != "" {
		s += "#" + heading
	}
	if title != "" {
		s += "|" + title
	}
	s += "]]"
	return s
}
```

- [ ] **Step 3: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/rename/...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
cd ~/mdita-lsp
git add internal/rename/
git commit -m "feat: rename refactoring for headings with cross-doc updates"
```

---

### Task 19: Code Actions Package

**Files:**
- Create: `internal/codeaction/codeaction.go`
- Create: `internal/codeaction/toc.go`
- Create: `internal/codeaction/codeaction_test.go`

- [ ] **Step 1: Write code action tests**

Create `internal/codeaction/codeaction_test.go`:

```go
package codeaction

import (
	"strings"
	"testing"

	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

func TestGenerateToC(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nIntro.\n\n## Section One\n\n### Subsection\n\n## Section Two\n")

	toc := GenerateToC(doc, []int{1, 2, 3, 4, 5, 6})
	if !strings.Contains(toc, "<!--toc:start-->") {
		t.Error("missing toc:start marker")
	}
	if !strings.Contains(toc, "<!--toc:end-->") {
		t.Error("missing toc:end marker")
	}
	if !strings.Contains(toc, "[Section One](#section-one)") {
		t.Errorf("missing Section One link, got:\n%s", toc)
	}
	if !strings.Contains(toc, "  - [Subsection](#subsection)") {
		t.Errorf("missing indented Subsection, got:\n%s", toc)
	}
}

func TestGetActions(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n\n[[missing-doc]]\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(0, 0, 10, 0), f)
	if len(actions) == 0 {
		t.Error("expected at least one code action (ToC)")
	}
	foundToC := false
	for _, a := range actions {
		if a.Title == "Generate table of contents" {
			foundToC = true
		}
	}
	if !foundToC {
		t.Error("missing ToC action")
	}
}
```

- [ ] **Step 2: Implement code actions**

Create `internal/codeaction/toc.go`:

```go
package codeaction

import (
	"fmt"
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/paths"
)

func GenerateToC(doc *document.Document, levels []int) string {
	levelSet := make(map[int]bool)
	for _, l := range levels {
		levelSet[l] = true
	}

	headings := doc.Index.Headings()
	var filtered []*document.Heading
	for _, h := range headings {
		if levelSet[h.Level] {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) == 0 {
		return ""
	}

	minLevel := filtered[0].Level
	for _, h := range filtered {
		if h.Level < minLevel {
			minLevel = h.Level
		}
	}

	var sb strings.Builder
	sb.WriteString("<!--toc:start-->\n")
	for _, h := range filtered {
		indent := strings.Repeat("  ", h.Level-minLevel)
		slug := paths.Slugify(h.Text)
		sb.WriteString(fmt.Sprintf("%s- [%s](#%s)\n", indent, h.Text, slug))
	}
	sb.WriteString("<!--toc:end-->")

	return sb.String()
}
```

Create `internal/codeaction/codeaction.go`:

```go
package codeaction

import (
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type CodeAction struct {
	Title   string
	Kind    string
	DocURI  string
	Edit    *TextEdit
	Command *Command
}

type TextEdit struct {
	Range   document.Range
	NewText string
}

type Command struct {
	Title     string
	Command   string
	Arguments []string
}

func GetActions(doc *document.Document, rng document.Range, folder *workspace.Folder) []CodeAction {
	var actions []CodeAction
	cfg := folder.Config

	if cfg.CodeActions.ToC.Enable {
		actions = append(actions, CodeAction{
			Title:  "Generate table of contents",
			Kind:   "source",
			DocURI: doc.URI,
		})
	}

	if cfg.CodeActions.CreateMissingFile.Enable {
		for _, wl := range doc.Index.WikiLinks() {
			if rangesOverlap(rng, wl.Range) && wl.Doc != "" {
				target := folder.DocBySlug(paths.SlugOf(wl.Doc))
				if target == nil {
					actions = append(actions, CodeAction{
						Title:  "Create '" + wl.Doc + ".md'",
						Kind:   "quickfix",
						DocURI: doc.URI,
						Command: &Command{
							Title:     "Create file",
							Command:   "mdita-lsp.createFile",
							Arguments: []string{wl.Doc + ".md"},
						},
					})
				}
			}
		}
	}

	return actions
}

func rangesOverlap(a, b document.Range) bool {
	if a.End.Line < b.Start.Line || b.End.Line < a.Start.Line {
		return false
	}
	return true
}
```

- [ ] **Step 3: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/codeaction/...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
cd ~/mdita-lsp
git add internal/codeaction/
git commit -m "feat: code actions — ToC generation, create missing file"
```

---

### Task 20: Code Lens, Document Symbols, Semantic Tokens

**Files:**
- Create: `internal/codelens/codelens.go`
- Create: `internal/docsymbols/docsymbols.go`
- Create: `internal/semantic/semantic.go`
- Create: `internal/codelens/codelens_test.go`
- Create: `internal/docsymbols/docsymbols_test.go`
- Create: `internal/semantic/semantic_test.go`

- [ ] **Step 1: Write and implement code lens**

Create `internal/codelens/codelens_test.go`:

```go
package codelens

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
)

func TestCodeLens(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\n## Section\n")
	g := symbols.NewGraph()
	g.AddDefs(doc.URI, doc.Defs())

	lenses := GetLenses(doc, g)
	if len(lenses) < 2 {
		t.Errorf("got %d lenses, want >= 2", len(lenses))
	}
}
```

Create `internal/codelens/codelens.go`:

```go
package codelens

import (
	"fmt"

	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/symbols"
)

type Lens struct {
	Range   document.Range
	Command string
	Title   string
}

func GetLenses(doc *document.Document, graph *symbols.Graph) []Lens {
	var lenses []Lens
	for _, h := range doc.Index.Headings() {
		target := document.Symbol{Slug: h.Slug}
		refs := graph.FindRefs(target)
		lenses = append(lenses, Lens{
			Range:   h.Range,
			Command: "mdita-lsp.findReferences",
			Title:   fmt.Sprintf("%d references", len(refs)),
		})
	}
	return lenses
}
```

- [ ] **Step 2: Write and implement document symbols**

Create `internal/docsymbols/docsymbols_test.go`:

```go
package docsymbols

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestDocumentSymbols(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\n## Section\n\n### Sub\n\n## Other\n")
	syms := GetSymbols(doc)

	if len(syms) == 0 {
		t.Fatal("expected symbols")
	}
	if syms[0].Name != "Title" {
		t.Errorf("syms[0].Name = %q", syms[0].Name)
	}
}

func TestWorkspaceSymbols(t *testing.T) {
	docs := []*document.Document{
		document.New("file:///project/a.md", 1, "# Alpha\n"),
		document.New("file:///project/b.md", 1, "# Beta\n"),
	}
	syms := SearchWorkspace(docs, "alp")
	if len(syms) != 1 || syms[0].Name != "Alpha" {
		t.Errorf("SearchWorkspace = %v", syms)
	}
}
```

Create `internal/docsymbols/docsymbols.go`:

```go
package docsymbols

import (
	"strings"

	"github.com/aireilly/mdita-lsp/internal/document"
)

type DocSymbol struct {
	Name     string
	Kind     int // 5=class (file), 23=struct (section)
	Range    document.Range
	Children []DocSymbol
}

func GetSymbols(doc *document.Document) []DocSymbol {
	headings := doc.Index.Headings()
	if len(headings) == 0 {
		return nil
	}

	var root []DocSymbol
	var stack [](*[]DocSymbol)
	var levels []int

	stack = append(stack, &root)
	levels = append(levels, 0)

	for _, h := range headings {
		sym := DocSymbol{
			Name:  h.Text,
			Kind:  23,
			Range: h.Range,
		}
		if h.IsTitle() {
			sym.Kind = 5
		}

		for len(levels) > 1 && h.Level <= levels[len(levels)-1] {
			stack = stack[:len(stack)-1]
			levels = levels[:len(levels)-1]
		}

		parent := stack[len(stack)-1]
		*parent = append(*parent, sym)
		idx := len(*parent) - 1
		stack = append(stack, &(*parent)[idx].Children)
		levels = append(levels, h.Level)
	}

	return root
}

func SearchWorkspace(docs []*document.Document, query string) []DocSymbol {
	query = strings.ToLower(query)
	var results []DocSymbol
	for _, doc := range docs {
		for _, h := range doc.Index.Headings() {
			if strings.Contains(strings.ToLower(h.Text), query) {
				results = append(results, DocSymbol{
					Name:  h.Text,
					Kind:  23,
					Range: h.Range,
				})
			}
		}
	}
	return results
}
```

- [ ] **Step 3: Write and implement semantic tokens**

Create `internal/semantic/semantic_test.go`:

```go
package semantic

import (
	"testing"

	"github.com/aireilly/mdita-lsp/internal/document"
)

func TestSemanticTokens(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\n[[other]]\n")
	data := Encode(doc)
	if len(data) == 0 {
		t.Error("expected semantic token data")
	}
	if len(data)%5 != 0 {
		t.Errorf("data length %d not a multiple of 5", len(data))
	}
}

func TestSemanticTokensEmpty(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1, "# Title\n\nNo links.\n")
	data := Encode(doc)
	if len(data) != 0 {
		t.Errorf("expected empty data for no links, got %d", len(data))
	}
}
```

Create `internal/semantic/semantic.go`:

```go
package semantic

import (
	"sort"

	"github.com/aireilly/mdita-lsp/internal/document"
)

const (
	TokenTypeWikiLink = 0
	TokenTypeRefLink  = 1
)

var TokenTypes = []string{"class", "class"}

type token struct {
	line   int
	char   int
	length int
	typ    int
}

func Encode(doc *document.Document) []uint32 {
	var tokens []token

	for _, wl := range doc.Index.WikiLinks() {
		r := wl.Range
		if r.Start.Line == r.End.Line {
			tokens = append(tokens, token{
				line:   r.Start.Line,
				char:   r.Start.Character,
				length: r.End.Character - r.Start.Character,
				typ:    TokenTypeWikiLink,
			})
		}
	}

	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].line != tokens[j].line {
			return tokens[i].line < tokens[j].line
		}
		return tokens[i].char < tokens[j].char
	})

	var data []uint32
	prevLine := 0
	prevChar := 0
	for _, tok := range tokens {
		deltaLine := tok.line - prevLine
		deltaChar := tok.char
		if deltaLine == 0 {
			deltaChar = tok.char - prevChar
		}
		data = append(data, uint32(deltaLine), uint32(deltaChar), uint32(tok.length), uint32(tok.typ), 0)
		prevLine = tok.line
		prevChar = tok.char
	}
	return data
}
```

- [ ] **Step 4: Run all tests**

```bash
cd ~/mdita-lsp && go test ./internal/codelens/... ./internal/docsymbols/... ./internal/semantic/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/codelens/ internal/docsymbols/ internal/semantic/
git commit -m "feat: code lens, document symbols, semantic tokens"
```

---

### Task 21: LSP Server

**Files:**
- Create: `internal/lsp/server.go`
- Create: `internal/lsp/handler.go`
- Create: `internal/lsp/diagnostics.go`
- Create: `internal/lsp/server_test.go`

- [ ] **Step 1: Write server integration tests**

Create `internal/lsp/server_test.go`:

```go
package lsp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestInitializeResponse(t *testing.T) {
	s := NewServer()
	params := json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test"
	}`)

	result, err := s.handleInitialize(context.Background(), params)
	if err != nil {
		t.Fatalf("Initialize error: %v", err)
	}

	var initResult InitializeResult
	data, _ := json.Marshal(result)
	json.Unmarshal(data, &initResult)

	if initResult.Capabilities.CompletionProvider == nil {
		t.Error("missing completion provider")
	}
	if !initResult.Capabilities.DefinitionProvider {
		t.Error("missing definition provider")
	}
	if !initResult.Capabilities.HoverProvider {
		t.Error("missing hover provider")
	}
	if !initResult.Capabilities.ReferencesProvider {
		t.Error("missing references provider")
	}
	if !initResult.Capabilities.RenameProvider {
		t.Error("missing rename provider")
	}
}

func TestDidOpenAndDiagnostics(t *testing.T) {
	s := NewServer()

	s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test"
	}`))

	openParams := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///tmp/test/doc.md",
			"languageId": "markdown",
			"version": 1,
			"text": "# Title\n\n[[nonexistent]]\n"
		}
	}`)

	err := s.handleDidOpen(context.Background(), openParams)
	if err != nil {
		t.Fatalf("DidOpen error: %v", err)
	}

	doc, _ := s.workspace.FindDoc("file:///tmp/test/doc.md")
	if doc == nil {
		t.Fatal("document not found after open")
	}
}

func TestCompletion(t *testing.T) {
	s := NewServer()
	s.handleInitialize(context.Background(), json.RawMessage(`{
		"capabilities": {},
		"rootUri": "file:///tmp/test"
	}`))

	s.handleDidOpen(context.Background(), json.RawMessage(`{
		"textDocument": {
			"uri": "file:///tmp/test/intro.md",
			"languageId": "markdown",
			"version": 1,
			"text": "# Introduction\n\nContent.\n"
		}
	}`))

	s.handleDidOpen(context.Background(), json.RawMessage(`{
		"textDocument": {
			"uri": "file:///tmp/test/doc.md",
			"languageId": "markdown",
			"version": 1,
			"text": "# Doc\n\n[[int"
		}
	}`))

	result, err := s.handleCompletion(context.Background(), json.RawMessage(`{
		"textDocument": {"uri": "file:///tmp/test/doc.md"},
		"position": {"line": 2, "character": 5}
	}`))
	if err != nil {
		t.Fatalf("Completion error: %v", err)
	}
	items, ok := result.([]CompletionItemResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if len(items) == 0 {
		t.Error("expected completion items")
	}
}
```

- [ ] **Step 2: Implement LSP types and server**

Create `internal/lsp/server.go`:

```go
package lsp

import (
	"context"
	"encoding/json"

	"github.com/aireilly/mdita-lsp/internal/codeaction"
	"github.com/aireilly/mdita-lsp/internal/codelens"
	"github.com/aireilly/mdita-lsp/internal/completion"
	"github.com/aireilly/mdita-lsp/internal/config"
	"github.com/aireilly/mdita-lsp/internal/definition"
	"github.com/aireilly/mdita-lsp/internal/diagnostic"
	"github.com/aireilly/mdita-lsp/internal/document"
	"github.com/aireilly/mdita-lsp/internal/docsymbols"
	"github.com/aireilly/mdita-lsp/internal/hover"
	"github.com/aireilly/mdita-lsp/internal/references"
	"github.com/aireilly/mdita-lsp/internal/rename"
	"github.com/aireilly/mdita-lsp/internal/semantic"
	"github.com/aireilly/mdita-lsp/internal/symbols"
	"github.com/aireilly/mdita-lsp/internal/workspace"
)

type Server struct {
	workspace *workspace.Workspace
	graph     *symbols.Graph
	notify    func(method string, params interface{})
}

func NewServer() *Server {
	return &Server{
		workspace: workspace.New(),
		graph:     symbols.NewGraph(),
		notify:    func(string, interface{}) {},
	}
}

func (s *Server) SetNotify(fn func(method string, params interface{})) {
	s.notify = fn
}

type InitializeParams struct {
	RootURI      string          `json:"rootUri"`
	Capabilities json.RawMessage `json:"capabilities"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	TextDocumentSync   int                 `json:"textDocumentSync"`
	CompletionProvider *CompletionOptions  `json:"completionProvider,omitempty"`
	DefinitionProvider bool                `json:"definitionProvider"`
	HoverProvider      bool                `json:"hoverProvider"`
	ReferencesProvider bool                `json:"referencesProvider"`
	RenameProvider     bool                `json:"renameProvider"`
	CodeActionProvider bool                `json:"codeActionProvider"`
	CodeLensProvider   *CodeLensOptions    `json:"codeLensProvider,omitempty"`
	DocumentSymbolProvider bool            `json:"documentSymbolProvider"`
	WorkspaceSymbolProvider bool           `json:"workspaceSymbolProvider"`
	SemanticTokensProvider *SemanticTokensOptions `json:"semanticTokensProvider,omitempty"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type SemanticTokensOptions struct {
	Full   bool              `json:"full"`
	Legend SemanticTokensLegend `json:"legend"`
}

type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type TextDocumentItem struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
	Text    string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     document.Position      `json:"position"`
}

type DidOpenParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeParams struct {
	TextDocument struct {
		URI     string `json:"uri"`
		Version int    `json:"version"`
	} `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type DidCloseParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type CompletionItemResult struct {
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
	Kind       int    `json:"kind"`
}

type LocationResult struct {
	URI   string          `json:"uri"`
	Range document.Range  `json:"range"`
}

type HoverResult struct {
	Contents string         `json:"contents"`
	Range    document.Range `json:"range,omitempty"`
}

type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     document.Position      `json:"position"`
	NewName      string                 `json:"newName"`
}

type DiagnosticParams struct {
	URI         string                  `json:"uri"`
	Diagnostics []DiagnosticResult      `json:"diagnostics"`
}

type DiagnosticResult struct {
	Range    document.Range `json:"range"`
	Severity int            `json:"severity"`
	Code     string         `json:"code"`
	Source   string         `json:"source"`
	Message  string         `json:"message"`
}

func (s *Server) handleInitialize(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params InitializeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	if params.RootURI != "" {
		cfg := config.Default()
		folder := workspace.NewFolder(params.RootURI, cfg)
		folder.ScanFiles()
		s.workspace.AddFolder(folder)

		for _, doc := range folder.AllDocs() {
			s.graph.AddDefs(doc.URI, doc.Defs())
			s.graph.AddRefs(doc.URI, doc.Refs())
		}
	}

	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: 1,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"[", "#", "("},
			},
			DefinitionProvider:      true,
			HoverProvider:           true,
			ReferencesProvider:      true,
			RenameProvider:          true,
			CodeActionProvider:      true,
			CodeLensProvider:        &CodeLensOptions{},
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			SemanticTokensProvider: &SemanticTokensOptions{
				Full: true,
				Legend: SemanticTokensLegend{
					TokenTypes:     semantic.TokenTypes,
					TokenModifiers: []string{},
				},
			},
		},
	}, nil
}

func (s *Server) handleDidOpen(_ context.Context, rawParams json.RawMessage) error {
	var params DidOpenParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	doc := document.New(params.TextDocument.URI, params.TextDocument.Version, params.TextDocument.Text)

	folder := s.workspace.FolderForURI(doc.URI)
	if folder == nil {
		cfg := config.Default()
		folder = workspace.NewFolder(parentURI(doc.URI), cfg)
		s.workspace.AddFolder(folder)
	}

	folder.AddDoc(doc)
	s.graph.AddDefs(doc.URI, doc.Defs())
	s.graph.AddRefs(doc.URI, doc.Refs())
	s.publishDiagnostics(doc, folder)

	return nil
}

func (s *Server) handleDidChange(_ context.Context, rawParams json.RawMessage) error {
	var params DidChangeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil
	}

	if len(params.ContentChanges) > 0 {
		newDoc := doc.ApplyChange(params.TextDocument.Version, params.ContentChanges[0].Text)
		folder.AddDoc(newDoc)
		s.graph.AddDefs(newDoc.URI, newDoc.Defs())
		s.graph.AddRefs(newDoc.URI, newDoc.Refs())
		s.publishDiagnostics(newDoc, folder)
	}
	return nil
}

func (s *Server) handleDidClose(_ context.Context, rawParams json.RawMessage) error {
	var params DidCloseParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleCompletion(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return []CompletionItemResult{}, nil
	}

	items := completion.Complete(doc, params.Position, folder, s.graph)
	var results []CompletionItemResult
	for _, item := range items {
		results = append(results, CompletionItemResult{
			Label:      item.Label,
			Detail:     item.Detail,
			InsertText: item.InsertText,
			Kind:       item.Kind,
		})
	}
	return results, nil
}

func (s *Server) handleDefinition(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	locs := definition.GotoDef(doc, params.Position, folder, s.graph)
	var results []LocationResult
	for _, loc := range locs {
		results = append(results, LocationResult{URI: loc.URI, Range: loc.Range})
	}
	return results, nil
}

func (s *Server) handleHover(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	content := hover.GetHover(doc, params.Position, folder, s.graph)
	if content == "" {
		return nil, nil
	}
	return HoverResult{Contents: content}, nil
}

func (s *Server) handleReferences(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	locs := references.FindRefs(doc, params.Position, folder, s.graph)
	var results []LocationResult
	for _, loc := range locs {
		results = append(results, LocationResult{URI: loc.URI, Range: loc.Range})
	}
	return results, nil
}

func (s *Server) handleRename(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params RenameParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	edits := rename.DoRename(doc, params.Position, params.NewName, folder, s.graph)
	result := make(map[string][]map[string]interface{})
	for _, edit := range edits {
		result[edit.URI] = append(result[edit.URI], map[string]interface{}{
			"range":   edit.Range,
			"newText": edit.NewText,
		})
	}
	return map[string]interface{}{"changes": result}, nil
}

func (s *Server) handleCodeAction(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Range        document.Range         `json:"range"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, folder := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil || folder == nil {
		return nil, nil
	}

	actions := codeaction.GetActions(doc, params.Range, folder)
	var results []map[string]interface{}
	for _, a := range actions {
		results = append(results, map[string]interface{}{
			"title": a.Title,
			"kind":  a.Kind,
		})
	}
	return results, nil
}

func (s *Server) handleCodeLens(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	lenses := codelens.GetLenses(doc, s.graph)
	var results []map[string]interface{}
	for _, l := range lenses {
		results = append(results, map[string]interface{}{
			"range": l.Range,
			"command": map[string]interface{}{
				"title":   l.Title,
				"command": l.Command,
			},
		})
	}
	return results, nil
}

func (s *Server) handleDocumentSymbol(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	syms := docsymbols.GetSymbols(doc)
	return syms, nil
}

func (s *Server) handleSemanticTokensFull(_ context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	doc, _ := s.workspace.FindDoc(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	data := semantic.Encode(doc)
	return map[string]interface{}{"data": data}, nil
}

func (s *Server) publishDiagnostics(doc *document.Document, folder *workspace.Folder) {
	diags := diagnostic.Check(doc, folder)
	var results []DiagnosticResult
	for _, d := range diags {
		results = append(results, DiagnosticResult{
			Range:    d.Range,
			Severity: int(d.Severity),
			Code:     d.Code,
			Source:   d.Source,
			Message:  d.Message,
		})
	}
	s.notify("textDocument/publishDiagnostics", DiagnosticParams{
		URI:         doc.URI,
		Diagnostics: results,
	})
}

func parentURI(uri string) string {
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == '/' {
			return uri[:i]
		}
	}
	return uri
}
```

- [ ] **Step 3: Implement JSON-RPC handler dispatch**

Create `internal/lsp/handler.go`:

```go
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *ResponseError   `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Split(scanLSPMessages)

	s.SetNotify(func(method string, params interface{}) {
		notif := Notification{
			JSONRPC: "2.0",
			Method:  method,
			Params:  params,
		}
		writeMessage(out, notif)
	})

	for scanner.Scan() {
		body := scanner.Bytes()
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("unmarshal error: %v", err)
			continue
		}

		if req.ID != nil {
			result, err := s.dispatch(ctx, req.Method, req.Params)
			resp := Response{JSONRPC: "2.0", ID: req.ID}
			if err != nil {
				resp.Error = &ResponseError{Code: -32603, Message: err.Error()}
			} else {
				resp.Result = result
			}
			writeMessage(out, resp)
		} else {
			s.dispatchNotification(ctx, req.Method, req.Params)
		}
	}
	return scanner.Err()
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "initialize":
		return s.handleInitialize(ctx, params)
	case "textDocument/completion":
		return s.handleCompletion(ctx, params)
	case "textDocument/definition":
		return s.handleDefinition(ctx, params)
	case "textDocument/hover":
		return s.handleHover(ctx, params)
	case "textDocument/references":
		return s.handleReferences(ctx, params)
	case "textDocument/rename":
		return s.handleRename(ctx, params)
	case "textDocument/codeAction":
		return s.handleCodeAction(ctx, params)
	case "textDocument/codeLens":
		return s.handleCodeLens(ctx, params)
	case "textDocument/documentSymbol":
		return s.handleDocumentSymbol(ctx, params)
	case "textDocument/semanticTokens/full":
		return s.handleSemanticTokensFull(ctx, params)
	case "shutdown":
		return nil, nil
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func (s *Server) dispatchNotification(ctx context.Context, method string, params json.RawMessage) {
	switch method {
	case "initialized":
		// no-op
	case "textDocument/didOpen":
		s.handleDidOpen(ctx, params)
	case "textDocument/didChange":
		s.handleDidChange(ctx, params)
	case "textDocument/didClose":
		s.handleDidClose(ctx, params)
	case "exit":
		// handled by caller
	}
}

func writeMessage(w io.Writer, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	_, err = io.WriteString(w, header)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func scanLSPMessages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	headerEnd := strings.Index(string(data), "\r\n\r\n")
	if headerEnd < 0 {
		if atEOF {
			return 0, nil, fmt.Errorf("incomplete header")
		}
		return 0, nil, nil
	}

	header := string(data[:headerEnd])
	contentLength := 0
	for _, line := range strings.Split(header, "\r\n") {
		if strings.HasPrefix(line, "Content-Length: ") {
			cl := strings.TrimPrefix(line, "Content-Length: ")
			contentLength, _ = strconv.Atoi(strings.TrimSpace(cl))
		}
	}

	if contentLength == 0 {
		return 0, nil, fmt.Errorf("missing Content-Length")
	}

	totalLen := headerEnd + 4 + contentLength
	if len(data) < totalLen {
		if atEOF {
			return 0, nil, fmt.Errorf("incomplete message")
		}
		return 0, nil, nil
	}

	body := data[headerEnd+4 : totalLen]
	return totalLen, body, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd ~/mdita-lsp && go test ./internal/lsp/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add internal/lsp/
git commit -m "feat: LSP server with JSON-RPC handler dispatch and all LSP methods"
```

---

### Task 22: Entry Point and Full Integration

**Files:**
- Modify: `cmd/mdita-lsp/main.go`
- Create: `internal/lsp/integration_test.go`

- [ ] **Step 1: Write integration test**

Create `internal/lsp/integration_test.go`:

```go
package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func buildMessage(method string, id *int, params interface{}) string {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if id != nil {
		req["id"] = *id
	}
	if params != nil {
		req["params"] = params
	}
	body, _ := json.Marshal(req)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func intPtr(i int) *int { return &i }

func TestFullLSPLifecycle(t *testing.T) {
	var input bytes.Buffer
	var output bytes.Buffer

	input.WriteString(buildMessage("initialize", intPtr(1), map[string]interface{}{
		"capabilities": map[string]interface{}{},
		"rootUri":      "file:///tmp/lsp-test",
	}))

	input.WriteString(buildMessage("initialized", nil, nil))

	input.WriteString(buildMessage("textDocument/didOpen", nil, map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        "file:///tmp/lsp-test/doc.md",
			"languageId": "markdown",
			"version":    1,
			"text":       "# Hello World\n\n## Section\n\nSome [[#section]] link.\n",
		},
	}))

	input.WriteString(buildMessage("textDocument/hover", intPtr(2), map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": "file:///tmp/lsp-test/doc.md"},
		"position":     map[string]interface{}{"line": 0, "character": 3},
	}))

	input.WriteString(buildMessage("textDocument/documentSymbol", intPtr(3), map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": "file:///tmp/lsp-test/doc.md"},
	}))

	input.WriteString(buildMessage("shutdown", intPtr(4), nil))

	s := NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.Serve(ctx, &input, &output)
	if err != nil && !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("Serve error: %v", err)
	}

	out := output.String()
	if !strings.Contains(out, "\"id\":1") {
		t.Error("missing initialize response")
	}
	if !strings.Contains(out, "completionProvider") {
		t.Error("missing capabilities in initialize response")
	}
}
```

- [ ] **Step 2: Update main.go entry point**

Update `cmd/mdita-lsp/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aireilly/mdita-lsp/internal/lsp"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version)
			os.Exit(0)
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: mdita-lsp [--version] [--help]")
			fmt.Fprintln(os.Stderr, "  Runs as an LSP server over stdio.")
			os.Exit(0)
		}
	}

	logFile, err := os.CreateTemp("", "mdita-lsp-*.log")
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Printf("mdita-lsp %s starting", version)

	s := lsp.NewServer()
	ctx := context.Background()
	if err := s.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 3: Run all tests**

```bash
cd ~/mdita-lsp && go test -race ./...
```

Expected: ALL PASS

- [ ] **Step 4: Verify build**

```bash
cd ~/mdita-lsp && make build && ./mdita-lsp --version
```

Expected: prints `dev`

- [ ] **Step 5: Commit**

```bash
cd ~/mdita-lsp
git add cmd/mdita-lsp/main.go internal/lsp/integration_test.go
git commit -m "feat: entry point and integration tests — server ready for use"
```

---

### Task 23: Test Fixtures and End-to-End Verification

**Files:**
- Create: `testdata/diagnostics/missing-yaml.md`
- Create: `testdata/diagnostics/valid-task.md`
- Create: `testdata/ditamap/valid.mditamap`
- Create: `testdata/ditamap/broken-ref.mditamap`
- Create: `testdata/config/full.yaml`

- [ ] **Step 1: Create test fixtures**

Create `testdata/diagnostics/missing-yaml.md`:

```markdown
# No YAML Front Matter

This document has no YAML front matter.
```

Create `testdata/diagnostics/valid-task.md`:

```markdown
---
author: Test Author
$schema: urn:oasis:names:tc:dita:xsd:task.xsd
---
# Installing the Software

Follow these steps to install.

1. Download the package.
2. Run the installer.
3. Verify the installation.
```

Create `testdata/ditamap/valid.mditamap`:

```markdown
# Product Documentation

- [Installation](../diagnostics/valid-task.md)
```

Create `testdata/ditamap/broken-ref.mditamap`:

```markdown
# Broken Map

- [Missing Topic](nonexistent-file.md)
```

Create `testdata/config/full.yaml`:

```yaml
core:
  markdown:
    file_extensions: [md, markdown, mditamap]
    text_sync: incremental
    title_from_heading: true
  mdita:
    enable: true
    map_extensions: [mditamap]
completion:
  wiki_style: file-stem
  max_candidates: 100
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

- [ ] **Step 2: Run full test suite**

```bash
cd ~/mdita-lsp && go test -race -count=1 ./...
```

Expected: ALL PASS

- [ ] **Step 3: Run build for all platforms**

```bash
cd ~/mdita-lsp && CGO_ENABLED=0 go build -o mdita-lsp ./cmd/mdita-lsp && echo "Build OK"
```

Expected: `Build OK`

- [ ] **Step 4: Commit fixtures**

```bash
cd ~/mdita-lsp
git add testdata/
git commit -m "feat: test fixtures for diagnostics, ditamap, and config"
```

- [ ] **Step 5: Final commit — clean up go.sum**

```bash
cd ~/mdita-lsp && go mod tidy && git add go.mod go.sum && git diff --cached --stat
```

If changes:

```bash
git commit -m "chore: tidy go.mod"
```

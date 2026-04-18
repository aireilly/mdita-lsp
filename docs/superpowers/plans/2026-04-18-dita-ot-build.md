# DITA OT Build Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Shell out to the DITA OT `dita` CLI to build XHTML output from MDITA files, triggered by an LSP code action on `.mditamap` documents.

**Architecture:** New `internal/ditaot/` package handles binary resolution and build invocation with a mutex-based concurrency guard. The LSP server registers `mdita-lsp.ditaOtBuild` as an execute command, wired to a code action on map files. Build runs async in a goroutine; results reported via `window/showMessage` and `window/logMessage`.

**Tech Stack:** Go stdlib (`os/exec`, `sync`, `context`), existing config/workspace/codeaction infrastructure.

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `internal/ditaot/resolve.go` | Find the `dita` binary (configured path or `$PATH`) |
| Create | `internal/ditaot/resolve_test.go` | Tests for binary resolution |
| Create | `internal/ditaot/build.go` | `Builder` type with `TryAcquire`/`Release`/`Run` |
| Create | `internal/ditaot/build_test.go` | Unit + integration tests for build |
| Modify | `internal/config/config.go` | Add `BuildConfig` and `DitaOTConfig` types, defaults, merge |
| Modify | `internal/config/config_test.go` | Test parsing and merging of new config fields |
| Modify | `internal/codeaction/codeaction.go` | Add "Build XHTML with DITA OT" code action for map documents |
| Modify | `internal/codeaction/codeaction_test.go` | Tests for the new code action |
| Modify | `internal/lsp/server.go` | Register command, add `ditaBuilder` field, implement `executeDitaOtBuild` |
| Modify | `testdata/config/full.yaml` | Add `build` section to test fixture |
| Modify | `.github/workflows/ci.yml` | Add DITA OT install + integration test job |

---

### Task 1: Configuration — Types and Defaults

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Write the failing test for parsing build config**

Add to `internal/config/config_test.go`:

```go
func TestParseBuildConfig(t *testing.T) {
	input := `
build:
  dita_ot:
    enable: true
    dita_path: "/opt/dita-ot/bin/dita"
    output_dir: "build-output"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if !BoolVal(cfg.Build.DitaOT.Enable) {
		t.Error("Build.DitaOT.Enable should be true")
	}
	if cfg.Build.DitaOT.DitaPath != "/opt/dita-ot/bin/dita" {
		t.Errorf("DitaPath = %q, want %q", cfg.Build.DitaOT.DitaPath, "/opt/dita-ot/bin/dita")
	}
	if cfg.Build.DitaOT.OutputDir != "build-output" {
		t.Errorf("OutputDir = %q, want %q", cfg.Build.DitaOT.OutputDir, "build-output")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestParseBuildConfig -v`
Expected: FAIL — `cfg.Build` field does not exist.

- [ ] **Step 3: Add config types and defaults**

In `internal/config/config.go`, add after the `DiagnosticsConfig` type:

```go
type BuildConfig struct {
	DitaOT DitaOTConfig `yaml:"dita_ot"`
}

type DitaOTConfig struct {
	Enable    *bool  `yaml:"enable"`
	DitaPath  string `yaml:"dita_path,omitempty"`
	OutputDir string `yaml:"output_dir,omitempty"`
}
```

Add the `Build` field to the `Config` struct:

```go
type Config struct {
	Core        CoreConfig        `yaml:"core"`
	Completion  CompletionConfig  `yaml:"completion"`
	CodeActions CodeActionsConfig `yaml:"code_actions"`
	Diagnostics DiagnosticsConfig `yaml:"diagnostics"`
	Build       BuildConfig       `yaml:"build"`
}
```

In `Default()`, add inside the returned `&Config{}`:

```go
Build: BuildConfig{
	DitaOT: DitaOTConfig{
		Enable:    boolPtr(true),
		OutputDir: "out",
	},
},
```

In `Merge()`, add at the end before `return &merged`:

```go
merged.Build.DitaOT.Enable = mergeBool(base.Build.DitaOT.Enable, overlay.Build.DitaOT.Enable)
if overlay.Build.DitaOT.DitaPath != "" {
	merged.Build.DitaOT.DitaPath = overlay.Build.DitaOT.DitaPath
}
if overlay.Build.DitaOT.OutputDir != "" {
	merged.Build.DitaOT.OutputDir = overlay.Build.DitaOT.OutputDir
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestParseBuildConfig -v`
Expected: PASS

- [ ] **Step 5: Write test for build config defaults**

Add to `internal/config/config_test.go`, inside `TestDefault`:

```go
if !BoolVal(cfg.Build.DitaOT.Enable) {
	t.Error("default Build.DitaOT.Enable should be true")
}
if cfg.Build.DitaOT.OutputDir != "out" {
	t.Errorf("default OutputDir = %q, want %q", cfg.Build.DitaOT.OutputDir, "out")
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestDefault -v`
Expected: PASS

- [ ] **Step 7: Write test for build config merge**

Add to `internal/config/config_test.go`:

```go
func TestMergeBuildConfig(t *testing.T) {
	base := Default()
	overlay, _ := Parse([]byte(`
build:
  dita_ot:
    enable: false
    dita_path: "/custom/dita"
    output_dir: "custom-out"
`))
	merged := Merge(base, overlay)
	if BoolVal(merged.Build.DitaOT.Enable) {
		t.Error("merged Build.DitaOT.Enable should be false")
	}
	if merged.Build.DitaOT.DitaPath != "/custom/dita" {
		t.Errorf("merged DitaPath = %q, want %q", merged.Build.DitaOT.DitaPath, "/custom/dita")
	}
	if merged.Build.DitaOT.OutputDir != "custom-out" {
		t.Errorf("merged OutputDir = %q, want %q", merged.Build.DitaOT.OutputDir, "custom-out")
	}
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestMergeBuildConfig -v`
Expected: PASS

- [ ] **Step 9: Update test fixture**

Add to the end of `testdata/config/full.yaml`:

```yaml
build:
  dita_ot:
    enable: true
    dita_path: ""
    output_dir: "out"
```

- [ ] **Step 10: Run full config test suite**

Run: `go test ./internal/config/ -v`
Expected: All tests PASS

- [ ] **Step 11: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go testdata/config/full.yaml
git commit -m "feat: add build.dita_ot config types, defaults, and merge logic"
```

---

### Task 2: Binary Resolution — `internal/ditaot/resolve.go`

**Files:**
- Create: `internal/ditaot/resolve.go`
- Create: `internal/ditaot/resolve_test.go`

- [ ] **Step 1: Write failing tests for ResolveDitaPath**

Create `internal/ditaot/resolve_test.go`:

```go
package ditaot

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveDitaPath_Configured(t *testing.T) {
	tmp := t.TempDir()
	fakeDita := filepath.Join(tmp, "dita")
	if runtime.GOOS == "windows" {
		fakeDita += ".exe"
	}
	if err := os.WriteFile(fakeDita, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveDitaPath(fakeDita)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fakeDita {
		t.Errorf("got %q, want %q", got, fakeDita)
	}
}

func TestResolveDitaPath_ConfiguredNotFound(t *testing.T) {
	_, err := ResolveDitaPath("/nonexistent/dita")
	if err == nil {
		t.Fatal("expected error for missing configured path")
	}
}

func TestResolveDitaPath_EmptyFallbackNotOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := ResolveDitaPath("")
	if err == nil {
		t.Fatal("expected error when dita not on PATH")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ditaot/ -run TestResolveDitaPath -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement ResolveDitaPath**

Create `internal/ditaot/resolve.go`:

```go
package ditaot

import (
	"fmt"
	"os"
	"os/exec"
)

func ResolveDitaPath(configured string) (string, error) {
	if configured != "" {
		info, err := os.Stat(configured)
		if err != nil {
			return "", fmt.Errorf("configured dita path not found: %s", configured)
		}
		if info.IsDir() {
			return "", fmt.Errorf("configured dita path is a directory: %s", configured)
		}
		return configured, nil
	}
	path, err := exec.LookPath("dita")
	if err != nil {
		return "", fmt.Errorf("dita binary not found: install DITA OT and add to $PATH, or set build.dita_ot.dita_path in config")
	}
	return path, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ditaot/ -run TestResolveDitaPath -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ditaot/resolve.go internal/ditaot/resolve_test.go
git commit -m "feat: add ditaot.ResolveDitaPath for dita binary discovery"
```

---

### Task 3: Builder — `internal/ditaot/build.go`

**Files:**
- Create: `internal/ditaot/build.go`
- Create: `internal/ditaot/build_test.go`

- [ ] **Step 1: Write failing tests for TryAcquire/Release and Run**

Create `internal/ditaot/build_test.go`:

```go
package ditaot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTryAcquireRelease(t *testing.T) {
	b := &Builder{}
	if !b.TryAcquire() {
		t.Fatal("first TryAcquire should succeed")
	}
	if b.TryAcquire() {
		t.Fatal("second TryAcquire should fail while acquired")
	}
	b.Release()
	if !b.TryAcquire() {
		t.Fatal("TryAcquire should succeed after Release")
	}
	b.Release()
}

func TestRunSuccess(t *testing.T) {
	b := &Builder{}
	result, err := b.Run(context.Background(), "echo", "hello", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Elapsed <= 0 {
		t.Error("expected positive elapsed time")
	}
}

func TestRunFailure(t *testing.T) {
	b := &Builder{}
	result, err := b.Run(context.Background(), "false", "input.mditamap", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
}

func TestRunCapturesOutput(t *testing.T) {
	b := &Builder{}
	result, err := b.Run(context.Background(), "echo", "test-output", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output == "" {
		t.Error("expected captured output")
	}
}

func TestIntegrationBuild(t *testing.T) {
	ditaPath := os.Getenv("DITA_OT_PATH")
	if ditaPath == "" {
		t.Skip("DITA_OT_PATH not set, skipping integration test")
	}

	ditaBin := filepath.Join(ditaPath, "bin", "dita")
	tmp := t.TempDir()

	mapContent := "# Test Map\n\n- [Topic](topic.md)\n"
	topicContent := "---\n$schema: \"urn:oasis:names:tc:mdita:rng:topic.rng\"\n---\n\n# Test Topic\n\nHello world.\n"

	if err := os.WriteFile(filepath.Join(tmp, "test.mditamap"), []byte(mapContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "topic.md"), []byte(topicContent), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmp, "out")
	b := &Builder{}
	result, err := b.Run(context.Background(), ditaBin, filepath.Join(tmp, "test.mditamap"), outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("build failed, output:\n%s", result.Output)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("cannot read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected output files in out dir")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ditaot/ -run "TestTryAcquire|TestRun" -v`
Expected: FAIL — `Builder` type does not exist.

- [ ] **Step 3: Implement Builder**

Create `internal/ditaot/build.go`:

```go
package ditaot

import (
	"bytes"
	"context"
	"os/exec"
	"sync"
	"time"
)

type Builder struct {
	mu       sync.Mutex
	building bool
}

type BuildResult struct {
	Success bool
	Output  string
	Elapsed time.Duration
}

func (b *Builder) TryAcquire() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.building {
		return false
	}
	b.building = true
	return true
}

func (b *Builder) Release() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.building = false
}

func (b *Builder) Run(ctx context.Context, ditaPath, mapPath, outputDir string) (*BuildResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, ditaPath, "--input="+mapPath, "--format=xhtml", "--output="+outputDir)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	elapsed := time.Since(start)

	result := &BuildResult{
		Output:  buf.String(),
		Elapsed: elapsed,
	}

	if err != nil {
		result.Success = false
		return result, nil
	}
	result.Success = true
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ditaot/ -run "TestTryAcquire|TestRun" -v`
Expected: PASS (integration test skipped without `DITA_OT_PATH`)

- [ ] **Step 5: Commit**

```bash
git add internal/ditaot/build.go internal/ditaot/build_test.go
git commit -m "feat: add ditaot.Builder with TryAcquire/Release/Run"
```

---

### Task 4: Code Action — "Build XHTML with DITA OT"

**Files:**
- Modify: `internal/codeaction/codeaction.go`
- Modify: `internal/codeaction/codeaction_test.go`

- [ ] **Step 1: Write failing test for build action on map documents**

Add to `internal/codeaction/codeaction_test.go`:

```go
func TestBuildXHTMLActionOnMap(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Topic](topic.md)\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)

	actions := GetActions(mapDoc, document.Rng(0, 0, 3, 0), f)
	found := false
	for _, a := range actions {
		if a.Title == "Build XHTML with DITA OT" {
			found = true
			if a.Command == nil {
				t.Fatal("expected command")
			}
			if a.Command.Command != "mdita-lsp.ditaOtBuild" {
				t.Errorf("command = %q, want %q", a.Command.Command, "mdita-lsp.ditaOtBuild")
			}
			if len(a.Command.Arguments) != 1 || a.Command.Arguments[0] != mapDoc.URI {
				t.Errorf("arguments = %v, want [%s]", a.Command.Arguments, mapDoc.URI)
			}
		}
	}
	if !found {
		t.Error("missing 'Build XHTML with DITA OT' action for map document")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/codeaction/ -run TestBuildXHTMLActionOnMap -v`
Expected: FAIL — no such action returned.

- [ ] **Step 3: Write failing test — action not offered for topic documents**

Add to `internal/codeaction/codeaction_test.go`:

```go
func TestBuildXHTMLActionNotOnTopic(t *testing.T) {
	doc := document.New("file:///project/doc.md", 1,
		"# Title\n\nContent.\n")
	cfg := config.Default()
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(doc)

	actions := GetActions(doc, document.Rng(0, 0, 3, 0), f)
	for _, a := range actions {
		if a.Title == "Build XHTML with DITA OT" {
			t.Error("should not offer build action for topic documents")
		}
	}
}
```

- [ ] **Step 4: Write failing test — action not offered when disabled**

Add to `internal/codeaction/codeaction_test.go`:

```go
func TestBuildXHTMLActionDisabled(t *testing.T) {
	mapDoc := document.New("file:///project/map.mditamap", 1,
		"# My Map\n\n- [Topic](topic.md)\n")
	cfg := config.Default()
	cfg.Build.DitaOT.Enable = boolPtr(false)
	f := workspace.NewFolder("file:///project", cfg)
	f.AddDoc(mapDoc)

	actions := GetActions(mapDoc, document.Rng(0, 0, 3, 0), f)
	for _, a := range actions {
		if a.Title == "Build XHTML with DITA OT" {
			t.Error("should not offer build action when disabled")
		}
	}
}
```

- [ ] **Step 5: Add `boolPtr` helper to test file**

Add to the top of `internal/codeaction/codeaction_test.go`:

```go
func boolPtr(v bool) *bool { return &v }
```

- [ ] **Step 6: Implement the code action**

In `internal/codeaction/codeaction.go`, add after the `addToMapActions` call in `GetActions()`:

```go
actions = append(actions, buildDitaOTActions(doc, folder)...)
```

Add the new function:

```go
func buildDitaOTActions(doc *document.Document, folder *workspace.Folder) []CodeAction {
	if doc.Kind != document.Map {
		return nil
	}
	if !config.BoolVal(folder.Config.Build.DitaOT.Enable) {
		return nil
	}
	return []CodeAction{{
		Title:  "Build XHTML with DITA OT",
		Kind:   "source",
		DocURI: doc.URI,
		Command: &Command{
			Title:     "Build XHTML",
			Command:   "mdita-lsp.ditaOtBuild",
			Arguments: []string{doc.URI},
		},
	}}
}
```

- [ ] **Step 7: Run all three new tests**

Run: `go test ./internal/codeaction/ -run "TestBuildXHTML" -v`
Expected: PASS

- [ ] **Step 8: Run full codeaction test suite**

Run: `go test ./internal/codeaction/ -v`
Expected: All tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/codeaction/codeaction.go internal/codeaction/codeaction_test.go
git commit -m "feat: add 'Build XHTML with DITA OT' code action for map files"
```

---

### Task 5: LSP Server Integration

**Files:**
- Modify: `internal/lsp/server.go`

- [ ] **Step 1: Add import for ditaot package**

In `internal/lsp/server.go`, add to the import block:

```go
"github.com/aireilly/mdita-lsp/internal/ditaot"
```

- [ ] **Step 2: Add ditaBuilder field to Server struct**

Change the `Server` struct to:

```go
type Server struct {
	workspace    *workspace.Workspace
	graph        *symbols.Graph
	notify       func(method string, params any)
	diagBounce   *debouncer
	version      string
	ditaBuilder  *ditaot.Builder
}
```

- [ ] **Step 3: Initialize ditaBuilder in NewServer**

In `NewServer()`, add `ditaBuilder: &ditaot.Builder{}` to the return:

```go
func NewServer() *Server {
	return &Server{
		workspace:    workspace.New(),
		graph:        symbols.NewGraph(),
		notify:       func(string, any) {},
		diagBounce:   newDebouncer(200 * time.Millisecond),
		version:      "dev",
		ditaBuilder:  &ditaot.Builder{},
	}
}
```

- [ ] **Step 4: Register the command**

In `handleInitialize()`, add `"mdita-lsp.ditaOtBuild"` to the `Commands` slice:

```go
ExecuteCommandProvider: &ExecuteCommandOptions{
	Commands: []string{
		"mdita-lsp.createFile",
		"mdita-lsp.findReferences",
		"mdita-lsp.addToMap",
		"mdita-lsp.ditaOtBuild",
	},
},
```

- [ ] **Step 5: Add dispatch case**

In `handleExecuteCommand()`, add the new case:

```go
switch params.Command {
case "mdita-lsp.createFile":
	return s.executeCreateFile(params.Arguments)
case "mdita-lsp.addToMap":
	return s.executeAddToMap(params.Arguments)
case "mdita-lsp.ditaOtBuild":
	return s.executeDitaOtBuild(params.Arguments)
}
```

- [ ] **Step 6: Implement executeDitaOtBuild**

Add after `executeAddToMap`:

```go
func (s *Server) executeDitaOtBuild(args []string) (any, error) {
	if len(args) < 1 {
		return nil, nil
	}
	mapURI := args[0]

	folder := s.workspace.FolderForURI(mapURI)
	if folder == nil {
		s.notify("window/showMessage", ShowMessageParams{
			Type:    LogError,
			Message: "DITA OT build: no workspace folder found",
		})
		return nil, nil
	}

	ditaPath, err := ditaot.ResolveDitaPath(folder.Config.Build.DitaOT.DitaPath)
	if err != nil {
		s.notify("window/showMessage", ShowMessageParams{
			Type:    LogError,
			Message: err.Error(),
		})
		return nil, nil
	}

	mapPath, err := paths.URIToPath(mapURI)
	if err != nil {
		return nil, nil
	}

	outputDir := folder.Config.Build.DitaOT.OutputDir
	if outputDir == "" {
		outputDir = "out"
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(folder.RootPath(), outputDir)
	}

	if !s.ditaBuilder.TryAcquire() {
		s.notify("window/showMessage", ShowMessageParams{
			Type:    LogWarning,
			Message: "DITA OT build already in progress",
		})
		return nil, nil
	}

	mapName := filepath.Base(mapPath)
	s.logMessage(LogInfo, "DITA OT build started for "+mapName)

	go func() {
		defer s.ditaBuilder.Release()

		result, err := s.ditaBuilder.Run(context.Background(), ditaPath, mapPath, outputDir)
		if err != nil {
			s.notify("window/showMessage", ShowMessageParams{
				Type:    LogError,
				Message: "DITA OT build error: " + err.Error(),
			})
			return
		}

		if result.Output != "" {
			s.logMessage(LogInfo, result.Output)
		}

		if result.Success {
			s.notify("window/showMessage", ShowMessageParams{
				Type:    LogInfo,
				Message: fmt.Sprintf("DITA OT build complete (%s). Output: %s", result.Elapsed.Round(time.Millisecond), outputDir),
			})
		} else {
			s.notify("window/showMessage", ShowMessageParams{
				Type:    LogError,
				Message: "DITA OT build failed. See output log for details.",
			})
		}
	}()

	return nil, nil
}
```

- [ ] **Step 7: Verify build**

Run: `go build ./...`
Expected: Build succeeds with no errors.

- [ ] **Step 8: Run full test suite**

Run: `make test`
Expected: All tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/lsp/server.go
git commit -m "feat: wire ditaOtBuild execute command into LSP server"
```

---

### Task 6: CI — DITA OT Integration Test Job

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Add integration test job**

Replace the contents of `.github/workflows/ci.yml` with:

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
          go-version-file: go.mod
      - run: go vet ./...
      - run: make test
      - run: make build
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
  integration:
    runs-on: ubuntu-latest
    env:
      DITA_OT_VERSION: "4.2.3"
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/setup-java@v4
        with:
          distribution: temurin
          java-version: "17"
      - name: Cache DITA OT
        id: cache-dita-ot
        uses: actions/cache@v4
        with:
          path: /opt/dita-ot
          key: dita-ot-${{ env.DITA_OT_VERSION }}
      - name: Install DITA OT
        if: steps.cache-dita-ot.outputs.cache-hit != 'true'
        run: |
          curl -sL "https://github.com/dita-ot/dita-ot/releases/download/${DITA_OT_VERSION}/dita-ot-${DITA_OT_VERSION}.zip" -o dita-ot.zip
          unzip -q dita-ot.zip -d /opt
          mv /opt/dita-ot-${DITA_OT_VERSION} /opt/dita-ot
      - name: Run integration tests
        env:
          DITA_OT_PATH: /opt/dita-ot
        run: go test ./internal/ditaot/ -run TestIntegration -v -timeout 300s
```

- [ ] **Step 2: Verify YAML is valid**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"`
Expected: No error.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add DITA OT integration test job with caching"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `make test`
Expected: All tests PASS (integration tests skipped locally)

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: No lint errors

- [ ] **Step 3: Build binary**

Run: `make build`
Expected: Binary builds successfully

- [ ] **Step 4: Verify test count increased**

Run: `go test ./... 2>&1 | grep -E "^ok" | awk '{sum += $NF} END {print sum}'`
Expected: Test count is higher than the previous 235.

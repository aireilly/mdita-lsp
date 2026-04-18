# DITA OT Build Integration — Design Spec

## Overview

Add a DITA OT integration to mdita-lsp that shells out to the `dita` CLI to build XHTML output from MDITA files. Exposed as an LSP code action on `.mditamap` files that fires an execute command.

## Requirements

- **Trigger:** Code action ("Build XHTML with DITA OT") on `.mditamap` files fires `mdita-lsp.ditaOtBuild` execute command
- **DITA OT location:** Config field `build.dita_ot.dita_path` takes precedence; falls back to `dita` on `$PATH`
- **Output directory:** Configurable via `build.dita_ot.output_dir`, defaults to `out/` relative to workspace root
- **Feedback:** `window/showMessage` for build outcome, `window/logMessage` for full DITA OT stdout/stderr
- **Execution:** Async via goroutine — command returns immediately, results pushed when done
- **Concurrency:** One build at a time; second request rejected with "Build already in progress" message

## Configuration

New `build` section in `.mdita-lsp.yaml`:

```yaml
build:
  dita_ot:
    enable: true
    dita_path: ""       # Path to dita binary. Empty = search $PATH
    output_dir: "out"   # Relative to workspace root
```

Added to `internal/config/config.go`:

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

- `Config` struct gets a new `Build BuildConfig` field
- Default: `enable: true`, `output_dir: "out"`, `dita_path: ""`
- Merging works via the existing 3-level merge (defaults < user < workspace)

## New Package: `internal/ditaot/`

### `resolve.go` — Binary discovery

```go
func ResolveDitaPath(configured string) (string, error)
```

- If `configured` is non-empty, checks it exists and is executable
- Otherwise, calls `exec.LookPath("dita")`
- Returns clear error if not found: "dita binary not found: install DITA OT and add to $PATH, or set build.dita_ot.dita_path in config"

### `build.go` — Build invocation

```go
type Builder struct {
    mu       sync.Mutex
    building bool
}

type BuildResult struct {
    Success bool
    Output  string          // combined stdout+stderr
    Elapsed time.Duration
}

func (b *Builder) TryAcquire() bool
func (b *Builder) Release()
func (b *Builder) Run(ctx context.Context, ditaPath, mapPath, outputDir string) (*BuildResult, error)
```

- `TryAcquire` checks and sets `b.building` under mutex — returns false if already running
- `Release` clears `b.building` under mutex
- `Run` executes: `dita --input=<mapPath> --format=xhtml --output=<outputDir>`
- Captures combined stdout/stderr
- Returns `BuildResult` with success, output, and elapsed time

## Code Action

In `internal/codeaction/codeaction.go`:

- New code action "Build XHTML with DITA OT" returned by `GetActions()` when:
  - Document `Kind == document.Map`
  - `config.BoolVal(cfg.Build.DitaOT.Enable)` is true
- Carries a `Command` pointing to `"mdita-lsp.ditaOtBuild"` with argument `[]string{docURI}`

## LSP Server Integration

In `internal/lsp/server.go`:

**Registration:**
- Add `"mdita-lsp.ditaOtBuild"` to `ExecuteCommandProvider.Commands` in `handleInitialize()`

**Server state:**
- Add `ditaBuilder *ditaot.Builder` field to `Server` struct, initialized in constructor

**Command dispatch:**
- New case in `handleExecuteCommand()`: `"mdita-lsp.ditaOtBuild"` → `s.executeDitaOtBuild(params.Arguments)`

**`executeDitaOtBuild(args []string)` method:**
1. Extracts `mapURI` from args, finds folder via `s.workspace.FolderForURI(mapURI)`
2. Resolves `dita` binary via `ditaot.ResolveDitaPath(folder.Config.Build.DitaOT.DitaPath)`
3. Converts `mapURI` to filesystem path, computes output dir relative to `folder.RootPath()`
4. Sends `window/logMessage` — "DITA OT build started for \<map\>"
5. Calls `s.ditaBuilder.TryAcquire()` — if already building, sends `window/showMessage` warning "Build already in progress" and returns immediately
6. Launches goroutine:
   - Defers `s.ditaBuilder.Release()`
   - Calls `s.ditaBuilder.Run(ctx, ditaPath, mapPath, outputDir)`
   - Success: `window/showMessage` info — "Build complete (\<elapsed\>). Output: \<outputDir\>"
   - Build error: `window/showMessage` error — "Build failed" + `window/logMessage` with full output
7. Returns `nil, nil` immediately (async)

The concurrency guard (`TryAcquire`/`Release`) is split from the build logic (`Run`) so the "already building" rejection happens synchronously in the handler before the goroutine is launched.

## Testing

### Unit tests (`internal/ditaot/`)

- `ResolveDitaPath` — valid path returns it, invalid path errors, empty string with no `dita` on PATH errors
- `Builder` mutex — concurrent call returns "already building" error
- `BuildResult` — stdout/stderr captured, elapsed time populated

### Integration tests (`internal/ditaot/`)

- Gated behind `DITA_OT_PATH` env var — skipped if not set
- Creates temp dir with minimal `.mditamap` and one `.md` topic
- Runs real build, asserts XHTML output files exist
- Asserts `BuildResult.Success == true`

### Code action tests (`internal/codeaction/`)

- "Build XHTML with DITA OT" appears for map documents when config enabled
- Does not appear for non-map documents
- Does not appear when `build.dita_ot.enable` is false

### LSP-level tests (`internal/lsp/`)

- Send `workspace/executeCommand` with `mdita-lsp.ditaOtBuild`, verify accepted
- No full build — relies on `ditaot` package tests

## CI Integration

New step in GitHub Actions CI workflow:

- **DITA OT install:** Downloads known DITA OT release (4.2.x) from GitHub releases, extracts to temp dir, sets `DITA_OT_PATH` and adds `bin/` to `$PATH`
- **Caching:** `actions/cache` keyed on DITA OT version to avoid re-downloading ~200MB per run
- **Integration tests:** Runs `go test ./internal/ditaot/ -run TestIntegration -v` with `DITA_OT_PATH` set
- **Separation:** `make test` runs unit tests (integration tests skip without env var); CI runs both sequentially

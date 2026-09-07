# Base project rules — f4

> Auto-detected conventions from codebase analysis. Edit as needed.

## Naming Conventions

- Files: `snake_case.go`, named after the feature they hold
  (`action_registry.go`, `panels_frame.go`, `os_vfs_search.go`)
- Test files: the source file name plus `_test.go`, kept next to the source
- Platform files: build-tag suffixes carry the platform, not runtime branching —
  `*_windows.go`, `*_unix.go`, `*_other.go`, plus `//go:build` lines in 182 files
- Variables and functions: Go standard `camelCase` unexported, `PascalCase` exported
- Types: `PascalCase`; Far-derived structures keep the names of their C++ originals
  even when Go style would suggest otherwise
- Packages: single lowercase word (`vfs`, `wincon`, `ttyx`, `cloudfox`, `envman`)

## Module Structure

- `cmd/f4/` — the application, one flat `package main` of ~687 files. New
  application code belongs here unless it is genuinely reusable.
- `vfs/` — the filesystem abstraction all panels and plugins go through
- `plugins/<name>/` — one package per plugin
- `internal/` — platform helpers that must not be imported outside the module
- `sdk/`, `plugring/`, `luaplug/` — plugin API, registry, Lua engine
- `piecetable/`, `textlayout/`, `sheet/`, `colorer/`, `fusefs/`, `vtvibe/` —
  self-contained subsystems consumed by `cmd/f4`
- `tools/` — developer tooling, not shipped in the binary
- UI and input live outside this repository, in the `vtui` and `vtinput` libraries

## Error Handling

- Wrap with context: `fmt.Errorf("doing X: %w", err)` (~395 call sites)
- Compare with `errors.Is` / `errors.As`, never string matching (~159 call sites)
- Sentinel errors via `errors.New` at package level (~141)
- Error strings frequently become dialog text the user reads, so they are written for
  a human — this is why `ST1005` is disabled in `.golangci.yml`
- Startup and fatal paths report to stderr with the `f4: ` prefix:
  `fmt.Fprintf(os.Stderr, "f4: cannot create %q: %v\n", path, err)`
- `errcheck` and `gosec` run in CI: deliberately ignored errors are written as
  `_ = f()`, not left bare

## Control Flow

- Prefer flat, readable control flow over deeply nested conditionals. Use guard
  clauses, early `return`/`continue`, small named helper methods, or explicit
  classification logic when they make the code easier to follow. Handle edge cases
  and irrelevant branches early so the main path stays visible.

## Logging

- No logging framework. Diagnostics go through the `VTUI_DEBUG` environment
  variable, which `vtui` writes to `<profile>/logs/debug.log`
  (`cmd/f4/debug_log.go`); `--debug` / `--log=1` set it up
- User-facing failures go to stderr with the `f4: ` prefix
- Do not add a logging dependency; do not log to stdout — it is the rendered UI

## Testing

- Standard library `testing` only, no testify (552 files import `"testing"`)
- Helpers call `t.Helper()` as the first statement (~398 call sites)
- Table-driven tests with `for _, tt := range` in ~94 files
- Tests are the review mechanism for this AI-only codebase: a change lands with a
  test, a bug fix lands with a regression test
- Tests are mostly sequential — `t.Parallel()` is the exception, not the default,
  because much of `cmd/f4` shares global terminal state
- Use the system Go build cache (`go env GOCACHE`); never redirect `GOCACHE`

## Build & Portability

- `CGO_ENABLED=0` is non-negotiable: the single static binary depends on it.
  FFI goes through `purego` / `ffibridge`
- Every change must keep the full cross-platform matrix building, including the
  exotic targets (mips, riscv64, loong64, ppc64, Illumos, Solaris)
- New dependencies are weighed against the ~110 MB binary
- Subsystem changes update the matching document in `docs/`

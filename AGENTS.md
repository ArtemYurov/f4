# AGENTS.md

> Structural map of the repository for AI agents and new contributors. Keep it
> factual — describe only what exists. Update it when the structure changes.

## Project Overview

`f4` is a cross-platform TUI file manager written entirely in Go that reproduces the
features, UX and internal structures of `far2l` / Far Manager. It ships as a single
static binary and runs either in a terminal or as a standalone graphical window.

## Tech Stack

- **Programming language:** Go 1.26.6, `CGO_ENABLED=0`
- **Framework:** none — custom TUI; UI and input come from the external `vtui` and
  `vtinput` libraries
- **Database:** none for the application; `plugins/sqlite` browses user SQLite files
- **Lint:** golangci-lint v2 (staticcheck, errcheck, ineffassign, unused, gosec)

## Project Structure

```
cmd/f4/          # the application: 687 files in one flat package main
                 # panels, dialogs, editor, viewer, actions, macros, terminal
vfs/             # filesystem abstraction used by every panel and plugin
  hostfs/        #   host filesystem access
  hostmode/      #   host console mode
  hostpath/      #   path translation
plugins/         # one package per plugin: archive, cloudfox, netfox, mediainfo,
                 # envman, ios, android, sqlite, visren, id3editor, chroma
                 # dummy_internal / dummy_rpc / dummy_lua are transport fixtures
sdk/             # plugin API: f4plugin, f4rpc, lua, extui
plugring/        # plugin registry
luaplug/         # Lua plugin engine
piecetable/      # piece table backing the editor
textlayout/      # text layout and wrapping
sheet/           # spreadsheet mode
colorer/         # colorer4go syntax highlighting integration
fusefs/          # FUSE mounting
vtvibe/          # vtvibe session/provider layer
internal/        # module-private platform helpers
  wincon/        #   Windows console
  ttyx/          #   tty extensions
  netproxy/      #   network proxy
  testutil/      #   test scaffolding shared across packages; _test.go use only
  paneltest/     #   the same, for helpers that need a panels frame
  hideconsole/   #   console hiding on Windows
tools/           # developer tooling, incl. the ttytest terminal harness
docs/            # 48 subsystem documents — read the relevant one before editing
packaging/       # distribution packaging
artifacts/       # build artifacts
.ai-factory/     # AI Factory context: config, description, rules, plans
```

## Key Entry Points

| File | Purpose |
| --- | --- |
| `cmd/f4/main.go` | Program entry point, CLI flags, startup mode selection |
| `cmd/f4/api.go` | Internal API surface used across the application package |
| `cmd/f4/actions.go`, `cmd/f4/action_registry.go` | Action definitions and dispatch |
| `embedded.go` | Assets embedded into the binary |
| `go.mod` | Module `github.com/unxed/f4`, Go 1.26.6, dependency set |
| `f4.example.ini` | Reference configuration file |
| `highlight.ini` | Syntax highlighting configuration |
| `.golangci.yml`, `.golangci-strict.yml` | Lint configuration |
| `.github/workflows/build.yml` | CI: cross-platform build matrix, tests, releases |

## Documentation

| Document | Path | Description |
| --- | --- | --- |
| README | `README.md` | Project overview, downloads, backends, philosophy |
| Subsystem docs | `docs/*.md` | 48 documents: VFS, PLUGINS, MACROS, KEYMAP, TERMINAL, CONPTY, WINCON, UX_GUIDELINES and others |
| Issue reviews | `docs/ISSUES/` | Per-issue solution reviews |
| Spreadsheet | `SPREADSHEET.md` | Spreadsheet mode specification |

## AI Context Files

| File | Purpose |
| --- | --- |
| `AGENTS.md` | This structural map of the repository |
| `.ai-factory/DESCRIPTION.md` | Project specification: stack, features, architecture notes |
| `.ai-factory/ARCHITECTURE.md` | Architecture pattern, boundaries and dependency rules |
| `.ai-factory/rules/base.md` | Detected code conventions: naming, errors, logging, tests |
| `.ai-factory/config.yaml` | AI Factory configuration: paths, language, git workflow |
| `.mcp.json` | MCP servers for this project: CodeGraph code-graph index |

## Agent Rules

### Code graph (CodeGraph MCP)

- The MCP server is wired in `.mcp.json` and runs through `npx`, so no global
  install is needed. The index lives in `.codegraph/` and is git-ignored.
- **A fresh clone has no index.** The server does not build one on its own — if a
  tool reports `No .codegraph/`, run `npx -y @colbymchenry/codegraph@1.6.0 init`
  once in the repository root. A new index is picked up live, no restart.
- There is no `codegraph` binary on PATH. Every invocation takes the form
  `npx -y @colbymchenry/codegraph@1.6.0 <command>`; the commands below are the
  `<command>` part.
- Use it instead of `grep` for symbol questions — `cmd/f4` is one flat
  `package main` of ~109k lines, where grep is both slow and imprecise:
  - `callers <symbol>` — who calls it
  - `callees <symbol>` — what it calls
  - `impact <symbol>` — what a change touches
  - `explore <query>` — relevant symbols with source and call paths
  - `node <symbol|file>` — one symbol's source plus its caller trail
- The index auto-syncs on file changes; after a large rebase run `sync`.
- Grep stays the right tool for text that is not a symbol: comments, error
  strings, build tags, config keys.

### Where new code goes

- Put a new file in the package that owns its subject. If no package owns it,
  create one — do not widen a neighbouring package because it is close enough,
  and never park the file in `internal/app`.
- `internal/app` is the composition root: it wires packages together and does not
  implement features. Code that lands there for lack of a better place is code
  whose owner was not decided.
- Do not add to `cmd/f4`. It holds `main.go`, the wiring tests, the module-wide
  auditors and the Windows `.syso` files, and nothing else.
- Inside a package, name files `<topic>.go` and `<topic>_<aspect>.go`, where the
  prefix is the topic inside the package, not the package name — `panel/frame.go`,
  never `panel/panel_frame.go`. Platform suffixes go on the end:
  `frame_dragdrop_windows.go`.
- Need something from a higher layer? Declare an interface in your package and
  let the caller supply the implementation. Never import upward, and never reach
  across a boundary through a shared mutable global.
- Logic belongs here but the type belongs elsewhere? Write a function taking the
  type, not a method — a method would drag the whole file into the type's
  package.
- `cmd/f4/architecture_test.go` enforces the layer rules. If a change needs an
  exemption there, the architecture document is what changes first, not the test.
- The full rules, with the reasoning, are in `.ai-factory/ARCHITECTURE.md`.

### Go build cache

- Use the system Go build cache reported by `go env GOCACHE` for all Go builds and tests.
- Do not redirect `GOCACHE` to `/tmp`, the repository, or another task-local directory unless the user explicitly asks for it.
- If the system cache is unavailable or not writable, report that constraint instead of silently creating a substitute cache.

### Shell commands

- Run shell commands one step at a time instead of chaining them, so a failing step is visible.
  - Wrong: `git checkout main && git pull`
  - Right: first `git checkout main`, then `git pull origin main`

### Portability

- `CGO_ENABLED=0` must stay: the single static binary depends on it. FFI goes through `purego` / `ffibridge`.
- Platform differences belong in build-tag files (`*_windows.go`, `*_unix.go`), not runtime branching.
- Changes must keep the full CI matrix building, exotic targets included.

### Tests

- This is an AI-only codebase; the test suite is the review mechanism. New behaviour lands with a test, a bug fix lands with a regression test.

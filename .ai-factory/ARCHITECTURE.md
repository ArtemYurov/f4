# Architecture: Modular Monolith

## Overview

`f4` is a single static binary that ships a complete file manager: two panels,
editor, viewer, terminal integration, plugin hosts and a virtual filesystem layer.
The architectural pattern is a **Modular Monolith**: one deployment unit, one
process, but hard module boundaries drawn along Go package lines. A module is a Go
package; its public API is its exported identifiers; nobody reaches into another
module's internals.

The current tree already follows this pattern above `cmd/f4`: `vfs/`, `sdk/`,
`plugins/*`, `piecetable/`, `textlayout/`, `sheet/`, `fusefs/`, `vtvibe/`,
`luaplug/` and `internal/*` are real modules with an acyclic dependency graph and
no upward edges. Two things are missing. The application itself — `cmd/f4` — holds
345 non-test files and ~109k lines in one flat `package main`, the one place where
the compiler enforces nothing. And the repository root mixes the public contract
(`sdk/`, `vfs/`) with module-private subsystems and loose data directories, so
nothing about the layout says which is which.

This document defines the target structure that closes both gaps: **the root holds
entry points and the plugin contract with its implementations; the application core
lives under `internal/`.** Once the layout states the rule, the compiler enforces it
and agents reproduce it without being told again.

## Decision Rationale

- **Project type:** cross-platform TUI file manager, single static binary.
- **Tech stack:** Go 1.26.6, `CGO_ENABLED=0`, no framework; UI and input come from
  the external `vtui` / `vtinput` libraries.
- **Size:** ~700 Go files outside plugins, 560 `_test.go` files, AI-generated
  codebase where the test suite is the review mechanism.
- **Key factors:**
  1. A file manager is one interactive process with shared terminal state. There is
     no network traffic between subsystems that would pay for service boundaries —
     microservices and layered ports/adapters both add ceremony with no return.
  2. The subsystems are genuinely independent (piece table, spreadsheet, FUSE,
     Lua engine, VFS backends). Packages already express that; the pattern only has
     to name it and extend it to `cmd/f4`.
  3. `CGO_ENABLED=0` and the cross-platform matrix mean structure has to survive
     build tags. Package-per-module with `*_windows.go` / `*_unix.go` suffixes does;
     folder-per-technical-layer does not.
  4. Third-party plugins compile against `sdk/` and `vfs/`. Those are a public
     contract and must stay importable from outside the module — they cannot move
     under `internal/`.

## Folder Structure

Target layout. Entries marked **(move)** exist today and relocate; **(extract)**
means the package is carved out of today's flat `cmd/f4`.

```text
f4/
├── cmd/
│   └── f4/
│       ├── main.go                # Composition Root: flags, startup mode, wiring
│       └── *_test.go              # only tests for the wiring itself
│
├── sdk/                           # ── PUBLIC API (third-party plugin authors) ──
│   ├── f4plugin/                  #   in-process Go plugin contract
│   ├── f4rpc/                     #   RPC transport mux
│   ├── extui/                     #   external UI model
│   └── lua/                       #   Lua-facing surface
│
├── vfs/                           # ── PUBLIC: filesystem abstraction ──
│   ├── hostfs/  hostmode/  hostpath/
│   └── *_unix.go  *_windows.go    #   per-OS files, not runtime branching
│
├── plugins/                       # ── SHIPPED PLUGINS: implementations of sdk/ ──
│   ├── archive/ cloudfox/ netfox/ mediainfo/ envman/
│   ├── ios/ android/ sqlite/ visren/ id3editor/ chroma/
│   └── dummy_internal/ dummy_rpc/ dummy_lua/   # transport fixtures
│
├── internal/                      # ── THE APPLICATION CORE IS MODULE-PRIVATE ──
│   │
│   │  # application core, extracted from the flat package main
│   ├── app/          (extract)    # bootstrap, event loop, global app state, actions
│   ├── panel/        (extract)    # file panels, sorting, navigation, quick view, info
│   ├── editor/       (extract)    # F4 editor on top of internal/piecetable
│   ├── viewer/       (extract)    # F3 viewer, hex, disasm
│   ├── dialog/       (extract)    # modal dialogs, command palette, menus, help
│   ├── cmdline/      (extract)    # command line, prefixes, history, apply_command_*
│   ├── macro/        (extract)    # macro engine and Lua macro API
│   ├── plughost/     (extract)    # plugin host: in-process / RPC / Lua / WASM transports
│   ├── term/         (extract)    # pty, console host, ttyx, ANSI parser, kitty/sixel
│   ├── gui/          (extract)    # GUI backends, fonts, window position and icon
│   ├── media/        (extract)    # image, audio, video decode and preview
│   ├── sysinfo/      (extract)    # cpu / mem / fs / gpu info, drives
│   ├── update/       (extract)    # self-update, elevation, helper args
│   ├── settings/     (extract)    # ini, config, lang packs, colors, keymap, hotkeys
│   ├── fileops/      (extract)    # copy/move/delete, background jobs, clipboard
│   │
│   │  # self-contained subsystems, module-private (moved off the root)
│   ├── piecetable/   (move)       # piece table backing the editor
│   ├── textlayout/   (move)       # text layout and wrapping
│   ├── sheet/        (move)       # spreadsheet mode
│   ├── fusefs/       (move)       # FUSE mounting
│   ├── vtvibe/       (move)       # vtvibe session/provider layer
│   ├── luaplug/      (move)       # Lua plugin engine
│   │
│   │  # platform helpers, already here
│   └── wincon/  ttyx/  netproxy/  hideconsole/
│
├── embedded.go                    # root package: embeds README.md and the .hrd
│                                  # scheme. Must stay in the root — //go:embed
│                                  # cannot reach above its own directory.
│
├── assets/           (new)        # screenshot.png, colour schemes, plugin ring data
│   ├── colorer/      (move)       #   colorer4go .hrd schemes (data, not Go)
│   └── plugring/     (move)       #   plugin ring index.yaml + Lua (data, not Go)
│
├── scripts/          (new)        # *.sh moved out of the repository root
├── docs/                          # subsystem documents; SPREADSHEET.md and
│   └── ISSUES/                    #   ISSUE_*.md land here, not in the root
├── tools/                         # developer tooling incl. the ttytest harness
├── packaging/                     # distribution packaging
├── .github/workflows/             # CI build matrix, nightly and tagged releases
│
└── README.md  LICENSE  go.mod  go.sum  f4.example.ini  highlight.ini
                                   # reference configs stay next to the README
```

The rule the root expresses: **entry points and the plugin contract with its
implementations live at the top; the application core lives under `internal/`.**

- `sdk/` — third-party plugins compile against it. Under `internal/` Go refuses the
  import and every external plugin stops building.
- `vfs/` — same contract: a plugin implementing a filesystem backend needs the
  types. Twelve in-tree plugins plus `fusefs` and `vtvibe` already depend on it.
- `plugins/` — the shipped implementations of that contract, and the reference an
  external plugin author reads before writing their own. Extensibility through four
  transports is a defining property of f4, so the tree says so at the top level.
  They are leaves of the dependency graph: hiding them buys no enforcement, and
  297 files of extensions sitting beside `panel/` and `editor/` would blur the line
  between the core and what plugs into it.
- `cmd/f4` — the entry point; it is `package main` and nothing can import it anyway.
- `embedded.go` — the root package that bridges root-level files into the binary.
  `//go:embed` cannot reach above its own directory, and `README.md` has to stay in
  the root to render on GitHub, so this one file is pinned there by the toolchain.

Everything else is module-private, so the compiler answers "may I import this?"
before a reviewer has to.

**Embedded resources travel with their package.** The same `//go:embed` rule means
`cmd/f4/styles/`, `cmd/f4/lang/`, `cmd/f4/help/` and `cmd/f4/assets/icon/` move
together with the code that embeds them — into `internal/settings`,
`internal/dialog` and `internal/gui`. Data that is embedded from the root package
(`assets/colorer/…/radiola.hrd`) can live anywhere below the root, because the root
package's embed scope is the whole tree.

## Dependency Rules

Dependencies point from the application inward to the subsystems. `cmd/f4/main.go`
is the only place where everything is assembled. Layers, bottom up:

**Layer 0 — kernel, no intra-module dependencies:** `vfs`, `sdk`,
`internal/piecetable`, `internal/sheet`, `internal/wincon`, `internal/ttyx`,
`internal/netproxy`, `internal/hideconsole`, `internal/settings`,
`internal/sysinfo`.

**Layer 1 — subsystems over the kernel:** `internal/textlayout` →
`internal/piecetable`; `internal/fusefs` → `vfs`; `internal/vtvibe` → `vfs`;
`internal/luaplug`; `internal/term`, `internal/gui`, `internal/media`,
`internal/fileops`.

**Layer 2 — plugins and hosts:** `plugins/*` → `vfs`, `sdk`, `internal/*`;
`internal/plughost` → `sdk`, `internal/luaplug`, `vfs`.

**Layer 3 — interactive subsystems:** `internal/panel`, `internal/editor`,
`internal/viewer`, `internal/dialog`, `internal/cmdline`, `internal/macro`,
`internal/update`.

**Layer 4 — application:** `internal/app`, then `cmd/f4`.

Rules:

- ✅ `cmd/f4` → any package (Composition Root).
- ✅ `internal/app` → every layer 0-3 package; it owns the event loop and the
  global application state that the interactive subsystems share.
- ✅ `plugins/*` → `vfs`, `sdk`, `internal/*` helper packages. Nothing imports a
  plugin back: they are leaves, wired in through `internal/plughost`.
- ✅ Any module → `internal/settings`, `internal/sysinfo`, `vfs`.
- ✅ Higher-layer modules talk to lower ones by calling exported constructors and
  methods; lower ones call back through interfaces they define themselves.
- ❌ `vfs` / `sdk` → any `internal/*` package. They are the public contract: an
  `internal/` import makes them uncompilable for third-party plugins, and the
  breakage surfaces only in someone else's build.
- ❌ `internal/piecetable` / `internal/sheet` → higher-layer packages. Kernel
  subsystems stay leaf nodes; that is what makes them testable in isolation.
- ❌ Any package → `cmd/f4`. It is `package main`; nothing can import it, and
  nothing should want to.
- ❌ Layer 0-3 modules → `internal/app`. Shared state flows down through
  constructor arguments, never up through an import.
- ❌ Import cycles between subsystem packages. If two need each other, the shared
  type belongs in a lower layer, or one of them defines an interface the other
  satisfies.
- ❌ Runtime `if runtime.GOOS == …` branching for platform differences. Use
  build-tag files (`*_windows.go`, `*_unix.go`, `*_other.go`).
- ❌ New cgo. FFI goes through `purego` / `ffibridge`.

## Layer / Module Communication

- **Composition Root.** `cmd/f4/main.go` parses flags, picks the startup mode
  (terminal, GUI backend, `--update`, plugin scaffolding) and constructs the
  application. Modules do not construct their own dependencies; a `New(...)`
  constructor takes what it needs, so a half-initialised struct is not reachable.
- **Interactive subsystems ↔ app.** `internal/app` owns the event loop and
  dispatches input to the focused subsystem. Panels, editor, viewer and dialogs
  receive the state they need at construction; they do not import `internal/app`
  to reach back for it.
- **Everything filesystem-shaped goes through `vfs`.** Panels, plugins, the viewer
  and FUSE mounting all address files through the `vfs` contract, which is why a
  panel can browse an archive, an SFTP host or an S3 bucket without knowing which.
- **Plugins are hosted, not linked.** `internal/plughost` owns all four transports
  (in-process Go, RPC, Lua via `internal/luaplug`, WASM via `wazero`). The rest of the
  application talks to plugins through the host, never to a transport directly.
  The contract a plugin compiles against is `sdk/`.
- **Rendering is external.** UI and input primitives live in the `vtui` and
  `vtinput` libraries outside this repository. Modules render through them; there
  is no in-tree TUI engine to organise.
- **Platform differences are file-level.** Anything touching the console, the
  filesystem or process spawning gets a per-OS file with a build tag. The exotic
  targets (mips, riscv64, loong64, ppc64, Illumos, Solaris) are part of the
  contract, not a nice-to-have.

## Key Principles

1. **Module = Go package.** One responsibility per package, public API = its
   exported identifiers. The compiler enforces the boundary — this is why breaking
   up `cmd/f4` matters more than any naming convention.

2. **Composition Root in `main.go`.** All wiring in one place. No package-level
   singletons initialised by `init()`, no post-construction mutation of a struct
   another goroutine already reads.

3. **Far heritage stays.** Structures ported from `far2l` / Far Manager keep the
   names of their C++ originals even where Go style would say otherwise. Moving a
   file into a package does not license renaming its types.

4. **Public surface is deliberate.** `sdk/` and `vfs/` are the third-party contract
   and change with care. `plugins/` sits beside them as the reference
   implementation. The application core belongs under `internal/`, where the
   compiler keeps it private.

5. **Tests move with the code.** This is an AI-only codebase where the test suite
   is the review mechanism. A file relocated to a new package takes its
   `_test.go` neighbour along in the same commit; a package extraction that drops
   coverage is not done.

6. **Portability is a boundary condition.** `CGO_ENABLED=0`, build-tag files, the
   full CI matrix green. A restructuring commit that only builds on the developer's
   own platform is a broken commit.

7. **The repository root is for entry points, not artefacts.** A newcomer should
   reach `README.md` without scrolling. Scripts go to `scripts/`, media and data to
   `assets/`, prose to `docs/`. Per-issue write-ups belong in `docs/ISSUES/` — or,
   when produced through this harness, in the research → plan → archive chain under
   `.ai-factory/`.

## Legacy vs New Code Policy

- **New features:** new code goes into the module it belongs to. If no module fits,
  create the package — do not add file number 346 to `cmd/f4`.
- **Existing code:** do not opportunistically refactor unrelated files in
  `cmd/f4` while fixing a bug. Extraction is its own task, with its own commit, so
  a behavioural change never hides inside a move diff.
- **Migration is mechanical and staged.** One subsystem per commit: `git mv` the
  files and their tests, add the package clause, export what `cmd/f4` still needs,
  fix imports, run the matrix. No rewrites inside a move commit — a reviewer must
  be able to confirm the diff is a rename.
- **Extraction order:** leaf-first. `internal/sysinfo`, `internal/update`,
  `internal/media` and `internal/settings` have the fewest inbound edges and go
  first; `internal/app` and `internal/panel` come last, once everything they
  depend on has left `cmd/f4`.
- **Interoperability:** while a subsystem is half-extracted, the extracted package
  must not import `cmd/f4` back — that is impossible for `package main` anyway,
  which is precisely what makes leaf-first ordering the only workable order.

## Code Examples

### Composition Root — construction, not global state

```go
// cmd/f4/main.go
func main() {
    flags := parseFlags()
    cfg := settings.Load(flags.ConfigPath)

    fs := vfs.NewHost()                       // layer 0
    host := plughost.New(cfg, fs)             // layer 2: owns all four transports
    term := term.New(cfg.Terminal)            // layer 1

    left := panel.New(cfg, fs, panel.Left)    // layer 3
    right := panel.New(cfg, fs, panel.Right)

    application := app.New(cfg, fs, host, term, left, right)
    if err := application.Run(context.Background()); err != nil {
        fmt.Fprintf(os.Stderr, "f4: %v\n", err)
        os.Exit(1)
    }
}
```

### Dependency direction — a lower layer defines the interface

```go
// internal/panel/panel.go — the panel says what it needs from a plugin host;
// it does not import internal/plughost, so plughost can depend on panel types
// later without producing a cycle.
type PluginColumns interface {
    ColumnsFor(ctx context.Context, path string) ([]Column, error)
}

func New(cfg *settings.Config, fs vfs.FileSystem, side Side) *Panel { … }
```

```go
// internal/app/app.go — the app is the only place that knows both sides exist.
func New(cfg *settings.Config, fs vfs.FileSystem, host *plughost.Host,
    t *term.Terminal, left, right *panel.Panel) *App { … }
```

### Platform differences stay in build-tag files

```go
// internal/sysinfo/cpu_info_linux.go
//go:build linux

package sysinfo

func readCPUInfo() (Info, error) { … }   // /proc/cpuinfo
```

```go
// internal/sysinfo/cpu_info_windows.go
//go:build windows

package sysinfo

func readCPUInfo() (Info, error) { … }   // registry
```

### Errors carry context and reach the user

```go
// Error strings frequently become dialog text, so they are written for a human.
// ST1005 is disabled in .golangci.yml for exactly this reason.
if err := fs.Copy(ctx, src, dst); err != nil {
    return fmt.Errorf("copying %q to %q: %w", src, dst, err)
}
```

## Anti-Patterns

- ❌ **Adding to the flat package.** A new feature landing as another file in
  `cmd/f4` moves the project backwards; the whole point of the target layout is
  that the compiler, not a convention, keeps subsystems apart.
- ❌ **Moving `vfs` or `sdk` under `internal/`.** It compiles here and breaks every
  third-party plugin, because Go forbids importing `internal/` from outside the
  module.
- ❌ **Rewriting inside a move commit.** Renamed identifiers and reshuffled logic
  hidden in a 300-file `git mv` diff cannot be reviewed, and this codebase reviews
  by reading diffs and running tests.
- ❌ **Package-level mutable state as a shortcut across a boundary.** A `var
  currentPanel *Panel` in one package read by another is an import cycle that the
  compiler happens not to catch.
- ❌ **Renaming Far-derived types during a move.** `PanelViewSettings` stays
  `PanelViewSettings`; navigability for people who know the Far API is a stated
  project goal.
- ❌ **Runtime OS branching instead of build tags.** It compiles unreachable code
  into every target and inflates a binary that is already ~110 MB.
- ❌ **Packages named after technical layers** (`services/`, `handlers/`,
  `models/`). There is no HTTP, no database and no request lifecycle here; the
  subsystems are the domain.

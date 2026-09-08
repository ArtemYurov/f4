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
│       ├── *_test.go              # tests for the wiring itself, plus the
│       │                          # module-wide auditor (see below)
│       └── rsrc_windows_*.syso    # linked only from the built package's dir
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
│   ├── dummy_internal/ dummy_rpc/ dummy_lua/   # transport fixtures
│   └── plugring/     (move)       # installable-plugin catalogue: index.yaml and
│                                  # the plugins it points at. Data, no Go files.
│                                  # Keeps the Far-era name it is modelled on.
│
├── internal/                      # ── THE APPLICATION CORE IS MODULE-PRIVATE ──
│   │
│   │  # application core, extracted from the flat package main
│   ├── app/          (extract)    # event loop and bootstrap only — see the split
│   │                              # below; the shared primitives leave first
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
│   ├── unpack/       (extract)    # zip / tar.gz / 7z over a directory, and the
│   │                              # path guard its three callers share
│   ├── ini/          (extract)    # the ini parser the four leaves below share;
│   │                              # its own package because none of them may
│   │                              # import another of ours
│   ├── config/       (extract)    # F4Config, ini parsing, config overlay
│   ├── i18n/         (extract)    # language packs + embedded lang/
│   ├── theme/        (extract)    # colours, colour space, styles + embedded styles/
│   ├── keymap/       (extract)    # key remap, hotkeys, input translation
│   ├── fileops/      (extract)    # copy/move/delete, background jobs, clipboard
│   │
│   │  # self-contained subsystems, module-private (moved off the root)
│   ├── piecetable/   (move)       # piece table backing the editor
│   ├── textlayout/   (move)       # text layout and wrapping
│   ├── sheet/        (move)       # spreadsheet mode
│   ├── fusefs/       (move)       # FUSE mounting
│   ├── vtvibe/       (move)       # vtvibe session/provider layer
│   ├── luaplug/      (move)       # Lua plugin engine
│   ├── colorer/      (extract)    # colorer4go integration + embedded radiola.hrd
│   │
│   │  # platform helpers, already here
│   ├── wincon/  ttyx/  netproxy/
│   └── hideconsole/                # NOT our code: a vendored fork of
│                                   # github.com/ebitengine/hideconsole, wired in
│                                   # by `replace` in go.mod. Own go.mod, own
│                                   # module path — leave the path alone.
│
├── embedded.go                    # root package: embeds README.md, and only that.
│                                  # Must stay in the root — //go:embed cannot
│                                  # reach above its own directory, and README.md
│                                  # has to sit there to render on GitHub.
│
├── scripts/          (new)        # *.sh moved out of the repository root
├── docs/                          # subsystem documents; SPREADSHEET.md lands
│   └── ISSUES/                    #   here. Per-issue reviews are named
│                                  #   ISSUE_<number>_<SLUG>.md — the number
│                                  #   addresses the issue, the slug says what it
│                                  #   was about.
├── tools/                         # developer tooling incl. the ttytest harness
├── packaging/                     # distribution packaging
├── .github/
│   ├── workflows/                 # CI build matrix, nightly and tagged releases
│   └── assets/       (new)        # screenshot.png and other README media
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
- `plugins/plugring/` — the catalogue of installable plugins, named after the
  Far-era plugin ring it reproduces, next to the plugins
  that ship in the binary. It holds data, not Go files, so `./...` ignores it.
  Its `index.yaml` is fetched over HTTP from its repository path
  (`PlugRingCatalogURL`, `internal/plughost/plugring.go:24`), which makes that path a
  published contract: moving it means already-installed builds stop resolving
  the catalogue until they update. That is accepted here — the catalogue holds a
  single demonstration plugin, and application updates go through GitHub
  Releases, not this URL — but it is called out in the pull request rather than
  buried, and three references move with it: `PlugRingCatalogURL`, the developer
  fallback at `internal/plughost/plugring.go:55-56`, and the `url:` inside `index.yaml`
  that points at its own neighbour.
- `cmd/f4` — the entry point; it is `package main` and nothing can import it anyway.
- `embedded.go` — the root package that bridges root-level files into the binary.
  `//go:embed` cannot reach above its own directory, and `README.md` has to stay in
  the root to render on GitHub, so this one file is pinned there by the toolchain.

Everything else is module-private, so the compiler answers "may I import this?"
before a reviewer has to.

**`app` is two things, and only one of them is the root.** Today 37 call edges
run *into* the would-be `app` from lower layers — `toast.go` is called from six
different domains, `framework_actions.go` from four, the action registry from
panels and hotkeys. The dependency rule below forbids exactly that, so the pile
splits in two. Layer-0 utilities leave `cmd/f4` **first**, before any package
that calls them; the real composition root (`main.go`, `startup_*`,
`runtime_mode`) leaves last and keeps the name `app`. Extracting them in the
other order makes every intermediate commit uncompilable.

The split runs through files, not between them — three of these files are
themselves mixed, and moving one whole is what would break the build:

| file | layer-0 part | stays with the views |
|---|---|---|
| `action_registry.go` | the `Action` type and `RegisterAction`; its fields are `func() bool` closures, so the mechanism depends on nothing above layer 0 | the 2553-line `init()` registration table, which names `PanelsFrame` 109 times and `EditorView` 48 |
| `actions.go` | four functions — the far2l history helpers | everything else, and it does not go to one place: 61 carry a view type (52 take `*PanelsFrame` in the signature), and of the remaining view-free ones eight belong to `dialog`, three to `i18n`, three to `viewer`, one to `editor` |
| `misc.go` | the numeric helpers (`bounded*`, `nonNegativeUint64`, `runeCodepoint`) — called from eight future packages, the most widely shared code in the file | `ScreenRow` (used only by four test files) and `ReleaseHeavyMemory` (two files) |

`toast` and `path_identity` move whole. The history files — `history_provider.go`,
`history_dialog.go`, `command_history_paths.go`, `search_history.go`,
`menu_history.go` — are a cluster rather than strays: none of them touches a
message, a config field, a toast or a view type, so they form a layer-0 package
of their own.

`framework_actions.go` is not in this table: 18 of its 25 functions have no
external callers at all — they are `Handler:` values referenced from the
registration table. It reads as widely used only because registration looks like
calling. It travels whole with the composition root.

**`sysinfo` keeps its own copy of the one numeric helper it needs.** Its single
outbound edge is one call to `boundedUint64ToInt` (`cpu_info_darwin.go:33`).
Importing the shared numeric package would *create* the edge the rules forbid for
a layer-0 leaf rather than remove it, so a five-line private copy is what makes
its outbound count genuinely zero. Everyone else imports the shared package.

**One test does not follow its subject.** `command_palette_coverage_test.go`
walks the whole module and checks a global invariant: every `ProcessKey` and
every `vtui.NewVMenu` is either reachable from the command palette or listed as
a deliberate exception. That inventory cannot be split per package — a
per-package copy sees only its own subtree, and a handler added in a third
package passes unnoticed, which is the thing the test exists to catch. It stays
in `cmd/f4`.

Its 42 audit keys, however, are keyed by file path
(`cmd/f4/file_panel.go:(*FileSystemPanel).ProcessKey`), so every move rewrites
them. Re-key them once to the qualified symbol —
`panel.(*FileSystemPanel).ProcessKey` — before the extraction starts: a package
changes far less often than a path, files move freely inside their package, and
the package name is what identifies the subject anyway. The keys have to be
touched regardless; doing it as a re-keying instead of a path update ends the
tax rather than paying it on every commit.

**Embedded resources travel with their package.** The same `//go:embed` rule moves
each resource directory into the package that embeds it:

| resource | embedded by | lands in |
|---|---|---|
| `cmd/f4/styles/*.ini` | `style.go` | `internal/theme` |
| `cmd/f4/lang/*.lng` | `lang.go` **and** `lang_packs.go` | `internal/i18n` |
| `cmd/f4/help/en.hlf` | `help.go` | `internal/dialog` |
| `cmd/f4/assets/icon/generated/f4.icns` | `window_icon_darwin.go` | `internal/gui` |
| `colorer/…/radiola.hrd` | root `embedded.go` today | `internal/colorer` |

`lang/` is embedded from two different files, so both must land in the same
package or the directory ends up duplicated. Moving `radiola.hrd` to its single
consumer leaves root `embedded.go` with `README.md` alone — the case it exists
for. Windows `.syso` files are the exception that does not move: the toolchain
links them only from the directory of the package being built, so
`rsrc_windows_*.syso` stay in `cmd/f4` even though the icon code leaves.

## File Naming Inside a Package

A package's files are named `<topic>.go` for the core and `<topic>_<aspect>.go`
for everything that extends it. The prefix is the **topic inside the package**,
never the package name — `panel/frame.go`, not `panel/panel_frame.go`, the same
way `vfs/hostpath` holds `path.go` rather than `vfs_path.go`.

```text
internal/panel/
├── frame.go                # type PanelsFrame and its core methods
├── frame_dragdrop.go       # was cmd/f4/dragdrop.go
├── frame_translator.go     # was cmd/f4/translator.go
├── frame_semantic.go       # the PanelsFrame slice of cmd/f4/semantic.go
├── list.go                 # type FileSystemPanel
├── list_reconnect.go       # was cmd/f4/reconnect.go
└── list_semantic.go        # the FileSystemPanel slice of cmd/f4/semantic.go
```

Why it matters here specifically: Go requires a type's methods to live in the
type's package, so a package that owns a large type accumulates files. Sorted by
topic they read as one subject; named after unrelated features
(`dragdrop.go`, `translator.go`, `semantic.go`) they read as a pile, and a file
holding methods of six different types cannot be placed at all.

The convention is already half-present in the tree — `editor_view.go`,
`editor_base64.go`, `editor_find_all.go`, `command_palette_*.go`, `image_*.go`.
Extraction finishes it rather than introducing it. Platform suffixes compose on
the end as usual: `frame_dragdrop_windows.go`.

**Splitting a multi-type file.** When one file carries methods of several types
(`semantic.go` holds six), it is split along type lines and each piece lands in
its own package under its own topic name — `frame_semantic.go`,
`editor_semantic.go`, `viewer_semantic.go`. A method whose type lives elsewhere
but whose logic belongs to this package becomes a plain function taking the type
(`func handleDrop(pf *panel.PanelsFrame)`) rather than forcing the whole file
into the type's package.

## Dependency Rules

Dependencies point from the application inward to the subsystems. `cmd/f4/main.go`
is the only place where everything is assembled. Layers, bottom up:

**Layer 0 — kernel, no intra-module dependencies:** `vfs`, `sdk`,
`internal/piecetable`, `internal/sheet`, `internal/wincon`, `internal/ttyx`,
`internal/netproxy`, `internal/hideconsole`, `internal/config`, `internal/i18n`,
`internal/theme`, `internal/keymap`, `internal/sysinfo`, `internal/numeric`,
`internal/ini`, `internal/unpack`.

**Layer 1 — subsystems over the kernel:** `internal/textlayout` →
`internal/piecetable`; `internal/fusefs` → `vfs`; `internal/vtvibe` → `vfs`;
`internal/luaplug`; `internal/term`, `internal/gui`, `internal/media`,
`internal/fileops`, `internal/update` (self-update is a leaf with 3 outbound
edges, not an interactive subsystem).

**Layer 2 — plugins and hosts:** `plugins/*` → `vfs`, `sdk`, `internal/*`;
`internal/plughost` → `sdk`, `internal/luaplug`, `vfs`.

**Layer 3 — interactive subsystems:** `internal/panel`, `internal/editor`,
`internal/viewer`, `internal/dialog`, `internal/cmdline`, `internal/macro`.

**Layer 4 — application:** `internal/app`, then `cmd/f4`.

Rules:

- ✅ `cmd/f4` → any package (Composition Root).
- ✅ `internal/app` → every layer 0-3 package; it owns the event loop and the
  global application state that the interactive subsystems share.
- ✅ `plugins/*` → `vfs`, `sdk`, `internal/*` helper packages. Nothing imports a
  plugin back: they are leaves, wired in through `internal/plughost`.
- ✅ Any module → `internal/config`, `internal/i18n`, `internal/theme`,
  `internal/keymap`, `internal/sysinfo`, `vfs`. A layer-0 package imports layer-0
  packages and nothing else, which is what lets `config.App` stay a package-level
  global without creating a cycle: the package can only ever reach layer 0.
- ✅ Higher-layer modules talk to lower ones by calling exported constructors and
  methods; lower ones call back through interfaces they define themselves.
- ❌ `vfs` / `sdk` → any `internal/*` package. They are the public contract: an
  `internal/` import makes them uncompilable for third-party plugins, and the
  breakage surfaces only in someone else's build.
- ❌ `internal/piecetable` / `internal/sheet` → higher-layer packages. Kernel
  subsystems stay leaf nodes; that is what makes them testable in isolation.
- ❌ Any package → `cmd/f4`. It is `package main`; nothing can import it, and
  nothing should want to.
- ❌ Any package → a higher layer. A package at layer N imports layers N and
  below; `internal/app` at layer 4 is the case that matters most, but the rule is
  general and `architecture_test.go` checks every edge, not just that one. Shared
  state flows down through constructor arguments, never up through an import;
  where a lower layer needs something that lives above it, it declares the seam
  and the composition root fills it in.
- ❌ Import cycles between subsystem packages. If two need each other, the shared
  type belongs in a lower layer, or one of them defines an interface the other
  satisfies.
- ❌ Runtime `if runtime.GOOS == …` branching for platform differences. Use
  build-tag files (`*_windows.go`, `*_unix.go`, `*_other.go`).
- ❌ New cgo. FFI goes through `purego` / `ffibridge`.

**Not every directory here is one module.** The repository holds six `go.mod`
files: the main module, `internal/hideconsole` (a vendored fork substituted via
`replace`), and four under `tools/`. `go build ./...` and `go test ./...` see
only the main module's 38 packages — the others are built and tested separately,
which is why a broken test can sit in `tools/` unnoticed. Restructuring must not
rewrite the module path of a vendored fork, and moving one of these directories
means updating the `replace` directive that points at it (`go.mod:187`).

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

2. **Composition Root in `main.go`.** All wiring in one place; a `New(...)`
   takes what it needs, so a half-built struct is never reachable. Two things are
   banned outright: `init()` with side effects beyond assignment — today
   `queue_manager.go:312` starts a goroutine on import — and mutating a struct
   after construction that another goroutine already reads. A plain package-level
   value in a layer-0 leaf is not covered by this: `config.App` stays a global
   because 133 files read it and its package imports nothing, so it cannot create
   a cycle.

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
   coverage is not done. Shared test scaffolding gets its own home before the
   packages that use it move: `swapFrameManager` is used by 62 test files and
   `setupMockPanelsFrame` by 28. They cannot share one neutral package:
   `setupMockPanelsFrame` constructs a terminal view, a command line and a file
   panel, so a package holding it imports `panel`, `cmdline` and `term` — and
   those packages' own in-package tests then cannot import it back. The harness
   splits by what it touches: neutral `vtui` glue in `internal/testutil`, the
   frame mock in `internal/paneltest`, and the tests inside the three packages it
   depends on move to `package X_test`. Left undivided, the test import graph
   contradicts the production one.

6. **Portability is a boundary condition.** `CGO_ENABLED=0`, build-tag files, the
   full CI matrix green. A restructuring commit that only builds on the developer's
   own platform is a broken commit.

7. **The repository root is for entry points, not artefacts.** A newcomer should
   reach `README.md` without scrolling. Scripts go to `scripts/`, media to
   `.github/assets/`, prose to `docs/`. Per-issue write-ups belong in
   `docs/ISSUES/` — or, when produced through this harness, in the
   research → plan → archive chain under `.ai-factory/`.

8. **Documentation is part of the change, not its aftermath.** The 48 subsystem
   documents describe where things live, so a move that leaves them stale makes
   them actively misleading — worse than absent. Every move commit closes its own
   references (see the migration policy), and the restructuring as a whole revises
   `docs/` rather than only patching paths in it.

   Per-issue reviews are named `docs/ISSUES/ISSUE_<number>_<SLUG>.md`, keeping
   the existing SCREAMING_SNAKE style — `ISSUE_165_SORT_GROUPS.md`,
   `ISSUE_546_CONPTY_FOLLOWUP.md`. The convention is not invented here:
   `ISSUE_91_FREEBSD_CONSOLE_DIAGNOSIS.md` already carries a slug and is the
   precedent. The other 40 read `ISSUE_<n>_SOLUTION_REVIEW.md` — names
   distinguished only by a number, so finding the review of a subject requires
   already knowing its issue number. The slug replaces the constant
   `SOLUTION_REVIEW` tail, which carried no information: every file in the
   directory is a solution review. Renaming updates the links that point at
   them, by the same rule as any other move.

## Package Ownership

Every package answers for one subject, and every file belongs to exactly one
package. This outlives the extraction: once the tree is split, the rules below
are what keeps it split.

- **A new file goes to the package that owns its subject.** If none owns it,
  create the package — do not widen a neighbouring one because it is close
  enough, and never park it in `internal/app`. A composition root that accretes
  unrelated code is the flat package growing back one file at a time.
- **`internal/app` holds wiring, not features.** It constructs and connects; it
  does not implement. Code that lands there because nothing else fitted is code
  whose owner was not decided.
- **Cross-package work goes through the lower layer's own interface.** A package
  that needs something from a higher layer declares what it needs and lets the
  caller supply it. Reaching upward through an import, or through a shared
  mutable global, is the same mistake wearing two hats.
- **A method whose type lives elsewhere is a function.** When logic belongs here
  but the type belongs there, write `func doX(t *other.Type)` rather than
  dragging the file into the type's package.
- **The compiler is the reviewer.** `cmd/f4/architecture_test.go` asserts the
  layer rules — no `sdk`/`vfs` import of `internal/`, nothing importing the main
  package, no import of a higher layer, no cycles, and every `internal/*` package
  placed in the layer map, because an unplaced one is unchecked rather than
  exempt. A change
  that needs an exemption there is a change to this document first, not a test
  edit.

**Schema travels with the field; application stays behind.** When a package owns a
setting, it owns the setting's *schema* — the enumeration of what the field may
hold, the parser that normalises it, the default it falls back to. What it does
not own is the *use* of that setting by a running application. `config` holds
`StartupMode` and the function that parses it; the code that acts on a startup
mode lives with the code that starts things. `saveSettingsGroups` does not move
into `config` either — capturing the window geometry, writing the ini and saving
the session is orchestration, and orchestration belongs to whoever orchestrates.

The test is what a change would follow. A new value for an enumeration changes the
type and its parser: schema, so it moves with the field. A new place that reacts to
that value changes a caller: application, so it stays. Applied consistently this
keeps a configuration package from slowly becoming the place where everything that
mentions a setting ends up — the failure this whole layout exists to prevent, in
miniature.

Role separation is the same rule seen from the other side: `sdk/` and `vfs/`
define contracts, `plugins/` implement them, `internal/*` runs the application,
`cmd/f4` wires it together, `tools/` serves developers and ships in nothing. A
file that would do two of these jobs is two files.

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
- **Registration order is behaviour, not detail.** `action_registry.go:254` holds
  a 2553-line `init()` with 174 `RegisterAction` calls, and that order is what the
  menus and the command palette display. Inside one package Go runs `init()` in
  filename order; across packages it follows the import graph. Splitting the
  registry therefore reorders the menu silently, and no test catches it. Make the
  order explicit — sort at registration or register from one ordered list — before
  the files separate.
- **A move is not done until the prose agrees.** 24 documents reference paths that
  change, `docs/VTVIBE.md` alone 34 times, `AGENTS.md` 12. Each move commit greps
  `docs/`, `README.md` and `AGENTS.md` for the old path; a surviving reference is
  an unfinished move, not a follow-up.
- **Extraction order:** leaf-first, ranked by **outbound** edges — how much a
  package still drags out of `cmd/f4`, not how many callers it has. A package
  with many callers and few dependencies is an early candidate, not a late one.
  Measured on the call graph: `sysinfo` (1 outbound), `update` (3), the config
  group (`config`/`i18n`/`theme`/`keymap`), then `dialog`/`plughost`/`gui`/`macro`
  (7 each), `viewer` (9), `term` (12) ahead of `media` (10) because six of media's
  ten point at term, `fileops` (13), `editor` (23), `panel` (30), `cmdline` (41).
  The composition root goes last.
- **Interoperability:** while a subsystem is half-extracted, the extracted package
  must not import `cmd/f4` back — that is impossible for `package main` anyway,
  which is precisely what makes leaf-first ordering the only workable order.

## Code Examples

### Composition Root — construction, not global state

```go
// cmd/f4/main.go
func main() {
    flags := parseFlags()
    cfg := config.Load(flags.ConfigPath)

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
// internal/panel/frame.go — the panel says what it needs from a plugin host;
// it does not import internal/plughost, so plughost can depend on panel types
// later without producing a cycle.
type PluginColumns interface {
    ColumnsFor(ctx context.Context, path string) ([]Column, error)
}

func New(cfg *config.Config, fs vfs.FileSystem, side Side) *Panel { … }
```

```go
// internal/app/app.go — the app is the only place that knows both sides exist.
func New(cfg *config.Config, fs vfs.FileSystem, host *plughost.Host,
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

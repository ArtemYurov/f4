# Implementation Plan: Restructure f4 into internal packages

Branch: feature/restructure-into-internal-packages
Created: 2026-09-07
Base revision: `089fdc64` (rebased onto `upstream/main` at `f9fc5141`)

## Original Request

> Задание: составить план реструктуризации репозитория f4 через /aif-plan full. Все
> архитектурные решения уже приняты и закоммичены — их не пересматривать, а превратить
> в исполнимый план.
>
> Что от тебя нужно: план с фазами и задачами, где каждая задача называет конкретные
> файлы и правки, а не «перенести подсистему». Порядок обязан быть таким, чтобы КАЖДЫЙ
> промежуточный коммит собирался на всей матрице — в частности, общие примитивы (toast,
> реестр действий, framework_actions, истории) обязаны покинуть cmd/f4 до пакетов,
> которые их зовут, иначе получится циклический импорт internal/panel ↔ internal/app.
>
> Цифры в этом сообщении — вход, а не истина: проверяй их через codegraph, прежде чем
> строить на них фазу. Где расходится — доверяй графу и скажи об этом.

## Settings

- Testing: yes — the 346 `cmd/f4/*_test.go` files move with their subjects; one new
  architecture-boundary test is added in Phase 0.
- Logging: project convention, nothing added. Diagnostics stay on `VTUI_DEBUG`
  (`cmd/f4/debug_log.go`), user-facing failures on stderr with the `f4: ` prefix.
  A move commit that introduces a log line is not a move commit.
- Docs: yes — mandatory `/aif-docs` checkpoint after the last extraction commit, on
  top of the per-commit reference sweep required by ARCHITECTURE.md.

Delivery: **one pull request to `unxed/f4`** with meaningful commits inside. The
harness scaffolding (`.ai-factory/`, `.claude/`, `.mcp.json`, `AGENTS.md`) ships in
that PR deliberately, as part of the proposal.

---

## Graph Corrections

Every number below was re-derived from the CodeGraph index and the working tree at
`089fdc64`. Where it disagrees with the reconnaissance input or with
`ARCHITECTURE.md`, the graph wins and the plan is built on the graph.

### Corrections that change the plan

1. **`actions.go` is not a layer-0 unit.** It holds 81 functions: 80 free functions
   plus one `*PanelsFrame` method (`hostConsoleLogFallback`, `actions.go:2194`). Of
   the 81, **61 reference a view type** and **52 take `*PanelsFrame` in their
   signature**; only 20 are free of `PanelsFrame` / `EditorView` / `ViewerView` /
   `FileSystemPanel`. `ARCHITECTURE.md` lists `actions.go` among the shared
   primitives that leave first — as a *file* it cannot, because 5332 of its lines
   depend on layer-3 types. The primitives wave therefore moves **the 20 view-free
   helpers**, and the other 61 travel with the view they serve.

2. **The action registry splits the same way.** The `Action` struct
   (`action_registry.go:24`) is clean: its `Checked` / `Visible` / `Handler` fields
   are `func() bool` closures, so the type and `RegisterAction` depend on nothing
   above layer 0. But the 2553-line `init()` (`action_registry.go:254-2807`, 174
   `RegisterAction` calls) mentions `PanelsFrame` 109 times and `EditorView` 48
   times inside those closures. Mechanism is layer 0; the registration table is
   layer 4. They must be separated before the file can move.

3. **`framework_actions.go` is not clean either.** 25 functions over
   `vtui.AppScreen`, but `forkNearestPanelsFrame` (`framework_actions.go:141`) and
   7 more references reach `PanelsFrame`. Same treatment as `actions.go`.

4. **Three config field types live outside `config.go`, not two.** `F4Config`
   (`config.go:352`, 132 fields) has 5 named-type fields. Two are declared in
   `config.go` (`PanelScrollbarMode:125`, `WorkspaceTabNumberingMode:144`). Three
   are not:
   - `PanelNavigationMode` — `navigation_mode.go:7`
   - `compareOptions` — `compare_folders.go:61`
   - **`StartupMode` — `startup_backend.go:11`**
   The third is the dangerous one: `startup_*` is named in `ARCHITECTURE.md` as part
   of the composition root that leaves **last**, so `internal/config` (layer 0)
   would depend on `internal/app` (layer 4). All three move into `config.go` in
   Phase 0.

5. **`misc.go`'s split is inverted relative to `ARCHITECTURE.md`.** The document
   keeps "`misc.go` minus its numeric helpers" as a primitive. The graph says the
   numeric helpers are the *widest-shared* part — `bounded*` / `nonNegativeUint64` /
   `runeCodepoint` are called from `cpu_info_darwin.go` (sysinfo), `extui_host.go`
   (plughost), `semantic.go` (five future packages), `disasm.go` +
   `quick_view_panel.go` (viewer), `session_unix.go` (term),
   `input_translation.go` (keymap), `macro_lua_api.go` (macro),
   `translate_kitty.go` (term). Meanwhile `ScreenRow` is called **only from four
   test files** and `ReleaseHeavyMemory` only from `editor_view.go` and
   `viewer_view.go`.

6. **`sysinfo` cannot import the numeric helpers.** The dependency rule states
   `internal/sysinfo` "imports no other `internal/*` package". Its single outbound
   edge is one call to `boundedUint64ToInt` at `cpu_info_darwin.go:33`. Extracting
   the helper to a shared package would *create* the edge the rule forbids, not
   remove it. `internal/sysinfo` keeps a private 5-line copy; everyone else imports
   the shared package. That is what makes its outbound count actually 0.

7. **The shared test harness cannot be one neutral package.**
   `setupMockPanelsFrame` (`panels_frame_test.go:794`) calls `NewTerminalView`
   (term), `NewCommandLine` (cmdline), `NewFileSystemPanel` (panel),
   `PanelsFrame.initPTY` (panel) and constructs `PanelsFrame`. A single
   `internal/testutil` holding it would import `panel`, `cmdline` and `term` — and
   then the in-package tests of those very packages (`package panel`) importing it
   is an import cycle the compiler rejects. Design is fixed in Phase 0, task T07.

8. **CI shard definitions do not need per-commit recomputation.** The lint shards
   (`build.yml:983`) split `./cmd/f4/...` against everything else, and the race
   `packages` scope is computed as `go list ./... | grep -Ev '…/cmd/f4$'`
   (`build.yml:1332`). As files leave `cmd/f4` they migrate between shards on their
   own; both halves stay correct, only progressively imbalanced. Rebalance once, at
   the end (T35), not fourteen times.

9. **One CI trap is a silent test drop, not a red build.** `build.yml:1187` runs the
   suite with `-skip '^TestAllDialogs_LayoutValidation$'` globally, and re-runs that
   test single-threaded only when the target list contains `./...` or
   `github.com/unxed/f4/cmd/f4` (`build.yml:1193-1194`). `TestAllDialogs_LayoutValidation`
   lives in `dialog_layouts_test.go` → `internal/dialog`. The moment it moves, it is
   skipped everywhere and re-run nowhere. Green build, test gone. Fixed in the
   dialog commit (T22).

10. **`plugring_test.go` works today and will break on the move.**
    `plugring_test.go:40` resolves the catalogue through `runtime.Caller(0)` →
    `../../plugring/index.yaml`, which succeeds. It is `plugring_policy_test.go:40`
    that reads a CWD-relative `plugring/index.yaml` and always hits `t.Skipf`. The
    move must fix the first and decide the second.

11. **`README.md:3` is an absolute remote URL, not a relative path:**
    `https://raw.githubusercontent.com/unxed/f4/refs/heads/main/screenshot.png`.
    Moving the file to `.github/assets/` and updating the URL leaves the README
    image 404 on the branch until the PR merges into `main`. Expected; call it out
    in the PR body rather than being surprised by it.

### Corrections that only sharpen the numbers

12. Type-method spread is wider than reported: `PanelsFrame` has 161 methods across
    **17** non-test files (not 16), `EditorView` 187 across **16** (not 15),
    `FileSystemPanel` 107 across **7** (not 6), `coreAPI` 17 across 7 ✓.
13. `swapFrameManager` is referenced in 65 files, but two of those are prose
    comments in `config.go:1204` and `queue_manager.go:138` — **63 test files**
    actually use it. `setupMockPanelsFrame`: 28 ✓.
14. `cmd/f4` holds 345 non-test `.go` files (109 402 lines) and **346** `_test.go`
    files (96 495 lines), not 344. Repository-wide: 656 non-test, 564 test.
15. `AppConfig` is referenced in 133 files, of which **55 are non-test**. The other
    78 are tests.
16. Documentation debt is 28 files, not 24: `docs/VTVIBE.md` 34 mentions ✓,
    `AGENTS.md` **13** (not 12), 20 files under `docs/` mention `cmd/f4`.
17. Action registration is spread over **7** files with `init()`, not one:
    `action_registry.go`, `fuse_mount_action.go` (two `init()`s),
    `fuse_mount_list.go`, `sheet_actions.go`, `sqlite_actions.go`,
    `static_direct_actions.go`, `vtvibe_host.go`. 19 `init()` total in non-test
    code ✓.
18. `colorer/` holds exactly one file: `colorer/configs/base/hrd/rgb/radiola.hrd`.
19. Build tags confirmed, including both traps: `pty_unix.go` is `//go:build linux`
    and `solaris_pty.go` is `//go:build !windows`. 97 non-test files carry a tag,
    across 28 distinct tag expressions.

---

## Ground Rules

These hold for every commit in this branch. A commit that breaks one is not done.

- **Compiles on the whole matrix.** `CGO_ENABLED=0 go build ./...` plus the
  cross-compile matrix, exotic targets included. Move files by their `//go:build`
  line, never by their filename.
- **No rewrites inside a move commit.** A reviewer must be able to read the diff as
  a rename. Behaviour changes (splitting `init()`, turning a method into a function,
  exporting an identifier) happen in their own commit, before or after.
- **The extracted package never imports `cmd/f4`.** Impossible for `package main`,
  which is exactly why leaf-first is the only workable order.
- **Extraction gate.** A package leaves `cmd/f4` only when every symbol it calls
  already lives in an extracted package, in itself, or outside the module. Verify
  before each wave with
  `npx -y @colbymchenry/codegraph@1.6.0 callees <symbol>` over the wave's exported
  entry points, and by the fact that it compiles.
- **Prose agrees or the move is unfinished.** Each move commit greps `docs/`,
  `README.md` and `AGENTS.md` for every path it changed. A surviving reference is
  part of this commit, not a follow-up.
- **Infrastructure travels in the same commit as the files it points at**
  (`.github/workflows/build.yml`, `.github/actions/affected-packages/action.yml`,
  `tools/`, `plugins/netfox/lang_test.go`).
- **Baseline comparison after every commit.** New red test = regression of this
  move. Old red test = known state (T01).
- Use the system Go build cache (`go env GOCACHE`); never redirect it.
- `git mv` for every file already tracked by git.

---

## Commit Plan

Fourteen groups. Each is one commit unless noted; each is independently green.

| # | Commit | Tasks |
|---|---|---|
| 1 | `test(ci): record the pre-restructuring test baseline` | T01 |
| 2 | `refactor(actions): make registration order explicit` | T02, T03 |
| 3 | `refactor(config): give F4Config's field types a home in config.go` | T04 |
| 4 | `refactor(app): start the operation queue from the root, not from init()` | T05 |
| 5 | `test(arch): add the module boundary auditor` | T06 |
| 6 | `test: give the shared frame harness a home` | T07 |
| 7 | `chore: clear the repository root` (issue #505) | T08–T13 |
| 8 | `refactor: move the self-contained subsystems under internal/` | T14, T15 |
| 9 | `refactor: extract the shared primitives from cmd/f4` | T16–T18 |
| 10 | **fourteen commits**, one per wave: `refactor(<pkg>): extract internal/<pkg> from cmd/f4` | T19–T32 |
| 11 | `refactor(app): extract the composition root` | T33, T34 |
| 12 | `ci: rebalance the shards for the split tree` | T35, T36 |
| 13 | `docs: describe the restructured tree` | T37 |
| 14 | `chore: drop the migration baseline` | T38 |

Twenty-seven commits in total: thirteen groups of one, plus fourteen wave commits.

---

## Tasks

### Phase 0 — Baseline and barrier removal

No file moves in this phase. It removes the four structural barriers that would
make an intermediate commit uncompilable, and it establishes what "green" meant
before we started.

- [ ] **T01 — Record the pre-restructuring test baseline.**
  Write `.ai-factory/RESTRUCTURE_BASELINE.md`: revision (`089fdc64`), platform, Go
  version, and one section per module. The repository holds **six** `go.mod` files
  and `go test ./...` sees only the main module's 38 packages, so each of the other
  five is run and recorded separately:
  - main module — `go build ./...`, `go vet ./...`, `go test -timeout 25m ./...`:
    all green, 30 packages ok, 8 without tests, 0 FAIL.
  - `tools/conptyreconcile` — ok.
  - `tools/icons` — **red before us**: `TestRenderScalesStrokesAndGradients`,
    `main_test.go:71` opens `../../assets/icon/f4.svg`, a path that exists neither
    now nor in the target tree (`cmd/f4/assets/icon/` today,
    `internal/gui/assets/icon/` after). Record it as pre-existing and **do not fix
    it by pointing the path at `cmd/f4/assets/icon`** — that decision belongs to
    whoever owns the icon pipeline, not to a move commit.
  - `tools/wine_syscall_probe` — does not build on darwin/arm64
    (`main.go:5`, `func rawGetpid() uint64` with its body in `probe_amd64.s`);
    builds clean under `GOOS=windows GOARCH=amd64`. Record it as a platform probe,
    **not** as breakage, so nobody "fixes" a working tool.
  - `tools/icons/third_party/oksvg` — vendored third-party, out of scope.
  - `internal/hideconsole` — a vendored fork of
    `github.com/ebitengine/hideconsole` wired in by `replace` (`go.mod:187`); no
    tests, module path is not ours to change.
  Also record `cmd/f4/plugring_policy_test.go:40` as green-but-inert: it reads a
  CWD-relative `plugring/index.yaml` from `cmd/f4/`, always hits `t.Skipf`, and
  asserts nothing.
  Finally, list the tests that cannot simply travel with their file and must be
  reworked instead: `command_palette_coverage_test.go` (T02) and everything on the
  `swapFrameManager` / `setupMockPanelsFrame` harness (T07).
  *No logging changes. Files: `.ai-factory/RESTRUCTURE_BASELINE.md`.*

- [ ] **T02 — Re-key the command-palette auditor to qualified symbols.**
  `cmd/f4/command_palette_coverage_test.go` carries 42 audit keys shaped
  `cmd/f4/file_panel.go:(*FileSystemPanel).ProcessKey`. Every move rewrites them.
  Re-key once to `panel.(*FileSystemPanel).ProcessKey` — package plus qualified
  symbol — and adjust the walker that produces the keys to emit the package name
  instead of the file path. The test itself stays in `cmd/f4` (it is the only
  module-wide invariant; a per-package copy would not see a handler added in a
  third package). Verify the key set is identical in size and content modulo the
  rewrite before and after.
  *Files: `cmd/f4/command_palette_coverage_test.go`.*

- [ ] **T03 — Make action registration order explicit.**
  Inside one package Go runs `init()` in filename order; across packages it follows
  the import graph. 174 `RegisterAction` calls sit in `action_registry.go:254-2807`
  and more in six other files (`fuse_mount_action.go` ×2, `fuse_mount_list.go`,
  `sheet_actions.go`, `sqlite_actions.go`, `static_direct_actions.go`,
  `vtvibe_host.go`), and that order is what menus and the command palette display.
  Splitting the registry reorders the menu silently and no test catches it.
  Give `Action` an explicit ordinal, or register from one ordered slice built at
  registration time and sorted before `actionOrder` is read. Add a golden test over
  the resulting `actionOrder` so a future reorder is a red test, not a UI surprise.
  This is a behaviour-preserving change and gets its own commit before anything
  moves.
  *Files: `cmd/f4/action_registry.go` (+ the six registration files), new
  `cmd/f4/action_registry_order_test.go`.*

- [ ] **T04 — Move `F4Config`'s field types into `config.go`.**
  `internal/config` must be a leaf that imports no other `internal/*` package.
  Three of `F4Config`'s five named-type fields are declared elsewhere:
  `PanelNavigationMode` (`navigation_mode.go:7`, with its `String()` and the
  `NavigationClassic/Vim/SearchFirst` constants), `compareOptions`
  (`compare_folders.go:61`), and `StartupMode` (`startup_backend.go:11`). Move the
  three type declarations and their constants into `config.go`; leave the behaviour
  that uses them where it is. `StartupMode` is the load-bearing one:
  `startup_backend.go` is composition-root code that leaves last, so leaving the
  type there means layer 0 depends on layer 4.
  *Files: `cmd/f4/config.go`, `cmd/f4/navigation_mode.go`,
  `cmd/f4/compare_folders.go`, `cmd/f4/startup_backend.go`.*

- [ ] **T05 — Stop starting a goroutine from `init()`.**
  `queue_manager.go:308-317` constructs `GlobalQueueManager` and calls
  `go GlobalQueueManager.workerLoop()` on import. Once `queue_manager.go` is in a
  package, importing that package starts a worker in every binary and every test
  process that touches it. Keep the zero-value construction (tests rely on a usable
  zero-value manager, see the `workerLoop` fallback comment), move the `go` call to
  an explicit `StartQueueWorker()` invoked from the startup path, and have the
  tests that need a running worker start it themselves.
  *Files: `cmd/f4/queue_manager.go`, `cmd/f4/main.go` (or the startup file that
  owns process-wide services), `cmd/f4/queue_manager_test.go`.*

- [ ] **T06 — Add the module boundary auditor.**
  New `cmd/f4/architecture_test.go`, standard library only, driven by
  `go list -f '{{.ImportPath}} {{join .Imports " "}}' ./...` — no new dependency,
  no `go/packages`. It asserts four rules from `ARCHITECTURE.md`:
  1. no package under `sdk/` or `vfs/` imports anything under `internal/`;
  2. no package imports `github.com/unxed/f4/cmd/f4`;
  3. no package below layer 4 imports `internal/app`;
  4. the import graph over the module's own packages is acyclic.
  Written now, it is green on today's tree and turns red the first time a wave
  introduces a violation — which is the whole point of adding it before the moves
  rather than after. Keep the layer table in the test as a single ordered
  `map[string]int` so each wave updates one line.
  *Files: new `cmd/f4/architecture_test.go`.*

- [ ] **T07 — Give the shared frame harness a home, with the cycle designed out.**
  `swapFrameManager` (`frame_manager_test_helpers_test.go:129`) is used by 63 test
  files; `setupMockPanelsFrame` (`panels_frame_test.go:794`) by 28. They cannot both
  go into one neutral package: `setupMockPanelsFrame` calls `NewTerminalView`
  (term), `NewCommandLine` (cmdline), `NewFileSystemPanel` and
  `PanelsFrame.initPTY` (panel), so a package holding it imports three layer-3
  packages — and their own in-package tests then cannot import it. Split by
  dependency depth:
  - `internal/testutil` — `SwapFrameManager(t, drains ...func(*testing.T))`,
    `SetFrameManagerScreens`, `CloseFrameManagerFrames`, `PumpUntilToastActive`,
    `WaitForToastExpiry`, plus `ScreenRow` (which `misc.go:12` exposes for tests
    only — four callers, all `_test.go`). Imports `vtui`/`vtinput` and nothing from
    `internal/*`. The two production drains `swapFrameManager` performs today —
    `waitForAsyncClipboard` (`clipboard_async.go:27`) and `waitForDirectoryLoads`
    (`frame_manager_test_helpers_test.go:83`, which waits on the
    `directoryLoadWorkers` production global) — become caller-supplied `drains`, so
    the harness stops reaching down into fileops and panel.
  - `internal/paneltest` — `SetupMockPanelsFrame`, created after `panel`, `cmdline`
    and `term` exist. Tests inside those three packages that need it become
    external test packages (`package panel_test`); tests outside them import it
    directly.
  Do this as one commit while everything is still in `cmd/f4`: the helpers get
  their new call shape and every caller is updated in a diff that touches only test
  files. `internal/paneltest` is created empty here with its contract documented,
  and filled in T30.
  *Files: new `internal/testutil/*.go`; `cmd/f4/frame_manager_test_helpers_test.go`,
  `cmd/f4/panels_frame_test.go`, and the 63 + 28 call sites.*

### Phase 1 — Clear the repository root (issue #505)

Independent of the Go package work and noisy in the diff, so it lands early and
alone.

- [ ] **T08 — `*.sh` → `scripts/`.**
  `git mv filelist_update.sh test_plugins.sh test_resurrect.sh scripts/`. Update
  `tools/test_runner.sh:7` (`./filelist_update.sh` → `./scripts/filelist_update.sh`)
  and any invocation in `docs/`.

- [ ] **T09 — `screenshot.png` → `.github/assets/`.**
  Update `README.md:3`. The reference is an absolute
  `raw.githubusercontent.com/unxed/f4/refs/heads/main/screenshot.png` URL, so the
  new URL 404s until the PR merges into `main` — note it in the PR body.

- [ ] **T10 — Loose prose → `docs/`, and `time.txt` out.**
  `git mv SPREADSHEET.md docs/` (update `AGENTS.md:74` and the docs table),
  `git mv ISSUE_95_FOLLOWUP_SOLUTION_REVIEW.md issue-703-solution.md docs/ISSUES/`
  under their new names (T11's scheme), `git rm time.txt` (three blank lines,
  referenced by nothing). `f4.example.ini` and `highlight.ini` stay: they are
  reference configs the README points at.

- [ ] **T11 — Rename the 41 issue reviews to `ISSUE_<number>_<SLUG>.md`.**
  All 41 currently read `ISSUE_<n>_SOLUTION_REVIEW.md` — 41 identical names
  distinguished only by a number. Derive each slug from the document's own subject
  (`ISSUE_165_SORT_GROUPS.md`, `ISSUE_546_CONPTY_FOLLOWUP.md`), keeping
  SCREAMING_SNAKE. `git mv` each, then grep `docs/`, `README.md`, `AGENTS.md` and
  the source tree for every old filename and update the links.

- [ ] **T12 — `colorer/` → `internal/colorer/`.**
  The directory holds exactly one file,
  `colorer/configs/base/hrd/rgb/radiola.hrd`. Move it under `internal/colorer/`,
  move the `//go:embed` from root `embedded.go:12` into a new
  `internal/colorer/embedded.go`, and repoint `colorer_plugin.go` (its only
  consumer) at the new symbol. Root `embedded.go` is left embedding `README.md`
  alone — the single case it exists for. This is the one extraction that can happen
  before the waves, because it has one consumer and no callers.
  *Files: `colorer/…/radiola.hrd`, `embedded.go`, new
  `internal/colorer/embedded.go`, `cmd/f4/colorer_plugin.go`, `docs/` refs.*

- [ ] **T13 — `plugring/` → `plugins/plugring/`.**
  Data only, no Go files. Three references move with it: `PlugRingCatalogURL`
  (`cmd/f4/plugring.go:20`), the developer fallback comparing against that same URL
  (`cmd/f4/plugring.go:48-49`), and the `url:` inside `plugring/index.yaml:7` that
  points at its own neighbour `hello_plugring.lua`. Fix
  `cmd/f4/plugring_test.go:40`, whose `runtime.Caller`-relative
  `../../plugring/index.yaml` works today and breaks on the move. Decide
  `cmd/f4/plugring_policy_test.go:40` explicitly: it reads a CWD-relative path,
  always skips, and asserts nothing — either point it at the real catalogue or
  delete it, but do not leave a test that silently checks nothing.
  The catalogue URL is a published contract: already-installed builds stop
  resolving it until they update. Accepted (one demonstration plugin; application
  updates go through GitHub Releases) but stated in the PR body, not buried.

### Phase 2 — Self-contained subsystems under `internal/`

Pure directory moves. No package is split, no identifier is renamed; only import
paths change.

- [ ] **T14 — `piecetable/`, `textlayout/`, `sheet/` → `internal/`.**
  `git mv` each, rewrite `github.com/unxed/f4/{piecetable,textlayout,sheet}` →
  `.../internal/…` across the module, update `docs/` (one file each for piecetable
  and textlayout) and `AGENTS.md`.

- [ ] **T15 — `fusefs/`, `vtvibe/`, `luaplug/` → `internal/`.**
  Same mechanics. `docs/VTVIBE.md` carries 34 path mentions and is the single
  largest documentation edit in the whole plan — do it here, in the commit that
  creates the drift.

### Phase 3 — The shared primitives leave `cmd/f4`

This is the phase that makes every later wave possible. Nothing here may depend on
a view type, which is why T16 splits the files first.

- [ ] **T16 — Separate the action registry's mechanism from its table.**
  Split `cmd/f4/action_registry.go` (2807 lines) in two, in place, before moving
  anything:
  - `action_registry.go` keeps the `Action` struct (`:24`), `DisplayLabel` /
    `DisplayDescription`, `actionRegistry`, `actionOrder`, `RegisterAction`
    (`:121`) and the lookup/ordering helpers. Verified clean: the type's
    `Checked` / `Visible` / `Handler` fields are `func() bool`, so nothing here
    names a view type.
  - `action_table.go` takes the 2553-line `init()` and its 174 `RegisterAction`
    calls, which mention `PanelsFrame` 109 times and `EditorView` 48 times. It
    stays in `cmd/f4` until the composition root moves (T32).
  Same treatment for the two grab-bags, by the counts in Correction 1 and 3:
  `actions.go` → the 20 view-free functions into `actions_shared.go`, the 61 others
  stay; `framework_actions.go` → everything except `forkNearestPanelsFrame` and its
  7 `PanelsFrame` references. `hostConsoleLogFallback` (`actions.go:2194`), the
  single `*PanelsFrame` method in the file, becomes a free function taking the
  frame, per the project decision on cross-package methods.
  *No files move in this task — it is a pure in-package split so the next task's
  diff is a rename.*

- [ ] **T17 — Create `internal/numeric` and `internal/vtuix`.**
  - `internal/numeric` — the seven `bounded*` helpers, `nonNegativeUint64`,
    `runeCodepoint` and `ReleaseHeavyMemory` from `misc.go`. Zero imports, layer 0.
    Their `#nosec G115` annotations travel verbatim; `gosec` runs in CI.
    **`internal/sysinfo` does not import it**: `cpu_info_darwin.go:33` is the one
    call site there and gets a private 5-line copy, which is what makes sysinfo a
    true zero-outbound leaf under the dependency rule.
  - `internal/vtuix` — f4's conventions over the `vtui`/`vtinput` widgets, named
    after the existing `internal/ttyx` precedent: `toast.go` (15 lines, wrapping
    `vtui.ShowToast` with the test-only `toastDurationOverride` seam; called from
    16 files across 8 future packages), `search_history.go` (`attachHistory`,
    `attachHistoryUseLast`, `commitHistory`, `inputBoxEdit` — `attachHistory` alone
    has 7 caller files), `menu_history.go` minus `actionSelectLastMenuItem`, which
    is an action handler and stays behind.
  Export what leaves; `showToast` → `vtuix.ShowToast` and so on across all call
  sites. This is the one large mechanical rename in the plan and it belongs here,
  before any package that calls these exists.
  *Files: new `internal/numeric/*.go`, `internal/vtuix/*.go`; `cmd/f4/misc.go`,
  `toast.go`, `search_history.go`, `menu_history.go`, `cpu_info_darwin.go`, and the
  call sites.*

- [ ] **T18 — Create `internal/action`.**
  Move `action_registry.go`'s mechanism half (T16) plus the view-free part of
  `framework_actions.go` into `internal/action`. `Action.DisplayLabel` calls `Msg()`
  — that is `internal/i18n`, layer 0, and the import is allowed once T21 lands, so
  either sequence this task after T21 or have `internal/action` take the localizer
  as a function value. Prefer the second: it keeps `internal/action` importing
  nothing and removes an ordering constraint.
  `cmd/f4/action_table.go` now calls `action.RegisterAction` from `cmd/f4`, which
  is legal in both directions of the migration.
  *Files: new `internal/action/*.go`; `cmd/f4/action_table.go` and every
  registration file.*

### Phase 4 — Extraction waves

One package per commit, ranked leaf-first by **outbound** edges as recorded in
`ARCHITECTURE.md`. Each wave follows the same five steps, and the task text below
records only what is specific to that wave:

1. Confirm the gate: run `callees` over the wave's entry points and check that no
   target is still an unextracted `cmd/f4` symbol. Where one is, resolve it first —
   usually by moving the target into an earlier wave or turning a cross-package
   method into a free function.
2. `git mv` the files **and their `_test.go` neighbours**, selecting per-OS files by
   their `//go:build` line and never by their name (`pty_unix.go` is
   `//go:build linux`; `solaris_pty.go` is `//go:build !windows` and contains no
   PTY code).
3. Rename to the package's file convention: `<topic>.go` / `<topic>_<aspect>.go`,
   prefix is the topic inside the package, never the package name — `panel/frame.go`,
   not `panel/panel_frame.go`. Platform suffixes compose on the end.
4. Export what `cmd/f4` still needs; leave everything else unexported.
5. Sweep `docs/`, `README.md`, `AGENTS.md`; fix the CI and tooling references this
   wave touches; re-run the T01 baseline.

- [ ] **T19 — `internal/sysinfo` (1 outbound).**
  `cpu_info*.go`, `mem_info*.go`, `fs_info*.go`, `gpu_info*.go`, `drives_unix.go`,
  `drives_windows.go` and their tests. The single outbound edge is resolved by T17's
  private copy. `drive_bookmarks*.go`, `drive_menu_options*.go` are UI over the
  drive list and stay for the panel wave.

- [ ] **T20 — `internal/update` (3 outbound).**
  Self-update, elevation and helper-argument handling. Layer 1, not an interactive
  subsystem.

- [ ] **T21 — `internal/config`, `internal/i18n`, `internal/theme`, `internal/keymap`
  (15 outbound as a group).**
  Four layer-0 leaves that must be split apart in one commit because they reference
  each other's globals today.
  - `internal/config` — `config.go` with `F4Config` (132 fields) and the `AppConfig`
    global. It stays a package-level global by design: 133 files read it (55
    non-test), and the package imports no other `internal/*`, so it cannot create a
    cycle. T04 already gave its five named field types a home here.
  - `internal/i18n` — `lang.go`, `lang_packs.go`, `Msg()` (79 non-test files use
    it), and the embedded `cmd/f4/lang/` directory. `lang/` is embedded from **two**
    files, so both must land here or the directory ends up duplicated.
    Update `tools/langfmt/main.go:47` (default `-source`),
    `build.yml:87` (the explicit `-check cmd/f4/lang/*.lng`),
    `build.yml:178/406/580/812` (`cp -r cmd/f4/lang cmd/f4/help build/`),
    `cmd/f4/lang/README.md:5-6,24`, and
    `plugins/netfox/lang_test.go:22`, whose `hostStringsPath` constant reads the
    file at test runtime.
  - `internal/theme` — `colors.go`, `style.go` and the embedded `cmd/f4/styles/`.
  - `internal/keymap` — hotkeys, key remapping, `input_translation.go`.
  Note that the `help/` half of the `cp -r cmd/f4/lang cmd/f4/help` CI lines does
  not move until T22 — that command is touched twice, once here and once there.

- [ ] **T22 — `internal/dialog` (7 outbound).**
  Modal dialogs, command palette, menus, `help.go`, `help_search.go`, `grabber.go`,
  `themed_table.go`, and the embedded `cmd/f4/help/en.hlf`.
  Two CI edits belong to this commit and nowhere else:
  - the `help` half of `build.yml:178/406/580/812`;
  - **`build.yml:1193-1194`** — `TestAllDialogs_LayoutValidation` moves out of
    `cmd/f4` with `dialog_layouts_test.go`, and the global
    `-skip '^TestAllDialogs_LayoutValidation$'` at line 1187 then hides it while the
    isolated single-threaded re-run no longer matches its target. Repoint the
    re-run at `./internal/dialog` (and keep the skip global). Confirm afterwards
    that the test actually ran — its absence is silent.

- [ ] **T23 — `internal/plughost` (7 outbound).**
  All four transports. Per the graph, `api.go` (the `coreAPI` type, 17 methods
  across 7 files), `sqlite_actions.go` and `plugin_hotkeys.go` belong here despite
  their names. `extui_host.go` comes here too; it is the heaviest consumer of the
  `internal/numeric` helpers.

- [ ] **T24 — `internal/gui` (7 outbound).**
  GUI backends, fonts, window position, and the embedded
  `cmd/f4/assets/icon/generated/f4.icns` (`window_icon_darwin.go`).
  `dragdrop.go` and `winepath_{other,windows}.go` are assigned here by the call
  graph, but `dragdrop.go` carries five `*PanelsFrame` methods and one
  `*FileSystemPanel` method, so it travels **whole** to `internal/panel` (T30) as
  `frame_dragdrop.go`. Do not split it.
  Infrastructure in this commit: `build.yml:93` (`go generate ./cmd/f4`),
  `95-97` (the `cmd/f4/assets/icon/generated` and `rsrc_windows_*.syso` paths — the
  `.syso` files themselves stay in `cmd/f4`, only the icon directory moves),
  `191/195/418/422` and `304`.
  `tools/icons/main.go` has three path constructions and **only two of them move**:
  `iconDir` (`:37`) and `outDir` (`:38`) become
  `filepath.Join(root, "internal", "gui", "assets", "icon"…)`, while
  `cmd.Dir = filepath.Join(root, "cmd", "f4")` at `:160` **stays** — its own comment
  says why (`rsrc_windows_*.syso` must sit in the main package directory to be
  linked). Rewriting all three is the obvious mistake here. The doc comment at
  `:1-2` moves with `iconDir`.
  `tools/icons/main_test.go:71` is already red for an unrelated reason (T01): it
  reads `../../assets/icon/f4.svg` while the tool itself reads
  `../../cmd/f4/assets/icon/` — the test never agreed with its subject. Leave it red
  and say so in the commit message rather than fixing it blind.

- [ ] **T25 — `internal/macro` (7 outbound).**
  Macro engine and the Lua macro API. `macro_lua_api.go` uses
  `numeric.BoundedRune`.

- [ ] **T26 — `internal/viewer` (9 outbound).**
  F3 viewer, hex, disasm. The graph assigns `top_bar.go`, `file_title.go` and
  `url_links.go` here, which is what removes the `editor ↔ viewer` cycle — take
  them even though their names suggest otherwise. `semantic.go`'s three
  `*ViewerView` methods come here as `viewer_semantic.go` (T31 owns the split).

- [ ] **T27 — `internal/term` (12 outbound), ahead of media.**
  pty, console host, ttyx glue, ANSI parser. Files the graph reassigns here against
  their names, each with the caller that proves it:
  - `kitty_graphics.go`, `kitty_placements.go`, `sixel_decode.go`,
    `sixel_terminal.go` — called by `ansi_parser.go` itself;
  - `command_runner{,_unix,_windows}.go`, `shell_mode.go`,
    `wine_probe{,_windows,_other}.go` — called by
    `pty_bsd.go` / `pty_darwin.go` / `solaris_pty.go` / `pty_unix.go` /
    `pty_ptm_*.go`. Leaving them in `cmdline` inverts the layers;
  - `graphics_compat.go`, `graphics_probe_decision.go`,
    `graphics_probe_windows.go`, `far2l_image.go` — they decide what the terminal
    supports, not what the GUI draws;
  - `clipboard.go`, `clipboard_async.go`, `background_jobs.go` — pulled forward into
    this wave specifically to avoid a `fileops ↔ term` cycle in T29.
  `waitForAsyncClipboard` (`clipboard_async.go:27`) is one of the two drains T07
  turned into caller-supplied functions; check that the `internal/testutil` callers
  now reference the exported form.

- [ ] **T28 — `internal/media` (10 outbound, six of them into term).**
  Image, audio and video decode and preview; `player_panel.go` lands here.

- [ ] **T29 — `internal/fileops` (13 outbound).**
  Copy / move / delete, background job orchestration, `file_op_dialog.go`,
  `queue_manager.go` (whose `init()` T05 already defused). `path_identity.go` lands
  here as well: its two functions are called from `file_ops.go` (fileops) and
  `panels_frame.go` / `file_panel.go` (panel), and panel → fileops is a legal
  layer-3 → layer-1 edge, so this keeps two exported functions off the public `vfs`
  surface.

- [ ] **T30 — `internal/editor` (23 outbound).**
  The F4 editor over `internal/piecetable`. `mapped_file{,_unix,_windows}.go` come
  here — the file's own comment calls it the piece table's original buffer.
  `EditorView` has 187 methods across 16 files, all of which must land in this
  package; `semantic.go`'s five `*EditorView` methods arrive as
  `editor_semantic.go`.

- [ ] **T31 — `internal/panel` (30 outbound), and the multi-type files.**
  `PanelsFrame` (161 methods across 17 files) and `FileSystemPanel` (107 across 7)
  both live here, which is what makes the awkward files simple:
  - `dragdrop.go` (5 × `PanelsFrame` + 1 × `FileSystemPanel`) → `frame_dragdrop.go`,
    whole;
  - `translator.go` (1 each) → `frame_translator.go`, whole;
  - `panel_plugins.go` (13 × `pluginPanelInstance`, 2 × `PanelsFrame`,
    1 × `coreAPI`) → cut the single `coreAPI` method out to `internal/plughost` as
    a free function; the rest moves whole;
  - `semantic.go` (808 lines, methods of six types) → this wave takes the four
    `*PanelsFrame` and one `*FileSystemPanel` methods as `frame_semantic.go` and
    `list_semantic.go`. The `*TerminalView` and `*CommandLine` methods went with
    T27 and T32; `*EditorView` with T30, `*ViewerView` with T26. The 14 free
    functions in the file follow their callers; `semanticInt` uses
    `numeric.BoundedUint64ToInt`.
  Also here: `drive_bookmarks*.go`, `drive_menu_options*.go`, `temp_panel.go`,
  `quick_view_panel.go`, `info_panel.go`, `reconnect.go` → `list_reconnect.go`.
  Fill `internal/paneltest` (T07) with `SetupMockPanelsFrame` in this commit and
  convert the in-package tests that use it to `package panel_test`.

- [ ] **T32 — `internal/cmdline` (41 outbound).**
  Command line, prefixes, history, `apply_command_*`. `CommandLine`'s 12 methods
  across 2 files, plus `semantic.go`'s single `*CommandLine` method. Explicitly
  **not** here: `command_runner*.go`, `shell_mode.go`, `wine_probe*.go` — they went
  to `internal/term` in T27.

### Phase 5 — The composition root

- [ ] **T33 — `internal/app`.**
  What is left after Phase 4: the event loop, bootstrap, `runtime_mode`, the global
  application state the interactive subsystems share, and `action_table.go` (the
  2553-line registration table from T16, whose closures reach `PanelsFrame` and
  `EditorView` and which therefore could never have been layer 0). `startup_*` moves
  here too — T04 already extracted `StartupMode` so this no longer drags the config
  package up a layer.
  Verify `cmd/f4/architecture_test.go` rule 3 (nothing below layer 4 imports
  `internal/app`) is green without exemptions.

- [ ] **T34 — Reduce `cmd/f4` to the entry point.**
  What remains: `main.go` (flags, startup mode, wiring), the tests for the wiring
  itself, `command_palette_coverage_test.go` (the module-wide auditor, re-keyed in
  T02), `architecture_test.go` (T06), and `rsrc_windows_amd64.syso` /
  `rsrc_windows_arm64.syso` — the toolchain links `.syso` files only from the
  directory of the package being built, so they stay even though the icon code left
  in T24.
  `.github/actions/affected-packages/action.yml:48-54, 76-77` treats `cmd/f4` as an
  indivisible unit "because its files always affect its tests at run time". That is
  no longer true of a package holding `main.go`; drop the special case here.

### Phase 6 — Infrastructure and documentation

- [ ] **T35 — Rebalance the CI shards.**
  The `cmd/f4` vs `rest` split exists because "cmd/f4's suite alone takes as long
  under the detector as every other package combined" (`build.yml:1262-1268`). After
  Phase 5 that is false, and the three letter-range race shards
  (`build.yml:1271-1273`, `1318`, `1326`) are splitting an empty package. Replace
  the name-based shards with a split computed from `go list ./...` so no shard
  definition names a package path, and drop the now-redundant
  `grep -Ev '…/cmd/f4$'` at `build.yml:1332`. Do this **once, here** — the shards
  stayed correct throughout Phase 4 (Correction 8), only imbalanced.
  Re-measure the wall-clock of the lint and race jobs before and after; a
  rebalancing that makes CI slower is not done.

- [ ] **T36 — Run the incremental lint against `origin/main` before the PR.**
  `.golangci.yml` / `.golangci-strict.yml` contain no paths and need no edit, but
  the incremental lint runs `--new-from-rev=origin/main`. A changed `package` clause
  can defeat golangci-lint's rename detection and surface the ~2450-finding backlog
  as new findings on this PR. Run it locally on the full branch before opening the
  PR; if the backlog does surface, say so in the PR body with the measured number
  rather than letting the maintainer discover a red job.

- [ ] **T37 — `/aif-docs` checkpoint.**
  Per-commit sweeps kept the 28 affected documents from going stale, but the
  restructuring as a whole changes what the documentation *describes*, not only the
  paths in it. Revise `AGENTS.md` (13 path mentions, plus the whole "Project
  Structure" block that still says "687 files in one flat package main"),
  `README.md`, `.ai-factory/rules/base.md` ("Module Structure" describes the
  pre-move tree), and the `docs/` pages whose subject moved. Run `/aif-docs` and
  treat its findings as part of this commit.

- [ ] **T38 — Drop the migration baseline.**
  `.ai-factory/RESTRUCTURE_BASELINE.md` has served its purpose once the last wave is
  green; remove it in the final commit, or keep it and say in the PR body why it is
  worth keeping. Do not leave the decision implicit.

---

## Risks

- **The extraction gate is the whole plan.** Outbound-edge ranking is a heuristic
  for the order; the actual gate is "does it compile". Expect one or two waves to
  reveal a symbol that has to be pulled forward, exactly as `clipboard*.go` and
  `background_jobs.go` were pulled into T27 to avoid a `fileops ↔ term` cycle. Pull
  it forward into an earlier commit; never add an import that points upward.
- **Exporting is where behaviour hides.** Step 4 of every wave renames identifiers
  across the module. It is the one part of a "move commit" that a reviewer cannot
  verify by reading it as a rename, so keep it mechanical and let the compiler and
  the T01 baseline be the proof.
- **`tools/` is invisible to CI.** No workflow tests any of the four `tools/`
  modules, which is how `tools/icons`' failure has survived. Anything this plan
  changes under `tools/` must be run by hand (T01, T24).
- **The catalogue URL is a published contract** (T13) and the README image URL is
  branch-scoped (T09). Both are intentional, both belong in the PR body.

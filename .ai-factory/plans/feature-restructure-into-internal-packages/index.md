<!-- aif:plan-mode:ultra -->
# Ultra Implementation Plan: Restructure f4 into internal packages

Mode: ultra
Branch: feature/restructure-into-internal-packages
Created: 2026-09-07
Base revision: `0cda22a7`, level with `upstream/main` at `ef3640c7`
(`git rev-list --left-right --count upstream/main...HEAD` reports `0` on the
left). Every count in this bundle is verified against `0cda22a7`.

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
>
> Следующий шаг: пересобрать его в ultra-бандл, /aif-plan ultra. Ultra здесь не
> «спланировать заново», а «раскрыть каждую фазу до уровня, на котором её выполняет
> модель послабее без догадок».

## Settings

- Testing: yes — the `cmd/f4/*_test.go` files (346 at the base revision, 343 once
  Phase 1 has moved five helper-only files out and added three tests) move with
  their subjects; a
  module boundary auditor is added in Task 8; `internal/numeric` gets the one new
  test suite in the plan.
- Logging: project convention, nothing added. Diagnostics stay on `VTUI_DEBUG`
  (`cmd/f4/debug_log.go`); user-facing failures go to stderr with the `f4: `
  prefix. A move commit that introduces a log line is not a move commit.
- Docs: yes — mandatory `/aif-docs` checkpoint (Task 40), plus the per-commit
  reference sweep required by `ARCHITECTURE.md`, plus a dedicated
  `ARCHITECTURE.md` rewrite (Task 41).

Delivery: **one pull request to `unxed/f4`** with meaningful commits inside. The
harness scaffolding (`.ai-factory/`, `.claude/`, `.mcp.json`, `AGENTS.md`) ships
in that PR deliberately, as part of the proposal.

## Architecture and Decisions

Cross-phase decisions an implementer must not re-open. Everything here was
re-derived from the CodeGraph index against the working tree; where it disagrees
with `ARCHITECTURE.md` or with the reconnaissance input, the graph won and the
disagreement is stated.

**Package names chosen during planning.** `ARCHITECTURE.md` says the shared
primitives leave first but does not name their package. These four are the plan's
own choice and the one place the user may want to overrule with a single reply:

| Package | Holds | Why this name |
|---|---|---|
| `internal/action` | `Action`, `RegisterAction`, registry order and lookup | Exact and unambiguous |
| `internal/toast` | `toast.Show` and its test-duration seam | One subject, one file; a package of one file is normal in Go |
| `internal/history` | `history_provider.go`, `history_dialog.go`, `command_history_paths.go`, `search_history.go`, `menu_history.go`, plus four far2l helpers from `actions.go` | A real cluster, verified free of `Msg`, `AppConfig`, `showToast` and every view type |
| `internal/numeric` | seven `bounded*`, `NonNegativeUint64`, `RuneCodepoint`, `ReleaseHeavyMemory` | Justified by count: **seven** consuming packages after `internal/sysinfo` takes its private copy — plughost, term, keymap, macro, viewer, panel, editor |

An earlier draft proposed one `internal/vtuix` for toast plus the histories. It
was split on review: the name rests on no established abbreviation the way
`internal/ttyx` does, and it would have held two unrelated subjects.

**The primitives are symbols, not files.** `ARCHITECTURE.md` names
`actions.go` and the action registry among the files that leave first. Measured:

- `actions.go` holds 81 functions — 80 free plus one `*PanelsFrame` method at
  `:2194`. **61 reference a view type; 52 take `*PanelsFrame` in their
  signature.** Of the 20 that do not, only four are layer-0. The rest scatter to
  `internal/dialog` (8), `internal/i18n` (3), `internal/viewer` (3),
  `internal/editor` (1) and the external-editor path (1).
- `action_registry.go` splits: the `Action` type is clean — `Checked`, `Visible`
  and `Handler` are `func() bool` — but its 2554-line `init()`
  (`:264-2817`, 173 `RegisterAction` calls) mentions `PanelsFrame` 114 times and
  `EditorView` 47 times. Locate it as `grep -n '^func init()'`, never by the line
  number: it has already moved once, ten lines down, when Task 3 documented the
  ordering above it. Mechanism is layer 0; the table is layer 4.
- `framework_actions.go` is **not** a primitive at all. Of its 25 functions, 18
  have no external callers — they are `Handler:` values reached from the table,
  plus `main.go:actionScreenDump`. It travels whole with `internal/app`.

Moving the named *files* instead of the named *symbols* produces an uncompilable
commit. Phase 4 carries the full table.

**Five more corrections that change the plan.**

1. Three `F4Config` field types are declared outside `config.go`, not two:
   `PanelNavigationMode` (`navigation_mode.go:7`), `compareOptions`
   (`compare_folders.go:61`) and **`StartupMode` (`startup_backend.go:11`)** —
   the third in composition-root code that leaves last, which would make layer 0
   depend on layer 4. Task 4.
2. `misc.go`'s split is inverted relative to the document: the numeric helpers
   have the widest fan-out; `ScreenRow` has five callers and all are `_test.go`.
   Tasks 9 and 19.
3. `internal/sysinfo` **cannot** import the extracted numeric helpers — the rule
   says it imports no other `internal/*`, so the extraction would create the
   forbidden edge rather than remove it. It keeps a private five-line copy for its
   two call sites at `cpu_info_darwin.go:39` and `:47`. Task 19.
4. The shared test harness cannot be one package. `setupMockPanelsFrame` calls
   `NewTerminalView`, `NewCommandLine`, `NewFileSystemPanel` and
   `PanelsFrame.initPTY`, so a package holding it imports three layer-3 packages —
   and their own in-package tests then cannot import it back. Split into
   `internal/testutil` (vtui glue, caller-supplied drains) and
   `internal/paneltest`, with the affected tests becoming `package X_test`.
   Tasks 9 and 34.
5. Two more barriers the reconnaissance did not name: the drive registry
   (`DriveEntry`, `DriveRegistry`, `RegisterDrive`) is parked on
   `panels_frame.go:26-40` although it needs only `sync` and `vfs`, and
   `gpu_info_linux.go:113` is the single `Msg` call inside the sysinfo family.
   Both must go before the first wave. Tasks 6 and 7.

**CI does not need per-commit resharding, but it does hide one silent failure.**
The lint shards split `./cmd/f4/...` against everything else and the race
`packages` scope is `go list ./... | grep -Ev '…/cmd/f4$'` (`build.yml:1332`), so
files migrate between shards on their own and both halves stay correct — only
progressively imbalanced. Rebalance once, in Task 38. The trap is elsewhere:
`build.yml:1187` skips `TestAllDialogs_LayoutValidation` globally and re-runs it
single-threaded only for `./...` or `cmd/f4` (`:1193-1194`). When
`dialog_layouts_test.go` moves in Task 25, the test is skipped everywhere and
re-run nowhere — a green build with a missing test.

**The extraction gate.** A package leaves `cmd/f4` only when every symbol it calls
already lives in an extracted package, in itself, or outside the module. The
mechanical form is per **(file, destination)**: count references to types whose
own package is extracted *later*, and resolve each non-zero before moving.
Measured over all 345 non-test files at the base revision: **235 score zero on
every type** and are a pure `git mv`; 58 score 1-3; 52 score 4 or more. Phase 1
leaves 346: `navigation_mode.go` is deleted, `drive_registry.go` and
`action_order.go` are created, and all three score zero, so the zero bucket
becomes 236.

**The eight-type grep is necessary, not sufficient.** It finds view types only. A
file also may not reference any *other* symbol still in `cmd/f4`, and the gate is
silent about those. `plugin_permissions_ui.go` scores `0` on all eight and still
cannot move to `internal/dialog`, because its entry point takes a
`*PermissionStore` that lives in `plugin_permissions.go` and leaves with
`internal/plughost` one task later. The `codegraph callees` half of wave-procedure
step 1 is the binding check; a zero score licenses nothing on its own.

**Tests travel by subject, not by filename.** 154 of the 346 `_test.go` files have
no same-named source — they are named for the scenario they exercise. "Take the
`_test.go` neighbour" therefore strands 45% of the suite. Task 43 holds the
assignment, measured on the CodeGraph index: a table of all 154, a roster of all
346 per wave, the 61 tests whose unexported references span more than one
package (each with a host and an export list), the eight tests that read
`lang/`, `help/` or `styles/` from disk, and the scaffolding — 48 shared helpers
and `TestMain` — the waves would otherwise strand.

**Ground rules for every commit.**

- **Builds on the whole matrix — 26 targets, exotic ones included, `CGO_ENABLED=0`.**
  Verified at two levels, because the two cost differently:
  - *After every commit, locally.* `CGO_ENABLED=0 GOOS=… GOARCH=… go build ./...`
    across the tag-sensitive targets — windows, linux, darwin, solaris, illumos,
    freebsd, plus one exotic arch such as linux/mips. Seconds per target, and it
    catches exactly the mistake this plan is most likely to make: a file selected
    by its name instead of its `//go:build` line.
    **freebsd and netbsd need `-gcflags=github.com/go-webgpu/goffi/internal/fakecgo=-std`**
    (the flag the matrix itself passes, `build.yml:880`). Without it the build
    fails on `//go:cgo_export_dynamic … only allowed in cgo-generated code`, which
    looks exactly like a breakage we caused and is not one. Verified on the
    current tree: all six sampled targets build clean, freebsd only with the flag.
  - *After every phase, in CI.* `gh workflow run build.yml --ref <branch>`.
    It must be `workflow_dispatch`, **not** a pull request: `build-batch`, which
    holds every exotic target, is gated on
    `github.event_name != 'pull_request'` (`build.yml:324`), and so is the
    cross-libc smoke test (`build.yml:231`). A PR therefore builds only the six
    desktop cells and would report green while the targets most at risk went
    unbuilt. Note also that a commit touching only `.md` and `docs/` skips CI
    entirely on a PR (`paths-ignore`), which is why the documentation phases
    cannot be checked this way at all.
  - *Not after every commit in CI.* One run is ~30 jobs against 20 free-tier
    runners, and `concurrency` cancels the in-flight run on the same ref
    (`build.yml:28-30`), so consecutive pushes would queue up and kill each
    other. Eleven phase runs give the same coverage as twenty-seven commit runs.
- **Lint what you touched, before you commit.**
  `golangci-lint run --new-from-rev=origin/main <the packages you changed>`, at the
  version CI pins (v2.13.1). Incremental mode says nothing about existing code —
  roughly 2450 findings of backlog sit behind it — but every line the diff calls
  new is checked, and that has two consequences. An edit inside a file is checked
  at once. And a file that *moves* changes its `package` line, so if git does not
  detect the rename the whole file counts as new and empties its share of the
  backlog into the report; Task 39 is written for exactly that.

  Note what "new" means here: the base is `origin/main`, the fork's own main, not
  `upstream/main`. Whatever the fork is behind by is reported as yours. Task 39
  levels them before it measures anything.

- **Move by `//go:build` line, never by filename.** `pty_unix.go` is
  `//go:build linux`; `solaris_pty.go` is `//go:build !windows` and holds no PTY
  code. 97 non-test files carry a tag across 28 distinct expressions.
- No rewrites inside a move commit. A reviewer must read the diff as a rename.
- Tests move with their subject in the same commit; a test without a subject
  moves with the wave Task 43's roster names.
- Compare against `.ai-factory/RESTRUCTURE_BASELINE.md`; **never rewrite it.** It
  is an immutable snapshot of the Task 0 revision, and only Task 42 touches it.
  Run all six modules — `go test ./...` sees only the main module's 38 packages.
- Each commit closes its own references in `docs/`, `README.md`, `AGENTS.md` and
  `.ai-factory/rules/base.md`, and fixes the CI lines it breaks.
- `git mv` for anything git already tracks; the executable bit and build tags must
  survive.
- **Track upstream continuously; merge it on a trigger, not on a schedule.**
  Upstream is active — 29 commits landed on this branch's base in a day — so a
  single sync at the start is not a plan, it is a deferral.

  *Check after every commit.* It costs a second and changes nothing:
  ```
  git fetch upstream --quiet
  git rev-list --count HEAD..upstream/main
  git diff --name-only HEAD...upstream/main | grep '^cmd/f4/'
  ```

  *Rebase on either of two triggers:* a **phase boundary**, with every task in
  the phase closed and the tree consistent; or **upstream touching a file the
  next two or three tasks own**, which overrides the schedule. Merging a change
  into `cmd/f4/actions.go` while it is still one file is an ordinary three-way
  merge. Merging the same change once the file has been cut into six pieces
  across four packages is a hand reconstruction of somebody else's intent.

  *Never mid-task*, and always behind a backup branch named with the date and
  time — that is the user's standing rule for any rebase. Keep the two most
  recent and delete the rest; twenty identically named branches are worth
  nothing.

  *After each rebase:* compare against the baseline, run the cross-compilation
  sweep, and **re-measure every number the next tasks stand on**. This is not
  ceremony: the first mid-work rebase moved `action_registry.go`'s `init()` by
  ten lines, took `PanelsFrame` mentions inside it from 109 to 114, and added a
  106th method to `FileSystemPanel` — all of them quoted in task text.

  *The cost of waiting grows.* Early phases merge cheaply because the files are
  where upstream expects them. From the extraction waves onward every deferred
  merge is one more foreign change landing on a file that has moved, been
  renamed and changed its `package` clause. Later phases need this more often,
  not less, which is the opposite of how it feels.

**Open questions:** none blocking. The four package names above are the only
planning decision the user may wish to overrule, and doing so changes four task
titles, not the ordering.

## Phase Index

1. [Phase 1: Upstream Sync, Baseline and Barrier Removal](phase-01-baseline-and-barriers.md) — Tasks 0-9 and 43
2. [Phase 2: Clear the Repository Root](phase-02-repository-root.md) — Tasks 10-15
3. [Phase 3: Self-Contained Subsystems Under internal/](phase-03-subsystems.md) — Tasks 16-17
4. [Phase 4: The Shared Primitives Leave cmd/f4](phase-04-shared-primitives.md) — Tasks 18-21
5. [Phase 5: Leaf Packages](phase-05-leaf-packages.md) — Tasks 22-24
6. [Phase 6: Hosts and Services](phase-06-hosts-and-services.md) — Tasks 25-28
7. [Phase 7: Viewer, Terminal and Media](phase-07-view-and-terminal.md) — Tasks 29-31
8. [Phase 8: File Operations and the Editor](phase-08-fileops-and-editor.md) — Tasks 32-33
9. [Phase 9: Panels and the Command Line](phase-09-panel-and-cmdline.md) — Tasks 34-35
10. [Phase 10: The Composition Root](phase-10-composition-root.md) — Tasks 36-37
11. [Phase 11: CI, Lint and Documentation](phase-11-ci-and-docs.md) — Tasks 38-42

## Cross-Phase Dependencies

- **Task 1 depends on Task 0** — a baseline is valid only for the revision it was
  taken at, so the rebase must immediately precede it. Run them as one sitting.
- **Task 18 depends on Task 3** — splitting the registry across a package boundary
  reorders the menu unless the order is already explicit and golden-tested.
- **Task 21 depends on Task 18** — the mechanism must be separated from the table
  before it can move.
- **Task 22 depends on Tasks 6, 7, 19, 20, 21 and 43** — the drive registry must be
  off the panel type, the `Msg` call out of `gpu_info_linux.go`, and the private
  numeric copy in place, or `internal/sysinfo` is not a leaf and cannot go first.
  Task 20 is what makes `toast.Show` callable from a package: `showToast` has 16
  call sites spanning eight future packages, so every wave from here on needs it.
  Task 43 is what tells this wave which files it owns.
- **Task 24 depends on Task 4** — `internal/config` cannot be a leaf while
  `StartupMode` lives in composition-root code.
- **Tasks 22-35 depend on Tasks 19-21** — 37 call edges run into what would
  otherwise be `internal/app` from lower layers. Extracting in the other order
  makes every intermediate commit uncompilable.
- **Task 31 depends on Task 30** — six of media's ten outbound edges point at
  `internal/term`, which is why term precedes media despite the higher count.
- **Task 32 depends on Tasks 5 and 30** — `clipboard.go`, `clipboard_async.go` and
  `background_jobs.go` are pulled into the term wave specifically to prevent a
  `fileops ↔ term` cycle here; and `queue_manager.go` only becomes movable once
  Task 5 has taken the goroutine launch out of its `init()`.
- **Task 33 depends on Tasks 14 and 29** — the `editor ↔ viewer` cycle is removed
  by the viewer wave taking `top_bar.go`, `file_title.go` and `url_links.go`; and
  `colorer_plugin.go` reads `colorer.RadiolaHRD`, which does not exist until
  Task 14 creates `internal/colorer`.
- **Task 34 depends on Tasks 26, 29, 30, 32 and 33** — `panel_plugins.go`'s
  `coreAPI` method is cut out in Task 26, and `internal/panel` imports viewer,
  term, fileops and editor.
- **Task 35 depends on Task 34** — `internal/cmdline` may import
  `internal/panel`; the reverse edge is forbidden and Task 34 step 4 removes it.
- **Task 36 depends on every wave** — the composition root is what is left.
- **Task 41 depends on Task 37** — `ARCHITECTURE.md` can describe the tree as a
  fact only once the tree is the tree.

## Tasks

### Phase 1: Upstream Sync, Baseline and Barrier Removal
- [x] Task 0: Rebase onto `upstream/main` behind a backup branch, re-verify the plan's counts ([details](phase-01-baseline-and-barriers.md#task-0-synchronize-with-upstreammain))
- [x] Task 1: Record the immutable pre-restructuring baseline across all six modules ([details](phase-01-baseline-and-barriers.md#task-1-record-the-pre-restructuring-test-baseline)) (depends on 0)
- [x] Task 2: Re-key the command-palette auditor's 42 entries to qualified symbols ([details](phase-01-baseline-and-barriers.md#task-2-re-key-the-command-palette-auditor-to-qualified-symbols))
- [x] Task 3: Make action registration order explicit and golden-tested ([details](phase-01-baseline-and-barriers.md#task-3-make-action-registration-order-explicit))
- [x] Task 4: Move `F4Config`'s three stray field types into `config.go` ([details](phase-01-baseline-and-barriers.md#task-4-move-f4configs-field-types-into-configgo))
- [x] Task 5: Stop `queue_manager.go` starting a goroutine from `init()` ([details](phase-01-baseline-and-barriers.md#task-5-stop-starting-a-goroutine-from-init))
- [x] Task 6: Lift the drive registry out of `panels_frame.go` ([details](phase-01-baseline-and-barriers.md#task-6-lift-the-drive-registry-out-of-panels_framego))
- [x] Task 7: Remove sysinfo's last localization call (`gpu_info_linux.go:113`) ([details](phase-01-baseline-and-barriers.md#task-7-remove-sysinfos-last-localization-call))
- [x] Task 8: Add the module boundary auditor `cmd/f4/architecture_test.go` ([details](phase-01-baseline-and-barriers.md#task-8-add-the-module-boundary-auditor))
- [x] Task 9: Split the shared frame harness into `internal/testutil` + `internal/paneltest` ([details](phase-01-baseline-and-barriers.md#task-9-give-the-shared-frame-harness-a-home))
- [x] Task 43: Assign every `cmd/f4` file to a wave — 23 stray sources, 154 subject-less tests, 61 multi-package tests, 48 shared helpers ([details](phase-01-baseline-and-barriers.md#task-43-assign-every-cmdf4-file-to-a-wave)) (depends on 8)

### Phase 2: Clear the Repository Root
- [x] Task 10: Move the three shell scripts to `scripts/` ([details](phase-02-repository-root.md#task-10-move-the-shell-scripts-to-scripts)) (depends on 1)
- [x] Task 11: Move `screenshot.png` to `.github/assets/` ([details](phase-02-repository-root.md#task-11-move-screenshotpng-to-githubassets))
- [x] Task 12: Move the loose prose into `docs/` and delete `time.txt` ([details](phase-02-repository-root.md#task-12-move-the-loose-prose-into-docs-and-delete-timetxt))
- [x] Task 13: Rename the 40 issue reviews to `ISSUE_<number>_<SLUG>.md` ([details](phase-02-repository-root.md#task-13-rename-the-issue-reviews-to-issue_number_slugmd)) (depends on 12)
- [ ] Task 14: Move `colorer/` to `internal/colorer/` with its own embed ([details](phase-02-repository-root.md#task-14-move-colorer-to-internalcolorer))
- [ ] Task 15: Move `plugring/` to `plugins/plugring/` and fix its three URLs ([details](phase-02-repository-root.md#task-15-move-plugring-to-pluginsplugring))

### Phase 3: Self-Contained Subsystems Under internal/
- [ ] Task 16: Move `piecetable`, `textlayout` and `sheet` under `internal/` ([details](phase-03-subsystems.md#task-16-move-piecetable-textlayout-and-sheet)) (depends on 15)
- [ ] Task 17: Move `fusefs`, `vtvibe` and `luaplug` under `internal/` ([details](phase-03-subsystems.md#task-17-move-fusefs-vtvibe-and-luaplug))

### Phase 4: The Shared Primitives Leave cmd/f4
- [ ] Task 18: Split `action_registry.go` into mechanism and table, in place ([details](phase-04-shared-primitives.md#task-18-separate-the-action-registrys-mechanism-from-its-table)) (depends on 3, 17)
- [ ] Task 19: Create `internal/numeric`; give sysinfo its private copy ([details](phase-04-shared-primitives.md#task-19-create-internalnumeric)) (depends on 9)
- [ ] Task 20: Create `internal/toast` and `internal/history` ([details](phase-04-shared-primitives.md#task-20-create-internaltoast-and-internalhistory))
- [ ] Task 21: Create `internal/action` with a localizer hook ([details](phase-04-shared-primitives.md#task-21-create-internalaction)) (depends on 18)

### Phase 5: Leaf Packages
- [ ] Task 22: Extract `internal/sysinfo` (1 outbound) ([details](phase-05-leaf-packages.md#task-22-extract-internalsysinfo)) (depends on 6, 7, 19, 20, 21, 43)
- [ ] Task 23: Extract `internal/update` (3 outbound) ([details](phase-05-leaf-packages.md#task-23-extract-internalupdate)) (depends on 22)
- [ ] Task 24: Extract `internal/config`, `internal/i18n`, `internal/theme`, `internal/keymap` ([details](phase-05-leaf-packages.md#task-24-extract-internalconfig-internali18n-internaltheme-internalkeymap)) (depends on 4, 23)

### Phase 6: Hosts and Services
- [ ] Task 25: Extract `internal/dialog`, and fix the silent dialog-test drop ([details](phase-06-hosts-and-services.md#task-25-extract-internaldialog)) (depends on 24)
- [ ] Task 26: Extract `internal/plughost`; cut `panel_plugins.go`'s `coreAPI` method ([details](phase-06-hosts-and-services.md#task-26-extract-internalplughost)) (depends on 25)
- [ ] Task 27: Extract `internal/gui`; move two of three `tools/icons` paths ([details](phase-06-hosts-and-services.md#task-27-extract-internalgui)) (depends on 26)
- [ ] Task 28: Extract `internal/macro` ([details](phase-06-hosts-and-services.md#task-28-extract-internalmacro)) (depends on 27)

### Phase 7: Viewer, Terminal and Media
- [ ] Task 29: Extract `internal/viewer`, removing the `editor ↔ viewer` cycle ([details](phase-07-view-and-terminal.md#task-29-extract-internalviewer)) (depends on 28)
- [ ] Task 30: Extract `internal/term`, including eleven misfiled files ([details](phase-07-view-and-terminal.md#task-30-extract-internalterm)) (depends on 29)
- [ ] Task 31: Extract `internal/media` ([details](phase-07-view-and-terminal.md#task-31-extract-internalmedia)) (depends on 30)

### Phase 8: File Operations and the Editor
- [ ] Task 32: Extract `internal/fileops` ([details](phase-08-fileops-and-editor.md#task-32-extract-internalfileops)) (depends on 5, 30, 31)
- [ ] Task 33: Extract `internal/editor` ([details](phase-08-fileops-and-editor.md#task-33-extract-internaleditor)) (depends on 14, 29, 32)

### Phase 9: Panels and the Command Line
- [ ] Task 34: Extract `internal/panel`, finish `semantic.go`, fill `internal/paneltest` ([details](phase-09-panel-and-cmdline.md#task-34-extract-internalpanel)) (depends on 26, 33)
- [ ] Task 35: Extract `internal/cmdline`; delete `semantic.go` ([details](phase-09-panel-and-cmdline.md#task-35-extract-internalcmdline)) (depends on 34)

### Phase 10: The Composition Root
- [ ] Task 36: Extract `internal/app` ([details](phase-10-composition-root.md#task-36-extract-internalapp)) (depends on 35)
- [ ] Task 37: Reduce `cmd/f4` to the entry point ([details](phase-10-composition-root.md#task-37-reduce-cmdf4-to-the-entry-point)) (depends on 36)

### Phase 11: CI, Lint and Documentation
- [ ] Task 38: Rebalance the CI shards; measure before and after ([details](phase-11-ci-and-docs.md#task-38-rebalance-the-ci-shards)) (depends on 37)
- [ ] Task 39: Run the incremental lint, and verify every commit builds, before opening the PR ([details](phase-11-ci-and-docs.md#task-39-run-the-incremental-lint-against-originmain-before-opening-the-pr)) (depends on 38)
- [ ] Task 40: `/aif-docs` checkpoint; rewrite `AGENTS.md` and `rules/base.md` ([details](phase-11-ci-and-docs.md#task-40-aif-docs-checkpoint)) (depends on 37)
- [ ] Task 41: Rewrite `ARCHITECTURE.md` from target to fact ([details](phase-11-ci-and-docs.md#task-41-rewrite-architecturemd-from-target-to-fact)) (depends on 40)
- [ ] Task 42: Drop the migration baseline ([details](phase-11-ci-and-docs.md#task-42-drop-the-migration-baseline)) (depends on 41)
- [ ] Task 44: Review the finished tree before calling it done ([details](phase-11-ci-and-docs.md#task-44-review-the-finished-tree-before-calling-it-done)) (depends on 42)

## Commit Plan

Thirty-nine commits. Task 0 is a rebase, Task 39 a measurement, and Task 43 a
classification recorded in this bundle rather than in the tree; none produces one.

| # | Tasks | Message |
|---|---|---|
| 1 | 1 | `test: record the pre-restructuring test baseline` |
| 2 | 2 | `test(palette): key the coverage auditor by qualified symbol` |
| 3 | 3 | `refactor(actions): make registration order explicit` |
| 4 | 4 | `refactor(config): give F4Config's field types a home in config.go` |
| 5 | 5 | `refactor(queue): start the worker from the root, not from init()` |
| 6 | 6 | `refactor(drives): lift the drive registry off the panels frame` |
| 7 | 7 | `refactor(sysinfo): return a message key from the GPU probe` |
| 8 | 8 | `test(arch): add the module boundary auditor` |
| 9 | 9 | `test: give the shared frame harness a home` |
| 10 | 10-12 | `chore: move scripts, media and prose out of the repository root` |
| 11 | 13 | `docs: name the issue reviews by their subject` |
| 12 | 14 | `refactor(colorer): move the colour scheme to its consumer` |
| 13 | 15 | `chore(plugring): move the catalogue beside the plugins` |
| 14 | 16 | `refactor: move piecetable, textlayout and sheet under internal/` |
| 15 | 17 | `refactor: move fusefs, vtvibe and luaplug under internal/` |
| 16 | 18 | `refactor(actions): split the registry mechanism from its table` |
| 17 | 19 | `refactor: extract internal/numeric` |
| 18 | 20 | `refactor: extract internal/toast and internal/history` |
| 19 | 21 | `refactor: extract internal/action` |
| 20-33 | 22-35 | one per wave: `refactor(<pkg>): extract internal/<pkg> from cmd/f4` |
| 34 | 36 | `refactor(app): extract the composition root` |
| 35 | 37 | `refactor: reduce cmd/f4 to the entry point` |
| 36 | 38 | `ci: rebalance the shards for the split tree` |
| 37 | 40 | `docs: update the structural map and the subsystem pages` |
| 38 | 41 | `docs(ai-factory): describe the architecture as built` |
| 39 | 42 | `chore: drop the migration baseline` |

## Definition of Done

- `cmd/f4` holds `main.go`, the four module-wide auditors
  (`command_palette_coverage_test.go`, `architecture_test.go`,
  `frame_manager_capture_test.go`, `hardcoded_strings_test.go`), the five-line
  `TestMain` they need, and `rsrc_windows_{amd64,arm64}.syso`. Nothing else: no
  test of the wiring exists today, and one written later lives here too.
- Every package in `ARCHITECTURE.md`'s target tree exists, and every package that
  exists is in the document.
- `go test ./cmd/f4 -run '^TestArchitecture'` passes with all five rules active
  and **no exemptions** — including rule 3, that nothing below layer 4 imports
  `internal/app`, and the sysinfo leaf rule.
- `command_palette_coverage_test.go` passes with 42 keys, none containing a path.
- The full 26-target build matrix is green, `CGO_ENABLED=0`, exotic targets and
  the `noffi` tag included.
- `go test -timeout 25m ./...` plus the five other modules match the Task 1
  baseline's green set. The three pre-existing failures — `tools/icons`,
  `tools/wine_syscall_probe` on darwin/arm64, the inert
  `plugring_policy_test.go` — are unchanged and recorded somewhere that outlives
  this branch.
- No document describes the migration. `ARCHITECTURE.md` describes the tree;
  `AGENTS.md` and `.ai-factory/rules/base.md` describe the packages that exist,
  with counts re-derived rather than estimated.
- The pull-request body states, in the author's own words rather than buried in a
  diff: the README image URL being branch-scoped and 404 until merge; the
  incremental-lint finding count measured against `origin/main`; and the three
  pre-existing failures.
- It also states the PlugRing move **as a question, not as a decision**: that the
  catalogue moved to `plugins/plugring/`, that this changes a published URL and
  older builds will stop finding it after the merge, and that the move is offered
  rather than argued — if the maintainer would rather keep the catalogue where it
  is, say so and it goes back. Everything else in this branch is internal; this is
  the one change users outside the repository can notice, so it does not get
  decided in a diff.

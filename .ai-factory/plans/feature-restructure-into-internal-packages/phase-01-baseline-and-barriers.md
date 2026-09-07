# Phase 1: Upstream Sync, Baseline and Barrier Removal

Plan: [index.md](index.md)
Tasks: 0-9 and 43
Depends on: none

## Objective

The branch sits on the newest `upstream/main`, the tree still looks exactly as it
does today — no file has moved — the four structural barriers that would make an
intermediate commit uncompilable are gone, and "green" has a written definition to
compare against.

Every task in this phase is a behaviour-preserving change to `cmd/f4` in place.
None of them creates a package, and none of them may be folded into a later move
commit: a move commit must read as a rename.

## Current-Code Evidence

| Path | Symbols / lines | Why it matters |
|---|---|---|
| `cmd/f4/action_registry.go:24` | `type Action` | `Checked`/`Visible`/`Handler` are `func() bool` — the type is layer-0 clean |
| `cmd/f4/action_registry.go:121` | `RegisterAction` | 174 calls from `init()` here, more from six other files |
| `cmd/f4/action_registry.go:254-2807` | `func init()` | 2553 lines; mentions `PanelsFrame` 109×, `EditorView` 48× |
| `cmd/f4/config.go:352` | `var AppConfig = F4Config{…}` | 132 fields, read from 133 files (55 non-test) |
| `cmd/f4/navigation_mode.go:7` | `type PanelNavigationMode` | config field type declared outside `config.go` |
| `cmd/f4/compare_folders.go:61` | `type compareOptions` | same |
| `cmd/f4/startup_backend.go:11` | `type StartupMode` | same, and in composition-root code that leaves last |
| `cmd/f4/queue_manager.go:312-318` | `func init()` | constructs `GlobalQueueManager` and starts `workerLoop` on import |
| `cmd/f4/panels_frame.go:26-40` | `DriveEntry`, `DriveRegistry`, `RegisterDrive` | drive registry parked on the panel type; needs only `vfs` + `sync` |
| `cmd/f4/gpu_info_linux.go:113` | `Msg("InfoPanel.GPUWSLVirt")` | the one localization call inside the sysinfo family |
| `cmd/f4/command_palette_coverage_test.go` | 42 audit keys | keyed by `cmd/f4/<file>.go:(*Type).Method` |
| `cmd/f4/frame_manager_test_helpers_test.go:129` | `swapFrameManager` | 63 test files; calls `waitForAsyncClipboard`, `waitForDirectoryLoads` |
| `cmd/f4/panels_frame_test.go:794` | `setupMockPanelsFrame` | 28 test files; constructs `PanelsFrame`, `TerminalView`, `CommandLine`, `FileSystemPanel` |
| `cmd/f4/misc.go:12` | `ScreenRow` | five callers, all `_test.go` |
| `cmd/f4/frame_manager_test_helpers_test.go:52` | `appendFrameManagerScreenForTest` | 2 test files; moves with the harness |
| `cmd/f4/frame_manager_test_helpers_test.go:66` | `taskPumpGoroutineProfile` | 2 test files; the harness's own test is built on it |

## Files to Change

| Path | Action | Required change |
|---|---|---|
| backup branch | create | `backup/restructure-into-internal-packages-YYYY-MM-DD` before the rebase |
| `index.md` | modify | Base revision replaced with the post-rebase HEAD |
| `.ai-factory/RESTRUCTURE_BASELINE.md` | create | Written definition of "green before we started", per module |
| `cmd/f4/command_palette_coverage_test.go` | modify | Re-key 42 audit entries to qualified symbols |
| `cmd/f4/action_registry.go` | modify | Explicit registration ordinal + deterministic sort |
| `cmd/f4/action_registry_order_test.go` | create | Golden test over `actionOrder` |
| `cmd/f4/config.go` | modify | Receives three type declarations |
| `cmd/f4/navigation_mode.go` | modify | Loses `PanelNavigationMode` and its constants |
| `cmd/f4/compare_folders.go` | modify | Loses `compareOptions` |
| `cmd/f4/startup_backend.go` | modify | Loses `StartupMode` and its constants |
| `cmd/f4/queue_manager.go` | modify | `init()` no longer starts a goroutine |
| `cmd/f4/main.go` | modify | Starts the queue worker explicitly |
| `cmd/f4/panels_frame.go` | modify | Loses the drive registry |
| `cmd/f4/drive_registry.go` | create | `DriveEntry`, `DriveRegistry`, `RegisterDrive` |
| `cmd/f4/gpu_info_linux.go` | modify | Returns a message key, not a localized string |
| `cmd/f4/info_panel.go` | modify | Localizes the GPU model at render time |
| `cmd/f4/architecture_test.go` | create | Module boundary auditor |
| `internal/testutil/*.go` | create | Frame-manager harness with caller-supplied drains |
| `internal/paneltest/doc.go` | create | Empty package with its contract documented; filled in Task 34 |
| `cmd/f4/frame_manager_test_helpers_test.go` | modify | Harness moves out; local drains remain |
| 63 + 28 `_test.go` call sites | modify | New harness call shape |

---

## Task 0: Synchronize with `upstream/main`

### Intent

While files still sit in `cmd/f4`, an upstream commit merges into them
automatically. Once two hundred of them have moved to `internal/*` with a changed
`package` clause, git's rename detection starts missing, and the same upstream
commit becomes a manual conflict resolution inside someone else's change — one the
implementer did not write and does not understand.

This is not hypothetical. `c31f9f50` ("test: isolate terminal busy drive menu
frame") touches `cmd/f4/panels_frame_test.go` — the file that holds
`setupMockPanelsFrame` at line 794, which Task 9 refactors and Task 34 moves.
Upstream gained a commit in a first-wave file in the course of one day.

**One sync has already been performed**, onto `upstream/main` = `c31f9f50`; the
branch is `83177611` and `git rev-list --left-right --count upstream/main...HEAD`
currently reports `0 14`. That does not retire this task: time passes between
planning and execution, and the window has to be re-measured rather than assumed.
Run the steps below and expect them to be cheap, not to be no-ops.

Task 0 and Task 1 run as one sitting: a baseline is only meaningful for the
revision it was taken at, and rebasing after recording one invalidates it.

### Implementation Steps

1. `git fetch upstream`
2. **Create the backup branch before rebasing** — this is a standing project
   requirement for any rebase, not a suggestion:
   ```
   git branch backup/restructure-into-internal-packages-YYYY-MM-DD
   ```
3. `git rebase upstream/main`
4. Resolve any conflict. At this point every file is still in `cmd/f4`, so a
   conflict is a normal textual one in a file both sides edited — which is exactly
   why this happens now and not later.
5. Re-verify the plan's load-bearing numbers against the new revision. Upstream may
   have moved them, and a shift here changes Tasks 16-21:
   ```
   ls cmd/f4/*.go | grep -v '_test\.go$' | wc -l            # expect 345
   ls cmd/f4/*_test.go | wc -l                              # expect 346
   grep -c '^func ' cmd/f4/actions.go                       # expect 81
   # Receiver-anchored. The looser '^func (.*\*T)' form counts methods of OTHER
   # types that merely take a *T parameter and inflates every one of these.
   grep -lE '^func \([a-z]+ \*PanelsFrame\)' $(ls cmd/f4/*.go|grep -v _test) | wc -l      # expect 16
   grep -lE '^func \([a-z]+ \*EditorView\)' $(ls cmd/f4/*.go|grep -v _test) | wc -l       # expect 15
   grep -lE '^func \([a-z]+ \*FileSystemPanel\)' $(ls cmd/f4/*.go|grep -v _test) | wc -l  # expect 6
   grep -lE '^func \([a-z]+ \*TerminalView\)' $(ls cmd/f4/*.go|grep -v _test) | wc -l     # expect 5
   awk 'NR>=254 && /^}/ {print NR; exit}' cmd/f4/action_registry.go  # expect 2807
   ```
   Also check whether upstream touched any file named in Phases 1-4:
   ```
   git diff --name-only <old-base>..upstream/main -- cmd/f4/
   ```
6. Where a number moved, update the affected task in this bundle **before**
   implementing it. A stale count in Task 16's split instruction produces an
   uncompilable commit.
7. Replace the base revision in `index.md`'s header with the new HEAD.
8. **Then do not touch upstream again until the pull request.** If the work runs
   long enough that a sync is genuinely needed, do it **between waves**, on a green
   commit with a consistent tree — never in the middle of one.

### Required Interfaces and Contracts

- The branch is `feature/restructure-into-internal-packages`. Do not recreate it;
  do not rename it.
- The backup branch exists before `git rebase` runs and is not deleted until the
  PR is merged.
- After this task, `git rev-list --left-right --count upstream/main...HEAD` reports
  `0` on the left.
- The eight harness commits already on the branch (`.ai-factory/`, `.claude/`,
  `.mcp.json`, `AGENTS.md`) are preserved. They ship in the PR deliberately.

### Error Handling and Logging

Not applicable — this is a git operation. A rebase conflict is resolved by hand,
not by `--strategy`; the branch's own commits are documentation and harness
configuration, and an automatic resolution can silently drop a hunk.

### Tests

The rebase is verified by the build and by Task 1's baseline run, which follows
immediately:

```
CGO_ENABLED=0 go build ./...
go vet ./...
```

### Acceptance Criteria

- `git rev-list --left-right --count upstream/main...HEAD` reports `0` upstream
  commits missing.
- A backup branch exists.
- Every number in step 5 either matches or has been corrected in this bundle.
- `index.md`'s base revision is the new HEAD.

### Verification

- `git rev-list --left-right --count upstream/main...HEAD`
- Expected result: `0` followed by the branch's own commit count.
- `git branch --list 'backup/*'`
- Expected result: one entry, dated today.

---

## Task 1: Record the pre-restructuring test baseline

### Intent

345 files are about to move. A red test after a move commit must be
distinguishable from a test that was already red, on a matrix of 26 build targets,
without re-investigating from scratch each time. Every later task's Verification
section compares against this file.

### Implementation Steps

1. Create `.ai-factory/RESTRUCTURE_BASELINE.md` with a header recording revision
   (`git rev-parse HEAD` — the post-rebase HEAD from Task 0, not the revision this
   plan was written at), `go version`, and the host platform the run was taken on.
2. Run and record the main module:
   ```
   CGO_ENABLED=0 go build ./...
   go vet ./...
   go test -timeout 25m ./...
   ```
   Record the exit code, the count of `ok` / `no test files` / `FAIL` packages,
   and the total package count from `go list ./... | wc -l`.
3. Run and record each of the other five modules separately. `go test ./...`
   from the repository root sees only the main module's 38 packages, so a failure
   in `tools/` is invisible to it:
   ```
   go -C tools/conptyreconcile test ./...
   go -C tools/icons test ./...
   go -C tools/wine_syscall_probe build ./...
   GOOS=windows GOARCH=amd64 go -C tools/wine_syscall_probe build ./...
   go -C internal/hideconsole build ./...
   ```
4. Record the three known non-green states verbatim, each with the reason, so
   nobody "fixes" them mid-migration:
   - `tools/icons` — `TestRenderScalesStrokesAndGradients` fails at
     `main_test.go:71`: it opens `../../assets/icon/f4.svg` while the tool itself
     (`main.go:37`) reads `../../cmd/f4/assets/icon/`. The test never agreed with
     its subject. **Pre-existing. Do not repoint it at `cmd/f4/assets/icon` as
     part of this work.**
   - `tools/wine_syscall_probe` — does not build on darwin/arm64: `main.go:5`
     declares `func rawGetpid() uint64` with its body in `probe_amd64.s`. Builds
     clean under `GOOS=windows GOARCH=amd64`. **A platform probe, not breakage.**
   - `tools/icons/third_party/oksvg` — vendored third-party, out of scope.
5. Record `cmd/f4/plugring_policy_test.go:40` as green-but-inert: it reads a
   CWD-relative `plugring/index.yaml` from `cmd/f4/`, always reaches `t.Skipf`,
   and asserts nothing. Task 15 decides its fate.
6. Record the tests that cannot travel with their file and must be reworked
   instead: `command_palette_coverage_test.go` (Task 2) and every test on the
   `swapFrameManager` / `setupMockPanelsFrame` harness (Task 9).

### Required Interfaces and Contracts

The file is prose consumed by a human and by the Verification step of every later
task. Structure it as one `##` section per module so a diff against a later run is
readable. Record counts, not full logs.

**The file is immutable.** It is a snapshot of the Task 0 revision and is never
rewritten from a later run. No wave updates it; the only task that touches it
after this one is Task 42, which deletes it. Rewriting it from a fresh run turns a
regression introduced on wave N into "known red" on wave N+1 and loses the signal
permanently — at wave fourteen, when nobody remembers whether that test was red to
begin with. Later tasks *compare against* it; they do not refresh it.

When a wave legitimately changes the test inventory — Task 9 converts three
packages' tests to `package X_test`, Task 2 re-keys the palette auditor's 42 audit
entries — that goes in the commit message, not in this file. The baseline answers
"what was true before all of this", never "what was true yesterday".

### Error Handling and Logging

Not applicable — no product code changes. If a command in step 2 fails, that is a
finding to record, not an error to suppress.

### Tests

No new tests. This task *is* the test record.

### Acceptance Criteria

- `.ai-factory/RESTRUCTURE_BASELINE.md` exists and names the revision it was taken
  at.
- All six modules have a section.
- The three known non-green states are recorded with their reason.

### Verification

- `git rev-parse HEAD` matches the revision recorded in the file.
- Expected result: identical strings.

---

## Task 2: Re-key the command-palette auditor to qualified symbols

### Intent

`command_palette_coverage_test.go` walks the whole module and asserts that every
`ProcessKey` and every `vtui.NewVMenu` is either reachable from the command
palette or listed as a deliberate exception. It is the only module-wide invariant
in the suite and it stays in `cmd/f4` — a per-package copy would see only its own
subtree and would miss a handler added in a third package, which is the thing the
test exists to catch.

Its 42 audit keys are file paths, so every one of the 27 commits in this plan
would rewrite some of them. Re-key once, to something that does not change when a
file moves.

### Implementation Steps

1. Locate the key-producing code in `cmd/f4/command_palette_coverage_test.go` and
   change the emitted key from `<file path>:<qualified symbol>` to
   `<package name>.<qualified symbol>`:
   `cmd/f4/file_panel.go:(*FileSystemPanel).ProcessKey` becomes
   `panel.(*FileSystemPanel).ProcessKey`.
2. Rewrite all 42 entries in the exception/audit list to the new shape, using the
   **target** package name so the list is never touched again.
   The 42 keys come from 35 distinct source files
   (`grep -o 'cmd/f4/[a-z0-9_]*\.go' cmd/f4/command_palette_coverage_test.go | sort -u`),
   and their destinations are **ten** packages, not six:
   - `app` — `actions.go`, `ai_chat_panel.go`, `arkanoid.go`, `sheet_frame.go`
   - `cmdline` — `apply_command_output.go`, `command_line.go`
   - `dialog` — `bookmarks_dialog.go`, `codepage_settings.go`,
     `command_palette_ui.go`, `find_file.go`, `grabber.go`, `hotkeys_ui.go`
   - `editor` — `editor_base64.go`, `editor_find_all.go`, `editor_view.go`
   - `fileops` — `fuse_mount_list.go`, `queue_manager.go`
   - `macro` — `macro.go`
   - `media` — `image_view.go`, `player_panel.go`, `video_view.go`
   - `panel` — `drive_bookmarks_ui.go`, `file_associations_editor.go`,
     `file_associations_ui.go`, `file_panel.go`, `info_panel.go`,
     `panel_plugins.go`, `panels_frame.go`, `quick_view_panel.go`,
     `temp_panel.go`, `user_menu_ui.go`, `viewer_editor_history.go`
   - `plughost` — `plugin_hotkeys.go`, `rpc_panel.go`
   - `viewer` — `viewer_view.go`

   `term` owns none of the 42 and must not appear in the list.
3. **The walker must emit the same target name from day one, while every file is
   still in `cmd/f4`** — otherwise the audit list says `panel.X`, the walker says
   `main.X`, and the test is red from this commit until Task 36. A
   file→target-package map cannot do this: at this task there is only one directory.
   Use a **file→target-package** map instead, seeded with all 35 entries above, and
   have the walker fall back to the file's actual directory for anything not in it.
   Each wave then deletes the entries it has satisfied — the map shrinks to nothing
   by Task 37, and the emitted keys never change.
4. Assert in the test that the audit list and the discovered set have the same
   cardinality (42), so a silent drop is a failure rather than a smaller list. Run
   the test at the end of this task and require `--- PASS`, not merely compilation:
   a mismatch here is the failure mode step 3 exists to prevent.

### Required Interfaces and Contracts

- Key format: `<package>.<receiver-qualified symbol>`, e.g.
  `panel.(*FileSystemPanel).ProcessKey`, `editor.(*EditorView).ProcessKey`.
- The file→target-package map is the only place a path appears. Its contract: one
  entry per audited file whose target package differs from its current directory,
  and it is empty once the last wave lands.
- Cardinality invariant: 42 audit keys before and after this task, and after every
  wave. If a key's subject genuinely disappears, that is a finding to report, not a
  number to adjust.

### Error Handling and Logging

The test's failure message must name the key that was not found and the key set it
searched, so a rename shows up as a readable diff and not as "expected 42, got
41". No logging changes — this is a test.

### Tests

This task modifies a test. Its own verification is that the test still passes and
still has 42 keys:

```
go test ./cmd/f4 -run '^TestCommandPalette' -v
```

### Acceptance Criteria

- No key in `command_palette_coverage_test.go` contains a `/` or `.go`.
- The test passes.
- The audit list has exactly 42 entries, unchanged in subject.

### Verification

- `grep -c 'cmd/f4/' cmd/f4/command_palette_coverage_test.go`
- Expected result: `0` inside the key list; the file→target-package map's 35
  entries are keyed by bare basename, not by path.
- `go test ./cmd/f4 -run '^TestCommandPalette' -v`
- Expected result: `--- PASS`. A failure naming a `main.`-prefixed key means step 3
  was implemented as a directory map.

---

## Task 3: Make action registration order explicit

### Intent

Inside one package Go runs `init()` in filename order; across packages it follows
the import graph. Today 174 `RegisterAction` calls sit in
`action_registry.go:254-2807` and the rest in six more files
(`fuse_mount_action.go` — which has two `init()`s — `fuse_mount_list.go`,
`sheet_actions.go`, `sqlite_actions.go`, `static_direct_actions.go`,
`vtvibe_host.go`). That order is what the menus and the command palette *display*.
Splitting the registry across packages therefore reorders the user's menu silently,
and no test catches it.

### Implementation Steps

1. Add an explicit ordinal to the registry. `RegisterAction`
   (`action_registry.go:121`) currently appends to `actionOrder` on first sight;
   change it to record a monotonically increasing sequence number per action, or
   accept an explicit `Order int` on `Action`. Prefer the sequence number — it
   requires no edit to the 174 call sites.
2. Make every consumer of `actionOrder` sort by that ordinal explicitly rather
   than relying on append order. Find them with
   `npx -y @colbymchenry/codegraph@1.6.0 callers actionOrder`.
3. Because a sequence number assigned at call time still depends on `init()`
   order across packages, pin the order that must survive: capture today's
   `actionOrder` as a golden fixture (Task 3's test below) *before* changing
   anything, then make the sort key the position in that fixture for actions the
   fixture knows, appending unknown actions at the end. This turns the menu order
   from an emergent property of file names into recorded data.
4. Record in a comment on the ordering function that the order is observable
   behaviour and that changing it changes the user's menu.

### Required Interfaces and Contracts

- `RegisterAction(action Action)` keeps its signature. Adding a parameter would
  touch 174 call sites for no gain.
- Ordering is deterministic and independent of file names, package names and
  import order. This is the contract the later waves depend on.
- Duplicate registration keeps today's behaviour: `RegisterAction` already
  ignores a second registration of the same lowercase name and must continue to.

### Error Handling and Logging

No new failure modes and no new logging. A duplicate or missing action is a test
failure, not a runtime log line: this codebase reports user-facing failures to
stderr with the `f4: ` prefix and diagnostics through `VTUI_DEBUG`, and neither
applies to a registry that is fully determined at startup.

### Tests

New `cmd/f4/action_registry_order_test.go`:

- `TestActionOrderIsStable` — asserts the full ordered list of action names
  against a golden slice captured from the pre-change build. This is the test that
  makes a silent menu reorder loud.
- `TestActionOrderCoversRegistry` — every key in `actionRegistry` appears exactly
  once in `actionOrder`, and the lengths match.
- `TestActionOrderIndependentOfRegistrationSequence` — register a small synthetic
  set in two different sequences and assert the resulting order is the same.

Capture the golden slice by running the current binary's registry dump before the
change, not by transcribing `action_registry.go` by hand.

### Acceptance Criteria

- Menus and the command palette present actions in exactly the order they do
  today.
- Ordering does not consult file names, package names, or `init()` sequence.
- The three tests above pass.

### Verification

- `go test ./cmd/f4 -run '^TestActionOrder'`
- Expected result: `ok`, three tests passing.
- `go test ./cmd/f4 -run '^TestCommandPalette'`
- Expected result: `ok` — the palette's own coverage test still agrees.

---

## Task 4: Move `F4Config`'s field types into `config.go`

### Intent

`internal/config` must import no other `internal/*` package — that is the rule
that lets `config.App` stay a package-level global without creating a cycle.
`F4Config` (`config.go:352`) has five named-type fields; three of those types are
declared outside `config.go`. One of them, `StartupMode`, is declared in
composition-root code that leaves `cmd/f4` **last**, which would make layer 0
depend on layer 4.

### Implementation Steps

1. Move `type PanelNavigationMode int` and its constants
   (`NavigationClassic`, `NavigationVim`, `NavigationSearchFirst`) plus the
   `String()` method from `cmd/f4/navigation_mode.go:7-27` into `cmd/f4/config.go`,
   next to `PanelScrollbarMode` (`config.go:125`) and
   `WorkspaceTabNumberingMode` (`config.go:144`).
2. Move `type compareOptions struct` from `cmd/f4/compare_folders.go:61` into
   `config.go`. Keep the field comments verbatim: they document the Advanced
   Compare dialog field by field.
3. Move `type StartupMode int` and its constants from
   `cmd/f4/startup_backend.go:11` into `config.go`.
4. Leave every function and method that *uses* these types where it is. Only the
   declarations move.
5. Confirm no fourth type is hiding: re-derive the list with
   ```
   awk '/^type F4Config struct/,/^}/' cmd/f4/config.go |
     grep -E '^\s+[A-Z][A-Za-z0-9_]*\s+[A-Z]'
   ```
   and check each named type's declaration site.

### Required Interfaces and Contracts

- No identifier is renamed. `PanelNavigationMode` stays `PanelNavigationMode`;
  Far-derived names are never renamed during a move.
- `F4Config`'s field set, order and tags are unchanged — `config.go:352`'s literal
  initialiser must still compile untouched.
- After this task, every type named in a `F4Config` field is declared in
  `config.go`.

### Error Handling and Logging

None. This is a declaration move inside one package; the compiler is the only
check that matters.

### Tests

No new tests. The existing config suite must pass unchanged:

```
go test ./cmd/f4 -run '^TestConfig|^TestNavigationMode|^TestCompare'
```

Do not add a test asserting "these types live in config.go" — Task 8's boundary
auditor covers the invariant that actually matters (no upward import) once the
packages exist.

### Acceptance Criteria

- `grep -n '^type' cmd/f4/navigation_mode.go` returns nothing.
- `grep -n '^type StartupMode' cmd/f4/startup_backend.go` returns nothing.
- `grep -n '^type compareOptions' cmd/f4/compare_folders.go` returns nothing.
- All five `F4Config` field types are declared in `config.go`.

### Verification

- `CGO_ENABLED=0 go build ./...`
- Expected result: exit 0.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.

---

## Task 5: Stop starting a goroutine from `init()`

### Intent

`queue_manager.go:308-317` constructs `GlobalQueueManager` and calls
`go GlobalQueueManager.workerLoop()` on import. Once this file lives in
`internal/fileops`, *importing that package* starts a worker — in every binary
that links it and in every test process that touches it, including tests that have
nothing to do with the queue. The composition-root principle bans exactly this:
`init()` with side effects beyond assignment.

### Implementation Steps

1. Keep the assignment half of `init()`. The zero-value manager must stay usable —
   `workerLoop`'s periodic fallback exists so that zero-value test managers work
   (see the comment at `queue_manager.go:138`), and tests rely on it.
2. Extract the `go GlobalQueueManager.workerLoop()` line into a new exported
   `StartQueueWorker()` in the same file, idempotent (guard with a `sync.Once` so a
   double call from a test cannot start two workers).
3. Call `StartQueueWorker()` from the startup path in `cmd/f4/main.go`, at the
   point where other process-wide services are started, before the event loop
   begins. It must run before the first enqueue; find the earliest enqueue with
   `npx -y @colbymchenry/codegraph@1.6.0 callers GlobalQueueManager`.
4. In `cmd/f4/queue_manager_test.go` and any other test that depends on a running
   worker, call `StartQueueWorker()` explicitly in the test setup. Tests that only
   enqueue and inspect state need no worker and must keep working without one.

### Required Interfaces and Contracts

```go
// StartQueueWorker starts the background scheduler. It is idempotent: the
// worker is started at most once per process.
func StartQueueWorker()
```

- `GlobalQueueManager` stays a package-level variable with its current type and
  field initialisation. Only the goroutine launch moves.
- Invariant preserved: enqueueing before the worker starts must still work — the
  wake channel is buffered (`wake: make(chan struct{}, 1)`) and `workerLoop` has a
  periodic fallback, so a task enqueued before start is picked up on the first
  cycle. Do not add a "not started" error path; that would be a behaviour change
  inside what must remain a mechanical refactor.

### Error Handling and Logging

No new failure modes. `StartQueueWorker` cannot fail. Do not add a log line for
"worker started": the project has no logging framework and this path is not a
user-facing failure.

### Tests

- Existing `cmd/f4/queue_manager_test.go` must pass. Add `StartQueueWorker()` to
  the setup of the tests that previously relied on the import-time start.
- Add `TestStartQueueWorkerIsIdempotent` — call it twice, assert exactly one
  worker goroutine, using the same `runtime/pprof` goroutine-profile technique
  `frame_manager_test_helpers_test.go` already uses for
  `TestFrameManagerShutdownStopsTaskPumpAndUnblocksPostTask`. That helper is
  `taskPumpGoroutineProfile` (`:66`); Task 9 moves it out as
  `testutil.TaskPumpGoroutineProfile`, so a test written here against the local
  name needs the import swapped when Task 9 lands. Task 32, which moves
  `queue_manager.go` and its tests into `internal/fileops`, depends on this task.

### Acceptance Criteria

- `grep -n 'go GlobalQueueManager' cmd/f4/queue_manager.go` returns nothing inside
  `init()`.
- No goroutine is running after importing the package in a test that does not call
  `StartQueueWorker`.
- The full suite matches the Task 1 baseline.

### Verification

- `go test ./cmd/f4 -run '^TestQueue|^TestStartQueueWorker' -race`
- Expected result: `ok`.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.

---

## Task 6: Lift the drive registry out of `panels_frame.go`

### Intent

`drives_unix.go` and `drives_windows.go` build `[]DriveEntry`, but `DriveEntry`,
`DriveRegistry` and `RegisterDrive` are declared at `panels_frame.go:26-40` — on
the panel type's file. The architecture assigns drives to `internal/sysinfo`, a
layer-0 leaf; leaving the type on `panels_frame.go` would mean sysinfo (Task 22,
the *first* wave) depends on panel (Task 34, the second-to-last). The registry
itself has no panel dependency at all: `DriveEntry` is `{Name string; Factory
func() vfs.VFS}` and `RegisterDrive` needs only `sync`.

### Implementation Steps

1. Create `cmd/f4/drive_registry.go` and move `type DriveEntry`
   (`panels_frame.go:26-29`), `var DriveRegistry` (`:31`), the mutex that guards it
   (`pluginRegistryMu`, `:32`) and `RegisterDrive` (`:34`) into it.
2. Check whether `pluginRegistryMu` guards anything besides the drive registry —
   its name suggests a wider role. `grep -n 'pluginRegistryMu' cmd/f4/*.go`. If it
   does, give the drive registry its own mutex rather than moving a shared one,
   and leave `pluginRegistryMu` where it is.
3. Leave `drives_unix.go`, `drives_windows.go` and `drive_menu_options.go`
   untouched — they reference the identifiers, which are still in the same package.
4. Do **not** move `drive_menu_options*.go` or `drive_bookmarks*.go`: those are
   menu UI over the registry and belong to `internal/panel`, which will import
   `internal/sysinfo` (layer 3 → layer 0, allowed).

### Required Interfaces and Contracts

```go
type DriveEntry struct {
    Name    string
    Factory func() vfs.VFS
}

var DriveRegistry []DriveEntry

func RegisterDrive(name string, factory func() vfs.VFS)
```

- Semantics unchanged, including `RegisterDrive`'s replace-in-place behaviour for
  an existing name.
- The only import the new file needs is `sync` and `github.com/unxed/f4/vfs`. If it
  needs anything else, something was moved that should not have been.

### Error Handling and Logging

None. No new failure modes.

### Tests

No new tests. The drive-menu tests must pass unchanged:

```
go test ./cmd/f4 -run '^TestDrive'
```

### Acceptance Criteria

- `cmd/f4/drive_registry.go` imports only `sync` and `vfs`.
- `grep -n 'DriveEntry\|DriveRegistry\|RegisterDrive' cmd/f4/panels_frame.go`
  returns only *uses*, no declarations.
- The suite matches the Task 1 baseline.

### Verification

- `CGO_ENABLED=0 go build ./...` and `GOOS=windows GOARCH=amd64 go build ./...`
- Expected result: exit 0 for both — `drives_windows.go` is the other consumer.

---

## Task 7: Remove sysinfo's last localization call

### Intent

`internal/sysinfo` is named in the dependency rules as a leaf that imports no
other `internal/*` package, and it is the first wave to leave `cmd/f4`. The
sysinfo family is clean apart from a single call: `gpu_info_linux.go:113` builds
`Model: Msg("InfoPanel.GPUWSLVirt")`. Left as is, `internal/sysinfo` would import
`internal/i18n` and stop being a leaf — and, since the config group leaves in Task
24, sysinfo could not go first at all.

### Implementation Steps

1. In `cmd/f4/gpu_info_linux.go:113`, return the message *key*
   (`"InfoPanel.GPUWSLVirt"`) instead of the localized string. Mark it so the
   renderer can tell a key from a vendor-supplied model name — the cleanest way
   with no new type is a sibling field, e.g. `ModelKey string`, left empty for real
   model names.
2. In the GPU rendering site, localize at render time: if `ModelKey` is non-empty,
   display `Msg(ModelKey)`, otherwise display `Model`. Find the site with
   `npx -y @colbymchenry/codegraph@1.6.0 callers gpuInfo` (or the accessor the file
   exposes) — it is expected to be `cmd/f4/info_panel.go`, which lands in
   `internal/panel` and may legally import `internal/i18n`.
3. Re-run the leaf check over the whole family and confirm it comes back empty:
   ```
   grep -lE '\b(Msg|AppConfig|showToast)\b' \
     cmd/f4/cpu_info*.go cmd/f4/mem_info*.go cmd/f4/fs_info*.go \
     cmd/f4/gpu_info*.go cmd/f4/drives_*.go cmd/f4/drive_registry.go
   ```

### Required Interfaces and Contracts

- The GPU info struct gains one field; no existing field changes meaning.
- Invariant: nothing under the sysinfo family calls `Msg`, reads `AppConfig`, or
  calls `showToast`. This is what makes its outbound edge count zero.
- The user-visible string is unchanged — same key, same catalogue, localized one
  layer up.

### Error Handling and Logging

A missing catalogue key already renders as `{Key}` via `Msg`'s existing fallback
(`Action.DisplayLabel` relies on the same behaviour). Do not add a new fallback.

### Tests

- Existing GPU info tests pass unchanged.
- Add one case to the info-panel test asserting that the WSL virtual-GPU row
  renders the localized string, not the raw key — this is the regression the move
  could introduce.

```
go test ./cmd/f4 -run '^TestGPU|^TestInfoPanel'
```

### Acceptance Criteria

- The `grep` in step 3 returns nothing.
- The WSL row renders the same text as before the change.

### Verification

- `GOOS=linux GOARCH=amd64 go build ./...`
- Expected result: exit 0 — `gpu_info_linux.go` only compiles on linux.
- `go test ./cmd/f4 -run '^TestGPU|^TestInfoPanel'`
- Expected result: `ok`.

---

## Task 8: Add the module boundary auditor

### Intent

Four of the dependency rules are mechanically checkable. Written now, the test is
green against today's tree and turns red the first time a wave introduces a
violation. Added after the migration it would only confirm what already happened.

### Implementation Steps

1. Create `cmd/f4/architecture_test.go`, standard library only. Obtain the import
   graph by shelling out to the toolchain — no new dependency, no
   `golang.org/x/tools/go/packages`:
   ```go
   out, err := exec.Command("go", "list", "-f",
       "{{.ImportPath}} {{join .Imports \" \"}}", "./...").Output()
   ```
   Run it with the module root as the working directory (`..`/`..` from
   `cmd/f4`, or resolve via `runtime.Caller` the way `plugring_test.go:40` does).
2. Assert rule 1 — the public contract stays public: no package whose import path
   is under `<module>/sdk/` or `<module>/vfs/` imports anything containing
   `/internal/`.
3. Assert rule 2 — nothing imports the entry point: no package imports
   `<module>/cmd/f4`.
4. Assert rule 3 — no upward imports into the application: no package other than
   `<module>/cmd/f4` imports `<module>/internal/app`.
5. Assert rule 4 — the module's own import graph is acyclic. Depth-first search
   over the graph restricted to `<module>/…` paths; report the cycle as a path,
   not as a boolean.
6. Encode the layer table as one ordered `map[string]int` at the top of the file,
   e.g. `{"internal/config": 0, "internal/sysinfo": 0, … "internal/app": 4}`.
   Rules 3 and any future layer assertion read from it. Each wave updates one line.
   **Seed it with the `internal/*` packages that already exist**, or the map is
   incomplete from the first commit and Task 41's "every package in the layer map
   appears in the document, and vice versa" can never pass — `ARCHITECTURE.md:285-286`
   already places all of them at layer 0:
   ```go
   "internal/netproxy": 0,   // 22 importers today
   "internal/ttyx":     0,   // 9;  internal/term imports it (Task 30)
   "internal/wincon":   0,   // 3;  internal/term imports it (Task 30)
   ```
   `internal/hideconsole` is **not** in the map: it is a vendored fork with its own
   `go.mod` (`replace` at `go.mod:187`), so `go list ./...` never returns it. Say so
   in a comment beside the map, or the next reader adds it and the test goes red.
7. Skip the whole test when the `go` binary is unavailable
   (`t.Skip("go toolchain not available")`) so a restricted sandbox does not turn
   this into a spurious failure.

### Required Interfaces and Contracts

- Test name prefix `TestArchitecture` so `-run '^TestArchitecture'` selects the
  whole group.
- One subtest per rule, named for the rule, so a failure message names the rule
  that broke.
- Failure output lists every offending edge as `importer -> imported`, not just
  the first.
- The layer map is the single place a package name appears. Adding a package to
  the map is the only edit a wave makes to this file.

### Error Handling and Logging

`go list` failing is a test failure with the command's stderr attached, not a
skip — a silent skip would disable the auditor for the rest of the migration. The
only skip is the missing-toolchain case in step 7.

### Tests

This task *is* a test. It must be green on today's tree — that is its acceptance
criterion. Verify it can actually fail by temporarily adding an import of
`internal/wincon` to a `vfs` file, confirming rule 1 goes red, then reverting.

### Acceptance Criteria

- `go test ./cmd/f4 -run '^TestArchitecture'` passes on the current tree.
- Deliberately introducing each of the four violations makes the matching subtest
  fail with a message naming the offending edge.
- The file adds no module dependency: `go.mod` is unchanged.

### Verification

- `go test ./cmd/f4 -run '^TestArchitecture' -v`
- Expected result: four subtests, all pass.
- `git diff --stat go.mod go.sum`
- Expected result: empty.

---

## Task 9: Give the shared frame harness a home

### Intent

`swapFrameManager` is used by 63 test files and `setupMockPanelsFrame` by 28. Left
in `cmd/f4`, every wave would strand its own tests. But they cannot share one
package: `setupMockPanelsFrame` (`panels_frame_test.go:794`) calls
`NewTerminalView` (term), `NewCommandLine` (cmdline), `NewFileSystemPanel` and
`PanelsFrame.initPTY` (panel), and constructs `PanelsFrame` — so a package holding
it imports three layer-3 packages, and the in-package tests of those same three
packages cannot import it back without an import cycle.

Split by dependency depth, and do it now, while everything is still one package:
the resulting diff touches only `_test.go` files and can be reviewed as one change
of call shape.

### Implementation Steps

1. Create `internal/testutil`. Move from
   `cmd/f4/frame_manager_test_helpers_test.go` into it, exported:
   - `SwapFrameManager(t *testing.T, drains ...func(*testing.T)) func()` — was
     `swapFrameManager` (`:129`)
   - `SetFrameManagerScreens` — was `setFrameManagerScreensForTest` (`:36`)
   - `AppendFrameManagerScreen` — was `appendFrameManagerScreenForTest` (`:52`)
   - `CloseFrameManagerFrames`, `CloseFrameManagerScreens` — were
     `closeFrameManagerFrames` (`:27`), `closeFrameManagerScreens` (`:14`)
   - `TaskPumpGoroutineProfile` — was `taskPumpGoroutineProfile` (`:66`). Not
     optional: `TestFrameManagerShutdownStopsTaskPumpAndUnblocksPostTask` is built
     on it and moves to `internal/testutil` in this same task, and Task 5's
     idempotence test uses the same technique.
   - `PumpUntilToastActive`, `WaitForToastExpiry`
   - `ScreenRow` — was `misc.go:12`; its five callers are all `_test.go`
     (`image_gallery_test.go`, `file_panel_test.go`, `image_view_overlay_test.go`,
     `editor_find_all_test.go`, `ai_chat_panel_test.go`), so it is test scaffolding
     that happens to live in production code today.

   After these ten helpers leave, `cmd/f4/frame_manager_test_helpers_test.go`
   retains only `waitForDirectoryLoads` (`:83`), which stays until Task 34.
2. Break the two production drains out of `SwapFrameManager` and make them
   parameters. Today it calls `waitForAsyncClipboard` (`clipboard_async.go:27`,
   production, lands in `internal/term`) and `waitForDirectoryLoads`
   (`frame_manager_test_helpers_test.go:83`, which waits on the
   `directoryLoadWorkers` production global that lands in `internal/panel`).
   Reaching down into two layer-1/3 packages is exactly the cycle this task
   exists to avoid. New shape:
   ```go
   // SwapFrameManager replaces the global vtui.FrameManager with a fresh
   // instance and returns a restore function. Each drain runs before the swap
   // and before the restore: a caller passes the waits for whichever background
   // workers its own package leaves running.
   func SwapFrameManager(t *testing.T, drains ...func(*testing.T)) func()
   ```
3. `internal/testutil` must import only `testing`, `time`, `runtime/pprof`,
   `bytes`, `strings`, `github.com/unxed/vtui` and
   `github.com/unxed/vtui/vreactive`. If it needs an `internal/*` import,
   something was moved that should have stayed a drain.
4. Update all 63 `swapFrameManager` call sites to
   `testutil.SwapFrameManager(t, waitForAsyncClipboard, waitForDirectoryLoads)`
   — or the subset each test needs. Keep `waitForDirectoryLoads` in
   `cmd/f4/frame_manager_test_helpers_test.go` for now; it travels to
   `internal/panel`'s test files in Task 34. Two files mention `swapFrameManager`
   in prose only (`config.go:1204`, `queue_manager.go:138`); update the comments,
   they are not call sites.
5. Create `internal/paneltest` with a `doc.go` only. Document its contract: it
   will hold `SetupMockPanelsFrame`, it may import `internal/panel`,
   `internal/cmdline` and `internal/term`, and any test *inside* those three
   packages that uses it must be an external test package (`package panel_test`).
   It is filled in Task 34, when those packages exist.
6. Leave `setupMockPanelsFrame` in `cmd/f4/panels_frame_test.go` unchanged for
   now. Moving it before `panel` exists would only move the problem.

### Required Interfaces and Contracts

- Everything moved is exported; `internal/testutil` is a normal package, and Go
  will not let a `_test.go`-only package be imported.
- `SwapFrameManager` semantics are unchanged apart from the drains: it must still
  replace `vtui.FrameManager`, swap `vreactive.GlobalUpdateQueue` and
  `vreactive.GlobalAnimationManager`, and its restore function must close the
  fresh manager's frames, call `Shutdown`, and restore all three globals.
- Drain ordering is part of the contract: drains run *before* the swap and *before*
  the restore, in the order given. `swapFrameManager`'s current comments explain
  why (a directory-load worker still reading the old manager is what the race
  detector reports against whichever test does the replacing) — carry those
  comments across verbatim.
- `internal/testutil` and `internal/paneltest` are test scaffolding that ships in
  the module. That is the price of 91 cross-package call sites; note it in the
  package doc so nobody imports them from production code. Task 8's auditor can
  grow a fifth rule for this later, but not in this task.

### Error Handling and Logging

Failure modes are unchanged: the 30-second timeout in `waitForDirectoryLoads` is
still a `t.Fatal`, and toast pumps still `t.Fatal` on timeout. Keep the existing
messages verbatim — they are what a failing CI job shows.

### Tests

The moved helpers are themselves exercised by the 63 + 28 tests that use them, so
the acceptance test is the suite. In addition, keep
`TestFrameManagerShutdownStopsTaskPumpAndUnblocksPostTask` — it asserts the
harness does not leak task pumps and is the one test that tests the harness
itself. It moves with the helpers, into `internal/testutil`.

### Acceptance Criteria

- `internal/testutil` imports no `internal/*` package.
- `grep -rn 'func swapFrameManager\|func setFrameManagerScreensForTest\|func appendFrameManagerScreenForTest\|func taskPumpGoroutineProfile' cmd/f4/`
  returns nothing.
- `cmd/f4/frame_manager_test_helpers_test.go` declares exactly one function,
  `waitForDirectoryLoads`.
- `internal/paneltest` exists with `doc.go` and no other file.
- The full suite matches the Task 1 baseline.

### Verification

- `go list -f '{{join .Imports "\n"}}' ./internal/testutil | grep internal/`
- Expected result: no output.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.
- `go test -race -shuffle=on -timeout 5m ./cmd/f4 -run '^TestFrameManager'`
- Expected result: `ok` — the harness is the thing most likely to regress under
  the detector.

---

## Task 43: Assign every `cmd/f4` file to a wave

### Intent

The wave tasks name the files they own. Two sets of files are named by nobody, and
both fall through to Task 36 step 4's "and whatever else remains", which is how
`internal/app` becomes the flat package that Phase 10 lists as its own top risk.

- **21 non-test sources** appear in no task and match no glob the bundle writes.
- **154 of the 346 `_test.go` files** have no same-named source. Wave-procedure
  step 3 says "take every `_test.go` neighbour", which is a filename rule, so it
  strands 45% of the suite: those tests stay in `cmd/f4` while their subjects
  leave, and either stop compiling or — worse — keep passing against nothing.

This task produces the assignment. It writes no Go code and produces no commit of
its own; its output is a table in this bundle that the waves then execute.

### Implementation Steps

1. Re-derive both sets on the current revision, so the task is self-checking:
   ```
   # sources named by no task
   comm -23 <(ls cmd/f4/*.go | grep -v '_test\.go$' | sed 's|cmd/f4/||' | sort) \
            <(grep -rhoE '[a-z0-9_]+\.go' .ai-factory/plans/feature-restructure-into-internal-packages/*.md | sort -u)
   # tests with no same-named source
   comm -23 <(ls cmd/f4/*_test.go | sed 's|cmd/f4/||;s|_test\.go$||' | sort) \
            <(ls cmd/f4/*.go | grep -v '_test\.go$' | sed 's|cmd/f4/||;s|\.go$||' | sort)
   ```
   Expect 47 and 154. The first list minus the six families the bundle covers by
   glob (`command_palette*`, `drive_bookmarks*`, `drive_menu_options*`,
   `file_associations*`, `user_menu*`, `host_input_modes*`) is the 21.
2. Assign each of the 21. The graph already answers most of them; these are the
   readings, and each still needs its gate score confirmed before the wave moves it:

   | Files | Wave |
   |---|---|
   | `ttyx_probe.go`, `ttyx_probe_parse.go`, `ttyx_probe_unix.go`, `ttyx_probe_windows.go`, `ttyx_session.go` | Task 30 — `internal/term`; they decide what the terminal supports |
   | `terminal_log_console_other.go`, `terminal_log_console_windows.go`, `terminal_log_vfs.go` | Task 30 — `internal/term` |
   | `console_host_windows.go`, `console_overlay_other.go`, `console_overlay_windows.go` | Task 30 — `internal/term`; score them, `console_overlay_*` may belong to `internal/media` |
   | `process_environment.go` (gate 4), `process_environment_runtime_unix.go`, `process_environment_runtime_windows.go`, `process_environment_shell.go` | score first; the runtime pair is gate 0, `process_environment.go` is not |
   | `plugring.go`, `plugring_meta.go`, `plugring_ui.go` | Task 26 — `internal/plughost`. Task 15 only edits `plugring.go`'s catalogue URL; it never assigns it a package |
   | `compare_folders_ui.go` (gate 4, `Msg` ×28) | Task 25 — `internal/dialog`, beside the other settings dialogs |
   | `colorer_downloader.go` | Task 33 — `internal/editor`, with `colorer_plugin.go` |
   | `action_menu.go`, `external_ui.go`, `info_usage.go` | score each; `action_menu.go` and `external_ui.go` read as `internal/dialog`, `info_usage.go` as `internal/panel` |

3. Classify the 154 tests **by the symbols they call, not by their filename**.
   108 of them reference no view type at all, so the eight-name gate cannot answer
   for them; use `codegraph callees` on the test's own functions and send the file
   to the package that owns what it exercises.
4. **Name the tests that exercise more than one future package.** They cannot
   travel whole and must be split the way `semantic.go` is split, by an explicit
   instruction rather than implementer judgement. Measured, there are five:

   | Test | Packages it touches |
   |---|---|
   | `cloudfox_real_ui_test.go` | panel, editor, viewer |
   | `codepage_issue875_test.go` | panel, editor, viewer |
   | `command_palette_dynamic_test.go` | panel, plughost |
   | `terminal_selection_test.go` | panel, term |
   | `editor_binary_open_test.go` | panel, editor |

   For each, decide split-or-host and record it here: a test that only *constructs*
   another package's type can often stay whole as an external test package
   (`package panel_test`) in the package that owns most of its assertions.
5. Watch for the inverse hazard: a test with no same-named source often covers
   *several* sources at once (`ttyx_probe_test.go` covers `ttyx_probe_parse.go`;
   `process_environment_test.go` covers 39 of 47 functions in
   `process_environment_shell.go`). When those sources land in different packages,
   the test follows one and silently stops covering the others. Record every such
   test and the coverage it will lose, or split it.
6. Write the result into this bundle as a table under this task, and delete Task 36
   step 4's closing phrase "and whatever else remains" — after this task there is
   nothing that remains.

### Required Interfaces and Contracts

- Every one of the 345 non-test files and 346 test files is assigned to exactly one
  wave, or explicitly marked "stays in `cmd/f4`" with the reason. There is no
  residual category.
- A test is assigned by its subject. Filename similarity is evidence, never the
  rule.
- The assignment is data for the waves; this task moves nothing.

### Error Handling and Logging

Not applicable — no product code changes.

### Tests

No new tests. Step 1's two `comm` invocations are the check, and they must both
come back empty when re-run against the finished table.

### Acceptance Criteria

- Both `comm` commands in step 1, re-run with the completed table folded into the
  bundle, return nothing.
- Each of the five multi-package tests in step 4 has a written split-or-host
  decision.
- Task 36 step 4 no longer ends in a catch-all.

### Verification

- The two `comm` commands from step 1.
- Expected result: no output from either.
- `ls cmd/f4/*.go | wc -l` equals the number of rows in the assignment table plus
  the files explicitly marked as staying.

---

## Phase Risks and Mitigations

- **Risk:** the work runs long and upstream accumulates commits in files that have
  already moved, turning each one into a manual conflict inside a stranger's
  change.
  **Mitigation:** Task 0 syncs once, before anything moves, and forbids touching
  upstream again until the PR. The waves should run close together: the cost of
  this risk grows with the length of the window, and upstream gained a commit in a
  first-wave file within a single day of this plan being written.
- **Risk:** Task 3's golden order is captured after an accidental change, freezing
  a wrong menu order forever.
  **Mitigation:** capture the golden slice from a build of the unmodified tree
  (`git stash` if needed) and record the revision it came from in the test file.
- **Risk:** Task 9's drain refactor changes when workers are joined and surfaces a
  latent race in 63 tests at once.
  **Mitigation:** run the race shard before and after
  (`go test -race -shuffle=on ./cmd/f4`), and keep the drain ordering comments
  verbatim so the reason for each wait survives.
- **Risk:** Task 7's `ModelKey` field is set but never read, silently blanking the
  WSL GPU row.
  **Mitigation:** the added info-panel assertion in Task 7's Tests section fails if
  the row renders a key or an empty string.
- **Risk:** Task 5 starts the queue worker after the first enqueue, and a startup
  operation waits a full fallback cycle.
  **Mitigation:** locate the earliest `GlobalQueueManager` use with the graph
  before choosing the call site, as step 3 requires.

## Phase Completion Checklist

- Every Task 0-9 and Task 43 satisfies its acceptance criteria.
- The branch is level with `upstream/main` and a backup branch exists.
- `CGO_ENABLED=0 go build ./...`, `go vet ./...` and `go test -timeout 25m ./...`
  match `.ai-factory/RESTRUCTURE_BASELINE.md` exactly.
- `go test ./cmd/f4 -run '^TestArchitecture'` passes.
- No file has moved between directories in this phase.
- Every `cmd/f4` file has a wave, per Task 43; no residual category remains.
- `index.md` task checkboxes 0-9 and 43 are ticked.

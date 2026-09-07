# Phase 10: The Composition Root

Plan: [index.md](index.md)
Tasks: 36-37
Depends on: Phase 9

## Objective

`internal/app` owns the event loop, the bootstrap and the global application
state the interactive subsystems share. `cmd/f4` is reduced to what the target
layout says it holds: `main.go`, the four module-wide auditors, and the Windows
`.syso` files.

After this phase the compiler, not a convention, keeps every boundary in
`ARCHITECTURE.md`.

## Current-Code Evidence

| Path | Signal | Consequence |
|---|---|---|
| `cmd/f4/action_table.go` | 2553 lines, 174 `RegisterAction`, `PanelsFrame` ×109, `EditorView` ×48 | created in Task 18; this is where it lands |
| `cmd/f4/framework_actions.go` | 25 functions; 18 have **no external callers** — reached only as `Handler:` values and from `main.go:actionScreenDump` | composition-root code, moves whole |
| `cmd/f4/actions.go` | after the earlier waves took 20 functions, the 61 view-bound handlers remain | distributed or moved here with the table |
| `cmd/f4/main.go` | flags, startup mode, wiring | stays in `cmd/f4` |
| `cmd/f4/startup_backend.go` | Task 4 removed `StartupMode` from it | moves to `internal/app` without dragging config up a layer |
| `cmd/f4/runtime_mode.go` | 1 gate reference | moves here |
| `cmd/f4/debug_log.go` | owns `VTUI_DEBUG` | moves here — it is process-wide diagnostics setup |
| `cmd/f4/command_palette_coverage_test.go` | 42 audit keys, module-wide invariant | **stays in `cmd/f4`** |
| `cmd/f4/architecture_test.go` | four rules plus the sysinfo leaf rule | **stays in `cmd/f4`** |
| `cmd/f4/rsrc_windows_amd64.syso`, `rsrc_windows_arm64.syso` | linked only from the built package's directory | **stay in `cmd/f4`** |
| `.github/actions/affected-packages/action.yml:50-54, 64-67, 76-77` | `cmd/f4/*)` treated as one indivisible unit, and the comment that justifies it | the reason for the special case is gone |

## Files to Change

| Path | Action | Required change |
|---|---|---|
| `internal/app/` | create | Event loop, bootstrap, action table, workspaces |
| `cmd/f4/main.go` | modify | Becomes wiring only |
| `.github/actions/affected-packages/action.yml` | modify | Drop the `cmd/f4` special case |
| `cmd/f4/architecture_test.go` | modify | Layer-4 entry; rule 3 now has teeth |

---

## Task 36: Extract `internal/app`

### Intent

What is left after Phase 9 is the application itself: the event loop that
dispatches input to the focused subsystem, the bootstrap, workspace management,
and the 2553-line action registration table whose closures reach into every
interactive package. This is the only place in the tree that is allowed to know
that all of them exist.

### Implementation Steps

1. Move `action_table.go` (Task 18). Its closures call into `internal/panel`,
   `internal/editor`, `internal/viewer`, `internal/dialog`, `internal/cmdline` and
   `internal/term` — legal here and nowhere else, because `internal/app` is the
   only package the dependency rules let import every layer 0-3 package.
2. Move `framework_actions.go` whole. The graph shows its 18 dependency-free
   functions have no external callers; they are `Handler:` values, and the seven
   that touch `PanelsFrame`/`QueueFrame` are workspace management, which belongs
   here. This is why Phase 4 declined to split it.
3. Move the remaining `actions.go` handlers. After Tasks 20, 24, 25, 29, 32 and 33
   took their slices, what is left is the 61 view-bound handlers plus
   `hostConsoleLogFallback`, the single `*PanelsFrame` method
   (`actions.go:2194`). Convert that method to a plain function taking the frame,
   per the project decision on cross-package methods — its type now lives in
   `internal/panel` and the method cannot follow it here.
4. Move the bootstrap: `startup_backend.go`, `startup_settings.go` (whatever Task
   25 did not take), `runtime_mode.go`, `debug_log.go`, `hang_dump_unix.go` /
   `hang_dump_windows.go`, `detach_unix.go` / `detach_windows.go`,
   `child_env.go`, `nested_input_other.go` / `nested_input_windows.go`,
   `console_ctrl_handler_other.go` / `console_ctrl_handler_windows.go`,
   `host_input_modes*.go`, `pe_subsystem.go`, `libc_default.go` / `libc_musl.go`
   (`//go:build linux && goffi_musl` and its negation), `title*.go`,
   `arkanoid.go`, `ai_chat_panel.go`, `vtvibe_host.go`, `vtvibe_ap.go`,
   `sheet_actions.go`, `sheet_dialogs.go`, `sheet_frame.go`, `sheet_palette.go`,
   `static_direct_actions.go`, `external_tools.go`, `highlight_files.go`,
   `farmenu_file.go`, `far2l_auth.go` and `async_buffer.go`. Score each with the
   gate first — several will turn out to belong to a package that already exists,
   and moving them there is a better answer than parking them in `app`.

   `main.go` is not exempt. It holds startup and session logic that is not
   wiring — `nestedInputMode`, `shouldTryGui`, `shouldPersistGUIWindowSize`,
   `startupDirs`, `startupDirsFor`, `startupDirArgs`, `rememberStartupDirs`,
   `sudoDispatcherPath`, `sudoStartupMode`, `LoadSession`, `SaveSession`,
   `mergeWorkspaceSessionSave`, `getSessionIniPath`, `formatVersionSHA`,
   `configureUnicodeInput` — and Task 37's single-digit file count is reachable
   only if that logic moves here. Its tests follow (`nested_input_mode_test.go`,
   `should_try_gui_test.go`, `session_test.go`, `startup_dir_test.go`,
   `sudo_dispatcher_args_test.go`, `autosave_settings_test.go`,
   `unicode_input_test.go`): nothing can import `package main`, so a test of
   `main.go` code can live nowhere else.

   **This list is closed.** There is no "and whatever else remains": Task 43
   assigned every `cmd/f4` file to a wave, so anything still sitting here that is
   not on this list is a Task 43 miss to be fixed there, not a file to park in
   `app`. Re-run Task 43 step 1's two `comm` commands before starting; both must
   come back empty.

   Note there is no `attributes.go` in `cmd/f4` — only `attributes_dialog.go` and
   its `_unix`/`_windows` pair, all three already taken by Task 32.
5. `internal/app` owns the process-wide startup calls Phase 1 made explicit:
   `StartQueueWorker()` (Task 5) and `action.Localize = i18n.Msg` (Tasks 21, 24).
   Both move from `cmd/f4/main.go` into the app's `New` / `Run`.
6. Rename to the topic convention: `app.go`, `loop.go`, `bootstrap.go`,
   `bootstrap_unix.go`, `bootstrap_windows.go`, `actions_table.go`,
   `actions_framework.go`, `workspace.go`, `debug.go`, `title.go`,
   `title_unix.go`, `title_windows.go`.
7. Add `"internal/app": 4` to the auditor's layer map. Rule 3 — nothing below
   layer 4 imports `internal/app` — now has something to check, and it must pass
   with **no exemptions**. An exemption here means a wave left an upward edge
   behind, and that is the failure this whole ordering exists to prevent.
8. Move `cmd/f4/action_table_order_test.go` — `TestActionOrderIsStable` and its
   golden slice, split out under that name in Task 21 step 6 — to `internal/app`,
   where the table now lives. The golden slice is unmodified since Task 3.

### Required Interfaces and Contracts

```go
// internal/app — the only place that knows every subsystem exists.
func New(cfg *config.Config, fs vfs.FileSystem, host *plughost.Host,
    t *term.Terminal, left, right *panel.Panel) *App

func (a *App) Run(ctx context.Context) error
```

- `internal/app` may import every layer 0-3 package. No layer 0-3 package may
  import it. Shared state flows down through constructor arguments, never up
  through an import.
- Registration order is unchanged: `TestActionOrderIsStable` is the proof, and it
  travels with the table.
- No `init()` in `internal/app` has a side effect beyond assignment. The queue
  worker starts from `Run`, not from import.
- Nothing mutates a struct after construction that another goroutine already
  reads. `config.App` is exempt by the stated rule: it is a plain package-level
  value in a layer-0 leaf.

### Error Handling and Logging

- `Run` returns an error; `main` reports it as
  `fmt.Fprintf(os.Stderr, "f4: %v\n", err)` and exits non-zero. That is the
  project's fatal-path convention and it must survive the move verbatim.
- `debug_log.go` keeps `VTUI_DEBUG` as the only diagnostic channel; `--debug` and
  `--log=1` still set it up. Add no logging framework and no stdout writes —
  stdout is the rendered UI.

### Tests

The 52 files Task 43's roster lists for `internal/app` move; after this commit
nothing remains in `cmd/f4` except the four module-wide auditors and their
`TestMain`. Among them: `framework_actions_test.go`, `actions_test.go`,
`arkanoid_test.go`, `ai_chat_panel_test.go`, `sheet_actions_test.go`,
`sheet_frame_test.go`, `vtvibe_host_test.go`, `startup_backend_test.go`, the
seven `main.go` tests named in step 4, the four `cloudfox_real_*_test.go`
end-to-end tests, the five registry-driven tests
(`action_copy_window_title_test.go`, `action_shortcut_conflict_test.go`,
`action_menu_visibility_test.go`, `command_palette_menu_test.go`,
`folder_history_actions_test.go`) and the handler-driven scenario tests the
multi-package table hosts here (`attributes_test.go`, `bom_test.go`,
`delete_trash_test.go`, `editor_binary_open_test.go`,
`command_palette_dynamic_test.go`, `history_hint_test.go`,
`folder_history_navigation_test.go`, `issue821_test.go`,
`updater_issue635_test.go`, `updater_repro_test.go`). **Not**
`action_registry_test.go` (it went to `internal/action` in Task 21), **not**
`macro_test.go` (Task 28), **not** `workspace_session_test.go` and
`workspace_routing_test.go` (panel tests, Task 34).

```
go test ./internal/app/...
go test -race -shuffle=on -timeout 10m ./internal/app/...
```

### Acceptance Criteria

- `go test ./cmd/f4 -run '^TestArchitecture'` passes, with rule 3 active and no
  exemption list.
- `TestActionOrderIsStable` passes in `internal/app` with its golden slice
  unmodified since Task 3.
- No `init()` in `internal/app` starts a goroutine or touches the filesystem.
- `grep -rn 'internal/app' --include='*.go' . | grep -v '^./cmd/f4\|^./internal/app'`
  returns nothing.

### Verification

- `go test ./cmd/f4 -run '^TestArchitecture' -v`
- Expected result: five subtests pass, rule 3 included.
- `go test ./internal/app -run '^TestActionOrderIsStable' -v` and
  `git diff --stat` on the golden file.
- Expected result: `--- PASS`, empty diff.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.

---

## Task 37: Reduce `cmd/f4` to the entry point

### Intent

`cmd/f4` becomes what the target layout says it is: the Composition Root and
nothing else. The four module-wide auditors stay because a per-package copy of
any of them would see only its own subtree.

### Implementation Steps

1. Confirm what remains and remove anything that does not belong:
   - `main.go` — flags, startup mode selection (terminal, GUI backend,
     `--update`, plugin scaffolding), and the construction of `app.New(...)`.
   - No test of the wiring itself exists today (`main_test.go` does not exist);
     one written later lives here. The `TestMain` that remains is the five-line
     wrapper around `testutil.Main` the auditors need.
   - `command_palette_coverage_test.go` — the module-wide palette auditor. Its 42
     keys are qualified symbols after Task 2 and its file→target-package map now has
     an entry per extracted package. **This is the commit where the map's `cmd/f4
     → main` entry becomes vestigial**; remove it if nothing audited remains in
     `main`.
   - `architecture_test.go` — the module boundary auditor.
   - `frame_manager_capture_test.go` — the third module-wide auditor: it parses
     every production file, through the palette auditor's
     `commandPaletteParseProductionGo`, for background work that captures
     `vtui.FrameManager`. It stays with the parser it shares.
   - `hardcoded_strings_test.go` — the fourth: `hardcode.Scan` over the module
     root against `tools/hardcoded_baseline.txt`, the L10N CI gate. Its
     `moduleRootDir` helper is `testutil.ModuleRootDir` after Task 9.
   - `rsrc_windows_amd64.syso`, `rsrc_windows_arm64.syso` — the toolchain links
     `.syso` only from the directory of the package being built, so they stay even
     though the icon code left in Task 27.
2. `.github/actions/affected-packages/action.yml` lines 50-54 and 76-77 map every
   path under `cmd/f4/` to the single unit `cmd/f4`; the comment at lines 64-67
   ("cmd/f4's other files always do because that package embeds non-Go
   resources") explains the second case. That was true of a 345-file flat package
   and is false of a package holding `main.go`. Remove the special
   case so the action computes affected packages normally. Verify against a
   synthetic diff that touches one `internal/` package and confirm the action
   reports that package alone.
3. Sweep the last references: `grep -rn 'cmd/f4/' . --exclude-dir=.git` should
   return only the build invocations (`go build ./cmd/f4` at `build.yml:170`,
   `174`, `400`, `403`, `579`, `808`, `810`), the `.syso` cache paths
   (`build.yml:96-97`), `tools/icons/main.go:160`, and `README.md:233`/`241`.
   Every one of those is correct and must stay.
4. Update `README.md:237` — "If `cmd/f4/assets/icon/f4.svg` is changed" — to the
   new icon location from Task 27, if Phase 6 did not already.

### Required Interfaces and Contracts

- `cmd/f4` is `package main`. Nothing can import it and nothing should want to;
  the auditor's rule 2 enforces the second half.
- `go build ./cmd/f4` and `go build -ldflags="-H windowsgui" ./cmd/f4` continue to
  produce the two binaries CI expects at the same paths.
- The `.syso` files are linked by name from this directory. Do not rename them and
  do not move them.
- The palette auditor's 42 keys stay 42. If a key's subject genuinely disappeared,
  that is a finding to report, not a number to adjust.

### Error Handling and Logging

`main` keeps the fatal-path convention exactly:

```go
if err := application.Run(context.Background()); err != nil {
    fmt.Fprintf(os.Stderr, "f4: %v\n", err)
    os.Exit(1)
}
```

Nothing else in `cmd/f4` reports errors.

### Tests

```
go test ./cmd/f4/...
go build ./cmd/f4 && ./f4 --version
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /dev/null ./cmd/f4
```

The `--version` run is the cheapest end-to-end proof that the composition root
still wires a working binary.

### Acceptance Criteria

- `ls cmd/f4/*.go | grep -v _test | wc -l` is a single-digit number.
- `ls cmd/f4/*.syso | wc -l` returns `2`.
- The Windows build links the icon and version resource — check with
  `GOOS=windows GOARCH=amd64 go build -o /tmp/f4.exe ./cmd/f4` and confirm the
  binary is larger than a `.syso`-less build.
- `affected-packages` reports a single `internal/` package for a single-package
  diff.

### Verification

- `go build ./cmd/f4 && ./f4 --version`
- Expected result: the version string, exit 0.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.
- `go test ./cmd/f4 -run '^TestArchitecture|^TestCommandPalette' -v`
- Expected result: both pass; 42 palette keys.

---

## Phase Risks and Mitigations

- **Risk:** `internal/app` becomes the new flat package — everything unclaimed by
  a wave is parked there.
  **Mitigation:** Task 43 assigns every `cmd/f4` file to a wave before the waves
  start, so nothing arrives here unclaimed; Task 36 step 4's roster is closed and
  requires scoring each remaining file with the gate. A file that belongs to an
  existing package goes there instead. The measure of success is that
  `internal/app` holds the loop and the table, not 80 files.
- **Risk:** rule 3 fails and is silenced with an exemption list.
  **Mitigation:** Task 36 step 7 forbids exemptions and names what a failure means
  — an upward edge a wave left behind.
- **Risk:** the `.syso` files are moved or renamed with the icon code and the
  Windows binary silently loses its icon and version resource.
  **Mitigation:** Task 37's acceptance criteria check both the file count and the
  linked binary size.
- **Risk:** removing the `affected-packages` special case makes CI skip a package
  that a `cmd/f4` change really does affect.
  **Mitigation:** Task 37 step 2 requires a synthetic-diff check before the change
  is trusted.

## Phase Completion Checklist

- Every Task 36-37 satisfies its acceptance criteria.
- `cmd/f4` holds `main.go`, the four module-wide auditors, their `TestMain` and
  two `.syso` files.
- `go test ./cmd/f4 -run '^TestArchitecture'` passes with rule 3 active and no
  exemptions.
- `./f4 --version` runs from a fresh build.
- `go test -timeout 25m ./...` matches the Task 1 baseline.
- `index.md` task checkboxes 36-37 are ticked.

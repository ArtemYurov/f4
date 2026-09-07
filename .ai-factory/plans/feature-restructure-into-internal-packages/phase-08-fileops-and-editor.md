# Phase 8: File Operations and the Editor

Plan: [index.md](index.md)
Tasks: 32-33
Depends on: Phase 7, plus Task 5 (Task 32) and Task 14 (Task 33)

## Objective

`internal/fileops` (13 outbound) and `internal/editor` (23) leave `cmd/f4`. After
this phase the only large subsystems still inside the flat package are the panels,
the command line and the composition root.

## The Extraction Gate

Per **(file, destination)** pair. Only `PanelsFrame`, `FileSystemPanel`,
`pluginPanelInstance` (→ `internal/panel`, Task 34) and `CommandLine`
(→ `internal/cmdline`, Task 35) are still "later" types at this point. `EditorView`
is a later type for Task 32 and this wave's own type for Task 33.

```
for t in PanelsFrame FileSystemPanel pluginPanelInstance CommandLine EditorView; do
  printf "%-22s %s\n" "$t" "$(grep -c "\b$t\b" cmd/f4/<file>.go)"
done
```

Non-zero for a later type → the file does not move whole. Move by `//go:build`
line, never by filename. Take the `_test.go` files Task 43 assigns to this wave —
not the same-named neighbours. Rename to `<topic>.go` /
`<topic>_<aspect>.go`. Export only what has an external caller. One line to
`architecture_test.go`'s layer map, one to the palette auditor's file→target-package
map. Close every `docs/` reference in the same commit.

## Current-Code Evidence

| File | Gate | Reading |
|---|---|---|
| `file_ops.go` | `PanelsFrame` ×5 | resolve five references |
| `file_op_dialog.go`, `file_op_tracker.go`, `queue_manager.go`, `atomic_file.go`, `file_state.go`, `compare_folders.go` | 0 | move whole |
| `path_identity.go` | 0 | callers are `file_ops.go` (here) and `panels_frame.go`/`file_panel.go` (panel, later) — panel → fileops is a legal layer-3 → layer-1 edge |
| `queue_manager.go` | 0 | its import-time goroutine was removed in Task 5 |
| `editor_view.go` | `EditorView` ×125 | own type — moves whole |
| `editor_base64.go`, `editor_fade.go`, `editor_find_all.go`, `editor_grapheme.go`, `editor_index_status.go`, `editor_multicursor.go`, `editor_replace_confirm.go`, `editor_save_as.go`, `editor_search_remote.go`, `editor_status.go`, `editor_wrap_safety.go` | `EditorView` only | move whole |
| `mapped_file.go` | `EditorView` ×2 | own destination's type — the file's own comment calls it the piece table's original buffer |
| `mapped_file_unix.go`, `mapped_file_windows.go` | 0 | move with it, by tag |
| `colorer_async.go`, `colorer_plugin.go` | 3 each, all `EditorView` | move whole |
| `text_editor_bridge.go` | `PanelsFrame` — `var _ vfs.TextEditorHost = (*PanelsFrame)(nil)` at `:16` | **not editor** — the assertion binds it to panel |
| `visren_editor_bridge.go` | `PanelsFrame` | same |

`EditorView` has 186 methods across 15 non-test files (counted receiver-anchored, `grep -hE '^func \([a-z]+ \*EditorView\)'`; the looser `grep 'func (.*EditorView)'` returns 187/16 by catching a function that merely takes the type). Go requires a type's
methods in the type's package, so all 15 land in `internal/editor` — that is the
package's size, not a choice.

## Files to Change

| Path | Action | Required change |
|---|---|---|
| `internal/fileops/` | create | Copy/move/delete, jobs, comparison, path identity |
| `internal/editor/` | create | The F4 editor over `internal/piecetable` |
| `cmd/f4/architecture_test.go` | modify | Two layer-map entries |
| `docs/` | modify | File-operation and editor pages |

---

## Task 32: Extract `internal/fileops`

### Intent

Copy, move, delete, the background job queue, folder comparison and the
attributes dialogs. Thirteen outbound edges. The `fileops ↔ term` cycle this wave
would otherwise hit was pre-empted in Task 30, which pulled `clipboard.go`,
`clipboard_async.go` and `background_jobs.go` into `internal/term`.

### Implementation Steps

1. Move the zero-score files whole: `file_op_dialog.go`, `file_op_tracker.go`,
   `queue_manager.go` (movable only because Task 5 took the goroutine launch out of
   its `init()`), `atomic_file.go`, `file_state.go`, `file_mask.go`,
   `compare_folders.go`, `archive_index.go`
   (`//go:build !dragonfly && !netbsd && !solaris && !illumos`),
   `archive_index_fallback.go` (`dragonfly || netbsd || solaris || illumos`),
   `attributes_dialog.go`, `attributes_dialog_unix.go`,
   `attributes_dialog_windows.go`, `fuse_mount_action.go`, `fuse_mount_list.go`.
   Score each before moving; the list is a starting roster, not a verdict.
2. `file_ops.go` scores `PanelsFrame` ×5. Resolve each: a copy operation that
   refreshes a panel afterwards belongs to the panel, not to fileops. Leave those
   five call sites behind and give `internal/fileops` a completion callback the
   caller supplies.
3. Take `path_identity.go` (gate 0). Its two functions —
   `normalizedURIIdentity`, `isPersistentURIPath` — are called from `file_ops.go`
   here and from `panels_frame.go` / `file_panel.go` in the panel wave. Panel →
   fileops is layer 3 → layer 1 and legal, and this keeps two exported functions
   off the public `vfs` surface.
4. `fuse_mount_action.go` has **two** `init()` blocks and `fuse_mount_list.go` one;
   all three register actions. After Phase 4 they call `action.RegisterAction`.
   Confirm `TestActionOrderIsStable` still passes — this is the wave most likely to
   reorder the menu, because it moves registration files across a package boundary.
5. `compare_folders.go` no longer declares `compareOptions` (Task 4 moved the type
   to `config.go`); it uses `config.CompareOptions`. Confirm the import.
6. Rename to the topic convention: `ops.go`, `ops_dialog.go`, `tracker.go`,
   `queue.go`, `atomic.go`, `state.go`, `mask.go`, `compare.go`, `archive_index.go`,
   `archive_index_fallback.go`, `attributes.go`, `attributes_unix.go`,
   `attributes_windows.go`, `fuse_mount.go`, `fuse_list.go`, `identity.go`.
7. Add `"internal/fileops": 1` to the auditor's layer map.

### Required Interfaces and Contracts

- `internal/fileops` may import `internal/config`, `internal/i18n`,
  `internal/theme`, `internal/toast`, `internal/action`, `internal/numeric`,
  `internal/fusefs`, `vfs`. Not `internal/panel`, `internal/editor`,
  `internal/viewer`, `internal/cmdline`, `internal/app`, and — after Task 30 —
  not `internal/term` either.
- The queue's public shape is unchanged: `GlobalQueueManager` plus
  `StartQueueWorker()` from Task 5. It stays a package-level value in a layer-1
  package; that is permitted, but the worker is started by the composition root.
- Error strings on the copy/move/delete paths become dialog text and are covered
  by the disabled `ST1005`. Reword nothing.
- `normalizedURIIdentity` deliberately does not `filepath.Clean` or decode:
  provider IDs and escaped dot segments are opaque persistent identity. Carry that
  comment across — it is the invariant, and it is not visible from the code.

### Error Handling and Logging

- Every operation already wraps with context and compares with `errors.Is`.
  Preserve the sentinel errors declared at package level.
- Deliberately ignored errors stay `_ = f()`; `errcheck` and `gosec` run in CI.
- No logging is added. A failed file operation reaches the user as a dialog.

### Tests

The 19 files Task 43's roster lists for `internal/fileops` move, among them
`file_ops_test.go`, `file_mask_far2l_test.go`, `queue_manager_test.go`,
`compare_folders_test.go`, `dialog_reporter_test.go`, `file_ops_safety_test.go`,
`file_state_key_test.go`, `issue149_test.go` and `issue815_test.go`. **Not**
`attributes_test.go`: it drives `actionFileAttributes` on a mock frame (41 panel
references) and goes to `internal/app` in Task 36; this wave exports the four
`showAttributes*` functions it needs (Task 43's multi-package table). **Not**
`delete_trash_test.go` either, for the same reason: it drives `actionDelete`,
and this wave exports `calculateDeleteStats` and `deletePathWithDisposition`.
`queue_manager_test.go` gained an explicit `StartQueueWorker()` in Task 5; confirm
it still compiles against the exported name.

```
go test ./internal/fileops/...
go test -race ./internal/fileops/...
for t in linux/amd64 windows/amd64 dragonfly/amd64 netbsd/amd64 solaris/amd64 illumos/amd64; do
  extra=(); case $t in freebsd/*|netbsd/*) extra=(-gcflags=github.com/go-webgpu/goffi/internal/fakecgo=-std);; esac
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build "${extra[@]}" ./... || echo "FAIL $t"
done
```

The six targets are chosen for `archive_index.go`'s tag split, which is the one
platform decision in this package.

### Acceptance Criteria

- `go list -f '{{join .Imports "\n"}}' ./internal/fileops | grep -E 'internal/(panel|editor|viewer|cmdline|term|app)'`
  returns nothing.
- `TestActionOrderIsStable` passes with its golden slice unmodified.
- The cross-compile loop prints no `FAIL`.

### Verification

- `go test ./cmd/f4 -run '^TestActionOrderIsStable' -v`
- Expected result: `--- PASS`; the golden slice in
  `cmd/f4/action_table_order_test.go` (Task 21 step 6) is unmodified.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.

---

## Task 33: Extract `internal/editor`

### Intent

The F4 editor on top of `internal/piecetable`. Twenty-three outbound edges — the
largest of the interactive subsystems apart from panel and cmdline. `EditorView`
carries 186 methods across 15 files, and Go requires every one of them in this
package.

### Implementation Steps

1. Move `editor_view.go` (`EditorView` ×125, its own type) and the eleven
   `editor_*.go` aspect files, all of which score only on `EditorView`.
2. Move `mapped_file.go`, `mapped_file_unix.go`, `mapped_file_windows.go`. The
   graph and the file's own comment agree: it is the piece table's original
   buffer, not a filesystem utility. Read the two build tags from the files.
3. Move `colorer_async.go` and `colorer_plugin.go` (3 `EditorView` references
   each), plus `colorer_downloader.go`, which Task 43 assigns here. **This task
   depends on Task 14**: `colorer_plugin.go:156` reads `colorer.RadiolaHRD`, which
   does not exist until Task 14 creates `internal/colorer`. Its second consumer,
   `colorer_plugin_test.go:56`, travels in the same commit; confirm both imports
   resolve.
4. Move `external_editor_process_unix.go` (`!windows`) and
   `external_editor_process_windows.go` (`windows`), plus
   `configuredExternalEditorCommand` from `actions.go` (Phase 4's table — it reads
   `config.App` five times and nothing else).
5. Take `editorHeaderIsBinary` from `actions.go` and `semantic.go`'s five
   `*EditorView` methods as `internal/editor/view_semantic.go`.
6. **`text_editor_bridge.go` and `visren_editor_bridge.go` do not come here**
   despite their names. `text_editor_bridge.go:16` asserts
   `var _ vfs.TextEditorHost = (*PanelsFrame)(nil)` — the interface is satisfied by
   the panel type, so the file belongs to `internal/panel` (Task 34).
7. `editor_view.go` calls `numeric.ReleaseHeavyMemory`; confirm the import.
8. Rename to the topic convention: `view.go`, `view_semantic.go`, `base64.go`,
   `fade.go`, `findall.go`, `grapheme.go`, `index_status.go`, `multicursor.go`,
   `replace_confirm.go`, `save_as.go`, `search_remote.go`, `status.go`,
   `wrap_safety.go`, `buffer_mapped.go`, `buffer_mapped_unix.go`,
   `buffer_mapped_windows.go`, `colorer.go`, `colorer_async.go`,
   `external_unix.go`, `external_windows.go`.
9. Add `"internal/editor": 3` to the auditor's layer map, and `editor` to the
   palette auditor's file→target-package map.

### Required Interfaces and Contracts

- `internal/editor` may import `internal/piecetable`, `internal/textlayout`,
  `internal/colorer`, `internal/config`, `internal/i18n`, `internal/theme`,
  `internal/keymap`, `internal/numeric`, `internal/toast`, `internal/history`,
  `internal/action`, `internal/dialog`, `internal/fileops`, `vfs`. Not
  `internal/viewer` — that cycle was removed in Task 29 by moving `top_bar.go`,
  `file_title.go` and `url_links.go` to the viewer. Not `internal/panel`,
  `internal/cmdline`, `internal/app`.
- `EditorView`'s method set and names are unchanged; Far-derived names stay.
- `(*EditorView).ProcessKey` is an audited symbol: its palette-auditor key becomes
  `editor.(*EditorView).ProcessKey`.
- The piece-table contract is unchanged. `mapped_file.go` provides the original
  buffer; its mmap semantics per platform are load-bearing and its build tags are
  copied verbatim.

### Error Handling and Logging

- Save failures reach the user as a dialog and are wrapped with the path. Keep the
  strings.
- `mapped_file_*.go`'s mmap failure path already falls back to a read; preserve it,
  including the `_ = f()` form on the unmap.
- No logging is added.

### Tests

The 35 files Task 43's roster lists for `internal/editor` move:
`editor_view_test.go`, `editor_find_all_test.go`, `editor_target_line_test.go`,
the aspect files' neighbours, the twenty-one scenario tests named
`editor_*_test.go` without a same-named source, `colorer_plugin_test.go`,
`mapped_file_test.go`, `external_editor_process_unix_test.go`,
`external_editor_test.go` (its subject, `configuredExternalEditorCommand`, comes
here in step 4) and `goto_test.go` (dialog exports `parseGotoOffset` and
`showGotoOffsetDialog` for it). **Not** `editor_binary_open_test.go`: Task 43's
ruling hosts it in `internal/app` — it drives `showEditor` and
`findOpenedEditor` — and this wave exports `awaitOffsetAsync`, `cancelColorer`,
`indexIsComplete` and `newEditorView` for it. `editor_save_inplace_test.go` comes
here, but its one case that touches `async_buffer.go`'s `prewarm` splits out to
`internal/app`. Several call `testutil.SwapFrameManager`; verify the drains they
pass are still correct now that `waitForAsyncClipboard` lives in
`internal/term`.

```
go test ./internal/editor/...
go test -race -shuffle=on ./internal/editor/...
for t in linux/amd64 darwin/arm64 windows/amd64 freebsd/amd64 solaris/amd64; do
  extra=(); case $t in freebsd/*|netbsd/*) extra=(-gcflags=github.com/go-webgpu/goffi/internal/fakecgo=-std);; esac
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build "${extra[@]}" ./internal/editor/... || echo "FAIL $t"
done
```

### Acceptance Criteria

- `go list -f '{{join .Imports "\n"}}' ./internal/editor | grep -E 'internal/(viewer|panel|cmdline|app)'`
  returns nothing.
- All 15 `EditorView` method files are in `internal/editor`;
  `grep -rlnE '^func \([a-z]+ \*EditorView\)' cmd/f4/` returns nothing.
- `ls cmd/f4/text_editor_bridge.go cmd/f4/visren_editor_bridge.go` still succeed.
- The palette auditor still has 42 keys and passes.

### Verification

- `go test ./internal/editor/... ./cmd/f4/... -count=1`
- Expected result: `ok`.
- `go test ./cmd/f4 -run '^TestArchitecture'`
- Expected result: `ok`, acyclic, two new layer entries.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.

---

## Phase Risks and Mitigations

- **Risk:** `text_editor_bridge.go` and `visren_editor_bridge.go` are filed with
  the editor on their names, dragging `PanelsFrame` into a layer-3 package that
  panel later has to import back.
  **Mitigation:** Task 33 step 6 names the interface assertion at
  `text_editor_bridge.go:16` that decides it.
- **Risk:** moving `fuse_mount_action.go` and the other registration files across
  a package boundary reorders the menu silently.
  **Mitigation:** Task 3 made the order explicit and `TestActionOrderIsStable` is
  in this phase's acceptance criteria for both tasks.
- **Risk:** `file_ops.go`'s five `PanelsFrame` references are resolved by importing
  `internal/panel` from `internal/fileops`, inverting layers 1 and 3.
  **Mitigation:** Task 32 step 2 specifies the completion-callback shape instead.
- **Risk:** `archive_index.go`'s four-platform negative tag is transcribed
  incorrectly and one BSD loses archive indexing silently.
  **Mitigation:** the six-target cross-compile loop in Task 32's Tests section
  covers both sides of the split.

## Phase Completion Checklist

- Every Task 32-33 satisfies its acceptance criteria.
- `internal/fileops` imports no interactive subsystem and no `internal/term`.
- `internal/editor` and `internal/viewer` do not import each other.
- `TestActionOrderIsStable` passes in both commits.
- `go test -timeout 25m ./...` matches the Task 1 baseline.
- `index.md` task checkboxes 32-33 are ticked.

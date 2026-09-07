# Phase 7: Viewer, Terminal and Media

Plan: [index.md](index.md)
Tasks: 29-31
Depends on: Phase 6

## Objective

`internal/viewer` (9 outbound), `internal/term` (12) and `internal/media` (10)
leave `cmd/f4`. Term goes **ahead of** media even though its outbound count is
higher, because six of media's ten edges point at term.

This phase resolves the largest single body of misfiled code in the tree: eleven
files whose names say GUI, command line or graphics, and whose callers say
terminal.

## The Extraction Gate

Per **(file, destination)** pair. Count only references to types whose package is
extracted **later** than this file's destination:

| Type | Lands in | Extracted at |
|---|---|---|
| `ViewerView` | `internal/viewer` | Task 29 |
| `TerminalView` | `internal/term` | Task 30 |
| `EditorView` | `internal/editor` | Task 33 |
| `PanelsFrame`, `FileSystemPanel`, `pluginPanelInstance` | `internal/panel` | Task 34 |
| `CommandLine` | `internal/cmdline` | Task 35 |

```
for t in PanelsFrame FileSystemPanel EditorView ViewerView TerminalView CommandLine coreAPI pluginPanelInstance; do
  printf "%-22s %s\n" "$t" "$(grep -c "\b$t\b" cmd/f4/<file>.go)"
done
```

Non-zero for a later type → the file does not move whole: the offending function
stays with its view, or becomes a plain function taking the type. Move by
`//go:build` line, never by filename. Take the `_test.go` files Task 43 assigns to
this wave — not the same-named neighbours. Rename to
`<topic>.go` / `<topic>_<aspect>.go`. Export only what has an external caller. One
line to `architecture_test.go`'s layer map, one to
`command_palette_coverage_test.go`'s file→target-package map when this wave owns an
audited symbol. Close every `docs/` reference in the same commit.

## Current-Code Evidence

Per-type gate scores, measured on the current tree. The columns are `PanelsFrame`
/ `FileSystemPanel` / `EditorView` / `ViewerView` / `TerminalView` / `CommandLine`:

| File | PF | FSP | EV | VV | TV | CL | Reading |
|---|---|---|---|---|---|---|---|
| `viewer_view.go` | 0 | 0 | 0 | 38 | 0 | 0 | own type only — moves whole |
| `viewer_backend.go`, `viewer_text.go`, `disasm.go`, `word_nav.go` | 0 | 0 | 0 | 0 | 0 | 0 | move whole |
| `top_bar.go`, `file_title.go`, `url_links.go` | 0 | 0 | 0 | 0 | 0 | 0 | reassigned to viewer by the graph; removes the editor↔viewer cycle |
| `ansi_parser.go` | 0 | 0 | 0 | 0 | 2 | 0 | own destination's type — moves whole |
| `kitty_graphics.go`, `sixel_decode.go` | 0 | 0 | 0 | 0 | 0 | 0 | move whole |
| `kitty_placements.go` | 0 | 0 | 0 | 0 | 14 | 0 | **only** `TerminalView` — moves whole to term |
| `sixel_terminal.go` | 0 | 0 | 0 | 0 | 5 | 0 | same |
| `far2l_image.go` | 0 | 0 | 0 | 0 | 4 | 0 | same |
| `graphics_compat.go`, `graphics_probe_decision.go`, `graphics_probe_windows.go` | 0 | 0 | 0 | 0 | 0 | 0 | move whole |
| `command_runner*.go`, `shell_mode.go`, `wine_probe*.go` | 0 | 0 | 0 | 0 | 0 | 0 | called by `pty_*` — term, not cmdline |
| `clipboard.go`, `clipboard_async.go`, `background_jobs.go` | 0 | 0 | 0 | 0 | 0 | 0 | pulled into term to avoid a `fileops ↔ term` cycle |
| `background_jobs_window.go` | 1 | 0 | 0 | 0 | 0 | 0 | one reference to resolve |
| `terminal_view.go` | **1** | 0 | 0 | 0 | 79 | 0 | one `PanelsFrame` reference to resolve |
| `terminal_workspace.go` | 2 | 0 | 0 | 0 | 0 | 0 | resolve |
| `session_unix.go` | 6 | 0 | 0 | 0 | 0 | 0 | resolve |
| `console_passthrough.go` | **16** | 0 | 0 | 0 | 0 | **3** | **not term** — panel/cmdline code |
| `pty_interface.go`, `terminal_redraw.go` | 0 | 0 | 0 | 0 | 0 | 0 | move whole |
| `image_*.go`, `audio_*.go`, `video_*.go` | 0 | 0 | 0 | 0 | 0 | 0 | move whole |
| `player_panel.go` | 0 | **3** | 0 | 0 | 0 | 0 | resolve — panel is extracted after media |

Build-tag traps in this phase, verified from the files themselves:
`pty_unix.go` is `//go:build linux`, `solaris_pty.go` is `//go:build !windows`
and holds no PTY code, `pty_bsd.go` is `freebsd || dragonfly`, `pty_ptm_*.go`
split on netbsd/openbsd, `session_unix.go` and `session_windows.go` split on
`!windows`/`windows`. There are 28 distinct tag expressions in `cmd/f4`.

## Files to Change

| Path | Action | Required change |
|---|---|---|
| `internal/viewer/` | create | F3 viewer, hex, disasm, top bar, titles, URL links |
| `internal/term/` | create | pty, console host, ANSI parser, kitty/sixel, clipboard |
| `internal/media/` | create | Image, audio and video decode and preview |
| `cmd/f4/architecture_test.go` | modify | Three layer-map entries |
| `docs/TERMINAL.md`, `docs/CONPTY.md` and neighbours | modify | Path references |

---

## Task 29: Extract `internal/viewer`

### Intent

The F3 viewer with its hex and disassembly modes. Nine outbound edges. The graph
reassigns three files here against their names — `top_bar.go`, `file_title.go`,
`url_links.go` — and taking them is what removes the `editor ↔ viewer` cycle: left
with the editor, each side needs the other.

### Implementation Steps

1. Move the zero-score files whole: `viewer_backend.go`, `viewer_text.go`,
   `disasm.go`, `word_nav.go`, `top_bar.go`, `file_title.go`, `url_links.go`.
2. `viewer_view.go` scores 38, all of them `ViewerView` — its own type, landing in
   this package. It moves whole.
3. Take the three viewer-search functions stranded in `actions.go` (Phase 4's
   table): `viewerSearchOffset`, `viewerSearchMatch`, `readViewerSearchData`. All
   three take `*ViewerBackend` and use nothing else.
4. Take `semantic.go`'s three `*ViewerView` methods as
   `internal/viewer/view_semantic.go`. The file is split across five packages;
   Task 34 owns the split and this is its viewer slice — coordinate by moving the
   slice here and leaving the rest in place until each wave claims it.
5. `url_links.go` calls `showToast`; after Phase 4 that is `toast.Show`. Confirm
   the import resolves.
6. `disasm.go` uses `numeric.NonNegativeUint64`.
7. Rename to the topic convention: `view.go`, `view_semantic.go`, `backend.go`,
   `text.go`, `disasm.go`, `topbar.go`, `title.go`, `links.go`, `wordnav.go`.
8. Add `"internal/viewer": 3` to the auditor's layer map, and the
   `viewer` entry to the palette auditor's file→target-package map.

### Required Interfaces and Contracts

- `internal/viewer` may import `internal/config`, `internal/i18n`,
  `internal/theme`, `internal/keymap`, `internal/numeric`, `internal/toast`,
  `internal/history`, `internal/action`, `internal/textlayout`,
  `internal/piecetable`, `vfs`. Not `internal/editor`, `internal/panel`,
  `internal/cmdline`, `internal/app`.
- `ViewerView` and `ViewerBackend` keep their names.
- The viewer's `ProcessKey` is an audited symbol in
  `command_palette_coverage_test.go`; its key becomes
  `viewer.(*ViewerView).ProcessKey` through the one-line map change from Task 2.

### Error Handling and Logging

Unchanged. Read failures already wrap with context and reach the user as viewer
status text. No logging is added.

### Tests

`viewer_view_test.go`, `disasm_test.go`, `title_test.go` and
`uri_navigation_test.go` move with their subjects.
**`editor_binary_open_test.go` does not come here.** It was hedged as "if it tests
the viewer path", and Task 33's Tests section claims it outright — a file claimed
twice is claimed by nobody. Measured, it touches `FileSystemPanel` ×1 and
`EditorView` ×1 and no viewer type at all, so it is one of the five multi-package
tests Task 43 step 4 rules on. Follow that ruling; do not take it here.

Each test that moves uses `testutil.SwapFrameManager` after Task 9 — verify the
drains it passes are still the right ones.

```
go test ./internal/viewer/...
go test -race ./internal/viewer/...
```

### Acceptance Criteria

- `go list -f '{{join .Imports "\n"}}' ./internal/viewer | grep -E 'internal/(editor|panel|cmdline|app)'`
  returns nothing — the cycle is gone.
- `ls cmd/f4/top_bar.go cmd/f4/file_title.go cmd/f4/url_links.go` all fail.
- The palette auditor still has 42 keys and passes.

### Verification

- `go test ./internal/viewer/... ./cmd/f4/... -count=1`
- Expected result: `ok`, matching the Task 1 baseline.
- `go test ./cmd/f4 -run '^TestArchitecture'`
- Expected result: `ok`, acyclic.

---

## Task 30: Extract `internal/term`

### Intent

The largest wave: pty backends across nine platforms, the console host, the ANSI
parser, kitty and sixel graphics, clipboard and background jobs. Twelve outbound
edges, and it goes before media because six of media's ten edges point here.

**Roster size.** Steps 1-3 below name 41 non-test files. Task 43 adds eleven more
that no task named — `ttyx_probe.go`, `ttyx_probe_parse.go`, `ttyx_probe_unix.go`,
`ttyx_probe_windows.go`, `ttyx_session.go`, `terminal_log_console_other.go`,
`terminal_log_console_windows.go`, `terminal_log_vfs.go`, `console_host_windows.go`
and the two `console_overlay_*.go` (score the overlays: they may belong to
`internal/media`). That is 52 non-test files, plus whatever `_test.go` files Task
43's table assigns. Count the roster before starting and again before committing;
this is the wave where a dropped file is least likely to be noticed.

This is the wave where filenames lie the most. Eleven files whose names suggest
another package belong here because their *callers* are in `ansi_parser.go` and
`pty_*`; leaving them where their names suggest inverts the layers.

### Implementation Steps

1. Move the graph-reassigned files, each with the caller that proves it:
   - `kitty_graphics.go`, `kitty_placements.go`, `sixel_decode.go`,
     `sixel_terminal.go` — called by `ansi_parser.go` itself. Gate scores are
     `TerminalView` only (14 and 5), which lands here.
   - `command_runner.go`, `command_runner_unix.go`, `command_runner_windows.go`,
     `shell_mode.go`, `wine_probe.go`, `wine_probe_windows.go`,
     `wine_probe_other.go` — called by `pty_bsd.go`, `pty_darwin.go`,
     `solaris_pty.go`, `pty_unix.go` and `pty_ptm_*.go`. Leaving them in
     `internal/cmdline` inverts the layers: the pty layer would depend on the
     command line.
   - `graphics_compat.go`, `graphics_probe_decision.go`,
     `graphics_probe_windows.go`, `far2l_image.go` — they decide what the
     *terminal* supports, not what the GUI draws.
   - `clipboard.go`, `clipboard_async.go`, `background_jobs.go` — pulled into this
     wave specifically to avoid a `fileops ↔ term` cycle in Task 32.
2. Move the pty family by build tag, reading each `//go:build` line from the file:
   `pty_interface.go` (none), `pty_unix.go` (**`linux`**, despite the name),
   `pty_darwin.go` (`darwin`), `pty_bsd.go` (`freebsd || dragonfly`),
   `pty_bsd_freebsd.go`, `pty_bsd_dragonfly.go`, `pty_ptm.go`,
   `pty_ptm_netbsd.go`, `pty_ptm_openbsd.go`, `pty_solaris.go`,
   `pty_windows.go`, `pty_diag_unix.go`, `pty_diag_windows.go`,
   `solaris_pty.go` (**`!windows`**, and it contains no PTY code — read it before
   deciding), `solaris_streams.go`.
3. Resolve the non-zeros: `terminal_view.go` has 79 `TerminalView` (fine) and
   **one** `PanelsFrame` — resolve that single reference. `terminal_workspace.go`
   (2), `session_unix.go` (6), `background_jobs_window.go` (1) likewise.
4. **`console_passthrough.go` does not come here.** It scores `PanelsFrame` ×16
   and `CommandLine` ×3 with zero `TerminalView`: it is panel and command-line
   code that happens to mention the console. It travels with `internal/panel`
   (Task 34).
5. Take `semantic.go`'s single `*TerminalView` method as
   `internal/term/view_semantic.go`.
6. `waitForAsyncClipboard` (`clipboard_async.go:27`) is one of the two drains Task
   9 turned into caller-supplied functions. Export it and update the
   `testutil.SwapFrameManager` call sites that pass it.
7. Rename to the topic convention: `pty.go`, `pty_linux.go`, `pty_darwin.go`,
   `pty_bsd.go`, `pty_windows.go`, `ansi.go`, `kitty.go`, `kitty_placements.go`,
   `sixel.go`, `sixel_terminal.go`, `graphics_probe.go`, `graphics_compat.go`,
   `runner.go`, `runner_unix.go`, `runner_windows.go`, `shellmode.go`,
   `wineprobe.go`, `clipboard.go`, `clipboard_async.go`, `jobs.go`,
   `view.go`, `view_semantic.go`, `session_unix.go`, `session_windows.go`.
8. Add `"internal/term": 1` to the auditor's layer map, and `term` to the palette
   auditor's map.

### Required Interfaces and Contracts

- `internal/term` may import `internal/config`, `internal/i18n`,
  `internal/theme`, `internal/keymap`, `internal/numeric`, `internal/toast`,
  `internal/ttyx`, `internal/wincon`, `vfs`. Not `internal/panel`,
  `internal/cmdline`, `internal/editor`, `internal/viewer`, `internal/app`.
- Every `//go:build` line is copied verbatim from its file. This is the wave where
  a name-based move produces a build that succeeds on the developer's platform and
  fails on four others.
- `TerminalView`'s method set is unchanged; 96 methods across five files.
- The clipboard's async contract is unchanged: writes may block on far2l IPC, and
  `waitForAsyncClipboard` is what a caller uses to join them.

### Error Handling and Logging

- pty failures already report to stderr with the `f4: ` prefix during startup and
  reach the terminal view as text afterwards. Keep both paths.
- `VTUI_DEBUG` is the only diagnostic channel and `cmd/f4/debug_log.go` owns it;
  it stays in `cmd/f4` for now and moves with the composition root.
- Deliberately ignored errors stay written as `_ = f()` — `errcheck` runs in CI.

### Tests

Take the `_test.go` files Task 43 assigns to this wave, among them
`ansi_parser_test.go` (which contains one
of the two test-file `init()`s and the `mockPty` fixture), `terminal_view_test.go`
(the other), `issue863_terminal_test.go`, `clipboard_test.go`,
`shell_session_test.go`, `shell_integration_test.go` and the pty diagnostics
tests.

`ansi_parser_test.go`'s `mockPty` is used by `setupMockPanelsFrame`
(`panels_frame_test.go:794`). Since `internal/paneltest` will import
`internal/term`, export `mockPty` as `term.MockPty` in a `_test.go`-visible form —
or, simpler, move the fixture into `internal/paneltest` when Task 34 fills it.
Decide now and record it: **move `mockPty` to `internal/paneltest` in Task 34**,
leaving a local copy for term's own tests.

```
go test ./internal/term/...
go test -race -shuffle=on -timeout 5m ./internal/term/...
for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 \
         freebsd/amd64 dragonfly/amd64 openbsd/amd64 netbsd/amd64 illumos/amd64 solaris/amd64; do
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build ./... || echo "FAIL $t"
done
```

### Acceptance Criteria

- The cross-compile loop prints no `FAIL` — twelve targets, the pty family's real
  audience.
- `ls cmd/f4/console_passthrough.go` still succeeds (it is not part of this wave).
- `grep -l '//go:build' internal/term/*.go | wc -l` matches the count of tagged
  files moved.
- The race run is green: `internal/term` owns the clipboard workers.

### Verification

- The cross-compile loop above.
- Expected result: no `FAIL` line.
- `go test -race -shuffle=on -timeout 5m ./internal/term/...`
- Expected result: `ok`.
- `go test -timeout 25m ./...`
- Expected result: identical to the Task 1 baseline.

---

## Task 31: Extract `internal/media`

### Intent

Image, audio and video decode and preview — twenty-one files. Ten outbound edges,
six of which point at `internal/term`, which is why this wave follows Task 30
rather than preceding it on the raw count.

### Implementation Steps

1. Move the zero-score files whole: `image_bmp.go`, `image_qoi.go`,
   `image_decode.go`, `image_external.go`, `image_native_darwin.go`
   (`//go:build darwin`), `image_pipeline.go`, `image_preview.go`,
   `image_transform.go`, `image_view.go`, `image_gallery.go`,
   `image_slideshow.go`, `image_console_overlay.go`, `image_console_stats.go`,
   `image_x11_overlay.go`, `audio_decode.go`, `audio_engine.go`,
   `audio_engine_oto.go`, `audio_engine_stub.go`, `video_player.go`,
   `video_view.go`.
2. `player_panel.go` scores `FileSystemPanel` ×3. `internal/panel` does not exist
   until Task 34, so resolve the three references: the player's own logic stays
   here as functions taking the panel type once it exists, or the panel-facing
   glue stays behind in `cmd/f4` and moves to `internal/panel` in Task 34. Pick the
   second if the three references are a single embedding relationship — a media
   package that takes a panel type is a layer inversion waiting to happen.
3. Five of the image files have `init()` (`image_decode.go`, `image_qoi.go`,
   `image_bmp.go`, `image_external.go`, `image_native_darwin.go`). Confirm each
   only registers a decoder in a package-level map — assignment, not a side
   effect — which is what the composition-root principle permits. If any starts a
   goroutine or touches the filesystem, it gets Task 5's treatment.
4. `audio_engine_oto.go` / `audio_engine_stub.go` split on a build tag; copy it
   verbatim.
5. Rename to the topic convention: `image.go`, `image_bmp.go`, `image_qoi.go`,
   `image_decode.go`, `image_gallery.go`, `image_preview.go`, `image_view.go`,
   `audio.go`, `audio_oto.go`, `audio_stub.go`, `video.go`, `video_view.go`,
   `player.go`.
6. Add `"internal/media": 1` to the auditor's layer map.

### Required Interfaces and Contracts

- `internal/media` may import `internal/term` (six edges — image display goes
  through the terminal's graphics protocols), `internal/config`, `internal/i18n`,
  `internal/theme`, `internal/numeric`, `internal/toast`, `vfs`. Not
  `internal/panel`, `internal/editor`, `internal/viewer`, `internal/app`.
- The decoder registration maps keep their registration semantics; a duplicate
  format registration behaves as it does today.
- `CGO_ENABLED=0` holds; the audio engine's oto backend already respects it.

### Error Handling and Logging

Decode failures reach the user as a preview placeholder plus a message; keep the
strings verbatim. `image_external.go` shells out — its error wrapping names the
command, and that message is user-visible.

### Tests

`image_gallery_test.go`, `image_view_overlay_test.go` and the audio/video tests
move. Two of them call `testutil.ScreenRow` (moved in Task 9) — confirm the import.

```
go test ./internal/media/...
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./internal/media/...
```

### Acceptance Criteria

- `go list -f '{{join .Imports "\n"}}' ./internal/media | grep -E 'internal/(panel|editor|viewer|app)'`
  returns nothing.
- Every `init()` in the package is registration-only.
- The suite matches the Task 1 baseline.

### Verification

- `go test ./internal/media/... ./cmd/f4/... -count=1`
- Expected result: `ok`.
- `go test ./cmd/f4 -run '^TestArchitecture'`
- Expected result: `ok`, three new layer entries.

---

## Phase Risks and Mitigations

- **Risk:** the pty family is moved by filename and `pty_unix.go`
  (`//go:build linux`) lands in a set that assumes "unix", or `solaris_pty.go`
  (`//go:build !windows`, no PTY code) is filed with the pty backends.
  **Mitigation:** Task 30 step 2 lists both traps with their real tags, and the
  twelve-target cross-compile loop is the acceptance test.
- **Risk:** `console_passthrough.go` is filed with term because of its name,
  dragging 16 `PanelsFrame` references into a layer-1 package.
  **Mitigation:** Task 30 step 4 forbids it and the evidence table records the
  score.
- **Risk:** `internal/media` takes `player_panel.go` whole and inverts the layers
  by importing `internal/panel`.
  **Mitigation:** Task 31 step 2 requires resolving the three `FileSystemPanel`
  references and recommends leaving the glue behind.
- **Risk:** `mockPty` ends up needed by both `internal/term`'s own tests and
  `internal/paneltest`, and someone exports it from a `_test.go` file, which Go
  will not allow across packages.
  **Mitigation:** Task 30's Tests section fixes the decision now — the fixture goes
  to `internal/paneltest` in Task 34, with a local copy for term.

## Phase Completion Checklist

- Every Task 29-31 satisfies its acceptance criteria.
- The twelve-target cross-compile loop is clean.
- `internal/viewer` does not import `internal/editor`, and vice versa.
- `go test -race -shuffle=on ./internal/term/...` is green.
- `go test -timeout 25m ./...` matches the Task 1 baseline.
- `index.md` task checkboxes 29-31 are ticked.

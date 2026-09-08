# Handoff — where this stands

Written at `a1da963e`, on a clean, green tree. Everything here is either a
thing that must happen first or a thing that lives nowhere else.

## Start here: the upstream merge

`upstream/main` is **12 commits ahead** and was attempted at `a1da963e`. The
merge was aborted rather than half-resolved, because a session was ending and a
partly-merged tree is worse to inherit than a clean one. Nothing is lost; redo
it as the first action.

```
git merge upstream/main
```

Four files conflict, eight hunks. What each needs:

- **`cmd/f4/macro.go`** — modify/delete. The file is gone: Task 28 split it into
  `internal/macro/engine.go` (recording, playback, storage, the assign dialog)
  and `cmd/f4/macro_dispatch.go` (the key router). Upstream's change is four
  lines and belongs to the router. Apply it inside `macroFilter`, immediately
  after the `configuredHotkeyAction` block that resolves the palette chord:

  ```go
  if commandPaletteLegacyShortcut(currentArea, e) {
      RunAction(commandPaletteActionName)
      return true
  }
  ```

  `commandPaletteLegacyShortcut` arrives with the merge; the hunk does not
  compile before the rest of the merge is in place. Then `git rm cmd/f4/macro.go`.

- **`internal/dialog/settings_portable.go`** — four hunks, and the expensive
  one. This is `cmd/f4/portable.go` renamed by Task 25, with four names
  exported for `cmd/f4` (`SetPortableMode`, `EnsureProfileLayout`,
  `CopyProfileDir`, `MoveProfileDir`, `ShowPortableSettings`,
  `PortableProfileSubdirs`) and `Msg` qualified as `i18n.Msg`. Do **not** take
  upstream's file wholesale — it references `cmd/f4` symbols Task 25 already
  resolved differently, and taking it costs more than merging by hand. The
  method that worked last time: keep our side, then re-apply upstream's hunks
  one at a time against it.

- **`cmd/f4/action_table.go`** (2 hunks), **`cmd/f4/actions.go`** (1),
  **`cmd/f4/portable_test.go`** (1) — ordinary content conflicts. Watch for
  `internal/terminal` and `internal/media` qualifiers on our side that
  upstream's incoming code will not have.

After the merge: full suite, cross-build, `GOOS=… go vet` on four systems.

## Where the work stands

Tasks 26-31 are done and committed; phases 6 and 7 are closed. `cmd/f4` is
down from 596 files to 404. The checkboxes in `index.md` match the tree —
verified at `a1da963e`.

Next is **Task 32, `internal/fileops`**. The package already exists: the viewer
wave created it early with `state.go` and `codepage.go`, because the viewer
reads a remembered codepage and Task 24 had already found that
`codepage_state.go` could not move without `file_state.go`. Task 32 fills in
the rest as written.

## Deviations from the plan, and where each is recorded

Every one is written into the task it belongs to. This list exists so the next
session can check the record rather than rediscover it.

| Deviation | Recorded in |
|---|---|
| `internal/plughost` interface is seven methods, not five | `index.md`, the Task 26 finding |
| `api.go`, `plugin_hotkeys.go`, `plugring_ui.go`, `sqlite_actions.go` stay behind | `phase-06`, "What the wave actually found" |
| `internal/gui` is layer 2, not 1 | `phase-06` |
| `internal/term` → `internal/terminal`, layer 3 not 1 | `phase-07`, commit `a1da963e` |
| `internal/media` is layer 3, not 1 | `phase-07` |
| Nine `term` roster files could not move | `phase-07` |
| `internal/textsearch` and early `internal/fileops`, neither planned | `phase-07` |
| `fuse_mount_*.go` go to `internal/app`, not `fileops` | `phase-08`, before the wave |
| `commands.go` cannot go to `internal/cmdline` | `phase-07` |
| Package-name question for Task 44 | `phase-11`, Task 44 step 3 |

## Open tails

1. **The upstream merge above.** First thing.
2. **`attributes_dialog.go` scores `PanelsFrame` ×8.** Its name asks for
   `internal/dialog`; both dialog and fileops are layer 3, so the gate will not
   settle it. Measure on the Task 32 wave. Recorded in `phase-08`.
3. **`commands.go`.** Positional `vtui.CmApp + iota` constants that
   `panels_frame.go` names, planned for `internal/cmdline` — the one import
   Task 35 forbids the panel from having. Task 34 or 35 must settle it; the
   destination has to be reachable by panel, editor, viewer and the dialogs.
4. **`internal/viewer/semantic_fields.go`** carries a `ponytail:` marker: three
   generic readers copied from `semantic.go` rather than hoisted, because Task
   34 owns that file's split across five packages. Give them one home when the
   last slice leaves.
5. **The darwin mackeys flake and the two CI-only failures** in `index.md`'s
   Open Findings are untouched by this session.

## Plan-versus-tree discrepancies seen and not acted on

- Task 43's roster assigns `coreAPI` to `internal/plughost` in nine test rows.
  The Task 26 decision moved `api.go` to `internal/app` instead, so those rows
  are stale. Harmless — they describe exports that turned out unnecessary — but
  a reader will trip on them.
- The plan's layer numbers for `gui` (1), `term` (1), `media` (1) and
  `fileops` (1) were assigned before the edges existed. Three are corrected in
  the auditor; `fileops` is still 1 and is still true.

## Tools

Live in the session scratchpad and are copied to the shared location for the
next session. `xbuild.sh` there was broken — `| head` swallowed `go build`'s
exit status, so every target printed OK — and is replaced.

| Script | What it does |
|---|---|
| `xbuild.sh` | ten cross-build targets, fixed status check, `-o /dev/null` |
| `baseline.sh` | the six-module baseline run |
| `qual2.py` | the compiler-driven export-and-qualify loop |
| `exportmethods2.py` | export by receiver name, skipping string literals |
| `exportsyms.py` | export one package symbol the compiler says is undefined |
| `fixselectors.py` | follow the compiler's "but does have" hints |
| `literalkeys.py` | composite-literal keys — **oscillates when two types share a field name; watch it** |
| `unexport.py` | undo an over-export in a test that moved into its package |
| `splittests.py` | split the tests naming a symbol back to `cmd/f4`; appends |
| `addimports.py` | add the imports the compiler names |
| `deadimports.py`, `verify-commits.sh` | as inherited |

**Three of these caused damage this session, all of it the same class.** A
`name:` → `Name:` repair rewrote three lines of a YAML fixture inside a raw
string; a `path` → `Path` export and its revert rewrote `case "Path":` in two
ini readers, which compiles and reads a key the user's file does not have. Only
round-trip tests caught them. `exportmethods2.py` skips string literals now.
The rule that actually holds: **grep the wave's own diff for changed string
content before running the suite, not after.**

```
git diff HEAD -M -- cmd/f4 internal | grep -E '^[+-]' | grep '"'
```

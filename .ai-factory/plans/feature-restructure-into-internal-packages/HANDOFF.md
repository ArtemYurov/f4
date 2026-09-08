# Handoff — where this stands

Everything here is either a thing that must happen first or a thing that lives
nowhere else.

## Where this stands

Written at `c92a6e72`, on a clean, green tree, level with `upstream/main`
(`git rev-list --count HEAD..upstream/main` reports 0). Two merges brought in
27 upstream commits; both are recorded in `index.md`'s Open Findings, along with
the live bug the first of them fixed.

## Where the work stands

Tasks 26-32 and 46 are done and committed; phases 6 and 7 are closed and phase
8 is half done. `cmd/f4` is down from 596 files to 378. The checkboxes in
`index.md` match the tree.

Next is **Task 33, `internal/editor`**. One thing is known about it before it
starts: `findPanelsFrameAnyScreen` is declared in `editor_view.go:5562` and
reads `pf.closed`, a private field of `PanelsFrame`, so Go requires it in the
panel's package. It cannot travel with the file; Task 33 has to lift it out.
`index.md` line 446 calls it composition-root code, which is also wrong.

Task 34 after that is the wave most likely to reorder the menu: it takes
`fuse_mount_action.go` and `fuse_mount_list.go`, and with them the last three
action registrations left in `cmd/f4`. `TestActionOrderIsStable` is the check
that says so.

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
| `fuse_mount_*.go` go to `internal/panel` (Task 34) — they read `fsp.vfs` and `pf.getActivePanel` | `phase-08`, Task 32 |
| `attributes_dialog.go` goes to `internal/dialog`, not `fileops` | `phase-08`, Task 32 |
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

## What the extraction gate does not ask

The gate counts what a file **references**. It says nothing about what a file
**declares** — and a function parked in a moving file travels with it silently,
away from callers that stay behind. Task 32 found seven of them in one wave, and
three were not cosmetic: `padLabel`, `ThemedForeground` and `UseTableColors`
made `internal/fileops` import `internal/dialog`, a layer 1 → layer 3 edge, and
with the attributes dialog pointing back it was a cycle. The compiler caught
that one only because the edge happened to be mutual.

So the wave procedure gains a backward pass, run **before** the move: for every
declaration in the files being moved, where are its callers, and are they going
to the same package?

```sql
WITH decl AS (SELECT id, name, file_path FROM nodes WHERE file_path IN (<wave files>))
SELECT d.name, s.file_path AS caller
FROM edges e JOIN nodes s ON s.id = e.source JOIN decl d ON d.id = e.target
WHERE e.kind IN ('calls','references') AND s.file_path NOT IN (<wave files>);
```

Three things about that index, all measured: run `codegraph sync` first, because
it lags commits and answers about yesterday's tree without saying so;
`is_exported` is 0 for every method, constant and variable regardless of case,
so filter on the first letter instead; and struct fields are not in the model at
all — `fsp.vfs` does not appear — so the query names candidates and grep
confirms them.

## Two mechanical traps, both hit once

**`git commit --only $(git diff --cached --name-only)` builds a commit that does
not compile.** With rename detection, `--name-only` prints only the new path;
the old one stays in the tree and the commit holds both copies. For a wave of
moves take the paths from `git status --short` instead. The point of the
pointed-commit rule is a commit that builds and takes nothing of anyone else's,
and this form quietly broke the first half of it.

**The palette auditor's target map empties itself, and a wave that forgets its
line leaves litter.** `commandPaletteTargetPackage` forward-declares where each
`cmd/f4` file will land so audit keys survive the move; the wave that moves a
file deletes its entry, at which point the directory gives the same answer.
Three entries were stale when Task 32 looked — `codepage_settings.go`,
`macro.go` and its own `queue_manager.go` — so it is worth checking the whole
map rather than only the file you moved:

```
sed -n '/^var commandPaletteTargetPackage/,/^}/p' cmd/f4/command_palette_coverage_test.go \
  | grep -oE '"[a-z_0-9]+\.go"' | tr -d '"' \
  | while read f; do [ -e "cmd/f4/$f" ] || echo "stale: $f"; done
```

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

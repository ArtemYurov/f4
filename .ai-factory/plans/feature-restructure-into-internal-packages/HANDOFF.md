# Handoff — where this stands

Everything here is either a thing that must happen first or a thing that lives
nowhere else.

## Where this stands

Written at `f59248f4`, on a clean, green tree, level with `upstream/main`
(`git rev-list --count HEAD..upstream/main` reports 0). Two merges brought in
27 upstream commits; both are recorded in `index.md`'s Open Findings, along with
the live bug the first of them fixed.

**Nothing is pushed since `7d7f5309`.** A phase boundary without a push is a
boundary without CI, and this wave is the largest commit on the branch: 252
files, 70 of them renames. Twenty-six matrix cells have not seen it.

## Where the work stands

Tasks 26-34 and 46 are done and committed; phases 6, 7 and 8 are closed and
Task 34 closes the larger half of phase 9. `cmd/f4` is down from 596 files to
241, `internal/panel` holds 75, and there are 39 packages under `internal`. The
checkboxes in `index.md` match the tree.

Next is **Task 35, `internal/cmdline`**. Task 34 measured its dependency away:
all five files the roster called "cmdline" read private members of the panel
frame, so they are panel code and `cmdline -> panel` is zero edges.

Everything that waited for Task 34 is closed. `text_editor_bridge.go` and
`visren_editor_bridge.go` are `internal/panel/bridge_texteditor.go` and
`bridge_visren.go`; `panel_lookup.go` is `internal/panel/lookup.go`;
`fuse_mount_*.go` are `fuse_mount.go` and `fuse_list.go`; `cmd/f4/semantic.go`
split, with the frame's half in `internal/panel/frame_semantic.go`; and the two
`semantic_fields.go` copies are gone into `internal/semantic`.
`TestActionOrderIsStable` passes, so the menu did not move.

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
| `hotkeys.go` did move after all — the manager to `internal/keymap`, its conditions to `internal/panel` | `phase-09`, "What the wave actually found" |
| `plugin_hotkeys.go` goes to `internal/panel`, not `internal/app` — it follows `HotkeyManager` | `phase-09` |
| The plugin menu and global-hotkey registries go to `internal/plughost` | `phase-09` |
| 36 panel tests stay in `cmd/f4`: they need the action table, which no seam can supply | `phase-09` |
| Package-name question for Task 44 | `phase-11`, Task 44 step 3 |

## Open tails

1. **`internal/panel/panels_frame_test.go` and its four neighbours are still
   mixed.** The 36 tests that needed the action table were moved out one at a
   time, by running them and watching which failed. That found every test that
   *fails* without dispatch; it cannot find one that passes for the wrong reason
   — a test asserting "nothing happened" passes with an inert seam whatever it
   is really testing. The honest split is by what each test asserts, and nobody
   has read them one by one.
2. **`internal/paneltest` duplicates two helpers into `internal/panel`.**
   `frame_manager_test_helpers_test.go` and the mocks the moved tests share
   exist on both sides, because an in-package test cannot import a package that
   imports it. Twenty lines of scaffolding; the alternative is making every
   panel test an external test package, which the doc comment on
   `internal/paneltest/doc.go` already contemplates.
3. **`internal/editor/view.go`'s `saveUndo` op classes stay private.** The one
   external caller gets `Checkpoint()` instead. If a second appears, the enum is
   the thing to export, not another method.
4. **The darwin mackeys flake and the two CI-only failures** in `index.md`'s
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

## Fourteen files placed by name, and what the graph said

The count is worth keeping because it is the branch's most reliable finding: a
file's name is evidence, and the graph is the verdict. `kitty_*` (media by name,
terminal by graph), `command_runner*` (cmdline, terminal), `colors.go`,
`attributes_dialog.go` (fileops, dialog), `fuse_mount_*` ×2 (fileops, panel),
`async_buffer.go` (app, editor), and then five at once in Task 35's roster —
`cmd_session.go`, `apply_command.go`, `simple_exec.go`,
`command_prefix_registry.go`, `remote_command.go`: all named for the command
line, all reading private members of the panel types, all `internal/panel`.

Two of the fourteen came from blocks written to *correct* the roster, which is
the part worth remembering: a correction goes stale like the thing it corrects,
and the check that catches it is the same one — measure before moving.

## Four mechanical traps, each hit once

**Both obvious ways of listing paths for a pointed commit are wrong, in
opposite directions.** `git diff --cached --name-only` prints only the *new*
path of a rename, so the old one stays in the tree and the commit holds both
copies — it does not compile. `git status --short | awk '{print $2}'` prints
only the *old* path, so the new files are left out of the commit entirely. One
wave hit each. The form that works:

```
git commit --only $(git status --short | sed 's/^...//' | sed 's/ -> /\n/' | tr '\n' ' ') -F <msg>
```

The rule exists so a commit builds and takes nothing of anyone else's; both
shortcuts broke the first half of that silently. `git ls-tree HEAD <old path>`
is what tells you afterwards, and `git commit --amend --only <all paths>` is the
repair.

**A compiler-driven rename loop must never rewrite bare identifiers.** Rewrite
selectors (`.name`) and declarations bound to a named receiver; leave everything
else alone. `qual2.py` and a loop written for Task 33 both ignored that and
wrecked packages that merely shared a name — `closeOnce` became
`fileops.CloseOnce` across four `internal/terminal` files, and `vfs`, the
*package qualifier*, became `Vfs` in 217 files at once. `exportmethods2.py` is
the shape that works, and it takes the receiver names for exactly this reason.

Two tells are worth knowing because they are what actually bites. A loop that
flips between two spellings has found two types sharing a field name — stop it
and resolve by hand; that happened four times in one wave (`vfs`, `indexWG`,
`showSearchDialog`, `showCodepageDialog`). And a loop keyed on `git ls-files`
cannot see the files the wave just created, so it spins without converging:
walk the tree instead.

**Cut test functions with `go/parser`, not with a regexp.** A regexp that finds
a function's start by scanning backwards for a blank line eats the previous
function's closing brace, and the result is still valid Go — a test that
silently moved to the wrong file, or vanished with its assertions. The compiler
cannot see it. Task 33 lost three test tails and five whole tests that way, and
only the literal-diff check plus a comparison against the previous revision
found them. One parser costs less than that comparison did.

**Deleting a line from the palette auditor's target map and adding the package
to the layer map is one operation, not two.** The auditor counts f4's own
surfaces from *both* maps — a file taken out of the first and not entered in the
second stops being counted, silently. On the cmdline wave the count fell from
42 to 40 and that was the only sign. Same family as a sweep that finds nothing
and a test that passes by never dispatching: a check that quietly starts
measuring less than it should.

**The palette auditor's target map empties itself, and a wave that forgets its
line leaves litter.** `commandPaletteTargetPackage` forward-declares where each
`cmd/f4` file will land so audit keys survive the move; the wave that moves a
file deletes its entry, at which point the directory gives the same answer.
Three entries were stale when Task 32 looked — `codepage_settings.go`,
`macro.go` and its own `queue_manager.go` — so check the whole map rather than
only the file you moved:

```
sed -n '/^var commandPaletteTargetPackage/,/^}/p' cmd/f4/command_palette_coverage_test.go \
  | grep -oE '"[a-z_0-9]+\.go"' | tr -d '"' \
  | while read f; do [ -e "cmd/f4/$f" ] || echo "stale: $f"; done
```

## A test whose subject is a binding belongs with the table

`internal/editor` reaches the action layer through a seam, and the registry
behind it is filled by `action_table.go`'s `init` in `cmd/f4`. Nineteen tests
that press a key and expect an action would therefore have passed in the
editor's package **by finding nothing to do** — green because the registry was
empty. They stayed with the table.

**And the wrapper is the material form of the rule.** In `cmd/f4` such a test
presses through `pressKey`, which installs `GlobalHotkeysMgr` and
`macro.MacroMgr` and passes the real macro filter. `testutil.PressKey` with a
`nil` filter skips the dispatch entirely, so the test measures the widget's own
key handling and goes green on every platform where the widget happens to
handle that key. Nineteen tests were moved for the action layer and then
converted to the nil form in the same wave; two platforms disagreed about
whether the widget handled the key, which is the only reason it surfaced.

Checked against the waves already done, and the class had not fired before:
`internal/*` holds two `RunAction` occurrences and both are mock methods, no
test there names `LookupHotkey`, and `RegisterAction` is called only from
`cmd/f4`. Every other key-pressing test drives the widget's own `ProcessKey`,
which is local. Tasks 34 and 35 sit next to the table and will meet it again.

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

Closes #505.

`cmd/f4` held 691 Go files in one flat `package main`, 345 of them non-test.
Every symbol in the application was visible to every other symbol, so the
compiler enforced no boundary anywhere above `vfs/` and `sdk/`. This moves the
application into 40 packages under `internal/`, leaves `cmd/f4` as five files —
`main.go` and four module-wide auditors — and adds a test that fails the build
when an import crosses a layer the wrong way.

Issue #505 asked to tidy the repository root. That is Phase 2 of this branch and
it is done, but it is a third of the change. The root cannot be tidied
meaningfully while everything above it lives in one package: half of what
accumulates there accumulates because there is no other place to put it. The
rest of the branch is what makes the tidying hold.

## What is in the diff

    1326 files changed, 105 591 insertions, 42 746 deletions, 830 renames
    (163 of them byte-identical).

About 55 000 of the insertions are not code:

    .claude/      153 files   27 751 lines   Claude Code agent and skill definitions
    .agents/       81 files   17 263 lines   the same
    .ai-factory/   17 files   10 300 lines   the plan this work followed

The first two are the author's tooling. The third is the plan bundle, and it
ships deliberately — it records why each file went where it went, and every
deviation from the plan with the measurement behind it. To review the code
alone:

    git diff --stat -M upstream/main...HEAD -- ':!.claude' ':!.agents' ':!.ai-factory'
    # 1075 files changed, 50 266 insertions, 42 746 deletions

**`docs/ISSUES/` will look like a mass deletion and is not one.** Both trees
hold 43 files there and the name sets barely overlap: upstream moved its
root-level `*_SOLUTION_REVIEW.md` documents there keeping their names, this
branch moved the same documents and renamed them for their subject
(`ISSUE_165_CONPTY_SYNC_MARKER.md`). git resolves 42 of them as renames — 41
byte-identical, one at 98% — so if the web diff shows 42 deletions beside 42
additions, that is what it is.

## The answer to the refusal on the issue

The proposal was declined as "a broad repository-reorganization proposal without
a narrowly defined bug or testable acceptance criteria", with the note that
doing it autonomously "would require a maintainer-approved project structure and
migration plan".

That is a list of what was missing rather than a no, and both items are in this
PR. `.ai-factory/ARCHITECTURE.md` is the structure: layers, dependency rules,
file naming, package ownership, and what each package owns. The bundle under
`.ai-factory/plans/` is the migration plan, with acceptance criteria on every
one of its 47 tasks. Whether to take them is yours; they are no longer missing.

## The layout, and where it differs from the sketch in the issue

The sketch was the right direction. The boundaries came from measuring the call
graph over 345 files, not from general principle, and four things came out
differently:

- **`tui/` was not created.** The terminal engine is not in this repository —
  it is `vtui` and `vtinput`. There is nothing to put in it.
- **`desktop/` was not created.** No screen manager or modal stack exists in
  this code; `vtui` owns the window stack.
- **`vfs/` stayed in the root** instead of moving under `internal/`. Third-party
  plugins compile against it, and `internal/` would forbid that import.
- **`job/` became `fileops` and `keybind/` became `keymap`**, named for what
  they actually hold.

Packages the sketch did not have — `terminal`, `media`, `plughost`, `sysinfo`,
`i18n`, `theme`, `colorer`, `numeric`, `action`, `toast`, `history` — came out
of the graph.

**No `external/`.** `internal/` is a compiler rule, not a naming convention, and
it has no counterpart. The public surface is whatever sits outside it: `sdk/` and
`vfs/`. An `external/` would restate in a directory name what the language
already enforces.

## What the restructuring found

These were already in the code. They are the argument for doing this at all, and
they are kept separate from anything this work broke and fixed:

- **`internal/editor` raced its own teardown.** `EditorView.Close` cancelled the
  highlighting goroutine without waiting for it, then called `BaseFrame.Close`,
  which writes the field that goroutine reads through `IsDone` between slices.
  Two lines above, the indexing goroutine is joined with `indexWG.Wait()` — the
  asymmetry is the bug. It races in ordinary use, not only under test. Fixed
  here with a matching `highlightWG`.
- **A keybar caption reaches the user as `{KeyBar.EditorAltF8}`.** The key is
  called from the editor and no `.lng` file defines it, including in
  `upstream/main`; `vtui.Msg` renders a miss as `{key}`. Not fixed here — it is
  your string, and the sweep that found it is in `internal/i18n`.
- **`tools/icons`' own test never passed.** It read `../../assets/icon/f4.svg`
  while the tool it tests read `cmd/f4/assets/icon/`, one directory apart, and
  the module has its own `go.mod` so `go test ./...` from the root never ran it.
  Fixed.
- **`plugring_policy_test.go` asserted nothing.**
  `TestShippedCatalogMeetsItsOwnPolicy` resolved `plugring/index.yaml` relative
  to the working directory, which from `cmd/f4` is nothing, so every run reached
  `t.Skipf` and reported as passing. It resolves the path from its own file now.

## What the moves exposed about the checks

Moving 600 files past a CI matrix is an unusually thorough audit of that matrix,
and three of its checks turned out to pass on an empty result:

- **`TestAllDialogs_LayoutValidation` was excluded from every target and
  re-run against a hard-coded path.** When the test moved, `go test -run … ./cmd/f4`
  printed "no tests to run" and exited zero: excluded everywhere, re-run
  nowhere, matrix green. The step now requires the run marker in the output.
- **`//go:generate` is bound to the main package's directory** — the toolchain
  links a `.syso` only from there. The directive travelled with `main.go`, so
  `go generate ./cmd/f4` found no directives and exited zero. Icons would have
  silently stopped being generated with the matrix green. There is a guard for
  the directive's presence now.
- **The race shards filtered by test name over `./cmd/f4`** and covered four
  auditor files while the unfiltered job carried 235. Shards are computed by
  weight now, and no package is named in `build.yml`.

A fourth one answers the same way from the other end. `go vet` type-checks test
files, which makes it the only thing that ever compiles a `//go:build freebsd`
test — and under `GOOS=freebsd` it stops on a third-party dependency before
reaching any of this repository's code, unless it is given the same
`-gcflags=…fakecgo=-std` the build matrix passes. The matrix's own vet cell has
the flag; a local sweep written without it fails identically every time, so a
real type error under freebsd would have arrived looking exactly like that
noise.

The general form, which is worth more than the four fixes: **a check whose
answer never changes is not a check** — most often a check that names a place
and passes when it finds nothing there, and just as often one that fails on
something that is not the subject. `cp` refuses a missing source; `go test`,
`go generate` and a shard filter all succeed on nothing.

Two tests in this branch had the same shape from the other side — they passed
only because a sibling in the same flat package had run first, and `-shuffle`
found them once the packages were smaller. Both now do their own setup.

## Where the reasoning lives

`.ai-factory/ARCHITECTURE.md` is the contract: the layer table, the dependency
rules, the file-naming convention, and which package owns what. It describes the
tree as it is, and `cmd/f4/architecture_test.go` fails if the two disagree —
including if a package is added and not listed.

Layer 0 packages may not import upward, so where a lower package needs a
function that moved above it, the package declares a variable of that signature
and the composition root assigns the implementation:

```go
// internal/config/config.go
var Executable = func() (string, error) {
	return "", errors.New("config: the executable resolver is not wired; the root must set config.Executable")
}
```

`internal/config` never learns that `internal/update` exists, and who implements
what is visible in one file. **Where the default refuses rather than works, that
is deliberate**: a default that works turns forgotten wiring into silently wrong
behaviour. `os.Executable` on the universal Linux build returns the dynamic
loader's path and returns it *successfully*, so a forgotten assignment would not
fail — f4 would look for its ini next to `ld.so`, find none, and use a profile
the user never chose. "Settings are not saved" is a far worse symptom than a
refusal at startup. Where an inert default is harmless the seam takes one
instead: `internal/panel`'s host variables decline the command and show no menu,
which is wrong in a way somebody notices immediately. `internal/terminal` states
the same thing as an interface rather than a set of variables, because what it
needs from above is a coherent group. All three shapes are described in
`ARCHITECTURE.md`.

## Why a rule and not a tidy-up

`CI.md` appeared in the repository root while this branch was clearing it. That
is not a complaint about the file or its author — it is the argument. A root
stays tidy because something says where a file goes, not because somebody tidied
it once. `ARCHITECTURE.md` and `.ai-factory/rules/base.md` are that something,
and they are in this PR.

In the same spirit: `docs/FILELIST.md` is a generated listing that nothing reads
and no check regenerates. It described the base revision's tree for eleven
phases of this work without anybody noticing, which is what a generated file
maintained by memory does. It is correct again and its generator no longer walks
the working directory, but I would propose deleting both — `git ls-files`
answers the same question on demand and cannot go stale.

## What was deliberately not done

- **No behaviour change inside a move commit.** Where behaviour had to change,
  it is its own commit.
- **No Far-derived type was renamed.** The names are the ones the original
  documents use.
- **No package was split further than the graph justified.** `internal/media`
  was the plan's named candidate: its image, audio and video families share
  almost no symbols, but every caller uses at least two of them, so the split
  has no boundary behind it. It stays one package, and the measurement is in the
  bundle.
- **`internal/app` is not yet a composition root by construction.** It is one by
  layer and by import rule, but it reaches for `vtui.FrameManager`, `config.App`
  and their peers the way every other package does. Threading those through a
  constructor is a real change with a real price — it needs a test of the
  startup path first, and there is none — and it is proposed as a follow-up with
  its numbers rather than half-done here. A constructor whose body still reads
  globals is worse than neither, because the signature then asserts something
  the code does not do.

## Housekeeping

- The suppressions this branch added were reviewed as a list; each `#nosec`
  carries the reason beside it, and none hides an unhandled error where there is
  somebody to tell.
- The README screenshot moved from the repository root to `.github/assets/`,
  and README points at it by its raw URL on `main`. That URL 404s until this
  merges; it is not a broken link, it is a link to a path this PR creates.
- CodeGraph is wired in `.mcp.json` for caller/callee questions over a package.
  A fresh clone needs one `init`; the index is git-ignored.
- The branch is level with `upstream/main` as of opening.

## How this was built

Explore → plan → improve → implement → verify → review → security checklist →
archive, from the AI Factory skills vendored in `.claude/skills/`. Two things
are worth saying, because they are what adopting it would buy: every artefact
has one known location instead of loose markdown in the repository root — which
is what #505 complains about — and the plan outlives the session that wrote it,
so the work continues across context resets and across people.

The stronger argument for the method is what verification did to the plan. The
bundle was verified four times and each pass disproved the previous one: routes
measured by filename were wrong, the ordering criterion was inverted,
`actions.go` could not move as a file at all, and 154 tests had no assignment.
None of that survived to the tree.

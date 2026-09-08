# Phase 11: CI, Lint and Documentation

Plan: [index.md](index.md)
Tasks: 38-42
Depends on: Phase 10

## Objective

The build pipeline is rebalanced for a tree that no longer has one giant package,
the pull request opens with no surprise lint backlog, and the prose describes the
repository that now exists rather than the one that used to.

`ARCHITECTURE.md` gets its own task, not a line in the docs sweep, because it is
the one document that changes *genre*: today it describes a target, and after this
plan it describes a fact.

## Current-Code Evidence

| Path | Signal | Consequence |
|---|---|---|
| `build.yml:983` | `matrix: { include: [{ shard: 'cmd/f4', … }, { shard: rest, … }] }` | lint shards named by package path |
| `build.yml:1042` | `"$module/cmd/f4"\|"$module/cmd/f4"/*)` | the shard router |
| `build.yml:1056-1068` | `if [ "$SHARD" = "cmd/f4" ]` … `./cmd/f4/...` | shard-specific arguments |
| `build.yml:1262-1268` | comment: "cmd/f4's suite alone takes as long under the detector as every other package combined" | the stated reason for the split |
| `build.yml:1271-1273` | `cmd/f4 A` / `B-L` / `rest`, `run: '^TestA'` etc. | three race shards splitting one package by test-name letter |
| `build.yml:1318`, `:1326` | `github.com/unxed/f4/cmd/f4` guard and `go test -race … ./cmd/f4` | the shards' bodies |
| `build.yml:1395` | `go list ./... \| grep -Ev '^github.com/unxed/f4/cmd/f4$'` | the `packages` scope, which absorbed every migrated package automatically |
| `.golangci.yml`, `.golangci-strict.yml` | contain no paths | need no edit |
| incremental lint | `--new-from-rev=origin/main` | rename detection across a `package` clause change |
| `AGENTS.md` | 6 `cmd/f4` mentions; "687 files in one flat package main" (`:23`) | structural map, now wrong |
| `.ai-factory/rules/base.md` | "Module Structure" describes the pre-move tree | conventions file |
| `.ai-factory/ARCHITECTURE.md` | 559 lines; `(move)`/`(extract)` markers, migration policy, extraction order | changes genre |
| `docs/` | 48 top-level pages, 13 mentioning `cmd/f4`; 20 across the whole `docs/` tree including `ISSUES/` | per-commit sweeps kept paths current; subjects still need review |

## Files to Change

| Path | Action | Required change |
|---|---|---|
| `.github/workflows/build.yml` | modify | Shard definitions, `packages` scope |
| `AGENTS.md` | modify | Project Structure, counts, docs table |
| `README.md` | modify | Build and icon instructions |
| `.ai-factory/rules/base.md` | modify | Module Structure section |
| `.ai-factory/ARCHITECTURE.md` | modify | Target → fact |
| `docs/*.md` | modify | Subjects, not only paths |
| `.ai-factory/RESTRUCTURE_BASELINE.md` | delete | Or keep, with a stated reason |

---

## Task 38: Rebalance the CI shards

### Intent

The lint and race shards are named after `cmd/f4` because one package held 345
files and 96 495 lines of tests. That package now holds `main.go`. The shards
stayed *correct* throughout the migration — files migrated between them on their
own, and `build.yml:1395` computed the `packages` scope by exclusion — but they
are now badly imbalanced: three race runners split a package with almost no tests
while one runner carries fifteen packages.

This is done **once, here**, not fourteen times during the waves.

### Implementation Steps

1. Replace the lint matrix at `build.yml:983`. Instead of two shards named by
   package path, compute the split from `go list ./...` at job time — for example
   two shards taking alternate entries of the sorted package list, or a split on a
   stable hash of the import path. The requirement is that **no shard definition
   names a package path**, so the next restructuring does not have to touch it.
3. Simplify the shard router at `build.yml:1035-1075`. The `cmd_affected` special
   case and the `./cmd/f4/...` argument branch exist only to serve the named
   shards; with a computed split, the router reduces to "map each affected package
   to its shard".
4. Replace the race matrix at `build.yml:1271-1273`. Drop the `cmd/f4 A` /
   `B-L` / `rest` letter split and its `run:` filters, and drop the `packages`
   scope's exclusion at `build.yml:1395` — with no giant package there is nothing
   to exclude. Shard by package the same way as the lint job.
5. Keep the two behaviours that are not about sharding:
   - the global `-skip '^TestAllDialogs_LayoutValidation$'` at `build.yml:1235`
     and its single-threaded re-run. Confirm which package it points at — Task 25
     step 5 leaves it at `./internal/dialog`, `./cmd/f4` or `./internal/panel`
     depending on how `dialog_layouts_test.go` was split — and require an explicit
     `--- PASS` from that target here. The reason for the isolation is unchanged:
     layout validation mutates shared UI registries from parallel subtests.
   - the race-instrumented cache keys at `build.yml:1288-1296`. Update the
     `cache-key` values to match the new shard names; the reason for a separate key
     (the shared setup-go key is claimed by a non-race job) still holds.
6. **Measure.** Record the wall-clock of the `lint` and `race` jobs before and
   after on the same commit. A rebalancing that makes CI slower is not done.

### Required Interfaces and Contracts

- Every package in `go list ./...` is linted by exactly one shard and raced by
  exactly one shard. No package is covered twice and none is dropped — this is the
  property the named shards guaranteed by construction and a computed split must
  prove.
- The incremental-lint path (`--new-from-rev=origin/main`) is unchanged in
  behaviour; only the shard assignment changes.
- `fail-fast: false` stays: a shard failing must not cancel its siblings.

### Error Handling and Logging

An empty shard is a workflow error, not a silent skip — `build.yml:1336-1339`
already errors when the package list comes out empty, and that guard must survive
the rewrite. A shard computing to zero packages after a future rename is exactly
the failure mode it protects against.

### Tests

The workflow is the test. Validate before merging:

```
# every package assigned exactly once
go list ./... | sort > /tmp/all.txt
# run the shard-assignment snippet for each shard, concatenate, sort, compare
diff /tmp/all.txt /tmp/sharded.txt
```

Then push the branch and confirm both jobs go green with no package unaccounted
for.

### Acceptance Criteria

- `grep -n "shard: 'cmd/f4'\|cmd/f4 A\|cmd/f4 B-L" .github/workflows/build.yml`
  returns nothing.
- The `diff` above is empty.
- Lint and race wall-clock times are recorded in the commit message and are not
  worse than before.

### Verification

- The `diff /tmp/all.txt /tmp/sharded.txt` check.
- Expected result: no output.
- A CI run on the branch.
- Expected result: both jobs green, every shard non-empty.

---

## Task 39: Run the incremental lint against `origin/main` before opening the PR

### Intent

`.golangci.yml` and `.golangci-strict.yml` contain no paths and need no edit. The
risk is elsewhere: the incremental lint runs `--new-from-rev=origin/main`, and a
changed `package` clause can defeat golangci-lint's rename detection. If it does,
the project's existing backlog surfaces as new findings on this pull request —
roughly 2450 of them — and the maintainer's first impression of the PR is a red
job with thousands of entries.

Finding this locally costs one run. Finding it in review costs the PR.

### Implementation Steps

1. **Level `origin/main` with `upstream/main` first.** The incremental lint takes
   `origin/main` as its base, and the fork lags upstream — 37 commits at the time
   this step was added. Every one of those commits is reported as this branch's
   work: the first CI run of the branch returned two gosec findings in
   `cmd/f4/input_translation.go`, which came from the upstream commit `5b21864e`
   and not from this work at all. Until the bases agree, a backlog measurement
   says nothing, because there is no telling whose backlog it is.
2. Fetch the base and run the same command CI runs, over the whole branch:
   ```
   git fetch origin main
   golangci-lint run --new-from-rev=origin/main ./...
   ```
3. Record the finding count. Compare against a run on `origin/main` itself to
   establish what the pre-existing backlog is:
   ```
   git stash && git checkout origin/main
   golangci-lint run ./... 2>&1 | tail -1
   git checkout - && git stash pop
   ```
4. If the branch run reports substantially more than the genuine new-code findings,
   rename detection failed. Do not fix it by adding nolint directives. Instead
   state the measured numbers in the PR body — "golangci-lint's `--new-from-rev`
   does not track these renames; the incremental job reports N findings, of which
   the pre-existing backlog on `main` is M" — so the maintainer sees a known
   quantity rather than a mystery.
5. Run the strict configuration over the new packages only, where it is meaningful:
   ```
   golangci-lint run -c .golangci-strict.yml ./internal/numeric/... ./internal/toast/... ./internal/history/... ./internal/action/...
   ```
   These four are new code written during this work, so they can be held to the
   strict bar.

6. Verify that **every commit** on the branch builds, not only `HEAD`. The
   plan's central invariant is that any commit can be checked out and built, and
   checking `HEAD` alone never tests it:
   Walk the revisions in a scratch worktree, which reads and rewrites nothing:
   ```
   git worktree add -q --detach /tmp/verify upstream/main
   for c in $(git rev-list --reverse upstream/main..HEAD); do
       git -C /tmp/verify checkout -q --detach "$c"
       (cd /tmp/verify && CGO_ENABLED=0 go build ./... && go vet ./...) \
           || echo "FAIL $c $(git log -1 --format=%s "$c")"
   done
   git worktree remove --force /tmp/verify
   ```
   Skip a commit that touches no `.go`, `go.mod` or `go.sum` — its result is the
   previous commit's. Use `go vet` and not only `go build`: `build` ignores test
   files, and the failure this catches is a helper deleted one commit before its
   replacement arrives.

   `git rebase --exec` would do the same job and rewrite every commit id doing
   it. Worse, this branch merges upstream rather than rebasing onto it, so a
   plain `git rebase` would flatten those merges and check a history that is not
   the one being reviewed. If a rebase is used anyway it needs
   `--rebase-merges`; the worktree walk needs nothing and changes nothing, which
   is what a check should do.

   A broken commit in the middle of a 300-file restructuring is not cosmetic: it
   breaks `git bisect` for whoever debugs a regression a year from now, and it is
   the first thing a maintainer notices on a branch that claims every step is
   green. The failure mode to watch for is a staged deletion travelling in
   somebody else's commit — `git commit` takes the whole index, not the paths
   just handed to `git add`.

### Required Interfaces and Contracts

- No `nolint` directive is added to work around rename detection.
- No lint configuration is changed. The two `.golangci*.yml` files are outside the
  scope of this plan.
- `gosec`'s `G115` findings must be zero in `internal/numeric` — Task 19 required
  the `#nosec` annotations to travel verbatim, and this is where that is confirmed.
- Every commit from `upstream/main` to `HEAD` builds and vets clean.

### Error Handling and Logging

Not applicable. This task produces numbers and a paragraph for the PR body.

### Tests

The lint runs above are the test.

### Acceptance Criteria

- Both counts are recorded.
- `golangci-lint run -c .golangci-strict.yml ./internal/numeric/...` reports no
  `G115`.
- The per-commit walk reaches `HEAD` without a failure.
- If rename detection failed, the PR body says so with the measured numbers.

### Verification

- `golangci-lint run --new-from-rev=origin/main ./... 2>&1 | tail -3`
- Expected result: a finding count that the PR body accounts for.

---

## Task 40: `/aif-docs` checkpoint

### Intent

Each move commit closed its own path references — that is a ground rule and an
unclosed reference is an unfinished move. But the restructuring changes what the
48 subsystem documents *describe*, not only the paths inside them. A document that
says "panels live in the flat package alongside the editor" is not fixed by
updating a path.

### Implementation Steps

1. Run `/aif-docs` and treat its findings as part of this commit rather than as a
   follow-up.
2. `AGENTS.md` needs more than a path sweep. Rewrite:
   - the Project Structure block, which still opens with
     `cmd/f4/  # the application: 687 files in one flat package main`;
   - the Documentation table row for `SPREADSHEET.md`, moved in Task 12;
   - the Agent Rules section on CodeGraph, whose advice — "`cmd/f4` is one flat
     `package main` of ~109k lines, so grep over it is slow and matches
     identifiers it should not" — was true and is now false. The graph is still the
     right tool for symbol questions; the *reason* changed.
3. `.ai-factory/rules/base.md`'s Module Structure section lists the pre-move tree
   and tells new code where to go. Update the list, re-derive the counts it
   quotes, and drop the transitional wording — after this branch the tree is not
   "being split", it is split.
   Then give the section a **File Placement** part, because this is the file the
   AI Factory skills read as project rules and it currently answers "what exists"
   without answering "where does mine go":
   - the package that owns the subject; no package owns it, create one; never
     `internal/app`, which wires and does not implement;
   - nothing new in `cmd/f4` — it holds `main.go`, the wiring tests, the four
     module-wide auditors and the Windows `.syso` files;
   - inside a package, `<topic>.go` and `<topic>_<aspect>.go`, prefix naming the
     topic and not the package (`panel/frame.go`, never `panel/panel_frame.go`),
     platform suffix last;
   - a test lives with its subject; a test spanning packages is hosted by the
     latest one and splits or uses `package X_test` — the rule Task 43 applied to
     61 of them;
   - resources travel with the package that embeds them, and a test reading them
     from disk guards against the empty set.
   Keep it short and point at `ARCHITECTURE.md` for the reasoning: `base.md` is
   the rule, the architecture document is the argument. Duplicating the argument
   in both guarantees they drift.
4. Walk the 13 of 48 top-level `docs/*.md` pages that mention `cmd/f4` and check
   each for a claim about *structure* rather than a path (`grep -rl 'cmd/f4' docs/`
   returns 20 because it also walks `docs/ISSUES/`, which is a historical record
   and is not rewritten here): which subsystem owns what, what is in one
   package, what a contributor must not couple.
5. `README.md:233`, `:237`, `:241` — the build and icon-generation instructions.
   Confirm Tasks 11 and 27 left them correct.

### Required Interfaces and Contracts

- Documentation describes the current state. No document explains the migration,
  compares "before" with "after", or references how the code used to be organised —
  that history lives in git.
- Counts quoted in prose are re-derived, not estimated: package count from
  `go list ./... | wc -l`, file counts from `find`.

### Error Handling and Logging

Not applicable.

### Tests

No automated test covers prose. The check is the grep sweep:

```
grep -rn 'flat package\|one flat\|687 files\|345 files' docs/ README.md AGENTS.md .ai-factory/
```

### Acceptance Criteria

- The grep above returns nothing outside `ARCHITECTURE.md` (Task 41 owns that
  file) and the plan artifacts under `.ai-factory/plans/`.
- `AGENTS.md`'s Project Structure block lists the packages that exist.
- Every count quoted in `AGENTS.md` and `rules/base.md` is re-derived.

### Verification

- `grep -rn 'cmd/f4/' docs/ README.md AGENTS.md`
- Expected result: only the build invocations (`go build ./cmd/f4`,
  `go generate ./cmd/f4`) and the `.syso` note.

---

### Note: `docs/FILELIST.md` is generated

`scripts/filelist_update.sh` writes it from a `tree -a` of the repository, so
every commit that moves a file makes it stale. Do **not** regenerate it per
commit: the current snapshot predates this branch's own tooling, and a fresh run
would add `.ai-factory/`, `.claude/`, `.agents/` and `build/` to the diff in the
same breath as the file being moved. Regenerate it once, here, when the tree has
stopped moving — and check the result before committing it, since `tree -a`
happily lists directories that are git-ignored.

---

## Task 41: Rewrite `ARCHITECTURE.md` from target to fact

### Intent

Every other document needs updating. This one changes genre. It currently
describes a destination — `(move)` and `(extract)` markers on a tree that does not
exist yet, an extraction order, a migration policy, a table splitting files
between layer-0 and view-bound halves. Once the plan is executed, none of that
describes the repository; it describes the journey.

The project rule is explicit and applies to documents as much as to code
comments: text explains the **current** state, never the past. No "used to be X,
now Y", no account of how the code evolved. History lives in git. So this is not
"append a note saying it is done" — it is removing everything that describes a
transition and leaving a description of the result.

### Implementation Steps

0. **Locate the sections by heading, never by line number.** Start with
   `grep -n '^#\{1,3\} ' .ai-factory/ARCHITECTURE.md` and work from the offsets it
   prints. Every section below is named by its heading text for that reason: line
   numbers in a 559-line document drift with the first edit, and the sections in
   step 4 are bold paragraphs inside **Folder Structure**, not headings of their
   own — find them by their bold title.
1. **Folder Structure** (heading to the next one): strip every `(move)` and
   `(extract)` marker.
   Nothing moves any more; the tree is the tree. Keep the annotations that explain
   *why* a directory sits where it does — `sdk/` and `vfs/` being importable from
   outside the module, `embedded.go` being pinned by `//go:embed`,
   `rsrc_windows_*.syso` being linked only from the built package's directory,
   `internal/hideconsole` being a vendored fork.
2. Add the packages this plan created that the document does not yet name:
   `internal/action`, `internal/toast`, `internal/history`, `internal/numeric`,
   `internal/testutil` and `internal/paneltest`. Give the last two a line saying
   they are test scaffolding and no production file imports them.
3. **Overview** (first heading): "Two things are missing… `cmd/f4` holds 345 non-test
   files and ~109k lines in one flat `package main`" is false after Phase 10.
   Rewrite the paragraph to state what the layout *is* and what rule it expresses.
4. **Delete the sections that are scaffolding**, not description:
   - the bold paragraph "**`app` is two things, and only one of them is the
     root**" inside Folder Structure, with its file-split table. It is an
     instruction for performing the extraction.
   - the bold paragraph "**`sysinfo` keeps its own copy of the one numeric helper
     it needs**" — *keep the rule*, drop the justification framed as a migration
     decision. It is a live constraint: sysinfo is a leaf and must stay one.
   - the whole **Legacy vs New Code Policy** section: the extraction order, the
     one-subsystem-per-commit rule, "no rewrites inside a move commit", "a move is
     not done until the prose agrees". Scaffolding, all of it. What survives is the
     first bullet, reworded: new code goes into the module it belongs to, and if
     none fits, create the package.
5. **Keep unchanged** — these are permanent contracts, not migration aids:
   - **Decision Rationale**
   - **File Naming Inside a Package**, including the multi-type-file rule
   - **Dependency Rules** and the layer table, updated only with the new package
     names from step 2. It already places `internal/wincon`, `internal/ttyx`,
     `internal/netproxy` and `internal/hideconsole` at layer 0; Task 8 seeds the
     auditor's map with the first three so the two agree.
   - **Layer / Module Communication**
   - **Key Principles**, with principle 5's test-scaffolding paragraph rewritten to
     describe `internal/testutil` and `internal/paneltest` as they exist rather
     than as a plan
   - **Code Examples**
   - "Not every directory here is one module", inside Dependency Rules — six
     `go.mod` files is a standing fact
7. **Anti-Patterns** (last section): "**Adding to the flat package**" must be
   reworded. There is no flat package any more, but the rule it protects survives:
   a new feature belongs in the module it serves, and if none fits, in a new
   package — never appended to whichever package is nearest.
7. **Re-derive every number.** `345 non-test files`, `~109k lines`,
   `297 files of extensions`, `~700 Go files outside plugins`,
   `560 _test.go files`, `48 subsystem documents`, `133 files read AppConfig`,
   `37 call edges`, `42 audit keys`. Count them; do not adjust them by arithmetic.

### Required Interfaces and Contracts

- The document describes the repository as it is. A reader who has never seen the
  old tree must not be able to tell that a restructuring happened.
- No sentence contains "used to", "previously", "was moved", "after the
  extraction", "no longer", or a date.
- The dependency rules, layer table and naming convention keep their normative
  force — they are what the next contributor is held to.
- `cmd/f4/architecture_test.go` (Task 8) enforces four of the rules. Where the
  document states a rule the test checks, say so, so a reader knows which rules
  are mechanical.

### Error Handling and Logging

Not applicable.

### Tests

The auditor is the closest thing to a test of this document:

```
go test ./cmd/f4 -run '^TestArchitecture' -v
```

Every layer the document asserts must appear in the test's layer map, and vice
versa. A package in the document but not in the map is undocumented drift.

### Acceptance Criteria

- `grep -n '(move)\|(extract)' .ai-factory/ARCHITECTURE.md` returns nothing.
- `grep -niE 'used to|previously|no longer|after the extraction|migration' .ai-factory/ARCHITECTURE.md`
  returns nothing.
- Every package in `architecture_test.go`'s layer map appears in the document's
  folder structure, and vice versa.
- Every quoted count is re-derived.

### Verification

- `go test ./cmd/f4 -run '^TestArchitecture' -v` and a manual comparison of the
  layer map against the document's layer section.
- Expected result: the two agree package for package.
- `grep -n '(move)\|(extract)' .ai-factory/ARCHITECTURE.md`
- Expected result: no output.

---

## Task 42: Drop the migration baseline

### Intent

`.ai-factory/RESTRUCTURE_BASELINE.md` answered one question — "was this test red
before we started?" — for fourteen wave commits. With the last wave green, nothing
asks it any more.

This is the only task permitted to modify or remove that file.

### Implementation Steps

1. Confirm the last wave compared clean against it.
2. Decide, and state the decision in the commit message:
   - **delete** — `git rm .ai-factory/RESTRUCTURE_BASELINE.md`; the information it
     held is in the commit history of this branch; or
   - **keep** — because it records three pre-existing failures
     (`tools/icons`, `tools/wine_syscall_probe` on darwin/arm64, the inert
     `plugring_policy_test.go`) that outlive this work and that nobody else has
     written down. If keeping it, rename it to something that is not about a
     finished migration and move it to `docs/`.
   Prefer deleting: two of the three findings should by then be issues in the
   tracker, which is where they belong.
3. Whichever is chosen, make sure the three pre-existing findings do not vanish
   silently. If the file goes, they go into the PR body or into issues.

### Required Interfaces and Contracts

None. This is a repository-hygiene task.

### Error Handling and Logging

Not applicable.

### Tests

None.

### Acceptance Criteria

- The commit message states which option was taken and why.
- The three pre-existing findings are recorded somewhere that survives this
  branch.

### Verification

- `go test -timeout 25m ./...` and the five other module runs.
- Expected result: green everywhere except the three recorded pre-existing
  failures, which are unchanged from Task 1.

---

---

## Task 45: Write the pull request

### Intent

Six earlier tasks each end with "call this out in the PR body" and none of them
owns the body. A requirement everybody references and nobody writes is a
requirement that does not happen. This is where it is written, together with the
context a maintainer needs to judge a 300-file change he did not plan.

### Implementation Steps

1. **What and why**, in three or four sentences. `cmd/f4` held 345 non-test
   files in one flat `package main`; the compiler enforced no boundary anywhere
   in the application. Now it does.

2. **`Closes #505`.** The issue asked to tidy the repository root, which is
   Phase 2 of this branch. Say plainly that the PR does more than the issue
   asked, and why the rest belongs in the same change: the root cannot be tidied
   meaningfully while everything above it lives in one package.

3. **Answer the refusal in the issue thread on its own terms.** The bot
   maintainer declined it as "a broad repository-reorganization proposal without
   a narrowly defined bug or testable acceptance criteria", adding that doing it
   autonomously "would require a maintainer-approved project structure and
   migration plan". That is not "no", it is a list of what was missing — and the
   PR brings exactly those two things: `ARCHITECTURE.md` is the structure, and
   the archived bundle is the migration plan, with acceptance criteria on every
   task. Say so in one paragraph. It moves the PR from "here is my vision" to
   "here are the missing inputs; the decision is yours".

4. **Answer the `external/` question the issue asks.** No, and briefly why:
   `internal/` in Go is a compiler rule rather than a naming convention, and it
   has no counterpart. The public surface is whatever sits *outside* it — here
   `sdk/` and `vfs/`, which third-party plugins compile against. An `external/`
   would restate in a directory name something the language already enforces.

5. **Explain where the tree differs from the one sketched in the issue**, rather
   than diverging in silence. The sketch was the right direction; the boundaries
   came from measurement.
   - `tui/` (buffer, driver, event) — not created. The terminal engine lives
     outside this repository, in `vtui` and `vtinput`; there is nothing to put
     in it.
   - `desktop/` (screen manager, modal stack) — not created. No such thing
     exists in the code; `vtui` owns the window stack.
   - `vfs/` stayed in the root instead of moving under `internal/`: third-party
     plugins compile against it, and `internal/` would forbid that import.
   - `job/` became `fileops` and `keybind/` became `keymap`, named for what they
     actually hold.
   - Packages the sketch did not have — `term`, `media`, `plughost`, `sysinfo`,
     `i18n`, `theme`, `colorer`, `numeric`, `action`, `toast`, `history` — came
     out of the call graph over 345 files, not out of general principle.

6. **Where the reasoning lives.** `ARCHITECTURE.md` is the contract: layers,
   dependency rules, file naming, package ownership. The archived bundle is the
   plan that produced it. Both ship in this PR deliberately — a maintainer
   inheriting a restructuring needs to know why a file went where it went, and
   git history alone does not answer that.

7. **How it was built**, as a recommendation rather than a pitch. The chain is
   explore → plan (ultra) → improve → implement → verify → review →
   security-checklist → archive, from the AI Factory skills vendored in
   `.claude/skills/`. Two things are worth saying because they are what adopting
   it would actually buy: every artefact has one known location instead of loose
   markdown in the repository root — which is what #505 complains about — and the
   plan outlives the session that wrote it, so work continues across context
   resets and across people. Add that the bundle was verified four times and each
   pass disproved the previous one: routes measured by filename were wrong, the
   ordering criterion was inverted, `actions.go` could not move as a file, and
   154 tests had no assignment at all. That is the case for the method, and it is
   stronger than any claim about it.

8. **The graph tooling.** CodeGraph is wired in `.mcp.json` and answers
   caller/callee/impact questions over a package where grep is both slow and
   imprecise. Note that a fresh clone needs one `init` and that the index is
   git-ignored.

9. **Everything the earlier tasks asked to surface**, one line each: the README
   screenshot URL 404s until this merges into `main` (Task 11); the
   incremental-lint backlog number, if rename detection surfaced it (Task 39);
   the three pre-existing failures recorded in the baseline (Task 42); the
   structural review outcome and any follow-up proposals (Task 44).

10. **One line on why the rule matters more than the tidying.** `CI.md` appeared
    in the repository root while this branch was clearing it. That is not a
    complaint about the file or its author — it is the argument: a root stays
    tidy because something says where a file goes, not because somebody tidied
    it once. `ARCHITECTURE.md` and `rules/base.md` are that something, and they
    are in this PR.

11. **What was deliberately not done**: no behaviour change inside a move commit,
    no renamed Far-derived type, no further splitting of packages — that is
    proposed as follow-up with evidence rather than smuggled in.

### Required Interfaces and Contracts

- The body is **prepared, not sent**. No `git push` and no `gh pr create`: when
  the work goes out is the user's decision and this task does not take it.

### Error Handling and Logging

Not applicable.

### Tests

None. The body is prose.

### Acceptance Criteria

- `Closes #505` is present.
- The refusal in the thread is answered, and the `external/` question with it.
- Every divergence from the issue's sketch is named and explained.
- All four carried-over items from step 9 appear.
- The workflow recommendation is concrete and says what the maintainer gains.
- Somebody who did not plan this change can read the body and know what to
  review first.

### Verification

- Preview the body (`gh pr create --dry-run`, or read the file).
- Expected result: a body a maintainer can act on. Nothing is pushed and no pull
  request is opened.

## Phase Risks and Mitigations

- **Risk:** the computed CI shards drop a package and a whole area stops being
  linted or raced, silently.
  **Mitigation:** Task 38's `diff /tmp/all.txt /tmp/sharded.txt` check proves every
  package is assigned exactly once, and the empty-shard guard at
  `build.yml:1336-1339` survives the rewrite.
- **Risk:** the incremental lint's rename detection fails and the PR opens red
  with thousands of pre-existing findings.
  **Mitigation:** Task 39 measures it locally before the PR and puts the numbers in
  the PR body.
- **Risk:** `ARCHITECTURE.md` is updated by appending "this is now done", leaving a
  document that narrates a migration.
  **Mitigation:** Task 41's acceptance criteria grep for exactly that vocabulary,
  and the project rule against past-tense explanation is stated in the intent.
- **Risk:** the baseline is deleted along with the only written record of three
  pre-existing failures.
  **Mitigation:** Task 42 step 3 requires them to land somewhere else first.

## Phase Completion Checklist

- Every Task 38-42 satisfies its acceptance criteria.
- No shard definition in `build.yml` names a package path.
- The PR body accounts for the incremental-lint finding count, the plugring
  catalogue URL change, and the branch-scoped README image URL.
- No document describes the migration; `ARCHITECTURE.md` describes the tree.
- `go test -timeout 25m ./...` plus the five other module runs match the Task 1
  baseline's green set.
- `index.md` task checkboxes 38-42 are ticked.

---

## Task 44: Review the finished tree before calling it done

### Intent

Every package in the target layout was chosen from a call graph measured while
the code still lived in one flat package. That measurement was good enough to
order the work, but a package only reveals its real shape once it exists and has
callers. This task looks at the result and asks whether the split went far
enough, not far enough, or in the wrong place — and records the answer instead of
leaving it to whoever notices first.

It changes no code. Splitting a package further is a separate pull request with
its own evidence; doing it here would enlarge a change that already touches the
whole tree.

### Implementation Steps

1. **Measure every extracted package as it now stands.** File count, and for each
   one the list of packages that import it:
   `go list -f '{{.ImportPath}} {{join .Imports " "}}' ./... | grep internal/`.
   A package nobody imports but `internal/app` is a candidate for merging back;
   a package imported by everything is a candidate for splitting.
2. **Ask the split question where the parts have different callers.** The named
   candidate is `internal/media`: its `image_*`, `audio_*` and `video_*` families
   were measured as effectively unconnected before the move — one reference in
   total, `imageViewBackAttr`, which is a colour attribute and by then may live in
   `internal/theme`. If after extraction `panel` reaches only the image half and
   something else only the audio half, the boundary is real and worth a follow-up.
   If every caller uses all three, it is one package and stays one.
   Apply the same question to the largest results — `term`, `panel`, `app` — and
   to anything over roughly forty files.
3. **Decide the two files whose home is genuinely arguable** rather than leaving
   them where the wave put them by default: `player_panel.go` (a panel over the
   media engine — media or panel?) and `sixel_layers.go` (graphics — media or
   term?). State the reason, not just the choice.
4. **Check that no new flat package appeared.** The failure this whole branch
   exists to undo is one package accumulating unrelated code. Verify no extracted
   package holds files from two unrelated subjects, and that `internal/app` holds
   wiring rather than features that found no other home.
5. **Verify the tree against `ARCHITECTURE.md` as rewritten in Task 41** — the
   layer table, the dependency rules, the file-naming convention. Where the code
   and the document disagree, one of them is wrong; say which.
6. **Write the outcome into the PR body**, in a short section: what was reviewed,
   what stays as is, and what is proposed as a follow-up with the evidence behind
   it. A reviewer should not have to ask whether the structure was thought about
   after it was built.

### Required Interfaces and Contracts

None. Read-only review.

### Error Handling and Logging

Not applicable.

### Tests

None added. `cmd/f4/architecture_test.go` (Task 6) already asserts the layer
rules; this task reads its result rather than extending it.

### Acceptance Criteria

- Every extracted package has a recorded file count and caller list.
- `internal/media` has an explicit keep-or-split decision with the caller
  evidence behind it.
- `player_panel.go` and `sixel_layers.go` have a stated home and a reason.
- No package outside the composition root holds two unrelated subjects.
- The PR body carries the review outcome and any follow-up proposals.

### Verification

- `go list -f '{{.ImportPath}} {{join .Imports " "}}' ./...`
- `go test ./cmd/f4 -run TestArchitecture`

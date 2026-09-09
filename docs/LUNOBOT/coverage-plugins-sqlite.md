# Coverage: `plugins/sqlite`

## Scope

This change adds behavior-focused coverage for the SQLite plugin's browser UI:

- synchronous query, schema refresh, and invalid-SQL paths;
- writes with SQLite affinity confirmation, row deletion, and row insertion;
- F9, paging, Insert, Delete, and Enter-on-table key routing;
- filename, table-selection, and result-row helper contracts.

## Baseline

The package was selected under § 22 from Codecov for current `main` commit `ada29bd041dcf40d7631c5b74fc773868e6dd51a`: `plugins/sqlite` had 26.47% line coverage (230 of 869 lines). Its `ui.go` had no covered lines, while the package contains the real SQLite browser/editor logic.

## Verification

No local Go build or test was run, per the current LUNOBOT instructions. Static checks are `gofmt` and `git diff --check`; GitHub Actions is authoritative for this change.

The exact CI run and post-merge Codecov result will be recorded here as they become available.

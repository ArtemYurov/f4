# Coverage: `plugins/archive/zip_encoding.go`

## Scope

This change covers ZIP filename decoding for:

- names already marked as UTF-8;
- CP866 names from DOS creators 0, 6, and old OS=11 archives;
- Windows-1251 names from modern OS=11 archives;
- CP437 fallback for an unknown creator.

## Baseline

The file was selected from the live Codecov snapshot after the background-jobs
coverage task: `plugins/archive/zip_encoding.go` had 0% line coverage for 17
lines of real encoding-selection and decoding logic. The total repository
coverage in that snapshot was 60.87%.

## Verification

No local Go build or test is run, per the current LUNOBOT passport. Static
checks are `gofmt` and `git diff --check`; GitHub Actions is the authoritative
verification environment. CI and merge results will be recorded here after
the pull request is verified.

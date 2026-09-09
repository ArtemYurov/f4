# Coverage: `plugins/archive/password.go`

## Scope

This change adds deterministic tests for the archive password layer:

- lazy 7z read-error classification and ZipCrypto payload errors;
- retrying empty password answers and stopping on context or prompt errors;
- the no-UI password-prompt guard;
- password installation for value and pointer RAR/7z formats, including the
  empty-password and unsupported-format paths.

## Baseline

The file was selected from the fresh Codecov report after merge commit
`9b23c041e953ded7218301b2eaca9048b4265768`: `plugins/archive/password.go`
had 47.56% line coverage for 164 lines, with 75 missed lines. Total
repository coverage was 60.98%.

## Verification

No local Go build or test is run, per the current LUNOBOT passport. Static
checks are `gofmt` and `git diff --check`; GitHub Actions is the authoritative
verification environment. CI and merge results will be recorded here after
the pull request is verified.

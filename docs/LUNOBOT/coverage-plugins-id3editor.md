# Coverage: `plugins/id3editor`

## Scope

This change covers the ID3 editor plugin's command dispatch and dialog lifecycle:

- local-only, selection, and MP3-extension guards;
- opening a selected MP3 in the editor dialog;
- populating the editor controls and saving edited fields;
- closing the editor through Cancel;
- the existing ID3 v1/v2 comment and round-trip tests remain in place.

## Baseline

The package was selected from the Codecov report for merge `3ab39ef964ed065fbd9de1f81fde57e5f481dae6`: `plugins/id3editor` had 26.69% line coverage. The package contains real plugin logic; platform stubs and the actively claimed `internal/ttyx` package were excluded from this selection.

## Verification

No local Go build or test was run, per the current LUNOBOT instructions. Static checks are `gofmt` and `git diff --check`; GitHub Actions is the authoritative verification for this change.

The exact CI run and post-merge Codecov result will be recorded here as they become available.

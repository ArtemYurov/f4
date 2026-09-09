# Coverage: `internal/media/video_view.go`

## Scope

This change adds deterministic tests for the video frame without starting mpv
or requiring an X session:

- construction, title and key-label callbacks, frame/top-bar layout, and type;
- the pre-layout/session guard and idempotent start path;
- playback key routing, pause state, unrelated input, and all close-key aliases;
- closing a frame that has no player.

## Baseline

The file was selected from the fresh Codecov report for main commit
`729ed42d`: `internal/media/video_view.go` had 0% line coverage for 102 lines.
Total repository coverage was 61.31%.

## Verification

No local Go build or test is run, per the current LUNOBOT passport. Static
checks are `gofmt` and `git diff --check`; GitHub Actions is the authoritative
verification environment. CI and merge results will be recorded here after
the pull request is verified.

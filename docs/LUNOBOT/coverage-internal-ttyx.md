# Coverage: `internal/ttyx`

## Status

The first coverage slice adds deterministic lifecycle and no-display checks for
`Overlay`. These checks cover the public guard paths that do not require an X11
server; the existing X11-backed tests remain responsible for server behavior.

The package's broader coverage work remains outside this slice.

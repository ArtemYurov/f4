# Restructuring baseline

What "green" meant before any file moved out of `cmd/f4`. Every later task of the
restructuring plan compares its run against this file.

**This file is immutable.** It is a snapshot of one revision, never refreshed from
a later run: rewriting it turns a regression introduced on wave N into "known red"
on wave N+1 and loses the signal permanently. Later tasks compare against it. The
only task that touches it again is the one that deletes it.

- Revision: `0cda22a7`, the commit this file is added on top of
- Toolchain: `go version go1.26.6 darwin/arm64`
- Host: Darwin arm64
- Taken: 2026-09-08

The repository holds six Go modules. `go test ./...` from the root sees only the
first one, so the other five are recorded separately.

## Module 1 — `github.com/unxed/f4` (repository root)

| Command | Exit |
|---|---|
| `CGO_ENABLED=0 go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test -timeout 25m ./...` | 0 |

- `go list ./... | wc -l` → **38** packages
- `ok` → **30**, `no test files` → **8**, `FAIL` → **0**
- `cmd/f4` alone: `ok … 50.9s`

### Green set — the 30 packages that ran tests and passed

```
github.com/unxed/f4/cmd/f4
github.com/unxed/f4/fusefs
github.com/unxed/f4/internal/netproxy
github.com/unxed/f4/internal/ttyx
github.com/unxed/f4/internal/wincon
github.com/unxed/f4/luaplug
github.com/unxed/f4/piecetable
github.com/unxed/f4/plugins/android
github.com/unxed/f4/plugins/archive
github.com/unxed/f4/plugins/chroma
github.com/unxed/f4/plugins/cloudfox
github.com/unxed/f4/plugins/dummy_internal
github.com/unxed/f4/plugins/envman
github.com/unxed/f4/plugins/id3editor
github.com/unxed/f4/plugins/ios
github.com/unxed/f4/plugins/ios/internal/afcproto
github.com/unxed/f4/plugins/ios/internal/corefileservice
github.com/unxed/f4/plugins/mediainfo
github.com/unxed/f4/plugins/netfox
github.com/unxed/f4/plugins/netfox/fishplus
github.com/unxed/f4/plugins/sqlite
github.com/unxed/f4/plugins/visren
github.com/unxed/f4/sdk/extui
github.com/unxed/f4/sdk/f4rpc
github.com/unxed/f4/sheet
github.com/unxed/f4/textlayout
github.com/unxed/f4/tools/hardcode
github.com/unxed/f4/tools/langfmt
github.com/unxed/f4/vfs
github.com/unxed/f4/vtvibe
```

### The 8 packages with no test files

```
github.com/unxed/f4
github.com/unxed/f4/plugins/dummy_rpc
github.com/unxed/f4/sdk/f4plugin
github.com/unxed/f4/tools
github.com/unxed/f4/tools/wine_color_probe
github.com/unxed/f4/vfs/hostfs
github.com/unxed/f4/vfs/hostmode
github.com/unxed/f4/vfs/hostpath
```

The restructuring adds packages to this module. A later run is compared by
**which packages went from green to red**, not by the totals: the counts grow by
design as `internal/*` packages appear.

## Module 2 — `github.com/unxed/pinned-conpty-probe` (`tools/conptyreconcile`)

`go -C tools/conptyreconcile test ./...` → exit **0**, `ok` (cached).

## Module 3 — `github.com/unxed/f4/tools/icons`

`go -C tools/icons test ./...` → exit **1**. **Pre-existing failure, not caused by
this work and not to be fixed by it.**

```
--- FAIL: TestRenderScalesStrokesAndGradients (0.00s)
    main_test.go:74: open ../../assets/icon/f4.svg: no such file or directory
```

The test builds its source path at `main_test.go:71` as
`../../assets/icon/f4.svg`, while the tool it tests reads
`<root>/cmd/f4/assets/icon` (`main.go:37`). The test never agreed with its
subject. **Do not repoint it at `cmd/f4/assets/icon` as part of the
restructuring** — that is a separate decision with a separate reviewer.

## Module 4 — `github.com/srwiley/oksvg` (`tools/icons/third_party/oksvg`)

Vendored third-party, reached only through `replace github.com/srwiley/oksvg =>
./third_party/oksvg` (`tools/icons/go.mod:10`). It has no `go.sum` of its own, so
built or tested standalone it reports `missing go.sum entry` for `rasterx`,
`golang.org/x/image` and `golang.org/x/net`, and `[setup failed]`. **Out of
scope.** It is exercised only as a dependency of module 3.

## Module 5 — `wineprobe` (`tools/wine_syscall_probe`)

| Command | Exit | Note |
|---|---|---|
| `go -C tools/wine_syscall_probe build ./...` (host: darwin/arm64) | **1** | `./main.go:5:6: missing function body` |
| `GOOS=windows GOARCH=amd64 go -C tools/wine_syscall_probe build ./...` | 0 | |

`main.go:5` declares `func rawGetpid() uint64` whose body is the amd64 assembly in
`probe_amd64.s`. **A platform probe, not breakage**: it is expected to fail on
darwin/arm64 and to build for its own target.

## Module 6 — `github.com/ebitengine/hideconsole` (`internal/hideconsole`)

`go -C internal/hideconsole build ./...` → exit **0**. Vendored fork with its own
`go.mod` (`replace` at the root `go.mod:187`), so `go list ./...` never returns
it — which is why the module boundary auditor's layer map must not contain it.

## Green but inert

`cmd/f4/plugring_policy_test.go` — `TestShippedCatalogMeetsItsOwnPolicy` reads a
CWD-relative `plugring/index.yaml` from inside `cmd/f4/`, which does not exist, so
it always reaches `t.Skipf` and asserts nothing. It counts as passing today and
will keep counting as passing after the catalogue moves. Its fate is decided by
the task that moves `plugring/`.

## Tests that cannot travel with their file

Two groups will be reworked rather than moved, so their diff is expected and is
not a regression:

- `cmd/f4/command_palette_coverage_test.go` — its 42 audit keys are re-keyed from
  file paths to qualified symbols, so that later moves do not rewrite them.
- Every test built on the `swapFrameManager` (63 files) or `setupMockPanelsFrame`
  (28 files) harness — the harness is split into `internal/testutil` and
  `internal/paneltest`, and the call shape changes.

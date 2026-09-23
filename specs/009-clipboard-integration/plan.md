# Plan: Clipboard integration

Derived from `spec.md`. Consistent with `memory/constitution.md` (Articles
II, IV, VII, VIII) and `CONVENTIONS.md`.

## Architecture / approach

`Write(ctx context.Context, text string) error` is a thin production
entrypoint that computes `goos := runtime.GOOS` and, only when
`goos == "linux"`, `wsl := isWSL()` (wsl.go), then delegates to an
unexported, fully injectable core:

```go
func write(ctx context.Context, text, goos string, wsl bool, runner cmdRunner) error
```

This mirrors `codecontext.BuildCodeContexts` wrapping `buildCodeContexts(...,
runner)`, and `cmd/all`'s `stdinIsPiped()` being computed once and passed
in as a plain `bool` rather than re-derived deeper in the call stack.
Passing `goos`/`wsl` as plain values (not an injected detector interface)
means every OS/WSL/fallback branch in spec.md's FR2, FR4-FR6 is directly
testable by calling `write()` with any combination, regardless of which
OS actually runs `go test` (spec.md's User stories: this is developed on
WSL, but CI — once it exists — may run on other OSes).

`write()`'s dispatch, per spec.md FR2/FR4-FR6:

```
switch goos {
case "darwin":  tryOne(ctx, "pbcopy", nil, text, runner)
case "windows": tryOne(ctx, "clip.exe", nil, text, runner)
case "linux":
    if wsl { tryOne(ctx, "clip.exe", nil, text, runner) }
    else   { tryChain(ctx, []tool{"wl-copy", "xclip"}, text, runner) }
default: return ErrNoClipboardUtility // unrecognized GOOS, FR6
}
```

`tryOne` and `tryChain` share the same core loop (attempt each tool in
order via `runner.LookPath`/`runner.Run`; a found-but-failed tool
continues to the next candidate, an absent one is skipped) — `tryOne` is
just `tryChain` with a single-element list, so FR6's "single candidate"
wording in spec.md is naturally the same code path as FR5's chain, not a
separate implementation. `xclip` is invoked as `xclip -selection
clipboard` (the standard invocation for writing the X11 CLIPBOARD
selection, not just the legacy PRIMARY selection) — `wl-copy`, `pbcopy`,
and `clip.exe` take no arguments for a plain copy from stdin.

Error assembly: if every attempted tool was found but failed, wrap all
their individual errors into one `ErrClipboardWriteFailed`-wrapping error
naming each tool and its failure (mirrors `parser.ErrAmbiguous`'s
message-shape precedent: name every relevant candidate, not just the
last one). If zero tools were found at all, return
`ErrNoClipboardUtility` naming which tool(s) were checked.

## Stack & versions

Standard library only: `context`, `errors`, `fmt`, `os/exec`, `strings`,
`runtime`. No new third-party dependency (Article VIII) — same reasoning
`codecontext`'s `execGitRunner` already established for shelling out.

## Data model

None. This feature defines no new exported types beyond the two sentinel
errors (spec.md FR8) — it consumes and produces plain `string`/`error`
only, never touching `internal/contract` (see spec.md's Overview).

## File / module layout

```
internal/clipboard/
├── clipboard.go       Write() production entrypoint + write() injectable
│                       core; tryOne/tryChain dispatch (FR2, FR4-FR6)
├── wsl.go              isWSL(): production WSL detection (FR3) — checks
│                       WSL_DISTRO_NAME/WSL_INTEROP env vars first (no
│                       I/O), then falls back to reading /proc/version
│                       for a case-insensitive "microsoft" match
├── runner.go           cmdRunner interface + execCmdRunner production
│                       impl (LookPath presence check + Run). Run derives
│                       FR7's bound via context.WithTimeout(ctx,
│                       clipboardTimeout) from the caller-supplied ctx --
│                       clipboardTimeout is a package-level const (5 *
│                       time.Second), mirroring codecontext/runner.go's
│                       gitTimeout/execGitRunner exactly, so the 5s value
│                       lives in exactly one declaration
├── errors.go           ErrNoClipboardUtility, ErrClipboardWriteFailed
│                       (FR8), doc-commented in ErrUnparseable's style
├── clipboard_test.go   table-driven tests over write(), one table entry
│                       per spec.md acceptance criterion, using a
│                       hand-written fakeCmdRunner (FR9) — mirrors
│                       codecontext/runner_fake_test.go's pattern
├── wsl_test.go          covers isWSL() only insofar as its internal
│                         structure allows env-var/file injection without
│                         a real WSL environment; if no clean seam exists,
│                         isWSL() stays production-only and untested
│                         directly, documented as such (same accepted
│                         shape as cmd/all's stdinIsPiped)
└── integration_test.go  //go:build integration -- real execCmdRunner
                          against real clip.exe under WSL, mirrors
                          internal/codecontext/integration_test.go's
                          pattern exactly; run via `go test -tags
                          integration ./internal/clipboard/...`, not
                          gated on CI existing (Testing strategy below)
```

## API / contracts

```go
package clipboard

// Write sends text to the OS clipboard byte-for-byte via an
// OS-appropriate subprocess (constitution Article VII). See spec.md for
// full OS/WSL/fallback/timeout behavior.
func Write(ctx context.Context, text string) error

// ErrNoClipboardUtility and ErrClipboardWriteFailed distinguish "nothing
// usable was found" from "something was found but every attempt
// failed" -- see spec.md FR8.
var ErrNoClipboardUtility error
var ErrClipboardWriteFailed error
```

No other exported surface. `write()`, `cmdRunner`, `execCmdRunner`,
`isWSL()`, `tryOne`/`tryChain` are all unexported.

## Testing strategy

- **Table-driven tests** in `clipboard_test.go`, calling `write()`
  directly (not `Write()`) with explicit `goos`/`wsl` values and a
  `fakeCmdRunner` — one table entry per spec.md acceptance criterion:
  darwin found/absent, windows found/absent, non-WSL Linux
  wl-copy-succeeds / wl-copy-fails-then-xclip-succeeds /
  wl-copy-absent-then-xclip-succeeds / both-found-both-fail /
  wl-copy-fails-xclip-absent / wl-copy-absent-xclip-fails / neither-found
  (seven entries -- "not found" and "found but failed" are distinct
  `fakeCmdRunner` setups even where the two collapse to the same
  downstream outcome), WSL clip.exe-found/absent (with wl-copy/xclip
  present but never invoked, asserted via the fake's call log),
  unrecognized `goos`, and byte-for-byte stdin fidelity.
- **`fakeCmdRunner`** (hand-written, no mocking framework per
  `CONVENTIONS.md`): per-tool-name configurable `LookPath` result and
  `Run` result (success / non-nil error / block-until-`ctx.Done()` then
  return `ctx.Err()` for the timeout case), plus a call log so a test can
  assert a tool was *never* invoked (the WSL exclusivity acceptance
  criteria depend on this, not just on the final returned error).
- **Timeout test**: a `fakeCmdRunner.Run` that blocks past FR7's 5-second
  `clipboardTimeout`, asserting `write()` returns in bounded time and
  treats it as a normal failed attempt (folds into the same error
  handling as any other failure) — proves the internal
  `context.WithTimeout(ctx, clipboardTimeout)` actually fires, the same
  property `codecontext`'s own tests prove for `gitTimeout`.
- **One build-tag-gated real-subprocess test**, `integration_test.go`,
  mirroring `internal/codecontext/integration_test.go`'s
  `//go:build integration` pattern exactly: exercises the real
  `execCmdRunner` against real `clip.exe` under WSL (the one
  environment/tool combination this dev machine can actually run),
  asserting a round-trip write succeeds. Not gated on CI existing —
  run explicitly via `go test -tags integration ./internal/clipboard/...`,
  same as 004's. `pbcopy`/`wl-copy`/`xclip` are not covered by a real
  test (spec.md's Out of scope) since none of those binaries exist on
  this machine; only the fakes exercise those paths.

## Risks & open decisions

- `isWSL()`'s exact detection heuristic (env vars checked before
  `/proc/version`, case-insensitive substring match) matches commonly
  documented WSL-detection practice, but has not been smoke-tested
  against this actual machine's real WSL2 environment during planning —
  only reasoned about, not confirmed live. Worth a quick manual check
  (a throwaway `go run` printing `isWSL()`'s result on the real dev
  machine) during T-whichever-task implements `wsl.go`, before trusting
  it silently. Flagged here rather than assumed proven, per Article VI's
  spirit even though this isn't bundle-output-facing.
- `xclip -selection clipboard` vs. other invocation shapes (e.g.
  `-i -selection clipboard`) is asserted from common documented usage,
  not from a real invocation on this machine (no CI, no real-subprocess
  test per spec.md) — a real-argument-shape bug wouldn't be caught by
  the fake-only test suite. Explicitly accepted for v1 (spec.md's
  Non-functional requirements), not silently assumed safe.

## Alternatives considered

- **Injecting a WSL-detector interface**, mirroring `cmdRunner`, instead
  of a plain `bool` computed once in `Write` — rejected: the OS/WSL
  classification is a one-time, side-effect-free decision made before
  any subprocess call, not a repeated operation needing its own
  interface. A plain `bool` parameter to `write()` is simpler and
  sufficient, exactly mirroring `cmd/all`'s `stdinIsPiped()` pattern.
- **A single combined sentinel error** instead of two — rejected per
  spec.md FR8's interrogation decision (a future 002b needs to
  distinguish exit codes).
- **Presence-only (`LookPath`-only) Linux fallback**, no runtime-failure
  fallback from `wl-copy` to `xclip` — rejected per spec.md FR5's
  interrogation decision.
- **Configurable timeout via `.env`** — considered and rejected; full
  reasoning in spec.md's Non-functional requirements.
- **Skipping a build-tag-gated real-subprocess test entirely, on the
  reasoning that no CI exists to run it** — rejected: `004`'s own
  `internal/codecontext/integration_test.go` already proves that
  reasoning wrong, since it's run explicitly (`go test -tags integration
  ./...`) rather than by CI. Instead, a `clip.exe`-only integration test
  is added (Testing strategy below) — the one tool this feature can
  actually exercise for real on the dev machine; `pbcopy`/`wl-copy`/
  `xclip` stay fake-only since none of those three binaries exist on
  this machine (spec.md's Out of scope).

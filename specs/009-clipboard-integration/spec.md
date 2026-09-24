# Spec: Clipboard integration

**Status:** Approved
**Folder:** specs/009-clipboard-integration
**Depends on:** none

## Overview

Writes a final, already-rendered bundle string (007 Markdown or 008 JSON
output) to the OS clipboard via an OS-appropriate subprocess, per
constitution Article VII (shell out, don't embed). This is the last stage
of the detect → parse → render → clipboard pipeline that 002b will wire
together; this feature only builds the write function itself, not the
wiring.

Unlike 007/008, this feature has no dependency on `internal/contract` at
all: it operates on a plain string (whatever a renderer already produced),
never on a `contract.Bundle` — confirmed by reading `render.JSON`'s and
`render.Markdown`'s signatures, both of which already return `string`.

## User stories

- As a developer who just ran `stack-trace-bundler` on a real trace, I
  want the rendered bundle already sitting in my clipboard, so I can paste
  it straight into an AI chat without an extra manual copy step.
- As a developer working inside WSL (the actual environment this project
  is developed in), I want the tool to bridge to the Windows clipboard via
  `clip.exe`, so clipboard integration actually works in my real
  environment instead of silently failing because `wl-copy`/`xclip` have
  no display server to talk to.
- As a developer on native Linux with a Wayland compositor running, I want
  `wl-copy` used directly, with `xclip` only as a fallback, so the tool
  works correctly on both Wayland and X11 desktops without extra
  configuration.

## Functional requirements

1. Public API: `func Write(ctx context.Context, text string) error` in
   package `clipboard`, file `internal/clipboard/clipboard.go` (per
   `CONVENTIONS.md`'s file/folder layout: `internal/clipboard/` — OS-
   appropriate subprocess write). `Write` sends `text` to the OS clipboard
   byte-for-byte: no added/stripped newline, no re-encoding, no
   reformatting of any kind — this feature is not a renderer and adds no
   shape/format logic of its own (Article IV's discipline applied here:
   the renderer is the sole source of the string's exact bytes).
2. OS dispatch is based on `runtime.GOOS` (the same three values
   `contract.OS` already models — `darwin`/`linux`/`windows`):
   - `darwin`: `pbcopy`.
   - `windows`: `clip.exe`.
   - `linux`: first determine whether the process is running inside WSL
     (FR3). If so, use `clip.exe` exclusively (FR4). If not, use the
     `wl-copy` → `xclip` fallback chain (FR5).
3. WSL detection: the process is considered to be running inside WSL if
   either `WSL_DISTRO_NAME` or `WSL_INTEROP` is set in the environment, or
   `/proc/version`'s contents contain `"microsoft"` (case-insensitive) —
   the combination of environment-variable and kernel-version-string
   checks that WSL-detection utilities commonly use. Exact implementation
   (which check runs first, how it's made unit-testable without a real
   WSL environment) is `plan.md`'s concern, not decided here.
4. On WSL: `clip.exe` is resolved via `PATH` (`exec.LookPath`). `wl-copy`
   and `xclip` are never attempted on WSL, even if present on `PATH` —
   this is a deliberate, real-environment-driven choice (see User
   stories), not an oversight. If `clip.exe` is not found, `Write` returns
   an error satisfying `errors.Is(err, ErrNoClipboardUtility)` (FR8).
5. On non-WSL Linux: `wl-copy` is tried first. `wl-copy` is
   **unavailable** if it is either not found via `PATH`, or found but its
   invocation fails (non-zero exit, or the FR7 timeout elapses) — both
   count as the same "move to the next candidate" outcome, not two
   separate mechanisms; the failing case is what makes this a
   runtime-failure fallback rather than merely a presence check.
   Whenever `wl-copy` is unavailable, `xclip` is tried next. `Write`
   returns:
   - `ErrNoClipboardUtility` (FR8) if neither `wl-copy` nor `xclip` is
     found via `PATH` at all (both absent — no subprocess is ever
     invoked).
   - `ErrClipboardWriteFailed` (FR8) if at least one of `wl-copy`/`xclip`
     was found via `PATH` and invoked, but every tool that was actually
     invoked failed — regardless of whether the other tool also failed
     or was simply never found. The error names only the tool(s) that
     were actually found and invoked, and how each failed; a tool that
     was never found (because the other one already succeeded, or
     because it itself wasn't on `PATH`) is not named as a "failure."
6. On `darwin`/`windows`, the single candidate tool (`pbcopy`/`clip.exe`
   respectively) is resolved via `PATH` the same way: `ErrNoClipboardUtility`
   if not found, `ErrClipboardWriteFailed` if found but its invocation
   fails — the single-candidate case of FR5's failure semantics, not a
   separate mechanism. Any `runtime.GOOS` value other than
   `darwin`/`linux`/`windows` immediately returns `ErrNoClipboardUtility`
   with no attempt made — this project only supports the three OSes
   `contract.OS` already models (constitution's closed-enum precedent).
7. Every subprocess invocation (`pbcopy`/`wl-copy`/`xclip`/`clip.exe`) is
   bounded by a fixed 5-second timeout, applied independently per attempt
   via `context.WithTimeout` on the caller-supplied `ctx` — mirrors
   `codecontext`'s `gitTimeout` pattern (`internal/codecontext/runner.go`).
   A timeout counts as that attempt failing, folding into FR5/FR6's
   existing failure handling rather than being a distinct error type.
   This is a fixed, unexported constant for v1 (see Non-functional
   requirements) — not configurable.
8. `internal/clipboard` exposes two sentinel errors, in
   `internal/clipboard/errors.go` (mirrors `internal/parser/errors.go`'s
   `ErrUnparseable`/`ErrNoMatch`/`ErrAmbiguous` pattern):
   - `ErrNoClipboardUtility`: no usable clipboard tool could be found on
     `PATH` for the current OS/environment.
   - `ErrClipboardWriteFailed`: at least one clipboard tool was found on
     `PATH` but every attempt to invoke it failed (non-zero exit or
     timeout).
   A future caller (002b) distinguishes the two via `errors.Is` to map
   them to distinct CLI exit codes — which exit code(s), if any, get
   allocated is 002b's decision, not this feature's (mirrors 003b's
   `ErrNoMatch`/`ErrAmbiguous` precedent).
9. The actual subprocess-invoking logic sits behind a small injectable
   interface, mirroring `codecontext`'s `gitRunner` (`internal/codecontext/runner.go`).
   `internal/clipboard`'s own tests run entirely against a hand-written
   fake implementing that interface and never require
   `pbcopy`/`wl-copy`/`xclip`/`clip.exe` to actually be installed
   (`CONVENTIONS.md`'s testing section).

## Non-functional requirements

- No size limit is enforced by this package on `text` — whatever length
  limits exist are the OS clipboard/underlying tool's own concern, not
  something this feature second-guesses (Article VIII).
- No read-back or verification that a write actually landed in the system
  clipboard — `Write` reports subprocess success/failure only.
- The FR7 timeout (5s) is a fixed, unexported Go constant for v1 — not
  configurable via CLI flag, environment variable, or any config file.
  Considered and explicitly rejected during interrogation: a `.env`-based
  config mechanism, because none exists anywhere in this codebase today
  (confirmed via `go.mod` — only dependency is `pflag`), it would require
  either a new third-party dependency or a hand-rolled parser
  (`AGENTS.md`'s Boundaries: never add a new dependency without asking),
  and it doesn't fit Article I's CLI-first/scriptable shape (undefined
  file location, undefined precedence). If the fixed value ever proves
  wrong from real usage, making it configurable becomes its own future
  feature exposed as a CLI flag — the same pattern feature **011**
  already uses to make 004's fixed snippet-window size configurable
  later, rather than building speculative configurability into 004
  itself (Article VIII).
- A build-tag-gated real-subprocess test is added for the one tool
  actually exercisable on the real machine this project is developed on
  (`clip.exe` under WSL — see User stories), mirroring
  `004-own-code-context-extraction`'s existing
  `internal/codecontext/integration_test.go` precedent: that test proves
  a build-tag-gated test does not need CI to be useful, since it's run
  explicitly (`go test -tags integration ./...`), not automatically —
  the earlier reasoning here ("no CI means it would never run") was
  factually wrong given that precedent already exists in this repo.
  `pbcopy`/`wl-copy`/`xclip` remain untested against the real binaries
  for this feature (this machine can't exercise them), which is a
  genuine, narrower gap than "no real-subprocess tests at all" — revisit
  when a machine/CI runner that has them becomes available (see Out of
  scope).

## Out of scope

- Wiring `Write` into `cmd/all`/`cmd/java`/`cmd/typescript` or the full
  detect → parse → render → clipboard pipeline — 002b's job (`idea`
  status), same carve-out 007/008 already used for their own output.
- Any CI workflow — this repo has no CI today; standing this up is a
  separate concern from this feature, not built here. Flagged so it
  isn't silently forgotten, but not added to `memory/known-gaps.md`
  since it isn't a deferred acceptance criterion or an accepted
  limitation of this feature specifically — it's a repo-wide gap that
  predates this feature.
- A real-subprocess integration test for `pbcopy`/`wl-copy`/`xclip`
  specifically — unlike `clip.exe` (Non-functional requirements above),
  none of these three binaries are available on the machine this
  project is developed on, so no real test against them is added by
  this feature.
- A CLI flag, environment variable, or config file to make the FR7
  timeout configurable — deferred, same pattern as feature **011**
  (`idea` status) deferring `--context-lines` configurability out of 004.
- Read-back/verification that a write actually reached the system
  clipboard.
- Remote/SSH clipboard bridging (e.g. OSC 52 terminal escape sequences) —
  local OS clipboard only.
- Any platform other than `darwin`/`linux` (WSL or not)/`windows` —
  matches `contract.OS`'s closed three-value enum; treated identically to
  "no clipboard utility available" (FR6), with no dedicated handling.

## Acceptance criteria

- [x] Given `darwin` with `pbcopy` present (fake runner), when `Write` is
      called, then `pbcopy` is invoked with `text` on stdin and `Write`
      returns `nil`. (`TestWrite/darwin:_pbcopy_found`)
- [x] Given `darwin` with `pbcopy` absent, when `Write` is called, then it
      returns an error for which `errors.Is(err, ErrNoClipboardUtility)`
      is `true`. (`TestWrite/darwin:_pbcopy_absent`)
- [x] Given `windows` with `clip.exe` present, when `Write` is called,
      then `clip.exe` is invoked with `text` on stdin and `Write` returns
      `nil`. (`TestWrite/windows:_clip.exe_found`)
- [x] Given `windows` with `clip.exe` absent, when `Write` is called, then
      it returns an error for which
      `errors.Is(err, ErrNoClipboardUtility)` is `true`.
      (`TestWrite/windows:_clip.exe_absent`)
- [x] Given non-WSL Linux with `wl-copy` found and its invocation
      succeeding, when `Write` is called, then `wl-copy` is invoked,
      `xclip` is never attempted, and `Write` returns `nil`.
      (`TestWrite/linux_non-WSL:_wl-copy_found+succeeds,_xclip_never_attempted`)
- [x] Given non-WSL Linux with `wl-copy` unavailable (either not found on
      `PATH`, or found but its invocation failing) and `xclip` found and
      its invocation succeeding, when `Write` is called, then `xclip` is
      invoked and `Write` returns `nil`.
      (`TestWrite/linux_non-WSL:_wl-copy_found+fails,_xclip_found+succeeds`
      and `TestWrite/linux_non-WSL:_wl-copy_not_found,_xclip_found+succeeds`
      -- both permutations of "unavailable")
- [x] Given non-WSL Linux with `wl-copy` unavailable and `xclip` also
      unavailable, in any combination of "not found on `PATH`" and
      "found but its invocation failing" for the two tools, when `Write`
      is called, then it returns an error for which
      `errors.Is(err, ErrClipboardWriteFailed)` is `true` (naming exactly
      the tool(s) that were found and actually invoked, and how each
      failed) if at least one of the two was found on `PATH`, or for
      which `errors.Is(err, ErrNoClipboardUtility)` is `true` if neither
      was found on `PATH` at all.
      (`TestWrite/linux_non-WSL:_both_found,_both_fail`,
      `TestWrite/linux_non-WSL:_wl-copy_found+fails,_xclip_not_found`,
      `TestWrite/linux_non-WSL:_wl-copy_not_found,_xclip_found+fails`
      for `ErrClipboardWriteFailed`;
      `TestWrite/linux_non-WSL:_neither_found` for `ErrNoClipboardUtility`)
- [x] Given a simulated WSL environment with `clip.exe` present, when
      `Write` is called, then `clip.exe` is invoked directly and
      `wl-copy`/`xclip` are never attempted, even if also present.
      (`TestWrite/linux_WSL:_clip.exe_found+succeeds,_wl-copy/xclip_never_attempted`)
- [x] Given a simulated WSL environment with `clip.exe` absent, when
      `Write` is called, then it returns an error for which
      `errors.Is(err, ErrNoClipboardUtility)` is `true`, with no fallback
      to `wl-copy`/`xclip` attempted.
      (`TestWrite/linux_WSL:_clip.exe_absent,_no_fallback_attempted`)
- [x] Given a fake runner that blocks past the FR7 timeout, when `Write`
      is called, then that attempt is treated as a failure (folds into
      the same `ErrClipboardWriteFailed`/`ErrNoClipboardUtility` handling
      as any other failed attempt), and `Write` returns in bounded time.
      (`TestWrite_Timeout`)
- [x] Given any successful `Write`, the exact bytes handed to the
      underlying tool's stdin equal `text` exactly — no added or removed
      trailing newline, no re-encoding. (`TestWrite_ByteForByte`)
- [x] Given an unrecognized simulated `runtime.GOOS` value, when `Write`
      is called, then it returns `ErrNoClipboardUtility` immediately, with
      no subprocess invocation attempted.
      (`TestWrite/unrecognized_goos:_no_attempt_made`)

## Open questions

None remaining — all resolved during interrogation. See
`specs/009-clipboard-integration/progress.md` for the session log,
including the rejected `.env`-based timeout-configuration proposal and
why (Non-functional requirements above).

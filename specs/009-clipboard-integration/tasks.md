# Tasks: Clipboard integration

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.
Each task lists the `spec.md` functional requirements it satisfies, so a
failing acceptance criterion later traces back to exactly one task.

- [x] **T001 — Sentinel errors** (spec.md FR8)
      - Depends on: none
      Create `internal/clipboard/errors.go` with `ErrNoClipboardUtility`
      and `ErrClipboardWriteFailed`, doc-commented in
      `internal/parser/errors.go`'s style (what each means, which future
      caller/exit-code concern each maps to).
      - Acceptance: file compiles; `go build ./...`, `gofumpt -l
      ./internal/clipboard/`, `golangci-lint run ./internal/clipboard/...`
      clean.

- [ ] **T002 — `cmdRunner` interface + `execCmdRunner`** (spec.md FR7, FR9)
      - Depends on: none
      Create `internal/clipboard/runner.go`: the `cmdRunner` interface
      (`LookPath(name string) bool`, `Run(ctx context.Context, name
      string, args []string, stdin string) error`) and the production
      `execCmdRunner`, mirroring `internal/codecontext/runner.go`'s
      `gitRunner`/`execGitRunner` exactly: a package-level
      `const clipboardTimeout = 5 * time.Second` (mirrors `gitTimeout`
      in `internal/codecontext/runner.go` — one named constant to edit,
      not a literal to hunt for across files), and `Run` derives its
      bound via `context.WithTimeout(ctx, clipboardTimeout)` **from the
      caller-supplied `ctx`** — the same call shape as
      `execGitRunner.Run`'s `context.WithTimeout(ctx, gitTimeout)` — so
      the 5s ceiling stacks on top of whatever deadline/cancellation the
      caller's `ctx` already carries; it does not replace or ignore it.
      The timeout-vs-generic-error priority (deadline error wins over
      the generic exec error when `callCtx.Err() != nil`) mirrors
      `execGitRunner.Run`'s existing handling exactly.
      - Acceptance: `go build ./...`, 4 verification commands clean; no
      test yet (T005's fake supersedes real-runner testing per spec.md's
      Non-functional requirements — no CI to run a real-subprocess test
      against).

- [ ] **T003 — `isWSL()` detection** (spec.md FR3)
      - Depends on: none
      Create `internal/clipboard/wsl.go`: `isWSL() bool`, checking
      `WSL_DISTRO_NAME`/`WSL_INTEROP` env vars first, then falling back
      to a case-insensitive `"microsoft"` substring check against
      `/proc/version`'s contents. If a clean seam exists for injecting
      env/file reads without a real WSL environment, add
      `wsl_test.go` covering both signals; otherwise leave it
      production-only and note why in a doc comment (plan.md's File
      layout section already anticipates this either way).
      - Acceptance: `go build ./...`, 4 verification commands clean;
      per plan.md's Risks section, manually run a throwaway check on the
      real dev machine (WSL) confirming `isWSL()` actually returns
      `true` there — record the result in `progress.md`, don't just
      trust the reasoning.

- [ ] **T004 — `write()` core + `Write()` entrypoint, darwin/windows only**
      (spec.md FR1, FR2, FR6, FR8 partial)
      - Depends on: T001, T002, T003 (the public `Write()` wrapper calls
      `isWSL()`, defined in T003, when `goos == "linux"`)
      Create `internal/clipboard/clipboard.go`: `write(ctx, text, goos
      string, wsl bool, runner cmdRunner) error` with the `tryOne`/
      `tryChain` shared-loop helper (plan.md's Architecture), wired for
      the `darwin` (`pbcopy`) and `windows` (`clip.exe`) branches and the
      `default` (unrecognized `goos` → `ErrNoClipboardUtility`, FR6)
      branch only — the `linux` branch is a stub returning
      `ErrNoClipboardUtility` for now, completed in T006/T007. Add the
      public `Write(ctx, text) error` wrapper computing `runtime.GOOS`
      (and `isWSL()` only when `goos == "linux"`) and delegating to
      `write()`.
      - Acceptance: `go build ./...`, 4 verification commands clean; no
      test yet (T005 adds the harness this needs).

- [ ] **T005 — `fakeCmdRunner` harness + darwin/windows table tests**
      (spec.md acceptance criteria: darwin found/absent, windows
      found/absent)
      - Depends on: T004
      Create `internal/clipboard/clipboard_test.go`: hand-written
      `fakeCmdRunner` (per-tool-name configurable `LookPath`/`Run`
      results, plus a call log recording every tool actually invoked —
      built now even though the WSL-exclusivity assertions that need the
      call log land in T007, since retrofitting it later would mean
      touching every earlier test case). Add the first table-driven
      test, `TestWrite`, with darwin-found, darwin-absent, windows-found,
      windows-absent entries, and an unrecognized-`goos` entry (FR6).
      - Acceptance: `TestWrite`'s 5 entries pass; 4 verification commands
      clean.

- [ ] **T006 — Linux non-WSL fallback chain** (spec.md FR5)
      - Depends on: T004, T005
      Implement the `linux`/non-WSL branch in `write()` using `tryChain`
      over `["wl-copy", "xclip"]` (with `xclip -selection clipboard`,
      per plan.md). Add `TestWrite` entries covering every FR5
      permutation, not just the ones that happen to differ in outcome --
      "not found" and "found but failed" are different `fakeCmdRunner`
      setups exercising different branches (`LookPath` vs `Run`) even
      when the downstream behavior is the same:
      - wl-copy found+succeeds → xclip never attempted (assert via call
        log), `Write` returns `nil`.
      - wl-copy found+fails → xclip found+succeeds → `Write` returns
        `nil`.
      - wl-copy not found → xclip found+succeeds → `Write` returns
        `nil`.
      - wl-copy found+fails, xclip found+fails →
        `ErrClipboardWriteFailed` naming both tools' failures.
      - wl-copy found+fails, xclip not found →
        `ErrClipboardWriteFailed` naming only wl-copy's failure (xclip
        was never found, so it is not named as a "failure").
      - wl-copy not found, xclip found+fails →
        `ErrClipboardWriteFailed` naming only xclip's failure.
      - wl-copy not found, xclip not found → `ErrNoClipboardUtility`.
      - Acceptance: all 7 new `TestWrite` entries pass, including the
      call-log assertion that xclip is never invoked in the
      wl-copy-succeeds case; 4 verification commands clean.

- [ ] **T007 — WSL branch + exclusivity** (spec.md FR4)
      - Depends on: T004, T005
      Implement the `linux`/`wsl == true` branch in `write()`: `clip.exe`
      only, via `tryOne`, never falling through to `wl-copy`/`xclip`. Add
      `TestWrite` entries: WSL + clip.exe found+succeeds (with
      wl-copy/xclip also configured as available in the fake, asserting
      via the call log that neither is ever invoked), WSL + clip.exe
      absent (`ErrNoClipboardUtility`, no fallback attempted — same
      call-log assertion).
      - Acceptance: both new entries pass, including both call-log
      assertions; 4 verification commands clean.

- [ ] **T008 — Timeout handling + byte-fidelity test** (spec.md FR7,
      part of FR1)
      - Depends on: T004, T005
      Add a `fakeCmdRunner.Run` variant that blocks until `ctx.Done()`
      then returns `ctx.Err()`, and a `TestWrite_Timeout` (or table
      entry) asserting `write()` returns in bounded time and treats the
      timeout as a normal failed attempt, per plan.md's Testing
      strategy. Add `TestWrite_ByteForByte` (or equivalent), asserting
      the fake's recorded stdin for the invoked tool equals the input
      `text` exactly — no added/stripped newline, no re-encoding (FR1).
      - Acceptance: both new tests pass; 4 verification commands clean.

- [ ] **T009 — Real `clip.exe` integration test (WSL)** (plan.md's
      Testing strategy; spec.md Non-functional requirements)
      - Depends on: T002, T007
      Create `internal/clipboard/integration_test.go`, gated behind
      `//go:build integration`, mirroring
      `internal/codecontext/integration_test.go`'s pattern exactly:
      calls the real `execCmdRunner` (T002) against real `clip.exe`
      under WSL and asserts the call succeeds -- the one real
      subprocess/environment combination actually available on this
      dev machine. Not gated on CI existing; run explicitly via
      `go test -tags integration ./internal/clipboard/...`, same as
      004's. Does not attempt `pbcopy`/`wl-copy`/`xclip` (none of those
      binaries exist on this machine -- spec.md's Out of scope).
      - Acceptance: `go test -tags integration ./internal/clipboard/...`
      passes on this machine; excluded from the default
      `go test ./...` run (verify via a plain `go test ./...` still
      passing with no WSL/clip.exe dependency).

- [ ] **T010 — Acceptance criteria review pass**
      - Depends on: T001-T009
      Re-read `spec.md`'s Acceptance criteria list top to bottom; confirm
      each has an exact corresponding passing test, recorded as an
      inline comment next to each checkbox (008's T008 pattern). Update
      `spec.md`'s Status line if needed and `specs/INDEX.md`'s 009 row
      from `planned` to `done`.
      - Acceptance: every acceptance criterion in `spec.md` checked off
      with a named test; `specs/INDEX.md`'s 009 row updated.

<!-- Wiring internal/clipboard.Write into cmd/all/cmd/java/cmd/typescript
     is explicitly out of scope for this feature (spec.md) -- that's
     002b's job, not a task here. -->

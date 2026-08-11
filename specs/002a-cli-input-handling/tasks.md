# Tasks: CLI Input Handling

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.
Do not start a task without explicit authorization, per the project's
standing rules.

- [x] **T001** — Scaffold `internal/cli` package: `Input` struct in
  `input.go`, package doc comment.
  - Depends on: none
  - Acceptance: package compiles; `Input` has `RawText`,
    `RawInputTruncated`, `LangHint`, `Format` fields, `omitempty`-style
    doc comments explaining `LangHint == ""` semantics per Article VI.

- [x] **T002** — Implement `readTrace` (`read.go`): file-arg vs. stdin
  source selection, both-present precedence (file wins, returns
  `stdinIgnored=true` — `readTrace` itself does not log, per plan.md),
  no-input TTY fast-fail, bounded read (`io.LimitReader` ~1MB), calls
  `contract.TruncateRawInput`, empty/whitespace-only detection. Unit
  tests in `read_test.go` with fake readers and `t.TempDir()` files.
  - Depends on: T001
  - Acceptance: `go test ./internal/cli/... -run TestReadTrace` passes,
    covering: file-only, stdin-only, both-present (`stdinIgnored=true`),
    neither-present (TTY=false), file-not-found, empty content,
    whitespace-only content, input > 512KB (truncated + flag set).

- [x] **T003** — Implement `validateFormat` and `validateLang`
  (`parse.go`), each returning a wrapped, specific error on invalid
  input. Unit tests.
  - Depends on: T001
  - Acceptance: valid values pass through unchanged; invalid values
    return an error whose message names the bad value and the accepted
    set.

- [x] **T004** — Implement `ParseAll` (`parse.go`): registers `--lang`
  and `--format` in `flag.ContinueOnError` mode, validates flags **before**
  calling `readTrace` (fixed order, per plan.md), wires in `readTrace` +
  both validators, logs `Debug` "stdin ignored" when `readTrace` returns
  `stdinIgnored=true`, returns wrapped/distinct errors per spec
  requirement 11. Unit tests.
  - Depends on: T002, T003
  - Acceptance: `go test ./internal/cli/... -run TestParseAll` passes,
    covering valid combinations, invalid `--lang`, invalid `--format`,
    each `readTrace` failure mode surfacing through with a distinct
    message, the "stdin ignored" Debug log firing on `stdinIgnored=true`,
    and a combined-failure case (invalid flag + no input available)
    confirming the flag error wins.

- [x] **T005** — Implement `ParseFixedLang` (`parse.go`): registers only
  `--format`, validates it **before** calling `readTrace` (fixed order,
  per plan.md), fixes `LangHint` to the given language, rejects a passed
  `--lang` via the flag package's "not defined" error, logs `Debug`
  "stdin ignored" when `readTrace` returns `stdinIgnored=true`. Unit
  tests.
  - Depends on: T002, T003
  - Acceptance: `go test ./internal/cli/... -run TestParseFixedLang`
    passes, covering `lang="java"` and `lang="typescript"`, valid/invalid
    `--format`, `--lang` being rejected, and the "stdin ignored" Debug
    log firing on `stdinIgnored=true`.

- [x] **T006** — Implement log-level helper: maps `-v`/`-vv` occurrence
  count to `Warn`/`Info`/`Debug`, shared by all three `main.go` files.
  Unit tests on the pure mapping function.
  - Depends on: T001
  - Acceptance: 0 occurrences → `Warn` (default), 1 → `Info`, 2+ →
    `Debug`.

- [x] **T006b** — Register `-v`/`-vv` directly on `ParseAll`/
  `ParseFixedLang`'s existing `flag.FlagSet` (alongside `--lang`/
  `--format`), replacing the originally-assumed "main.go parses `-v`/`-vv`
  separately" approach discovered to be ambiguous once T007 design started
  (see plan.md's Architecture / approach and Alternatives considered for
  the full rationale). Change both signatures to
  `(Input, int, error)`; add `verbosityFromFlags(v, vv bool) int` in
  `log.go`. Scope is `internal/cli` only — no `cmd/*/main.go` changes
  here, those stay T007/T008's job. Update `TestParseAll` and
  `TestParseFixedLang` call sites for the new signature; add cases for
  `-v`/`-vv`/both/neither, order-independence relative to `--lang`/
  `--format`, and a positional arg literally named `-v` after a `--`
  terminator not being misread as the flag.
  - Depends on: T004, T005
  - Acceptance: `go test ./internal/cli/... -run 'TestParseAll|TestParseFixedLang'`
    passes with the updated signatures; verbosity is `0`/`1`/`2` per
    plan.md's API/contracts section on every success case, always `0` on
    every error case; flag order relative to `--lang`/`--format` doesn't
    change the result; `-- -v` as a positional arg is not treated as the
    verbosity flag.

- [x] **T006c** — Fix a logging-ordering bug found while designing T007:
  `ParseAll`/`ParseFixedLang` no longer call `slog.Debug` internally for
  the "stdin ignored" case (their internal call happened before `main.go`
  could configure `slog`'s level from the returned verbosity, so the
  message could never actually be observed under any `-v`/`-vv` setting
  — see plan.md's Architecture section for the full trace). Add
  `Input.StdinIgnored bool` (`input.go`); `ParseAll`/`ParseFixedLang` set
  it instead of logging; drop the now-unused `log/slog` import from
  `parse.go`. Replace `TestParseAll_StdinIgnoredLogging` and
  `TestParseFixedLang_StdinIgnoredLogging` (which asserted on captured
  `slog` output) with plain `StdinIgnored` field checks folded into
  `TestParseAll`/`TestParseFixedLang`'s existing tables; add a
  both-file-and-stdin-present case to `TestParseAll`'s table (previously
  only exercised at the `readTrace` level in T002, never through
  `ParseAll` itself).
  - Depends on: T004, T005, T006b
  - Acceptance: `go test ./internal/cli/...` passes with no test in the
    package touching `slog.SetDefault`/buffer capture anywhere;
    `Input.StdinIgnored` is `true` exactly when a file arg was given
    while stdin was also piped, `false` otherwise, verified via both
    `ParseAll` and `ParseFixedLang`'s tables.

- [~] **T007** — Wire `cmd/all/main.go`: TTY-detection one-liner
  (`os.Stdin.Stat()`), call `cli.ParseAll` (now returning
  `(Input, int, error)` per T006b), on error `slog.Error` + `os.Exit(2)`,
  on success call `cli.LogLevel(verbosity)` to configure `slog`'s default
  handler, then (in that order) log the "stdin ignored" Debug line if
  `input.StdinIgnored` (per T006c — this only works now that the handler
  is configured before this call, not during `ParseAll`), the Info
  summary, and the Debug dump gated by the configured level. Confirm
  nothing is written to stdout.

  **In progress, blocked on T007a.** `main.go` itself is written; the
  manual run-through it depends on surfaced a real bug in `ParseAll`
  (below T007a), not in `main.go`'s own wiring. Re-run the run-through
  once T007a lands before marking this done — `main.go` may not need any
  further changes itself, but that needs confirming, not assuming.
  - Depends on: T004, T006, T006b, T006c, T007a
  - Acceptance: manual run-through — paste back terminal output for: a
    valid file, valid piped stdin, `--lang=cobol`, no input on a real
    terminal (Ctrl+D / no pipe), a file > 512KB, `-v`/`-vv` actually
    changing what's logged, and both a file arg + piped stdin together
    with `-vv` (confirming the "stdin ignored" Debug line actually
    appears now — this was unobservable before the T006c fix). Confirm
    stdout is empty in every case except normal shell prompt return.

- [x] **T007a** — Fix a flag/positional-argument ordering bug found
  during T007's manual run-through: stdlib `flag.Parse` stops
  recognizing flags once it hits the first positional argument, so
  `stba trace.txt -vv` silently dropped `-vv` entirely — exactly the
  natural way most people type a command, and exactly what T007's own
  acceptance criteria exercises. Full analysis and decision in
  `memory/decisions/0002-adopt-pflag-for-flag-parsing.md`. Replace
  stdlib `flag` with `github.com/spf13/pflag` in `ParseAll`/
  `ParseFixedLang` (`parse.go`): `pflag.FlagSet` permutes flags and the
  positional argument in any order and correctly honors `--`. `-v`/`-vv`
  become a single `pflag.CountVarP` shorthand (POSIX clustering: `-v`=1,
  `-vv`=2) instead of two separate bool flags — `verbosityFromFlags`
  (T006b) is removed as dead code; `LogLevel` (`log.go`) needs no change,
  its existing `default` case already treats any count `>= 2` as `Debug`.
  Re-verify all of `parse_test.go` against the new library's actual
  behavior (error message text, `--` handling, order-independence), and
  add a new case placing `-v`/`-vv`/`--lang`/`--format` *after* the
  positional file-path argument — the exact case that was silently
  broken and went undetected until T007's manual run-through caught it.
  - Depends on: T004, T005, T006b, T006c
  - Acceptance: `go get github.com/spf13/pflag` run, `go.mod`/`go.sum`
    updated; `go test ./internal/cli/...` passes, including the new
    flag-after-positional-argument case; `specs/002a-cli-input-handling/verify-t007.sh`
    re-run end to end using the same command shapes that previously
    produced no output for `-v`/`-vv` (e.g. `stba file.txt -vv`),
    confirmed passing this time.

- [ ] **T008** — Wire `cmd/java/main.go` and `cmd/typescript/main.go`
  the same way, calling `cli.ParseFixedLang` (now returning
  `(Input, int, error)` per T006b, `Input.StdinIgnored` per T006c).
  - Depends on: T005, T006, T006b, T006c, T007a
  - Acceptance: manual run-through — paste back terminal output for a
    valid run on each binary, and `--lang` being rejected on each.

- [ ] **T009** — Full acceptance pass: walk every checkbox in
  `spec.md`'s Acceptance Criteria section against the real binaries,
  check them off, append a `progress.md` entry summarizing the session
  and any deviations from `plan.md`.
  - Depends on: T007, T008
  - Acceptance: every acceptance-criteria checkbox in `spec.md` is either
    checked or explicitly noted as deferred with a reason;
    `progress.md` updated.

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

- [ ] **T007** — Wire `cmd/all/main.go`: TTY-detection one-liner
  (`os.Stdin.Stat()`), call `cli.ParseAll` (now returning
  `(Input, int, error)` per T006b), on error `slog.Error` + `os.Exit(2)`,
  on success call `cli.LogLevel(verbosity)` to configure `slog`'s default
  handler, log Info summary / Debug dump per the resulting level, confirm
  nothing is written to stdout.
  - Depends on: T004, T006, T006b
  - Acceptance: manual run-through — paste back terminal output for: a
    valid file, valid piped stdin, `--lang=cobol`, no input on a real
    terminal (Ctrl+D / no pipe), a file > 512KB, and `-v`/`-vv` actually
    changing what's logged. Confirm stdout is empty in every case except
    normal shell prompt return.

- [ ] **T008** — Wire `cmd/java/main.go` and `cmd/typescript/main.go`
  the same way, calling `cli.ParseFixedLang` (now returning
  `(Input, int, error)` per T006b).
  - Depends on: T005, T006, T006b
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

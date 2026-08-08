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

- [ ] **T002** — Implement `readTrace` (`read.go`): file-arg vs. stdin
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

- [ ] **T003** — Implement `validateFormat` and `validateLang`
  (`parse.go`), each returning a wrapped, specific error on invalid
  input. Unit tests.
  - Depends on: T001
  - Acceptance: valid values pass through unchanged; invalid values
    return an error whose message names the bad value and the accepted
    set.

- [ ] **T004** — Implement `ParseAll` (`parse.go`): registers `--lang`
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

- [ ] **T005** — Implement `ParseFixedLang` (`parse.go`): registers only
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

- [ ] **T006** — Implement log-level helper: maps `-v`/`-vv` occurrence
  count to `Warn`/`Info`/`Debug`, shared by all three `main.go` files.
  Unit tests on the pure mapping function.
  - Depends on: T001
  - Acceptance: 0 occurrences → `Warn` (default), 1 → `Info`, 2+ →
    `Debug`.

- [ ] **T007** — Wire `cmd/all/main.go`: TTY-detection one-liner
  (`os.Stdin.Stat()`), call `cli.ParseAll`, on error `slog.Error` +
  `os.Exit(2)`, on success log Info summary / Debug dump per `-v`/`-vv`,
  confirm nothing is written to stdout.
  - Depends on: T004, T006
  - Acceptance: manual run-through — paste back terminal output for: a
    valid file, valid piped stdin, `--lang=cobol`, no input on a real
    terminal (Ctrl+D / no pipe), a file > 512KB. Confirm stdout is empty
    in every case except normal shell prompt return.

- [ ] **T008** — Wire `cmd/java/main.go` and `cmd/typescript/main.go`
  the same way, calling `cli.ParseFixedLang`.
  - Depends on: T005, T006
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

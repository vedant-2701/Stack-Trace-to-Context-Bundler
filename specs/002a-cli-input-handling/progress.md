# Progress Log: CLI Input Handling

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:**
**Task(s):**
**What happened:**
**Deviations from plan (if any):**
**New open questions:**

---

**Date:** 2026-08-08
**Task(s):** T001
**What happened:** Scaffolded `internal/cli` package: created `input.go` with the package doc comment and the `Input` struct (`RawText`, `RawInputTruncated`, `LangHint`, `Format`), per tasks.md's acceptance criteria.
**Deviations from plan (if any):** plan.md's Data model section shows `Input` with no JSON struct tags. Added `json` tags (camelCase, matching `internal/contract`'s convention) to every field, with **no** `omitempty` anywhere. Rationale: spec requirement 13's Debug-level full struct dump will use `json.MarshalIndent(Input)` for readability instead of `%+v`, which needs tags to be useful. No field gets `omitempty` because applying `internal/contract`'s own cross-cutting rule correctly (omitempty is for "not applicable", never for "empty-but-meaningful") means all four fields stay present at their zero value -- same reasoning as `contract.RawInputTruncated` / `contract.Runtime.VersionSource`. plan.md's Data model code sample has been updated to match (tags + rationale added directly to plan.md, per Vedant's confirmation).
**New open questions:** none. `go build ./...`, `go vet ./internal/cli/...`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean (Vedant-confirmed). T001 marked done.

---

**Date:** 2026-08-09
**Task(s):** T002
**What happened:** Implemented `readTrace` in `read.go` (file-arg vs. stdin source selection, both-present precedence with `stdinIgnored`, TTY-no-input fast-fail, `io.LimitReader` bounded read at a provisional 1MB cap, empty/whitespace-only rejection, `contract.TruncateRawInput` for the final 512KB cap) plus table-driven tests in `read_test.go` covering all cases tasks.md's acceptance criteria lists.
**Deviations from plan (if any):** One extra wrapped error not named in plan.md's three examples: an I/O error reading from stdin itself (`"reading trace from stdin: %w"`), added for symmetry with the file-read error path; not currently covered by a test case since it's hard to trigger with a `strings.Reader`. Also: both `read.go` and `read_test.go` initially had a `// internal/cli/read.go`-style file-path comment directly above `package cli`, which failed `golangci-lint`'s `revive` `package-comments` check (it treats any comment immediately preceding `package X` as an attempted package doc, and fails it unless it's `"Package X ..."`). Fixed by removing those header comments (only `input.go` should carry the real package doc). Documented in `CONVENTIONS.md`'s Formatting & linting section and in Claude's own memory so this isn't repeated on later files/tasks.
**New open questions:** none. `go build ./...`, `go test ./internal/cli/... -run TestReadTrace -v`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean after the fix (Vedant-confirmed). T002 marked done.

---

**Date:** 2026-08-09
**Task(s):** T003
**What happened:** Implemented `validateFormat` and `validateLang` in `parse.go`, plus table-driven tests in `parse_test.go` covering valid and invalid values for each.
**Deviations from plan (if any):** None from plan.md's API contract. One explicit design call, consistent with (not a deviation from) `input.go`'s T001 doc comment: `validateFormat("")` returns `("markdown", nil)` rather than an error — empty is treated as omitted/defaulted, matching what `Input.Format`'s doc comment already committed to. `validateLang("")` returns `("", nil)` unchanged — empty stays the meaningful defer-to-auto-detection signal, never defaulted.
**New open questions:** none. `go build ./...`, `go test ./internal/cli/... -run 'TestValidateFormat|TestValidateLang' -v`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean (Vedant-confirmed). T003 marked done.

---

**Date:** 2026-08-09
**Task(s):** T004
**What happened:** Implemented `ParseAll` in `parse.go`: `flag.ContinueOnError` FlagSet registering `--lang`/`--format`, output discarded via `io.Discard` so `main.go`'s future `slog.Error` call is the single formatting path, fixed validation order (`flag.Parse` -> `validateLang` -> `validateFormat` -> `readTrace`), `stdinIgnored` logged at `Debug` via `slog.Debug`. Tests: `TestParseAll` (valid combos, invalid lang/format, no-input, combined-failure ordering, unknown-flag rejection) and `TestParseAll_StdinIgnoredLogging` (captures `slog` output via a temporary `slog.SetDefault` swap to confirm the Debug line fires exactly when `stdinIgnored=true`, and doesn't otherwise).
**Deviations from plan (if any):** None from plan.md's API contract or validation-order requirement. One flagged-and-accepted design call: `fs.SetOutput(io.Discard)` (needed to keep error formatting single-channel per spec requirement 12) means `-h`/`--help` now produces no usage listing, just a generic "help requested" error through `slog.Error`. `spec.md` doesn't mention `-h`/`--help` at all, so this is minimal-scope-per-what's-specified rather than an oversight — flagged to Vedant, no objection raised. Also undocumented in spec.md: extra positional args beyond the first are silently ignored (`fs.Arg(0)` semantics) — same minimal-scope reasoning, noted inline in the code comment.
**New open questions:** Should `-h`/`--help` get real usage-text support as a follow-up task? Not blocking, parked for whenever it comes up. `go build ./...`, `go test ./internal/cli/... -run TestParseAll -v`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean (Vedant-confirmed). T004 marked done.

---

**Date:** 2026-08-09
**Task(s):** T005
**What happened:** Implemented `ParseFixedLang` in `parse.go`, mirroring `ParseAll` but registering only `--format` (`--lang` never registered on this FlagSet, so passing it is automatically rejected via the flag package's own "not defined" error, satisfying spec requirement 2 with no special-case code). Tests: `TestParseFixedLang` (java/typescript, valid/invalid format, `--lang` rejection, no-input), `TestParseFixedLang_InvalidLangPanics`, `TestParseFixedLang_StdinIgnoredLogging`.
**Deviations from plan (if any):** One explicit design call, not in plan.md's API contract text but consistent with `CONVENTIONS.md`'s error-handling section: an invalid `lang` argument (anything other than `"java"`/`"typescript"`) causes `ParseFixedLang` to **panic**, not return an error. Rationale: `lang` is only ever supplied internally as a hardcoded literal by `cmd/java`/`cmd/typescript`, never from user input -- an invalid value here is a genuine programmer-error invariant ("panics are reserved for genuine programmer-error invariants, not runtime conditions" per CONVENTIONS.md), not a usage error. Also deliberately did not reuse `validateLang` for this check: it treats `""` as valid (defer-to-detection), which is the wrong semantics for a caller-fixed language. Minor formatting slip during editing (broken indentation in `parse_test.go` from a str_replace-style edit) caught and fixed before running the gates, not by `gofumpt` itself.
**New open questions:** none new (the `-h`/`--help` question from T004 still stands, same reasoning applies here). `go build ./...`, `go test ./internal/cli/... -run TestParseFixedLang -v`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean (Vedant-confirmed). T005 marked done.

---

**Date:** 2026-08-09
**Task(s):** T006
**What happened:** Implemented `LogLevel(verbosity int) slog.Level` in a new `internal/cli/log.go` (0 -> `Warn`, 1 -> `Info`, 2+ -> `Debug`, capped), a pure function per tasks.md's acceptance criteria wording -- it does not touch `slog`'s default logger; wiring the returned level into an actual handler is `main.go`'s job in T007/T008. Table-driven tests in `log_test.go`.
**Deviations from plan (if any):** `plan.md`'s File/module layout didn't list a home for this helper (it predates the idea of a dedicated file). Added `log.go`/`log_test.go` and updated `plan.md`'s layout section to match, same sync approach as T001's Data model update.
**New open questions:** none. `go build ./...`, `go test ./internal/cli/... -run TestLogLevel -v`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean (Vedant-confirmed). T006 marked done.

---

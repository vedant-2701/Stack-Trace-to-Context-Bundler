# Plan: CLI Input Handling

Derived from `spec.md`. Must be consistent with `memory/constitution.md`.

## Architecture / approach

Lives in `internal/cli` (per CONVENTIONS.md's existing file layout — this
package is shared with 002b, which will later add the
detect → parse → render → clipboard wiring alongside what this feature
builds). `main.go` in each of `cmd/all`, `cmd/java`, `cmd/typescript` stays
a thin wrapper: compute whether stdin is piped, call into `internal/cli`,
log the result via `slog`, `os.Exit` on error.

Two entry points into the package, sharing internal helpers for reading,
truncating, and format validation:

- `cli.ParseAll(args, stdin, stdinIsPiped)` — registers `--lang`,
  `--format`, `-v`, and `-vv`, used by `cmd/all`.
- `cli.ParseFixedLang(args, stdin, stdinIsPiped, lang)` — registers only
  `--format`, `-v`, and `-vv`, `LangHint` fixed to `lang`, used by
  `cmd/java` / `cmd/typescript`.

TTY detection (`stdinIsPiped`) is computed once in each `main.go` via a
tiny `os.Stdin.Stat().Mode()&os.ModeCharDevice` helper and passed in as a
plain `bool` — keeps the core parsing/reading logic pure and testable with
fake readers, no real terminal needed in tests (mirrors CONVENTIONS.md's
"behind a small interface with a hand-written fake" principle, applied to
TTY state rather than a subprocess).

`readTrace` itself never logs — it stays a pure function, consistent with
the testability goal above. When the file argument wins over piped stdin
(spec requirement 5), it signals this via a returned `stdinIgnored bool`
rather than calling `slog.Debug` directly; `ParseAll`/`ParseFixedLang` are
what actually log it, keeping every logging decision at the same layer as
validation and error wrapping, not split across the package.

Both entry points return `(Input, int, error)` — the `int` is a verbosity
count (0/1/2, see `LogLevel` in `log.go`) for `main.go` to pass straight
into `cli.LogLevel` and configure `slog`'s default handler. All
exit-code/`os.Exit` logic stays in `main.go` — the package itself never
calls `os.Exit` or panics for expected failure paths, per CONVENTIONS.md's
error-handling section.

Validation order is fixed and explicit: `ParseAll`/`ParseFixedLang` run
flag parsing and `validateLang`/`validateFormat` **before** calling
`readTrace`. Flag validation is cheap and does no I/O, so it fails fast
ahead of anything that opens a file or reads stdin. This matters because
some invocations trigger more than one failure condition at once (e.g.
`--lang=cobol` with no file arg and stdin not piped) — the flag error is
what the person sees in that case, deterministically, not whichever check
happened to run first in an unspecified order.

**Verbosity flags (`-v`/`-vv`) are registered directly on the same
`flag.FlagSet` already inside `ParseAll`/`ParseFixedLang`**, alongside
`--lang`/`--format`, rather than parsed in a separate pass in `main.go`.
This was a mid-implementation revision (pre-T007, see task T006b) after a
proposed alternative — a hand-rolled pre-pass over `argv` to strip
`-v`/`-vv` before handing the rest to `ParseAll` — turned out to have
real correctness gaps; see Alternatives considered below. Registering on
the single existing `FlagSet` inherits the stdlib `flag` package's
correct `--` terminator handling and single/double-dash equivalence for
free, and removes the ordering problem (`-v --lang=java trace.txt` vs.
`--lang=java -v trace.txt` both working identically) by construction,
since there's only one `Parse` call instead of two racing against each
other's assumptions about argument order.

## Stack & versions

No new dependencies. Stdlib only: `flag` (in `ContinueOnError` mode, per
spec requirement 12), `os`, `io`, `strings`, `log/slog`. Reuses
`internal/contract.TruncateRawInput` from 001.

## Data model

```go
// internal/cli/input.go
package cli

// Input holds everything read/parsed from the command line before any
// detection, parsing, or rendering happens (002b's job once it exists).
type Input struct {
    RawText           string `json:"rawText"`           // rune-safe, <=512KB (contract.TruncateRawInput)
    RawInputTruncated bool   `json:"rawInputTruncated"`
    LangHint          string `json:"langHint"`          // "", "java", or "typescript"
    Format            string `json:"format"`            // "json" or "markdown"
}
```

Tagged for JSON, no field `omitempty`: spec requirement 13's Debug-level
full struct dump uses `json.MarshalIndent(Input)` for readability instead
of `%+v`. Applying `internal/contract`'s own cross-cutting rule correctly
(`omitempty` for "not applicable", never for "empty-but-meaningful")
means every field here stays present at its zero value — none of the four
represent a "not applicable" case.

`LangHint == ""` is the explicit "defer to 003 auto-detection" signal —
never a zero-value stand-in for "unset" being confused with a real value,
consistent with constitution Article VI. This is why `LangHint` is not
`omitempty` either: omitting it from the JSON dump when empty would hide
the deliberate defer-to-auto-detect signal, making it indistinguishable
from a field that was never captured.

## File / module layout

```
cmd/
├── all/main.go          -- calls cli.ParseAll
├── java/main.go          -- calls cli.ParseFixedLang(..., "java")
└── typescript/main.go    -- calls cli.ParseFixedLang(..., "typescript")
internal/
└── cli/
    ├── input.go          -- Input struct
    ├── parse.go          -- ParseAll, ParseFixedLang, flag registration/validation
    ├── read.go            -- source selection (file vs stdin), bounded read, truncation call
    ├── log.go             -- LogLevel: verbosity count -> slog.Level, shared by all three main.go
    ├── parse_test.go
    ├── read_test.go
    └── log_test.go
```

## API / contracts

```go
func ParseAll(args []string, stdin io.Reader, stdinIsPiped bool) (Input, int, error)
func ParseFixedLang(args []string, stdin io.Reader, stdinIsPiped bool, lang string) (Input, int, error)
```

The `int` is a verbosity count for `main.go` to pass into `cli.LogLevel`
(`log.go`, T006): `0` if neither `-v` nor `-vv` was passed, `1` for `-v`,
`2` for `-vv` (or both together — max, not sum, since nothing beyond
`Debug` is defined). Always `0` on any error return: `-v`/`-vv` only gate
the success-path Info summary / Debug dump (spec requirement 13), and
specifically for a `flag.Parse` failure the verbosity flags may not have
been reached yet depending on argument order, so a real value there would
be unreliable rather than just unused — `0` is a flat, unambiguous rule
with nothing built on top of it to get inconsistent.

Internal (unexported) helpers:

```go
func readTrace(fileArg string, stdin io.Reader, stdinIsPiped bool) (raw string, truncated bool, stdinIgnored bool, err error)
func validateFormat(v string) (string, error)
func validateLang(v string) (string, error) // ParseAll only
func verbosityFromFlags(v, vv bool) int // max(1 if v, 2 if vv, else 0)
```

Every returned `error` is wrapped with context per CONVENTIONS.md
(`fmt.Errorf("...: %w", err)`) and is specific enough to satisfy spec
requirement 11 when logged — e.g. distinct wrapped errors for
"no input: stdin not piped and no file argument given",
"reading trace file %s: %w", "input is empty after reading", each
distinguishable in the logged message text.

## Testing strategy

Table-driven unit tests in `internal/cli`, no real files or real TTY
needed:

- `read_test.go`: fake `io.Reader`s (`strings.Reader`/`bytes.Reader`) and
  temp files (`t.TempDir()`) cover: file-arg-only, stdin-only, both
  present (file wins, `stdinIgnored=true`), neither present + `stdinIsPiped=false`
  (immediate error, no blocking), file-not-found, empty/whitespace-only
  content from both sources, input over 512KB (bounded read +
  `RawInputTruncated=true`).
- `parse_test.go`: covers `--lang`/`--format` valid values, invalid
  values, `--lang` on `ParseFixedLang` paths (should never be registered,
  so passing it must produce flag package's "not defined" error, still
  routed through `slog.Error`), default `--format=markdown`, flag
  validation running before `readTrace` (assert via a case with both an
  invalid flag and no available input — the flag error must be what's
  returned), and the `Debug`-level "stdin ignored" log firing exactly when
  `readTrace` returns `stdinIgnored=true`.
- Log output assertions: redirect `slog`'s output to a buffer in tests to
  assert Info-summary/Debug-dump content and level gating. `-v`/`-vv`
  parsing and the verbosity-count return value are tested directly in
  `parse_test.go` (T006b); the pure count->`slog.Level` mapping is
  `LogLevel` in `log.go` (T006, already tested in `log_test.go`).
  Actually configuring `slog`'s default handler with that level, and the
  Info-summary/Debug-dump content itself, is `main.go`'s job (T007/T008)
  and tested at that level via a real subprocess run, per those tasks'
  acceptance criteria.
- No table-driven requirement here is mandated by CONVENTIONS.md (that's
  specifically called out for `internal/parser/*`), but used anyway since
  the error-path enumeration maps naturally to table entries.

## Risks & open decisions

- **1MB `LimitReader` cap is provisional**, same spirit as ADR 0001's
  "provisional 30s" timeout — if real-world traces routinely approach it
  before the 512KB truncation point, revisit the constant. Not a blocker
  for this feature.
- **TTY detection cross-platform reliability**: `os.ModeCharDevice` is
  stdlib and works on Linux/macOS; Windows behavior should be verified
  before 010 (distribution) ships a Windows target — flagging now so it
  isn't forgotten, same pattern as ADR 0001's non-Linux benchmark gap.
- **`internal/cli` is shared with 002b**: this feature only populates the
  arg-parsing/input-reading half of that package. Don't let 002b's
  wiring code get implemented inside 002a's task list by accident — scope
  boundary is enforced by spec.md's "Out of scope" section.

## Alternatives considered

- `--lang` available on all three binaries — rejected, redundant since
  the binary already implies the language (spec decision, cmd/java/cmd/
  typescript reject it as an usage error instead).
- Blocking on stdin with an interactive prompt when nothing is piped —
  rejected, violates constitution Article I (no interactive prompts
  required for core operation).
- Letting the stdlib `flag` package's default `ExitOnError` handle flag
  errors natively — rejected, produces inconsistent message formatting
  vs. the input-reading error paths, which is exactly the debuggability
  problem flagged during spec review. `ContinueOnError` + `slog.Error`
  keeps every usage-error path uniform.
- Unbounded read then truncate — rejected, memory-exhaustion risk on a
  large accidental input before truncation ever runs.
- A hand-rolled pre-pass over `argv` (`ExtractVerbosity`) to strip
  `-v`/`-vv` before handing the remaining args to `ParseAll`/
  `ParseFixedLang`'s own `FlagSet` — rejected after review found three
  concrete gaps: (1) a blind token scan would strip a `-v` appearing
  after a `--` terminator, even though `--` means "everything after this
  is positional," e.g. a trace file literally named `-v`; (2) it would
  only recognize the exact strings `-v`/`-vv`, not `--v`/`--vv`, an
  inconsistent dash-count convention vs. `--lang`/`--format`, which the
  stdlib `flag` package treats as equivalent; (3) ambiguous semantics for
  `-v -v` repeated (spec.md's "`-v`/`-vv` respectively" reads as two
  distinct flags, not a repeatable counter, but the pre-pass would have
  had to pick an interpretation silently). Registering `-v`/`-vv`
  directly on the existing `FlagSet` avoids all three by construction —
  see Architecture / approach above.

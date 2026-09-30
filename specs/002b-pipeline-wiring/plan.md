# Plan: Pipeline wiring

Derived from `spec.md`. Consistent with `memory/constitution.md` and
`CONVENTIONS.md`. References below to Q1-Q18 are `qa-log.md` entries; FR
numbers are `spec.md`'s Functional requirements.

## Architecture / approach

002a's three `main.go` files currently do their own flag parsing, error
handling, and stub logging. 002b centralizes everything past "read raw
input" into `internal/cli`, so each `main.go` becomes a few lines that
declare what that binary supports and call one shared entry point.

Each binary supplies a `cli.BinaryConfig`:

```go
// internal/cli/run.go
type DependencyResolver func(ctx context.Context, workDir string, chain []contract.ExceptionNode) *contract.Dependencies

type BinaryConfig struct {
	ProgramName string                              // filepath.Base(os.Args[0]), computed in main.go (Q14/FR20's synopsis line -- see "Program name" below)
	FixedLang   string                              // "" for cmd/all (registers --lang); else "java"/"typescript"
	Candidates  []parser.LanguageParser              // parsers this binary knows about
	Resolvers   map[contract.Language]DependencyResolver
	Examples    []Example                            // for -h/--help (Q14)
}

func Run(args []string, stdin io.Reader, stdinIsPiped bool, cfg BinaryConfig, stdout, stderr io.Writer) int
```

`Run` does, in order:

1. Parse flags and read input via `ParseAll` (FixedLang == "") or
   `ParseFixedLang` (FixedLang != ""), passing `cfg.Candidates` through
   (see "Candidate filtering" below -- this is where FR9/FR10's
   pre-input-read checks live, inside the existing parse functions, not
   in `Run` itself, because `Run` only gets `Input` back after
   `readTrace` has already run).
2. On `pflag.ErrHelp` (`errors.Is`): print `usageText(cfg)` to stderr,
   return 0 (FR19-20). On any other error: `slog.Error(err.Error())`,
   return 2 -- unchanged from 002a's existing behavior, which already
   covers flag errors, empty/no input, an unsupported `--lang`/`--format`
   value, and (new) the FR10 unsupported-language message and the FR17
   `--output` usage errors, because all of these now originate inside the
   same `ParseAll`/`ParseFixedLang` call.
3. Configure `slog` from verbosity (unchanged from 002a). Log
   `stdin ignored` if set (unchanged). Do NOT log `parsed input` or dump
   `Input` (FR22 -- 002a's stub is removed here).
4. Re-run candidate filtering (cheap, pure, no I/O -- see below) to get
   the actual candidate list, then `parser.DetectLanguage(in.RawText,
   filtered)`. Map `ErrNoMatch`/`ErrAmbiguous` to exit 4 with the FR11/12
   messages; log at Info: `language resolved` (attribute `language`) --
   always -- and, additionally, when the hint doesn't match the detected
   language, a second, separate Info line using FR9's exact wording
   (`detected language differs from --lang hint`, attributes `hint` and
   `detected`). Two distinct log calls in the mismatch case, not one
   line with a note folded in.
5. Call the resolved parser's `Parse(ctx, in.RawText)`. Map
   `ErrUnparseable` to exit 3 (FR12), anything else to exit 1.
6. `workDir, err := os.Getwd()`; `err != nil` -> exit 1 (FR5).
7. `gitMeta := codecontext.BuildGitMetadata(ctx, workDir)`.
8. `codeContexts := codecontext.BuildCodeContexts(ctx, chain, language,
   gitMeta)`.
9. `deps := cfg.Resolvers[language]` if present, called as
   `deps(ctx, workDir, chain)`; nil otherwise (FR4).
10. Assemble `contract.Bundle` (FR4).
11. Render via `render.Markdown`/`render.JSON` (FR6).
12. Deliver: stdout or `--output` file first (with the FR15 trailing
    newline), exit 1 on failure with nothing further attempted (FR16);
    then clipboard unless `--no-clipboard`, Warn+exit 0 on failure
    (FR14/16). Log `bundle delivered` at Info (FR22).

`ctx` is `context.Background()` throughout (FR8, Q12) -- no signal
handling, no deadline.

### Candidate filtering (FR3, FR9, FR10)

A single pure helper, independent of any binary:

```go
// internal/cli/candidates.go
var hintLanguages = map[string][]contract.Language{
	"typescript": {contract.LanguageJavaScript, contract.LanguageTypeScript},
	"java":       {contract.LanguageJava},
}
var hintDisplayName = map[string]string{"java": "Java", "typescript": "TypeScript"}

func selectCandidates(hint string, registered []parser.LanguageParser) ([]parser.LanguageParser, error)
```

`hint == ""` returns `registered` unfiltered. A non-empty hint returns
only the registered parsers whose `Language()` is in `hintLanguages[hint]`;
an empty result is a `*languageUnsupportedError` (see below), never
`ErrNoMatch` -- that sentinel is `DetectLanguage`'s, reserved for "the
trace's content didn't match," not "this binary doesn't have that
parser at all."

`selectCandidates` is called twice with identical inputs: once inside
`ParseAll`/`ParseFixedLang` (fail-fast, before `readTrace`, per FR10 --
this is why `Candidates` is now a parameter of those two functions,
alongside the existing `args`/`stdin`/`stdinIsPiped`), and once inside
`Run` after input is read (to build the actual `DetectLanguage` call).
Recomputing it is a few slice comparisons with no I/O -- cheaper and
simpler than adding a fourth return value or a new struct just to carry
the filtered slice one call forward (Article VIII).

```go
// internal/cli/errors.go
type languageUnsupportedError struct{ displayName string }
func (e *languageUnsupportedError) Error() string { return e.displayName + " is not supported yet" }
```

A concrete type (not a wrapped sentinel) because the exact user-facing
text (FR10: "no feature IDs or internal names") IS `Error()`'s return
value -- nothing gets appended when it's logged. `Run` still uses
`errors.As` to detect it in `Run`'s existing default-exit-2 path; it
doesn't need special exit-code handling beyond that, since 002a's
existing "any `ParseAll`/`ParseFixedLang` error -> exit 2" behavior
already covers it.

### Order inside `ParseAll`/`ParseFixedLang` (extends parse.go's existing
documented order)

`fs.Parse` -> `validateLang` -> `validateFormat` -> candidate
fail-fast (`selectCandidates`) -> `validateOutput` -> `readTrace`.
Candidate fail-fast before `validateOutput` so `cmd/java --format=bogus`
reports the format error (acceptance criterion), and `cmd/java -o
/no/such/dir` reports "Java is not supported yet" rather than the output
error -- both are exit 2 either way, but the order determines which
single message a multi-problem invocation gets, and this keeps it
deterministic (matches the existing comment's own reasoning for why
`validateLang` precedes `validateFormat`).

```go
// internal/cli/read.go (new function, same file)
func validateOutput(outputPath, fileArg string) error
```
Empty `outputPath` is a no-op. Otherwise: `--output`'s parent directory
must exist and be a directory (usage error); if `fileArg != ""`, the
resolved `outputPath` must not be the same file as `fileArg`, checked
via `os.SameFile` -- but only when `os.Stat` succeeds on *both* paths
(FR17, Q4). `outputPath` not existing yet is the ordinary case (creating
a new output file) and is never itself a conflict; a `fileArg` that
doesn't exist is left for `readTrace`'s own "file not found" handling
afterward, not decided here. Not applicable when input is stdin-only.

### Delivery (FR13-17)

```go
// internal/cli/output.go
func deliver(bundle string, outputPath string, stdout io.Writer) error
```
Writes `bundle` (plus one `\n` if it doesn't already end with one -- FR15)
to `stdout` when `outputPath == ""`, else opens `outputPath`
(`os.O_WRONLY|os.O_CREATE|os.O_TRUNC`, silently overwriting -- FR17) and
writes there. Called only after rendering succeeds. The clipboard write
(`clipboard.Write`, unchanged from 009) happens only after `deliver`
returns nil, and only when `!in.NoClipboard`; its error is Warn-logged
(FR16's exact text) and does not affect the exit code.

### Help text (FR19-20)

```go
// internal/cli/help.go
type Example struct{ Command, Description string }

var exitCodeDocs = []struct{ Code int; Meaning string }{
	{0, "success (including --help)"},
	{1, "unexpected/internal error"},
	{2, "usage error (bad flags, no input, unsupported language, bad --output path)"},
	{3, "input could not be parsed"},
	{4, "source language ambiguous or undetected"},
}

func usageText(programName string, fs *pflag.FlagSet, examples []Example) string
```
`exitCodeDocs` is the one Go-data source for the exit-code table (Q14);
`CONVENTIONS.md`'s prose table is updated to match by hand as part of
this feature's completion (spec.md's last acceptance criterion), since
it's documentation, not code, and nothing currently generates one from
the other. `fs` is built (flags registered, not yet parsed) once more
inside `usageText` purely for `FlagUsages()` -- reusing the exact same
registration code as `ParseAll`/`ParseFixedLang` (extracted into a small
shared `registerFlags(fs *pflag.FlagSet, fixedLang string)` helper) so
the listing can never drift from what's actually accepted.

Program name: `filepath.Base(os.Args[0])`, computed in each `main.go`
and passed into `BinaryConfig` (not read from inside `internal/cli`, to
keep that package free of a direct `os.Args` reference and consistent
with the existing pattern of `main.go` doing `os.Stdin`/`os.Args`
plumbing and `internal/cli` staying pure/injectable).

## Stack & versions

No new third-party dependency. `pflag` (already adopted, ADR 0002) is
the only external package touched, via its existing `ErrHelp` sentinel
and `FlagUsages()` -- both already part of the version in `go.mod`.

## Data model

`internal/cli.Input` gains two fields (both always present, neither
`omitempty` at the Go level -- they're plain struct fields, not part of
`contract`):

```go
Output      string // "" means "no --output given" -- this one IS the not-applicable case, since contract's omitempty convention doesn't apply here (Input is never serialized to the bundle)
NoClipboard bool
```

No change to `internal/contract`: `Bundle.OS` is populated as
`contract.OS(runtime.GOOS)` verbatim for every `GOOS` value, including
ones outside the three declared constants (Q11) -- confirmed during
planning (see "Risks" below) that nothing validates `OS` against them.
`Bundle.Dependencies` is populated from `cfg.Resolvers[language]`;
`javascript` and `typescript` both map to
`typescript.ResolveDependencies` (006b). No schema bump.

## File / module layout

```
internal/cli/
├── run.go           NEW -- BinaryConfig, Run (orchestration, FR1-2,4-8,21-22)
├── candidates.go    NEW -- selectCandidates, hintLanguages, hintDisplayName (FR3,9,10)
├── errors.go        NEW -- languageUnsupportedError
├── output.go         NEW -- deliver (FR13,15,17), clipboard call site (FR14,16)
├── help.go          NEW -- Example, exitCodeDocs, usageText, registerFlags (FR19-20)
├── parse.go         CHANGED -- ParseAll/ParseFixedLang gain a candidates
│                    parameter and the new validateOutput/candidate-check
│                    steps; flag registration extracted into registerFlags
├── input.go         CHANGED -- Input gains Output, NoClipboard
├── read.go          CHANGED -- adds validateOutput
├── log.go           UNCHANGED
cmd/all/main.go        CHANGED -- becomes: build BinaryConfig{Candidates:
                       [js, ts], Resolvers: {js: tsResolver, ts: tsResolver}},
                       call cli.Run(..., os.Stdout, os.Stderr), os.Exit
cmd/java/main.go       CHANGED -- BinaryConfig{FixedLang: "java",
                       Candidates: nil, Resolvers: nil}
cmd/typescript/main.go CHANGED -- BinaryConfig{FixedLang: "typescript",
                       Candidates: [js, ts], Resolvers: {js: tsResolver, ts: tsResolver}}
```

`internal/contract`, `internal/parser/*`, `internal/codecontext`,
`internal/dependency/*`, `internal/render/*`, `internal/clipboard`: no
changes (spec.md Out of scope).

## API / contracts

Key new/changed signatures, restated from above for reference:

```go
func ParseAll(args []string, stdin io.Reader, stdinIsPiped bool, candidates []parser.LanguageParser) (Input, int, error)
func ParseFixedLang(args []string, stdin io.Reader, stdinIsPiped bool, lang string, candidates []parser.LanguageParser) (Input, int, error)
func Run(args []string, stdin io.Reader, stdinIsPiped bool, cfg BinaryConfig, stdout, stderr io.Writer) int
```

`Run` takes `stdout`/`stderr` as explicit `io.Writer` parameters
(revised during technical design review -- the original design wrote
directly to real `os.Stdout`/`os.Stderr` with no way for `run_test.go`
to observe `Run`'s output short of swapping the real, process-wide
`os.Stdout`/`os.Stderr` variables, which the Testing strategy below
never actually called for and which isn't parallel-test-safe). `slog`'s
handler is configured inside `Run` against the injected `stderr`, not
`os.Stderr` directly; `deliver` (see Delivery above) receives the
injected `stdout` unchanged. Each `main.go` passes real `os.Stdout`/
`os.Stderr` when calling `Run`; `run_test.go` passes a `*strings.Builder`
(or similar) for both, so its assertions on rendered bundle content,
usage text, and log line content read directly from those buffers
rather than needing any real-I/O capture trick.

## Testing strategy

- `internal/cli/candidates_test.go`: table-driven over
  (hint, registered parsers) -> (filtered list | unsupported error),
  using two tiny fake `parser.LanguageParser` implementations (already
  the pattern 003b's own tests use) -- no real JS/TS parser needed.
- `internal/cli/parse_test.go` (extended): every existing case gains the
  new `candidates` parameter; new cases for the FR10 fail-fast (exit-2
  error, `readTrace` never reached -- assert via a stdin fake that
  records whether it was read), FR17's two `--output` usage errors, and
  the FR-in-plan ordering (`--format=bogus` wins over an unsupported
  `--lang`).
- `internal/cli/output_test.go`: `deliver` against a real temp file and
  a `strings.Builder` stdout stand-in; trailing-newline idempotency;
  overwrite-silently; write-failure path (a temp dir removed after
  `Stat` succeeds, forcing the actual write to fail) -> error returned,
  not swallowed.
- `internal/cli/help_test.go`: for each real `BinaryConfig` (built the
  same way each `main.go` builds one), every `exitCodeDocs` entry's code
  appears in `usageText`'s output, and every `Example.Command`'s flags
  parse without a flag error through the matching
  `ParseAll`/`ParseFixedLang` (using a fixed fake file argument
  substituted for any placeholder path in the example) -- this is the
  anti-drift test from Q14.
- `internal/cli/run_test.go`: the exit-code mapping (FR21) exercised
  with fake `parser.LanguageParser`s that return `ErrNoMatch`/
  `ErrAmbiguous`/`ErrUnparseable`/an arbitrary error/success, and a fake
  clipboard writer (matching 009's own `cmdRunner` fake pattern) for the
  order-of-delivery and clipboard-failure-is-non-fatal cases (FR15-16).
  `os.Getwd` is called through a package-level `var getwd = os.Getwd`
  variable so a test can override it (same pattern as `009`'s injectable
  clipboard runner), rather than a new interface. `Run`'s injected
  `stdout`/`stderr` (`*strings.Builder` in tests, see API/contracts
  above) let these same test cases assert directly on rendered bundle
  content, usage-text content, and log-line content (`stdin ignored`,
  `language resolved`, `bundle delivered`), rather than needing a
  separate real-I/O-capture mechanism or being limited to testing
  `deliver`/`usageText` in isolation.
- No test calls real `git`, `node`, or a real clipboard utility
  (unchanged from every existing feature -- `BuildGitMetadata`,
  `ResolveDependencies`, and `clipboard.Write`'s own runners are already
  fake-tested by 004/006b/009; 002b's tests fake at the boundary of
  those packages, e.g. a `parser.LanguageParser` fake, not a git fake).
- `cmd/*/main.go` are not unit-tested beyond `go build` succeeding
  (unchanged from 002a -- they're thin wiring).

## Risks & open decisions

- **`ParseAll`/`ParseFixedLang` signature change.** Both gain a
  `candidates` parameter. Every existing call site in `parse_test.go`
  needs updating. This is the one place 002b reaches back into 002a's
  code rather than only adding new files -- necessary because FR10's
  fail-fast has to happen before `readTrace`, which only these two
  functions control.
- **`Bundle.OS` verified, not assumed** (carried over from Q11): task 1
  greps `internal/render` and `internal/contract` for any comparison
  against `OSLinux`/`OSDarwin`/`OSWindows` before FR4 is implemented. If
  one exists, this plan's Q11 answer needs revisiting before proceeding
  -- flagged here so it's not discovered mid-implementation.
- **Existing flag help strings** (carried over from Q17): task 1 also
  confirms `--lang`, `--format`, `-v` already have non-empty help
  strings in `parse.go` (they do, per the code read during
  interrogation) -- only `-o`/`--output` and `--no-clipboard` need new
  ones.
- **`CONVENTIONS.md` drift risk stays partly manual.** `exitCodeDocs` is
  the Go-side source of truth and is tested against the help text, but
  `CONVENTIONS.md`'s own exit-code line is prose that a human/agent must
  remember to update (spec.md's last acceptance criterion) -- no
  automated check ties the two together. Accepted per Q14's discussion;
  flagged again here rather than silently assumed fixed.

## Alternatives considered

- **A `pipeline` package separate from `cli`.** Rejected: `CONVENTIONS.md`
  already assigns "wiring detect -> parse -> render -> clipboard" to
  `internal/cli`, and splitting it out would just add an import hop with
  no reuse benefit (nothing outside the three `main.go` files calls this
  wiring).
- **Passing the filtered candidate list through `Input` instead of
  recomputing it.** Rejected in favor of recomputing (see "Candidate
  filtering" above) -- avoids growing `Input`'s or `ParseAll`'s already
  multi-value return shape for a computation cheap enough to just repeat.
- **A `RunPipeline` that returns `(contract.Bundle, error)` and a
  separate `main.go`-level exit-code switch.** Considered, to keep
  exit-code mapping out of `internal/cli`. Rejected: it would duplicate
  the same switch three times (once per `main.go`) or require yet
  another shared helper -- `Run` returning the exit code directly is one
  fewer layer for the same result, and 002a's `main.go` files already
  call `os.Exit` directly today.

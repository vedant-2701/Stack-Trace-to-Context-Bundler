# Plan: Language auto-detection registry

Derived from `spec.md`. Must be consistent with `memory/constitution.md`.

## Architecture / approach

A single new file, `internal/parser/detect.go`, adds `DetectLanguage` to the
existing `internal/parser` package (alongside `registry.go`'s
`LanguageParser` interface and `errors.go`'s sentinels). No new package, no
registration side effects, no stateful type -- `DetectLanguage` is a pure
function over an explicit `[]LanguageParser` the caller already assembled
(spec.md FR9's resolution of 003a's reserved scope in favor of the simpler,
explicit-parameter design over a global/package-level registry).

## Stack & versions

Stdlib only: `errors`, `fmt`, `strings`. No new third-party dependency.

## Data model

No new types. Consumes `LanguageParser` (003a) and, indirectly via
`Language()`'s return type, `contract.Language` (001) -- `contract.Language`
is a plain `string`-based type, so formatting it into an error message needs
no import of `internal/contract` in `detect.go` itself; `LanguageParser`'s
own method signature (declared in `registry.go`, same package) already
carries that type information.

## File / module layout

```
internal/parser/
  registry.go   # unchanged -- LanguageParser interface (003a)
  errors.go     # + ErrNoMatch, ErrAmbiguous (alongside existing ErrUnparseable)
  detect.go     # new -- DetectLanguage
  detect_test.go  # new -- package parser_test (external), see Testing strategy
```

## API / contracts

```go
package parser

import "errors"

var ErrUnparseable = errors.New("...") // unchanged, already exists

// ErrNoMatch is wrapped by DetectLanguage when no candidate LanguageParser's
// Detect() returned true for the given raw trace. Distinguishes "the trace
// doesn't look like any registered language" from ErrAmbiguous ("it looks
// like more than one") -- callers (002b) map both to CLI exit code 4
// (CONVENTIONS.md) but log a different, specific message for each via
// errors.Is. The wrapping error's message names every checked candidate's
// Language() value (comma-joined, prefixed "checked "), mirroring
// ErrAmbiguous's message shape below -- so a caller logging this error
// doesn't need to re-derive which parsers were even in play.
var ErrNoMatch = errors.New("no registered parser matched this trace")

// ErrAmbiguous is wrapped by DetectLanguage when two or more candidate
// LanguageParsers' Detect() both returned true for the given raw trace.
// The wrapping error's message names every matched candidate's Language()
// value, so a caller doesn't need to re-derive which languages collided.
var ErrAmbiguous = errors.New("trace matched more than one registered language")
```

```go
package parser

import (
	"fmt"
	"strings"
)

// DetectLanguage calls Detect(rawTrace) on every element of candidates, in
// order, and returns the single LanguageParser that claims it.
//
// candidates must be non-empty -- an empty slice means the calling binary
// registered zero parsers, which is a build-time wiring bug (e.g. cmd/all's
// main.go forgot to list any), never something real trace input can
// trigger. DetectLanguage panics in that case rather than returning
// ErrNoMatch, per CONVENTIONS.md's guidance that panics are for genuine
// programmer-error invariants, not runtime conditions.
//
// Returns:
//   - exactly one candidate matched: that LanguageParser, nil error.
//   - zero candidates matched: nil, an error wrapping ErrNoMatch, naming
//     every checked candidate's Language() value.
//   - two or more candidates matched: nil, an error wrapping ErrAmbiguous,
//     naming every matched candidate's Language() value.
//
// DetectLanguage performs no I/O and takes no context.Context: every
// Detect() call it makes is already required (003a's LanguageParser
// interface) to be fast, in-memory, and side-effect-free, so there is
// nothing here that could block or need cancellation.
func DetectLanguage(rawTrace string, candidates []LanguageParser) (LanguageParser, error) {
	if len(candidates) == 0 {
		panic("parser.DetectLanguage: candidates is empty -- caller wired zero parsers")
	}

	var matched []LanguageParser
	for _, c := range candidates {
		if c.Detect(rawTrace) {
			matched = append(matched, c)
		}
	}

	switch len(matched) {
	case 0:
		names := make([]string, len(candidates))
		for i, c := range candidates {
			names[i] = string(c.Language())
		}
		return nil, fmt.Errorf("checked %s: %w", strings.Join(names, ", "), ErrNoMatch)
	case 1:
		return matched[0], nil
	default:
		names := make([]string, len(matched))
		for i, m := range matched {
			names[i] = string(m.Language())
		}
		return nil, fmt.Errorf("%s: %w", strings.Join(names, ", "), ErrAmbiguous)
	}
}
```

## Testing strategy

`detect_test.go` uses `package parser_test` (external test package), not
`package parser` -- deliberately, so it can import
`internal/parser/typescript` for the real-fixture cases below without
creating an import cycle (`internal/parser/typescript` already imports
`internal/parser` for the `LanguageParser` interface).

- **Single real match**: `typescript.NewJavaScriptParser()` and
  `typescript.NewTypeScriptParser()` as candidates, against a real
  `.ts`-frame trace (`testdata/ts-native-execution.txt` content, copied
  inline as a string literal -- not a shared testdata file across package
  boundaries) -- asserts `typescriptParser` is returned.
- **Real no-match**: same two real candidates, against the real
  `bare-stack-fetch-cause.txt` content (`"TypeError: fetch failed"`,
  inlined as a string literal with a comment noting it mirrors
  `internal/parser/typescript/testdata/bare-stack-fetch-cause.txt`) --
  asserts `errors.Is(err, parser.ErrNoMatch)` and that the error message
  names both checked candidates (`javascript`, `typescript`).
- **Ambiguous**: two hand-written fake `LanguageParser` implementations
  declared locally in `detect_test.go` (not a shared/exported fake --
  CONVENTIONS.md's "hand-written fakes, no mocking framework" pattern),
  both `Detect()` hardcoded to return `true`, different `Language()`
  values -- asserts `errors.Is(err, parser.ErrAmbiguous)` and that the
  error message contains both fake languages' names.
- **Empty candidates**: asserts `DetectLanguage(rawTrace, nil)` panics,
  via `recover()`.

No fake is needed for the "single real match" and "real no-match" cases --
006a's already-exported, already-tested `NewJavaScriptParser`/
`NewTypeScriptParser` constructors are real production code, not test
doubles.

## Risks & open decisions

- The ambiguous branch is untested against any real cross-language
  collision -- see `spec.md`'s Out of scope and the corresponding
  `memory/known-gaps.md` row (owned by 005a). The fake-based test proves
  the branching logic is correct; it can't prove two real heuristics never
  collide.
- `002a`'s `--lang` flag values (`java`/`typescript`) predate 006a's
  two-parser split and don't cleanly map to `{javascriptParser,
  typescriptParser}` vs. a single parser. Not this feature's problem to
  solve (see spec.md's Out of scope), but 002b's spec interrogation should
  not silently paper over it.

## Alternatives considered

- **A global/package-level registry with a `Register()` function** (the
  `database/sql` driver pattern). Rejected: adds init-time side effects and
  hidden state for no real benefit here, since every binary (`cmd/all`,
  `cmd/java`, `cmd/typescript`) already knows exactly which parsers it
  wants to register at `main()` time -- an explicit slice literal is just
  as easy to write and far easier to test (no global state to reset between
  test cases). Confirmed with Vedant during interrogation.
- **Defaulting to one language when detection is ambiguous** instead of
  returning an error. Rejected outright: directly conflicts with
  `memory/constitution.md` Article VI ("never present a guess as fact") --
  silently picking one of two-or-more real matches is exactly the kind of
  unverified guess the constitution rules out, regardless of how rare the
  ambiguous case is in practice today.
- **A richer `Detect()` return type** (e.g. a confidence score) to help
  resolve ambiguity by ranking candidates. Already rejected at the 003a
  interface level (see 003a's own Alternatives considered) for the same
  reason: no tie-breaking use case exists to justify it, and 003a's
  `Detect() bool` shape is fixed input to this feature, not something this
  feature reopens.
- **A single combined `ErrAmbiguousOrNoMatch` sentinel** instead of two
  separate ones. Rejected: `002a`'s own established convention (FR11) is
  that every usage-error/failure path logs a message that "specifically
  identifies which condition triggered it" -- collapsing two genuinely
  different conditions into one sentinel would make that harder for 002b
  to honor, for no simplification benefit (two `var ... = errors.New(...)`
  lines is not meaningfully more code than one).

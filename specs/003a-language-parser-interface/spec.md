# Spec: Language parser interface

**Status:** Approved
**Folder:** specs/003a-language-parser-interface

## Overview

Defines `LanguageParser`, the shared Go interface that every language-specific
parser (005a Java, 006a TS/JS) must implement, plus the one sentinel error the
interface's error-handling contract depends on. This feature produces no
parsing logic itself -- it pins down the *shape* both parsers must conform to
(method signatures, doc-comment contract for frame ordering, bucket
completeness, and error semantics) so 005a and 006a can be implemented and
tested independently, and so 003b's auto-detection registry has a single,
stable interface to iterate over.

## User stories

- As the developer implementing 005a (Java parser), I want a precisely
  specified `LanguageParser` interface so I know exactly what shape of data
  to produce, without guessing at chain/runtime/error semantics that 001 or
  004 left to "the parser's own scope."
- As the developer implementing 006a (TS/JS parser), I want the same
  interface so both parsers are structurally interchangeable from any
  caller's perspective.
- As the developer implementing 003b (auto-detection registry), I want
  `Detect()` guaranteed fast and side-effect-free so I can call it once per
  registered parser per invocation without a performance or reliability
  concern.
- As a maintainer, I want `Parse()`'s frame-ordering and bucket-completeness
  guarantees pinned in a doc comment, not left as an implicit assumption --
  `contract.ComputeFingerprint` already depends on `Frames[0]` being the
  originating frame, and nothing currently enforces that on any future
  parser.

## Functional requirements

1. `internal/parser` defines a `LanguageParser` interface in `registry.go`
   with exactly three methods:
   ```go
   type LanguageParser interface {
       Language() contract.Language
       Detect(rawTrace string) bool
       Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error)
   }
   ```
2. `Parse()`'s doc comment must state that frames within each returned
   `ExceptionNode` are ordered exactly as they appear in the raw trace, with
   `Frames[0]` being the frame where that node's exception was thrown.
   `contract.ComputeFingerprint` assumes this ordering; it must be a binding
   part of the interface, not an implicit convention.
3. `Parse()`'s doc comment must state that every `Frame.Bucket` in a returned
   chain is fully assigned before return. Neither `ExceptionNode` nor `Frame`
   has a field to represent a partial/degraded bucketing state, so this is a
   hard requirement, not a style preference.
4. `Parse()` must return either a complete, valid chain and runtime with a
   nil error, or a nil chain and zero-value runtime with a non-nil error --
   never a partial result.
5. `internal/parser/errors.go` declares a sentinel:
   ```go
   var ErrUnparseable = errors.New("...")
   ```
   `Parse()` implementations must wrap this sentinel (`fmt.Errorf("...: %w",
   ErrUnparseable)`) when the raw trace matched this language's general
   shape (i.e. `Detect()` would return `true` for it) but could not actually
   be converted into a valid chain. Any other non-nil error from `Parse()`
   is an unexpected internal error, not an expected parse failure --
   callers distinguish the two via `errors.Is(err, parser.ErrUnparseable)`.
6. `Parse()`'s first parameter is `ctx context.Context`, to allow caller
   cancellation. The interface does not require or promise that `ctx`
   carries a caller-set deadline. An implementation that shells out to a
   subprocess internally (e.g. inferring `Runtime.Version` via
   `contract.VersionSourceLocalEnvironment`) is responsible for imposing its
   own bounded timeout derived from the passed-in `ctx` -- matching the
   existing pattern in `internal/codecontext/runner.go`'s `gitTimeout`.
7. `Detect(rawTrace string) bool` takes no `ctx` and must not perform any
   I/O, subprocess call, or other blocking operation -- it is a fast,
   in-memory check on the raw trace text only. A future auto-detection
   registry (003b) may invoke `Detect()` once per registered parser on every
   invocation where the language isn't hinted.
8. Both `Detect()` and `Parse()` may assume `rawTrace` is non-empty and
   already bounded, per `internal/cli`'s existing 512KB cap (via
   `contract.TruncateRawInput`) applied before either method is ever called.
   Implementations are not required to defensively handle an empty string.
9. `registry.go`, as built by this feature, contains only the
   `LanguageParser` interface declaration and its doc comments -- no
   registration map, `Register()` function, or detection-orchestration
   logic. That is 003b's scope.
10. No package under `internal/parser/<lang>/` may import from another
    language's parser package. Cross-language sharing happens only through
    the `LanguageParser` interface itself or genuinely language-agnostic
    shared code (`internal/codecontext`) -- restating `AGENTS.md`'s existing
    boundary here since this interface is exactly the seam that enforces it.
11. `Language()`'s doc comment must note that its viability as a static,
    per-implementation method is provisional. See `memory/known-gaps.md`'s
    entry owned by 006a: TypeScript compiles to `.js` before running, which
    may make JavaScript and TypeScript indistinguishable from parsed trace
    content alone in the common case, and a static return may need to
    become a `Parse()`-time determination instead once real traces are
    examined.

## Non-functional requirements

- No new third-party dependency: this feature needs only `context` and
  `errors` from the standard library.
- `Detect()`'s no-I/O constraint (requirement 7) is also a performance
  requirement: calling it across every registered parser must stay cheap
  enough that auto-detection (003b) doesn't need to think about cost.

## Out of scope

- Bucketing rules, chain-elision rules (`elidedFrameCount` computation), and
  runtime-detection heuristics -- all 005a/006a's own scope, same pattern
  001 already used for `Bucket`.
- Actual parser implementations for Java (005a) and TS/JS (006a).
- The auto-detection registry itself: registration mechanism,
  ambiguous/undetected resolution logic (CLI exit code 4), mapping
  `--lang` hint values to registered parsers -- all 003b.
- Any error-code taxonomy beyond the single plain sentinel `ErrUnparseable`
  -- deferred, see `plan.md`'s Alternatives considered.
- Full pipeline wiring (002b).

## Acceptance criteria

- [ ] Given the `LanguageParser` interface as defined, when a representative
      real-shaped Java trace (top-level exception with a `Caused by:` chain,
      including own-code, dependency, and runtime-bucket frames) is
      hand-traced through `Detect()` and `Parse()` as pseudocode in
      `plan.md`, then every piece of information the trace contains maps
      cleanly onto the interface's return values, with nothing
      unrepresentable and no workaround needed.
- [ ] Given the same interface, when a representative real-shaped TS/JS
      trace (`Error` with an `[cause]` chain, including own-code,
      dependency, and runtime-bucket frames, and column numbers) is
      hand-traced the same way, then the same holds.
- [ ] Given `internal/parser/registry.go`, when reviewed, then `Parse()`'s
      doc comment explicitly states the `Frames[0]`-is-originating-frame
      rule, the full-chain-or-error rule, and the `ErrUnparseable`-wrapping
      rule; `Detect()`'s doc comment explicitly states the no-I/O
      constraint; and both `Detect()`'s and `Parse()`'s doc comments
      explicitly state the `rawTrace` non-empty precondition (requirement
      8).
- [ ] Given `internal/parser/registry.go` and `internal/parser/errors.go`
      with zero implementations of `LanguageParser` existing yet, when
      `gofumpt -l`, `golangci-lint run`, and `go build ./...` are run, then
      all three pass cleanly.
- [ ] Given `registry.go`'s imports, when reviewed, then it imports only
      `internal/contract` and stdlib (`context`) -- nothing from any
      `parser/<lang>/` package.

## Open questions

None remaining -- all resolved during interrogation (see
`specs/003a-language-parser-interface/progress.md` for the session log).

# Spec: Language auto-detection registry

**Status:** Approved
**Folder:** specs/003b-language-auto-detection-registry
**Depends on:** 001-data-contract (done), 003a-language-parser-interface (done), 006a-ts-js-parser (done)

## Overview

Given a raw trace and an explicit list of candidate `LanguageParser` (003a)
implementations, determines which single parser should handle it by calling
each candidate's `Detect()` and branching on how many claim a match: exactly
one is the normal case, zero means the trace didn't look like any registered
language, and two or more means the trace is genuinely ambiguous between
registered languages. This feature owns only that selection logic -- it has
no opinion on which candidates a caller passes in (e.g. a `--lang` hint
narrowing the list is 002b's job), how registration happens (there is no
global/package-level registry; callers build the candidate slice
themselves), or how the resulting error becomes a CLI exit code or
user-facing message (also 002b).

`005a` (Java parser) was originally listed as a dependency of this feature.
That was a mistake in `specs/INDEX.md`: this feature needs at least one real
`LanguageParser` to exist to be meaningful, not a second one. It was
corrected to depend on `006a` only. Verifying that two independently
developed real parsers' heuristics never accidentally overlap is a related
but separate concern -- see Out of scope.

## User stories

- As the developer wiring 002b's pipeline, I want a single function that
  tells me which parser to use (or a specific, distinguishable reason it
  can't), so I don't have to hand-roll the 0/1/2+ counting logic myself.
- As a developer whose trace doesn't match any registered parser, I want
  that to fail clearly and distinctly from an ambiguous match, so I get a
  message that actually explains what happened instead of a generic
  failure.
- As a developer running `cmd/typescript` (which registers both
  `javascriptParser` and `typescriptParser`, per 006a's two-value design,
  with no `--lang` flag available to disambiguate -- 002a FR2), I want the
  tool to still correctly pick JavaScript vs. TypeScript automatically from
  the trace content.
- As a maintainer adding a future language parser, I want confidence that
  this feature's ambiguous-detection branch is real, exercised logic --
  not dead code that's never actually run -- even before a second real
  language exists to prove it end-to-end.

## Functional requirements

1. `internal/parser` exposes, in a new file `internal/parser/detect.go`:
   ```go
   func DetectLanguage(rawTrace string, candidates []LanguageParser) (LanguageParser, error)
   ```
2. `DetectLanguage` calls `Detect(rawTrace)` on every element of
   `candidates`, in order, and collects every one that returns `true`.
3. Exactly one match: returns that `LanguageParser` and a `nil` error.
4. Zero matches: returns `nil` and an error wrapping a new sentinel
   `ErrNoMatch` (added to `internal/parser/errors.go`, alongside
   `ErrUnparseable`), with the error's message text naming every checked
   candidate's `Language()` value (comma-joined, prefixed "checked "),
   mirroring FR5's `ErrAmbiguous` message shape -- so a caller logging
   this error doesn't need to re-derive which parsers were registered.
5. Two or more matches: returns `nil` and an error wrapping a new sentinel
   `ErrAmbiguous` (same file), with the error's message text naming every
   matched candidate's `Language()` value (comma-joined), so a caller
   logging this error doesn't need to re-derive which languages collided.
6. `candidates` being empty is a programmer-error invariant (a binary wired
   with zero registered parsers is a build-time wiring bug, not something
   real trace input can trigger) -- `DetectLanguage` panics rather than
   returning `ErrNoMatch`, consistent with `CONVENTIONS.md`'s existing
   panic guidance ("Panics are reserved for genuine programmer-error
   invariants, not runtime conditions").
7. `DetectLanguage` performs no I/O itself and takes no
   `context.Context` -- it only calls already-guaranteed-side-effect-free
   `Detect()` implementations (003a's interface constraint), so there is
   nothing here that could block or need cancellation.
8. This feature does not implement any hint-based narrowing of
   `candidates` (e.g. mapping a `--lang` CLI value to a subset) -- a caller
   (002b) that wants to narrow the list before calling `DetectLanguage`
   builds that narrowed slice itself.
9. This feature does not implement a global/package-level parser registry
   (no `Register()` function, no init-time side effects) -- `candidates` is
   supplied explicitly by the caller on every call. `003a`'s `registry.go`
   doc comment reserved "registration map, `Register()` function, or
   detection-orchestration logic" as this feature's scope; this resolves
   that reservation as an explicit-parameter design rather than a stateful
   one.

## Non-functional requirements

- No new third-party dependency: `errors`, `fmt`, `strings` from stdlib
  only.
- Cost is O(len(candidates)) calls to `Detect()`, each of which is already
  required (003a) to be fast and side-effect-free -- no additional
  performance concern introduced here.

## Out of scope

- Mapping the CLI `--lang` hint (002a) to a candidate subset -- 002b's job.
  `002a`'s existing `java`/`typescript` flag values were written before
  006a split the "typescript" family into two separately-registered
  parsers (`javascriptParser`, `typescriptParser`); a hint of `typescript`
  no longer maps to a single parser. This feature does not resolve that --
  it's flagged here as a real open question for whoever specs 002b next,
  not silently absorbed into this feature's scope.
- Any global/package-level parser registration mechanism -- deliberately
  rejected; see FR9.
- CLI exit-code mapping and user-facing message construction from
  `ErrNoMatch`/`ErrAmbiguous` -- 002b's job. `CONVENTIONS.md` already
  reserves exit code 4 for "source language ambiguous or undetected,"
  covering both sentinels; 002b distinguishes which one occurred via
  `errors.Is`.
- Non-`Error` JavaScript/TypeScript thrown values (`throw "string"`,
  `throw {...}`). Confirmed during this feature's interrogation, by
  running `detectNodeTrace` against the real
  `full-machine-reverify` #11 capture, that these actually return
  `javascriptParser.Detect() == true` (via the crash-preamble/
  trailing-version-line signal 006a's FR4 added to rescue
  `zero-stack-trace-limit.txt`), then fail at `Parse()` with
  `ErrUnparseable` -- they never reach this feature's 0-match path at all.
  `memory/known-gaps.md` previously and incorrectly attributed a "no
  parser matched" outcome for this case to this feature; that entry was
  corrected (removed) during this feature's interrogation.
- Verifying that two real, independently-developed language parsers'
  `Detect()` heuristics don't accidentally overlap on some pathological
  real trace. The 0/1/2+ branching logic itself is fully implemented and
  tested here (see `plan.md`'s Testing strategy) using a hand-written fake
  second parser -- `javascriptParser`/`typescriptParser` are constructed to
  never both match the same real trace (006a's own doc comment confirms
  this), so no real fixture can exercise the ambiguous branch today.
  Confirming no real false-positive overlap needs an actual second real
  language parser to exist against. Recorded in `memory/known-gaps.md`,
  owned by 005a (the next real language-parser feature on
  `specs/INDEX.md`).

## Acceptance criteria

- [ ] Given a real trace that only `typescriptParser.Detect()` returns
      `true` for (e.g. a trace with a `.ts`-suffixed frame, such as
      `internal/parser/typescript/testdata/ts-native-execution.txt`) and
      both `javascriptParser`/`typescriptParser` as candidates, when
      `DetectLanguage` runs, then it returns `typescriptParser` and a
      `nil` error.
- [ ] Given the real `bare-stack-fetch-cause.txt` content (literally
      `"TypeError: fetch failed"` -- the confirmed real case where both
      `javascriptParser.Detect()` and `typescriptParser.Detect()` return
      `false`) with both as candidates, when `DetectLanguage` runs, then
      it returns a `nil` parser and an error for which
      `errors.Is(err, parser.ErrNoMatch)` is `true`, and the error's
      message names both checked candidates' `Language()` values
      (`javascript`, `typescript`).
- [ ] Given two candidate parsers (hand-written fakes -- no real
      two-language combination produces this today, see Out of scope)
      whose `Detect()` both return `true` for the same input, when
      `DetectLanguage` runs, then it returns a `nil` parser and an error
      for which `errors.Is(err, parser.ErrAmbiguous)` is `true`, and the
      error's message names both candidates' `Language()` values.
- [ ] Given an empty `candidates` slice, when `DetectLanguage` runs, then
      it panics rather than returning `ErrNoMatch`.
- [ ] Given `internal/parser/detect.go` and the updated `errors.go`, when
      `gofumpt -l`, `golangci-lint run`, `go build ./...`, and
      `go test ./...` are run, then all pass cleanly.

## Open questions

None remaining -- all resolved during interrogation (see
`specs/003b-language-auto-detection-registry/progress.md` for the session
log).

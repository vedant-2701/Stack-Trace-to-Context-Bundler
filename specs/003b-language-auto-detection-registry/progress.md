# Progress Log: Language auto-detection registry

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-09-13
**Task(s):** Pre-implementation — spec.md, plan.md, tasks.md written; no
code tasks (T001-T006) started yet.
**What happened:**
- `specs/INDEX.md`'s dependency list for this feature originally read
  `001, 003a, 005a, 006a`. 005a (Java parser) was flagged as a blocker
  during the initial dependency check (status: idea, not started) and
  Vedant confirmed it was a documentation mistake — this feature needs at
  least one real `LanguageParser` to exist, not two. Vedant corrected
  `specs/INDEX.md` to `001, 003a, 006a` before interrogation continued.
- During grounding reads, found and verified (by copying the real
  `detectNodeTrace` code into a throwaway Go program and running it
  against the real `full-machine-reverify` #11 fixture) that
  `memory/known-gaps.md` contained a stale entry: it claimed non-`Error`
  thrown JS/TS values are undetectable and that the resulting "no parser
  matched" outcome was owed to this feature. Actually,
  `javascriptParser.Detect()` returns `true` for these (via the
  crash-preamble/trailing-version-line OR-relaxation added in 006a's
  T003), and they fail downstream at `Parse()` with `ErrUnparseable`
  instead — never reaching this feature's 0-match path. Vedant confirmed;
  the stale row was removed from `memory/known-gaps.md`.
- Also verified `javascriptParser.Detect()`/`typescriptParser.Detect()`
  are constructed to be mutually exclusive on any real trace (one requires
  no `.ts`/`.tsx` frame, the other requires at least one) — confirmed
  against `internal/parser/typescript/typescript.go`. This means no real
  fixture can currently exercise this feature's ambiguous (2+ match)
  branch; that's tracked as a known gap owned by 005a rather than blocking
  this feature (resolved via hand-written fake parsers for testing
  instead — see Deviations).
- Resolved scope: this feature takes an explicit `[]LanguageParser` from
  the caller (no global/package-level registry, no `Register()` function)
  and does not handle `--lang` CLI hint mapping (002b's job). Flagged, but
  did not resolve, a real conflict: `002a`'s `--lang` flag values
  (`java`/`typescript`) predate 006a's split of the "typescript" family
  into two separately-registered parsers and don't map cleanly onto them.
  Left for 002b's own spec interrogation.
- Resolved API shape: `DetectLanguage(rawTrace string, candidates
  []LanguageParser) (LanguageParser, error)` in new file
  `internal/parser/detect.go`; two new sentinels `ErrNoMatch`/
  `ErrAmbiguous` in `errors.go`; empty `candidates` panics (programmer
  error) rather than returning `ErrNoMatch`.
**Deviations from plan (if any):** None yet — plan.md's Testing strategy
(real fixtures for the no-match/single-match cases, hand-written fakes only
for the ambiguous case) was decided jointly with Vedant during
interrogation, not deviated from afterward.
**New open questions:** None for this feature. Carried forward to 002b (not
this feature's to resolve): how should `--lang=typescript` interact with
006a's two registered parsers now that the hint no longer maps 1:1 to a
parser?

---

**Date:** 2026-09-13
**Task(s):** Pre-implementation spec-integrity audit (no code tasks
started).
**What happened:**
- Audit surfaced that `plan.md`'s two API/contracts code blocks
  (`errors.go` additions, `detect.go`) each carried a file-path label
  comment directly above `package parser` (e.g. `// internal/parser/
  detect.go`) -- exactly the anti-pattern `CONVENTIONS.md` bans, since
  `revive`'s `package-comments` check (enabled in `.golangci.yml`) fails
  any comment immediately preceding `package X` unless it starts with
  `"Package X ..."`. If T001/T002 had transcribed those blocks literally,
  `golangci-lint run` would have failed at T006's full-repo gate despite
  T001/T002's own (narrower) acceptance criteria appearing to pass.
  Fixed: both label-comment lines removed from `plan.md`.
- Also flagged: the `ErrNoMatch` branch (`fmt.Errorf("%w", ErrNoMatch)`)
  added no context beyond the sentinel itself, unlike the `ErrAmbiguous`
  branch three lines below it, which names every matched candidate --
  inconsistent with `CONVENTIONS.md`'s "always wrap with context, never a
  bare re-throw" rule and structurally asymmetric with its sibling
  branch. Resolved (Vedant's call): the zero-match branch now also names
  every *checked* candidate's `Language()` value, prefixed `"checked "`,
  mirroring `ErrAmbiguous`'s shape -- e.g. `"checked javascript,
  typescript: no registered parser matched this trace"`. Updated
  `plan.md` (doc comment, code, testing-strategy bullet), `spec.md` (FR4,
  second acceptance criterion), and `tasks.md` (T003's acceptance
  description) to match.
**Deviations from plan (if any):** The `DetectLanguage` zero-match
branch now builds a `names` slice from `candidates` (all of them, since
none matched) in addition to the existing `matched`-based one in the
ambiguous branch -- two separate loops, not shared, since they iterate
different slices. Not a deviation from the *intent* already recorded
above (`ErrNoMatch` was always meant to wrap the sentinel), just a
late-added requirement on top of it.
**New open questions:** None.

---

**Date:** 2026-09-14
**Task(s):** T001 — Add `ErrNoMatch` and `ErrAmbiguous` sentinels to `internal/parser/errors.go`.
**What happened:**
- Added `ErrNoMatch` (`"no registered parser matched this trace"`) and
  `ErrAmbiguous` (`"trace matched more than one registered language"`) as
  package-level `var`s in `internal/parser/errors.go`, directly below the
  existing `ErrUnparseable`, each with a doc comment matching `plan.md`'s
  API/contracts block (cross-referencing `DetectLanguage`, not yet
  implemented, and the exit-code-4 mapping owned by 002b).
- No file-level package comment existed above `package parser` in this
  file, so no risk of the `revive` package-comments trap noted in this
  feature's earlier progress entry.
- Verified: `go build ./...`, `gofumpt -l internal/parser/errors.go`,
  `golangci-lint run ./internal/parser/...`, `go test ./internal/parser/...`
  all clean (Vedant ran and confirmed).
**Deviations from plan (if any):** None — matches `plan.md`'s
API/contracts block verbatim.
**New open questions:** None.

---

**Date:** 2026-09-14
**Task(s):** T002 — Implement `DetectLanguage` in new file `internal/parser/detect.go`.
**What happened:**
- Implemented `DetectLanguage(rawTrace string, candidates []LanguageParser) (LanguageParser, error)`
  in `internal/parser/detect.go`, matching `plan.md`'s API/contracts block
  verbatim: empty `candidates` panics; 0 matches wraps `ErrNoMatch` naming
  every checked candidate's `Language()`; exactly 1 match returns that
  parser; 2+ matches wraps `ErrAmbiguous` naming every matched candidate.
  Confirmed against the actual `LanguageParser` interface in `registry.go`
  before implementing — no drift from `plan.md`'s assumed shape.
- Verified: `go build ./...`, `gofumpt -l internal/parser/detect.go`,
  `golangci-lint run ./internal/parser/...`, `go test ./internal/parser/...`
  all clean (Vedant ran and confirmed).
**Deviations from plan (if any):** None — matches `plan.md`'s
API/contracts block verbatim.
**New open questions:** None.

---

**Date:** 2026-09-14
**Task(s):** T003 — `detect_test.go`: single real match and real no-match cases.
**What happened:**
- Added `internal/parser/detect_test.go` as `package parser_test`
  (external test package, per `plan.md`'s Testing strategy — required to
  import `internal/parser/typescript` without an import cycle).
- `TestDetectLanguage_RealSingleMatch`: real `typescript.NewJavaScriptParser()`
  + `typescript.NewTypeScriptParser()` as candidates against the real
  `ts-native-execution.txt` content (inlined as a const string literal,
  copied verbatim from disk) — asserts the returned parser's `Language()`
  is `contract.LanguageTypeScript`.
- `TestDetectLanguage_RealNoMatch`: same two real candidates against the
  real `bare-stack-fetch-cause.txt` content (`"TypeError: fetch
  failed\n"`, inlined verbatim) — asserts `errors.Is(err,
  parser.ErrNoMatch)` and that the error message contains both
  `"javascript"` and `"typescript"` (confirmed exact string values via
  `internal/contract/types.go`'s `LanguageJavaScript`/`LanguageTypeScript`
  constants before writing the assertion).
- No fakes used — both cases exercise real, already-tested 006a
  production code, per `plan.md`.
- Verified: `go build ./...`, `gofumpt -l internal/parser/detect_test.go`,
  `golangci-lint run ./internal/parser/...`, `go test ./internal/parser/...`
  all clean (Vedant ran and confirmed).
**Deviations from plan (if any):** None — matches `plan.md`'s Testing
strategy verbatim.
**New open questions:** None.

---

**Date:** 2026-09-14
**Task(s):** T004 — `detect_test.go`: ambiguous case (hand-written fakes) and empty-candidates panic.
**What happened:**
- Added `fakeLanguageParser` (hand-written fake, no mocking framework, per
  `CONVENTIONS.md`) to `internal/parser/detect_test.go`: a struct with a
  configurable `lang` and `matches` field implementing `LanguageParser`.
- `TestDetectLanguage_Ambiguous`: two `fakeLanguageParser` candidates,
  both `Detect()` returning `true`, distinct `Language()` values
  (`"fake-a"`, `"fake-b"`) — asserts `errors.Is(err,
  parser.ErrAmbiguous)` and that the error message names both.
- `TestDetectLanguage_EmptyCandidatesPanics`: calls `DetectLanguage(...,
  nil)` inside a `recover()`, asserts a panic occurred.
- First lint run flagged two `revive` unused-parameter issues
  (`fakeLanguageParser.Detect`'s `rawTrace`, `.Parse`'s `ctx`) — fixed by
  renaming both to `_`, since the fake never needs to inspect them
  (`Detect` always returns the hardcoded `matches` field; `Parse` is
  never called by `DetectLanguage` or these tests).
- Verified: `go build ./...`, `gofumpt -l internal/parser/detect_test.go`,
  `golangci-lint run ./internal/parser/...`, `go test ./internal/parser/...`
  all clean (Vedant ran and confirmed) after the fix.
**Deviations from plan (if any):** None beyond the unused-parameter fix
above, which is a lint-driven implementation detail, not a change to
`plan.md`'s Testing strategy itself.
**New open questions:** None.

---

**Date:** 2026-09-14
**Task(s):** T005 — Record the real cross-language ambiguity gap in `memory/known-gaps.md`.
**What happened:**
- Re-read `memory/known-gaps.md` fresh (not a cached copy), per the
  file's own edit discipline, before editing.
- Added one new row to the "Deferred acceptance criteria" table: source
  feature `003b-language-auto-detection-registry`, criterion noting
  `DetectLanguage`'s ambiguous (2+ match) branch is proven correct only
  via hand-written fakes (T004) and that no real fixture can exercise it
  because `javascriptParser`/`typescriptParser` (006a) are constructed to
  never both match the same real trace, owner `005a`, status `pending`.
- Verified the row reads correctly against the file's existing pipe-table
  format by reading the file back after the edit, per the task's own
  acceptance criterion.
- Doc-only change — no Go code touched, so the usual 4 gate commands
  don't apply; Vedant confirmed the doc content directly.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

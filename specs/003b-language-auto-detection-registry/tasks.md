# Tasks: Language auto-detection registry

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.

- [x] **T001** — Add `ErrNoMatch` and `ErrAmbiguous` sentinels to `internal/parser/errors.go`
  - Depends on: none
  - Acceptance: both `var`s exist alongside the existing `ErrUnparseable`,
    with doc comments matching `plan.md`'s API/contracts block; `go build
    ./...` passes; `gofumpt -l` reports no diff for the file.

- [ ] **T002** — Implement `DetectLanguage` in new file `internal/parser/detect.go`
  - Depends on: T001
  - Acceptance: signature matches `spec.md` FR1 exactly
    (`func DetectLanguage(rawTrace string, candidates []LanguageParser) (LanguageParser, error)`);
    implements the 0/1/2+ branching (FR3-5) and the empty-`candidates`
    panic (FR6); `golangci-lint run` and `go build ./...` pass clean.

- [ ] **T003** — `detect_test.go`: single real match and real no-match cases
  - Depends on: T002
  - Acceptance: new file `internal/parser/detect_test.go`, `package
    parser_test` (external test package — required to import
    `internal/parser/typescript` without an import cycle, per `plan.md`'s
    Testing strategy). Covers:
    - a real `.ts`-frame trace against real `NewJavaScriptParser()` +
      `NewTypeScriptParser()` candidates → asserts `typescriptParser`
      returned, matching `spec.md`'s first acceptance criterion;
    - the real `bare-stack-fetch-cause.txt` content against the same two
      real candidates → asserts `errors.Is(err, parser.ErrNoMatch)` and
      that the error message names both checked candidates
      (`javascript`, `typescript`), matching `spec.md`'s second
      acceptance criterion.
    Both subtests pass; no other tests in the package regress.

- [ ] **T004** — `detect_test.go`: ambiguous case (hand-written fakes) and empty-candidates panic
  - Depends on: T003
  - Acceptance: two hand-written fake `LanguageParser` types declared
    locally in `detect_test.go`, both `Detect()` hardcoded `true`,
    distinct `Language()` values → asserts `errors.Is(err,
    parser.ErrAmbiguous)` and that the error message names both languages,
    matching `spec.md`'s third acceptance criterion. A separate subtest
    calls `DetectLanguage(rawTrace, nil)` inside a `recover()` and asserts
    a panic occurred, matching `spec.md`'s fourth acceptance criterion.
    `go test ./internal/parser/... -run TestDetectLanguage -v` shows all
    subtests passing.

- [ ] **T005** — Record the real cross-language ambiguity gap in `memory/known-gaps.md`
  - Depends on: T004
  - Acceptance: re-read `memory/known-gaps.md` fresh (per its own edit
    discipline), add one row to the "Deferred acceptance criteria" table:
    source feature `003b-language-auto-detection-registry`, describing
    that the ambiguous branch is proven correct only via hand-written
    fakes (T004) and that no real fixture can exercise it because
    `javascriptParser`/`typescriptParser` are constructed to never both
    match the same trace (006a), owner `005a`, status `pending`. Row
    reads correctly against the file's existing table format (verify by
    reading the file back after the edit).

- [ ] **T006** — Full gate run and `progress.md` close-out
  - Depends on: T005
  - Acceptance: `gofumpt -l ./...`, `golangci-lint run`, `go build
    ./...`, `go test ./...` all run clean across the whole repo (not just
    `internal/parser`) — confirms no regression in `internal/parser/typescript`
    or elsewhere. `progress.md` has a final entry marking all tasks
    `[x]` and gates clean. Commit message provided (header + body) per
    `COMMIT_CONVENTIONS.md`. Do not push without Vedant's explicit
    confirmation.

<!-- Keep each task small enough to implement and verify in a single sitting.
     If a task feels big, split it. -->

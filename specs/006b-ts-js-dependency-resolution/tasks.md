# Tasks: TS/JS dependency resolution

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.

- [x] **T000** — Contract amendment (blocks all other work)
  - Depends on: none
  - Change `internal/contract/types.go`: `Bundle.Dependencies` ->
    `*Dependencies` (`json:"dependencies,omitempty"`); bump
    `SchemaVersion` `"2.0.0"` -> `"3.0.0"`; relax `LockedDependency`'s
    doc comment (Note may accompany a present Version).
  - While touching `types_test.go`'s `exampleTSBundle()`: fix the
    existing unresolved-dependency `Note` text, which currently reads
    "no local npm cache on this checkout (Article IX, decision 0001)"
    — copy-pasted from the Java example and factually wrong for TS/JS
    (this feature never checks an "npm cache"; Article IX/decision 0001
    is Java's mvn/gradle offline-cache decision). Replace with
    TS/JS-accurate wording (e.g. "no package-lock.json entry found for
    this package"). Independent of whether 006b's actual resolver
    exists yet — this fixture is hand-authored test data, not derived
    from a real resolver run.
  - Regenerate `internal/contract`'s golden fixtures
    (`testdata/example_java.json`/`example_ts.json`,
    `go test ./internal/contract/... -update`, Vedant-run).
  - Update `001-data-contract/spec.md` and `plan.md` in place to reflect
    the new shape (per the practice 004 already established).
  - Acceptance: `go build ./...`, `go test ./internal/contract/...`,
    `golangci-lint run ./internal/contract/...`, `gofumpt -l` on all
    changed files all clean; golden fixtures regenerated and reviewed.

- [x] **T001** — `codecontext.FindRepoRoot`
  - Depends on: T000 (not a hard dependency, but keeps the contract
    change isolated as its own reviewable commit first)
  - Add `FindRepoRoot(ctx context.Context, workDir string) (root string, ok bool)`
    to `internal/codecontext/gitmeta.go`, reusing the existing
    `gitRunner`/`gitTimeout`/`execGitRunner`. `git rev-parse
    --show-toplevel`; `ok=false` on any error or timeout.
  - Tests via the existing `fakeGitRunner` pattern
    (`runner_fake_test.go`): success, git error, timeout.
  - Acceptance: all 4 verification commands clean; `TestFindRepoRoot_*`
    covers success/error/timeout.

<!-- T002 and T003a below have no real dependency on T000: neither
     touches internal/contract.Dependencies or SchemaVersion, so
     T000-first is a hygiene/sequencing choice here, not a technical
     requirement. See T003b and T004 for where T000 actually starts
     to matter. -->

- [x] **T002** — npm manifest parsing (`Direct`)
  - Depends on: none. Parses `package.json` into a plain
    `map[string]string` and touches no `internal/contract` type at
    all — can be worked in parallel with T000.
  - Add `internal/dependency/typescript/manifest.go`: parse
    `package.json`, extract `dependencies`+`devDependencies`+
    `optionalDependencies` (excluding `peerDependencies`) into a
    `map[string]string` (declared range). Malformed/missing file ->
    `ok=false`.
  - `manifest_test.go`: valid manifest (all 3 sections), malformed JSON,
    missing file, peerDependencies-excluded case.
  - Acceptance: all 4 verification commands clean.

- [x] **T003a** — npm lockfile parsing
  - Depends on: none. Pure JSON parsing/validation; touches no
    `internal/contract` type.
  - Add `internal/dependency/typescript/lockfile.go`: parse
    `package-lock.json` (`lockfileVersion`, `packages` map).
    `lockfileVersion` other than 2/3, or malformed JSON -> `ok=false`
    with a reason string.
  - `lockfile_test.go`: valid v2/v3 parse, malformed lockfile, missing
    lockfile, `lockfileVersion: 1`, `yarn.lock`-only (treated as
    missing).
  - Acceptance: all 4 verification commands clean.

- [x] **T003b** — `packageDirKey` + exact/fallback lookup
  - Depends on: T003a (needs the parsed `packages` map to look
    anything up in). Also T000, but only as a documentation-
    consistency matter, not a compile dependency: this task's fallback
    case constructs `contract.LockedDependency{Version, Note}`
    together, which matches the *documented* invariant only once
    T000's relaxed comment has landed — the struct's fields themselves
    are unchanged, so nothing actually fails to build either way.
  - Add `internal/dependency/typescript/lookup.go`: implement
    `packageDirKey` (plan.md's exact algorithm) and a per-frame
    exact-match / top-level-fallback lookup function. Returns one
    frame's own result only (version/note/matched) — does not
    aggregate across multiple frames sharing a package name; that
    aggregation, and the conflict rule it feeds, is T004's job.
  - `lookup_test.go`: exact match, scoped package, nested-duplicate-
    version (proves `packageDirKey` returns the right nested key, not
    the top-level one, when both exist), top-level fallback (no exact
    match), fully unresolved (neither match).
  - Acceptance: all 4 verification commands clean; every case above has
    a dedicated test.

- [ ] **T004** — `ResolveDependencies` orchestration
  - Depends on: T001, T002, T003a, T003b. Also design-coupled to T000
    (its return type `*contract.Dependencies` is shaped to match
    T000's contract change) but not compile-blocked by it:
    `&contract.Dependencies{...}` compiles regardless of what type
    `Bundle.Dependencies` is declared as elsewhere, since it only takes
    the address of a locally-built literal. T000-first remains correct
    for hygiene/consistency, not because anything here fails to build
    otherwise.
  - Add `internal/dependency/typescript/resolve.go`: `ResolveDependencies`
    per plan.md's signature. Scopes `Direct`/`Locked` to only packages
    referenced by a `dependency`-bucket frame's `PackageName` in the
    given `Chain` (spec.md FR9). Implements the `workDir` fallback
    (spec.md FR6/FR7, plan.md step 1a): `FindRepoRoot` fails -> try
    `package.json` directly in `workDir`, no upward walk; still no
    manifest -> `nil`. `FindRepoRoot` succeeds but no manifest at that
    root -> `nil` directly, `workDir` is NOT additionally checked.
    Implements the multi-frame conflict rule (spec.md FR13): for each
    referenced package, if its frames' `lookup.go` results (across any
    combination of exact and fallback matches) disagree on version,
    override with the conflict outcome (`Version` omitted, `Note`
    names the conflicting versions) rather than keeping whichever
    frame was processed last. Adds centralized `slog.Warn`/`slog.Info`
    calls per plan.md's "Logging" section. Never returns a Go error,
    and never panics, in any of these cases.
  - `resolve_test.go`: full end-to-end table across the spec.md
    acceptance-criteria list (referenced-vs-unreferenced package
    scoping, Direct/Locked combinations for dependencies/dev/optional/
    peer/transitive, no-manifest -> nil) plus the three root-resolution
    cases explicitly: no git repo + manifest in `workDir` (resolves via
    fallback), no git repo + no manifest in `workDir` either (nil, and
    a dedicated test asserting no panic/no process exit), git succeeds
    + no manifest at that root (nil, fallback NOT attempted); plus the
    multi-frame conflict override (two frames, same `PackageName`,
    disagreeing versions via exact+exact and exact+fallback) and its
    companion case (one frame resolves, the other simply has no match
    at all — the resolved version stands, no conflict, no blanking).
  - Acceptance: all 4 verification commands clean; every spec.md
    acceptance criterion has a corresponding test, cross-checked before
    marking this task done.

- [ ] **T005** — Feature close-out
  - Depends on: T000, T001, T002, T003a, T003b, T004
  - Cross-check all 15 functional requirements and all 20 acceptance
    criteria in `spec.md` against actual code/tests; check off each
    satisfied criterion with a pointer to the test(s) that satisfy it,
    same style as `001-data-contract/spec.md`'s close-out pass.
  - Update `specs/INDEX.md`: 006b status `planned`/`in-progress` -> `done`.
  - Update `CONVENTIONS.md`'s File/folder layout diagram to add the new
    `internal/dependency/typescript/` package (not previously listed),
    with a short note that a `java/` sibling is expected for 005b --
    same practice as 004 updating `001-data-contract`'s docs in place
    for a structural change.
  - Append a final `progress.md` entry summarizing close-out.

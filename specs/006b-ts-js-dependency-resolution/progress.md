# Progress Log: TS/JS dependency resolution

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-09-14
**Task(s):** Spec/plan/tasks interrogation (pre-implementation)
**What happened:** Full interrogation session, resolved via one-question-at-
a-time elicitation. Key decisions, in order:
1. npm only for v1 (no yarn.lock/pnpm-lock.yaml) -- pnpm-lock.yaml is real
   YAML and go.mod has zero YAML dependencies today.
2. npm lockfileVersion 2/3 only (flat `packages` map) -- v1's nested-tree
   format is a different parser, npm <=6 is years EOL.
3. Version-conflict resolution: match a dependency-bucket frame's own
   FilePath against the lockfile's path-keyed `packages` entries (exact
   nested match) rather than a blanket "always use the top-level version"
   rule -- the blanket rule would confidently report a version for a
   frame provably running a different nested copy (Article VI). Falls
   back to a top-level bare-name match, labeled inexact via `Note`, when
   no exact path match exists.
4. `Direct` sections: dependencies+devDependencies+optionalDependencies
   (peerDependencies excluded -- declares a range without an install).
5. `Direct`/`Locked` scope: only packages referenced by a dependency-
   bucket frame in this trace, not every package.json-declared
   dependency -- mirrors `CodeContexts`' own-bucket-only precedent.
6. Repo-root discovery: `git rev-parse --show-toplevel` (new consumer --
   004 deliberately never computed one). No existing utility; added as
   `codecontext.FindRepoRoot`, reusing that package's existing
   `gitRunner`/`gitTimeout`.
7. No manifest found (repo-root undeterminable, package.json absent, OR
   malformed/invalid JSON) -> `Bundle.Dependencies` nil, all three cases
   treated identically.
8. No lockfile (absent, unsupported-format-only, malformed, or
   lockfileVersion != 2/3) -> per-package `Locked` entries: `Version`
   omitted, `Note` explains why. Never aborts the whole bundle.
9. **Contract conflict caught and resolved:** the inexact-fallback design
   (decision 3) needs `Version` and `Note` set together, but
   `LockedDependency`'s existing doc comment (from `001`, shipped) said
   "Note is present only when Version is absent." Resolved by relaxing
   that doc comment (no wire-format change, both fields already
   independently `omitempty`) rather than demoting inexact matches to
   fully unresolved.
**Deviations from plan (if any):** N/A -- pre-implementation.
**New open questions:** None -- all resolved. See `spec.md` for the full
functional-requirements writeup and `plan.md` for the resulting
architecture (new `internal/dependency/typescript/` package,
`codecontext.FindRepoRoot` addition, and the two `001` contract
amendments scoped as this feature's first task, T000).

---

**Date:** 2026-09-15
**Task(s):** Spec amendment (pre-implementation, before T000 started)
**What happened:** User caught a real gap: FR6 as originally written
treated "not inside a git working tree" identically to "no manifest
found" with no fallback, which means a legitimate non-git-initialized
JS project (package.json present, no `.git` yet) would silently get no
dependency resolution at all. User proposed fixing this by falling back
to checking `workDir` directly for `package.json` when git detection
fails (assuming the CLI is invoked from the project root), and, if that
also fails, panicking.

Pushed back on the panic half: a Go panic, unrecovered, kills the whole
process -- not just the `Dependencies` section -- which directly
contradicts FR7's own "valid, representable nil outcome, never an
error" framing (itself mirroring `BuildGitMetadata`'s established
pattern) and would be a much worse regression than the gap being fixed.
User agreed: no panic, `Dependencies` nil instead, consistent with
every other unresolvable-manifest case in the spec.

Amended FR6 (added the `workDir` fallback, no upward walk, applies only
when git detection itself fails -- not when git succeeds but the root
lacks a manifest) and FR7 (nil applies uniformly across both root
sources, explicitly "never a panic"). Added 2 net new acceptance
criteria (16 -> 18) covering: no-git + manifest-in-workDir (resolves),
no-git + no-manifest-anywhere (nil, not a panic), and git-succeeds +
no-manifest-at-that-root (nil, no fallback retry). Updated plan.md
(architecture step 1a, `ResolveDependencies` pseudocode, testing
strategy, a new risk entry for the timeout-grouped-with-no-repo
simplification, and two new "Alternatives considered" entries) and
tasks.md (T004, T005's acceptance-criteria count) to match. Decided
(not asked, flagged instead): `codecontext.FindRepoRoot` itself stays
pure git-only -- the `workDir` fallback is 006b-specific logic in
`internal/dependency/typescript`, not baked into the shared
codecontext function, since `BuildGitMetadata`'s other use of
"no repo found" genuinely means no blame info, not a guessable root.
**Deviations from plan (if any):** N/A -- pre-implementation, no code
written yet to deviate from.
**New open questions:** None -- resolved. See `spec.md` FR6/FR7 and the
updated acceptance criteria for the final shape.

---

**Date:** 2026-09-16
**Task(s):** T000 -- Contract amendment
**What happened:** Changed `internal/contract/types.go`:
`Bundle.Dependencies` `Dependencies` -> `*Dependencies`
(`json:"dependencies,omitempty"`), same nil-when-absent pattern as
`GitMetadata`; bumped `SchemaVersion` `"2.0.0"` -> `"3.0.0"` (MAJOR,
since this is a field changing from always-required to sometimes-
absent, per FR6's own bump policy); relaxed `LockedDependency.Note`'s
doc comment so it can accompany a present `Version` (needed for T003b's
inexact-fallback design, decision 9 above) -- no wire-format change,
both fields already independently `omitempty`. Fixed `types_test.go`'s
`exampleTSBundle()`: its unresolved-dependency `Note` had been
copy-pasted from the Java example ("no local npm cache ... Article IX,
decision 0001", which is Java's mvn/gradle-specific decision, and this
feature never checks an npm cache) -- replaced with "no package-lock.json
entry found for this package". Updated all `Dependencies`-field
literals in `types_test.go` (`TestRoundTrip_Bundle`, `minimalBundle`,
`exampleJavaBundle`, `exampleTSBundle`) from value to pointer literals,
and `TestSchemaVersion`'s expected value. Regenerated both golden
fixtures (`testdata/example_java.json`/`example_ts.json`) via
`go test ./internal/contract/... -run TestGolden -update` and reviewed
both byte-for-byte. Updated `001-data-contract/spec.md` FR13 (documented
the new pointer/optional behavior and the `Note` relaxation, mirroring
how FR12 already documents `GitMetadata`'s prior pointer change) and
`plan.md` (added the same optional-at-bundle-level comment to the
`dependencies` block in the Data model, next to `gitMetadata`'s existing
one). Also caught and fixed a pre-existing staleness gap while there:
`plan.md`'s "API / contracts" section still listed `contract.SchemaVersion`
as the exported constant `"1.0.0"`, never updated when 004 bumped it to
`"2.0.0"` -- now says "currently `3.0.0`" with a pointer to the bump
history instead of hardcoding a value that will go stale again.
**Verification:** `go build ./...`, `go test ./internal/contract/...`
(`ok`), `golangci-lint run ./internal/contract/...` (0 issues), and
`gofumpt -l` on all four changed files (`types.go`, `types_test.go`,
`fingerprint.go`, `rawinput.go`) -- all clean, confirmed by Vedant.
**Deviations from plan (if any):** None.
**New open questions:** None. T001 is next.

---

**Date:** 2026-09-16
**Task(s):** T001 -- `codecontext.FindRepoRoot`
**What happened:** Added `FindRepoRoot(ctx, workDir) (root string, ok bool)`
to `internal/codecontext/gitmeta.go`, following the same public-wraps-
injectable split `BuildGitMetadata`/`buildGitMetadata` already use
(rather than plan.md's illustrative inline-`execGitRunner{}` snippet),
so it's testable via the existing `fakeGitRunner` pattern with no real
git binary needed. `findRepoRoot(ctx, workDir, runner)` runs `git
rev-parse --show-toplevel`; any error (not-a-repo or a timeout, both
indistinguishable, same reasoning `isInsideGitWorkTree` already uses)
collapses to `ok=false` rather than a separate branch. Left
`isInsideGitWorkTree`/`BuildGitMetadata`'s own detection call
untouched -- `FindRepoRoot` is a second, independent detection call for
006b's new need (an actual root path), not a replacement.
Added `TestFindRepoRoot_Success`, `_NotARepo`, `_Timeout` to
`gitmeta_test.go`, same fake-runner shape as the existing
`TestBuildGitMetadata_NoRepoFound`/`_DetectionTimeout` tests.
Deliberately left `internal/codecontext`'s package doc comment
unchanged (still scoped to "own-code context") -- plan.md's Risks
section already flags this as a known, non-blocking scope mismatch;
not revisited here.
**Verification:** `go build ./...`, `go test ./internal/codecontext/...`
(`ok`), `golangci-lint run ./internal/codecontext/...` (0 issues), and
`gofumpt -l` on both changed files -- all clean, confirmed by Vedant.
**Deviations from plan (if any):** None (implementation detail only --
injectable-runner split vs. plan.md's inline snippet -- required by
tasks.md's own testability requirement for this task).
**New open questions:** None. T002 is next.

---

**Date:** 2026-09-16
**Task(s):** T002 -- npm manifest parsing (`Direct`)
**What happened:** Created the new `internal/dependency/typescript`
package (parallel to `internal/parser/typescript`'s layout, per
plan.md). Added `manifest.go`: unexported `npmManifest` struct scoped
to only `dependencies`/`devDependencies`/`optionalDependencies` --
`peerDependencies` has no field at all, not just an unused one, so it
can never leak into `Direct` even via a careless future "range over
every field" edit. `readManifest(path) (direct map[string]string, ok
bool)` reads+parses `package.json`, merging all three sections into one
map (no override-precedence logic needed -- npm itself doesn't allow a
package in more than one section); missing file and invalid JSON both
collapse to `ok=false` identically, matching spec.md FR7's "both outcomes
are Bundle.Dependencies nil" treatment.

The package doc comment (`// Package typescript ...`) currently lives on
`manifest.go`, not `resolve.go` -- `resolve.go`, the package's actual
orchestration entry point per plan.md, doesn't exist until T004. Flagged
in manifest.go's own comment; will move it once T004 lands, per
CONVENTIONS.md's "one file carries the doc comment, usually the file
most central to the package" rule.

Note: `readManifest` returns the UNSCOPED map (every package.json-declared
package) -- filtering to only trace-referenced packages (spec.md FR9) is
explicitly T004's job (`buildDirect` in resolve.go's pseudocode), not
this function's.

Added `manifest_test.go`: all-three-sections-merged, peerDependencies-
excluded, malformed JSON, missing file -- table-style via a shared
`writeManifest` helper (`t.TempDir()` + `os.WriteFile`, same pattern as
`codecontext/snippet_test.go`'s `writeLines`).
**Verification:** `go build ./...`, `go test
./internal/dependency/typescript/...` (`ok`), `golangci-lint run
./internal/dependency/typescript/...` (0 issues), and `gofumpt -l` on
both new files -- all clean, confirmed by Vedant.
**Deviations from plan (if any):** None.
**New open questions:** None. T003a is next.

---

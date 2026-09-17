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

**Date:** 2026-09-16
**Task(s):** T003a -- npm lockfile parsing
**What happened:** Added `lockfile.go`: unexported `npmLockfile`
(`LockfileVersion`, `Packages`) and `npmLockfilePackage` (`Version`
only) structs, matching plan.md's Data model exactly. `readLockfile(path)
(packages map[string]string, ok bool, reason string)` reads+parses
`package-lock.json`, flattening `Packages` down to a plain `dirKey ->
version` map -- the per-entry struct wrapper is discarded here since
lookup.go/resolve.go (T003b/T004) only ever need the version string.
Three distinct `ok=false` cases, each with its own `reason` string for
a later `LockedDependency.Note`: file not found (also covers the
"only yarn.lock exists" case, since this function only ever looks for
`package-lock.json` by exact path and never inspects `yarn.lock` at
all), malformed JSON, and `lockfileVersion` present but not 2 or 3
(covers npm <=6's lockfileVersion 1 nested-tree format).

Added `lockfile_test.go`: valid v2, valid v3 (including a scoped-
package key), malformed JSON, missing file, `lockfileVersion: 1`, and
yarn-only (confirms `yarn.lock`'s mere presence changes nothing) --
same `t.TempDir()`-based table style as `manifest_test.go`, via a
shared `writeLockfile` helper.
**Verification:** `go build ./...`, `go test
./internal/dependency/typescript/...` (`ok`), `golangci-lint run
./internal/dependency/typescript/...` (0 issues), and `gofumpt -l` on
both new files -- all clean, confirmed by Vedant.
**Deviations from plan (if any):** None.
**New open questions:** None. T003b is next.

---

**Date:** 2026-09-16
**Task(s):** T003b -- `packageDirKey` + exact/fallback lookup
**What happened:** Before the task itself, resolved an open decision
plan.md's Risks section flagged as due now, not deferrable: extracted
a new exported `FindLastNodeModulesSegment(path) (segments []string,
lastIdx int)` out of `internal/parser/typescript/bucket.go`'s
`splitAfterLastNodeModules` (006a), so the normalize-backslashes +
find-last-"node_modules"-occurrence logic lives in exactly one place
rather than being duplicated a second time by this task. Refactored
`splitAfterLastNodeModules` to call it; behavior unchanged (all
existing `TestAssignBucket` cases still pass unmodified), added
`TestFindLastNodeModulesSegment` directly against the new export.
This touches a file from an already-completed feature (006a) --
flagged to Vedant before implementing, confirmed.

Added `internal/dependency/typescript/lookup.go`: `packageDirKey(filePath,
repoRoot) (key, ok)` computes the lockfile key relative to repoRoot,
preserving the FULL nested `node_modules/.../node_modules/...` chain
(unlike bucket.go's bare-trailing-name want), reusing the shared helper
above. `lookupFrame(filePath, packageName, repoRoot, packages)
(version, note, matched)` resolves one frame's own contribution only:
exact `packageDirKey` hit -> version, no note; else top-level
`node_modules/<packageName>` fallback (tried regardless of why the
exact match missed) -> version + a note flagging the inexact match;
else `matched=false` -- deliberately not a final per-package outcome,
since aggregating across a package's frames and applying the
multi-frame conflict rule (spec.md FR13) needs every referenced frame
at once, which is T004's job.

Added `lookup_test.go`: `packageDirKey` (top-level, scoped, nested-
duplicate preserving the full chain, no-node_modules-segment) and
`lookupFrame` (exact match, scoped exact match, nested-duplicate-
version proving the nested copy's version wins over a same-named
top-level copy, top-level fallback, fully unresolved). Not tested:
`packageDirKey`'s `filepath.Rel`-error branch (outside-repoRoot case)
-- flagged to Vedant as realistically unreachable on Linux between two
absolute paths (Rel produces a "../"-prefixed result rather than
erroring), kept as a defensive check only.
**Verification:** `go build ./...`, `go test
./internal/parser/typescript/... ./internal/dependency/typescript/...`
(both `ok`), `golangci-lint run` on both packages (0 issues), and
`gofumpt -l` on all four touched/added files -- all clean, confirmed
by Vedant.
**Deviations from plan (if any):** Scope grew beyond this task's own
files to include the `bucket.go` refactor above -- flagged and
confirmed before implementing, not a silent expansion.
**New open questions:** None. T004 is next.

---

**Date:** 2026-09-16
**Task(s):** T004 -- `ResolveDependencies` orchestration
**What happened:** Flagged and resolved a testability gap before
writing the task's own code: plan.md's pseudocode has `ResolveDependencies`
call `codecontext.FindRepoRoot` directly, which would force
`resolve_test.go`'s root-resolution cases through a real git repo or
the git binary, violating CONVENTIONS.md's "anything that shells out
must be tested behind a fake" rule. Split it the same way
`BuildGitMetadata`/`FindRepoRoot` are already split: public
`ResolveDependencies` wraps a private `resolveDependencies` taking
`findRepoRoot` as an injected function parameter, defaulting to
`codecontext.FindRepoRoot` in production; tests pass stub functions
instead.

Added `resolve.go`: `referencedPackages` (every dependency-bucket
frame's `PackageName`, keyed by name, every referencing frame's
`FilePath` preserved -- not deduplicated); `buildDirect` (scopes
manifest to referenced names, spec.md FR9); `buildLocked` (reads
`package-lock.json` once, reuses its failure reason across every
affected package rather than re-deriving it per package);
`resolvePackage`/`unresolvedPackage`/`conflictingPackage`/`agreedPackage`
implementing spec.md FR10-FR14's per-package decision after collecting
every referencing frame's own `lookupFrame` result. `slog.Warn`/`Info`
calls per plan.md's Logging section (manifest missing/invalid, lockfile
unusable, per-package unresolved/conflict/fallback-only, final resolved
count). Moved the package doc comment off `manifest.go` onto this file,
per the placeholder flagged during T002.

One judgment call flagged to Vedant and confirmed before implementing:
spec.md doesn't say what happens when multiple frames agree on the same
version via a MIX of exact and fallback matches. Resolved as: any
contributing exact match confirms the package (no note) regardless of
an agreeing fallback-only frame elsewhere; only an all-fallback
agreement carries a note (reusing one representative frame's own note
text, chosen deterministically by frame order). The multi-version
conflict rule (FR13) itself needed no such call -- spec.md is explicit
there.

Added `resolve_test.go`: exact match, scoped exact match, top-level
fallback, conflict (two frames, exact-vs-exact to different nested
copies), one-resolves-one-doesn't (via `resolvePackage` directly, to
genuinely isolate a true non-match from a same-package fallback),
fully unresolved, lockfile missing/malformed/`lockfileVersion: 1`/
yarn-only, manifest missing/malformed, all three root-resolution paths
(`workDir` fallback success, `workDir` fallback failure with an
explicit no-panic assertion, git-succeeds-but-no-manifest with no
`workDir` retry), `Direct` scoping across dependencies/dev/optional
(in) vs. peer (out) vs. transitive (Locked-only), and declared-but-
unreferenced exclusion.
**Verification:** `go build ./...`, `go test
./internal/dependency/typescript/...` (`ok`), `golangci-lint run
./internal/dependency/typescript/...` (0 issues), and `gofumpt -l` on
all three touched files -- all clean, confirmed by Vedant.
**Deviations from plan (if any):** The `findRepoRoot`-injection split
(flagged and confirmed before implementing, required for testability,
no behavior change in production since `ResolveDependencies` still
defaults to the real `codecontext.FindRepoRoot`).
**New open questions:** None. T005 (feature close-out) is next.

---

**Date:** 2026-09-17
**Task(s):** T005 -- Feature close-out
**What happened:** Cross-checked all 15 functional requirements and all
20 acceptance criteria in `spec.md` against actual code/tests before
checking anything off. Found and fixed a real bug during that pass, not
after: `resolve_test.go`'s `TestResolveDependencies_OneResolvesOneDoesNot_NoConflictNoBlanking`
(written during T004) reused `validLockfileV2`, which happens to include
a top-level `node_modules/lodash` entry -- so its "non-matching" second
frame actually matched via the top-level fallback too (same package
name, fallback is keyed by name not path), meaning the test accidentally
exercised the exact-overrides-agreeing-fallback judgment call instead of
a genuine non-match. Fixed by rewriting it against a lockfile with only
a nested entry and no top-level fallback target, so the second frame is
a true non-match; split the judgment-call coverage the original test
had accidentally provided out into its own explicit test,
`TestResolveDependencies_ExactConfirmsDespiteAgreeingFallbackFrame`.
Vedant re-ran verification after this fix; still clean.

Checked off all 20 acceptance criteria in `spec.md`, each with a pointer
to its satisfying test(s) -- same style as `001-data-contract/spec.md`'s
own close-out pass. Updated `spec.md`'s Status line from "Spec'd" to
"Implemented." Flagged (not silently skipped) that FR14 -- a package
reaching `Direct`/`Locked` scope with no frame tying it to a path at
all -- is vacuously satisfied by construction: `referencedPackages` can
never produce such an entry, since it only ever adds a package when a
frame is encountered, so there's no independent test for it.

Updated `specs/INDEX.md`: 006b's status `in-progress` -> `done`.

Updated `CONVENTIONS.md`'s File/folder layout diagram to add
`internal/dependency/typescript/` (previously unlisted), with a note
that `internal/dependency/java/` is expected once 005b lands, and why
`dependency/` is a sibling to `parser/` rather than nested under it
(006b never shells out, Article IX; 005b will) -- same practice 004
already established for updating `001-data-contract`'s docs in place
for a structural change.
**Verification:** Documentation-only task; no code changed beyond the
test fix above (already independently verified by Vedant). `spec.md`,
`specs/INDEX.md`, and `CONVENTIONS.md` all reviewed for internal
consistency against the actual shipped code.
**Deviations from plan (if any):** None (the test fix was a bug found
during the close-out cross-check itself, not a plan deviation).
**New open questions:** None. 006b is complete; 005b (Java dependency
resolution) and 002b (pipeline wiring) are the next specs/INDEX.md
entries with 006b as a listed dependency.

---

**Date:** 2026-09-17
**Task(s):** Post-close-out fix -- `lookupFrame` empty-version handling
**What happened:** GitHub Copilot's automated PR review caught a real
gap `lookup_test.go`'s existing cases didn't exercise: a
`package-lock.json` `packages` entry can legitimately have an empty/
missing `version` (npm's own shape for a workspace `"link": true`
entry, which points at a local sibling package rather than an
installed copy). `lookupFrame` treated any map hit as a match
regardless of the version string's content, so an empty-version exact
key would have produced `LockedDependency{Version: "", Note: ""}` --
indistinguishable from a silently resolved dependency, exactly the
ambiguity FR12/Article VI exist to prevent.

Fixed both branches of `lookupFrame` (exact match and top-level
fallback) to require a non-empty version string before reporting a
match; an empty-version exact hit now falls through to the fallback
attempt rather than short-circuiting it, same as a key that's missing
outright. Added `TestLookupFrame_EmptyVersionExactKey_TreatedAsNoMatch`
and `TestLookupFrame_EmptyVersionExactKey_FallsThroughToRealFallback`.
Not a spec.md change -- none of the 20 acceptance criteria mention this
edge case specifically; it's a defensive correctness fix within FR12's
existing "no match -> explained absence" framing, not new scope.
**Verification:** `go build ./...`, `go test
./internal/dependency/typescript/...` (`ok`), `golangci-lint run
./internal/dependency/typescript/...` (0 issues), and `gofumpt -l` on
both changed files -- all clean, confirmed by Vedant.
**Deviations from plan (if any):** None -- bug fix, not a plan change.
**New open questions:** None.

---

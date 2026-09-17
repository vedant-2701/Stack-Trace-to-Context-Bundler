# Spec: TS/JS dependency resolution

**Status:** Implemented (`internal/codecontext`, `internal/dependency/typescript`,
T000-T004 all complete). All 15 functional requirements and all 20
acceptance criteria below are satisfied -- see each criterion's pointer
to its test(s); no deferred criteria for this feature.
**Folder:** specs/006b-ts-js-dependency-resolution
**Depends on:** 001-data-contract (done — this feature requires two
amendments to it, see "Contract changes required" below), 006a-ts-js-parser (done)

## Overview

Resolves concrete installed versions for `dependency`-bucket frames in a
parsed TS/JS trace, by reading `package.json` and `package-lock.json`
directly at the repo root — never shelling out to `npm`/`yarn`/`pnpm`
(constitution Article IX's own text already carves this out from the
offline-first/time-boxed subprocess pattern used for Java). This is what
lets an AI assistant see "you're on lodash 4.17.21" instead of a bare
`node_modules/lodash/lodash.js` path it has to go check itself.

This feature does not do frame bucketing (006a already assigns `Bucket`/
`PackageName`), pipeline wiring (002b), or Java's `mvn`/`gradle` shell-out
(005b) — it only resolves versions for TS/JS `dependency`-bucket frames
already present in a parsed `Chain`.

## User stories

- As a developer debugging a crash inside a third-party package, I want
  the bundle to tell the AI assistant which exact version of that package
  is installed, so it can reason about a version-specific bug instead of
  guessing which release I'm on.
- As a developer whose lockfile is missing, stale, or in an unsupported
  format, I want the bundle to say so explicitly per package rather than
  silently omit the section or, worse, report a version that isn't
  actually installed.

## Functional requirements

### Scope

1. v1 supports **npm only** — `package-lock.json`, lockfileVersion 2 or 3
   (npm ≥ 7's flat `packages` map, keyed by node_modules-relative path).
   lockfileVersion 1 (npm ≤ 6, nested `dependencies` tree) is a different
   parser entirely and is out of scope for v1 (see Out of scope).
2. `yarn.lock` and `pnpm-lock.yaml` are out of scope for v1.
   `pnpm-lock.yaml` specifically requires a YAML parser, and this repo
   has no YAML dependency (`go.mod` has exactly one third-party
   dependency, `pflag`, per decision 0002) — adding one is a decision
   for a future feature, not silently pulled in here (AGENTS.md
   Boundaries: no new third-party dependency without asking first).
3. No subprocess calls to `npm`/`yarn`/`pnpm` at all, for any purpose —
   confirmed already-decided by constitution Article IX's own text
   ("TypeScript/JavaScript dependency resolution is unaffected — it
   parses `package.json` and the lockfile directly and never shells out
   to a subprocess"). The only subprocess call this feature makes is
   `git rev-parse --show-toplevel` (requirement 4), for repo-root
   discovery, not dependency resolution itself.

### Repo-root & manifest location

4. Repo root is found via `git rev-parse --show-toplevel`, reusing
   `internal/codecontext`'s existing `gitRunner`/`gitTimeout` (10s)
   machinery (Article V: shared logic lives once — no second
   git-subprocess implementation). No existing feature computes a
   repo-root path today (004 deliberately dropped `--show-toplevel`,
   see `codecontext.isInsideGitWorkTree`'s own doc comment) — this is
   the first consumer.
5. `package.json` and `package-lock.json` are looked for at exactly that
   repo root — no walking up further, no monorepo/workspace support
   (constitution's existing "Multi-manifest / multi-module dependency
   resolution" out-of-scope entry already covers this).
6. If repo-root detection fails (not inside a git working tree, or the
   `rev-parse` call times out — both treated the same, no separate
   handling for a timeout vs. a confirmed non-repo), the system falls
   back to treating `workDir` (the directory the binary was invoked
   from) as the root directly — no upward walk to a parent directory.
   This assumes the CLI is invoked from the project root when there's
   no git repository to confirm it, matching how most JS tooling
   (npm, tsc, jest) already defaults to cwd absent other config. This
   fallback applies **only** when git detection itself fails — if git
   succeeds but the reported repo root simply has no `package.json`
   there, that stays "no manifest found" (requirement 7); it does not
   additionally retry against `workDir` (that would quietly reopen the
   already-out-of-scope multi-manifest/monorepo case).

### Manifest (`package.json`) handling

7. If `package.json` cannot be found at the resolved root (the
   git-detected repo root, or the `workDir` fallback from requirement
   6 when git detection fails), **or exists but is not valid JSON**,
   `Bundle.Dependencies` is nil (omitted from the JSON entirely) — see
   "Contract changes required" below. This is a valid, representable
   outcome, not an error and never a panic: mirrors
   `codecontext.BuildGitMetadata`'s "never returns an error, nil is a
   real outcome" pattern, and applies uniformly whether the search
   used the git root or the `workDir` fallback — there is no third,
   harder failure mode when both come up empty.
8. `Dependencies.Direct` is populated from the `dependencies`,
   `devDependencies`, and `optionalDependencies` sections of
   `package.json` (declared version range, verbatim as written).
   `peerDependencies` is excluded — it declares a range without itself
   causing an install.
9. `Dependencies.Direct`/`Dependencies.Locked` are scoped to **only the
   packages that actually appear as a `dependency`-bucket frame's
   `PackageName` somewhere in this trace's parsed `Chain`** — not every
   package.json-declared dependency (which could be 50+, mostly
   irrelevant to this specific bug). This mirrors `CodeContexts`'
   existing "only for own-bucket frames" scoping precedent. A package
   that's a referenced dependency-bucket frame but isn't declared in any
   of requirement 8's three sections (a purely transitive dependency)
   simply doesn't appear in `Direct` — no error, no note; `Direct` has
   no per-entry note field, and the absence itself is the correct
   signal. `Locked` still attempts resolution for it regardless (its
   presence in `package.json` is irrelevant to whether it's actually
   installed and lockfile-resolvable).

### Lockfile (`package-lock.json`) resolution

10. For each referenced package name (requirement 9's scope), resolution
    is attempted **per frame carrying that `PackageName`**, using the
    frame's own `Frame.FilePath`:
    a. Compute `FilePath` relative to repo root (backslash-normalized
       first, matching `internal/parser/typescript/bucket.go`'s existing
       normalization — Windows paths use `\`).
    b. Truncate that relative path to the **package directory key**: the
       lockfile's `packages` map keys are directory paths, not file
       paths — e.g. a frame file at
       `node_modules/foo/node_modules/lodash/lodash.js` truncates to
       `node_modules/foo/node_modules/lodash`. For a scoped package, the
       key includes the scope segment (e.g.
       `node_modules/@babel/core`). This preserves the FULL nested
       `node_modules/.../node_modules/...` prefix chain — unlike
       006a's `PackageName` (bare display name only, from the segment
       after the LAST `node_modules/`), this key must exactly match
       the lockfile's own path-based keying to find the specific
       nested copy that frame is actually running.
    c. Look up that exact key in `package-lock.json`'s `packages` map.
       If found: `LockedDependency.Version` = that entry's `version`
       field. No `Note` — this is an exact, verified match.
11. **Evaluated per frame, same as requirement 10** — not gated on
    whether some *other* frame of the same `PackageName` already found
    an exact match. If a given frame's own exact path-derived key
    (requirement 10) does not match (stale lockfile vs. actual
    `node_modules`, `npm link`, or a trace captured on a different
    machine/environment — `memory/known-gaps.md`'s existing
    cross-machine caveat), that frame falls back, **for its own
    contribution only**, to a **top-level bare-name lookup**: the key
    `node_modules/<PackageName>` (no nested prefix). If that exists,
    this frame's contribution to `LockedDependency.Version` is that
    entry's value, **and** its contribution to `LockedDependency.Note`
    explains this was an inexact/fallback match (not tied to the
    frame's exact nested path) — see "Contract changes required" below
    for why `Version` and `Note` can now coexist. Requirement 13
    covers what happens when different frames' contributions — whether
    from requirement 10 or this fallback — disagree with each other.
12. If **every** referenced frame of a package has neither an exact
    (requirement 10) nor a top-level fallback (requirement 11) match —
    or `package-lock.json` itself is missing, is present in an
    unsupported format only (e.g. only `yarn.lock`/`pnpm-lock.yaml`
    exist, no `package-lock.json`), has a lockfileVersion other than 2
    or 3, or is malformed JSON — `LockedDependency.Version` is omitted
    and `Note` explains why (satisfies the deferred acceptance
    criterion below). A single frame's own unresolved outcome does
    **not** override a *different* frame of the same package that did
    resolve — this rule only fires when nothing resolved anywhere for
    the package. Separately, this is a **per-package** outcome across
    different packages too: one package's lockfile problem never
    blocks resolving a different package that does resolve cleanly.
13. **Multi-frame conflict.** If two or more frames referencing the same
    `PackageName` resolve — via any combination of an exact match
    (requirement 10) and/or a top-level fallback match (requirement 11)
    — to different `version` values, this is a genuine conflict, not an
    ambiguous single case: `LockedDependency.Version` is omitted and
    `Note` names each distinct version found (and, where useful, which
    frame each came from), rather than one match silently overriding
    the other. This takes priority over both requirement 10's and
    requirement 11's ordinary per-match assignment for that package — a
    resolvable-but-disagreeing set of frames is a materially different,
    less certain outcome than either a single clean match or a total
    non-match, and constitution Article VI's "never guess silently"
    applies as much to picking between two real matches as it does to
    inventing one from nothing. This is expected to occur in practice —
    nested duplicate copies of the same package at different versions
    are the normal reason npm's nested `node_modules` layout exists —
    not a rare edge case being flagged for completeness only.
14. A package with no frame tying it to a specific path at all (i.e. it
    only reached `Direct`/`Locked` scope some other way — not expected
    given requirement 9's scoping, but kept as an explicit fallback
    order) resolves via the top-level bare-name lookup directly
    (requirement 11), same as the "no exact match" fallback path.

### Deferred acceptance criterion (folded in from `memory/known-gaps.md`)

15. **Satisfies the `001-data-contract`/`memory/known-gaps.md` deferred
    criterion** ("Package with no locally resolvable version →
    `dependencies.locked[pkg].version` omitted, `.note` explains why"):
    requirement 12 above is this criterion's end-to-end implementation
    for the TS/JS half (005b owns the Java half separately).

## Contract changes required (blocks all other work in this feature)

Two amendments to `internal/contract` (from `001-data-contract`, status
`done`) — same category of change as 004's `GitMetadata` pointer patch,
which is the established precedent for touching a `done` feature's files
mid-project:

1. **`Bundle.Dependencies` becomes `*Dependencies`**
   (`json:"dependencies,omitempty"`). Today it's a non-pointer struct
   with no "not applicable" representation — the same ambiguity
   `GitMetadata` had before 004's patch. Required for requirement 7 (no
   manifest found → omitted entirely, not zero-valued fields that could
   be misread as "manifest is `package.json` with zero dependencies").
   `contract.SchemaVersion` bumps `"2.0.0"` → `"3.0.0"` (MAJOR — a
   field changing from always-required to sometimes-absent, per
   `001-data-contract`'s own bump-trigger rule).
2. **`LockedDependency`'s doc comment is relaxed**: "Note is present
   only when Version is absent" → Note may also accompany a present
   Version, for the inexact-fallback-match case (requirement 11). No
   wire-format change — both fields are already independently
   `omitempty`, so this is a documented-invariant correction, not a
   schema change; no additional version bump beyond item 1's.

Both `001-data-contract`'s golden fixtures
(`testdata/example_java.json`/`example_ts.json`) must be regenerated
after the struct change, and `001`'s own `spec.md`/`plan.md` updated in
place to reflect the new shape — per the practice 004 already
established. Scoped as this feature's first task in `tasks.md`.

## Non-functional requirements

- No new third-party Go dependencies (requirement 2) — `encoding/json`
  (stdlib) only for both `package.json` and `package-lock.json` parsing.
- The one subprocess call this feature makes (`git rev-parse
  --show-toplevel`) reuses the existing 10-second `gitTimeout` — no new
  timeout value invented for it.
- No network access of any kind (requirement 3) — this feature is pure
  local file reads plus one bounded git subprocess call.

## Out of scope

- `yarn.lock`, `pnpm-lock.yaml` — any lockfile format other than npm's
  (requirement 2).
- npm lockfileVersion 1 (npm ≤ 6) — a structurally different, nested
  format (requirement 1).
- Multi-manifest / monorepo / npm workspaces — single manifest at repo
  root only (requirement 5, constitution's existing out-of-scope entry).
- Falling back to the process's current working directory **when git
  detection succeeds** but the reported root lacks a `package.json`
  (requirement 6) — the `workDir` fallback applies only when git
  detection itself fails, never as a second attempt after a
  successful-but-manifest-less git root.
- Walking up from `workDir` to a parent directory looking for
  `package.json` (requirement 6) — only `workDir` itself is checked in
  the fallback case, no ancestor search.
- Populating `Direct`/`Locked` for every package.json-declared
  dependency regardless of trace relevance (requirement 9) — scoped to
  referenced packages only.
- Pipeline wiring / real stdout bundle output (002b).
- Java dependency resolution (005b) — separate feature, separate
  mechanism (`mvn`/`gradle` shell-out, offline-first/time-boxed per
  Article IX), not touched here beyond sharing the same deferred
  acceptance criterion (requirement 15).

## Acceptance criteria

- [x] Given a trace with a `dependency`-bucket frame whose full path
      exactly matches a `packages` entry in a valid lockfileVersion
      2/3 `package-lock.json`, when resolved, then `Locked[pkg].Version`
      is set from that entry with no `Note`. Satisfied by `lookup.go`'s
      `packageDirKey`/`lookupFrame` and `resolve.go`'s `agreedPackage`;
      see `TestPackageDirKey`, `TestLookupFrame_ExactMatch`, and
      end-to-end `TestResolveDependencies_ExactMatch`.
- [x] Given a `dependency`-bucket frame under a scoped package (e.g.
      `@babel/core`) with an exact lockfile path match, when resolved,
      then the key used (both in `Locked` and internally) is
      `"@babel/core"`, matching `Frame.PackageName`'s existing format.
      Satisfied by `packageDirKey`'s scoped-package branch; see
      `TestPackageDirKey`'s scoped case,
      `TestLookupFrame_ScopedPackageExactMatch`, and end-to-end
      `TestResolveDependencies_ScopedPackageExactMatch`.
- [x] Given a `dependency`-bucket frame's exact nested path has no
      lockfile match, but a top-level `node_modules/<pkg>` entry exists,
      when resolved, then `Locked[pkg].Version` is set from the
      top-level entry **and** `Note` explains it's an inexact/fallback
      match. Satisfied by `lookupFrame`'s fallback branch; see
      `TestLookupFrame_TopLevelFallback` and end-to-end
      `TestResolveDependencies_TopLevelFallback`.
- [x] Given two `dependency`-bucket frames sharing the same
      `PackageName` whose resolutions disagree on installed version
      (whether both are exact matches to different nested copies, or
      one exact and one top-level-fallback), when resolved, then
      `Locked[pkg].Version` is omitted and `Note` names the conflicting
      versions -- neither frame's match silently wins (requirement 13).
      Satisfied by `resolve.go`'s `conflictingPackage`; see
      `TestResolveDependencies_Conflict` (exact-vs-exact case).
- [x] Given two `dependency`-bucket frames sharing the same
      `PackageName` where one resolves (exact or fallback) and the
      *other* matches neither exactly nor via fallback, when resolved,
      then `Locked[pkg].Version` is set from the one frame that did
      resolve -- the other frame's own non-match neither triggers the
      conflict rule (requirement 13) nor blanks out the successful
      resolution (requirement 12). Satisfied by `resolvePackage`/
      `agreedPackage`; see
      `TestResolveDependencies_OneResolvesOneDoesNot_NoConflictNoBlanking`.
      A related, spec.md-unstated case -- one exact match and one
      *agreeing* fallback-only match for the same package -- is
      resolved as "confirmed, no note" (a judgment call, see
      `agreedPackage`'s doc comment and progress.md's T004 entry);
      see `TestResolveDependencies_ExactConfirmsDespiteAgreeingFallbackFrame`.
- [x] Given a package with neither an exact nor top-level match in a
      valid, present lockfile, when resolved, then `Locked[pkg].Version`
      is omitted and `Note` explains no match was found. Satisfied by
      `resolve.go`'s `unresolvedPackage`; see `TestLookupFrame_Unresolved`
      and end-to-end `TestResolveDependencies_FullyUnresolved`.
- [x] Given `package-lock.json` is entirely absent (but `package.json`
      exists and parses), when resolved, then every referenced
      package's `Locked` entry has `Version` omitted and a `Note`
      explaining no lockfile was found. Satisfied by `lockfile.go`'s
      `readLockfile` (missing-file branch); see
      `TestReadLockfile_MissingFile` and end-to-end
      `TestResolveDependencies_LockfileMissing`.
- [x] Given only `yarn.lock` (no `package-lock.json`) is present, when
      resolved, then this is treated identically to "no lockfile found"
      (previous criterion) -- not a crash, not a silent skip. Satisfied
      by `readLockfile` (only ever looks for `package-lock.json` by
      exact path); see `TestReadLockfile_YarnOnly_TreatedAsMissing` and
      end-to-end `TestResolveDependencies_YarnOnly_TreatedAsMissing`.
- [x] Given `package-lock.json` exists but is not valid JSON, when
      resolved, then every referenced package's `Locked` entry has
      `Version` omitted and a `Note` explaining the parse failure.
      Satisfied by `readLockfile`'s JSON-unmarshal-error branch; see
      `TestReadLockfile_MalformedJSON` and end-to-end
      `TestResolveDependencies_LockfileMalformed`.
- [x] Given `package-lock.json`'s `lockfileVersion` is `1`, when
      resolved, then every referenced package's `Locked` entry has
      `Version` omitted and a `Note` explaining the unsupported format
      -- not a crash, not an attempt to parse it as v2/v3. Satisfied by
      `readLockfile`'s lockfileVersion check; see
      `TestReadLockfile_LockfileVersion1` and end-to-end
      `TestResolveDependencies_LockfileVersion1`.
- [x] Given `package.json` cannot be found at repo root, when the
      bundle is built, then `Bundle.Dependencies` is nil and omitted
      from the JSON entirely. Satisfied by `resolve.go`'s
      `resolveDependencies` (manifest-not-found branch) plus T000's
      `*Dependencies` pointer/`omitempty`; see
      `TestResolveDependencies_ManifestMissing`.
- [x] Given `package.json` exists but is not valid JSON, when the
      bundle is built, then `Bundle.Dependencies` is nil, same as the
      not-found case. Satisfied by `readManifest`'s JSON-unmarshal-error
      branch; see `TestReadManifest_MalformedJSON` and end-to-end
      `TestResolveDependencies_ManifestMalformed`.
- [x] Given the tool is run outside any git working tree (no `.git`
      anywhere) but `package.json` exists directly in `workDir`, when
      the bundle is built, then `Bundle.Dependencies` is populated using
      `workDir` as the root, same as if git had reported it. Satisfied
      by `resolveDependencies`'s `workDir` fallback; see
      `TestResolveDependencies_WorkDirFallback_ManifestPresent`.
- [x] Given the tool is run outside any git working tree and
      `package.json` does **not** exist in `workDir` either, when the
      bundle is built, then `Bundle.Dependencies` is nil -- not a panic,
      not a crash of the whole bundle. Satisfied by the same `workDir`
      fallback path; see
      `TestResolveDependencies_WorkDirFallback_ManifestAbsent_NoPanic`
      (explicit `recover()` assertion).
- [x] Given git detection succeeds but the reported repo root has no
      `package.json`, when the bundle is built, then `Bundle.Dependencies`
      is nil -- `workDir` is not additionally checked as a second
      attempt in this case. Satisfied by `resolveDependencies`'s
      single, non-retrying manifest lookup; see
      `TestResolveDependencies_GitSucceedsButNoManifest_NoWorkDirRetry`.
- [x] Given a package declared in `package.json`'s `dependencies` and
      referenced by a frame, when resolved, then it appears in both
      `Direct` (declared range) and `Locked` (resolved version or
      explained absence). Satisfied by `buildDirect`/`buildLocked`; see
      `TestResolveDependencies_DirectScoping`'s `lodash` case.
- [x] Given a package declared only in `devDependencies` or
      `optionalDependencies` and referenced by a frame, when resolved,
      then it appears in `Direct` the same as a `dependencies` entry.
      Satisfied by `manifest.go`'s three-section merge; see
      `TestReadManifest_AllThreeSectionsMerged` and end-to-end
      `TestResolveDependencies_DirectScoping`'s `jest`/`fsevents` cases.
- [x] Given a package declared only in `peerDependencies` and referenced
      by a frame, when resolved, then it does not appear in `Direct` --
      only in `Locked`, if lockfile-resolvable. Satisfied by
      `npmManifest` having no `peerDependencies` field at all; see
      `TestReadManifest_PeerDependenciesExcluded` and end-to-end
      `TestResolveDependencies_DirectScoping`'s `react` case.
- [x] Given a purely transitive package (not declared in `package.json`
      at all) referenced by a frame, when resolved, then it does not
      appear in `Direct` but does appear in `Locked` if the lockfile
      resolves it. Satisfied by `buildDirect`'s manifest-membership
      check plus `buildLocked` iterating `referenced` independently of
      `Direct`; see `TestResolveDependencies_DirectScoping`'s
      `left-pad` case.
- [x] Given a package declared in `package.json` but never appearing as
      a `dependency`-bucket frame in this trace, when the bundle is
      built, then it does not appear in `Direct` or `Locked` at all.
      Satisfied by `referencedPackages` only ever including frame-backed
      names; see
      `TestResolveDependencies_DeclaredButUnreferenced_ExcludedEntirely`.

Note on requirement 14 (a package reaching `Direct`/`Locked` scope with
no frame tying it to a path at all): not independently testable as a
separate scenario -- `referencedPackages` only ever creates a map entry
when a `dependency`-bucket frame is encountered, so a package with zero
associated frames cannot enter `Direct`/`Locked` scope by construction.
Requirement 14 is therefore vacuously satisfied, not silently skipped:
flagged here rather than presented as independently verified.

## Open questions

None remaining — all resolved during interrogation (see
`specs/006b-ts-js-dependency-resolution/progress.md` for the session
log).

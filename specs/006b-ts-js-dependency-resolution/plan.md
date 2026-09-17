# Plan: TS/JS dependency resolution

Derived from `spec.md`. Consistent with `memory/constitution.md` and
`CONVENTIONS.md`.

## Architecture / approach

Three-step pipeline, run once per bundle (only if `Bundle.Language` is
`javascript`/`typescript` — 002b's dispatch concern, not this package's):

1. **Repo-root discovery** (spec.md FR4/FR6) — new exported
   `codecontext.FindRepoRoot(ctx, workDir) (root string, ok bool)`,
   added to the existing `internal/codecontext` package rather than a
   new package, since it reuses that package's already-unexported
   `gitRunner`/`gitTimeout`/`execGitRunner` machinery (Article V — no
   second git-subprocess implementation). Runs `git rev-parse
   --show-toplevel`; `ok=false` on any git error or timeout, mirroring
   `BuildGitMetadata`'s existing "never error, false/nil is a real
   outcome" shape. `FindRepoRoot` itself stays a pure git-based lookup —
   it does NOT know about the `workDir` fallback (step 1a below); that
   fallback is 006b-specific (a dependency-resolution concern, not a
   general "find me a project root" one) and lives in
   `internal/dependency/typescript` instead, so `FindRepoRoot`'s
   semantics stay unchanged for `codecontext`'s other (git-metadata)
   use, where "no repo found" genuinely should mean no blame info
   available, not a guessed root.
   1a. **`workDir` fallback** (spec.md FR6) — when `FindRepoRoot`
       returns `ok=false` (no git repo, or the `rev-parse` call itself
       times out — both treated identically), `ResolveDependencies`
       checks for `package.json` directly in `workDir` before giving
       up. No upward walk to a parent directory. Only triggers when
       `FindRepoRoot` fails outright — if it succeeds but the reported
       root lacks `package.json`, that's requirement 7's "no manifest
       found" outcome with no further fallback attempt.
2. **Manifest + lockfile parsing** (spec.md FR7-FR12, FR14) — new
   package `internal/dependency/typescript/`, parallel to
   `internal/parser/typescript/`'s existing layout. Pure `encoding/json`
   parsing, no subprocess calls, no network (spec.md non-functional
   requirements).
3. **Resolution** — for each `dependency`-bucket frame's `PackageName`
   referenced in the `Chain` (spec.md FR9's scoping), attempt exact
   path-key match first, then top-level bare-name fallback, then give
   up with a `Note` (spec.md FR10-FR12, FR14). Before finalizing each
   package's entry, check whether its referenced frames disagreed on
   version across any combination of exact/fallback matches — if so,
   override that per-package result with the conflict outcome (spec.md
   FR13) instead of letting whichever frame resolved last silently win.

Top-level entry point mirrors `codecontext.BuildGitMetadata`'s shape —
never returns a Go error, and never panics, since every failure mode
(no repo AND no manifest in `workDir` either, malformed JSON, no
lockfile) is a valid, representable `nil` or per-package `Note` outcome,
not an exceptional one:

```go
func ResolveDependencies(ctx context.Context, workDir string, chain []contract.ExceptionNode) *contract.Dependencies
```

## Stack & versions

No new third-party dependencies (spec.md FR2, AGENTS.md Boundaries).
Standard library only: `encoding/json`, `path/filepath`, `strings`.

## Data model

Contract changes (spec.md "Contract changes required" — done as this
feature's first task, against `internal/contract/types.go`):

```go
// Bundle
Dependencies *Dependencies `json:"dependencies,omitempty"` // was: Dependencies (value)

// SchemaVersion
const SchemaVersion = "3.0.0" // was: "2.0.0"

// LockedDependency's doc comment updated:
// Note may be present whether or not Version is set -- either
// explaining why Version is absent, or, when Version IS set via an
// inexact/fallback match (006b's top-level-lookup case), flagging
// that the match isn't tied to the exact frame path.
```

Internal-only structs (not part of `internal/contract`, per Article IV —
these are 006b's own npm-format parsing shapes, never a second copy of
the bundle contract):

```go
// manifest.go
type npmManifest struct {
    Dependencies         map[string]string `json:"dependencies"`
    DevDependencies       map[string]string `json:"devDependencies"`
    OptionalDependencies  map[string]string `json:"optionalDependencies"`
}

// lockfile.go
type npmLockfile struct {
    LockfileVersion int                            `json:"lockfileVersion"`
    Packages        map[string]npmLockfilePackage  `json:"packages"`
}
type npmLockfilePackage struct {
    Version string `json:"version"`
}
```

## File / module layout

```
internal/codecontext/
└── gitmeta.go          # + FindRepoRoot(ctx, workDir) (string, bool),
                         #   reusing existing gitRunner/gitTimeout

internal/dependency/
└── typescript/
    ├── resolve.go        # ResolveDependencies -- orchestrates the
    │                      # 3-step pipeline, scopes Direct/Locked to
    │                      # referenced packages only (FR9), detects
    │                      # and resolves multi-frame version conflicts
    │                      # (FR13), logs degraded outcomes via slog
    │                      # (see "Logging" below)
    ├── manifest.go         # package.json parsing -> Direct map
    │                        # (dependencies+devDependencies+
    │                        # optionalDependencies, FR8)
    ├── lockfile.go          # package-lock.json parsing only --
    │                         # lockfileVersion + packages map (FR12)
    ├── lookup.go            # packageDirKey (FR10b) + per-frame
    │                         # exact/fallback lookup (FR10c, FR11).
    │                         # Returns one frame's own result only --
    │                         # cross-frame conflict detection (FR13)
    │                         # is resolve.go's job, not this file's,
    │                         # since it needs every referenced frame's
    │                         # result at once, not one at a time.
    ├── resolve_test.go
    ├── manifest_test.go
    ├── lockfile_test.go
    ├── lookup_test.go
    └── testdata/
        ├── valid-v3/                  # package.json + package-lock.json,
        │                              # exact-match + scoped-package +
        │                              # nested-duplicate-version cases
        ├── conflicting-versions/      # two frames, same PackageName,
        │                              # different resolved versions
        │                              # (FR13)
        ├── malformed-manifest/        # invalid package.json
        ├── malformed-lockfile/        # invalid package-lock.json
        ├── lockfile-v1/               # lockfileVersion: 1
        ├── no-lockfile/               # package.json only
        └── yarn-only/                 # package.json + yarn.lock, no
                                        # package-lock.json
```

## API / contracts

```go
// internal/codecontext/gitmeta.go
func FindRepoRoot(ctx context.Context, workDir string) (root string, ok bool) {
    out, err := execGitRunner{}.Run(ctx, workDir, "rev-parse", "--show-toplevel")
    if err != nil {
        return "", false
    }
    return strings.TrimSpace(out), true
}

// internal/dependency/typescript/resolve.go
func ResolveDependencies(ctx context.Context, workDir string, chain []contract.ExceptionNode) *contract.Dependencies {
    root, ok := codecontext.FindRepoRoot(ctx, workDir)
    if !ok {
        // No git repo (or rev-parse timed out) -- fall back to workDir
        // itself, no upward walk. Never a panic: absence of a manifest
        // here is exactly as valid an outcome as absence at a real git
        // root (FR7).
        root = workDir
    }
    manifest, ok := readManifest(filepath.Join(root, "package.json"))
    if !ok {
        return nil
    }
    referenced := referencedPackages(chain) // dependency-bucket frames only, FR9
    direct := buildDirect(manifest, referenced)
    // buildLocked resolves each referenced package via lookup.go's
    // per-frame matcher, then detects and overrides any package whose
    // frames disagree on version (FR13) before returning.
    locked := buildLocked(root, referenced, chain)
    return &contract.Dependencies{
        ManifestFile: contract.ManifestFilePackageJSON,
        Direct:       direct,
        Locked:       locked,
    }
}

// internal/dependency/typescript/lockfile.go
// packageDirKey computes the lockfile "packages" map key for a frame's
// absolute FilePath, relative to repoRoot -- preserves the FULL nested
// node_modules/.../node_modules/... prefix chain (unlike 006a's
// bucket.go, which only wants the bare trailing package name). Returns
// ok=false if FilePath isn't under repoRoot, or has no node_modules
// segment at all.
func packageDirKey(filePath, repoRoot string) (key string, ok bool) {
    rel, err := filepath.Rel(repoRoot, filePath)
    if err != nil {
        return "", false
    }
    segments := strings.Split(strings.ReplaceAll(rel, "\\", "/"), "/")
    lastIdx := -1
    for i, s := range segments {
        if s == "node_modules" {
            lastIdx = i
        }
    }
    if lastIdx == -1 || lastIdx+1 >= len(segments) {
        return "", false
    }
    end := lastIdx + 1
    if strings.HasPrefix(segments[end], "@") && end+1 < len(segments) {
        end++
    }
    return strings.Join(segments[:end+1], "/"), true
}
```

## Logging

Mirrors `codecontext.context.go`'s centralized `warnDegraded` pattern (a
single helper called at each degraded-but-continuing decision point,
rather than log calls scattered across `manifest.go`/`lockfile.go`/
`lookup.go`) — lives in `resolve.go` since it's the only place that sees
every outcome across all referenced packages:

- `slog.Warn` at each degraded-but-continuing outcome: manifest not
  found/malformed (FR7), lockfile not found/malformed/unsupported
  `lockfileVersion` (FR12), an inexact top-level-fallback match used
  (FR11), no match found at all for a package (FR12), and the
  multi-frame version conflict (FR13) — each carrying the same
  reasoning as the `Note` text it produces, per `CONVENTIONS.md`'s
  Logging section ("Warn — degraded but continuing — file not found,
  fell back to a note").
- `slog.Info` once per successful resolution — "N dependencies
  resolved", where N counts `Locked` entries that actually got a
  `Version` (exact or fallback), not every entry in the map (an
  unresolved/conflict entry has a map entry too, just with `Version`
  omitted, so counting `len(locked)` verbatim would overstate how much
  actually resolved) — matching `CONVENTIONS.md`'s own Info-level
  example in spirit.
- No `slog.Error` calls: this feature never fails the whole bundle, per
  FR7's "nil is a valid outcome, never a panic" framing; `Error` is
  reserved for outcomes that exit the process non-zero, which nothing
  here does.

## Testing strategy

- Table-driven tests per file (`manifest_test.go`, `lockfile_test.go`,
  `lookup_test.go`, `resolve_test.go`), matching `CONVENTIONS.md`'s
  testing section.
- Synthetic fixtures, not real-captured ones — unlike 006a's stack-trace
  formatting (which had genuine undocumented quirks only discoverable by
  real capture, e.g. the `[cause]:` bracket format), npm's
  `package-lock.json` shape is a stable, publicly documented format.
  Hand-built fixtures covering: exact path match, scoped package exact
  match, nested-duplicate-version (top-level + nested copies of the same
  package at different versions, proving `packageDirKey` picks the right
  one), top-level fallback (no exact match, bare-name match exists),
  fully unresolved (neither match), two-frame version conflict — same
  `PackageName`, two different resolved versions, via exact+exact and
  exact+fallback combinations (FR13), malformed manifest, malformed
  lockfile, missing lockfile, `yarn.lock`-only, `lockfileVersion: 1`.
- `codecontext.FindRepoRoot` tested via the existing `fakeGitRunner`
  pattern (`runner_fake_test.go`) — no real `git` binary required, per
  `CONVENTIONS.md`.
- `resolve_test.go` covers the `workDir` fallback explicitly: no git
  repo + `package.json` present in `workDir` (resolves normally), no
  git repo + no `package.json` in `workDir` either (nil, and
  specifically asserted to not panic), and git succeeds but the root
  has no manifest (nil, with `workDir` deliberately NOT retried).
- `resolve_test.go` also covers the multi-frame conflict override
  (FR13) explicitly: two referenced frames of the same `PackageName`
  whose individual `lookup.go` results disagree, asserting the
  package's final `Locked` entry is the conflict outcome regardless of
  which frame is processed first or last.
- Every acceptance criterion in `spec.md` maps to at least one test case.

## Risks & open decisions

- **`internal/codecontext`'s package doc comment** currently scopes it
  to "own-code context" specifically (004's own framing). Adding
  `FindRepoRoot` broadens its actual scope slightly (a repo-root lookup
  isn't really "own-code context") — flagged, not blocking: the
  alternative (a new package duplicating `gitRunner`) is a clearer
  Article V violation than a slightly-broadened doc comment. Revisit if
  a third consumer's need makes the mismatch more pronounced.
- **`packageDirKey`'s Windows-path handling** inherits the same
  unverified-but-low-risk status `bucket.go`'s `splitAfterLastNodeModules`
  already carries (`memory/known-gaps.md`) — no real Windows-generated
  fixture exists in this repo yet.
- **Grouping "not a git repo" and "`rev-parse` timed out" as one
  `FindRepoRoot` failure mode**, both triggering the same `workDir`
  fallback: a slow-but-real repo (huge monorepo, slow filesystem) that
  times out would fall back to `workDir` rather than its true, possibly
  different, root. Accepted as a low-probability edge case rather than
  adding a second failure classification — revisit if it proves
  common in practice.
- **`packageDirKey` duplicates `bucket.go`'s node-modules-segment-finding
  logic.** Both `lookup.go`'s `packageDirKey` and
  `internal/parser/typescript/bucket.go`'s `splitAfterLastNodeModules`
  independently normalize `\`->`/` and find the last `node_modules`
  segment, then diverge (bare name vs. full nested prefix). Given this
  logic's own Windows-path handling is already flagged above as
  unverified, a future fix landing in only one copy is a real risk.
  Open decision: export the shared "find last node_modules index"
  fragment from `internal/parser/typescript` (006b already depends on
  006a per `specs/INDEX.md`, so this isn't a new coupling) rather than
  extracting a new shared package. Not blocking this feature, but
  should be decided before or during T003b's implementation rather than
  left open indefinitely.

## Alternatives considered

- **A single flat top-level-only resolution rule** (no per-frame exact
  path matching) — rejected during interrogation: would confidently
  report a version for a specific frame that's provably running a
  *different* nested copy, violating constitution Article VI.
- **Treating inexact fallback matches as fully unresolved** (never
  report a version we can't tie to the exact frame) — considered as the
  strict reading of the pre-existing `LockedDependency` contract, but
  rejected in favor of relaxing that contract's doc comment: an
  honestly-labeled inexact match is more useful to an AI assistant than
  an empty result, and the wire format already supports both fields
  together.
- **Walking up from `workDir` to the nearest `package.json`** (Node's
  own resolution convention, no git dependency) — rejected as the
  *primary* mechanism in favor of `git rev-parse --show-toplevel`, to
  match the constitution's explicit "repo root" framing and stay
  consistent with 004's git-based repo detection. `workDir` is still
  used, but only as a direct (non-walking) fallback specifically when
  git detection fails outright (FR6) — not as a replacement for git
  when git is available.
- **Hard-failing (panic) when neither the git root nor the `workDir`
  fallback has a `package.json`** — rejected: a panic kills the whole
  process, not just this bundle section, contradicting FR7's own
  "valid, representable nil outcome" framing and the
  `BuildGitMetadata` precedent it's built on. `Dependencies` is nil in
  this case, same as every other unresolvable-manifest scenario.
- **Silently preferring one frame's match over another's when the same
  `PackageName` resolves to conflicting versions** (e.g. first-seen in
  chain order, or exact-match-over-fallback priority) — rejected: chain
  order is an artifact of stack unwinding, not a signal of which
  installed copy is "more correct," and an exact match from one frame
  doesn't make a *different* frame's own version wrong. Reporting a
  specific version as fact for a frame that's actually running a
  different copy is the same Article VI violation the flat top-level-
  only alternative above was already rejected for.

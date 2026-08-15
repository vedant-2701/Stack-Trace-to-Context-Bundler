# Plan: Own-code context extraction

Derived from `spec.md`. Must be consistent with `memory/constitution.md`.

## Architecture / approach

New package `internal/codecontext` (per `CONVENTIONS.md`'s existing file
layout — this package name is already reserved there: "shared,
language-agnostic: file read, snippet windowing, git blame"). Everything
here is a pure function or a function taking a small injected interface
for subprocess calls, per the project's existing testing conventions.

Two independent top-level entry points, matching the split already visible
in `contract.Bundle`'s shape (`GitMetadata` is bundle-level; `CodeContexts`
is per-frame):

- `BuildGitMetadata(ctx context.Context, workDir string) *contract.GitMetadata`
  — requirement 3/4/8. Returns `nil` for every "no repo" / timeout path.
  Never returns an error — a failure to find/read a repo is a valid,
  representable outcome (`nil`), not an error condition the caller must
  branch on separately. This mirrors `contract.TruncateRawInput`'s
  "outcome as a value, not an error" shape from `001-data-contract`.
- `BuildCodeContexts(ctx context.Context, chain []contract.ExceptionNode, language contract.Language, gitMeta *contract.GitMetadata) []contract.CodeContext`
  — requirements 1-2, 5-7, 9-10. Iterates every `ExceptionNode`/`Frame`,
  skips non-`own` buckets, builds one `CodeContext` per own-bucket frame.
  Takes the same `*contract.GitMetadata` `BuildGitMetadata` already
  produced — `hasRepo` collapses to `gitMeta != nil`; there is no separate
  `repoRoot` parameter anywhere in this feature, because it isn't needed:
  per-file `git status`/`git blame` calls run with `dir` set to that
  frame's own file's directory (`Frame.FilePath`, already on hand per
  frame), relying on git's own upward repo search rather than a
  passed-in root (this is the same mechanism requirement 9 already
  relies on for the multi-repo edge case). Also takes `language
  contract.Language`: `contract.CodeContext.Language` is required
  (non-`omitempty`), and nothing else in this feature has a source for
  it, since `Frame` carries no per-frame language (a bundle is
  single-language by design) — added at T007 implementation time, not
  originally listed here. Neither function takes a `runner` parameter:
  each is a thin public wrapper around an unexported, runner-injectable
  twin (`buildGitMetadata`/`buildCodeContexts`) that tests call directly
  via same-package fakes — see API/contracts below, the actual final
  shape both functions ended up with.

A small unexported `gitRunner` interface abstracts the actual subprocess
call, so `go test ./...` never needs a real `git` binary
(`CONVENTIONS.md`'s testing section):

```go
type gitRunner interface {
    // Run executes `git <args...>` with cwd set to dir, bounded by the
    // 10s timeout (spec.md requirement 8) via ctx. Returns stdout on
    // success; a non-nil error covers both a real git error and a
    // context-deadline timeout — callers distinguish via errors.Is(err,
    // context.DeadlineExceeded) where requirement 8 calls for
    // different handling per call type.
    Run(ctx context.Context, dir string, args ...string) (stdout string, err error)
}
```

Production implementation shells out via `os/exec` (Article VII). Tests use
a hand-written fake implementing the same interface (no mocking framework,
per `CONVENTIONS.md`), returning canned output or a simulated timeout per
test case.

Repo detection (`git rev-parse --is-inside-work-tree` only — see "Flagged
for recheck" below, resolved at T003: `--show-toplevel` was dropped),
staleness (`git status --porcelain <file>`), and blame
(`git blame --porcelain -L <start>,<end> <file>`) are three separate
`gitRunner.Run` calls with independent 10s timeouts (requirement 8) — not
one combined call, so one slow file's blame can't block another file's
staleness check or the initial repo detection.

Every degraded-but-continuing path this feature produces — no repo found,
a frame's `stale` result (modified, untracked, or a `status` timeout), a
blame failure, or a blame timeout — is exactly the `Warn` case
`CONVENTIONS.md`'s logging section describes ("degraded but continuing —
file not found, fell back to a note"). `context.go`'s `buildOneCodeContext`
calls `slog.Warn` (package-level default logger, stderr-only per
constitution Article II) at each point it sets `cc.Note` to a
degraded-path reason, carrying that same reasoning as log fields, so the
diagnostic is visible without `-v`/`-vv`. Centralized in `context.go`
rather than spread across `gitmeta.go`/`status.go`/`blame.go`
individually (originally planned per-file, revised at T007
implementation time): those three functions each return a value/error
pair, not a `note` string tied to a specific `CodeContext` field, so
`context.go`'s `buildOneCodeContext` is the one place every degraded
outcome's final `note` text actually exists, and logging there covers
all paths without a duplicated log line per helper. `Debug`-level logging
inside individual `gitRunner.Run` calls (e.g. the exact command run) is
left to implementation judgment, not specified further here.

## Stack & versions

No new external dependencies. `os/exec` (stdlib, subprocess), `context`
(stdlib, per-call timeout via `context.WithTimeout`), `bufio`/`os` (stdlib,
bounded file reads for the snippet window). Consistent with the project's
stdlib-first posture (`AGENTS.md`) — no diff/blame-parsing library needed,
`git blame --porcelain`'s output format is simple enough to parse directly.

## Data model

### Contract change (prerequisite — see spec.md's "Contract change
required" section for full rationale)

```go
// internal/contract/types.go
type Bundle struct {
    // ...
    GitMetadata *GitMetadata `json:"gitMetadata,omitempty"` // was: GitMetadata GitMetadata `json:"gitMetadata"`
    // ...
}

const SchemaVersion = "2.0.0" // was "1.0.0"
```

`GitMetadata`'s own three fields are unchanged (no `omitempty` added to
them) — once the pointer is non-nil, all three are always populated
(spec.md requirement 4).

### New: `internal/codecontext` internals (not part of the public contract)

```go
// gitStatus is the parsed result of `git status --porcelain <file>`,
// used internally to decide ok/stale (spec.md requirement 5) before
// constructing a contract.CodeContext.
type gitStatus int

const (
    gitStatusClean gitStatus = iota
    gitStatusModified          // tracked, uncommitted changes
    gitStatusUntracked         // never committed
    gitStatusUnknown           // status check itself timed out/failed — spec.md requirement 8's cautious "stale" path
)
```

`gitStatusModified`, `gitStatusUntracked`, and `gitStatusUnknown` all map
to `contract.StatusStale` when building the actual `CodeContext` — the
distinction only affects `note` text (spec.md requirements 5 and 8),
never the emitted `status` enum value.

## File / module layout

```
internal/codecontext/
  gitmeta.go          # BuildGitMetadata, repo detection (rev-parse)
  gitmeta_test.go
  status.go           # per-file staleness check (git status --porcelain), gitStatus type
  status_test.go
  snippet.go           # windowed file read + clamping (spec.md requirement 1)
  snippet_test.go
  blame.go             # git blame --porcelain parsing, contiguous-range grouping
  blame_test.go
  context.go           # BuildCodeContexts orchestrator, wires snippet+status+blame per own-bucket frame
  context_test.go
  runner.go            # gitRunner interface + real os/exec implementation
  runner_fake_test.go  # hand-written fake gitRunner shared by *_test.go files above
```

Window size (spec.md requirement 1) is an exported constant in
`snippet.go`, `DefaultContextLines = 5` (originally planned unexported;
revised at T004 implementation time since feature 011 needs to reference
it from outside this package — see API/contracts below, the authoritative
section for this), passed as an explicit parameter to the windowing
function rather than hardcoded inside it — so feature 011 can later
expose it as a real parameter without touching this function's internals,
only its caller.

## API / contracts (if applicable)

Public Go API surface of `internal/codecontext` (in-process library, no
HTTP/RPC surface, same pattern as `internal/contract`):

- `codecontext.BuildGitMetadata(ctx context.Context, workDir string) *contract.GitMetadata`
  — production entry point (wraps the real `gitRunner`; the interface
  itself stays unexported, consistent with `CONVENTIONS.md`'s "default to
  unexported" naming rule — tests reach it via same-package fakes, not a
  public seam).
- `codecontext.BuildCodeContexts(ctx context.Context, chain []contract.ExceptionNode, language contract.Language, gitMeta *contract.GitMetadata) []contract.CodeContext`
  — production entry point (wraps the real `gitRunner`, same pattern as
  `BuildGitMetadata` above). Takes the already-built `*contract.GitMetadata`
  (nil or non-nil) directly — `hasRepo` is just `gitMeta != nil`, and no
  `repoRoot` parameter exists anywhere in this feature (see Architecture
  section: per-file git commands use each frame's own file directory, not
  a shared root) — so 002b (pipeline wiring, when it exists) calls
  `BuildGitMetadata` once and passes the result straight into
  `BuildCodeContexts`, matching spec.md requirement 3's "detection happens
  once" rule structurally, not just by convention. Also takes `language
  contract.Language`, added at T007 implementation time (see Architecture
  section above for why).
- `codecontext.DefaultContextLines` — exported constant, `5`, for 011 to
  reference as its own default without duplicating the number.

## Testing strategy

- **Table-driven tests per file** (`CONVENTIONS.md`'s parser-testing
  convention applies equally well here): one table entry per real-world
  case — clean/tracked, modified, untracked, not-found, unreadable,
  no-repo, blame-fails, each timeout variant — using the hand-written fake
  `gitRunner`.
- **Snippet clamping tests**: target line at line 1, at the last line, and
  a file shorter than the window itself (fewer than 11 total lines).
- **Blame-grouping tests**: a snippet window spanning two different
  commits, confirming `BlameEntry` ranges split at the actual commit
  boundary, not one entry per line.
- **`go test ./...` must pass with no `git` binary on `PATH`** — enforced
  implicitly by never invoking the real `runner.go` implementation from
  any test, only the fake.
- A separate build-tag-gated test (matching `CONVENTIONS.md`'s existing
  pattern for `git`/`mvn`/`gradle`-dependent tests) exercises the real
  `os/exec` path against a throwaway `t.TempDir()` git repo, CI-only.

## Risks & open decisions

- **Flagged design call** (spec.md requirement 2): permission-denied and
  other unreadable-but-existing files reuse `"not_found"` rather than a
  new status value, distinguished only by `note` text. Revisit if this
  proves confusing in practice — no acceptance criterion currently
  exercises the specific wording, only that `status` is `"not_found"` and
  `note` is non-empty.
- **Flagged design call** (spec.md requirement 8): a `git status` timeout
  defaults to `"stale"` (cautious) rather than `"ok"` (optimistic) —
  deliberate per Article VI, but means a slow filesystem/large repo could
  produce more `"stale"` results than are actually warranted. No retry is
  attempted (consistent with Article IX's mvn/gradle precedent: no
  automatic online/retry fallback).
- Detached-HEAD `branch` value is the literal string `"HEAD"` (whatever
  `git rev-parse --abbrev-ref HEAD` returns) — no special sentinel value,
  flagged in spec.md requirement 4 but not separately interrogated; low
  stakes, easy to revisit.
- The MAJOR `schemaVersion` bump (`1.0.0` → `2.0.0`) is the first
  externally-visible contract break since `001-data-contract` shipped —
  worth confirming no other in-progress work (there is none currently
  in-progress per `specs/INDEX.md`) depends on `GitMetadata` being a
  required value type before this lands.
- 10s per-call git timeout is a first-pass estimate, not benchmarked
  (unlike Article IX's mvn/gradle 30s figure, which came from actual
  measurement per `memory/decisions/0001-benchmark-protocol.md`). No
  benchmarking round planned for this feature — flagged as an accepted,
  unvalidated estimate, revisit if real usage shows 10s is too tight for
  large repos' blame calls.

## Alternatives considered

- **A pre-pass "peek" at git status covering all own-bucket files in one
  call** (e.g. `git status --porcelain` with no path filter, then match
  paths in memory) instead of one `git status --porcelain <file>` call
  per frame. Rejected for this pass: fewer subprocess calls is a real
  efficiency win, but it also means a single slow/hanging `git status`
  call blocks every frame's staleness check instead of just one file's
  (same reasoning spec.md requirement 8 already applies to keeping
  rev-parse/status/blame as separate calls) — and multiple frames
  referencing the same file would need this result cached/reused anyway,
  adding complexity for a bundle-size scale (typically single-digit own
  frames) where the efficiency win is unlikely to matter. Revisit if
  profiling ever shows subprocess-call count is a real bottleneck.
- **A 4th `CodeContextStatus` enum value** (e.g. `"unknown"` or
  `"unreadable"`) instead of reusing `"not_found"`/`"stale"` with
  distinguishing `note` text. Rejected per spec.md's interrogation: adding
  a second contract change stacked on top of the `GitMetadata` pointer
  change (already a MAJOR bump) for a distinction the `note` field already
  carries wasn't worth the additional schema churn in one feature.
- **Single combined git call for rev-parse + status + blame** to reduce
  process-spawn overhead. Rejected: these three checks have genuinely
  different failure/timeout semantics per spec.md requirement 8 (a
  rev-parse failure means "no repo," a status failure means "assume
  stale," a blame failure means "omit blame") — collapsing them into one
  call would require inventing a way to distinguish which sub-check failed
  from a single combined error, adding parsing complexity to save process
  overhead that git subprocess calls (already used elsewhere per Article
  VII) don't meaningfully suffer from at this scale.

## Flagged for recheck at implementation time (design review)

**Resolved at T003.** `gitmeta.go`'s repo detection was documented above
as two calls: `git rev-parse --is-inside-work-tree` then
`--show-toplevel`. Reviewed against the concern that the binary can be
invoked from a subdirectory of the repo, not just its root, and something
needs the actual project root in that case.

Checked at T003: every value `BuildGitMetadata` actually produces
(`currentCommit`, `branch`, `uncommittedChanges` via `rev-parse HEAD`,
`rev-parse --abbrev-ref HEAD`, `git status --porcelain` with no pathspec)
resolves correctly regardless of which subdirectory `workDir` is — git
performs its own upward `.git` discovery on every one of these commands,
and `git status --porcelain` with no pathspec reports the whole working
tree's status, not just the invoking subdirectory's. This is the same
upward-search mechanism this plan already relies on for per-file commands
(requirement 9). `contract.GitMetadata` also has no field to hold a repo
root path even if `--show-toplevel` were called. No consumer of the
second call's output was found anywhere in this feature.

**Decision: `--show-toplevel` was dropped.** Repo detection
(`isInsideGitWorkTree` in `gitmeta.go`) is a single
`git rev-parse --is-inside-work-tree` call — one fewer subprocess call,
and it halves the detection-timeout budget for no loss of behavior. If a
future feature needs the actual repo root, that's a new, distinct need at
that point, not a revival of this dropped call.

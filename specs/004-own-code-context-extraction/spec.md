# Spec: Own-code context extraction

**Status:** Approved
**Folder:** specs/004-own-code-context-extraction

## Overview

Shared, language-agnostic logic that turns each own-bucket stack frame into
a `contract.CodeContext`: a windowed source snippet around the frame's
line, plus git blame for that window when available, plus one bundle-level
`contract.GitMetadata` object describing repo state. This is what lets an
AI assistant see the actual code at the point of failure — not just a bare
`file:line` reference it has to go ask the user to paste separately.

This feature does not decide which frames are "own" (that's 005a/006a's
job — `internal/contract/types.go`'s own doc comment on `Bucket` states
the classification logic "belongs to the parsers (005a/006a), not this
package," and `001-data-contract/spec.md`'s functional requirement 9
says the same for `bucket`); it consumes already-bucketed frames and
builds context for the ones marked `own`.

## User stories

- As a developer debugging with an AI assistant, I want each own-code stack
  frame's surrounding source lines and git blame included in the bundle, so
  the assistant has real code context instead of a bare file+line reference
  it has to ask follow-up questions about.
- As a solo developer who hasn't run `git init` (or is working in a
  throwaway/prototype folder), I want the tool to still work and produce a
  useful bundle, so the absence of git doesn't block me from using the tool
  at all.
- As a developer with local uncommitted edits (or a brand-new file that's
  never been committed), I want to be warned that a frame's file might not
  reflect a known-good, committed version, so I don't mistake unstable code
  for the code that actually produced the trace.

## Functional requirements

1. For every own-bucket frame, the system must extract a fixed-size text
   window of source lines centered on that frame's `lineNumber`: 5 lines
   before, the target line, 5 lines after (11 lines total), clamped to the
   file's actual first/last line when the window would run past either
   boundary. This window size is fixed in this feature — not
   user-configurable here. Exposing it as a CLI flag is **out of scope**,
   tracked separately as feature 011 (`specs/INDEX.md`), which depends on
   this feature's exported window-size parameter.
2. If the referenced file does not exist on disk at all, `CodeContext`
   `status` must be `"not_found"`, with a `note` explaining that.
   A file that exists but can't be read (e.g. permission denied) is
   **also** reported as `"not_found"` rather than a distinct status value
   — the contract's status enum stays closed at three values (per the
   contract-change decision below) — but the `note` must state the actual
   reason (e.g. "permission denied"), not claim the file is literally
   absent, so the distinction isn't lost even though the enum value is
   shared. *(Flagged design call — reusing an existing enum value rather
   than adding a fourth; revisit if this proves confusing in practice.)*
3. Before doing any per-file git work, the system must attempt to detect
   whether the tool is running inside a git working tree (via
   `git rev-parse`, from the current working directory — see requirement 9
   for the single-repo-per-bundle assumption). If no repository is found,
   or detection times out (requirement 8), git is treated as **entirely
   unavailable** for this bundle:
   - `Bundle.GitMetadata` is omitted (nil) — see the contract-change
     decision below.
   - Every own-bucket frame's `CodeContext.status` is `"ok"`, `blame` is
     omitted, and `note` explains that no git repository was found.
   - The tool does not fail or exit non-zero because of this — a missing
     git repo must never block a bundle from being produced (this is the
     core reason for the "Contract change required" section below).
4. If a repository **is** found, `Bundle.GitMetadata` must be populated
   exactly once per bundle: `currentCommit` (`git rev-parse HEAD`),
   `branch` (`git rev-parse --abbrev-ref HEAD`; literally `"HEAD"` when in
   a detached-HEAD state — no special-casing), and `uncommittedChanges`
   (repo-wide, from `git status --porcelain`).
5. For each own-bucket frame's file, when a repository is present, the
   system must run `git status --porcelain <file>` to decide between
   `"ok"` and `"stale"`. Any non-empty output — whether the file is
   modified-but-tracked or entirely untracked (never committed) — means
   `status: "stale"`, `blame` omitted. The `note` must distinguish which
   case applies ("uncommitted local changes" vs. "untracked, never
   committed") even though both share the same status value.
6. When a repository is present and a frame's file is tracked and clean
   (empty `git status --porcelain` output for that file), the system must
   run `git blame` (porcelain format) over the snippet's line range and
   populate `CodeContext.blame`: grouped into contiguous same-commit
   ranges (matching how `git blame -L` itself groups output — not one
   entry per line), with `commitHash`, `author`, `commitDate` (ISO 8601,
   derived once from porcelain's `author-time`/`author-tz`), and
   `summary` (first line of the commit message) per entry.
7. If `git blame` itself fails or errors despite the file being tracked
   and clean (e.g. the `git` binary is missing from `PATH`, or blame
   returns an unexpected error) — `status` stays `"ok"`, `blame` is
   omitted, and `note` explains the failure. This must not be treated as
   fatal to the bundle.
8. Every git subprocess call this feature makes (`rev-parse`, `status`,
   `blame`) must be bounded by a 10-second hard timeout, independently per
   call. A timeout is handled per call type, never left to hang or crash
   the bundle:
   - `rev-parse` (repo detection) timeout → treated identically to "no
     repository found" (requirement 3).
   - `status` (staleness/untracked check) timeout → treated as
     `"stale"`, with a `note` stating that git status could not be
     verified within the timeout — presenting an unverified file as
     confidently clean would be exactly the kind of silent guess Article
     VI prohibits, so the cautious outcome is chosen deliberately, not
     defaulted to for convenience.
   - `blame` timeout → treated identically to a blame failure
     (requirement 7).
9. `Bundle.GitMetadata` reflects **one repository per bundle** — own-code
   frames spanning multiple git repositories (monorepo submodules, etc.)
   is explicitly out of scope for v1, the same simplification already
   applied to `Dependencies`' single-manifest assumption. Repo detection
   (requirement 3) runs once, from the current working directory. Per-file
   git commands (status/blame) still resolve correctly against whichever
   repository actually contains that file — git's own upward directory
   search handles this — but if a frame's file is outside the repo found
   from the working directory entirely, it is treated the same as "no
   repository found" for that frame specifically (git commands run against
   it will naturally fail with "not a git repository", handled by
   requirements 6-7's existing failure paths — no new logic needed for
   this case).
10. `Bundle.GitMetadata` detection (requirement 3) always runs once per
    bundle build, regardless of whether the chain has any own-bucket
    frames at all — it's bundle-level repo state, not conditioned on
    whether there's anything to blame.

## Contract change required (blocks all other work in this feature)

`internal/contract.GitMetadata` (from `001-data-contract`, status `done`)
currently has no `omitempty` on any of its three fields, and `Bundle`
embeds it as a value (`GitMetadata GitMetadata`), not a pointer. Neither
supports "this bundle has no git data at all" — the only way to represent
that today would be to serialize empty/zero-value strings and `false`,
which is indistinguishable from "in a clean repo with no commits/branch
info," exactly the ambiguity constitution Article VI exists to prevent.

This feature requires:
- `Bundle.GitMetadata` becomes `*GitMetadata` (`json:"gitMetadata,omitempty"`)
  — nil and omitted entirely when requirement 3's "no repo" path fires.
  (Go's `omitempty` has no effect on non-pointer struct fields — a
  zero-value struct is never "empty" to `encoding/json` — so the pointer
  change is required, not optional, to make omission actually work.)
- When non-nil, all three inner fields (`currentCommit`, `branch`,
  `uncommittedChanges`) stay required/non-omitempty as before — once a
  repo is confirmed to exist, all three are always determinable.
- `contract.SchemaVersion` bumps from `"1.0.0"` to `"2.0.0"` — a field
  changing from always-required to sometimes-absent is a breaking change
  under this project's own semver policy (`001-data-contract/spec.md`'s
  functional requirement 6, restated in `001-data-contract/plan.md` and
  in `internal/contract/types.go`'s own `SchemaVersion` doc comment: "a
  field renamed, removed, or its type changed" triggers MAJOR), a
  **MAJOR** bump, not MINOR.
- `001-data-contract`'s golden fixtures
  (`testdata/example_java.json`/`example_ts.json`) must be regenerated
  (`go test ./internal/contract/... -update`, Vedant-run) after the struct
  change, and `001`'s own `spec.md`/`plan.md` updated in place to reflect
  the new shape — not left stale, per this project's own documented
  practice.

This is scoped as this feature's **first task** (`tasks.md`), touching a
`done`-status feature's files — that's normal (`005a` will touch
`internal/contract` too, and `001` is done there as well); it doesn't
reopen `001`'s own closed acceptance criteria, it extends the shape a new
feature legitimately needs.

## Non-functional requirements

- Snippet extraction must only read the needed window of a file, not load
  the entire file into memory — relevant for accidentally-huge own-code
  files (generated code, etc.), consistent with the project's general
  bounded-read posture (`rawInput`'s 512 KB cap, stdin's ~1 MB cap in
  `internal/cli`).
- All git subprocess calls must be testable without a real `git` binary
  installed (`go test ./...` must never require it) — per
  `CONVENTIONS.md`'s testing section, shelled-out commands sit behind a
  small interface with a hand-written fake, no mocking framework.

## Out of scope

- Exposing the snippet window size as a CLI flag — feature 011.
- AST-based snippet extraction — line-window text extraction only
  (constitution's existing out-of-scope list).
- Own-code frames spanning multiple git repositories in one bundle
  (requirement 9) — single-repo-per-bundle for v1.
- `codeContexts.status: "stale"` firing on line-number drift alone with a
  clean working tree (already out of scope per `001-data-contract`) — this
  feature only implements the uncommitted-changes/untracked case.
- A fourth `CodeContextStatus` enum value distinguishing "not found" from
  "found but unreadable" — reuses `"not_found"` with a clarifying `note`
  (requirement 2).

## Acceptance criteria

- [ ] Given an own-bucket frame whose file exists, is git-tracked, and is
      clean, when its `CodeContext` is built, then `status` is `"ok"` and
      `blame` contains one or more entries grouped by contiguous
      same-commit ranges within the snippet window.
- [ ] Given an own-bucket frame whose file does not exist on disk, when its
      `CodeContext` is built, then `status` is `"not_found"` with an
      explanatory `note`. **Satisfies the `001-data-contract` deferred
      criterion** (`memory/known-gaps.md`).
- [ ] Given an own-bucket frame whose file exists but can't be read (e.g.
      permission denied), when its `CodeContext` is built, then `status` is
      `"not_found"` with a `note` stating the actual reason.
- [ ] Given an own-bucket frame whose file has uncommitted local changes,
      when its `CodeContext` is built, then `status` is `"stale"`, `blame`
      is omitted, and `note` says so. **Satisfies the `001-data-contract`
      deferred criterion** (`memory/known-gaps.md`).
- [ ] Given an own-bucket frame whose file is untracked (never committed)
      in an otherwise-git-initialized repo, when its `CodeContext` is
      built, then `status` is `"stale"`, `blame` is omitted, and `note`
      distinguishes this from the uncommitted-changes case.
- [ ] Given a bundle built outside any git repository, when it's built,
      then `Bundle.GitMetadata` is omitted entirely (nil, not present in
      JSON), and every own-bucket frame's `CodeContext.status` is `"ok"`
      with `blame` omitted and a `note` explaining no repo was found.
- [ ] Given a bundle built inside a git repository, when it's built, then
      `Bundle.GitMetadata` is present with `currentCommit`, `branch`
      (literally `"HEAD"` if detached), and `uncommittedChanges` all
      populated.
- [ ] Given a frame's `lineNumber` near the start or end of its file, when
      its snippet is built, then the window is clamped to the file's
      actual bounds rather than requesting out-of-range lines.
- [ ] Given `git blame` fails despite a tracked, clean file (simulated via
      the hand-written fake, e.g. git binary missing), when the
      `CodeContext` is built, then `status` stays `"ok"`, `blame` is
      omitted, and `note` explains the blame failure.
- [ ] Given any git subprocess call exceeds 10 seconds (simulated via the
      fake), when the relevant `CodeContext`/`GitMetadata` is built, then
      the timeout is handled per requirement 8's per-call-type rule, and
      the bundle is still produced (never a hang or crash).

## Open questions

None — all resolved during interrogation.

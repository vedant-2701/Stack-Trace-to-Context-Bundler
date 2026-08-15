# Progress Log: Own-code context extraction

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:**
**Task(s):**
**What happened:**
**Deviations from plan (if any):**
**New open questions:**

---

**Date:** 2026-08-13
**Task(s):** T001
**What happened:** Patched `internal/contract/types.go`: `Bundle.GitMetadata`
changed from `GitMetadata` to `*GitMetadata` (`json:"gitMetadata,omitempty"`),
with a doc comment explaining nil/omission semantics; `SchemaVersion` bumped
from `"1.0.0"` to `"2.0.0"` (MAJOR, per functional requirement 6's own bump
policy — a field going from always-required to sometimes-absent). Regenerated
`testdata/example_java.json`/`example_ts.json` via
`go test ./internal/contract/... -update` (Vedant-run), with both golden
builders updated to construct `GitMetadata: &GitMetadata{...}`.
`specs/001-data-contract/spec.md` (functional requirement 12) and `plan.md`
(Data model section) updated in place to reflect the pointer type, the
omission behavior, and the version bump, so those files stay consistent with
the actual struct. Full gate passed: `go build ./...`, `go test ./...`,
`gofumpt -l`, `golangci-lint run` — all clean (Vedant-run/confirmed).
**Deviations from plan (if any):** `types_test.go` gained one additional
test not explicitly called for in `plan.md`'s testing strategy —
`TestSchemaVersion`, asserting `SchemaVersion == "2.0.0"` directly. Minor,
additive, no scope change.
**New open questions:** None.

---

**Date:** 2026-08-13
**Task(s):** T002
**What happened:** Created `internal/codecontext/` (new package). Added
`runner.go`: the unexported `gitRunner` interface (`Run(ctx, dir, args...)
(stdout string, err error)`), and `execGitRunner`, the production
`os/exec`-backed implementation. The 10s hard per-call timeout (spec.md
requirement 8) is applied *internally* by `execGitRunner.Run` via
`context.WithTimeout` on the passed-in `ctx` — flagged and decided
explicitly, since `plan.md`'s interface comment didn't pin down whether the
caller or `Run` itself owns the timeout, and T003/T007's production entry
points don't take a `runner` param they could use to manage it externally.
A real timeout is distinguished from an ordinary git error via `callCtx.Err()`
and surfaced wrapped so `errors.Is(err, context.DeadlineExceeded)` matches.
Added `runner_fake_test.go`: hand-written `fakeGitRunner` (a `fn` field,
no mocking framework, per `CONVENTIONS.md`) for T003/T005/T006's
table-driven tests to configure per case, plus three tests covering the
fake itself (arg/dir forwarding, error propagation, `DeadlineExceeded`
propagation). No test in the package invokes `execGitRunner`.
`go build ./...`, `go test ./internal/codecontext/...`, `gofumpt -l`, and
`golangci-lint run` all passed clean (Vedant-run/confirmed).
**Deviations from plan (if any):** None beyond the timeout-ownership
decision noted above.
**New open questions:** `runner.go` currently carries this package's doc
comment since it's the only real (non-test) file so far. `CONVENTIONS.md`'s
`package-comments` rule allows only one file to have it. When T007 adds
`context.go` (the orchestrator — arguably the more "central" file per
`plan.md`'s layout), consider moving the doc comment there.

---

**Date:** 2026-08-13
**Task(s):** T003
**What happened:** Added `internal/codecontext/gitmeta.go`: exported
`BuildGitMetadata(ctx, workDir)` wrapping unexported, runner-injectable
`buildGitMetadata(ctx, workDir, runner)`. Two decisions made and recorded
in code comments (both flagged to Vedant before implementing, per this
session's process):
1. Per `plan.md`'s own "flagged for recheck at T003" note, dropped the
   `git rev-parse --show-toplevel` call — confirmed nothing in this
   feature consumes a repo-root path (`contract.GitMetadata` has no such
   field; per-file git commands rely on git's own upward `.git` discovery
   per spec.md requirement 9). Detection is now a single
   `--is-inside-work-tree` call.
2. spec.md requirement 8 only names the *detection* call's timeout as
   mapping to "no repo." It's silent on a follow-up call
   (`rev-parse HEAD`/`--abbrev-ref HEAD`/repo-wide `status`) failing
   *after* a repo is confirmed to exist (e.g. a freshly-initialized repo
   with zero commits, where `HEAD` doesn't resolve). Decided: any such
   failure collapses to the same `nil` result as "no repo found," rather
   than a partially-populated `GitMetadata`, since requirement 4 requires
   all three fields always populated once a repo is confirmed, and a
   partial struct would violate that plus Article VI.
Also resolved a naming inconsistency between plan.md's Architecture
section (which names the function `BuildGitMetadata` with an exposed
`runner` param) and its API/contracts section (public entry point takes
no `runner`, interface stays unexported) — implemented per the
API/contracts section, via the exported/unexported function split above.
Added `gitmeta_test.go`: six table-driven-style tests via `fakeGitRunner`
— repo found clean, repo found dirty, detached HEAD, no repo found,
detection timeout, and repo-found-but-HEAD-unresolvable (covering the
second decision above). `go build ./...`, `go test
./internal/codecontext/... -run TestBuildGitMetadata`, `gofumpt -l`, and
`golangci-lint run` all passed clean (Vedant-run/confirmed).
**Deviations from plan (if any):** The two decisions above; both are
spec/plan gaps filled in during implementation, not deviations from an
existing explicit instruction.
**New open questions:** None.

---

**Date:** 2026-08-13
**Task(s):** T004
**What happened:** Added `internal/codecontext/snippet.go`: unexported
`buildSnippet(path, targetLine, contextLines)` streams the file line by
line via `bufio.Scanner`, discarding lines before the window and
breaking as soon as it scans past the window's end line, so a large file
with an early target line is never read past what's needed (non-functional
requirement). Clamps `StartLine`/`EndLine` to the file's actual bounds.
Errors are returned as-is from `os.Open`/scan so `errors.Is(err,
fs.ErrNotExist)` still works -- callers (T007) decide status/note wording
from that, same "caller decides" split as blame.go (T006).
Resolved a doc inconsistency: `plan.md`'s file-layout section called the
window-size constant unexported, but its API/contracts section listed it
as exported for feature 011 to reference externally. Implemented as
exported `DefaultContextLines = 5`, per the API/contracts section.
Added `snippet_test.go`: six tests matching the acceptance list exactly
(normal window, target line 1, target line at EOF, file shorter than the
window, file not found via `fs.ErrNotExist`, permission-denied via
`os.Chmod(0o000)` in `t.TempDir()`, auto-skipped under root).
Vedant's own gate run flagged one `golangci-lint` (errcheck) finding on
the unchecked `f.Close()` in the original `defer f.Close()`; fixed to
`defer func() { _ = f.Close() }()` (Vedant-applied, verified). `go build
./...`, `go test ./internal/codecontext/... -run TestSnippet`, `gofumpt
-l`, and `golangci-lint run` all passed clean after that fix.
**Deviations from plan (if any):** The `DefaultContextLines`
export-vs-unexported resolution above.
**New open questions:** None.

---

**Date:** 2026-08-13
**Task(s):** T005
**What happened:** Added `internal/codecontext/status.go`: the `gitStatus`
type from `plan.md`'s Data model section (`gitStatusClean`,
`gitStatusModified`, `gitStatusUntracked`, `gitStatusUnknown`), and
`checkFileStatus(ctx, filePath, runner)`, running `git status --porcelain
<basename>` with cwd set to the file's own directory. Empty output ->
clean; `??` prefix -> untracked; anything else non-empty -> modified.
Decision made and flagged to Vedant before implementing: extended the
cautious `gitStatusUnknown` collapse (spec.md requirement 8 names this
explicitly only for a timeout) to *any* git-status error, since that's
the same mechanism requirement 9 already describes for a file living
outside the detected repo ("git commands... naturally fail... no new
logic needed"). Distinct note wording for timeout vs. a generic failure.
Added `status_test.go`: the four cases in T005's acceptance list (clean,
modified, untracked, timeout), plus one extra
(`TestFileStatus_GenericFailure`) covering the requirement-9 edge case,
explicitly commented as going beyond the minimum list. `go build ./...`,
`go test ./internal/codecontext/... -run TestFileStatus`, `gofumpt -l`,
and `golangci-lint run` all passed clean (Vedant-run/confirmed).
**Deviations from plan (if any):** The requirement-8/requirement-9
gitStatusUnknown scope extension above.
**New open questions:** None.

---

**Date:** 2026-08-13
**Task(s):** T006
**What happened:** Added `internal/codecontext/blame.go`:
`buildBlame(ctx, filePath, startLine, endLine, runner)` runs `git blame
--porcelain -L <start>,<end> <file>` over the already-clamped snippet
range, same cwd-set-to-file's-directory approach as T003/T005. Parser
(`parseBlamePorcelain`) groups entries entirely off git's own
group-size field on the header line (present only on a group's first
line) rather than re-deriving grouping by comparing consecutive commit
hashes. Handles the non-contiguous same-commit reappearance case by
caching each commit's metadata (author, commitDate, summary) by hash on
first sight and reusing it when a later group repeats the hash without
repeating metadata; a hash referenced with nothing cached errors loudly
rather than emitting an empty Author/Summary (Article VI).
`CommitDate` computed once here (author-time + author-tz -> RFC3339),
verified against `testdata/example_java.json`'s existing format before
writing the code. Added `blame_test.go`: all five cases from T006's
acceptance list (single-commit window, multi-commit window split at the
real boundary, command failure, timeout, non-contiguous reappearance).
Vedant's gate run flagged one `golangci-lint` (staticcheck QF1001)
finding on `isHexSHA`'s De Morgan's-law-eligible condition; fixed
(Vedant-applied, verified equivalent and correctly applied). `go build
./...`, `go test ./internal/codecontext/... -run TestBlame`, `gofumpt
-l`, and `golangci-lint run` all passed clean after that fix.
**Deviations from plan (if any):** None beyond the QF1001 lint fix noted
above.
**New open questions:** None.

---

**Date:** 2026-08-13
**Task(s):** T007
**What happened:** Added `internal/codecontext/context.go`: exported
`BuildCodeContexts(ctx, chain, language, gitMeta)` wrapping unexported
`buildCodeContexts(ctx, chain, language, gitMeta, runner)`. Decision
flagged and made before implementing: `plan.md`'s listed signature had
no `language` parameter, but `contract.CodeContext.Language` is
required/non-omitempty and nothing else in this feature has a source
for it (`Frame` carries no per-frame language) -- added `language
contract.Language` as a parameter, mirroring `Bundle.Language` being a
single bundle-wide value. Also ensured the zero-own-frames case returns
a non-nil empty slice (`make([]contract.CodeContext, 0)`), not a nil
slice, since `Bundle.CodeContexts` has no `omitempty` and a nil Go slice
marshals to JSON `null`, contradicting `types.go`'s own "never null"
header comment.
`buildOneCodeContext` short-circuits through four stages in requirement
order: file read (T004) -> repo presence (`gitMeta != nil`) -> per-file
status (T005) -> blame (T006), stopping at the first stage that can't
proceed. A blame failure does not demote status away from `"ok"`
(requirement 7). `notFoundNote`/`blameFailureNote` distinguish
timeout/specific-error wording from the generic case, same pattern as
T005's `checkFileStatus`.
Added `context_test.go`: one test per acceptance-list combination
(clean/tracked with blame, not-found, unreadable, stale-modified,
stale-untracked, no-repo, blame-fails, zero own-bucket frames), plus
one extra (`TestBuildCodeContexts_FrameRefIndexing`) verifying
ChainIndex/FrameIndex across multiple nodes with non-own frames
interleaved -- nothing else in this task exercised that. `go build
./...`, `go test ./internal/codecontext/... -run TestBuildCodeContexts`,
`gofumpt -l`, and `golangci-lint run` all passed clean
(Vedant-run/confirmed).
**Deviations from plan (if any):** The `language` parameter addition
and the non-nil-empty-slice decision, both above.
**New open questions:** None.

---

**Date:** 2026-08-13
**Task(s):** T008
**What happened:** Added `internal/codecontext/integration_test.go`
(`//go:build integration`, excluded from default `go test ./...` and
lefthook/CI, run via `go test -tags integration
./internal/codecontext/...`): three tests against a real `git init`-ed
`t.TempDir()` repo, exercising `execGitRunner` for real rather than the
fake -- clean tracked file (`BuildGitMetadata` + `BuildCodeContexts`
against real `git blame` output, checked against the actual `HEAD`
commit hash and a real author name), uncommitted changes (`stale`), and
outside any repo (`nil`). This repo had no prior build-tag convention to
match, so used the standard Go idiom (`integration`).
Walked all 10 checkboxes in this feature's own `spec.md` Acceptance
Criteria section against real tests and checked every one, each with a
"Satisfied by ...; see TestName" note matching `001-data-contract/spec.md`'s
existing convention (checked that file first before writing these).
None deferred -- full coverage. Flipped this spec's Status line to
`Implemented`.
Closed the loop on `memory/known-gaps.md`'s two rows owned by this
feature (uncommitted-changes -> stale, file-not-present -> not_found):
removed both rows, and checked the corresponding two boxes in
`001-data-contract/spec.md` with "Satisfied by 004" notes, updating 001's
top status line from "five of ten" to "seven of ten" satisfied.
`go build ./...`, `go test ./internal/codecontext/...`, `go test -tags
integration ./internal/codecontext/...`, `gofumpt -l`, and `golangci-lint
run` all passed clean (Vedant-run/confirmed).
**Deviations from plan (if any):** None.
**New open questions:** None.

---

## Feature complete

All T001-T008 done. `internal/codecontext` now provides
`BuildGitMetadata` and `BuildCodeContexts` as this feature's public
surface, consumed by 002b (pipeline wiring) once that feature starts.
`DefaultContextLines` is the one other exported symbol, for 011 to
reference. See `specs/INDEX.md` for this feature's status flip to `done`
and what's next in the planned sequence.

---

**Date:** 2026-08-15
**Task(s):** Post-T008 PR review fix pass (Copilot)
**What happened:** Verified all seven Copilot review threads against the
actual files before acting on any of them. Six were legitimate, applied:
1. `bufio.Scanner`'s default ~64KiB token limit in `snippet.go` and
   `blame.go` could fail on a single minified/generated source line.
   Added `scanner.Buffer(...)` (1MiB ceiling, `maxScannerLineBytes`) to
   both. Added `TestSnippet_LongLine`/`TestBlame_LongLine`.
2. `blame.go`'s metadata-parsing loop could silently proceed if EOF hit
   before a group's terminating content line (malformed/truncated
   porcelain output despite `err == nil`, an unlikely but real gap since
   `execGitRunner` only guards against a non-zero exit, not a
   well-formed-but-truncated stdout). Added an explicit
   `sawContentLine` check that fails loudly instead. Added
   `TestBlame_TruncatedMetadataBlock`.
3. **Real code gap, not docs drift**: `plan.md`'s Architecture section
   has always mandated `slog.Warn` on every degraded-but-continuing path
   (no-repo, stale, blame failure) -- never implemented across
   T003/T005/T006/T007. Added `warnDegraded` in `context.go`, called at
   each of the four points `buildOneCodeContext` sets a degraded `cc.Note`
   -- centralized there rather than spread across
   `gitmeta.go`/`status.go`/`blame.go` as `plan.md` originally described,
   since those three return value/error pairs, not a `note` tied to a
   specific `CodeContext`; `context.go` is the one place every path's
   final note text actually exists. Added
   `TestBuildCodeContexts_LogsWarnOnDegradedPath`, which installs a
   capturing `slog.Handler` and asserts a Warn record actually fires --
   not just that the code compiles.
4. `plan.md` drift confirmed on four more points (all real, `plan.md`
   itself never updated after the T002/T003/T004 sessions that made these
   calls, even though `progress.md` recorded the reasoning at the time):
   public API signatures still showing a `runner` param and missing
   `language`; repo detection still described as two calls including
   `--show-toplevel`; the "Flagged for recheck" section left as an open
   question that was actually already resolved at T003;
   `DefaultContextLines` still described as unexported. Fixed all four
   locations plus the two above (six total edits to `plan.md`).
5. Empty-file edge case (`buildSnippet` returning `StartLine == EndLine
   == 0` for a zero-line file) -- flagged to Vedant rather than silently
   fixed, since it changes observable behavior. Vedant's direction: don't
   reuse `not_found`'s existing wording ambiguously, give it a genuinely
   distinct Go-level error. Added `errEmptyFile` sentinel in `snippet.go`,
   a `notFoundNote` branch for it in `context.go` (still surfaces as
   `contract.StatusNotFound` -- a 4th status value was already rejected
   once in this same `plan.md` for the permission-denied case, same
   reasoning applies here, not reopened). Added `TestSnippet_EmptyFile`
   and `TestBuildCodeContexts_EmptyFile`. Recorded as an 11th acceptance
   criterion in `spec.md` (explicitly noted as PR-review-surfaced, not
   from the original interrogation) and folded into requirement 2's text,
   rather than left as an implementation-only decision with no spec trail.
**Deviations from plan (if any):** All of the above; each is a genuine
gap (four in code, six in docs) this session's own earlier work left
behind, not scope creep.
**New open questions:** None.

---

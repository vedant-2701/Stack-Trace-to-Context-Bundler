# Progress Log: Markdown renderer

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-09-18
**Task(s):** Pre-implementation audit (before T001)
**What happened:** A cross-chat audit of `spec.md`/`plan.md`/`tasks.md`
against each other, `memory/constitution.md`, `CONVENTIONS.md`, and
`001-data-contract`'s current files surfaced four issues, all confirmed
and fixed before any task work started:
1. `plan.md`'s Architecture section (`map[contract.FrameRef]contract.CodeContext`)
   and Data model section (a private `frameKey` mirror type) described
   two different, incompatible map-key designs for the same lookup, and
   `tasks.md` T007 sided with the unused one. Fixed: dropped the
   `frameKey` type entirely -- `contract.FrameRef` is already exported
   and comparable, no mirror needed. Reworded T007 accordingly.
2. Spec.md requirement 18's example truncation note ("input truncated at
   the 512 KB cap") hardcoded a number owned by `001`
   (`rawInputCapBytes`), which was also unexported, so 007 had no way to
   read the real value even if it wanted to. Fixed cross-feature: `001`
   exported the constant (`RawInputCapBytes`, see
   `specs/001-data-contract/progress.md`'s matching entry) and req. 18
   now says to format from it rather than hardcode the figure.
3. `spec.md`'s own header still said "Spec'd" after `plan.md`/`tasks.md`
   were both fully written -- stale relative to `specs/INDEX.md`'s
   correct `planned` status and inconsistent with how `001`/`004` keep
   their own headers live. Fixed: header now says "Planned (plan.md +
   tasks.md complete)".
4. `plan.md`'s File/module layout planned hand-written `contract.Bundle`
   JSON literals as render-test fixtures, which sits in real tension
   with constitution Article IV ("no second copy of the [bundle] shape
   anywhere else in the repo... never hand-written in parallel") and
   has no precedent anywhere in this repo -- `001` itself only ever
   builds its own fixtures (`exampleJavaBundle()`/`exampleTSBundle()`)
   as Go struct literals, never hand-authored JSON. Fixed: fixtures are
   now `contract.Bundle{...}` Go literals in a new `fixtures_test.go`
   (compile-checked against `types.go`, so a field rename breaks the
   build instead of silently producing a stale-but-passing test), except
   `ts_basic`, which unmarshals the existing, already-generated
   `internal/contract/testdata/example_ts.json` rather than re-authoring
   it (`exampleTSBundle()` itself lives in a `_test.go` file and can't be
   imported across packages). Testing strategy section's golden-file-test
   bullet updated to match.
No fifth issue found. Verified: not run by the user directly in this
session -- these are doc-only edits to `spec.md`/`plan.md`/`tasks.md`
(plus the cross-feature `001` code change logged separately), no build/
test/lint gate applies to them directly, though `001`'s own gate should
still be run before that change is committed.
**Deviations from plan (if any):** N/A -- this session predates any task
implementation; these are corrections to the plan itself, not deviations
from it.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T001 — Package scaffold + `escapeMarkdown`
**What happened:** Created `internal/render/markdown.go` (package doc
comment + stub `Markdown(_ contract.Bundle) string`), `escape.go`
(`escapeMarkdown`, single forward-scan over runes against the curated
`escapeCharSet` constant — backslash, backtick, `*_{}[]()#+-.!|><`), and
`escape_test.go` (one case per character, a no-op plain-text case, a
combined-characters case, and two generic-type cases covering `<`/`>`
together, e.g. `List<String>`).
`golangci-lint`'s first run flagged `unused-parameter` on `Markdown`'s
stub `b` argument (`revive`) — fixed by naming it `_` and rewording the
doc comment to say "the bundle" instead of "b", rather than suppressing
the lint or giving the parameter artificial use.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine after the `revive` fix; all four clean, confirmed
"done, no errors".
**Deviations from plan (if any):** None — matches T001's scope exactly.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T002 — `renderPreamble` + `renderMetadata`
**What happened:** Added `renderPreamble` (static one-line blockquote,
wording not specified by spec.md beyond "orient the reader", not
load-bearing for any test), `renderMetadata` (Language/OS/Runtime/Git/
Fingerprint bullet list, Git line omitted when `GitMetadata` is nil), and
two private sub-helpers not named in plan.md's Architecture section —
`renderRuntime` and `renderGit` — to keep req. 20/21's branching logic
testable in isolation rather than inlined in `renderMetadata` (flagged to
and accepted by the user before implementation; not a change to the
public/traceable helper surface, just internal decomposition).
`renderRuntime` handles all four `VersionSource`-non-`trace` x
`Note`-present/absent combinations uniformly (Version and Note each
appended independently when present; falls back to "(version unknown)"
only when both are absent), rather than branching separately per
VersionSource enum value.
One test-authoring bug surfaced by the first lint/test run (not a code
bug): the `local-environment with note` test's expected string forgot
that `-` is itself in `escapeMarkdown`'s escape set, so "node -v" inside
the note renders as "node \-v" — fixed the test's `want` value, no
production code changed.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine after the test-string fix; all four clean, confirmed
"done, no errors".
**Deviations from plan (if any):** Two unexported helpers
(`renderRuntime`, `renderGit`) added beyond plan.md's named helper list—
internal decomposition only, no change to `renderMetadata`'s signature,
behavior, or the requirements it satisfies.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T003 — `renderSnippet`
**What happened:** Added `renderSnippet(s contract.Snippet, lang
contract.Language) string`: trims `Code`'s one unconditional trailing
`\n` before splitting (per `buildSnippet`'s own construction, rather than
splitting raw and discarding the last element -- equivalent result,
chosen for readability), computes line-number column width from
`EndLine`'s digit count so numbers stay aligned across the block, and
marks the `TargetLine` row with a leading `→ ` (two-space-wide non-target
marker keeping columns aligned). No escaping applied -- snippet content
is exempt (req. 19), the fence protects it.
Format chosen (not fully dictated by spec.md, which only requires
line-number prefixing + target marker + language fence): `<marker><right
-aligned line number> | <line content>`, e.g. `→ 11 |     bar()`.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None — matches T003's scope; the
exact line-rendering format was an open implementation choice spec.md
left to the renderer, not a deviation from anything specified.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T004 — `renderBlameTable`
**What happened:** Added `renderBlameTable(entries []contract.BlameEntry)
string`: a Markdown table with columns `Lines | Commit | Author | Date |
Summary`, one row per entry. `Lines` renders as a single number when
`StartLine == EndLine`, else `Start-End` (format not dictated by spec.md
beyond "one row per contiguous range"). `Commit` is the first 7
characters of `CommitHash`; `Date` is the first 10 characters of
`CommitDate` (safe since ISO 8601 is fixed-width up to `YYYY-MM-DD`).
`Author`/`Summary` go through `escapeMarkdown`, which is also what
satisfies the acceptance criterion that a `|` in `Summary` can't corrupt
the table -- `|` is already in the escaped character set, so no
table-specific escaping logic was needed beyond the existing helper.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** Second pre-implementation audit pass (before T001)
**What happened:** A follow-up audit, re-verifying the prior pass's four
fixes and looking for anything else, found two more issues:
5. `001/plan.md`'s "API / contracts" section wasn't updated when
   `RawInputCapBytes` was exported (issue 2 above) -- a doc-drift gap
   introduced by that very fix. Fixed: added it as a fifth bullet there
   (see `001/progress.md`'s matching entry).
6. `renderSnippet`'s Architecture-section signature
   (`renderSnippet(s contract.Snippet, lang contract.Language) string`)
   didn't match how its own narrow unit test was described in three
   places: `spec.md`'s Out of scope section, `plan.md`'s Testing
   strategy section, and `tasks.md` T003 all said the test "constructs a
   bare `CodeContext{Language: LanguageJava, ...}`" -- a type
   `renderSnippet` doesn't take. Traced to `spec.md`'s Out of scope
   section, written before `plan.md`'s Architecture section split
   `renderCodeContext` into finer helpers and settled on `renderSnippet`
   taking a `Snippet`+`Language` pair directly; `plan.md`/`tasks.md`
   inherited the stale `CodeContext` framing from `spec.md` rather than
   the other way around. Fixed all three to construct a bare
   `contract.Snippet{...}` and pass `contract.LanguageJava` directly,
   matching the real signature.
   Separately, this surfaced a genuine coverage gap the original wording
   had been gesturing at without landing anywhere real: nothing planned
   actually proves `CodeContext.Language` reaches `renderSnippet` through
   `renderCodeContext`'s real call path -- T003 tests `renderSnippet` in
   isolation with a language handed to it directly, and T005's
   `CodeContext`-level tests vary `Status`/`Blame`, not `Language`. Added
   one further case to T005: an `ok`-status `CodeContext` with `Language:
   contract.LanguageJava` set, asserting the fence tag comes through
   correctly end-to-end. Still no full Java-shaped `Bundle` (spec.md's
   Out of scope stands) -- one `CodeContext` value.
No further issues found. Verified: not run by the user directly in this
session -- doc-only edits to `spec.md`/`plan.md`/`tasks.md` (007) and
`plan.md`/`progress.md` (001), no build/test/lint gate applies.
**Deviations from plan (if any):** N/A, same reasoning as the prior
entry.
**New open questions:** None.

---

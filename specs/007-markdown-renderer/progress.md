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
**Task(s):** T005 — `renderCodeContext`
**What happened:** Added `renderCodeContext(cc contract.CodeContext)
string`, branching on `Status`: `not_found`/`stale` → a single flagged
`⚠ <Note>` line, nothing else; `ok` → always renders the snippet
(`renderSnippet(cc.Snippet, cc.Language)`), followed by
`renderBlameTable(cc.Blame)` when non-empty, or a flagged `⚠ <Note>` line
in its place when empty. Tests cover all four branches plus a fifth
case (`ok` status, `Language: contract.LanguageJava`) asserting the
rendered output opens with `` ```java `` -- this is the case that
actually proves `CodeContext.Language` reaches `renderSnippet` through
`renderCodeContext`'s real call path, since T003's own tests pass the
language directly and can't catch a hardcoded tag here. The two `ok`-
status tests assert against `renderSnippet(...)`+`renderBlameTable(...)`/
`"⚠ <Note>\n"` composed at test time rather than a separate literal
string -- appropriate here since this task is pure composition of
already-tested helpers, not new rendering logic of its own.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T006 — `renderFrame`
**What happened:** Added `renderFrame(f contract.Frame, cc
*contract.CodeContext) string`: `at [ClassName.]MethodName
(FilePath:LineNumber[:ColumnNumber]) — <suffix>\n`, `ClassName.` prefix
omitted cleanly when absent, `:ColumnNumber` omitted when 0 (Java never
carries one). `FilePath` rendered verbatim, never escaped -- it's a
normalized path, not developer-arbitrary prose (req. 19's escape list
doesn't include it). Suffix by bucket: `own`, `dependency: <PackageName>`
(identity only, no version -- req. 13), `runtime`. When `f.Bucket ==
BucketOwn && cc != nil`, appends `renderCodeContext(*cc)` right after the
frame line (req. 9); nil `cc` or a non-own bucket appends nothing.
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

**Date:** 2026-09-19
**Task(s):** T007 — `renderChain`
**What happened:** Added `renderExceptionNodeHeader` (private helper,
same internal-decomposition pattern as T002's `renderRuntime`/`renderGit`
-- not in plan.md's named list, flagged before implementation) for the
`### ClassName` heading + `Message` blockquote, escaping each line of
`Message` independently and rendering a wholly-empty line as a bare `>`
rather than an actual blank line, since CommonMark ends a blockquote at
the first `>`-less line. Added `renderChain`: builds the
`map[contract.FrameRef]contract.CodeContext` once up front, looks up
each own-bucket frame's context by `{ChainIndex, FrameIndex}` built from
loop position (never `Frame.Index`'s field value -- defense in depth per
plan.md, contract now guarantees the two agree), renders the
elided-frames line only when `ElidedFrameCount > 0`, and inserts
`\nCaused by ↓\n\n` between consecutive nodes only (never after the
last).
One test-authoring bug surfaced by the verification run (not a code
bug, same class as T002's): the multiline-message test's expected string
forgot `!` is itself in `escapeMarkdown`'s escape set, so "foo !== bar"
renders as "foo \!== bar" -- fixed the test's `want` value, no
production code changed.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T008 — `renderDependencies`
**What happened:** Added `renderDependencies(d *contract.Dependencies)
string`: `nil` → `""` (no heading, no content). Otherwise `##
Dependencies` heading, one bullet per `d.Locked` entry sorted ascending
lexically by package name (collected into a slice and `sort.Strings`ed --
never a bare `range d.Locked`, since Go randomizes map-range order on
every iteration and this feature's golden-file tests need deterministic
output). Bullet format: `declared <Direct[pkg]>, ` omitted entirely
(never left dangling) when `pkg` has no `Direct` entry; `resolved
<Version>` or the literal `resolved unresolved` when absent;
`Locked[pkg].Note` (escaped -- req. 19) appended in parentheses whenever
present regardless of `Version`.
Another test-authoring bug, same recurring class as T002/T007 (not a
code bug): the fallback-match test's expected string again forgot `-` is
in the escape set ("top-level lookup" → "top\-level lookup") -- fixed
the test's `want` value. Flagged this as a pattern (third occurrence) and
adopted a mitigation going forward: for `Note`/`Message`-bearing test
cases, compute the expected escaped fragment via a direct
`escapeMarkdown(...)` call in the test rather than hand-typing the
escaped literal, rather than continuing to hand-type and re-fix.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T009 — `renderRawInput`
**What happened:** Added `longestBacktickRun` (private helper, scans for
the longest consecutive-backtick run) and `renderRawInput(raw string,
truncated bool) string`: fence length is always `longestBacktickRun(raw)
+ 1`, minimum 3 (req. 17); truncation note appears only when `truncated`
is true, its KB figure computed from `contract.RawInputCapBytes / 1024`
rather than hardcoded (req. 18); `raw` itself is never escaped
(fence-exempt, req. 19).
While inserting this via a targeted edit, an anchor match accidentally
consumed the first line of `renderFrame`'s existing doc comment,
leaving it truncated mid-sentence -- caught immediately on re-reading the
file before writing tests, fixed with one more edit restoring the
missing line, no functional code affected.
Applied the T008 mitigation for the first time: the truncation-note test
asserts against a string built with `fmt.Sprintf(...,
contract.RawInputCapBytes/1024)` rather than a hand-typed KB figure.
**Verified:** user ran `go build ./...`, `go test ./internal/render/...`,
`golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/`
on their machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T010 — `Markdown()` composition
**What happened:** Replaced the stub with the real `Markdown(b
contract.Bundle) string`: `renderPreamble` → `renderMetadata` → (blank
line) → `renderChain` → (blank line, only when non-empty) →
`renderDependencies` → (blank line) → `renderRawInput`, matching spec.md
req. 1-5's document order. The blank-line placement around Dependencies
and before the chain/raw-input sections is a readability judgment call,
not something spec.md pins down exactly -- flagged to the user before
implementation; T011's hand-review of the first golden fixture against
spec.md's own rendered mockup is where this actually gets checked, not
this task. Pure composition -- no new rendering logic.
**Verified:** user ran `go build ./...`, `go vet ./...`, `go test
./internal/render/...`, `golangci-lint run ./internal/render/...`,
`gofumpt -l ./internal/render/` on their machine; all clean, confirmed
"done, no errors". No new test added, per this task's own acceptance
criteria -- T011 is the first end-to-end assertion.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T011 — Golden fixture: `ts_basic`
**What happened:** Built the golden-test harness (`-update` flag,
table-driven `TestMarkdown`, `assertGoldenMarkdown`, cloned from
`internal/contract/types_test.go`'s own `assertGolden` pattern) and the
first fixture -- but the fixture's design changed mid-task from what
`plan.md` had specified, on the user's pushback, and that pushback was
right:
1. Original plan (from the earlier pre-implementation audit) had
   `ts_basic` unmarshal `internal/contract/testdata/example_ts.json`
   rather than a hand-authored `contract.Bundle{...}` literal, citing
   constitution Article IV. The user objected: production code never
   round-trips a `Bundle` through JSON before rendering it -- `Markdown()`
   is always called on an in-memory Go value -- so testing via JSON is
   both unrealistic and, concretely, masked a real bug (below) that a
   freshly-authored fixture wouldn't have. Re-reading Article IV's actual
   text on request confirmed it: the article is about never hand-writing
   a second copy of the bundle *shape* (a JSON Schema, a mirror struct),
   not about which format other features' fixtures must be built in. A
   `contract.Bundle{...}` literal is an instance of the one canonical
   shape, compile-checked -- exactly what T012-T016's fixtures were
   already planned to be. Rewrote `tsBasicBundle` as a hand-authored
   literal and fixed `plan.md`'s File/module-layout and Testing-strategy
   sections, which had documented the JSON-reuse special-case as if
   Article IV required it.
2. While reviewing the (now-superseded) JSON-backed golden output by eye
   against `spec.md`'s requirements 1-22, found a real bug in
   `internal/contract/types_test.go`'s `exampleTSBundle()` (001's own
   fixture, not 007's code): both `CodeContext.Snippet` entries had
   `EndLine` one higher than their `Code`'s actual line count (e.g.
   `StartLine:25, EndLine:29` but only 4 real lines, 25-28). Moot once
   the fixture stopped depending on that JSON file -- left alone, not
   fixed, since it's now out of 007's scope; flagged to the user as a
   latent 001 bug they may want addressed separately.
3. The user also specified the real own-code snippet window size --
   ±5 lines around the target (11 total) -- which was verified (not
   taken on faith) against `specs/INDEX.md`'s 011 row ("fixed at ±5/side
   in 004") before use. The new fixture's two snippets (22-32 target 27;
   58-68 target 63) both use this real window size, replacing the
   original fixture's inconsistent 4-5 line windows.
One authoring bug caught before running anything: the raw string
literals for both `Snippet.Code` values already end with exactly one
trailing newline (from the line break before the closing backtick), and
an extra `+ "\n"` was mistakenly appended on top -- would have produced
a spurious 12th blank line once `renderSnippet` trimmed only one trailing
newline. Caught and fixed before the first test run.
**Verified:** user ran `go test ./internal/render/... -run TestMarkdown
-update` to generate the fixture from the new literal; content
hand-reviewed by Claude against spec.md reqs. 1-22 (all clean: preamble,
metadata order/omission, chain structure, frame-line formats including
dependency PackageName-only suffix, snippet exact-line-count with correct
target markers, blame table, `Caused by ↓` placement, Dependencies
formatting, raw-input fence, escaping) -- approved, then user ran `go
build ./...`, `go test ./internal/render/...`, `golangci-lint run
./internal/render/...`, `gofumpt -l ./internal/render/` on their
machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** `fixtures_test.go`'s `ts_basic` builder
and `plan.md`'s File/module-layout + Testing-strategy sections were
rewritten mid-task per the user's correction above -- not a deviation
from the *confirmed* T011 plan (which was updated to match before
implementation), but a correction to an earlier session's
pre-implementation-audit decision that turned out to rest on a
misreading of Article IV.
**New open questions:** Whether to fix `exampleTSBundle()`'s `EndLine`
off-by-one in `internal/contract/types_test.go` (001's own fixture) --
out of 007's scope, left to the user's discretion, not filed to
`known-gaps.md` yet.

---

**Date:** 2026-09-19
**Task(s):** T012 — Golden fixtures: `no_git_metadata`, `no_dependencies`
**What happened:** Added two deliberately minimal `contract.Bundle{...}`
literals (single exception node, one own-bucket frame with an 11-line
snippet + single blame row, one runtime frame each) rather than reusing
`ts_basic`'s larger two-node shape -- each fixture isolates exactly one
concern: `noGitMetadataBundle` (`GitMetadata: nil`, `Dependencies`
present) proves the `Git:` line is omitted from metadata without
disturbing anything else; `noDependenciesBundle` (`Dependencies: nil`,
`GitMetadata` present) proves the whole `## Dependencies` heading is
absent, not rendered empty. No harness changes needed -- two new
`TestMarkdown` table entries reusing T011's `assertGoldenMarkdown`.
**Verified:** user ran `go test ./internal/render/... -run TestMarkdown
-update` to generate both fixtures; both hand-reviewed by Claude against
spec.md reqs. 1-22 (clean -- confirmed the `Git:` line's clean omission,
the `## Dependencies` heading's complete absence with the unconditional
pre-raw-input blank line still intact so no double-blank/missing-blank
artifact appears where the section would have been, and that
`LocalEnvironment`+`Note` Runtime formatting and Message escaping stayed
consistent with earlier tasks); approved, then user ran `go build ./...`,
`go test ./internal/render/...`, `golangci-lint run ./internal/render/...`,
`gofumpt -l ./internal/render/` on their machine; all four clean,
confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

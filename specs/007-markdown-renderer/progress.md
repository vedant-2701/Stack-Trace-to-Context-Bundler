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

**Date:** 2026-09-19
**Task(s):** T013 — Golden fixtures: `code_context_not_found`,
`code_context_stale`, `code_context_ok_no_blame`
**What happened:** Added three deliberately minimal `contract.Bundle{...}`
literals, same T012 pattern (single node, one own-bucket frame carrying
the `CodeContext` under test, one runtime frame, `GitMetadata`/
`Dependencies` present and ordinary so neither is also under test),
each isolating one of spec.md req. 10's three non-table `CodeContext`
outcomes: `codeContextNotFoundBundle` (`Status: not_found`, zero-value
`Snippet`/`Blame`) and `codeContextStaleBundle` (`Status: stale`, same
shape) both prove only a flagged `⚠ <Note>` line renders; `codeContext
OKNoBlameBundle` (`Status: ok`, real 11-line snippet, `Blame: nil`,
`Note` set) proves the snippet still renders but the table is replaced
by a flagged `⚠ <Note>` line.
One authoring bug caught before running anything, same class as T011's:
the `codeContextOKNoBlameBundle` snippet (`StartLine:5, EndLine:15`)
was first written with 12 real lines instead of 11 -- caught by manually
recounting the raw string against `EndLine-StartLine+1` before running
any command, fixed by dropping the window's trailing closing brace
(matching `ts_basic`'s own precedent of a window ending mid-function).
Three new `TestMarkdown` table entries, no harness changes.
**Verified:** user ran `go test ./internal/render/... -run TestMarkdown
-update` to generate all three fixtures; all hand-reviewed by Claude
against spec.md req. 10 (clean -- confirmed `stale` behaves identically
to `not_found`, parens/semicolon escaping stayed consistent with earlier
tasks, and the `ok`+empty-`Blame` case's snippet-then-Note-line ordering
matched `renderCodeContext`'s real branching); approved, then user ran
`go build ./...`, `go test ./internal/render/...`, `golangci-lint run
./internal/render/...`, `gofumpt -l ./internal/render/` on their
machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T014 — Golden fixtures: `elided_frames`, `multiline_message`
**What happened:** Added `elidedFramesBundle` (two-node chain; node 0's
frames normal, `ElidedFrameCount: 0`; node 1's own frame carries
`ElidedFrameCount: 6`) proving the elided-frames line renders only for
the node with a nonzero count, placed immediately after that node's
frame list; and `multilineMessageBundle` (single node, `Message` with a
wholly empty line embedded in the middle) proving req. 6-7's blockquote
behavior -- already unit-tested directly in T007 -- also survives a full
end-to-end `Markdown()` render, not just `renderChain` in isolation.
Both snippets' line counts (11 each, `EndLine-StartLine+1`) and target-
line positions were manually recounted against the raw string literals
before running anything, per the T011/T013 lesson -- no authoring bugs
this time.
Two new `TestMarkdown` table entries, no harness changes.
**Verified:** user ran `go test ./internal/render/... -run TestMarkdown
-update` to generate both fixtures; both hand-reviewed by Claude against
spec.md req. 14 and req. 6-7 (clean -- confirmed the elided-frames line's
correct placement relative to the last node and the `Caused by ↓`
transition, and the embedded-blank-line blockquote staying unbroken with
correct escaping and target-line alignment); approved, then user ran `go
build ./...`, `go test ./internal/render/...`, `golangci-lint run
./internal/render/...`, `gofumpt -l ./internal/render/` on their
machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T015 — Golden fixtures: `markdown_special_chars`,
`backtick_run_in_raw_input`
**What happened:** Added `markdownSpecialCharsBundle` (`Message`
contains `Map<string, number>` and `List<String>` together) proving
`<`/`>` escaping survives a full end-to-end render -- T001 already
unit-tests `escapeMarkdown` directly on this shape, this confirms it
through the real `renderChain`/`Markdown()` path, and separately confirms
the snippet's own `<T>`/generic syntax stays unescaped inside the fence
(fence-exempt). Added `backtickRunInRawInputBundle` (`RawInput` contains
an embedded 3-backtick run wrapping "some embedded code block") proving
the enclosing fence upgrades to 4 backticks end-to-end -- T009 already
unit-tests `renderRawInput` directly on this shape.
`backtickRunInRawInputBundle`'s `RawInput` is a regular quoted Go string,
not a raw string literal, specifically because it needs to contain
literal backtick characters, which a Go raw string can't hold.
Two new `TestMarkdown` table entries, no harness changes.
**Verified:** user ran `go test ./internal/render/... -run TestMarkdown
-update` to generate both fixtures; both hand-reviewed by Claude against
spec.md req. 19 and req. 17 (clean -- confirmed both `<`/`>` occurrences
escaped in Message while the snippet's own generics render unescaped,
and the fence correctly escalating to 4 backticks around the embedded
``` run without prematurely closing); approved, then user ran `go build
./...`, `go test ./internal/render/...`, `golangci-lint run
./internal/render/...`, `gofumpt -l ./internal/render/` on their
machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T016 — Golden fixtures: `raw_input_truncated`,
`dependency_states`, `runtime_version_states`
**What happened:** Added `rawInputTruncatedBundle` (`RawInputTruncated:
true`) confirming the truncation note's correct KB figure end-to-end
(T009 already unit-tests `renderRawInput` on this flag directly).
Added `dependencyStatesBundle` (all three per-package dependency states
-- `react` exact-match, `lodash` fallback-match, `leftpad`
fully-unresolved -- in one bundle, sorted alphabetically) confirming all
three coexist correctly, extending T008's per-state unit tests to a full
render. Added `runtimeVersionStatesBundle` (`VersionSourceUnknown`, no
`Version`/`Note`) -- the one `Runtime` state no fixture so far had shown
(every prior fixture used `Trace` or `LocalEnvironment`+`Note`) --
rendering the bare `(version unknown)` literal end-to-end; used Java
since `types.go`'s own doc comment on `VersionSourceTrace` states Java's
`printStackTrace()` never includes JVM version, so Java is always
`LocalEnvironment` or `Unknown` -- a real case for this state, not a
contrived one. This is the last of T012-T016's golden fixtures.
Three new `TestMarkdown` table entries, no harness changes.
**Verified:** user ran `go test ./internal/render/... -run TestMarkdown
-update` to generate all three fixtures; all hand-reviewed by Claude
against spec.md req. 18, req. 16, and req. 20 (clean -- confirmed the
truncation note's KB figure, all three dependency states' formatting and
alphabetical ordering with correct dash-escaping in both Notes, and the
bare `(version unknown)` literal); approved, then user ran `go build
./...`, `go test ./internal/render/...`, `golangci-lint run
./internal/render/...`, `gofumpt -l ./internal/render/` on their
machine; all four clean, confirmed "done, no errors".
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-19
**Task(s):** T017 — Acceptance criteria review pass
**What happened:** Re-read `spec.md`'s 16 acceptance criteria top to
bottom against T001-T016's actual tests. Found three real issues, all
surfaced by the same root cause -- T011's mid-task course-correction --
and resolved before checking any boxes:
1. Criterion 16 still said the golden test reads
   `internal/contract/testdata/example_ts.json`; nothing does anymore
   since T011 moved to a hand-authored `contract.Bundle` literal.
   Reworded to describe the actual `ts_basic` fixture.
2. `spec.md`'s Out-of-scope section still said a hand-constructed
   Java-shaped `contract.Bundle` "was considered and explicitly
   rejected" -- but T012's `noDependenciesBundle` and T016's
   `runtimeVersionStatesBundle` are both exactly that, built as a
   mechanical side effect of T011's literal-everywhere correction, with
   nobody (Claude included) noticing the collision with this earlier
   decision until this review pass. Flagged to the user with a
   recommendation (keep the fixtures -- render's Non-functional
   requirements already confine Language-dependence to the one
   fence-tag branch, T003/T005 unit-test that in isolation, and 001's
   own `exampleJavaBundle()` is repo precedent for hand-built Java
   fixtures) rather than deciding unilaterally; user confirmed. Reworded
   the bullet to describe the reconsidered position instead of deleting
   it outright, so the original caution (this isn't a stand-in for
   005a's own parser-correctness testing) stays on record.
3. Criterion 1's "(own/dependency/runtime frames in each)" parenthetical
   read ambiguously -- literally, every node would need all three bucket
   types, which no fixture does (e.g. `ts_basic`'s second node has only
   an own frame). Flagged with a recommendation (adopt the looser
   "represented across the chain" reading -- bucket-suffix rendering is
   a per-frame concern, already fully unit-tested in T006, not a
   per-node one, so requiring every node to duplicate all three types
   would add repetition, not coverage); user confirmed. Reworded the
   criterion accordingly.
   All 16 criteria then checked off with the specific `TestMarkdown`/
   unit-test name(s) satisfying each, as inline comments in `spec.md`.
   Updated `spec.md`'s header from "Planned" to "Done" and
   `specs/INDEX.md`'s 007 row from `in-progress` to `done`.
**Verified:** N/A -- doc-only changes to `spec.md`/`tasks.md`/
`specs/INDEX.md`, no code touched, no build/test/lint gate applies.
**Deviations from plan (if any):** None -- T017's own task description
anticipated exactly this outcome ("any criterion found uncovered gets
its own fixture/test added here"), though in this case the fix was
spec-wording corrections rather than new tests, since the underlying
behavior was already correctly implemented and tested; only the
documentation had drifted.
**New open questions:** None. Feature 007 is now fully done -- all 17
tasks complete, all 16 acceptance criteria checked off with named tests.

---

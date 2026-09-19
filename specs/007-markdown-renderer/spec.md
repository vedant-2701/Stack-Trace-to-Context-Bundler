# Spec: Markdown renderer

**Status:** Planned (plan.md + tasks.md complete)
**Folder:** specs/007-markdown-renderer
**Depends on:** 001-data-contract (done)

## Overview

Renders a `contract.Bundle` into a single, self-contained Markdown string —
the clipboard-ready artifact a developer pastes into a third-party AI chat
to get help with a crash. This is the second of two renderers (008 JSON is
the other); both read the same canonical `contract.Bundle` and neither
defines or modifies its shape (constitution Article IV).

The renderer's job is not to reformat the raw stack trace — it exists to
surface, visibly and in the right place, the value every preceding feature
computed: 003a/006a's cause-chain reconstruction and frame bucketing, 004's
own-code snippets and git blame, and 006b's per-package dependency
resolution (exact / inexact-fallback / conflicting / unresolved). A design
that collapses any of those distinctions back into something that looks
like a bare stack trace defeats the purpose of the features that produced
them.

Real, end-to-end bundles for v1 are TS/JS-shaped only — 005a (Java parser)
and 005b (Java dependency resolution) are both `idea` status, not built.
The renderer's logic stays language-agnostic per `contract.Bundle`'s own
shape (matching 001's non-functional requirement that the shape "read
legibly... consumed by both a JSON renderer and a Markdown renderer"), but
Java-shaped end-to-end verification is out of scope until 005a exists (see
Out of scope).

## User stories

- As a developer who just hit a crash, I want one clipboard-ready Markdown
  document — cause chain, own-code snippets with blame, resolved dependency
  versions, in one place — so I can paste it into an AI chat without
  manually assembling context myself.
- As a developer who might be looking at the same recurring bug across
  multiple pastes, I want the bundle's fingerprint visible in the output,
  so I can compare two bundles by eye and recognize "this is the same
  crash as before" even though no dedup tooling exists yet (002b/010 are
  both still `idea`).
- As the AI assistant receiving this Markdown pasted into a chat with no
  other context, I want every uncertainty the pipeline already detected
  (a stale file, an unresolved dependency version, a missing git repo, a
  degraded raw-input parse) stated explicitly at the point it applies, so
  I don't treat an unverified fact as confirmed (constitution Article VI).

## Functional requirements

### Document structure (top to bottom)

1. A one-line preamble (blockquote) orienting a reader who has no other
   context: what this document is and roughly what it contains.
2. A compact metadata bullet list, immediately after the preamble and
   before the exception chain: `Language`, `OS`, `Runtime`, `Git` (omitted
   entirely when `Bundle.GitMetadata` is nil), `Fingerprint`. Order fixed
   as listed. `Bundle.SchemaVersion` is never rendered — contract-versioning
   bookkeeping with no reader-facing use. `Bundle.Fingerprint` IS always
   rendered — per 001's own acceptance criteria, it is specifically
   designed so two bundles for the same bug (even across a dependency
   version bump) match and two bundles for different bugs differ; until
   dedup tooling exists (002b/010, both `idea`), a human visually comparing
   two rendered bundles is the only consumer of that fact.
3. The exception chain: one block per `ExceptionNode`, outermost first, in
   `Chain` order. No wrapping heading (e.g. no `## Exception chain`) — the
   per-node headings alone are sufficient, and an empty wrapping heading
   would visually separate the chain from the metadata bullets it depends
   on for full context.
4. A `## Dependencies` section, only when `Bundle.Dependencies` is non-nil
   (omitted entirely, not rendered empty, when nil).
5. A collapsed raw-input section (`<details><summary>Raw input</summary>`),
   always last, always present (`Bundle.RawInput` has no omitempty — it is
   always populated).

### Exception chain block (per `ExceptionNode`)

6. Heading is `### ClassName` only — never `ClassName: Message` on one
   line. `Message` is developer-arbitrary text with no length or newline
   restriction (confirmed real case: Node's `assert.strictEqual` embeds a
   multi-line diff directly in the message — 006a's own acceptance
   criteria test this exact shape). A single-line Markdown heading cannot
   safely hold multi-line content without silently altering what the
   exception actually said, which Article VI forbids regardless of
   whether the cause is a missing field or the field's own content.
7. `Message` is rendered as a Markdown blockquote directly below the
   heading, verbatim (Markdown-escaped per requirement 19), preserving
   internal newlines as separate blockquote lines. A wholly empty line
   within `Message` still gets its own bare `>` marker (not an actual
   blank line with no marker at all) — CommonMark ends a block quote at
   the first line lacking a `>` prefix, blank or not, so an empty line
   rendered without one would silently fracture the blockquote and leak
   the remainder of a multi-line message (e.g. a diff with a blank
   separator line) out of quote context entirely.
8. Each `Frame` in the node renders as one line:
   `at [ClassName.]MethodName (FilePath:LineNumber[:ColumnNumber]) — <bucket-suffix>`.
   - `ClassName.` prefix is included only when `Frame.ClassName` is
     present; omitted cleanly (no dangling `.`) otherwise (real case: a
     bare JS function, per contract's own doc comment on `Frame.ClassName`).
   - `:ColumnNumber` is included only when present (JS/TS only; Java never
     carries it).
   - `FilePath` is rendered verbatim (the contract's own normalized
     absolute path) — never shortened to a repo-relative path, since the
     contract provides no repo-root field to relativize against, and
     guessing one would risk exactly the fabricated-looking-precise output
     Article VI warns against.
   - `<bucket-suffix>` is `own`, `dependency: <PackageName>` (identity
     only — never a version; see requirement 13 for why), or `runtime`.
9. Immediately following an `own`-bucket frame's line: that frame's
   own-code context, per requirements 10-12 (never a separate,
   cross-referenced "Code Context" section — every own frame's context sits
   right where it's needed).
10. When the matching `CodeContext.Status` is `not_found` or `stale`: no
    snippet or blame table. Instead, a single flagged line using
    `CodeContext.Note` verbatim (escaped): `⚠ <Note>`. (001/004 guarantee
    every own-bucket frame gets exactly one `CodeContext` entry, always —
    there is no "own frame with no matching context" case to handle.)
11. When `Status` is `ok`: render `Snippet.Code` as a fenced code block
    tagged with `CodeContext.Language` (`java`/`typescript`/`javascript`),
    each line prefixed with its real line number (`Snippet.StartLine` +
    offset; not present in `Snippet.Code` itself) and the line matching
    `Snippet.TargetLine` marked with a leading `→`. Line-number prefixing
    is the one place this renderer deliberately adds text not present in
    the raw field — without it, neither the target-line marker nor the
    blame table's `Lines` column (requirement 12) has anything to
    correlate against. `internal/codecontext.buildSnippet` always builds
    `Code` as its window's real lines joined by `\n` plus exactly one
    further trailing `\n` appended unconditionally (`code :=
    strings.Join(lines, "\n"); code += "\n"`) — regardless of whether the
    window's actual last source line has content or is itself blank. A
    raw `strings.Split(Code, "\n")` therefore always yields exactly one
    MORE element than `EndLine - StartLine + 1` real lines, the last
    split element always being an empty-string artifact of that
    unconditional trailing newline, never a real line of source.
    Rendering every split element as a numbered line — the naive,
    natural-looking implementation — would append a bogus, blank
    `EndLine + 1`-numbered line to the bottom of every single rendered
    snippet, not an isolated edge case: the renderer must discard that
    final split element (or trim exactly one trailing `\n` before
    splitting) and render exactly `EndLine - StartLine + 1` numbered
    lines.
12. Directly below the snippet, when `Blame` is non-empty: a table, one row
    per `BlameEntry` (one row per contiguous range, matching how
    `git blame -L` itself groups output — never one row per line), columns
    `Lines | Commit | Author | Date | Summary`. `Commit` is the short form
    (first 7 characters of `CommitHash`). `Date` is `CommitDate`'s date
    portion only (`YYYY-MM-DD`, dropping time-of-day) — a commit's day is
    what's useful for "is this recent," not the second. When `Status` is
    `ok` but `Blame` is empty (real case per 004: no git repo found at all,
    or `git blame` itself failed/timed out despite a clean tracked file —
    `Note` is guaranteed present in both cases): render `⚠ <Note>` in place
    of the table, not silent omission.
13. Dependency-bucket frame lines show `PackageName` only, never a version
    or resolution note inline. `LockedDependency` is a per-package fact
    (006b resolves and conflict-checks it once per package, potentially
    across several frames referencing the same package) — repeating it
    per-frame-occurrence would either duplicate it redundantly or risk one
    frame silently showing a stale/incomplete view of a fact that
    genuinely has one authoritative answer per bundle. The single
    `## Dependencies` section (requirement 15) is the one place that
    per-package fact is stated.
14. After a node's frame list, when `ElidedFrameCount > 0`: one line,
    language-neutral wording — `... N more frames (shared with enclosing
    exception)` — never the language-specific literal a parser's raw
    output used (Java's `"... N more"` vs. Node's
    `"... N lines matching cause stack trace ..."`). The contract's own
    doc comment on `ElidedFrameCount` states the field "only says what the
    parsed result means once produced" — the language-specific phrasing
    was intentionally normalized away by the parser; reproducing it here
    would undo that normalization at the last step.
15. Between consecutive nodes (never after the last node): a blank line,
    the literal text `Caused by ↓`, a blank line.

### Dependencies section

16. Rendered only when `Bundle.Dependencies` is non-nil. One bullet per
    entry in `Dependencies.Locked` (already scoped by 006b to only
    packages actually referenced by a frame — never the full manifest),
    in ascending lexical order by package name (`Dependencies.Locked` is
    a Go map; Go's `range` over a map is deliberately randomized on every
    iteration, so rendering it in map-iteration order would make output
    order non-deterministic across runs and break the byte-exact
    golden-file tests this feature depends on — the package-name sort
    key is stable, requires no new field, and matches how a human
    scanning the section would expect it ordered anyway):
    `- <package> — declared <Dependencies.Direct[package] or omitted if
    absent>, resolved <Locked[package].Version, or the literal "unresolved"
    if Version is absent>` with `Locked[package].Note` appended in
    parentheses whenever present, regardless of whether `Version` is also
    set (`Note` may explain an absent version OR flag an inexact/fallback
    match on a present one, per the contract's own doc comment on
    `LockedDependency.Note` — 007 renders whichever is present, and does
    not need to know which of 006b's internal cases produced it).

### Raw input section

17. `Bundle.RawInput` is rendered verbatim inside a fenced code block
    inside the collapsed `<details>` element. The fence length used must
    exceed the longest run of consecutive backticks found anywhere within
    `RawInput` (minimum 3), so pathological input cannot corrupt the
    enclosing document structure.
18. When `Bundle.RawInputTruncated` is `true`, a note appears with the raw
    input, formatted using `contract.RawInputCapBytes` rather than a
    hardcoded number (e.g. `⚠ input truncated at the 512 KB cap`, for the
    constant's current value) — omitted entirely when `false`, consistent
    with `rawInputTruncated`'s own contract semantics (a real,
    always-relevant status, not an omission). Reading the cap from the
    exported constant, rather than restating the figure, means this note
    can't drift out of sync if the cap ever changes.

### Escaping

19. Any developer-arbitrary text interpolated into prose/heading/blockquote
    context (`ExceptionNode.Message`, `CodeContext.Note`,
    `Runtime.Note`, `LockedDependency.Note`, `BlameEntry.Author`/`Summary`,
    `GitMetadata.Branch`) must have Markdown special characters escaped
    before insertion, so that e.g. an underscore in a class name or a pipe
    character in a commit summary cannot silently corrupt heading/table
    structure or be misread as emphasis/code — the same "never present a
    guess as fact" concern (Article VI) applies to accidentally-altered
    rendering, not only to missing data. Content inside fenced code blocks
    (snippets, raw input) is never escaped — the fence already protects it
    (see requirement 17 for the one real fence-collision risk that
    remains).

### Runtime rendering

20. `Runtime` renders as one metadata-block line. When
    `VersionSource == VersionSourceTrace`: `Runtime: <Name> <Version>`, no
    extra caveat. When `VersionSource` is `LocalEnvironment` or `Unknown`:
    append `Runtime.Note`'s own text (escaped) as a parenthetical rather
    than also printing the literal enum value — the `Note` field exists
    specifically so the renderer doesn't re-derive prose from the enum
    (same "state a fact once" reasoning as requirement 13). When `Version`
    is entirely absent (`VersionSourceUnknown`, no `Note`): `Runtime:
    <Name> (version unknown)`.

### Git line rendering

21. `GitMetadata`, when present, renders as one metadata-block line:
    `Git: <Branch> @ <first 7 chars of CurrentCommit> (<clean|uncommitted
    changes>)`, derived from `UncommittedChanges`.

### Function signature

22. Public API is `func Markdown(b contract.Bundle) string` in
    `internal/render` (package `render`, file `markdown.go`, per
    `CONVENTIONS.md`'s file/folder layout). No error return: every branch
    this function takes is a data-shape check on an already-validated
    `Bundle` (nil pointers, status enums, empty slices) — nothing that can
    fail the way file I/O or a subprocess call can. Writing the returned
    string to stdout, a file, or the clipboard is 002b's concern, not
    this feature's.

## Non-functional requirements

- Pure function, no I/O, no subprocess calls — matches `Detect()`'s
  purity constraint elsewhere in this codebase (003a), for the same
  reason: nothing about turning an already-built `Bundle` into text
  should need to touch the filesystem or network.
- Output must remain valid, renderable Markdown for any `Bundle` value
  satisfying the `contract` package's own invariants — including
  pathological developer-arbitrary string content (multi-line messages,
  Markdown-special characters, embedded backtick runs) — never assume
  "normal-looking" input.
- Language-agnostic: the only line of logic that branches on
  `Bundle.Language`/`CodeContext.Language` is the fenced-code-block
  language tag (requirement 11). No other requirement in this spec
  varies by language.

## Out of scope

- Pipeline wiring / writing the rendered string to stdout, a file, or the
  clipboard — 002b (`idea`) and 009 (`idea`).
- A `--verbose` mode or any other alternate rendering mode — v1 has
  exactly one Markdown output shape.
- Full Java-shaped golden-fixture bundles for testing — no real parser
  (005a, `idea`) can produce one yet; a hand-constructed synthetic
  Java-shaped `contract.Bundle` was considered and explicitly rejected
  (see progress.md) in favor of a narrow unit test covering only the one
  genuinely language-dependent branch (the fenced-code-block language
  tag, requirement 11) using a bare `contract.Snippet` value and a
  `contract.Language` of `"java"`, not a full bundle.
- Monorepo/multi-repo bundles, branching (`AggregateError`/`Suppressed`)
  chains — both already out of scope at the contract level (001), so
  nothing here needs to render them.
- Configurable snippet context window (011, `idea`) — this renderer
  displays whatever `Snippet` the bundle already contains; it does not
  control the window size.

## Acceptance criteria

- [ ] Given a `Bundle` with a two-node linear chain (own/dependency/runtime
      frames in each), when rendered, then the output shows each node as
      an independent `### ClassName` block with its `Message` in a
      blockquote below, separated by a `Caused by ↓` transition, in
      `Chain` order.
- [ ] Given an `own`-bucket frame with `CodeContext.Status == "ok"` and
      non-empty `Blame`, when rendered, then the frame's line is
      immediately followed by a line-numbered fenced snippet (target line
      marked `→`) and a blame table with one row per `BlameEntry`.
- [ ] Given an `own`-bucket frame with `CodeContext.Status == "ok"` and
      empty `Blame` (e.g. no git repo found), when rendered, then the
      snippet still renders, and `⚠ <Note>` appears in place of a blame
      table.
- [ ] Given an `own`-bucket frame with `CodeContext.Status` of
      `"not_found"` or `"stale"`, when rendered, then no snippet or blame
      table appears — only the frame line followed by `⚠ <Note>`.
- [ ] Given a `dependency`-bucket frame, when rendered, then its line shows
      only `PackageName` (no version, no note) — resolution detail appears
      exclusively in the `## Dependencies` section.
- [ ] Given `Bundle.Dependencies` with one exact-match package
      (`Locked[pkg].Version` set, no `Note`) and one fully-unresolved
      package (`Version` absent, `Note` present), when rendered, then the
      `## Dependencies` section shows the exact package with just its
      resolved version, and the unresolved package with the literal
      `unresolved` plus its `Note` text.
- [ ] Given `Bundle.Dependencies == nil`, when rendered, then no
      `## Dependencies` heading or content appears anywhere in the output.
- [ ] Given `Bundle.GitMetadata == nil`, when rendered, then no `Git:` line
      appears in the metadata block (no placeholder).
- [ ] Given an `ExceptionNode` with `ElidedFrameCount > 0`, when rendered,
      then a single language-neutral `... N more frames (shared with
      enclosing exception)` line appears after that node's frames —
      regardless of whether the source language was Java or JS/TS.
- [ ] Given `Runtime.VersionSource == "trace"`, when rendered, then the
      `Runtime:` line shows name+version with no parenthetical caveat.
      Given `VersionSource` of `"local-environment"` or `"unknown"`, when
      rendered, then `Runtime.Note`'s text (when present) appears as a
      parenthetical, and the literal enum value itself never appears in
      the output.
- [ ] Given an `ExceptionNode.Message` containing embedded newlines (e.g. a
      multi-line assertion diff), when rendered, then the heading contains
      only `ClassName`, and the full multi-line message appears intact in
      the blockquote below it — no content dropped or merged onto one
      line.
- [ ] Given a message, note, or commit summary containing Markdown special
      characters (e.g. `_`, `` ` ``, `|`, `#`, `<` as in a generic type
      like `List<String>`), when rendered, then those characters are
      escaped in prose context and do not corrupt surrounding
      heading/table/blockquote structure.
- [ ] Given `Bundle.RawInput` containing a run of backticks as long as or
      longer than 3, when rendered, then the raw-input fenced code block
      uses a longer fence and remains valid, unbroken Markdown.
- [ ] Given `Bundle.RawInputTruncated == true`, when rendered, then a
      truncation note appears alongside the raw input section; given
      `false`, no such note appears.
- [ ] Given a `CodeContext.Language` of `"java"` on an otherwise-ordinary
      `ok`-status snippet, when rendered, then the fenced snippet block
      uses a `java` fence tag rather than `typescript`/`javascript` — the
      one genuinely language-dependent behavior this renderer has,
      verified without needing a full Java-shaped bundle.
- [ ] Given the checked-in `internal/contract/testdata/example_ts.json`
      fixture, when rendered, then the output matches a checked-in golden
      Markdown fixture exactly (golden-file test, per `CONVENTIONS.md`'s
      testing section for `internal/render/*`).

## Open questions

None — all resolved during interrogation. Full session log in
`specs/007-markdown-renderer/progress.md`.

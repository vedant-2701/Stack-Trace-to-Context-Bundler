# Tasks: Markdown renderer

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.
Each task lists the `spec.md` functional requirements it satisfies, so a
failing acceptance criterion later traces back to exactly one task.

- [x] **T001 — Package scaffold + `escapeMarkdown`** (spec.md req. 19,
      plan.md's escaping character set)
      - Depends on: none
      Create `internal/render/markdown.go` (package doc comment + stub
      `func Markdown(b contract.Bundle) string { return "" }`) and
      `internal/render/escape.go` (`escapeMarkdown(s string) string`,
      the curated ASCII-punctuation escape set from plan.md, including
      `<`) + `escape_test.go` (table-driven, per-character + combined +
      no-op cases, including a `List<String>`-style generic-type case
      covering `<`/`>` together). No other helper exists yet — this is
      the one piece with no dependency on anything else in the package.
      - Acceptance: `escape_test.go` passes, including the `<`/`>`
      generic-type case; `go build`, `go test ./internal/render/...`,
      `golangci-lint run ./internal/render/...`, `gofumpt -l` all clean.

- [x] **T002 — `renderMetadata`** (spec.md req. 2, 20, 21)
      - Depends on: T001 (`Runtime.Note` and `GitMetadata.Branch` must
      be passed through `escapeMarkdown`)
      Implement `renderPreamble` (static string, req. 1) and
      `renderMetadata` (Language/OS/Runtime/Git/Fingerprint bullet list).
      Unit tests with hand-built `contract.Bundle` values (not golden
      files yet — these are still sub-component tests): full metadata
      present; `GitMetadata == nil` (Git line omitted); each
      `Runtime.VersionSource` case (`trace` / `local-environment` /
      `unknown`, with and without `Runtime.Note`).
      - Acceptance: unit tests pass, including a case with Markdown
      special characters in `Runtime.Note`/`GitMetadata.Branch` proving
      they come out escaped; 4 verification commands clean.

- [x] **T003 — `renderSnippet`** (spec.md req. 11, plan.md's language-tag
      unit test)
      - Depends on: none (no dependency on T001 — snippet content is
      never escaped, per req. 19)
      Implement line-numbered, target-marked, language-fence-tagged
      snippet rendering from a `contract.Snippet` + `contract.Language`.
      Build the test `Snippet.Code` values the same way
      `internal/codecontext.buildSnippet` actually does — real lines
      joined by `\n` plus one further trailing `\n` appended
      unconditionally — never a hand-typed string without that trailing
      newline, or the test would pass without ever exercising the real
      off-by-one risk (a raw split yields one more element than real
      lines; rendering that extra element appends a bogus blank
      `EndLine + 1` line to every snippet, not an edge case). Assert the
      rendered snippet has exactly `EndLine - StartLine + 1` numbered
      lines, no trailing blank one. Includes the one narrow non-golden
      unit test: a bare `contract.Snippet{...}` passed alongside
      `contract.LanguageJava` asserting the fence tag is ` ```java ` —
      the only place this feature's tests touch Java at the
      `renderSnippet` level in isolation (T005 adds a second,
      integration-level Java case one layer up, for a different reason).
      - Acceptance: unit tests pass including the `java` fence-tag case
      and the exact-line-count assertion; 4 verification commands clean.

- [x] **T004 — `renderBlameTable`** (spec.md req. 12)
      - Depends on: T001 (`BlameEntry.Author`/`Summary` must be passed
      through `escapeMarkdown`)
      Implement the `Lines | Commit | Author | Date | Summary` table from
      `[]contract.BlameEntry` — short (7-char) commit hash, date-only
      (`YYYY-MM-DD`) formatting, one row per entry (no per-line
      expansion). Unit test with a single-entry and a multi-entry (two
      different commits in one window) case.
      - Acceptance: unit tests pass, including a case with a `|` in
      `Summary` proving it doesn't corrupt the table; 4 verification
      commands clean.

- [x] **T005 — `renderCodeContext`** (spec.md req. 10-12, combining T003+T004)
      - Depends on: T001, T003, T004
      Implement full `CodeContext` status branching: `not_found`/`stale`
      → flagged `⚠ <Note>` line only; `ok` + non-empty `Blame` → snippet +
      blame table; `ok` + empty `Blame` (`Note` present) → snippet + `⚠
      <Note>` in place of the table. Unit tests for all four branches
      using hand-built `contract.CodeContext` values, plus one further
      `ok`-status case with `Language: contract.LanguageJava` set
      asserting the rendered snippet opens with ` ```java `. T003 already
      proves `renderSnippet` handles `LanguageJava` correctly in
      isolation; nothing before this task proves `CodeContext.Language`
      actually reaches `renderSnippet` through `renderCodeContext`'s real
      call path, so a hardcoded language tag there would otherwise pass
      every other planned test. Still no full Java-shaped `Bundle`
      (spec.md's Out of scope) — just this one `CodeContext` value.
      - Acceptance: all four status-branch unit tests pass plus the
      Java-language pass-through case; 4 verification commands clean.

- [x] **T006 — `renderFrame`** (spec.md req. 8-9, 13)
      - Depends on: T005
      Implement per-frame line rendering: `ClassName.`-prefix omission
      rule, `:ColumnNumber` inclusion rule, verbatim `FilePath`, the three
      bucket suffixes (`own` wires in `renderCodeContext` from T005 via a
      `*contract.CodeContext` parameter; `dependency` shows `PackageName`
      only, no version; `runtime` has no suffix detail beyond the label).
      Unit tests: one per bucket, plus the `ClassName` omitted/present
      cases and the `ColumnNumber` present/absent cases.
      - Acceptance: unit tests pass for all three buckets and both
      omission cases; 4 verification commands clean.

- [x] **T007 — `renderChain`** (spec.md req. 3, 6, 7, 14, 15)
      - Depends on: T001 (`ExceptionNode.Message` escaping), T006
      Implement the `contract.FrameRef`-map lookup (plan.md's data model)
      from `CodeContexts` to their owning frame, the per-node
      heading+blockquote (multi-line message preserved as multiple
      blockquote lines), the elided-frames line (language-neutral
      wording, only when `ElidedFrameCount > 0`), and the `Caused by ↓`
      transition between nodes (never after the last). The map key's
      `FrameIndex` is each frame's position within `node.Frames` (the
      range-loop index) — matching `codecontext.buildCodeContexts`'s own
      construction — not read back from `Frame.Index`'s field value.
      `001-data-contract` now states these as contractually equal
      (`Frame.Index`/`FrameRef` doc comments), so this is defense in
      depth against a future contract violation, not a live ambiguity
      (plan.md's Architecture section has the full reasoning). Unit
      tests: single node, two-node chain, a node with
      `ElidedFrameCount == 0` (no line rendered) vs `> 0` (line
      rendered), a multi-line `Message`, and a case where a frame's
      `Index` field is deliberately set to a different value than its
      slice position, proving the lookup still finds the right
      `CodeContext` by position rather than by the `Index` field.
      - Acceptance: all unit tests pass, including the
      Index-vs-position case; 4 verification commands clean.

- [x] **T008 — `renderDependencies`** (spec.md req. 4, 16)
      - Depends on: T001 (`LockedDependency.Note` escaping)
      Implement the `## Dependencies` section from `*contract.Dependencies`
      (nil → `""`). `d.Locked` is a Go map, and Go randomizes `range`
      order over maps on every iteration (not a one-time, per-instance
      randomization) — so this MUST sort `Locked`'s keys (ascending
      lexical by package name, spec.md req. 16) before iterating, never
      a bare `range d.Locked`, or the golden-file tests in T011/T016 will
      be flaky rather than reliably passing or failing. Unit tests:
      exact-match package (version only, no note), fallback-match
      package (version + note), fully-unresolved package (`unresolved`
      literal + note), the nil-pointer case, and a `Locked` map with 3+
      entries rendered several times in the same test run asserting
      identical, alphabetically-ordered output every time (catches a
      bare `range` regression that individual single/small-map test
      cases could pass by luck).
      - Acceptance: unit tests pass for all five cases, including the
      repeated-render determinism check; 4 verification commands clean.

- [x] **T009 — `renderRawInput`** (spec.md req. 5, 17, 18)
      - Depends on: none (`RawInput` is never escaped — fenced content is
      exempt per req. 19)
      Implement the collapsed `<details>` block: fence-length calculation
      (scan `RawInput` for the longest backtick run, use one longer,
      minimum 3), and the truncation note (present only when
      `RawInputTruncated == true`). Unit tests: plain input (fence length
      3), input containing a 3-backtick run (fence length 4+), truncated
      vs. not.
      - Acceptance: unit tests pass for all four cases; 4 verification
      commands clean.

- [x] **T010 — `Markdown()` composition**
      - Depends on: T002, T003, T004, T005, T006, T007, T008, T009
      Wire T002-T009's helpers together in `Markdown(b contract.Bundle)
      string`, in the document order spec.md req. 1-5 specifies. No new
      rendering logic here — pure composition.
      - Acceptance: `go build ./...` and `go vet ./...` clean; no test
      added in this task (T011 covers the first real end-to-end
      assertion).

- [ ] **T011 — Golden fixture: `ts_basic`** (first end-to-end proof)
      - Depends on: T010
      Add the golden-file test harness (table-driven, `-update` flag per
      plan.md) and the first fixture, reusing
      `internal/contract/testdata/example_ts.json` verbatim as input.
      Hand-verify the generated `.golden.md` once by eye against
      `spec.md`'s rendered mockup from interrogation before committing it.
      - Acceptance: `TestMarkdown/ts_basic` (or equivalent) passes;
      golden file hand-reviewed and committed; 4 verification commands
      clean.

- [ ] **T012 — Golden fixtures: `no_git_metadata`, `no_dependencies`**
      - Depends on: T011
      (spec.md acceptance criteria for nil `GitMetadata`/`Dependencies`)
      - Acceptance: both golden tests pass; 4 verification commands
      clean.

- [ ] **T013 — Golden fixtures: `code_context_not_found`,
      `code_context_stale`, `code_context_ok_no_blame`**
      - Depends on: T011
      (spec.md acceptance criteria for all three non-table `CodeContext`
      outcomes)
      - Acceptance: all three golden tests pass; 4 verification commands
      clean.

- [ ] **T014 — Golden fixtures: `elided_frames`, `multiline_message`**
      - Depends on: T011
      (spec.md acceptance criteria for req. 14 and req. 6-7).
      `multiline_message`'s `Message` must include at least one wholly
      empty line in the middle (not just non-empty lines), asserting the
      blockquote keeps its `>` marker on that line rather than emitting
      an actual blank line that would fracture the blockquote (req. 7).
      - Acceptance: both golden tests pass, including the embedded-blank-
      line case staying inside one continuous blockquote; 4 verification
      commands clean.

- [ ] **T015 — Golden fixtures: `markdown_special_chars`,
      `backtick_run_in_raw_input`**
      - Depends on: T011
      (spec.md acceptance criteria for req. 19 and req. 17).
      `markdown_special_chars` must include a generic-type case (e.g.
      `List<String>` or `Map<string, number>`) in `Message` or `Note`,
      covering `<`/`>` together, not just the original `_`/`` ` ``/`|`/`#`
      set.
      - Acceptance: both golden tests pass, including the generic-type
      `<`/`>` case; 4 verification commands clean.

- [ ] **T016 — Golden fixtures: `raw_input_truncated`,
      `dependency_states`, `runtime_version_states`**
      - Depends on: T011
      (spec.md acceptance criteria for req. 18, req. 16, req. 20)
      - Acceptance: all three golden tests pass; 4 verification commands
      clean.

- [ ] **T017 — Acceptance criteria review pass**
      - Depends on: T001-T016
      Re-read `spec.md`'s Acceptance criteria list top to bottom; confirm
      each has an exact corresponding passing test from T001-T016 (record
      which test in a comment next to each checkbox in `spec.md`, check
      the box). Any criterion found uncovered gets its own fixture/test
      added here before this task is marked done — this task is not
      "done" until every acceptance criterion in `spec.md` is checked off
      with a named test.
      - Acceptance: every acceptance criterion in `spec.md` checked off
      with a named test; `specs/INDEX.md`'s 007 row updated to `done`.

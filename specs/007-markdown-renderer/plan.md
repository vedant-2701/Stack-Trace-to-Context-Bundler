# Plan: Markdown renderer

Derived from `spec.md`. Consistent with `memory/constitution.md` (Articles
II, IV, V, VI, VIII) and `CONVENTIONS.md`.

## Architecture / approach

One pure function, `render.Markdown(b contract.Bundle) string`, built by
string concatenation via a `strings.Builder` — no templating library
(Article VIII: nothing here needs text/template's conditionals/loops badly
enough to justify a new dependency; Go's own string formatting is
sufficient and keeps every branch in requirement-traceable Go code rather
than template syntax).

Internal helpers (unexported, each independently unit-testable), one per
spec.md requirement group:

- `renderPreamble() string` — static string (requirement 1).
- `renderMetadata(b contract.Bundle) string` — requirements 2, 20, 21.
- `renderChain(chain []contract.ExceptionNode, codeContexts []contract.CodeContext) string` — requirements 3, 6-8, 14-15; looks up each frame's `CodeContext` via a `map[contract.FrameRef]contract.CodeContext` built once up front (avoids an O(n²) scan per frame). As of `001-data-contract`'s `Frame.Index`/`FrameRef` doc comments (2026-09-19), the contract now states as a required invariant that every parser assigns `Frame.Index` to exactly the frame's zero-based position within `node.Frames`, so `FrameRef.FrameIndex` is guaranteed equal to the referenced frame's `Index` field. `renderChain` should still construct the lookup key's `FrameIndex` from each frame's loop position while ranging over `node.Frames` — matching exactly how `internal/codecontext.buildCodeContexts` constructs it — rather than reading `Frame.Index` back off the struct: this needs no dependency on the field's value at all, and keeps `renderChain` correct even against a hypothetical future `Bundle` that violates the invariant, rather than trusting a doc comment a producer could fail to honor. T007's test suite includes a case that deliberately sets a frame's `Index` field to a different value than its slice position, asserting the lookup still resolves by position — this is a defense-in-depth test against a contract violation, not a demonstration of an unresolved ambiguity (the ambiguity itself is now resolved at the contract level).
- `renderFrame(f contract.Frame, cc *contract.CodeContext) string` — requirements 8-13 (`cc` is nil for non-own frames).
- `renderCodeContext(cc contract.CodeContext) string` — requirements 10-12.
- `renderSnippet(s contract.Snippet, lang contract.Language) string` — requirement 11. Splits `s.Code` on `"\n"` and discards the final split element unconditionally (or trims exactly one trailing `\n` before splitting) — `internal/codecontext.buildSnippet` always appends a trailing `\n` after its real lines regardless of whether the window's last source line is blank, so a raw split always has one more element than real lines; rendering every split element would append a bogus blank `EndLine + 1` line to every snippet, not an edge case.
- `renderBlameTable(entries []contract.BlameEntry) string` — requirement 12.
- `renderDependencies(d *contract.Dependencies) string` — requirements 4, 16 (returns `""` when `d == nil`). Iterates `d.Locked` via a sorted copy of its keys (`sort.Strings` over `maps.Keys(d.Locked)` or an equivalent explicit sort), never a bare `range d.Locked` — Go's map iteration order is randomized per iteration (not merely map-instance-specific), so an unsorted range would make this section's bullet order flap across test runs and break `markdown_test.go`'s byte-exact golden comparisons the first time `Locked` has 2+ entries (which `ts_basic`, the very first golden fixture, already does: `express` + `lodash`).
- `renderRawInput(raw string, truncated bool) string` — requirements 5, 17-18.
- `escapeMarkdown(s string) string` — requirement 19.

`Markdown` itself is a thin composition of these calls in document order —
kept small enough to read as a table of contents for the whole output.

## Stack & versions

No new dependencies. Standard library only: `strings`, `fmt`,
`strconv`. Matches `AGENTS.md`'s stdlib-first stance and Article VIII —
nothing in this feature's scope needs a Markdown-building library.

## Data model

Consumes `contract.Bundle` as-is; defines no new exported types, and no
unexported helper types either. `contract.FrameRef` (`ChainIndex`,
`FrameIndex`, both plain `int`) is already exported and directly
comparable, so `renderChain`'s `map[contract.FrameRef]contract.CodeContext`
(see Architecture section above) uses it as the map key as-is — no
mirror type needed.

## File / module layout

```
internal/render/
├── markdown.go       Markdown() + all unexported render* helpers
├── markdown_test.go  golden-file tests + the language-fence-tag unit test
├── escape.go         escapeMarkdown() + its own focused test
├── escape_test.go
├── fixtures_test.go  one contract.Bundle-building func per golden case below
│                     (except ts_basic, which unmarshals the existing
│                     internal/contract/testdata/example_ts.json --
│                     already machine-generated by 001's own golden test,
│                     not hand-written, so reusing it as input carries no
│                     Article IV risk)
└── testdata/
    └── golden/
        ├── ts_basic.golden.md
        ├── no_git_metadata.golden.md
        ├── no_dependencies.golden.md
        ├── code_context_not_found.golden.md
        ├── code_context_stale.golden.md
        ├── code_context_ok_no_blame.golden.md   -- Status:"ok", empty Blame, Note present (no repo / blame failed)
        ├── elided_frames.golden.md
        ├── multiline_message.golden.md          -- assert.strictEqual-style diff message
        ├── markdown_special_chars.golden.md     -- _, `, |, # in message/note/summary/author
        ├── backtick_run_in_raw_input.golden.md  -- RawInput containing a ``` run
        ├── raw_input_truncated.golden.md
        ├── dependency_states.golden.md          -- one exact, one fallback+note, one unresolved
        └── runtime_version_states.golden.md     -- trace / local-environment / unknown, each
```

Fixture inputs are `contract.Bundle{...}` Go struct literals (one
unexported builder function per case, in `fixtures_test.go`), not
hand-written JSON -- matching the pattern `internal/contract`'s own
`types_test.go` already established (`exampleJavaBundle()`/
`exampleTSBundle()`, each passed straight into a golden-comparison
helper). A field rename in `types.go` breaks these at compile time, where
a hand-written JSON fixture would only fail -- or silently pass with a
zero value -- at test run time. This also resolves the earlier plan's
tension with constitution Article IV ("no second copy of the [bundle]
shape anywhere else in the repo... never hand-written in parallel"): a
`contract.Bundle{...}` literal is an instance of the one canonical type,
not a second copy of its shape. Only the checked-in `*.golden.md` files
stay as hand-authored text -- Article IV governs the Bundle shape, not
the Markdown this feature produces from it.
`exampleTSBundle()` itself lives in `internal/contract`'s `_test.go` file
and can't be imported across packages, so `ts_basic` instead unmarshals
the already-generated `internal/contract/testdata/example_ts.json` into a
`contract.Bundle` at test time -- consuming 001's canonical generated
artifact, not re-authoring it.

Each fixture is scoped to exercise exactly one requirement group listed
above — not combined into fewer, denser fixtures — so a future regression
in, say, dependency rendering fails exactly the `dependency_states` golden
test and nothing else, matching `CONVENTIONS.md`'s stated reason for
preferring golden-file tests here ("a diff is easier to review than a
pile of field-by-field assertions") — that only holds if each diff is
already narrowed to one concern before it happens.

## API / contracts

```go
package render

// Markdown renders b as a single, self-contained Markdown document.
func Markdown(b contract.Bundle) string
```

No other exported surface. `internal/render/json.go` (008, separate
feature, not touched here) will live alongside this file per
`CONVENTIONS.md`'s file/folder layout but shares no code with it beyond
the `contract` package itself.

## Testing strategy

- **Golden-file tests** (`markdown_test.go`), one per fixture in the table
  above — table-driven, each entry naming its `fixtures_test.go` builder
  function (or, for `ts_basic`, the `example_ts.json` path to unmarshal)
  and its golden path, per `CONVENTIONS.md`. A `go test
  ./internal/render/... -update` flag regenerates golden files after an
  intentional output-format change (matching the pattern already
  established in `internal/contract` per 001's plan.md).
- **One narrow unit test**, not a golden file, for requirement 11's
  language-fence-tag branch: constructs a bare `contract.Snippet{...}`
  and passes `contract.LanguageJava` as `renderSnippet`'s second argument
  (not a full `Bundle`, not a `CodeContext` -- `renderSnippet`'s own
  signature above takes exactly these two values) and asserts the
  rendered snippet opens with ` ```java `. This is the single place Java
  is touched in this feature's tests at all -- no full Java-shaped
  bundle, synthetic or otherwise (spec.md's Out of scope).
- **`escapeMarkdown` unit tests** (`escape_test.go`): table-driven,
  covering each character in the escaped set individually and in
  combination, plus a no-op case (plain text unchanged) and an
  already-fenced-code-block case is explicitly NOT covered here since
  `escapeMarkdown` is never called on fenced content (requirement 19) —
  that non-call is asserted at the `markdown_test.go` golden-file level
  instead (the snippet/raw-input golden fixtures contain characters that
  would be escaped in prose but must appear unescaped inside their
  fences).
- No fakes/mocks needed — `Markdown` is pure, no subprocess or filesystem
  interaction, unlike `internal/codecontext`'s git-shelling tests.

## Escaping character set (resolves spec.md requirement 19 to concrete implementation)

`escapeMarkdown` prefixes each of the following with a backslash:
`` \ ` * _ { } [ ] ( ) # + - . ! | > < `` — a curated subset of
CommonMark's full ASCII-punctuation escape set (the full set also
includes `" $ % & ' , / : ; = ? @ ^ ~`, none of which affect this
document's Markdown constructs — headings, emphasis, links, code spans,
tables, blockquotes). `<` is included alongside its pair `>` (used for
blockquote escaping) specifically because an unescaped `<` can be read
as opening raw/unrecognized HTML by CommonMark-compliant renderers
(CommonMark's own spec uses `\<br/>` → "not a tag" as its canonical
example of this exact escape) — a real risk here given `Message` is
where Java/TS generic-type syntax (`List<String>`, `Map<string,
number>`) and diff-style assertion output are likely to land. Applied
to: `ExceptionNode.Message`,
`CodeContext.Note`, `Runtime.Note`, `LockedDependency.Note`,
`BlameEntry.Author`, `BlameEntry.Summary`, `GitMetadata.Branch`. Newlines
within `Message` are preserved as-is and handled at the blockquote-joining
level (each line gets its own `> ` prefix), not by `escapeMarkdown` itself.

## Risks & open decisions

- **Risk:** a future contract field added to `contract.Bundle` (e.g. by
  005a/005b landing) could go unrendered if this feature's per-field
  requirement list isn't revisited. Mitigation: none built into the code
  itself (Article VIII — no speculative "render any unknown field"
  generic fallback); instead, `known-gaps.md` should get an entry when
  005a/005b land, flagging that 007's spec should be re-checked against
  any new `contract` fields those features introduce.
- **Open decision, deferred, not blocking:** whether a future `--verbose`
  mode (out of scope here) would reuse these same helpers or need
  different ones — not decided now since the mode itself doesn't exist
  yet (Article VIII).

## Alternatives considered

- **`text/template`-based rendering** — rejected: no looping/conditional
  complexity here that stdlib string-building can't express directly, and
  a new dependency isn't justified for it (Article VIII).
- **A single monolithic fixture exercising every edge case at once** —
  rejected in favor of one fixture per concern (see File/module layout)
  for diff-locality reasons.
- **Cross-referenced "Code Context" appendix instead of inline
  snippets** — rejected during interrogation (spec.md requirement 9's
  rationale): forces exactly the frame-to-context lookup this tool exists
  to eliminate.
- **Per-frame inline dependency version** — rejected during interrogation
  (spec.md requirement 13's rationale): duplicates or corrupts a fact
  that is genuinely per-package, not per-frame.

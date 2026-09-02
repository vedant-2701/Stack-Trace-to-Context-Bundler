# Plan: TypeScript/JS parser

Derived from `spec.md`. Consistent with `memory/constitution.md` and
`CONVENTIONS.md`.

## Architecture / approach

One unexported internal parse engine, wrapped by two exported
`LanguageParser` values (`javascriptParser`, `typescriptParser`) in the
same package -- matches `CONVENTIONS.md`'s existing file-layout plan
(single `internal/parser/typescript/` folder), and avoids reopening
`003a`'s interface (`Language()` is static-per-value; two values is how
a per-trace-determined language is expressed without changing the
interface).

Parse pipeline, in order:
1. Split into logical sections: optional source-line+caret preamble,
   primary error block, zero or more `[cause]:`-nested error blocks,
   optional trailing `Node.js vX.Y.Z` line. (Shape (c), bare `.stack`,
   skips straight to a single primary block -- no brackets, no preamble,
   no trailing version line, detected by absence of the `[cause]:`
   marker and any brace structure.)
2. Per block: parse the `<ClassName>: <message>` header line, then
   consume frame lines matching the V8 pattern until a
   non-frame/blank/closing-brace/next-`[cause]:` line ends the block.
   The frame-line matcher strips leading whitespace rather than anchoring
   to a fixed column count (confirmed against real captures: a
   `[cause]:`-nested block's frames are indented 6 spaces vs. the
   enclosing block's 4, not a fixed 4 throughout -- see spec.md FR4,
   corrected during technical design review), and tolerates a trailing
   `{` on the last frame line of a block that's followed by `[cause]:`
   (e.g. `at node:internal/main/run_main_module:33:47 {`).

   A block's frame list may be followed by one or more `key: value,`
   property lines before the closing `}` (system-error properties like
   `errno`/`code`/`syscall`, or arbitrary properties on the top-level
   exception itself, e.g. `code: 'GenericFailure'`) -- these are
   recognized as block content and dropped, not mistaken for a new frame
   line or a parse error (see spec.md FR7).

   A block whose brace body contains `[errors]:` instead of `[cause]:`
   (an `AggregateError`) is treated as terminal -- the array is not
   walked into as further blocks; that block becomes a single
   `ExceptionNode` and parsing stops there (see spec.md Out of scope).
3. Elision-line detection (`... N lines matching cause stack trace
   ...`) is a per-block frame-line variant, not a separate pass --
   replaces exactly one position in that block's frame list with the
   parsed `N`, feeding `ExceptionNode.ElidedFrameCount`.
4. Bucketing and `PackageName`/`file://`-normalization run per-frame,
   after all blocks are parsed, since bucketing rules don't depend on
   block/chain position.
5. Runtime/version detection runs once per `Parse()` call, independent
   of the chain-parsing above (trailing-version-line check first, local
   shell-out fallback second).

## Stack & versions

No new third-party dependencies (constitution: new deps require
Article VII/VIII review). Standard library only: `regexp`, `strings`,
`os/exec`, `context`, `log/slog`.

## Data model

No changes to `internal/contract` beyond the `ElidedFrameCount` comment
correction already applied. This feature populates, never modifies,
`contract.ExceptionNode`, `contract.Frame`, `contract.Runtime` (Article
IV: contract is the single source of truth, parsers don't redefine it).

## File / module layout

```
internal/parser/typescript/
├── typescript.go       # exported javascriptParser, typescriptParser
│                        # (Language(), Detect(), Parse() per value);
│                        # thin wrappers over the shared engine below
├── engine.go            # unexported shared parse pipeline (steps 1-3
│                         # above): section splitting, header/frame-line
│                         # parsing, elision handling -> []contract.ExceptionNode
├── bucket.go             # frame bucketing + PackageName extraction +
│                          # file:// normalization (step 4 above)
├── runtime.go             # trailing-version-line detection + local
│                           # `node --version` shell-out fallback (step 5)
├── typescript_test.go     # javascriptParser/typescriptParser-level tests
├── engine_test.go         # chain/elision parsing table tests
├── bucket_test.go         # bucketing edge cases
├── runtime_test.go        # version-detection tests (fake exec, per
│                           # runner_fake_test.go's existing pattern in
│                           # internal/codecontext)
└── testdata/
    └── (copied from specs/006a-ts-js-parser/testdata/, plus synthetic
        fixtures for edge cases not naturally producible: scoped
        packages, nested node_modules, bundled paths, truncated/
        malformed traces, a genuinely-garbled/unparseable trace,
        zero-frame errors, browser-trace false-positive checks)
```

## API / contracts

```go
// typescript.go
package typescript

func NewJavaScriptParser() parser.LanguageParser { return javascriptParser{} }
func NewTypeScriptParser() parser.LanguageParser { return typescriptParser{} }

type javascriptParser struct{}
func (javascriptParser) Language() contract.Language { return contract.LanguageJavaScript }
func (javascriptParser) Detect(rawTrace string) bool {
    return looksLikeNodeTrace(rawTrace) && !hasTSExtensionFrame(rawTrace)
}
func (javascriptParser) Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error) {
    return parseEngine(ctx, rawTrace)
}

type typescriptParser struct{}
func (typescriptParser) Language() contract.Language { return contract.LanguageTypeScript }
func (typescriptParser) Detect(rawTrace string) bool {
    return looksLikeNodeTrace(rawTrace) && hasTSExtensionFrame(rawTrace)
}
func (typescriptParser) Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error) {
    return parseEngine(ctx, rawTrace) // identical body -- language only
                                       // affects Detect(), not parsing
}
```

Note: `Detect()`'s split (`hasTSExtensionFrame`) means exactly one of
`javascriptParser`/`typescriptParser` will match a given real trace, per
FR6 -- 003b's registry (once built) won't see both claim the same input.

## Testing strategy

- Table-driven tests against the real captured fixtures in `testdata/`
  as the primary correctness baseline -- not hand-constructed examples,
  per the lesson from `003a`'s original (incorrect) hand-traced
  pseudocode. T001 adds at least one further real capture with 3+ levels
  of `.cause` nesting (the original four captures only exercise a
  2-level chain), specifically to confirm empirically whether
  indentation keeps growing by 2 spaces per nesting level and where the
  trailing-`{` pattern lands, rather than assuming the 2-level case
  generalizes.
- Synthetic fixtures for cases the real captures don't cover: scoped
  `node_modules` packages, nested `node_modules`, bundled/minified
  single-file paths, a browser-trace false-positive check (V8 grammar,
  no Node-specific signal), truncated-mid-line input, cut-off `[cause]:`
  block, zero-frame error, `file://`-prefixed ESM frame path, and a
  genuinely-garbled/unparseable input (superficially matches the V8
  frame-line pattern but has no valid Error header and no usable frame at
  all) -- needed for T009's `ErrUnparseable` acceptance criterion; none of
  the other fixtures exercise this path.
- Real fixtures added during this amendment round (all in
  `specs/006a-ts-js-parser/testdata/`, to be copied into the package's
  own `testdata/` during T001): the async/unhandled-rejection preamble
  variant, native TypeScript execution (no transformer frame), extra
  brace-body properties on both a nested `[cause]` and a top-level
  exception, the `@swc/core` message-only false `"Caused by:"` case, the
  `AggregateError`/`[errors]:` degraded-parse case, and `assert`'s
  multi-line diff message as a frame-boundary stress test. See
  `testdata/CAPTURE-FINDINGS.md` for the full capture round writeup.
- `runtime_test.go` fakes the `node --version` subprocess call rather
  than actually shelling out in tests, mirroring
  `internal/codecontext/runner_fake_test.go`'s existing pattern for git.
- Every acceptance criterion in `spec.md` maps to at least one test case.

## Risks & open decisions

- **Bundled/minified-without-source-maps default-to-`BucketOwn`** (FR11)
  is a known, accepted misclassification source, not a solved problem --
  flagged in `spec.md`'s Out of scope and to be added to
  `memory/known-gaps.md` when implementation starts.
- **`Detect()`'s Node-specific-signal requirement (FR4)** is a heuristic,
  not a guarantee -- a sufficiently unusual real Node trace (e.g. every
  frame elided, zero `node:`/`node_modules` frames, no version line, not
  a crash) could theoretically fail to match. Judged low-risk (real
  traces bottom out in module-loading internals) but not proven
  exhaustively; revisit if real-world false negatives surface.
- **Local-environment shell-out timeout value (2s, FR15)** is a
  judgment call, not measured -- may need adjustment based on real
  observed `node --version` latency across platforms once implemented.
- **Frame ordering (FR21, `Frames[0]` = originating frame)** is satisfied
  by construction and checked by inspection against the real fixtures,
  but has no explicit test yet. **Flagged for reverification during the
  technical design review pass** -- confirm `engine.go`'s actual
  frame-collection order against every real fixture (including the
  elided-frames case) and add an explicit assertion for it, per spec.md's
  FR21 and its corresponding acceptance criterion.

## Alternatives considered

- **Reopening 003a to move `Language()` into `Parse()`'s return** --
  rejected (see `specs/006a-ts-js-parser/progress.md`): the two-value
  approach achieves the same per-trace result without touching a `done`,
  merged feature's shipped interface.
- **`"Caused by:"`-style cause-chain format** (matching Java's
  convention, and what `003a`'s original hand-traced pseudocode
  assumed) -- rejected after verifying against real Node 24.18.0 output
  and Node core's actual shipped implementation
  (`util.inspect`'s `[cause]:` bracket format, PR #41002). The Java-style
  assumption was wrong; corrected in `internal/contract/types.go` and
  `memory/known-gaps.md` before this plan was written.
- **Treating bare `.stack`-string logging as out of scope** -- rejected;
  it's a common real-world logging pattern and trivially parseable as a
  subset of the same grammar (FR2c), excluding it would buy no real
  simplicity.

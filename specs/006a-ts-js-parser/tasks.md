# Tasks: TypeScript/JS parser

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.

- [ ] **T001** -- Before writing any parsing code: capture more real
  Node.js error dumps and save them to `specs/006a-ts-js-parser/testdata/`
  (documented in that folder's `README.md`, same convention as the
  existing four). At minimum, a 3+-level-nested `.cause` chain -- the
  existing captures only go one level deep, which isn't enough to confirm
  whether indentation keeps growing by 2 spaces per nesting level or the
  trailing-`{`-before-`[cause]:` pattern holds at deeper nesting; don't
  assume it generalizes, capture it and check. Also capture any other
  edge cases that come up while doing this (e.g. multiple elision lines
  in one trace). This ordering matters: the frame-line regex in T002 is
  built from what these captures actually show, not from FR4's prose
  description alone -- see spec.md FR4 and plan.md's pipeline step 2 for
  what was already corrected once during technical design review from
  under-specified real behavior.

  Then scaffold `internal/parser/typescript/` package + copy all real
  fixtures (original four plus any newly captured ones) from
  `specs/006a-ts-js-parser/testdata/` into the package's own `testdata/`;
  add synthetic fixtures listed in `plan.md`'s Testing strategy
  (scoped/nested `node_modules`, bundled path, browser false-positive,
  truncated input, cut-off cause, zero-frame error, `file://` ESM path,
  genuinely-garbled/unparseable input -- this last one is required by
  T009's `ErrUnparseable` acceptance criterion, don't skip it).
  - Depends on: none
  - Acceptance: at least one new real 3+-level-nested-cause fixture
    exists in `specs/006a-ts-js-parser/testdata/`, capture method
    documented in that folder's `README.md`; package compiles (empty
    stubs); `go test ./...` runs (no tests yet, but fixtures load without
    error via a placeholder fixture-loading test).

- [ ] **T002** -- Implement the V8 frame-line regex and single-frame-line
  parser (one line -> `contract.Frame`, `Bucket`/`PackageName` not yet
  set). Per spec.md FR4/plan.md's pipeline step 2: the regex strips
  leading whitespace rather than anchoring to a fixed column count
  (indentation grows with `[cause]` nesting depth -- confirm the actual
  pattern against T001's newly captured 3+-level fixture, don't assume
  it's a flat +2/level without checking), and tolerates an optional
  trailing `{` on a frame line that's the last one before a `[cause]:`
  block.
  - Depends on: T001
  - Acceptance: table test covering a plain frame
    (`at Foo (bar.js:1:2)`), a bare frame (no function name), a
    non-matching line (returns "not a frame line", not an error), a
    frame line indented 6+ spaces (nested-cause case, from the real
    fixtures), and a frame line with a trailing `{` (last-frame-before-
    `[cause]:` case, from the real fixtures).

- [ ] **T003** -- Implement `Detect()`'s shared heuristic (FR4): V8
  frame-line pattern present AND at least one of
  node:/node_modules/version-line/preamble signals present.
  - Depends on: T002
  - Acceptance: true on all real captured fixtures; false on the
    browser-trace synthetic fixture; false on plain non-trace text.

- [ ] **T004** -- Implement source-line+caret preamble detection and
  trailing `Node.js vX.Y.Z` line extraction (distinguishes shape (a)
  crash from shape (b) logged-object, feeds FR14/`VersionSourceTrace`).
  - Depends on: T002
  - Acceptance: correctly extracts version + sets `VersionSourceTrace`
    on the crash fixture; correctly finds neither on the logged-object
    fixture.

- [ ] **T005** -- Implement local-environment shell-out fallback
  (`node --version`, ~2s timeout via `context.WithTimeout` +
  `exec.CommandContext`) and `VersionSourceUnknown` fallback, with a
  fake-exec test seam mirroring
  `internal/codecontext/runner_fake_test.go`.
  - Depends on: T004
  - Acceptance: fake-exec success sets `VersionSourceLocalEnvironment` +
    `Note`; fake-exec failure/timeout sets `VersionSourceUnknown`, no
    version.

- [ ] **T006** -- Implement `[cause]:` bracket-chain parsing into
  successive `ExceptionNode` entries, plus the
  `"... N lines matching cause stack trace ..."` elision-line detection
  feeding `ElidedFrameCount`.
  - Depends on: T002
  - Acceptance: real crash + logged-object fixtures both parse into the
    same 2-node chain with correct `ElidedFrameCount` on the outer node;
    a synthetic 3+-level nested cause fixture also parses correctly;
    `Frames[0]` of every `ExceptionNode` in every real fixture is the
    frame nearest that node's own error origin (spec.md FR21 -- flagged
    for reverification during technical review, confirm this assertion
    is actually added here and isn't just implied by the other checks).

- [ ] **T007** -- Implement bare `.stack`-string parsing (shape (c)):
  detect absence of `[cause]:`/brace structure, produce a single-node
  chain.
  - Depends on: T002
  - Acceptance: real bare-stack fixture parses to exactly one
    `ExceptionNode`, no cause ever detected, not treated as degraded.

- [ ] **T008** -- Implement frame bucketing (own/dependency/runtime),
  `PackageName` extraction (incl. scoped packages, nested
  `node_modules`), anonymous/native -> `BucketRuntime`, bundled-path ->
  `BucketOwn` default, and `file://` URI normalization.
  - Depends on: T002
  - Acceptance: covers every bucketing acceptance criterion in
    `spec.md` individually (own, dependency+plain package, dependency+
    scoped package, nested node_modules, anonymous, node: internal,
    bundled-no-sourcemap default, file:// normalization).

- [ ] **T009** -- Implement malformed/partial-trace tolerance (FR17-20):
  trailing incomplete line dropped + `slog.Warn`, cut-off cause dropped +
  `slog.Warn`, zero-frame error accepted, true unparseable ->
  `parser.ErrUnparseable`.
  - Depends on: T006, T007
  - Acceptance: covers the truncated-input, cut-off-cause, and
    zero-frame synthetic fixtures as successful (degraded) parses;
    covers a genuinely-garbled synthetic fixture as an
    `ErrUnparseable`-wrapped error.

- [ ] **T010** -- Implement `Language()` split
  (`hasTSExtensionFrame`) and wire `javascriptParser`/`typescriptParser`
  as the two exported `LanguageParser` values, composing T002-T009 into
  the full `Parse()`/`Detect()`/`Language()` per value.
  - Depends on: T003, T005, T006, T007, T008, T009
  - Acceptance: `javascriptParser`/`typescriptParser` both satisfy
    `parser.LanguageParser` (compile-time interface assertion); a
    `.ts`-path fixture and an all-`.js`-path fixture each `Detect()`
    true on exactly one of the two values, never both, never neither.

- [ ] **T011** -- Full integration pass: every `spec.md` acceptance
  criterion has a corresponding passing test; run the full gate
  (`gofumpt -l`, `golangci-lint run`, `go build ./...`, `go test ./...`).
  - Depends on: T010
  - Acceptance: all gate commands pass clean; every `spec.md` acceptance
    criterion checkbox can be marked `[x]`.

- [ ] **T012** -- Add the source-map/bundled-code known-limitation entry
  to `memory/known-gaps.md` (referenced but not yet written in `plan.md`
  Risks), and close out this feature's dependency-owed row(s) if any
  remain in `memory/known-gaps.md`'s tables.
  - Depends on: T011
  - Acceptance: `memory/known-gaps.md` accurately reflects this
    feature's actual shipped scope, no stale/pending rows left
    referencing 006a.

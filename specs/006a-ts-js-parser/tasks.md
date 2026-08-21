# Tasks: TypeScript/JS parser

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.

- [x] **T001** -- Capture round complete (see
  `specs/006a-ts-js-parser/testdata/CAPTURE-FINDINGS.md` for the full
  writeup): 3+-level nested `.cause` chain, extra brace-body properties
  (both cause-nested and top-level), the async/unhandled-rejection crash
  preamble, native TypeScript execution (no transformer frame), the
  `@swc/core` message-only false `"Caused by:"` case, `AggregateError`'s
  `[errors]:` shape, and `assert`'s multi-line diff message are all
  real-captured. (Multiple elision lines within a single block was
  investigated and not observed as a distinct real case -- elision
  occurs once per cause-boundary transition, confirmed via the 3-level
  chain; not pursued further.)

  Remaining work for this task: scaffold `internal/parser/typescript/`
  package + copy ALL real fixtures -- the original four, plus every
  fixture captured across `fetch-refused/`, `round2-altprint-tsxpath/`,
  and `full-machine-reverify/` that's actually relevant to parsing
  (skip the confirmed-dead alternate-print-method captures --
  `console.trace`, `JSON.stringify`, `console.dir` -- per spec.md's
  scope, those aren't part of shapes (a)/(b)/(c)) -- from
  `specs/006a-ts-js-parser/testdata/` into the package's own
  `testdata/`. Add synthetic fixtures listed in `plan.md`'s Testing
  strategy (scoped/nested `node_modules`, bundled path, browser
  false-positive, truncated input, cut-off cause, zero-frame error,
  `file://` ESM path, genuinely-garbled/unparseable input -- this last
  one is required by T009's `ErrUnparseable` acceptance criterion,
  don't skip it).
  - Depends on: none
  - Acceptance: package compiles (empty stubs); `go test ./...` runs (no
    tests yet, but every fixture -- real and synthetic -- loads without
    error via a placeholder fixture-loading test); `testdata/README.md`
    (or equivalent index) lists every fixture and which spec.md
    FR/acceptance criterion it exists to exercise.

- [x] **T002** -- Implement the V8 frame-line regex and single-frame-line
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
    `key: value,` property line (e.g. `errno: -111,` from a real
    system-error brace body -- must also return "not a frame line", not
    an error, per spec.md FR7), a frame line indented 6+ spaces
    (nested-cause case, from the real fixtures), and a frame line with a
    trailing `{` (last-frame-before-`[cause]:` case, from the real
    fixtures).

- [x] **T003** -- Implement `Detect()`'s shared heuristic (FR4): V8
  frame-line pattern present (OR the crash preamble present, OR a
  trailing version line present -- relaxed during T003 to admit a real
  zero-frame fixture that still carries both of those signals) AND at
  least one of node:/node_modules/version-line/preamble signals present.
  - Depends on: T002
  - Acceptance: true on all real captured fixtures EXCEPT
    `bare-stack-fetch-cause.txt` (zero frames AND none of the four
    Node-specific signals -- genuinely undetectable, confirmed with
    Vedant during T003, falls through to 003b's "no parser matched"
    path by design; see spec.md FR4 and `memory/known-gaps.md`); false
    on the browser-trace synthetic fixture; false on plain non-trace
    text.

- [x] **T004** -- Implement source-line+caret preamble detection --
  covering BOTH variants per spec.md FR2 (synchronous throw
  `/path/to/file.js:N\n    <source line>\n    ^` and unhandled-promise-
  rejection `node:internal/process/promises:N\n    triggerUncaughtException(err, true /* fromPromise */);\n    ^`) -- and
  trailing `Node.js vX.Y.Z` line extraction (distinguishes shape (a)
  crash from shape (b) logged-object, feeds FR14/`VersionSourceTrace`).
  - Depends on: T002
  - Acceptance: correctly extracts version + sets `VersionSourceTrace`
    on both the sync-throw crash fixture AND the async-rejection crash
    fixture; correctly finds neither on the logged-object fixture.

- [x] **T005** -- Implement local-environment shell-out fallback
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
    is actually added here and isn't just implied by the other checks);
    a fixture with extra `key: value,` properties in a brace body (both
    cause-nested and top-level, e.g. `errno`/`code`/`syscall` and
    `code: 'GenericFailure'`) parses with those properties dropped and a
    `slog.Warn` logged, not mistaken for a frame or a new block; the
    `@swc/core` fixture (plain-text `"Caused by:"` inside `.message`,
    no real `.cause`) parses as a single-node chain -- the
    `"Caused by:"` substring is never treated as a chain boundary; the
    `AggregateError` fixture (`[errors]:` instead of `[cause]:`) parses
    as a single terminal `ExceptionNode` with the array dropped and a
    `slog.Warn` logged, not attempted as a chain; the `assert`
    multi-line-diff fixture's indented diff lines are correctly excluded
    from frame detection (no `at ` prefix) and don't corrupt the
    message/frame boundary.

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
    `ErrUnparseable`-wrapped error. (Note: non-`Error` thrown values --
    `throw "string"`, `throw {...}` -- produce zero `at ...` frame
    lines, so `Detect()` returns `false` for them per FR4 and they never
    reach `Parse()` at all; this is a 003b "no parser matched" case, not
    something this task's `ErrUnparseable` path needs to cover -- see
    `memory/known-gaps.md`.)

- [ ] **T010** -- Implement `Language()` split
  (`hasTSExtensionFrame`) and wire `javascriptParser`/`typescriptParser`
  as the two exported `LanguageParser` values, composing T002-T009 into
  the full `Parse()`/`Detect()`/`Language()` per value.
  - Depends on: T003, T005, T006, T007, T008, T009
  - Acceptance: `javascriptParser`/`typescriptParser` both satisfy
    `parser.LanguageParser` (compile-time interface assertion); a
    `.ts`-path fixture (both the `tsx`-wrapped case WITH a transformer
    frame, and the native-execution case with NO transformer frame --
    see spec.md's native-TypeScript-execution acceptance criterion) and
    an all-`.js`-path fixture each `Detect()` true on exactly one of the
    two values, never both, never neither.

- [ ] **T011** -- Full integration pass: every `spec.md` acceptance
  criterion has a corresponding passing test; run the full gate
  (`gofumpt -l`, `golangci-lint run`, `go build ./...`, `go test ./...`).
  - Depends on: T010
  - Acceptance: all gate commands pass clean; every `spec.md` acceptance
    criterion checkbox can be marked `[x]`.

- [ ] **T012** -- Add the source-map/bundled-code known-limitation entry
  to `memory/known-gaps.md` (referenced but not yet written in `plan.md`
  Risks), and close out this feature's OWNED deferred-criterion row(s)
  (006a listed as Owner -- e.g. the 003a `Language()` static-return
  question) in `known-gaps.md`'s Deferred acceptance criteria table by
  removing that row (003a is `done`/merged -- its `spec.md` has no
  checkbox tracking this specific question, only FR11's doc-comment note
  that a static return is "provisional"; `known-gaps.md`'s row removal
  IS the closing mechanism here, nothing in `003a/spec.md` gets edited).
  Do NOT touch the row where 006a is listed as Source and 003b as Owner
  (the non-`Error`-thrown gap, added during this feature's amendment
  pass) -- that row is correctly still open, owned by a different
  not-yet-built feature, and stays until 003b closes it.
  - Depends on: T011
  - Acceptance: `memory/known-gaps.md` accurately reflects this
    feature's actual shipped scope: the 003a-owned-by-006a row is
    removed, the source-map/bundled-code limitation is added to Accepted
    v1 limitations, and the 006a-source/003b-owner non-`Error`-thrown
    row remains open and unchanged.

# Spec: TypeScript/JS parser

**Status:** Spec'd (interrogation complete, all [NEEDS CLARIFICATION] resolved)
**Folder:** specs/006a-ts-js-parser
**Depends on:** 001-data-contract (done), 003a-language-parser-interface (done), 004-own-code-context-extraction (done)

## Overview

Parses a raw Node.js JavaScript/TypeScript stack trace -- captured from a
CLI/server crash or logged error, on the same machine running
`stack-trace-bundler` -- into the `[]contract.ExceptionNode` cause chain
and `contract.Runtime` that `internal/parser.LanguageParser` (003a)
requires, so 007/008 can render a bundle from it same as they will for
Java (005a). This feature owns detection, cause-chain parsing (including
Node's frame-elision), frame bucketing, and runtime/version detection for
Node.js only -- it does not do dependency resolution (006b), pipeline
wiring (002b), or auto-detection ambiguity handling (003b).

## User stories

- As a developer who pastes a Node.js crash dump (uncaught exception,
  full terminal output) into `stack-trace-bundler`, I want the tool to
  correctly parse it -- including any `Error.cause` chain -- so I get a
  complete, structured bundle instead of a rejected/garbled result.
- As a developer whose code catches and logs an error via
  `console.error(err)` (not necessarily an uncaught crash), I want that
  logged-object dump parsed the same way, so I'm not forced to reproduce
  an uncaught crash just to get a usable bundle.
- As a developer whose logging only prints `err.stack` (a common pattern,
  distinct from logging the Error object itself), I want that flat-string
  form parsed too, even though it never carries cause-chain information --
  because a lot of real-world code logs this way, not always wrapped in a
  try/catch that happens to log the full object.
- As a developer running TypeScript directly (`ts-node`/`tsx`) as well as
  compiled-then-run TypeScript, I want the bundle to correctly say
  "typescript" or "javascript" based on what the trace actually shows,
  not a single hardcoded answer.

## Functional requirements

### Scope / input shapes

1. Recognizes Node.js-produced traces only for v1. Bun and Deno are
   explicitly out of scope (see Out of scope, and
   `memory/known-gaps.md`).
2. Recognizes three distinct real-world input shapes, all Node.js:
   - **(a) True uncaught crash** -- has two preamble variants: a
     **synchronous throw** (`/path/to/file.js:N\n    <source line>\n    ^\n\n`)
     or an **unhandled promise rejection**
     (`node:internal/process/promises:N\n    triggerUncaughtException(err, true /* fromPromise */);\n    ^\n\n`)
     -- either followed by `util.inspect`-formatted error output, followed
     by a trailing `Node.js vX.Y.Z` line. `Detect()`/`Parse()` accept
     either variant (confirmed via real capture, see
     `testdata/CAPTURE-FINDINGS.md`).
   - **(b) Logged object** -- `console.error(err)`/`console.log(err)`:
     the same `util.inspect`-formatted error output as (a), but with
     NO source-line-and-caret preamble and NO trailing `Node.js vX.Y.Z`
     line.
   - **(c) Bare `.stack` string** -- `console.error(err.stack)`: flat
     `Error: <message>\n    at ...` lines only. Never carries cause-chain
     information (`.stack` is computed independently of `util.inspect`).
     A valid, distinct grammar, not a degraded case of (a)/(b).
3. Assumes the trace was generated on the same machine currently running
   `stack-trace-bundler`. A trace pasted from a different machine, a CI
   log, or a container is out of scope for v1 (see
   `memory/known-gaps.md`) -- this affects runtime-version-detection
   accuracy specifically (FR14-16), not detection/parsing correctness.
4. `Detect(rawTrace string) bool` (003a's interface) returns true only if
   BOTH: (i) at least one line matches the V8 frame-line pattern (some
   amount of leading whitespace, then `at <description> (<file>:<line>:<col>)`
   or `at <file>:<line>:<col>`, optionally followed by a trailing `{` when
   it's the last frame line before a `[cause]:` block), **OR** the
   source-line-and-caret crash preamble is present, **OR** a trailing
   `Node.js vX.Y.Z` line is present -- this relaxation (added during T003)
   covers a genuine real fixture with zero frame lines
   (`Error.stackTraceLimit = 0`, see FR19 and `testdata/zero-stack-trace-limit.txt`)
   that still carries the crash preamble and version line -- that
   combination cannot plausibly appear outside a real Node crash dump, so
   requiring a frame-line match in addition to it would reject a genuine
   Node trace for no real benefit; AND (ii) at least one Node-specific positive signal is present: a `node:` internal frame, a
   `node_modules` path segment, a trailing `Node.js vX.Y.Z` line, or the
   source-line-and-caret crash preamble. Requiring (ii) prevents
   false-positive matches on browser (Chrome/V8-based) stack traces,
   which share the same frame-line grammar but lack all four Node-specific
   signals.

   **Known residual gap, not solved by the above**: a bare `.stack`-only
   log line with zero frames AND none of the four Node-specific signals
   (e.g. `console.error(err.stack)` on an error whose own stack was
   itself already empty -- confirmed real, see
   `testdata/bare-stack-fetch-cause.txt`, literally just
   `"TypeError: fetch failed"` with no file, no frame, no signal of any
   kind) is genuinely undetectable -- no heuristic can distinguish it
   from an arbitrary one-line string in any language, and loosening
   `Detect()` further to catch it would reopen the exact false-positive
   risk condition (ii) exists to prevent. `Detect()` correctly returns
   `false` for this fixture; it falls through to 003b's "no parser
   matched" path by design, not by omission (see
   `memory/known-gaps.md`). 

   **Leading whitespace is variable, not fixed at 4 spaces** --
   confirmed against real captures in `testdata/`: a `[cause]:`-nested
   block's frames are indented 6 spaces, two more than the enclosing
   block's 4. T001 captures a 3+-level-nested real fixture to confirm
   whether this +2-per-level pattern holds deeper, rather than assuming
   it from the 2-level case alone. The frame-line matcher strips/ignores
   leading whitespace entirely rather than anchoring to a fixed column
   count, and does not require end-of-line immediately after the column
   number, so the trailing-`{` case doesn't disqualify an otherwise-valid
   frame line. Flagged during technical design review -- the original FR4
   wording showed a literal 4-space-anchored pattern that would have
   failed to match either real multi-node fixture's `[cause]`-nested
   frames.

### Language determination (`javascript` vs `typescript`)

5. Implemented as two separate `LanguageParser` values --
   `javascriptParser` and `typescriptParser` -- both in
   `internal/parser/typescript`, sharing one unexported internal parse
   engine. `Language()` is static per value (per 003a's interface
   shape); the two values differ only in `Detect()`/`Language()`.
6. A trace is classified TypeScript if ANY frame's `FilePath` ends in
   `.ts` or `.tsx` (a real, achievable case: `ts-node`/`tsx` execute
   TypeScript source directly, preserving the extension in V8 frames).
   Otherwise JavaScript. Compiled-then-`node`-run TypeScript (all `.js`
   paths post-compilation) is indistinguishable from hand-written JS and
   is correctly classified `javascript` -- not a bug, a genuine limit of
   what the trace can reveal.

### Cause-chain parsing

7. For shapes (a) and (b): parses the `[cause]: <ClassName>: <message>`
   bracket-nested structure (confirmed against real captured Node
   24.18.0 output, see `specs/006a-ts-js-parser/testdata/`) into
   successive `contract.ExceptionNode` entries, outermost first.

   A block's brace body (top-level exception or `[cause]`-nested) may
   contain additional `key: value,` property lines after the frame list
   and before the closing `}` (e.g. `errno`, `code`, `syscall` on system
   errors, or `code: 'GenericFailure'` on the top-level exception itself
   -- this is not cause-specific). These are recognized as block content,
   not mistaken for a new frame or a malformed line, and dropped -- not
   surfaced on `contract.ExceptionNode`, which has no field for them --
   logging via `slog.Warn` when present.

   The `[cause]:` bracket marker is the *only* valid signal for a JS/TS
   cause-chain boundary. A literal `"Caused by:"` substring appearing in
   an error's own `.message` (confirmed possible via native-binding
   errors, e.g. `@swc/core` -- see `testdata/CAPTURE-FINDINGS.md`) MUST
   NOT be interpreted as a chain boundary -- that convention belongs to
   Java, not JS/TS.
8. Recognizes Node's frame-elision line
   (`... N lines matching cause stack trace ...`) and sets
   `ExceptionNode.ElidedFrameCount` to the parsed `N`. This is a real,
   meaningful, non-zero value in the common case for JS/TS -- unlike the
   original (incorrect) assumption recorded before this feature's
   interrogation, corrected in `internal/contract/types.go` and
   `memory/known-gaps.md`.
9. For shape (c) (bare `.stack`): produces a single-node chain (no
   cause ever detected) -- this is the correct, expected outcome for
   this shape, not a parse failure.

### Frame bucketing

10. `Bucket` is always fully assigned for every frame (no partial/unknown
    state, per 003a's interface constraint):
    - `BucketOwn`: file path is inside the project tree, not under
      `node_modules`, not a `node:` internal.
    - `BucketDependency`: file path contains a `node_modules` segment.
      `PackageName` is the path segment immediately after the LAST
      `node_modules/` occurrence (handles nested dependency trees); if
      that segment starts with `@`, the next segment is included too
      (scoped packages, e.g. `@babel/core`).
    - `BucketRuntime`: file path starts with `node:`, OR the frame has
      no file info at all (`<anonymous>`, native binding).
11. Bundled/minified single-file output (e.g. webpack/esbuild `dist/
    bundle.js` combining own and vendored code with no `node_modules`
    segment) defaults to `BucketOwn` when no `node_modules` segment is
    present. Source-map-aware bucketing is out of scope for v1 (see
    Out of scope) -- this is a known, accepted misclassification risk,
    not a silent gap.
12. `FilePath` values that arrive as `file://` URIs (Node ESM mode can
    emit these, not just Deno) are normalized to real filesystem paths
    before any other processing, per `internal/contract/types.go`'s
    `Frame.FilePath` doc comment ("never a URI").

### Runtime / version detection

13. `Runtime.Name` is always `"node"` for v1 (Bun/Deno deferred).
14. If a trailing `Node.js vX.Y.Z` line is present (shape (a) only, per
    FR2), sets `Runtime.Version` from it and
    `Runtime.VersionSource = VersionSourceTrace`.
15. Otherwise, shells out to the local environment: runs
    `node --version`, bounded by a short timeout (target: 2s -- `node
    --version` is near-instant; a longer timeout only delays failure on
    a broken/hung `node`, unlike git operations which can legitimately be
    slow). On success, sets `Runtime.VersionSource =
    VersionSourceLocalEnvironment` and a `Runtime.Note` explaining the
    version was inferred locally and may not reflect drift (e.g. an
    active `nvm`/`fnm` version, or a container) even on the same
    physical machine.
16. If the shell-out fails (`node` not on `PATH`, times out, or any
    other error), sets `Runtime.VersionSource = VersionSourceUnknown`
    with no `Runtime.Version`.

### Malformed / partial trace handling

17. `Parse()` tolerates a trailing incomplete line (doesn't match a
    complete frame pattern) by dropping it and logging via `slog.Warn` --
    still a successful parse. Applies regardless of cause (input
    truncation at `002a`'s 512KB cap, or a genuinely partial paste --
    `Parse()` has no way to distinguish these via its `(ctx,
    rawTrace string)` signature, and doesn't need to).
18. `Parse()` tolerates a cause chain that opens (`[cause]:` appears) but
    never resolves (cut off mid-cause): keeps the outer exception node,
    drops the incomplete cause node, logs via `slog.Warn`. Still a
    successful parse.
19. A message-only error with zero frames (e.g. `Error.stackTraceLimit =
    0`) is a valid degraded result -- `ExceptionNode.Frames` as an empty
    slice, `slog.Warn` logged -- not a failure.
20. `Parse()` returns an error wrapping `parser.ErrUnparseable` (003a)
    only when the outermost exception's own header and at least one
    usable frame cannot be extracted at all -- i.e. `Detect()` matched
    loosely but nothing coherent can actually be built.

### Frame ordering

21. Frames within each returned `ExceptionNode` are ordered exactly as
    they appear in the raw trace, matching V8's own convention of listing
    the frame closest to that exception's origin first: `Frames[0]` is
    that frame. `003a`'s `LanguageParser.Parse()` contract requires this,
    and `contract.ComputeFingerprint` depends on it for node identity.
    **Flagged for reverification during technical review**: this is
    satisfied by construction (`engine.go`'s frame-collection order is
    the raw trace's own line order) and was checked by inspection against
    the real fixtures in `testdata/` during interrogation, but is not yet
    backed by an explicit test assertion -- confirm during the technical
    design review pass that this holds for every real fixture, including
    the elided-frames case, and add an explicit test if one doesn't
    already exist by then.

## Non-functional requirements

- `Detect()` is pure text-pattern matching -- no I/O, no subprocess calls
  (003a's interface constraint). Only `Parse()` may shell out (FR15/16),
  and only bounded by a short timeout.
- No new third-party dependencies. Frame-line/cause/elision parsing uses
  the standard library (`regexp`, `strings`) only, per `AGENTS.md`'s
  stdlib-first stance.
- Following the offline-first pattern already established
  (`memory/decisions/0001-offline-first-time-boxed-dependency-version-resolution.md`,
  `internal/codecontext/runner.go`'s `gitTimeout`), the local-environment
  shell-out (FR15) is bounded via `context.WithTimeout` +
  `exec.CommandContext`, never blocking indefinitely.

## Out of scope

- Bun and Deno runtime detection/parsing (`memory/known-gaps.md`) --
  Bun uses JavaScriptCore, not V8; its format is not confirmed to match
  Node's, and Deno, while V8-based like Node, hasn't been verified
  either. Both need their own real-captured fixtures before being added.
- Browser-originated traces (Chrome/Firefox/Safari DevTools) -- see
  `specs/INDEX.md` #012 (future, separate feature) for a possible
  live-capture alternative.
- Cross-machine traces (pasted from a different machine, CI run, or
  container than the one running this binary) -- affects
  runtime-version-detection accuracy only, not detection/parsing
  correctness (`memory/known-gaps.md`).
- Source-map-aware frame bucketing for bundled/minified output
  (`memory/known-gaps.md` -- to be added alongside this spec's other
  entries once this feature starts implementation).
- Dependency resolution / manifest parsing (006b).
- Pipeline wiring, real stdout bundle output (002b).
- Auto-detection ambiguity/no-match handling across multiple registered
  parsers (003b) -- this feature only implements `Detect()`/`Parse()`
  for itself.
- `AggregateError` / `Suppressed`-style branching chains -- `Bundle.Chain`
  is strictly linear per `001`'s contract; out of scope for any parser.
  Concretely: when a block's brace body contains `[errors]:` instead of
  `[cause]:`, `Parse()` does not attempt to interpret it as a chain. It
  parses that block as a single terminal `ExceptionNode` (header + own
  frames, if any) and drops the `[errors]` array's nested errors entirely
  (not surfaced, not partially parsed), logging via `slog.Warn`. This is
  a successful (degraded) parse, not `ErrUnparseable` (confirmed real
  shape, see `testdata/CAPTURE-FINDINGS.md`).

## Acceptance criteria

- [ ] Given a true uncaught-crash trace with a `.cause` chain (shape a),
      when parsed, then the result is a multi-node chain with correct
      `ElidedFrameCount` per node, `Runtime.Version` set from the
      trailing `Node.js vX.Y.Z` line with `VersionSourceTrace`.
- [ ] Given the same error logged via `console.error(err)` (shape b, no
      preamble, no trailing version line), when parsed, then the result
      is the same chain structure as the crash-dump case, but
      `Runtime.VersionSource` is `VersionSourceLocalEnvironment` (or
      `VersionSourceUnknown` if the local shell-out fails).
- [ ] Given the same error logged via `console.error(err.stack)` (shape
      c), when parsed, then the result is a single-node chain with no
      cause ever detected -- not treated as a parse failure.
- [ ] Given a trace where every frame's `FilePath` ends in `.js`
      (compiled TypeScript or hand-written JS), when `Detect()`/
      `Language()` run, then the result is `LanguageJavaScript`.
- [ ] Given a trace where at least one frame's `FilePath` ends in `.ts`
      or `.tsx` (`ts-node`/`tsx` execution), when `Detect()`/`Language()`
      run, then the result is `LanguageTypeScript`.
- [ ] Given a trace with frames under a `node_modules/lodash/` path and
      frames under a `node_modules/@babel/core/` path, when bucketed,
      then both are `BucketDependency` with `PackageName` `"lodash"`
      and `"@babel/core"` respectively.
- [ ] Given a trace with an `<anonymous>` frame and a `node:internal/...`
      frame, when bucketed, then both are `BucketRuntime`.
- [ ] Given a Chrome DevTools-style browser trace (V8 frame grammar, no
      `node:`/`node_modules`/version-line/preamble signal), when
      `Detect()` runs, then it returns `false`.
- [ ] Given a trace truncated exactly at `002a`'s 512KB cap, cutting the
      last frame line mid-way, when parsed, then the result is a
      successful parse with the incomplete trailing line dropped and a
      `slog.Warn` logged.
- [ ] Given a trace whose `[cause]:` chain is cut off mid-cause, when
      parsed, then the outer exception node is kept, the incomplete
      cause node is dropped, and a `slog.Warn` is logged -- not a parse
      failure.
- [ ] Given input that superficially matches the V8 frame-line pattern
      but has no valid Error header or usable frame at all, when parsed,
      then `Parse()` returns an error wrapping `parser.ErrUnparseable`.
- [ ] Given a `file://`-prefixed frame path (Node ESM), when parsed,
      then `Frame.FilePath` is normalized to a real filesystem path.
- [ ] Given each real captured fixture in `testdata/` (the original
      four plus any added by T001, including the 3+-level-nested-cause
      capture), when parsed, then `Frames[0]` of every `ExceptionNode` is
      the frame nearest that node's own error origin, matching the raw
      trace's own line order (FR21). **Flagged for reverification during
      technical review** -- not yet exercised by an explicit test, only
      checked by inspection during interrogation.
- [ ] Given a trace produced by Node's native TypeScript execution
      (type-stripping, no `tsc`/`tsx` involved -- `.ts` frame paths via
      the ordinary CJS loader, no transformer frame), when
      `Detect()`/`Language()` run, then the result is
      `LanguageTypeScript`, identical to the `tsx`-wrapped case.
- [ ] Given a trace whose top-level exception's brace body contains
      `[errors]:` (an `AggregateError`) instead of `[cause]:`, when
      parsed, then the result is a single terminal `ExceptionNode` with
      the `[errors]` array dropped and a `slog.Warn` logged -- not
      `ErrUnparseable`.
- [ ] Given a trace whose message is a multi-line diff (e.g. Node's
      built-in `assert.strictEqual` output, containing indented
      brace-like lines that are not real frames), when parsed, then
      frame detection correctly ignores the diff lines (no `at ` prefix)
      and only real frame lines are extracted.

## Open questions

None remaining -- all resolved during interrogation. See
`specs/006a-ts-js-parser/progress.md` for the session log.

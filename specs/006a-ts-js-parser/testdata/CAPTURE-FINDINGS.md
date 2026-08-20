# 006a Capture Round — Consolidated Findings

Staging document. Nothing here has been applied to `spec.md`, `plan.md`,
or `tasks.md` yet — this is the "capture everything, then decide" record
per the agreed process. Each item below is either (a) a confirmation that
closes a previously-open question, or (b) a real finding that needs a
spec/plan decision.

Sources: `errors.md` (sandbox, retried), `fetch-refused/`,
`round2-altprint-tsxpath/`, `full-machine-reverify/` — all real captures,
re-verified on Vedant's own machine (Node v24.18.0) except where noted.

---

## A. Confirmations (close previously-open questions, no spec change needed)

1. **3+-level `.cause` chain indentation grows +2 spaces per level.**
   Confirmed via the level1/level2/level3 synthetic script. Also confirmed:
   closing a deep chain with no trailing `Node.js vX.Y.Z` footer requires
   multiple stacked `}` lines at decreasing indentation, not just one.

2. **`tsx`'s transformer frame path DOES contain a `node_modules` segment**
   (`/root/.npm/_npx/<hash>/node_modules/tsx/dist/register-*.cjs`).
   FR10's dependency-bucketing heuristic correctly buckets it as
   `BucketDependency`. Previously-open question, now closed.

3. **`@swc/core`'s "Caused by:" is confirmed NOT a real `.cause` property.**
   `Object.keys(err) = ['code']`, `err.cause = undefined`,
   `'cause' in err = false`. It's plain text inside `.message`. This is
   empirically proven now, not theorized.

4. **Scoped package frame paths correctly resolve** (`node_modules/@swc/core/index.js`)
   — `PackageName` extraction for scoped packages is provable from real data.

---

## B. New findings requiring a spec/plan decision

### B1. Native TypeScript execution (Node's built-in type-stripping) — HIGH PRIORITY
Node 23.6+/24 (default-on) can run `.ts` files directly with no tool at
all: `node app.ts` works and produces a clean stack via the normal CJS
loader, `.ts` frame paths, NO transformer frame, NO compiled `.js`:
```
at level3 (/home/.../app.ts:2:9)
at Module._compile (node:internal/modules/cjs/loader:1871:14)
```
This is a **third, previously-untracked TS execution path**, distinct from:
- `tsc`-compiled → run as `.js` (frame paths end in `.js`)
- `tsx`-wrapped (frame paths end in `.ts`, but WITH a transformer frame present)
- **native execution (frame paths end in `.ts`, NO transformer frame, plain CJS loader)**

FR6 currently doesn't distinguish this third case. Given this is Node's
default behavior on recent versions, it's likely to be *more* common than
`tsx` in the wild going forward. Needs explicit spec treatment.

### B2. Message/chain-keyword collision risk (Java `"Caused by:"` vs JS message text)
Confirmed by A3 above. A JS error's own `.message` can contain the literal
substring `"Caused by:"` without that indicating any real chain. If any
shared/cross-language detection logic ever pattern-matches on that
substring instead of requiring the `[cause]:` bracket marker specifically,
this fixture (`scoped-package-swc`) is a guaranteed false positive.
Needs an explicit note in spec.md/plan.md: **`[cause]:` bracket is the
only valid JS chain signal — never infer a chain from message text alone.**

### B3. Extra properties can appear on the TOP-LEVEL exception, not just a nested cause
Previously only confirmed nested inside a `[cause]:` block (fetch's
`errno`/`code`/`syscall`/`address`/`port`). The `@swc/core` capture shows
`code: 'GenericFailure'` attached directly to the outer/top-level
exception object, no `[cause]` involved. Broadens FR4/FR7 — the
extra-properties grammar isn't cause-specific.

### B4. `AggregateError` uses `[errors]:` (plural array), not `[cause]:` (singular)
```
[AggregateError: All promises were rejected] {
  [errors]: [
    Error: first failure
        at ...,
    Error: second failure
        at ...
  ]
}
```
Confirms the "linear chains only, branching deferred" decision is
justified — this genuinely doesn't fit the `[cause]:` grammar. But now
that the real shape is confirmed, it needs a **defined failure behavior**
(explicitly detected-and-skipped/unsupported vs. accidentally mis-parsed
as if `errors` were `cause`) rather than an undefined gap.

### B5. Non-`Error` thrown values have no header pattern at all
`throw "string"` → no `ClassName: message` line, just the raw string
printed + a `(Use --trace-uncaught ...)` hint line, no stack.
`throw {code: 1, message: "..."}` → raw `util.inspect` of the object,
still no `Error:`-style header. FR-whatever's message-header parsing
won't match either shape. Needs a documented v1 limitation / explicit
unparseable-input handling, not silent mis-parsing.

### B6. `assert`'s multi-line diff message stress-tests the message/frame boundary
```
AssertionError [ERR_ASSERTION]: Expected "actual" to be reference-equal...
+ actual - expected

  {
+   age: 30,
-   age: 31,
    name: 'Alice'
  }

    at Object.<anonymous> (...)
```
The message itself contains indented, brace-containing lines that
superficially resemble the "extra properties" brace pattern (B3) but
are message content, not object properties, and appear BEFORE the real
frame list starts. Confirms frame-line detection must key strictly on
the literal `at ` prefix — not on indentation or brace shape.

---

## C. Confirmed-useless / confirmed-unsafe alternate print methods (exclude from scope)

- **`console.trace(err)`**: concatenates the passed error's own stack (if
  any) with the trace call-site's own current stack, with NO delimiter
  between them — two unrelated call stacks glued together, structurally
  misleading. Confirms exclusion from FR2 is correct, not just deprioritized.
- **`JSON.stringify(err)`**: always returns `"{}"` — `message`/`stack`/`cause`
  are all non-enumerable by default, with or without a `.cause` chain.
  Zero value, safe to formally exclude.
- **`console.dir(err)` / `console.dir(err, {depth: null})`**: identical
  output to plain `console.error(err)` for every case tested here. No new
  information gained; not worth separate spec treatment.

---

## D. Accepted limitations (downgrade from "open question" to "documented, not pursued")

- **Deep multi-level `node_modules` nesting** (e.g.
  `express/node_modules/finalhandler/node_modules/statuses`): modern npm
  hoisting makes this rare in practice; flat `node_modules/<pkg>` is the
  realistic common case. Not worth engineering a synthetic deep-nest fixture.
- **Multiple elision lines within a single block**: not observed as a
  distinct case in any real capture; elision occurs once per cause-boundary
  transition (confirmed via the 3-level chain), not something to chase further.

---

## Async preamble variant (confirmed, not new this round, logging for completeness)
Sync throw:
```
/path/to/file.js:8
throw outer;
^
```
Async/unhandled-rejection:
```
node:internal/process/promises:394
    triggerUncaughtException(err, true /* fromPromise */);
    ^
```
Two structurally distinct shape-(a) preambles. FR2 currently describes
only the sync variant. Needs explicit treatment as a second variant.

# 006a real-capture fixtures

Captured by Vedant, Node.js v24.18.0, from a script equivalent to:

```js
const inner = new Error("inner failure");
const outer = new Error("outer failure", { cause: inner });
throw outer; // (or: console.error(outer); / console.error(outer.stack);)
```

These are real runtime output, not hand-constructed examples like 003a's
plan.md pseudocode was (explicitly labeled there as "constructed, not from
a real incident"). Use these to validate 006a's spec.md/plan.md against
actual behavior instead of assumption.

- `node-uncaught-crash-with-cause.txt` -- truly uncaught `throw`, no
  try/catch, no console.error call anywhere. Confirms: source-line+caret
  preamble, `util.inspect`-style `[cause]:` bracket chain, Node's real
  frame-elision (`... N lines matching cause stack trace ...`), trailing
  `Node.js vX.Y.Z` line (VersionSourceTrace signal per
  `internal/contract/types.go`).
- `node-console-error-object-with-cause.txt` -- same error, logged via
  `console.error(outer)` (the Error object itself). Same `[cause]`/elision
  body as the crash dump, but NO source-line+caret preamble and NO
  trailing `Node.js vX.Y.Z` line -- this is the distinguishing signal
  between "true crash" and "logged object" within the same format family.
- `node-console-error-bare-stack.txt` -- same error, logged via
  `console.error(outer.stack)` (the bare string property). Flat lines
  only. No `[cause]`, no elision, no braces -- cause chain is NEVER visible
  in this shape, by construction (`.stack` is computed independently of
  `util.inspect`). Confirms this is a genuinely different, simpler
  grammar from the two above, not a variant of it.
- `node-uncaught-crash-typeerror-no-cause.txt` -- bonus/incidental capture:
  a true uncaught crash from a built-in `TypeError` (calling a
  non-existent method), no `.cause` at all. Confirms the source-line+caret
  preamble and trailing `Node.js vX.Y.Z` line are general to ALL uncaught
  crashes, not specific to user-defined `Error` subclasses or to errors
  that happen to have a cause.

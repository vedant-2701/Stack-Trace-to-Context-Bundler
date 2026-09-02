# Round 2 Capture Scripts — Alternate Print Methods + tsx Path Confirmation

Run each script with the exact command shown. Paste raw, unmodified terminal
output into the matching numbered section in ERRORS.md — do not trim, do not
retype, do not summarize. If a line is long, paste it in full (this matters
for #3, where truncation is the thing we're trying to eliminate).

---

## 1. alt-print-with-cause

Tests `console.trace`, `err.toString()`, `JSON.stringify(err)`, and
`console.dir(err)` against an error that HAS a `.cause` chain (built-in
`fetch`, no npm install needed).

**File:** `alt-print-with-cause.mjs`
```js
try {
  await fetch('http://127.0.0.1:59999/api');
} catch (err) {
  console.log("=== console.trace(err) ===");
  console.trace(err);

  console.log("\n=== err.toString() ===");
  console.log(err.toString());

  console.log("\n=== JSON.stringify(err) ===");
  console.log(JSON.stringify(err));

  console.log("\n=== console.dir(err) ===");
  console.dir(err);

  console.log("\n=== console.dir(err, { depth: null }) ===");
  console.dir(err, { depth: null });
}
```

**Command:** `node alt-print-with-cause.mjs`

---

## 2. alt-print-no-cause

Same four methods, against a plain error with no `.cause` (real
`JSON.parse` SyntaxError, no npm install needed) — for comparison against #1.

**File:** `alt-print-no-cause.js`
```js
try {
  JSON.parse("{ invalid_json: ");
} catch (err) {
  console.log("=== console.trace(err) ===");
  console.trace(err);

  console.log("\n=== err.toString() ===");
  console.log(err.toString());

  console.log("\n=== JSON.stringify(err) ===");
  console.log(JSON.stringify(err));

  console.log("\n=== console.dir(err) ===");
  console.dir(err);
}
```

**Command:** `node alt-print-no-cause.js`

---

## 3. tsx-full-transformer-path

Re-run of the earlier `tsx app.ts` capture, specifically to get the FULL,
untruncated path of the `tsx` transformer frame
(`.../tsx/dist/register-*.cjs`) — the earlier capture had `...` in the
middle of that path and we need to confirm whether it contains a
`node_modules` segment (affects FR10 dependency-bucketing).

**File:** `app.ts`
```ts
function level3() {
  throw new Error("TS execution test error");
}
function level2() { level3(); }
function level1() { level2(); }
level1();
```

**Command:** `npx --yes tsx app.ts 2>&1`

Paste every frame line in full — especially the `Object.transformer (...)`
line — with no `...` shortening on your end. If your terminal itself wraps
or truncates long paths, run with output redirected to a file instead:
`npx --yes tsx app.ts > tsx-output.txt 2>&1` and paste the file contents.

---

## 4. scoped-package-cause-body (optional, lower priority)

Only if you have time: re-run the `@swc/core` scoped script but this time
also capture `Object.keys(err)` and `err.cause` explicitly, to confirm
whether swc's "Caused by:" text is really just part of `err.message` (no
real `.cause` property) or whether `.cause` is separately set to something
structured. This resolves the open question from the findings discussion —
whether a Java-style `"Caused by:"` substring inside a JS error's own
message could collide with Java chain-detection logic.

**File:** `scoped-cause-check.js`
```js
const swc = require('@swc/core');
try {
  swc.transformSync('const x = 1;', { jsc: { target: 'invalid-target' } });
} catch (err) {
  console.log("=== Object.keys(err) ===");
  console.log(Object.keys(err));
  console.log("\n=== err.cause ===");
  console.log(err.cause);
  console.log("\n=== 'cause' in err ===");
  console.log('cause' in err);
}
```

**Command:** `node scoped-cause-check.js` (run from the same `/tmp/swctest` install as before)

# Full Machine Re-verify — Scripts

Everything from `errors.md`'s sandbox report, re-run on your real machine to
confirm the sandbox output is trustworthy, plus 3 new structural axes not
captured anywhere yet. Does NOT include `tsx` full-path or `@swc/core`
`.cause`-check — those already live in
`round2-altprint-tsxpath/SCRIPTS.md` (#3 and #4). Does NOT include
`fetch-refused` — that has its own folder already.

Run each script, paste raw output into the matching heading in ERRORS.md.

---

## 1. pg-connection-refused

Needs: `npm install pg` (no real Postgres needed — connection just gets refused)

**File:** `pg-fail.js` (Shape A)
```js
const { Client } = require('pg');
const client = new Client({
  host: '127.0.0.1',
  port: 54321,
  user: 'invalid',
  password: 'bad',
  database: 'test',
  connectionTimeoutMillis: 1000
});

async function run() {
  await client.connect();
}
run();
```
**Command:** `node pg-fail.js`

**File:** `pg-catch.js` (Shape B + C)
```js
const { Client } = require('pg');
const client = new Client({
  host: '127.0.0.1',
  port: 54321,
  user: 'invalid',
  password: 'bad',
  database: 'test',
  connectionTimeoutMillis: 1000
});

async function run() {
  try {
    await client.connect();
  } catch (err) {
    console.log("=== Shape B: console.error(err) ===");
    console.error(err);
    console.log("\n=== Shape C: console.error(err.stack) ===");
    console.error(err.stack);
  }
}
run();
```
**Command:** `node pg-catch.js`

---

## 2. nested-deps-statuses-flat

Needs: `npm install statuses`

**File:** `nested-deps.js`
```js
const statuses = require("statuses");
statuses("999999");
```
**Command:** `node nested-deps.js`

**Optional — genuinely deep nesting:** modern npm hoisting means a real
`express/node_modules/finalhandler/node_modules/statuses` path likely won't
occur naturally. If you want a real (not engineered) deep-nested capture,
skip this — it's not worth manually engineering a fake directory structure
just to prove path-parsing that the flat case already proves. Flag if you
disagree.

---

## 3. json-parse-syntaxerror

No install needed.

**File:** `json-fail.js`
```js
try {
  JSON.parse("{ invalid_json: ");
} catch (err) {
  console.log("Shape B:");
  console.error(err);
  console.log("Shape C:");
  console.error(err.stack);
  console.log("Cause defined?", 'cause' in err);
}
```
**Command:** `node json-fail.js`

---

## 4. zero-stack-trace-limit

No install needed.

**File:** `zero-stack.js` (Shape A)
```js
Error.stackTraceLimit = 0;

function crash() {
  throw new Error("Zero stack frames requested");
}

crash();
```
**Command:** `node zero-stack.js`

**File:** `zero-stack-catch.js` (Shape C)
```js
Error.stackTraceLimit = 0;

function crash() {
  throw new Error("Zero stack frames requested");
}

try {
  crash();
} catch (err) {
  console.log("Shape C:");
  console.error(err.stack);
}
```
**Command:** `node zero-stack-catch.js`

---

## 5. esm-runtime-error

No install needed.

**File:** `esm-app.mjs`
```js
import fs from 'fs';

function main() {
  throw new Error("ESM runtime failure");
}

main();
```
**Command:** `node esm-app.mjs`

---

## 6. import-outside-module

No install needed.

**File:** `bad-esm.js`
```js
import fs from 'fs';
```
**Command:** `node bad-esm.js`

---

## 7. typescript-tsc-compiled

Needs: `npm install --no-save typescript`

**File:** `app.ts`
```ts
function level3() {
  throw new Error("TS execution test error");
}
function level2() { level3(); }
function level1() { level2(); }
level1();
```
**Build command:** `npx --yes -p typescript tsc app.ts`
**Run command:** `node app.js`

---

## 8. esbuild-minified-bundle

Needs: `npx --yes esbuild` (no persistent install)

**File:** `entry.js`
```js
function heavyCalculation() {
  const obj = null;
  return obj.nonExistentMethod();
}
heavyCalculation();
```
**Build command:** `npx --yes esbuild entry.js --bundle --minify --outfile=dist/bundle.js`
**Run command:** `node dist/bundle.js`

---

## 9. scoped-package-swc

Needs: `npm install @swc/core` (in an isolated folder, this can take a while — native binding)

**File:** `scoped.js`
```js
const swc = require('@swc/core');
swc.transformSync('const x = 1;', {
  jsc: { target: 'invalid-target' }
});
```
**Command:** `node scoped.js`

---

## 10. aggregate-error (NEW AXIS)

No install needed. Tests `AggregateError` — the `.errors` array shape,
structurally different from a linear `.cause` chain. Your project's design
defers branching chains for v1, so this capture is about confirming what
the *raw* shape actually looks like, so the parser's failure/skip behavior
on it can be deliberate rather than accidental.

**File:** `aggregate-uncaught.mjs` (Shape A)
```js
await Promise.any([
  Promise.reject(new Error("first failure")),
  Promise.reject(new Error("second failure")),
]);
```
**Command:** `node aggregate-uncaught.mjs`

**File:** `aggregate-catch.mjs` (Shape B + C)
```js
try {
  await Promise.any([
    Promise.reject(new Error("first failure")),
    Promise.reject(new Error("second failure")),
  ]);
} catch (err) {
  console.log("=== Shape B: console.error(err) ===");
  console.error(err);
  console.log("\n=== Shape C: console.error(err.stack) ===");
  console.error(err.stack);
  console.log("\n=== err.errors ===");
  console.error(err.errors);
}
```
**Command:** `node aggregate-catch.mjs`

---

## 11. non-error-thrown (NEW AXIS)

No install needed. Tests what Node prints when a non-`Error` value is
thrown — a real, common bug pattern (`throw "string"`, `throw {code: 1}`).
No stack trace exists in this case at all.

**File:** `throw-string.js`
```js
throw "just a plain string, not an Error object";
```
**Command:** `node throw-string.js`

**File:** `throw-object.js`
```js
throw { code: 1, message: "plain object, not an Error" };
```
**Command:** `node throw-object.js`

---

## 12. assert-multiline-diff (NEW AXIS)

No install needed (built-in `assert` module). Tests multi-line message
parsing on a NORMAL error (no `[cause]`, no third-party formatting quirk) —
comparison point against the `@swc/core` "Caused by:" anomaly.

**File:** `assert-fail.js`
```js
const assert = require('assert');
assert.strictEqual(
  { name: "Alice", age: 30 },
  { name: "Alice", age: 31 }
);
```
**Command:** `node assert-fail.js`

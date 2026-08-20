# fetch-refused Capture Scripts

Real capture, built-in `fetch()` only (no npm install needed). Two scripts:
one for the uncaught crash shape, one for the caught/logged shapes.

---

## 1. fetch-refused-uncaught (Shape A)

**File:** `fetch-refused.mjs`
```js
// Run as ES module
const res = await fetch('http://127.0.0.1:59999/api');
```

**Command:** `node fetch-refused.mjs`

---

## 2. fetch-refused-caught (Shape B + Shape C)

**File:** `fetch-catch.mjs`
```js
try {
  const res = await fetch('http://127.0.0.1:59999/api');
} catch (err) {
  console.log("=== Shape B: console.error(err) ===");
  console.error(err);
  console.log("\n=== Shape C: console.error(err.stack) ===");
  console.error(err.stack);
  console.log("\n=== cause ===");
  console.error(err.cause);
}
```

**Command:** `node fetch-catch.mjs`

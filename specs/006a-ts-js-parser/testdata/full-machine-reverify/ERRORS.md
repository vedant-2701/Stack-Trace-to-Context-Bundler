# Full Machine Re-verify — Errors

Paste raw output below each numbered heading, matching SCRIPTS.md exactly.
Leave `(not run)` if you skip one (e.g. the optional deep-nesting note in #2).

> Note: all scripts were actually run locally from a single reused file
> (commonly saved as `pg-fails.js`/`pg-fails.mjs`, content overwritten
> between runs) rather than one file per script name shown in SCRIPTS.md.
> The names in SCRIPTS.md remain the canonical labels for each capture —
> the literal on-disk filename during capture doesn't matter structurally.


---

## 1. pg-connection-refused

node:internal/process/promises:394
    triggerUncaughtException(err, true /* fromPromise */);
    ^

Error: connect ECONNREFUSED 127.0.0.1:54321
    at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {
  errno: -111,
  code: 'ECONNREFUSED',
  syscall: 'connect',
  address: '127.0.0.1',
  port: 54321
}

Node.js v24.18.0

=== Shape B: console.error(err) ===
Error: connect ECONNREFUSED 127.0.0.1:54321
    at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {
  errno: -111,
  code: 'ECONNREFUSED',
  syscall: 'connect',
  address: '127.0.0.1',
  port: 54321
}

=== Shape C: console.error(err.stack) ===
Error: connect ECONNREFUSED 127.0.0.1:54321
    at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16)

---

## 2. nested-deps-statuses-flat

/home/vedant/stack-trace-bundler/errors-test/node_modules/statuses/index.js:110
    throw new Error('invalid status code: ' + code)
    ^

Error: invalid status code: 999999
    at getStatusMessage (/home/vedant/stack-trace-bundler/errors-test/node_modules/statuses/index.js:110:11)
    at status (/home/vedant/stack-trace-bundler/errors-test/node_modules/statuses/index.js:142:12)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/pg-fails.js:2:1)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)
    at node:internal/main/run_main_module:33:47

Node.js v24.18.0

---

## 3. json-parse-syntaxerror

=== Shape B: console.error(err) ===
SyntaxError: Expected property name or '}' in JSON at position 2 (line 1 column 3)
    at JSON.parse (<anonymous>)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/pg-fails.js:2:8)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)
    at node:internal/main/run_main_module:33:47

=== Shape C: console.error(err.stack) ===
SyntaxError: Expected property name or '}' in JSON at position 2 (line 1 column 3)
    at JSON.parse (<anonymous>)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/pg-fails.js:2:8)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)
    at node:internal/main/run_main_module:33:47
Cause defined? false
---

## 4. zero-stack-trace-limit

/home/vedant/stack-trace-bundler/errors-test/pg-fails.js:4
  throw new Error("Zero stack frames requested");
  ^

[Error: Zero stack frames requested]

Node.js v24.18.0

=== Shape C: console.error(err.stack) ===
Error: Zero stack frames requested

---

## 5. esm-runtime-error

file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:4
  throw new Error("ESM runtime failure");
        ^

Error: ESM runtime failure
    at main (file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:4:9)
    at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:7:1
    at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
    at async node:internal/modules/esm/loader:643:26
    at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)

Node.js v24.18.0

---

## 6. import-outside-module

(node:18169) Warning: Failed to load the ES module: /home/vedant/stack-trace-bundler/errors-test/pg-fails.js. Make sure to set "type": "module" in the nearest package.json file or use the .mjs extension.
(Use `node --trace-warnings ...` to show where the warning was created)
/home/vedant/stack-trace-bundler/errors-test/pg-fails.js:1
import fs from 'fs';
^^^^^^

SyntaxError: Cannot use import statement outside a module
    at wrapSafe (node:internal/modules/cjs/loader:1804:18)
    at Module._compile (node:internal/modules/cjs/loader:1845:20)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)
    at node:internal/main/run_main_module:33:47

Node.js v24.18.0

---

## 7. typescript-tsc-compiled

When directly ran `node app.ts`:

/home/vedant/stack-trace-bundler/errors-test/app.ts:2
  throw new Error("TS execution test error");
  ^

Error: TS execution test error
    at level3 (/home/vedant/stack-trace-bundler/errors-test/app.ts:2:9)
    at level2 (/home/vedant/stack-trace-bundler/errors-test/app.ts:4:21)
    at level1 (/home/vedant/stack-trace-bundler/errors-test/app.ts:5:21)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/app.ts:6:1)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)

Node.js v24.18.0

When ran through `npx --yes -p typescript tsc app.ts && node app.js`:

/home/vedant/stack-trace-bundler/errors-test/app.js:3
    throw new Error("TS execution test error");
    ^

Error: TS execution test error
    at level3 (/home/vedant/stack-trace-bundler/errors-test/app.js:3:11)
    at level2 (/home/vedant/stack-trace-bundler/errors-test/app.js:5:21)
    at level1 (/home/vedant/stack-trace-bundler/errors-test/app.js:6:21)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/app.js:7:1)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)

Node.js v24.18.0

---

## 8. esbuild-minified-bundle

/home/vedant/stack-trace-bundler/errors-test/dist/bundle.js:1
(()=>{var e=(o,n)=>()=>{try{return n||o((n={exports:{}}).exports,n),n.exports}catch(t){throw n=0,t}};var u=e(()=>{function l(){return null.nonExistentMethod()}l()});u();})();
                                                                                       ^

TypeError: Cannot read properties of null (reading 'nonExistentMethod')
    at l (/home/vedant/stack-trace-bundler/errors-test/dist/bundle.js:1:140)
    at /home/vedant/stack-trace-bundler/errors-test/dist/bundle.js:1:160
    at /home/vedant/stack-trace-bundler/errors-test/dist/bundle.js:1:39
    at /home/vedant/stack-trace-bundler/errors-test/dist/bundle.js:1:166
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/dist/bundle.js:1:172)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)

Node.js v24.18.0

---

## 9. scoped-package-swc

/home/vedant/stack-trace-bundler/errors-test/node_modules/@swc/core/index.js:309
            return bindings.transformSync(isModule ? stringifyProgram(src) : src, isModule, toBuffer(newOptions));
                            ^

Error: Failed to deserialize buffer as swc::config::Options
JSON: {"jsc":{"target":"invalid-target"}}

Caused by:
    Unknown ES version: invalid-target at line 1 column 35
    at Compiler.transformSync (/home/vedant/stack-trace-bundler/errors-test/node_modules/@swc/core/index.js:309:29)
    at Object.transformSync (/home/vedant/stack-trace-bundler/errors-test/node_modules/@swc/core/index.js:415:21)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/pg-fails.js:2:5)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)
    at node:internal/main/run_main_module:33:47 {
  code: 'GenericFailure'
}


---

## 10. aggregate-error

node:internal/modules/run_main:107
    triggerUncaughtException(
    ^

[AggregateError: All promises were rejected] {
  [errors]: [
    Error: first failure
        at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:2:18
        at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
        at async node:internal/modules/esm/loader:643:26
        at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5),
    Error: second failure
        at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:3:18
        at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
        at async node:internal/modules/esm/loader:643:26
        at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)
  ]
}

Node.js v24.18.0

=== Shape B: console.error(err) ===
[AggregateError: All promises were rejected] {
  [errors]: [
    Error: first failure
        at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:3:20
        at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
        at async node:internal/modules/esm/loader:643:26
        at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5),
    Error: second failure
        at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:4:20
        at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
        at async node:internal/modules/esm/loader:643:26
        at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)
  ]
}

=== Shape C: console.error(err.stack) ===
AggregateError: All promises were rejected

=== err.errors ===
[
  Error: first failure
      at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:3:20
      at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
      at async node:internal/modules/esm/loader:643:26
      at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5),
  Error: second failure
      at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:4:20
      at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
      at async node:internal/modules/esm/loader:643:26
      at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)
]

---

## 11. non-error-thrown

node:internal/modules/run_main:107
    triggerUncaughtException(
    ^
just a plain string, not an Error object
(Use `node --trace-uncaught ...` to show where the exception was thrown)

Node.js v24.18.0

==

node:internal/modules/run_main:107
    triggerUncaughtException(
    ^
{ code: 1, message: 'plain object, not an Error' }

Node.js v24.18.0
---

## 12. assert-multiline-diff

node:assert:152
  throw new AssertionError(obj);
  ^

AssertionError [ERR_ASSERTION]: Expected "actual" to be reference-equal to "expected":
+ actual - expected

  {
+   age: 30,
-   age: 31,
    name: 'Alice'
  }

    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/pg-fails.js:2:8)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)
    at node:internal/main/run_main_module:33:47 {
  generatedMessage: true,
  code: 'ERR_ASSERTION',
  actual: { name: 'Alice', age: 30 },
  expected: { name: 'Alice', age: 31 },
  operator: 'strictEqual',
  diff: 'simple'
}

Node.js v24.18.0

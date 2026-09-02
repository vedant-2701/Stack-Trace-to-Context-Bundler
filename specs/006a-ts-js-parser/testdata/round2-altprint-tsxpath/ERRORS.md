# Round 2 Capture Errors — Alternate Print Methods + tsx Path Confirmation

Paste raw output below each numbered heading, matching SCRIPTS.md exactly.
Leave a section blank (with a `(not run)` note) if you skip it rather than
deleting the heading — keeps numbering stable between the two files.

---

## 1. alt-print-with-cause

=== console.trace(err) ===
Trace: [TypeError: fetch failed] {
  [cause]: Error: connect ECONNREFUSED 127.0.0.1:59999
      at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {
    errno: -111,
    code: 'ECONNREFUSED',
    syscall: 'connect',
    address: '127.0.0.1',
    port: 59999
  }
}
    at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.mjs:5:11
    at process.processTicksAndRejections (node:internal/process/task_queues:104:5)

=== err.toString() ===
TypeError: fetch failed

=== JSON.stringify(err) ===
{}

=== console.dir(err) ===
[TypeError: fetch failed] {
  [cause]: Error: connect ECONNREFUSED 127.0.0.1:59999
      at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {
    errno: -111,
    code: 'ECONNREFUSED',
    syscall: 'connect',
    address: '127.0.0.1',
    port: 59999
  }
}

=== console.dir(err, { depth: null }) ===
[TypeError: fetch failed] {
  [cause]: Error: connect ECONNREFUSED 127.0.0.1:59999
      at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {
    errno: -111,
    code: 'ECONNREFUSED',
    syscall: 'connect',
    address: '127.0.0.1',
    port: 59999
  }
}

---

## 2. alt-print-no-cause

=== console.trace(err) ===
Trace: SyntaxError: Expected property name or '}' in JSON at position 2 (line 1 column 3)
    at JSON.parse (<anonymous>)
    at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.mjs:2:8
    at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
    at async node:internal/modules/esm/loader:643:26
    at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)
    at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.mjs:5:11
    at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
    at async node:internal/modules/esm/loader:643:26
    at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)

=== err.toString() ===
SyntaxError: Expected property name or '}' in JSON at position 2 (line 1 column 3)

=== JSON.stringify(err) ===
{}

=== console.dir(err) ===
SyntaxError: Expected property name or '}' in JSON at position 2 (line 1 column 3)
    at JSON.parse (<anonymous>)
    at file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.mjs:2:8
    at ModuleJob.run (node:internal/modules/esm/module_job:439:25)
    at async node:internal/modules/esm/loader:643:26
    at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)

---

## 3. tsx-full-transformer-path

/home/vedant/stack-trace-bundler/errors-test/app.ts:2
  throw new Error("TS execution test error");
        ^

Error: TS execution test error
    at level3 (/home/vedant/stack-trace-bundler/errors-test/app.ts:2:9)
    at level2 (/home/vedant/stack-trace-bundler/errors-test/app.ts:4:21)
    at level1 (/home/vedant/stack-trace-bundler/errors-test/app.ts:5:21)
    at <anonymous> (/home/vedant/stack-trace-bundler/errors-test/app.ts:6:1)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/app.ts:6:8)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object.transformer (/home/vedant/.npm/_npx/fd45a72a545557e9/node_modules/tsx/dist/register-C557imBs.cjs:9:3619)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)

Node.js v24.18.0

---

## 4. scoped-package-cause-body

=== Object.keys(err) ===
[ 'code' ]

=== err.cause ===
undefined

=== 'cause' in err ===
false

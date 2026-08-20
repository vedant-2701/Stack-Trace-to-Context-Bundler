# fetch-refused Capture Errors

Paste raw output below each numbered heading, matching SCRIPTS.md exactly.

---

## 1. fetch-refused-uncaught

node:internal/modules/run_main:107
    triggerUncaughtException(
    ^

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

Node.js v24.18.0

---

## 2. fetch-refused-caught

=== Shape B: console.error(err) ===
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

=== Shape C: console.error(err.stack) ===
TypeError: fetch failed

=== cause ===
Error: connect ECONNREFUSED 127.0.0.1:59999
    at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {
  errno: -111,
  code: 'ECONNREFUSED',
  syscall: 'connect',
  address: '127.0.0.1',
  port: 59999
}

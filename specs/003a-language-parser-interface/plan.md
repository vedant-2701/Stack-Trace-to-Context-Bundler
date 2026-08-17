# Plan: Language parser interface

Derived from `spec.md`. Must be consistent with `memory/constitution.md`.

## Architecture / approach

A new package, `internal/parser`, holds exactly two files for this feature:
the `LanguageParser` interface (`registry.go`) and its one sentinel error
(`errors.go`). No registration logic, no implementations -- both are out of
scope (003b, 005a/006a respectively). Validation for an interface with no
logic of its own can't come from `go test`; instead, two representative
example traces (one Java, one TS/JS, both constructed to be realistic rather
than pulled from a real incident) are hand-traced through `Detect()`/
`Parse()` as pseudocode below, checking that every field either example
trace contains has somewhere to go in the interface's return values.

## Stack & versions

Stdlib only: `context`, `errors`. No new third-party dependency.

## Data model

No new types. This feature only defines a Go interface and one sentinel
error consuming types already defined in `internal/contract` (001):
`contract.Language`, `contract.ExceptionNode`, `contract.Runtime`.

## File / module layout

```
internal/parser/
  registry.go   # LanguageParser interface + full doc-comment contract
  errors.go     # ErrUnparseable sentinel
```

## API / contracts

```go
// Package parser defines the LanguageParser interface every
// language-specific parser (005a Java, 006a TS/JS) implements, and the
// registry that auto-detects which implementation applies to a given raw
// trace (003b). This file (registry.go) currently holds only the
// interface -- the registration/detection logic is 003b's scope.
package parser

import (
	"context"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// LanguageParser is implemented once per source language (005a for Java,
// 006a for TypeScript/JavaScript). No package under internal/parser/<lang>/
// may import another language's parser package -- cross-language sharing
// happens only through this interface or genuinely language-agnostic code
// (internal/codecontext).
type LanguageParser interface {
	// Language identifies which contract.Language this implementation
	// produces. Provisional for the TS/JS implementation: TypeScript
	// compiles to .js before running, which may make JavaScript and
	// TypeScript indistinguishable from parsed trace content alone in the
	// common case -- see memory/known-gaps.md's entry owned by 006a. If
	// that turns out to be a real per-trace distinction rather than a
	// per-implementation constant, this method's shape will need to
	// change (e.g. moving into Parse's return) as part of 006a.
	Language() contract.Language

	// Detect reports whether rawTrace looks like this implementation's
	// language, as a fast, in-memory, side-effect-free check -- no I/O, no
	// subprocess calls, no ctx. rawTrace is guaranteed non-empty and
	// already bounded to internal/cli's 512KB cap
	// (contract.TruncateRawInput) -- implementations do not need to
	// defensively handle an empty string. The future auto-detection
	// registry (003b) may call Detect once per registered parser on every
	// invocation where the language isn't hinted via --lang, so cost here
	// is not free to ignore.
	Detect(rawTrace string) bool

	// Parse converts rawTrace into a linear exception chain and the
	// detected runtime. rawTrace is guaranteed non-empty and already
	// bounded to internal/cli's 512KB cap (contract.TruncateRawInput) --
	// implementations do not need to defensively handle an empty string.
	//
	// Contract:
	//   - Frames within each ExceptionNode are ordered exactly as they
	//     appear in the raw trace: Frames[0] is the frame where that
	//     node's exception was thrown. contract.ComputeFingerprint depends
	//     on this ordering.
	//   - Every Frame.Bucket in the returned chain is fully assigned --
	//     never partially bucketed. Neither ExceptionNode nor Frame has a
	//     field to represent a degraded bucketing state, so there is no
	//     partial-success case to fall back on.
	//   - Outcome is binary: either a complete, valid chain and runtime
	//     with a nil error, or a nil chain and zero-value runtime with a
	//     non-nil error. Never a partial result.
	//   - When rawTrace matched this language's general shape (Detect
	//     would return true) but could not actually be converted into a
	//     valid chain, the returned error wraps ErrUnparseable (via %w).
	//     Any other non-nil error is an unexpected internal error, not an
	//     expected parse failure -- callers distinguish the two via
	//     errors.Is(err, parser.ErrUnparseable).
	//   - ctx allows caller cancellation. This interface does not require
	//     or promise a caller-set deadline. An implementation that shells
	//     out internally (e.g. inferring Runtime.Version via
	//     contract.VersionSourceLocalEnvironment) is responsible for its
	//     own bounded timeout derived from ctx, matching
	//     internal/codecontext/runner.go's gitTimeout pattern.
	Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error)
}
```

```go
package parser

import "errors"

// ErrUnparseable is wrapped by a LanguageParser implementation's Parse
// method when rawTrace matched that language's general shape (Detect
// would return true) but could not actually be converted into a valid
// exception chain. Distinguishes this expected failure mode (mapped by
// the caller to CLI exit code 3) from an unexpected internal error
// (exit code 1). See registry.go's Parse doc comment.
var ErrUnparseable = errors.New("trace matched this language's shape but could not be parsed into a valid exception chain")
```

## Testing strategy

No `_test.go` for this feature -- an interface declaration plus one
sentinel error has no logic to unit test, and `INDEX.md` scopes 003a's
validation to hand-traced pseudocode, not automated tests. The pre-commit
gate (`gofumpt` -> `golangci-lint` -> `go build` -> `go test`) still runs
and passes trivially: `go build ./...` succeeds with zero implementations
of `LanguageParser` (Go interfaces don't require an implementation to
exist to compile), and `go test ./...` has nothing to run in this package
yet. Real behavioral testing of everything this interface pins down
happens in 005a and 006a, against their own implementations.

### Hand-traced validation: Java example

Representative, constructed trace (not from a real incident):

```
java.lang.RuntimeException: Failed to process order
    at com.example.orders.OrderProcessor.process(OrderProcessor.java:45)
    at com.example.orders.OrderService.handleOrder(OrderService.java:23)
    at com.example.api.OrderController.createOrder(OrderController.java:67)
    at java.base/java.lang.Thread.run(Thread.java:833)
Caused by: java.sql.SQLException: Connection timed out
    at com.mysql.cj.jdbc.ConnectionImpl.connect(ConnectionImpl.java:823)
    at com.zaxxer.hikari.pool.HikariPool.createPoolEntry(HikariPool.java:401)
    at com.example.orders.OrderRepository.save(OrderRepository.java:112)
    ... 2 more
```

Pseudocode walkthrough:

```
javaParser.Detect(rawTrace)
  -> true (matches "at <pkg>.<Class>.<method>(<File>.java:<line>)" shape,
     "Caused by:" marker)

javaParser.Language() -> contract.LanguageJava   // fixed, no ambiguity for Java

javaParser.Parse(ctx, rawTrace) ->
  chain: [
    ExceptionNode{
      ClassName: "java.lang.RuntimeException",
      Message:   "Failed to process order",
      ElidedFrameCount: 0,   // outermost node, nothing elided
      Frames: [
        {Index:0, FilePath:".../OrderProcessor.java",  ClassName:"com.example.orders.OrderProcessor",  MethodName:"process",        LineNumber:45,  Bucket:own},        // Frames[0] == throw site
        {Index:1, FilePath:".../OrderService.java",     ClassName:"com.example.orders.OrderService",     MethodName:"handleOrder",    LineNumber:23,  Bucket:own},
        {Index:2, FilePath:".../OrderController.java",  ClassName:"com.example.api.OrderController",     MethodName:"createOrder",    LineNumber:67,  Bucket:own},
        {Index:3, FilePath:"<jdk>/Thread.java",          ClassName:"java.lang.Thread",                     MethodName:"run",            LineNumber:833, Bucket:runtime},   // java.base module -> runtime, not own/dependency
      ],
    },
    ExceptionNode{
      ClassName: "java.sql.SQLException",
      Message:   "Connection timed out",
      ElidedFrameCount: 2,   // "... 2 more" -- shares OrderController.createOrder + Thread.run with the enclosing node
      Frames: [
        {Index:0, FilePath:".../ConnectionImpl.java", ClassName:"com.mysql.cj.jdbc.ConnectionImpl", MethodName:"connect",          LineNumber:823, Bucket:dependency, PackageName:"com.mysql:mysql-connector-j"}, // Frames[0] == throw site for this node
        {Index:1, FilePath:".../HikariPool.java",      ClassName:"com.zaxxer.hikari.pool.HikariPool", MethodName:"createPoolEntry", LineNumber:401, Bucket:dependency, PackageName:"com.zaxxer:HikariCP"},
        {Index:2, FilePath:".../OrderRepository.java", ClassName:"com.example.orders.OrderRepository", MethodName:"save",            LineNumber:112, Bucket:own},
      ],
    },
  ]
  runtime: {Name:"jvm", Version:"17.0.9", VersionSource:local-environment, Note:"inferred from local `java -version`; may not match the environment that produced this trace"}
  err: nil
```

Coverage check against the interface: two-node chain (outer + `Caused by`)
exercises `ElidedFrameCount` on the inner node, all three `Bucket` values,
`PackageName` on dependency frames, and a runtime-bucket JDK frame. Every
field has somewhere to go -- no gap found.

### Hand-traced validation: TS/JS example

Representative, constructed trace (Node, `util.inspect`-style cause
rendering, user-code-caught-and-logged -- not the uncaught-crash path, so
no trailing `Node.js vX.Y.Z` line):

```
TypeError: Failed to fetch user
    at fetchUser (/app/src/services/userService.js:34:15)
    at async processRequest (/app/src/controllers/userController.js:12:22)
    at async Layer.handle [as handle_request] (/app/node_modules/express/lib/router/layer.js:95:5) {
  [cause]: Error: connect ECONNREFUSED 127.0.0.1:5432
      at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1595:16)
}
```

Pseudocode walkthrough:

```
tsJsParser.Detect(rawTrace)
  -> true (matches "at <fn> (<path>:<line>:<col>)" shape, "[cause]:" marker)

tsJsParser.Language() -> contract.LanguageJavaScript
  // every frame's FilePath ends in .js -- no .ts/.tsx anywhere in this
  // trace, so LanguageJavaScript is the only defensible static answer
  // here, which is exactly the case memory/known-gaps.md's 003a entry
  // flags: a compiled-then-node-run TS project can produce a trace
  // indistinguishable from plain JS.

tsJsParser.Parse(ctx, rawTrace) ->
  chain: [
    ExceptionNode{
      ClassName: "TypeError",
      Message:   "Failed to fetch user",
      ElidedFrameCount: 0,   // omitted/0 for JS/TS -- V8 doesn't elide shared frames
      Frames: [
        {Index:0, FilePath:"/app/src/services/userService.js",     MethodName:"fetchUser",      LineNumber:34, ColumnNumber:15, Bucket:own},        // Frames[0] == throw site
        {Index:1, FilePath:"/app/src/controllers/userController.js", MethodName:"processRequest", LineNumber:12, ColumnNumber:22, Bucket:own},
        {Index:2, FilePath:"/app/node_modules/express/lib/router/layer.js", MethodName:"Layer.handle", LineNumber:95, ColumnNumber:5, Bucket:dependency, PackageName:"express"},
      ],
    },
    ExceptionNode{
      ClassName: "Error",
      Message:   "connect ECONNREFUSED 127.0.0.1:5432",
      ElidedFrameCount: 0,
      Frames: [
        {Index:0, FilePath:"node:net", MethodName:"TCPConnectWrap.afterConnect", LineNumber:1595, ColumnNumber:16, Bucket:runtime}, // node: prefix -> runtime, not own/dependency; Frames[0] == throw site for this node
      ],
    },
  ]
  runtime: {Name:"node", VersionSource:unknown, Note:"no trailing Node.js version line in trace (not the uncaught-crash path); local-environment inference not attempted for this example"}
  err: nil
```

Coverage check against the interface: demonstrates `ColumnNumber` (JS/TS
only), `ElidedFrameCount` omitted/0, a `node:`-prefixed runtime frame
distinct from Java's JDK-module case, a `node_modules`-prefixed dependency
frame, and a `VersionSourceUnknown` runtime provenance state (distinct from
the Java example's `local-environment` state, for provenance-state
coverage). Also concretely illustrates the `Language()` open question from
`known-gaps.md`: this trace is real-shaped and 100% `.js`, so a TypeScript
source origin would be undetectable from content alone here -- exactly the
case 006a needs to resolve.

## Risks & open decisions

- `Language()`'s viability as a static per-implementation method is
  unresolved pending 006a -- see `memory/known-gaps.md`. The TS/JS
  hand-trace above demonstrates the concrete case that makes this an open
  risk rather than a hypothetical one.
- `Detect()`'s "fast, in-memory, no false positives/negatives between Java
  and TS/JS" property is asserted here but not actually tested -- no real
  implementation exists yet. Heuristic accuracy is 003b/005a/006a's concern
  once real parsers exist to test against (`INDEX.md`'s own note on 003b).

## Alternatives considered

- **A separate `Bucketize()` interface method.** Rejected: `Frame.Bucket`
  is already a field on the same struct `Parse()` returns
  (`contract.Frame`), and `internal/codecontext/context.go` already
  filters on `Bucket` assuming it's populated by the time it sees a chain.
  Nothing in the pipeline has an unbucketed chain that needs a second call
  to bucket it -- a separate method would add an unused seam.
- **A `Confidence()` / numeric return from `Detect()`**, instead of a plain
  `bool`. Rejected: CLI exit code 4 (ambiguous/undetected) is fully
  determined by counting how many registered parsers' `Detect()` returned
  `true` across the registry (0 = undetected, 2+ = ambiguous, 1 = proceed)
  -- no tie-breaking use case exists to justify a richer return type, and
  `INDEX.md` scopes ambiguity-resolution logic to 003b, not this feature.
- **A Postgres-style error-code taxonomy** (single umbrella code, multiple
  underlying reasons) layered onto parser errors. Rejected for v1: exactly
  one sentinel (`ErrUnparseable`) is needed today, for exactly one
  distinction (exit 3 vs exit 1). Building taxonomy infrastructure ahead of
  having enough distinct failure reasons to justify grouping is the
  speculative capability Article VIII rules out. Revisit additively once
  005a/006a/005b/006b have accumulated enough real sentinels that grouping
  them would actually help a caller.
- **Folding `contract.Language` into `Parse()`'s return** instead of a
  static `Language()` method, proposed mid-interrogation after noticing TS
  compiles to `.js` before running. Not rejected outright -- deferred:
  kept `Language()` for v1, flagged in `memory/known-gaps.md` (owned by
  006a) rather than decided here without real trace evidence either way.

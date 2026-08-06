# Progress Log: Data contract

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-08-05
**Task(s):** T000
**What happened:** Bootstrapped the Go module
(`github.com/vedant-2701/stack-trace-bundler`, go 1.25.0) and dev tooling.
Hit and fixed two real bugs along the way:
1. `AGENTS.md`'s documented `golangci-lint` install command was wrong —
   missing `/v2/` in the module path, so `go install
   .../golangci-lint/cmd/golangci-lint@latest` resolved to v1 even though
   `.golangci.yml` is v2-schema. Fixed in `AGENTS.md`.
2. A synthetic verification test (deliberately unformatted file named with
   a leading underscore) gave misleading `build`/`lint`/`test` failures —
   turned out to be because Go's toolchain ignores underscore-prefixed
   files entirely, not a real pipeline problem. Re-verified conceptually:
   `format`'s `test -z "$(gofumpt -l {staged_files})"` gate is confirmed
   working correctly (it operates on literal staged paths, bypassing the
   package-discovery mechanism the underscore trick fools) — it caught the
   bad formatting and blocked the commit as expected.
**Deviations from plan (if any):** None to the contract shape itself.
T000 was added as a prerequisite task after the fact (wasn't in the
original tasks.md) once the module-init gap was discovered.
**New open questions:** None.

---

**Date:** 2026-08-05
**Task(s):** T001
**What happened:** Wrote `internal/contract/types.go` — every struct from
`plan.md`'s Data model (`Bundle`, `Runtime`, `ExceptionNode`, `Frame`,
`FrameRef`, `Snippet`, `CodeContext`, `BlameEntry`, `GitMetadata`,
`Dependencies`, `LockedDependency`), plus typed-string enums with
constants for `Language`, `OS`, `Bucket`, `CodeContextStatus`,
`VersionSource`, `ManifestFile` (an implementation-level choice beyond
what plan.md specified verbatim, for compile-time safety; doesn't change
the JSON shape). `omitempty` applied per the cross-cutting rule, withheld
on `RawInputTruncated` and `VersionSource` (both always-present, not
not-applicable cases).
Verified via real command output (not assumed): `gofumpt -w`/`-l` clean,
`go build ./...` clean, `go vet ./...` clean, `golangci-lint run ./...`
clean after fixing 4 legitimate `revive` "exported const needs comment"
findings (comment was on the `type` line, not associated with the `const`
block below it due to the blank line between them — fixed by adding
per-const comments, matching the style already used elsewhere in the
file).
**Deviations from plan (if any):** Added typed consts/enums beyond the
plain-string fields plan.md described — internal Go idiom only, JSON
output is identical either way.
**New open questions:** None.

---

**Date:** 2026-08-06
**Task(s):** T002
**What happened:** Added exported `SchemaVersion = "1.0.0"` constant to
`internal/contract/types.go`, with a doc comment defining semver bump
triggers (MAJOR for breaking shape changes, MINOR for additive-only
fields, no bump for implementation-only changes). Added
`internal/contract/types_test.go` with `TestSchemaVersion` asserting
`SchemaVersion == "1.0.0"`.
Verified via real command output (paste-back from user, not assumed):
`go build ./...`, `go test ./internal/contract/...`,
`golangci-lint run ./internal/contract/...`, and `gofumpt -l` on both
changed files all clean.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-08-06
**Task(s):** T003
**What happened:** Added `internal/contract/fingerprint.go` with
`ComputeFingerprint(chain []ExceptionNode) string`. Per node: hashes
every own-bucket frame's file+className+methodName (excluding line
numbers) plus the originating frame (`Frames[0]`) if not already
own-bucket, joined with ASCII info-separator bytes to prevent field/frame
collisions, then SHA-256 over the whole chain, truncated to 16 hex chars.
Added `internal/contract/fingerprint_test.go`, table-driven, covering:
dependency version bump on a non-originating frame -> same fingerprint;
different own-bucket frame identity -> different fingerprint; identical
outer wrapper with a different inner cause -> different fingerprint
(proves every node is hashed, not just the outermost); plus two extra
guards (line-number-only difference -> same fingerprint; determinism
across repeated calls).
Verified via real command output (paste-back from user, not assumed):
`go build ./...`, `go test ./internal/contract/... -run
TestComputeFingerprint -v`, `golangci-lint run ./internal/contract/...`,
and `gofumpt -l` on both changed files all clean.
**Deviations from plan (if any):** None to the algorithm's functional
guarantees. One unresolved implementation assumption not pinned by
spec.md/plan.md: "the originating frame" is implemented as `Frames[0]`
(top-of-stack, throw-site convention) since neither doc specifies it
explicitly. Documented in the `ComputeFingerprint` doc comment; flagged
for the user to confirm against actual frame ordering when 005a (Java)
and 006a (TS/JS) are specced, since those parsers are what actually
populate `Frames[]` order (recorded outside this repo, in the user's own
cross-session notes for this project, since neither 005a nor 006a has a
spec.md yet to hold it).
**New open questions:** Frame-ordering assumption above, to be resolved
during 005a/006a spec interrogation, not before.

---

**Date:** 2026-08-06
**Task(s):** T004
**What happened:** Added tests to `internal/contract/types_test.go`: a
generic `roundTrip[T any]` helper plus round-trip (marshal/unmarshal/
compare) tests for every struct in the package (`Runtime`,
`ExceptionNode`, `Frame`, `FrameRef`, `Snippet`, `CodeContext`,
`BlameEntry`, `GitMetadata`, `Dependencies`, `LockedDependency`, plus one
fully nested `Bundle`), and five targeted raw-JSON key-presence tests for
the specific `omitempty` behaviors tasks.md calls out: `Frame.ColumnNumber`
absent for a Java-shaped frame / present for a JS/TS-shaped frame;
`LockedDependency` omits `version`/includes `note` when unresolved and the
reverse when resolved; `Bundle.RawInputTruncated` always present (both
`true` and `false`).
Design note recorded in the test file's own doc comment: round-trip
comparison alone cannot catch a missing/wrong `omitempty` (an omitted
zero value and a present zero value decode back to the same Go zero value
either way), so the key-presence tests inspect marshaled JSON via
`map[string]any`, not just struct-to-struct equality.
Verified via real command output (paste-back from user, not assumed):
`go build ./...`, `go test ./internal/contract/... -v`,
`golangci-lint run ./internal/contract/...`, and `gofumpt -l` on the
changed file all clean.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-08-06
**Task(s):** T005 (scoping, before implementation)
**What happened:** T005's original wording ("a fully-populated example
`Bundle` covering both a Java-flavored and a TS/JS-flavored path" ->
single `testdata/example.json`) was ambiguous: `Bundle.Language` is a
single field, so one bundle can't structurally represent two languages at
once for any real trace. Resolved with the user: since the shipped binary
selects one parser per invocation (factory-style dispatch by
runtime/language), a single invocation only ever produces one `Bundle` in
one language's shape -- a chimera fixture mixing both would misrepresent
what any real code path produces and could mislead 007/008 implementers
reading it as a reference. Decision: two separate example bundles and two
separate fixtures, `testdata/example_java.json` (realistic Java shape --
no `columnNumber`, `pom.xml`/`build.gradle` manifest) and
`testdata/example_ts.json` (realistic TS/JS shape -- `columnNumber` set,
`package.json` manifest), each with its own golden byte-for-byte test.
Amended `tasks.md`'s T005 wording and `plan.md`'s file layout + testing
strategy sections to match, rather than silently implementing against a
different shape than what those files describe.
**Deviations from plan (if any):** Yes -- plan.md and tasks.md both
amended in this same session, see diffs. Original single-fixture wording
was the deviation-worthy assumption, not this correction.
**New open questions:** None -- resolved.

---

**Date:** 2026-08-06
**Task(s):** T005 (implementation)
**What happened:** Added `exampleJavaBundle()` and `exampleTSBundle()` to
`internal/contract/types_test.go` -- two fully-populated, realistic
`Bundle` builders (per the scoping decision above): Java example with two
own-bucket frames in different files (`Handler.java`, `Repository.java`),
each with its own `codeContexts[].blame` entry, an elided-frame Java
`Caused by` chain, no `columnNumber` anywhere, `pom.xml` manifest, and one
unresolved `locked` dependency (`version` omitted, `note` set). TS/JS
example: two own-bucket frames in different files (`handler.ts`,
`service.ts`), `columnNumber` set on all frames, `Error.cause`-style
chain, `package.json` manifest, one unresolved `locked` dependency.
`Fingerprint` on both is computed via the real `ComputeFingerprint`, not
hardcoded, so the fixtures can never drift from the actual algorithm.
Added `TestGolden_ExampleJava`/`TestGolden_ExampleTS` plus a shared
`assertGolden` helper (marshal, compare byte-for-byte against
`testdata/example_*.json`; with `-update`, (re)writes the fixture
instead, the only way either fixture is ever produced, per Article IV).
Generated both fixtures via `go test ./internal/contract/... -run
TestGolden -update`.
Caught and fixed two hand-alignment mistakes in gofmt-style struct/map
literal column padding before the user ran anything (verified with a real
`gofmt` installed in a sandbox via apt, since gofumpt itself couldn't be
fetched -- its dependencies aren't on the network allowlist). This isn't
a substitute for the user's own `gofumpt -l`/`golangci-lint`/`go
build`/`go test` run, which remains the actual gate.
Verified via real command output (paste-back from user, not assumed):
`go test ./internal/contract/... -run TestGolden -update` (fixture
generation), then `go build ./...`, `go test ./internal/contract/... -v`,
`golangci-lint run ./internal/contract/...`, `gofumpt -l
internal/contract/types_test.go` all clean.
**Deviations from plan (if any):** None beyond the T005-scoping entry
above.
**New open questions:** None.

---

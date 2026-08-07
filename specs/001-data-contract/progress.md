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

**Date:** 2026-08-06
**Task(s):** T006 (scoping, before implementation)
**What happened:** Two gaps in T006's original wording, neither pinned by
spec.md/plan.md: (1) whether "512 KB" is 1024- or 1000-based -- resolved
as 524288 bytes (512*1024), the standard binary convention for a
software byte cap. (2) Whether byte-exact truncation (`s[:524288]`,
potentially cutting a multi-byte UTF-8 rune in half) or rune-safe
truncation is correct. User raised whether exceeding input should split
into multiple parts instead of discarding the remainder -- rejected:
`rawInput` is parse-fallback only (chain[] is the primary payload), and
the cap's stated purpose (spec.md non-functional requirements) is to
bound total bundle size for clipboard/LLM-context use; keeping the
remainder anywhere defeats that purpose and would need a schema change
(new field, schemaVersion bump) for no identified need (Article VIII).
Decision: keep discard-past-cap, but make the cut point UTF-8-rune-safe
(back up at most 3 bytes to the last valid rune boundary if the exact cap
falls mid-rune) rather than byte-exact, to eliminate the
invalid-UTF-8/corrupted-last-character risk the user flagged. Amended
tasks.md's T006 wording and acceptance criteria to match (byte-exact only
guaranteed for the common case; rune-boundary backup documented as the
correct behavior when the cap falls mid-rune).
**Deviations from plan (if any):** Yes -- tasks.md amended in this same
session, see diff. Original byte-exact-truncation wording was the
deviation-worthy gap, not this correction.
**New open questions:** None -- resolved.

---

**Date:** 2026-08-06
**Task(s):** T006 (implementation)
**What happened:** Added `internal/contract/rawinput.go` with
`TruncateRawInput(s string) (out string, truncated bool)` and the
`rawInputCapBytes = 512 * 1024` constant. Returns s unchanged with
`truncated=false` if it already fits; otherwise cuts to at most
`rawInputCapBytes`, backing up (via `unicode/utf8.RuneStart`, at most 3
bytes) to the last valid rune boundary if the exact cap falls mid-rune,
per the T006 scoping decision above. Added
`internal/contract/rawinput_test.go`, covering the four boundary cases
from the amended acceptance criteria: one byte under (unchanged, false),
exactly at the cap (unchanged, false), one byte over with ASCII input
(truncated to exactly 524288 bytes, true), and a multi-byte-rune input
where the cap falls mid-rune (truncated to the nearest valid rune
boundary, `utf8.ValidString` true, true).
Before the user ran anything: verified both new files with a real `go
build`/`go vet`/`go test -v` (all passing, including every prior test in
the package) and `gofmt -l` (clean) in a sandboxed Go 1.22 toolchain
installed via apt, using a mirrored copy of the package. This caught real
issues twice during T005 (see that entry) and confirms these two new
files compile and behave correctly independent of the user's own run
-- but is not a substitute for it, since gofumpt itself couldn't be
fetched (dependency chain requires golang.org/x/*, not on the network
allowlist) and the sandbox's Go 1.22 differs from the project's actual Go
1.25.0. User asked that further exploratory tool-installation attempts
stop, since they cost tokens for marginal benefit once the user is
willing to run commands directly -- noted for future sessions: verify
with real tools when a quick, working path exists, then hand off to the
user rather than iterating on installation workarounds.
Verified via real command output (paste-back from user, not assumed):
`go build ./...`, `go test ./internal/contract/... -v`,
`golangci-lint run ./internal/contract/...`, `gofumpt -l
internal/contract/rawinput.go internal/contract/rawinput_test.go` all
clean.
**Deviations from plan (if any):** None beyond the T006-scoping entry
above.
**New open questions:** None.

---

**Date:** 2026-08-06
**Task(s):** Feature close-out (final acceptance-criteria pass)
**What happened:** Cross-checked all 15 functional requirements against
the actual `internal/contract` code (all satisfied) and all 10 acceptance
criteria against actual tests. Found 5 of 10 acceptance criteria describe
end-to-end behavior ("when parsed", "when the bundle is built") that this
feature was never scoped to deliver -- it defines the contract layer
only (structs, `ComputeFingerprint`, `TruncateRawInput`, hand-built
fixtures), not a parser or pipeline. Checked ownership against
`specs/INDEX.md`: all 5 already map to an existing feature (005a, 006a,
004, 005b, 006b) -- no orphaned criterion. Checked off the 5 satisfied
criteria in `spec.md` with pointers to the tests that satisfy them; left
the other 5 unchecked with inline `**Deferred to X**` notes; updated
`spec.md`'s stale header Status line.
Created `memory/deferred-acceptance-criteria.md` (on-disk, not just
session memory, so it survives an account change or memory reset) as the
source of truth for these 5 deferred items, plus a matching entry in
Claude's cross-session memory for this project. Added one line to
`AGENTS.md`'s workflow step 2 pointing every future feature kickoff at
that file. Also fixed an unrelated stale reference found in the same
file while there: `AGENTS.md`'s Boundaries section still named the
pre-amendment single-fixture design (`example.json` /
`contract_test.go`); corrected to the actual two-fixture design
(`example_java.json`/`example_ts.json`, generated by `types_test.go`).
Updated `specs/INDEX.md`: feature 001's status `in-progress` -> `done`.
User is separately deleting `data-contract.md` (the original reference
doc this struct was built from), now redundant per constitution Article
IV now that `internal/contract` exists as the actual single source of
truth.
**Deviations from plan (if any):** None -- this is close-out
bookkeeping, not implementation.
**New open questions:** None. 5 acceptance criteria remain genuinely open
but are now tracked, not silently dropped -- see
`memory/deferred-acceptance-criteria.md`.

---

Feature 001-data-contract is complete: all tasks done, all satisfiable
acceptance criteria checked off, remaining criteria tracked for the
features that own them. `specs/INDEX.md` status: done.

---

**Date:** 2026-08-07
**Task(s):** PR review fixes (Copilot automated review, post-close-out)
**What happened:** Three findings from an automated PR review, all
confirmed valid and fixed:
1. `rawinput.go`'s doc comment claimed 3 bytes is "the longest a UTF-8
   rune can be" -- wrong, UTF-8 runes are up to 4 bytes. The code's
   "back up at most 3 bytes" was already correct (a 4-byte rune has 3
   continuation bytes), only the rationale text was wrong. Fixed wording,
   no behavior change.
2. `plan.md`'s "Architecture / approach" section still had a stale
   `testdata/example.json` (singular) reference -- missed during the
   T005 scoping amendment, which only caught the file-layout and
   testing-strategy sections, not this one. Fixed to match the actual
   two-fixture design.
3. `types_test.go`'s `minimalBundle()` didn't set `Dependencies.ManifestFile`,
   so it marshaled `"manifestFile":""` -- not a valid enum value per the
   contract. Low-stakes today (only used for a single-field
   `rawInputTruncated` assertion) but a real correctness gap that would
   bite anyone reusing the helper. Fixed to set `ManifestFilePomXML`.
Verified via real command output (paste-back from user, not assumed):
`go build ./...`, `go test ./internal/contract/... -v`,
`golangci-lint run ./internal/contract/...`, `gofumpt -l` on all three
changed files, all clean.
**Deviations from plan (if any):** None -- corrections to existing work,
not new scope.
**New open questions:** None.

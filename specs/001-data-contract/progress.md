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

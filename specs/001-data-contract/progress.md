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

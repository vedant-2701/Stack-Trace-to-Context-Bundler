# Progress Log: Language parser interface

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-08-17
**Task(s):** Spec kickoff — interrogation, spec.md, plan.md, tasks.md, INDEX.md status update.
**What happened:**
- Read AGENTS.md, constitution.md, CONVENTIONS.md, INDEX.md fresh. Confirmed
  003a's only dependency (001) is done.
- Resolved a stale-memory discrepancy: 003 was previously one feature, split
  into 003a (interface, this feature) and 003b (auto-detection registry)
  during 006a's spec creation. 003a now precedes 005a/006a in dependency
  order, not the reverse. 006a shows `specifying` in INDEX.md because it's
  in progress on branch `feature/006a-ts-js-parser`, not visible on disk
  from `feature/003a-language-parser-interface`.
- Interrogated one question at a time; full decision set:
  - `LanguageParser` interface: `Language() contract.Language`,
    `Detect(rawTrace string) bool`,
    `Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error)`.
  - No `Bucketize()` method — bucket is already a `Frame` field; `Parse()`
    must return it fully assigned.
  - No `Confidence()` on `Detect()` — plain `bool` is sufficient; ambiguity
    (exit code 4) is a count-of-matches decision, owned by 003b.
  - `Frames[0]` pinned as the originating/throw-site frame, raw-trace order
    preserved — binding doc-comment requirement, not just convention.
  - `Parse()` is binary: full chain+runtime or full error, never partial
    (schema has no degraded-state field on `ExceptionNode`/`Frame`).
  - Sentinel `ErrUnparseable` declared in new `internal/parser/errors.go`
    (not `registry.go`), wrapped via `%w` for recognized-but-unparseable
    input; distinguishes CLI exit 3 from exit 1.
  - Error-code taxonomy (Postgres-style, umbrella code + multiple reasons)
    explicitly deferred — not built now, plain sentinels only; revisit once
    005a/006a/005b/006b accumulate enough real sentinels to justify it.
    Recorded in plan.md's Alternatives considered.
  - `ctx` in `Parse()` for cancellation only, no caller-set-deadline
    guarantee; each parser self-imposes its own subprocess timeout,
    matching `internal/codecontext/runner.go`'s `gitTimeout` pattern
    (verified directly).
  - Acceptance criteria are hand-traced-pseudocode-based (per INDEX.md),
    not `go test`-based — no `_test.go` for this feature.
  - Trace examples for the hand-trace: constructed (not real incident data)
    — one Java (`RuntimeException` + `Caused by: SQLException`), one TS/JS
    (`TypeError` + `[cause]` chain).
  - `registry.go` in 003a = interface + doc comments only, no registration
    map/`Register()` — that's 003b's scope.
  - Mid-interrogation correction: initially proposed dropping `Language()`
    in favor of returning `contract.Language` from `Parse()`, reasoning
    that a single `typescript` package must produce either
    `LanguageJavaScript` or `LanguageTypeScript`. Corrected by the user: TS
    compiles to `.js` before running, so this is likely moot in practice
    (traces are just `.js`) — deferred to 006a with real trace evidence
    rather than decided here. `Language()` kept as a static method,
    flagged provisional in its doc comment and in `known-gaps.md`.
- Added an entry to `memory/known-gaps.md`'s deferred-acceptance-criteria
  table: `Language()`'s static-method viability, owned by 006a.
- Wrote `spec.md`, `plan.md` (including both hand-traced pseudocode
  walkthroughs), `tasks.md` (T001–T006).
- Updated `specs/INDEX.md`: 003a status `idea` → `planned`.
**Deviations from plan (if any):** None — this session followed the
AGENTS.md kickoff sequence as directed.
**New open questions:** None remaining for 003a itself. `Language()`'s
long-term shape is an open question owned by 006a (tracked in
`known-gaps.md`, not here).

---

**Date:** 2026-08-17
**Task(s):** T001 — created `internal/parser/registry.go`.
**What happened:**
- Updated `specs/INDEX.md`'s 003a row: `planned` → `in-progress`, ahead of
  starting T001.
- Created `internal/parser/registry.go` with the `LanguageParser` interface
  (`Language`, `Detect`, `Parse`), transcribed from `plan.md`'s
  API/contracts block with no changes — doc comments cover frame ordering
  (`Frames[0]` = throw site), full-bucket-assignment, `Parse`'s
  binary-outcome/`ErrUnparseable`-wrapping rule, `ctx`'s
  cancellation-only contract, `Detect`'s no-I/O constraint, the `rawTrace`
  non-empty precondition, and `Language`'s provisional-viability note
  pointing at `known-gaps.md`.
- Verified all referenced `contract` symbols (`Language`, `ExceptionNode`,
  `Runtime`, `VersionSourceLocalEnvironment`) exist in
  `internal/contract/types.go` with matching names before treating the
  file as correct.
- Confirmed via `git branch --show-current` that work is on
  `feature/003a-language-parser-interface`.
- User ran `gofumpt -l internal/parser/registry.go`,
  `golangci-lint run ./internal/parser/...`, and `go build ./...` locally;
  reported all three passed with 0 errors. Not independently verified by
  Claude beyond the symbol check above — no tool available to execute
  commands on the user's machine.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-08-17
**Task(s):** T002 — created `internal/parser/errors.go`.
**What happened:**
- Created `internal/parser/errors.go` with the `ErrUnparseable` sentinel,
  transcribed verbatim from `plan.md`'s API/contracts block, per spec.md
  requirement 5.
- No comment placed directly above `package parser` in this file —
  `registry.go` already carries the sole package doc comment, per
  `CONVENTIONS.md`'s `revive`/`package-comments` rule.
- User ran `gofumpt -l internal/parser/errors.go`,
  `golangci-lint run ./internal/parser/...`, `go build ./...`, and
  `go test ./...` locally; reported all passed with 0 errors. Not
  independently verified by Claude — no tool available to execute
  commands on the user's machine.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-08-17
**Task(s):** T003 — re-verified Java hand-trace against shipped `registry.go`.
**What happened:**
- Re-checked the Java hand-trace pseudocode (`RuntimeException` +
  `Caused by: SQLException`) from `plan.md` against the actual
  `internal/parser/registry.go` produced by T001.
- Confirmed: `Frames[0]`-as-throw-site holds for both nodes; all three
  `Bucket` values exercised; `PackageName` (`group:artifact` form) matches
  `contract.Frame.PackageName`'s doc comment; `ElidedFrameCount: 2` matches
  the field's type/semantics; `Runtime{Name:"jvm",
  VersionSource:local-environment}` matches `contract.Runtime` and
  `VersionSourceLocalEnvironment`; return shape matches `Parse`'s actual
  signature exactly.
- No drift found. No correction needed to `registry.go` or `plan.md`.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-08-17
**Task(s):** T004 — re-verified TS/JS hand-trace against shipped `registry.go`.
**What happened:**
- Re-checked the TS/JS hand-trace pseudocode (`TypeError` with `[cause]`
  chain, `ECONNREFUSED`) from `plan.md` against the actual
  `internal/parser/registry.go` produced by T001.
- Confirmed: `Language()`'s doc comment still explicitly flags static
  return as provisional for this exact JS-vs-TS-indistinguishable case,
  pointing at `known-gaps.md`; `ColumnNumber` matches
  `contract.Frame.ColumnNumber` (JS/TS-only, omitempty);
  `ElidedFrameCount: 0` matches (V8 doesn't elide); the `node:net` runtime
  frame (no `PackageName`) and `node_modules` dependency frame
  (`PackageName:"express"`) match `contract.Frame`'s bucket/PackageName
  rules; `Runtime{Name:"node", VersionSource:unknown}` matches
  `VersionSourceUnknown`; return shape matches `Parse`'s actual signature.
- No drift found. No correction needed to `registry.go` or `plan.md`.
  `Language()`/JS-vs-TS ambiguity this example demonstrates still holds
  against the real doc comment, unchanged from `plan.md`.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-08-17
**Task(s):** T005 — pre-commit gate.
**What happened:**
- No files changed by T003 or T004 (both pure-verification tasks, no
  drift found in either). No new code since T002's gate run, which
  already passed (`gofumpt -l`, `golangci-lint run`, `go build ./...`,
  `go test ./...`, all reported 0 errors). Closing T005 without
  re-running the gate, per user's explicit reasoning — not silently
  assumed.
**Deviations from plan (if any):** None — gate not literally re-run this
task, by agreement with user since nothing changed.
**New open questions:** None.

---

**Date:** 2026-08-17
**Task(s):** T006 — updated `specs/INDEX.md` status, confirmed `known-gaps.md`.
**What happened:**
- Updated `specs/INDEX.md`'s 003a row: `in-progress` → `done`.
- Re-checked `memory/known-gaps.md`'s 003a-owned row (`Language()`'s
  static-per-implementation viability, owner 006a, status pending)
  against the shipped `Language()` doc comment in `registry.go` — matches
  exactly, no edit needed. 003a is a source of this deferred criterion,
  not an owner of anything itself, so nothing else to check off here.
- 003a is now fully complete: T001–T006 all closed, spec.md's four
  acceptance criteria all satisfied per the T001–T005 record above.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

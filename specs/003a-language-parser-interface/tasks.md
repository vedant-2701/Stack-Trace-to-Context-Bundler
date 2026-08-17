# Tasks: Language parser interface

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.

- [ ] **T001** — Create `internal/parser/registry.go` with the
  `LanguageParser` interface (`Language`, `Detect`, `Parse`) and full
  doc comments per spec.md requirements 1-4, 6-11 (requirement 5's
  wrapping-behavior documentation also lives in this file's `Parse` doc
  comment; the sentinel's actual declaration is T002).
  - Depends on: none
  - Acceptance: File exists, defines the three-method interface with doc
    comments covering frame ordering (`Frames[0]` = throw site), bucket
    completeness, `Parse`'s binary-outcome/`ErrUnparseable`-wrapping rule,
    `ctx`'s cancellation-only contract, `Detect`'s no-I/O constraint, the
    `rawTrace` non-empty precondition, and `Language`'s provisional-viability
    note (pointing at `memory/known-gaps.md`). `registry.go`'s imports are
    reviewed and confirmed to be only `internal/contract` and stdlib
    (`context`) — nothing from any `parser/<lang>/` package. `gofumpt -l`
    and `golangci-lint run` pass on the file.

- [ ] **T002** — Create `internal/parser/errors.go` with the
  `ErrUnparseable` sentinel per spec.md requirement 5.
  - Depends on: T001
  - Acceptance: File exists; declares `var ErrUnparseable = errors.New(...)`
    with a doc comment; no comment directly above `package parser` in this
    file (T001's `registry.go` already carries the sole package doc comment
    — `revive`'s `package-comments` check, per `CONVENTIONS.md`). `gofumpt
    -l`, `golangci-lint run`, and `go build ./...` all pass with zero
    `LanguageParser` implementations existing yet.

- [ ] **T003** — Re-verify the Java hand-trace (`RuntimeException` +
  `Caused by:` `SQLException`, own/dependency/runtime frames) already
  written in `plan.md` against the actual `internal/parser/registry.go`
  produced by T001. The pseudocode walkthrough itself was written during
  spec kickoff (see `progress.md`), before `registry.go` existed as a real
  file — the walkthrough was checked against the plan.md-only design at
  that time, not against a shipped interface. This task re-checks it now
  that one exists.
  - Depends on: T001
  - Acceptance: Every field the example trace contains (chain, frames, all
    three buckets, `PackageName`, `ElidedFrameCount`, runtime version
    provenance) still maps cleanly onto `registry.go`'s actual method
    signatures and doc comments, with no gap or drift identified. If a
    mismatch turns up between the pseudocode and the shipped interface,
    fix whichever is wrong and record the correction in `progress.md`. If
    none turns up, record "re-verified against registry.go, no drift" in
    `progress.md`.

- [ ] **T004** — Re-verify the TS/JS hand-trace (`TypeError` with `[cause]`
  chain, own/dependency/runtime frames, column numbers) already written in
  `plan.md` against `registry.go`, the same way as T003.
  - Depends on: T001
  - Acceptance: Same completeness bar as T003, re-checked against the
    shipped interface rather than the plan.md-only design; explicitly
    reconfirms the `Language()`/JS-vs-TS ambiguity this example
    demonstrates still holds against the real `Language()` doc comment.
    Any drift corrected and recorded in `progress.md`, same as T003.

- [ ] **T005** — Run the full pre-commit gate and confirm clean.
  - Depends on: T002, T003, T004
  - Acceptance: `gofumpt -l -w .`, `golangci-lint run ./...`,
    `go build ./...`, and `go test ./...` all exit 0 (`go test ./...`
    passes trivially — no test files added by this feature).

- [ ] **T006** — Update `specs/INDEX.md`'s row for 003a (status → `done`)
  and confirm the `memory/known-gaps.md` entry added during interrogation
  still accurately describes the shipped interface.
  - Depends on: T005
  - Acceptance: `INDEX.md`'s 003a row shows `done`; `known-gaps.md`'s 003a
    row matches the final `Language()` doc comment as implemented, with no
    edits needed (or edited if implementation diverged from the plan).

<!-- Keep each task small enough to implement and verify in a single sitting.
     If a task feels big, split it. -->

# Tasks: Data contract

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.

- [x] **T000** — Bootstrap the Go module and dev tooling. Not part of the
  feature itself, but every other task's acceptance criteria depend on it:
  `go mod init github.com/vedant-2701/stack-trace-bundler`, then install
  `golangci-lint` v2, `gofumpt`, `lefthook`, and run `lefthook install`.
  - Depends on: none
  - Acceptance: confirmed via a real test commit (deliberately unformatted
    file, staged, committed) — `lefthook` fired, `format` correctly caught
    the bad formatting and blocked the commit, `golangci-lint version`
    resolves to v2 (config mismatch error from the v1 install is gone).

- [ ] **T001** — Scaffold `internal/contract` package with `types.go`
  containing every struct from `plan.md`'s Data model (`Bundle`,
  `ExceptionNode`, `Frame`, `CodeContext`, `BlameEntry`, `GitMetadata`,
  `Dependencies`, `LockedDependency`, `Runtime`), with correct `json`
  struct tags —
  `omitempty` applied per the cross-cutting omission rule, deliberately
  withheld on `RawInputTruncated`.
  - Depends on: T000
  - Acceptance: package compiles; `golangci-lint run` and `gofumpt -l`
    both pass clean on the new files; a `Bundle{}` literal can be
    constructed and marshaled to JSON without a runtime panic.

- [ ] **T002** — Add `SchemaVersion` exported constant, `"1.0.0"`.
  - Depends on: T001
  - Acceptance: unit test asserts `contract.SchemaVersion == "1.0.0"`.

- [ ] **T003** — Implement `ComputeFingerprint(chain []ExceptionNode) string`
  in `fingerprint.go`: SHA-256 over, per exception node, the file+method
  identity (excluding line numbers) of every own-bucket frame plus the
  single originating frame regardless of bucket; truncated to the first 16
  hex characters.
  - Depends on: T001
  - Acceptance: table-driven unit tests cover spec.md's fingerprint
    acceptance criteria directly — same bug + dependency version bump
    produces the same fingerprint; genuinely different bugs produce
    different fingerprints; two chains sharing an identical outer wrapper
    but differing in an inner cause produce different fingerprints.

- [ ] **T004** — Add JSON marshal/unmarshal round-trip tests per struct in
  `types_test.go`, specifically asserting `omitempty` behavior: a Java
  frame with no `ColumnNumber` set must not have that key appear in the
  output JSON at all; a JS/TS frame with it set must have it present. A
  `LockedDependency` with no resolved version must omit `version` and
  include `note`; one with a resolved version must include `version` and
  omit `note`. `RawInputTruncated` must always appear in output, `true`
  or `false`, never absent.
  - Depends on: T001
  - Acceptance: tests pass; deliberately reverting `omitempty` on a field
    that should have it (or adding it to `RawInputTruncated`, which
    shouldn't) causes an obvious, specific test failure.

- [ ] **T005** — Build a fully-populated example `Bundle` covering both a
  Java-flavored and a TS/JS-flavored path, including at least two
  `own`-bucket frames from *different* files each with their own
  `codeContexts[].blame` entries (exercises per-file blame association),
  and at least one `dependencies.locked` entry with `version` omitted and
  `note` set (exercises the fresh-clone unresolved case) — and generate
  `testdata/example.json` from it (via a test or `go generate`), per
  Article IV — the fixture must be produced from the struct, never
  hand-written in parallel.
  - Depends on: T001, T003
  - Acceptance: `testdata/example.json` exists, is valid JSON, and a golden
    test confirms re-marshaling the example struct reproduces the fixture
    byte-for-byte, catching any future silent drift between the two.

- [ ] **T006** — Implement `TruncateRawInput(s string) (out string, truncated bool)`
  in `rawinput.go`, enforcing the 512 KB cap.
  - Depends on: T001
  - Acceptance: unit tests cover the boundary precisely — one byte under
    the cap (unchanged, `false`), exactly at the cap (unchanged, `false`),
    one byte over (truncated to exactly 512 KB, `true`).

<!-- Keep each task small enough to implement and verify in a single sitting.
     If a task feels big, split it. -->

# Tasks: Own-code context extraction

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.
Do not start a task without explicit authorization, per the project's
standing rules.

- [x] **T001** — Patch `internal/contract` (prerequisite for everything
  else in this feature): change `Bundle.GitMetadata` from `GitMetadata` to
  `*GitMetadata` with `json:"gitMetadata,omitempty"`; bump
  `contract.SchemaVersion` from `"1.0.0"` to `"2.0.0"`. Regenerate
  `testdata/example_java.json`/`example_ts.json`
  (`go test ./internal/contract/... -update`, Vedant-run). Update
  `001-data-contract`'s own `spec.md` (functional requirement 12, the
  `gitMetadata` field) and `plan.md` (its "Data model" section) to reflect
  the pointer type and the version bump, so those files aren't left stale
  relative to the actual struct.
  - Depends on: none
  - Acceptance: `go build ./...` passes; `go test ./internal/contract/...`
    passes against the regenerated fixtures; `001-data-contract/spec.md`
    and `plan.md` both show `*GitMetadata` and `"2.0.0"`, not the old
    shape.

- [x] **T002** — Implement the `gitRunner` interface (`runner.go`): the
  real `os/exec`-backed implementation (10s timeout via
  `context.WithTimeout`, spec.md requirement 8) plus a hand-written fake
  (`runner_fake_test.go`) that the rest of this feature's tests share —
  canned stdout per call, or a simulated `context.DeadlineExceeded`.
  - Depends on: T001
  - Acceptance: `go test ./internal/codecontext/...` passes using only the
    fake; no test in the package invokes the real implementation.

- [x] **T003** — Implement `BuildGitMetadata` (`gitmeta.go`): repo
  detection (`git rev-parse --is-inside-work-tree` /
  `--show-toplevel`), then `currentCommit`/`branch`/`uncommittedChanges`
  when a repo is found. Returns `nil` for no-repo and rev-parse-timeout
  cases (spec.md requirements 3, 8). Table-driven tests via the T002 fake.
  - Depends on: T002
  - Acceptance: `go test ./internal/codecontext/... -run TestBuildGitMetadata`
    passes, covering: repo found (all three fields populated), no repo
    found, rev-parse timeout, detached HEAD (`branch` == `"HEAD"`).

- [ ] **T004** — Implement windowed snippet extraction (`snippet.go`):
  reads only the needed line range (spec.md's non-functional
  requirement), clamps at file start/end, and classifies
  not-found/unreadable files per spec.md requirement 2. Table-driven
  tests, no git involved.
  - Depends on: T001
  - Acceptance: `go test ./internal/codecontext/... -run TestSnippet`
    passes, covering: normal window, target line 1, target line at EOF,
    file shorter than the window, file not found, permission-denied
    (simulated via a file with no read permission in `t.TempDir()`).

- [ ] **T005** — Implement per-file staleness check (`status.go`): runs
  `git status --porcelain <file>`, maps output to the internal
  `gitStatus` type (clean/modified/untracked/unknown-on-timeout per
  `plan.md`'s Data model), all non-clean variants collapsing to
  `contract.StatusStale` with distinguishing `note` text (spec.md
  requirements 5, 8). Table-driven tests via the T002 fake.
  - Depends on: T002
  - Acceptance: `go test ./internal/codecontext/... -run TestFileStatus`
    passes, covering: clean, modified, untracked, status-check timeout
    (→ stale + cautious note).

- [ ] **T006** — Implement `git blame` parsing (`blame.go`): runs
  `git blame --porcelain -L <start>,<end> <file>`, parses porcelain output
  into `contract.BlameEntry` grouped by contiguous same-commit ranges
  (spec.md requirement 6), and surfaces blame failures/timeouts per
  requirements 7-8 (caller decides status/note; this function just
  reports success or a typed failure). Table-driven tests via the T002
  fake.
  - Depends on: T002
  - Acceptance: `go test ./internal/codecontext/... -run TestBlame`
    passes, covering: single-commit window (one `BlameEntry`),
    multi-commit window (entries split at the actual boundary, not
    per-line), blame command failure, blame timeout, and a window where
    the same commit appears in two non-contiguous line groups —
    `--porcelain` omits that commit's `author`/`author-time`/`summary`
    block on its second occurrence in the same output stream (it was
    already shown once), so the parser must cache metadata by commit
    hash from the first occurrence and reuse it for the second group
    rather than treating that group as metadata-less.

- [ ] **T007** — Implement `BuildCodeContexts` orchestrator (`context.go`):
  iterates `chain`, filters to `own`-bucket frames, wires T004 (snippet) +
  T005 (status) + T006 (blame) together per spec.md requirements 1-2,
  5-7, 9-10, assembling each `contract.CodeContext`. Table-driven tests
  covering full end-to-end combinations.
  - Depends on: T003, T004, T005, T006
  - Acceptance: `go test ./internal/codecontext/... -run TestBuildCodeContexts`
    passes, covering every `spec.md` acceptance-criteria row that involves
    a `CodeContext` (clean/tracked, not-found, unreadable, stale-modified,
    stale-untracked, no-repo, blame-fails), plus a chain with zero
    own-bucket frames (empty result, no panic).

- [ ] **T008** — Full acceptance pass: walk every checkbox in `spec.md`'s
  Acceptance Criteria section against real behavior; add the CI-only,
  build-tag-gated integration test (`plan.md`'s Testing strategy) that
  exercises the real `os/exec` `gitRunner` against a throwaway
  `t.TempDir()` git repo Vedant sets up and runs locally. Append a
  `progress.md` entry summarizing the session and any deviations.
  - Depends on: T007
  - Acceptance: every acceptance-criteria checkbox in `spec.md` is
    checked or explicitly noted as deferred with a reason; the real-git
    integration test passes when run with the build tag
    (Vedant-confirmed, since it needs a real `git` binary);
    `progress.md` updated; `specs/INDEX.md` row for `004` updated to
    `done`.

<!-- Keep each task small enough to implement and verify in a single sitting.
     If a task feels big, split it. -->

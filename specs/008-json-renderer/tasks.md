# Tasks: JSON renderer

Derived from `plan.md`. Work through these in order, one at a time.
Mark status as you go: `[ ]` todo, `[~]` in progress, `[x]` done.
Each task lists the `spec.md` functional requirements it satisfies, so a
failing acceptance criterion later traces back to exactly one task.

- [x] **T001 — `JSON()` core implementation + basic unit tests** (spec.md
      req. 1-4)
      - Depends on: none
      Create `internal/render/json.go`: `func JSON(b contract.Bundle) string`
      using a `json.Encoder` over a `bytes.Buffer` with
      `SetEscapeHTML(false)`, trimming the encoder's trailing `\n` before
      returning. If `Encode` returns a non-nil error, panic with a clear
      message -- `contract.Bundle` can never produce one (spec.md req. 1),
      so a non-nil error here means that invariant broke, not a runtime
      condition to swallow (qa-log.md Q7). Create
      `internal/render/json_test.go` with hand-built, small
      `contract.Bundle` literals (not the shared fixtures yet -- those
      come in T004+) asserting: output is valid JSON (`json.Valid`),
      output is compact (no `\n` or double-space run inside the JSON
      body), and a `<`/`>` pair and a `&` in a `Message` field come
      through literal (not `\u003c`/`\u003e`/`\u0026`). Also add
      `TestJSON_NoTrailingNewline` as its own named test function
      (matching plan.md's Testing Strategy), asserting
      `!strings.HasSuffix(JSON(b), "\n")` against this task's hand-built
      literal -- T004 adds a second case to this same test once
      `tsBasicBundle` is available, completing plan.md's "run against at
      least `tsBasicBundle`."
      - Acceptance: all T001 unit tests pass; `go build ./...`,
      `go test ./internal/render/...`, `golangci-lint run ./internal/render/...`,
      `gofumpt -l ./internal/render/` all clean.

- [x] **T002 — `TestJSON_RoundTrip`** (spec.md acceptance criterion for
      round-trip fidelity)
      - Depends on: T001
      Add a round-trip test: `json.Unmarshal([]byte(JSON(b)), &got)` then
      `reflect.DeepEqual(b, got)`, using a hand-built `contract.Bundle`
      literal exercising nested slices, a non-nil `GitMetadata`, and a
      non-nil `Dependencies` with 2+ `Locked`/`Direct` entries (map
      coverage). Must include one note-only entry (no `Version`) and one
      version+note entry -- the same `omitempty` combinations
      `dependencyStatesBundle` exercises, which is why this test needs
      map coverage at all; a literal with only plain-resolved entries
      would not actually prove that -- doesn't need the shared
      `fixtures_test.go` builders yet, T004 reuses those for the golden
      layer.
      - Acceptance: `TestJSON_RoundTrip` passes; 4 verification commands
      clean.

- [x] **T003 — `ampersandBundle` fixture** (spec.md req. 3's `&` case)
      - Depends on: none
      Create `internal/render/json_fixtures_test.go` (new file -- does
      NOT edit 007's `fixtures_test.go`) with one builder,
      `ampersandBundle(t *testing.T) contract.Bundle`, following the
      same minimal-fixture pattern as `noGitMetadataBundle`
      (single exception node, one own-bucket frame, ordinary
      `GitMetadata`/`Dependencies`), with a `Message` or `Note`
      containing a literal `&` (e.g. `"fetch failed: config.json & env
      both missing"`) -- the one HTML-escape target character (alongside
      `<`/`>`, already covered by reusing `markdownSpecialCharsBundle`)
      that no existing 007 fixture contains.
      - Acceptance: file compiles (`go build ./...`,
      `go vet ./internal/render/...`); no new test yet, T006 consumes
      this fixture.

- [x] **T004 — Golden harness + first fixture: `ts_basic`**
      - Depends on: T001
      Add `assertGoldenJSON(t, got, path)` to `json_test.go` (mirrors
      `assertGoldenMarkdown` from `markdown_test.go`, reusing that file's
      existing `var updateGolden` -- do NOT redeclare it, that's a
      compile error) and the first table-driven golden case, reusing
      `tsBasicBundle` from `fixtures_test.go`. Hand-review the generated
      `testdata/golden_json/ts_basic.golden.json` once by eye against
      spec.md's functional requirements (reqs. 1-6) before committing it.
      Also add a second case to T001's `TestJSON_NoTrailingNewline`
      asserting the same property against `tsBasicBundle`, now that it's
      in scope -- completes plan.md's "run against at least
      `tsBasicBundle`."
      - Acceptance: `TestJSON/ts_basic` (or equivalent) and the added
      `tsBasicBundle` case in `TestJSON_NoTrailingNewline` both pass;
      golden file hand-reviewed and committed; 4 verification commands
      clean.

- [x] **T005 — Golden fixtures: `no_git_metadata`, `no_dependencies`**
      - Depends on: T004
      Add two more table entries reusing `noGitMetadataBundle` and
      `noDependenciesBundle` from `fixtures_test.go`. Confirms
      requirement 5's omission behavior (nil-pointer fields entirely
      absent, not `null`) flows through unchanged into this feature's
      compact/unescaped output.
      - Acceptance: both golden tests pass; 4 verification commands
      clean.

- [x] **T006 — Golden fixtures: `markdown_special_chars`, `ampersand`**
      - Depends on: T004, T003
      Add table entries reusing `markdownSpecialCharsBundle` (`<`/`>`,
      from `fixtures_test.go`) and the new `ampersandBundle` (`&`, from
      T003). Confirms spec.md req. 3 (HTML-escaping disabled) for all
      three target characters, end-to-end. Also remove the `.golangci.yml`
      exclusion (`unused` linter, `internal/render/json_fixtures_test.go`)
      added during T003 to unblock its commit -- `ampersandBundle` is
      used starting with this task, so the exclusion is no longer needed
      and would otherwise silently stop `unused` from checking that file.
      - Acceptance: both golden tests pass, with hand-verification that
      neither golden file contains `\u003c`, `\u003e`, or `\u0026`; the
      `.golangci.yml` exclusion for `json_fixtures_test.go` is removed;
      4 verification commands clean.

- [x] **T007 — Golden fixtures: `dependency_states`, `runtime_version_states`**
      - Depends on: T004
      Add table entries reusing `dependencyStatesBundle` and
      `runtimeVersionStatesBundle` from `fixtures_test.go`. Confirms
      requirement 6 (deterministic, alphabetically-sorted map keys) and
      the `VersionSourceUnknown` omission case, end-to-end.
      - Acceptance: both golden tests pass, including a repeated-render
      check on `dependency_states` for map-key-order determinism; 4
      verification commands clean.

- [ ] **T008 — Acceptance criteria review pass**
      - Depends on: T001-T007
      Re-read `spec.md`'s Acceptance criteria list top to bottom; confirm
      each has an exact corresponding passing test, recorded as an
      inline comment next to each checkbox in `spec.md` (same pattern as
      007's T017). Update `spec.md`'s header from "Spec'd" to "Done" and
      `specs/INDEX.md`'s 008 row from `idea` to `done`.
      - Acceptance: every acceptance criterion in `spec.md` checked off
      with a named test; `specs/INDEX.md`'s 008 row updated.

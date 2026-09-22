# Progress Log: JSON renderer

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-09-22
**Task(s):** T005 — Golden fixtures: `no_git_metadata`, `no_dependencies`
**What happened:** Added two entries to `TestJSON`'s table in `internal/render/json_test.go`, reusing `noGitMetadataBundle` and `noDependenciesBundle` from `fixtures_test.go`. Generated both golden files via `-update` and hand-reviewed them: `no_git_metadata.golden.json` has no `gitMetadata` key anywhere (not `null`); `no_dependencies.golden.json` has no `dependencies` key anywhere -- confirms req. 5's omission behavior end-to-end, not just a byte-match against whatever got generated. `go build`, `go test ./internal/render/...`, `golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/` all clean per user.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-22
**Task(s):** T004 — Golden harness + first fixture: `ts_basic`
**What happened:** Added `assertGoldenJSON` and `TestJSON` (table-driven, `ts_basic` entry reusing `tsBasicBundle`) to `internal/render/json_test.go`, mirroring `assertGoldenMarkdown`/`TestMarkdown` and reusing `markdown_test.go`'s existing `updateGolden` flag. Converted `TestJSON_NoTrailingNewline` to table-driven with a second case against `tsBasicBundle`. Generated `internal/render/testdata/golden_json/ts_basic.golden.json` via `go test ./internal/render/... -run TestJSON -update`, hand-reviewed it against spec.md reqs 1-6 before committing (req. 3 not exercised by this fixture -- no `<`/`>`/`&` in `tsBasicBundle`; that's T006). `go build`, `go test ./internal/render/...`, `golangci-lint run ./internal/render/...`, `gofumpt -l ./internal/render/` all clean per user, after one fix.
**Deviations from plan (if any):** First lint run failed: `revive` flagged an unused `t` parameter in the `TestJSON_NoTrailingNewline` hand-built-literal closure (`func(t *testing.T) contract.Bundle { return jsonTestBundle() }`). Fixed by renaming to `_`. Not anticipated by tasks.md (unlike T003's expected `unused` failure).
**New open questions:** None.

---

**Date:** 2026-09-22
**Task(s):** T003 — `ampersandBundle` fixture
**What happened:** Added `internal/render/json_fixtures_test.go` (new file, separate from 007's `fixtures_test.go`) with `ampersandBundle(t *testing.T) contract.Bundle`: single `ExceptionNode`, one own-bucket `Frame`, ordinary non-nil `GitMetadata`/`Dependencies`, `Message` containing a literal `&`. Not yet consumed by any test -- T006 wires it into the golden table. `go build ./...` and `go vet ./internal/render/...` both reported clean per user. The Lefthook pre-commit hook still ran golangci-lint regardless of this task's narrower acceptance line, and `unused` failed on `ampersandBundle` as anticipated; user added a scoped exclusion to `.golangci.yml` (`unused` linter only, `internal/render/json_fixtures_test.go` only) to let the commit through.
**Deviations from plan (if any):** `.golangci.yml` gained a lint exclusion not called for in plan.md/tasks.md.
**New open questions:** Remove the `.golangci.yml` exclusion for `internal/render/json_fixtures_test.go` once T006 wires `ampersandBundle` into a test -- otherwise `unused` silently stops checking that file going forward.

---

**Date:** 2026-09-22
**Task(s):** T002 — `TestJSON_RoundTrip`
**What happened:** Added `roundTripTestBundle()` and `TestJSON_RoundTrip` to `internal/render/json_test.go` (no new files). The literal exercises nested slices (a `Chain` of 2 `ExceptionNode`s, each with its own `Frames`), a non-nil `GitMetadata`, and a non-nil `Dependencies` with 2 `Direct` entries and 2 `Locked` entries (one note-only, no `Version`; one with both `Version` and `Note`). Test does `json.Unmarshal([]byte(JSON(want)), &got)` then `reflect.DeepEqual(want, got)`. `go build`, `go test ./internal/render/...`, `golangci-lint run ./internal/render/...`, and `gofumpt -l ./internal/render/` all reported clean per user.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

**Date:** 2026-09-22
**Task(s):** T001 — `JSON()` core implementation + basic unit tests
**What happened:** Added `internal/render/json.go` (`JSON(b contract.Bundle) string`, `json.Encoder` over `bytes.Buffer`, `SetEscapeHTML(false)`, trims the encoder's trailing `\n`, panics on a non-nil `Encode` error per spec.md req. 1's invariant) and `internal/render/json_test.go` (hand-built literal, not shared fixtures yet) with `TestJSON_Valid`, `TestJSON_Compact`, `TestJSON_HTMLCharsLiteral`, and `TestJSON_NoTrailingNewline`. `go build`, `go test ./internal/render/...`, `golangci-lint run ./internal/render/...`, and `gofumpt -l ./internal/render/` all reported clean per user.
**Deviations from plan (if any):** None.
**New open questions:** None.

---

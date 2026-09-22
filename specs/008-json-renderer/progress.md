# Progress Log: JSON renderer

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-09-22
**Task(s):** T003 — `ampersandBundle` fixture
**What happened:** Added `internal/render/json_fixtures_test.go` (new file, separate from 007's `fixtures_test.go`) with `ampersandBundle(t *testing.T) contract.Bundle`: single `ExceptionNode`, one own-bucket `Frame`, ordinary non-nil `GitMetadata`/`Dependencies`, `Message` containing a literal `&`. Not yet consumed by any test -- T006 wires it into the golden table. `go build ./...` and `go vet ./internal/render/...` both reported clean per user; golangci-lint/gofumpt intentionally skipped this task per tasks.md's own narrower acceptance line (the fixture is genuinely unused until T006, and `unused` is enabled in `.golangci.yml`).
**Deviations from plan (if any):** None.
**New open questions:** None.

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

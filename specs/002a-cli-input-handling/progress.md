# Progress Log: CLI Input Handling

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:**
**Task(s):**
**What happened:**
**Deviations from plan (if any):**
**New open questions:**

---

**Date:** 2026-08-08
**Task(s):** T001
**What happened:** Scaffolded `internal/cli` package: created `input.go` with the package doc comment and the `Input` struct (`RawText`, `RawInputTruncated`, `LangHint`, `Format`), per tasks.md's acceptance criteria.
**Deviations from plan (if any):** plan.md's Data model section shows `Input` with no JSON struct tags. Added `json` tags (camelCase, matching `internal/contract`'s convention) to every field, with **no** `omitempty` anywhere. Rationale: spec requirement 13's Debug-level full struct dump will use `json.MarshalIndent(Input)` for readability instead of `%+v`, which needs tags to be useful. No field gets `omitempty` because applying `internal/contract`'s own cross-cutting rule correctly (omitempty is for "not applicable", never for "empty-but-meaningful") means all four fields stay present at their zero value -- same reasoning as `contract.RawInputTruncated` / `contract.Runtime.VersionSource`. plan.md's Data model code sample has been updated to match (tags + rationale added directly to plan.md, per Vedant's confirmation).
**New open questions:** none. `go build ./...`, `go vet ./internal/cli/...`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean (Vedant-confirmed). T001 marked done.

---

**Date:** 2026-08-09
**Task(s):** T002
**What happened:** Implemented `readTrace` in `read.go` (file-arg vs. stdin source selection, both-present precedence with `stdinIgnored`, TTY-no-input fast-fail, `io.LimitReader` bounded read at a provisional 1MB cap, empty/whitespace-only rejection, `contract.TruncateRawInput` for the final 512KB cap) plus table-driven tests in `read_test.go` covering all cases tasks.md's acceptance criteria lists.
**Deviations from plan (if any):** One extra wrapped error not named in plan.md's three examples: an I/O error reading from stdin itself (`"reading trace from stdin: %w"`), added for symmetry with the file-read error path; not currently covered by a test case since it's hard to trigger with a `strings.Reader`. Also: both `read.go` and `read_test.go` initially had a `// internal/cli/read.go`-style file-path comment directly above `package cli`, which failed `golangci-lint`'s `revive` `package-comments` check (it treats any comment immediately preceding `package X` as an attempted package doc, and fails it unless it's `"Package X ..."`). Fixed by removing those header comments (only `input.go` should carry the real package doc). Documented in `CONVENTIONS.md`'s Formatting & linting section and in Claude's own memory so this isn't repeated on later files/tasks.
**New open questions:** none. `go build ./...`, `go test ./internal/cli/... -run TestReadTrace -v`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean after the fix (Vedant-confirmed). T002 marked done.

---

**Date:** 2026-08-09
**Task(s):** T003
**What happened:** Implemented `validateFormat` and `validateLang` in `parse.go`, plus table-driven tests in `parse_test.go` covering valid and invalid values for each.
**Deviations from plan (if any):** None from plan.md's API contract. One explicit design call, consistent with (not a deviation from) `input.go`'s T001 doc comment: `validateFormat("")` returns `("markdown", nil)` rather than an error — empty is treated as omitted/defaulted, matching what `Input.Format`'s doc comment already committed to. `validateLang("")` returns `("", nil)` unchanged — empty stays the meaningful defer-to-auto-detection signal, never defaulted.
**New open questions:** none. `go build ./...`, `go test ./internal/cli/... -run 'TestValidateFormat|TestValidateLang' -v`, `gofumpt -l`, and `golangci-lint run ./internal/cli/...` all passed clean (Vedant-confirmed). T003 marked done.

---

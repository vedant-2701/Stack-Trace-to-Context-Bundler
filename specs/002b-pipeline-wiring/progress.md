# Progress Log: Pipeline wiring

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

**Date:** 2026-09-24
**Task(s):** Kickoff / spec interrogation (no code)
**What happened:** Checked dependencies in `specs/INDEX.md`: 002a, 003a, 003b, 004, 006a, 007, 008, 009 all `done`. Found the INDEX row's "Depends on" omits 006b (done; `ResolveDependencies` is needed to populate `Bundle.Dependencies`) and that its description ("003a/003b–009") implies 005a/005b, which are `idea` and don't exist. Created this folder from `specs/_templates/`; INDEX status moved `idea` -> `specifying`. Items already owed to 002b, to fold into spec.md: remove 002a's stub output from `cmd/*/main.go` (002a spec note); helpful "no parser matched" message for the bare-stack shape (`known-gaps.md`, 006a row); `--lang typescript` no longer maps to a single parser (003b spec, Out of scope); exit-code allocation for clipboard errors (009 FR8) and `ErrNoMatch`/`ErrAmbiguous`/`ErrUnparseable` (003b, CONVENTIONS.md); `-h`/`--help` parked since 002a T004.
**Decisions so far:** interrogation questions, options, answers and unresolved items are logged in `specs/002b-pipeline-wiring/qa-log.md`, not here.
**Deviations from plan (if any):** n/a (no plan yet)
**New open questions:** tracked in `qa-log.md` as `[NEEDS CLARIFICATION]` entries until `spec.md` is written.

---

**Date:** 2026-09-24
**Task(s):** Interrogation complete (Q1-Q18, `qa-log.md`); wrote `spec.md`,
`plan.md`, `tasks.md`; updated `specs/INDEX.md`.
**What happened:** All 18 questions resolved -- no `[NEEDS CLARIFICATION]`
remains in `spec.md`. Key shape: Java fails fast (exit 2, "Java is not
supported yet") until 005a/005b exist; stdout+clipboard by default,
`-o`/`--output` replaces stdout, `--no-clipboard` skips the copy;
clipboard failure is Warn+exit0 after a successful delivery, output
write failure is exit 1 before the clipboard is touched; `--lang`
narrows candidates but never bypasses `Detect`; sequential pipeline
(code-context then dependency resolution) for MVP; real `-h`/`--help`
with usage examples and an exit-code table, built now rather than
deferred. `plan.md` centralizes all of this into a new
`internal/cli.Run`/`BinaryConfig`, extends `ParseAll`/`ParseFixedLang`
with a `candidates` parameter so the FR10 Java fail-fast happens before
any input is read, and adds `candidates.go`/`errors.go`/`output.go`/
`help.go`/`run.go`. `tasks.md` breaks this into 22 ordered tasks (T001
verifies the two carried-forward risk items -- `Bundle.OS` validation,
existing flag help strings -- before any code changes). `specs/INDEX.md`:
002b's "Depends on" now includes 006b (real gap, found at kickoff);
status moved `specifying` -> `planned`.
**Deviations from plan (if any):** n/a
**New open questions:** none open in `spec.md`. Two items are deferred
to planning-time verification rather than left as spec ambiguity (see
plan.md's Risks section): whether anything validates `Bundle.OS` against
the three `contract.OS` constants, and which existing flags already have
help strings. Both are T001.
**Next step:** implementation, one task at a time from `tasks.md`,
starting at T001 -- not started in this pass, per instructions.

---

**Date:** 2026-09-26
**Task(s):** T001 -- verify the two carried-forward risk items (no code)
**What happened:** (1) Read every non-test `.go` file in `internal/contract`
(`types.go`, `fingerprint.go`, `rawinput.go`) and `internal/render`
(`escape.go`, `json.go`, `markdown.go`): `Bundle.OS` is only ever assigned
and interpolated with `%s` in `renderMetadata` -- no comparison against
`OSLinux`/`OSDarwin`/`OSWindows` exists anywhere. Q11's "pass `runtime.GOOS`
through verbatim" assumption holds; no validation site found, so no stop
triggered. (2) Read `internal/cli/parse.go`: `--lang`, `--format`, and
`-v`/`--verbose` all already have non-empty help strings (`validateLang`/
`validateFormat`'s flags and the `CountVarP` call). Only `-o`/`--output`
and `--no-clipboard` will need new help strings, starting at T002/T003.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T002 -- add `Output string` and `NoClipboard bool` to
`internal/cli.Input`
**What happened:** Added both fields to the end of the `Input` struct in
`input.go`, each with a doc comment matching the file's existing style
(references the relevant FRs and which later task wires/consumes the
field: T004's `validateOutput` and `output.go`'s `deliver` for `Output`;
FR16's non-fatal clipboard-failure handling for `NoClipboard`). No other
file changed. Vedant ran `go build ./internal/cli/...`, `gofumpt -l
internal/cli/input.go`, `golangci-lint run ./internal/cli/...`, and `go
test ./internal/cli/...`; confirmed done, no errors.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T003 -- extract flag registration into a shared
`registerFlags` helper
**What happened:** Added `registerFlags(fs *pflag.FlagSet, fixedLang
string) (langFlag, formatFlag, outputFlag *string, noClipboard *bool,
verbosity *int)` to `parse.go`, matching tasks.md's specified shape.
`fixedLang == ""` registers `--lang` (cmd/all); non-empty skips it
(cmd/java/cmd/typescript keep rejecting `--lang` as unknown, unchanged).
Also registered `--output`/`-o` and `--no-clipboard` here so they show up
in T008's `--help`, but neither flag's value is read into `Input` yet --
still parsed and discarded until T006/T007, per plan. `ParseAll` and
`ParseFixedLang` both now call `registerFlags` instead of registering
flags inline; doc comments on both updated to point at it. No change to
validation order or `Input` construction. Confirmed `parse_test.go` has
no case asserting `--output`/`--no-clipboard` are unknown flags before
making this change, so registering them now doesn't break anything
existing. Vedant ran `go build ./internal/cli/...`, `gofumpt -l
internal/cli/parse.go`, `golangci-lint run ./internal/cli/...`, and `go
test ./internal/cli/...`; confirmed done, no errors.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T004 -- add `validateOutput` to `read.go`
**What happened:** Added `validateOutput(outputPath, fileArg string)
error`: empty `outputPath` returns nil immediately (pass-through);
otherwise checks the parent directory exists and is a directory, then
checks `os.SameFile(outputPath, fileArg)` -- skipped when `fileArg == ""`
(stdin case) or when either path doesn't exist yet (a not-yet-existing
`--output` path can't be the same file as anything). An existing file at
`outputPath` that isn't `fileArg` is left alone here -- overwriting it is
T009's `deliver`, not validation's concern. Added `TestValidateOutput` to
`read_test.go`: pass-through (with/without a file arg), missing parent
dir, parent-is-a-file, same-file-as-input, same-file-check-skipped-on-
stdin, pre-existing unrelated output file, and the plain valid case. Not
wired into `ParseAll`/`ParseFixedLang` yet -- that's T006/T007. Vedant
ran `go build ./internal/cli/...`, `gofumpt -l internal/cli/read.go
internal/cli/read_test.go`, `golangci-lint run ./internal/cli/...`, and
`go test ./internal/cli/...`; confirmed done, no errors.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T005 -- add `selectCandidates`/`languageUnsupportedError`
**What happened:** Added `internal/cli/candidates.go`: `hintLanguages`
(`"typescript"` -> `[javascript, typescript]`, `"java"` -> `[java]`),
`hintDisplayName` (`"java"` -> `"Java"`), and `selectCandidates(hint
string, registered []parser.LanguageParser) ([]parser.LanguageParser,
error)`. Empty hint returns `registered` unchanged; a hint whose language
set matches zero registered parsers returns a `*languageUnsupportedError`
(deliberately, not an empty slice -- `parser.DetectLanguage` panics on an
empty candidates slice, so the caller must fail fast here, before T006/
T007 ever reach it); an unrecognized hint panics as a caller-bug
invariant, same reasoning as `ParseFixedLang`'s existing invalid-lang
panic. Added `internal/cli/errors.go`: `languageUnsupportedError`,
`Error()` returns exactly `"<Name> is not supported yet"`. Added
`candidates_test.go`: a two-field `fakeParser` implementing
`parser.LanguageParser`, `TestSelectCandidates` (empty hint, typescript
narrowing to both js+ts, java with none registered -> exact error
message via `errors.As`, java with one registered -> succeeds), and
`TestSelectCandidates_InvalidHintPanics`. Vedant ran `go build
./internal/cli/...`, `gofumpt -l internal/cli/candidates.go
internal/cli/errors.go internal/cli/candidates_test.go`, `golangci-lint
run ./internal/cli/...`, `go test ./internal/cli/... -run Candidates -v`,
and `go test ./internal/cli/...`; confirmed done, no errors.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

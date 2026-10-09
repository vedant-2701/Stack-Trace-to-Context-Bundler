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

**Date:** 2026-09-26
**Task(s):** T006 -- wire `selectCandidates`/`validateOutput` into
`ParseAll`
**What happened:** `ParseAll`'s signature is now `ParseAll(args
[]string, stdin io.Reader, stdinIsPiped bool, candidates
[]parser.LanguageParser) (Input, int, error)`. New order: `Parse ->
validateLang -> validateFormat -> selectCandidates (fail-fast, narrowed
slice discarded -- Run will re-derive it in T011) -> validateOutput ->
readTrace`. `Input.Output`/`Input.NoClipboard` now populated from the
flag pointers. Updated `parse_test.go`: added `candidates` and
`outputNonexistentParent` fields to `TestParseAll`'s table; gave the four
existing `--lang=java` cases (`valid file, explicit lang and format`,
both `-v`/`--lang` order-independence cases, the T007a case) a fake Java
parser so they still pass under the new candidate-checking step; added
three new cases (`--format=bogus --lang=java` reports the format error;
java hint fails fast before `--output` is validated with no java parser
registered; typescript hint reaches `--output` validation once candidate
fail-fast doesn't fire); added a `recordingReader` type and
`TestParseAll_UnsupportedHintFailsFastBeforeReadTrace` asserting `Read`
is never called on the fail-fast path. Flagged to Vedant: `cmd/all/main.go`
still calls the old 3-arg `ParseAll` and no longer compiles -- `go build
./...` (repo-wide) is broken until T016 rewires it; `go build
./internal/cli/...` and `cmd/java`/`cmd/typescript` (still on
`ParseFixedLang`, unchanged) are unaffected. This matches tasks.md's own
sequencing (T016 depends on T015), not a mistake. Vedant ran `go build
./internal/cli/...`, `gofumpt -l internal/cli/parse.go
internal/cli/parse_test.go`, `golangci-lint run ./internal/cli/...`, `go
test ./internal/cli/... -v -run TestParseAll`, and `go test
./internal/cli/...`; confirmed done, no errors.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T007 -- wire `selectCandidates`/`validateOutput` into
`ParseFixedLang`
**What happened:** `ParseFixedLang`'s signature is now `ParseFixedLang(
args []string, stdin io.Reader, stdinIsPiped bool, lang string,
candidates []parser.LanguageParser) (Input, int, error)`. Order: `Parse
-> validateFormat -> selectCandidates(fail-fast) -> validateOutput ->
readTrace` (no `validateLang` step -- `lang` is a fixed, already
panic-checked caller input, not a flag value). `Input.Output`/
`Input.NoClipboard` populated. Updated `parse_test.go`: gave every
existing success-path case with `lang: "java"`/`"typescript"` a matching
fake candidate (several, e.g. `"no input surfaces readTrace's error"`
with `lang: "typescript"`, would otherwise now fail with
`languageUnsupportedError` instead of their intended assertion); added
the java-fails-before-output and typescript-reaches-output cases;
added `TestParseFixedLang_UnsupportedHintFailsFastBeforeReadTrace`
(reusing T006's `recordingReader`); updated both `ParseFixedLang` call
sites (including `TestParseFixedLang_InvalidLangPanics`) to the 5-arg
signature. Corrected the T006 entry's build-breakage note to Vedant:
`cmd/java`/`cmd/typescript` now also fail to compile (old 4-arg
`ParseFixedLang` calls) until T017/T018 -- `go build ./...` stays broken
repo-wide through T018, not just until T016 as previously said.

After Vedant's first gate run, `golangci-lint`'s `gocyclo` flagged both
`TestParseAll` and `TestParseFixedLang` at complexity 16 (> 15) -- the
new `outputNonexistentParent` branch pushed both over. Fixed by
extracting the shared `buildTestTraceArgs` (fixture/args setup) and
`assertParseResult` (all the wantErrSubstr/LangHint/Format/verbosity/
StdinIgnored assertions) helpers, used by both tests' `t.Run` bodies,
which are now three straight-line calls each with no branching of their
own. Vedant ran `go build ./internal/cli/...`, `gofumpt -l
internal/cli/parse.go internal/cli/parse_test.go`, `golangci-lint run
./internal/cli/...`, `go test ./internal/cli/... -v -run
'TestParseFixedLang|TestParseAll'`, and `go test ./internal/cli/...`;
confirmed done, no errors (gocyclo included, after the fix). No commit
message given for this task -- `cmd/java`/`cmd/typescript` still don't
compile, so `lefthook`'s pre-commit gate would fail; per Vedant's
instruction, commit messages are only given once the repo-wide gate can
actually pass, not automatically after every task.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T008 -- add `internal/cli/help.go`/`help_test.go`
**What happened:** Added `Example{Command, Description string}`,
`exitCodeDocs` (5 entries matching `plan.md`'s table exactly), and
`usageText(programName string, fs *pflag.FlagSet, examples []Example)
string` (synopsis line, `fs.FlagUsages()`, an optional Examples section
-- omitted entirely when `examples` is empty, rather than printed as a
bare heading -- then the exit-code table). `fs` is passed in already
registered (via `registerFlags`) rather than built inside `usageText`,
so `Run` (T010) reuses the exact FlagSet it parses with -- the rendered
flag listing can't drift from what's actually accepted. Added
`help_test.go`: `--lang` present/absent for `fixedLang` `""` vs
`"java"`/`"typescript"`; every `exitCodeDocs` code number appears in the
rendered text; a `placeholderExamples` set (real examples land in
T016-018) round-trips through `ParseAll` with no `"parsing flags"`/
`"invalid --"` error, tolerating the expected `readTrace` "file not
found" error since the example paths don't exist on disk. Flagged to
Vedant, unresolved: `plan.md`'s `hintDisplayName` design includes
`"typescript": "TypeScript"`, but T005's accepted implementation only
has `"java": "Java"` -- a real plan/code gap, though unreachable in
practice (no planned `BinaryConfig` ever registers zero JS/TS parsers).
Not fixed yet, awaiting Vedant's call on whether to patch `candidates.go`
now or leave it. Vedant ran `go build ./internal/cli/...`, `gofumpt -l
internal/cli/help.go internal/cli/help_test.go`, `golangci-lint run
./internal/cli/...`, `go test ./internal/cli/... -v -run TestUsageText`,
and `go test ./internal/cli/...`; confirmed done, no errors. No commit
message -- `cmd/java`/`cmd/typescript` still don't compile (unchanged
from T007's state; this task touched no `cmd/*` file).
**Deviations from plan (if any):** n/a
**New open questions:** whether to fix the `hintDisplayName` "typescript"
entry now or leave it (see above).

---

**Date:** 2026-09-26
**Task(s):** hintDisplayName fix (from T008's open question) + T009 --
add `internal/cli/output.go`/`output_test.go`
**What happened:** Per Vedant's "proceed," added `"typescript":
"TypeScript"` to `candidates.go`'s `hintDisplayName`, matching
`plan.md`'s design (still unreachable in practice today -- no planned
`BinaryConfig` registers zero JS/TS parsers). Added `deliver(bundle,
outputPath string, stdout io.Writer) error` to `output.go`: appends a
trailing `\n` only if `bundle` doesn't already end with one (FR15);
empty `outputPath` writes to `stdout`; otherwise opens the file
(`O_WRONLY|O_CREATE|O_TRUNC`) only at call time -- never earlier, never
by `validateOutput` -- silently overwriting per FR17/Q4, and checks the
`Close()` error too, not just the write, since a flush failure on Close
is still a real write failure. Added `output_test.go`: stdout newline
cases (none, already-present, multiple, empty bundle), new-file write,
silent-overwrite-fully-truncated (not prefixed), and an open-failure
case (nonexistent parent directory) asserting both a non-nil error and
that nothing was created. Vedant ran `go build ./internal/cli/...`,
`gofumpt -l internal/cli/candidates.go internal/cli/output.go
internal/cli/output_test.go`, `golangci-lint run ./internal/cli/...`,
`go test ./internal/cli/... -v -run 'TestDeliver|TestSelectCandidates'`,
and `go test ./internal/cli/...`; confirmed done, no errors. No commit
message -- `cmd/java`/`cmd/typescript` still don't compile (unchanged
from T007; this task touched no `cmd/*` file).
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T010 -- add `internal/cli/run.go`/`run_test.go` (step 3
only)
**What happened:** Added `DependencyResolver`, `BinaryConfig`
(`ProgramName`, `FixedLang`, `Candidates`, `Resolvers`, `Examples`), and
`Run(args, stdin, stdinIsPiped, cfg, stdout, stderr) int`. Dispatches to
`ParseAll` (`cfg.FixedLang == ""`) or `ParseFixedLang` otherwise.
`errors.Is(err, pflag.ErrHelp)` -> prints `usageText` to `stderr`,
returns 0, before `slog` is touched at all. Any other error -> `slog`
configured against the injected `stderr` (not `os.Stderr`, per
`plan.md`'s API/contracts section, so `run_test.go` can assert on log
content without swapping real process-wide I/O) -> `slog.Error`, return
2. Success path: same `slog` configuration, then the Debug `stdin
ignored` line if `input.StdinIgnored`. `stdout` is unused past a `_ =
stdout` placeholder with a `// TODO T011+` marker -- steps 4-12 land in
T011-T014. Added `run_test.go`: help path (`-h`, `--help`, exit 0,
stderr contains the program name, stdout stays empty); flag-error path
(exit 2); the two FR19/Q14 ordering cases (`--lang=cobol -h` still helps
-- pflag reaches `-h` before our own post-Parse `validateLang` ever runs,
since `--lang=cobol` is a well-formed string flag at the pflag layer;
`--bogus -h` exits 2 -- pflag's left-to-right Parse stops at the unknown
flag first, `-h` is never reached); the stdin-ignored Debug case at
`-vv`, asserted by reading the injected `stderr` `strings.Builder`
directly. Vedant ran `go build ./internal/cli/...`, `gofumpt -l
internal/cli/run.go internal/cli/run_test.go`, `golangci-lint run
./internal/cli/...`, `go test ./internal/cli/... -v -run TestRun`, and
`go test ./internal/cli/...`; confirmed done, no errors. No commit
message -- `cmd/java`/`cmd/typescript` still don't compile (this task
touched no `cmd/*` file either).
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T011 -- extend `Run` through step 4 (candidate re-selection,
`DetectLanguage`, exit-4 mapping, language-resolved/hint-mismatch
logging)
**What happened:** `Run` now re-runs `selectCandidates(input.LangHint,
cfg.Candidates)` (defensively -- should be unreachable to error, since
`ParseAll`/`ParseFixedLang` already ran the identical check), then calls
`parser.DetectLanguage`. `ErrNoMatch` -> exit 4, `"could not detect the
language of the input: <err>"`, plus the exact FR12 tip appended as a
second sentence when `strings.TrimSpace(input.RawText)` has no newline.
`ErrAmbiguous` -> exit 4, `"the input matched more than one language:
<err>"`, never with the tip. Any other `DetectLanguage` error
(contractually unreachable per its own doc comment) -> exit 1,
defensive. On a match: `slog.Info("language resolved", "language",
...)` always; the separate FR9 `slog.Info("detected language differs
from --lang hint", "hint", ..., "detected", ...)` only when a hint was
given and differs from what matched. Added `configurableFakeParser` to
`run_test.go` (settable `Detect` result, unlike `candidates_test.go`'s
always-false `fakeParser`) and `TestRun_DetectLanguage`: no-match/
multi-line (no tip), no-match/single-line (tip present), both-match
(ambiguous, never a tip regardless of line count), hint-mismatch (both
Info lines with their attributes), hint-match (only one), no-hint
(only one). Caught and fixed two bugs before handing off the gate
commands: the no-hint case needed `-v` too -- Info is filtered at the
default Warn level, so it would have failed; and
`TestRun_StdinIgnoredLogsAtDebug` (from T010) would now panic --
`parser.DetectLanguage` panics on empty `candidates`, and that test had
never needed a real one before T011 added step 4 -- fixed by giving it
one always-matching `configurableFakeParser`. Vedant ran `go build
./internal/cli/...`, `gofumpt -l internal/cli/run.go
internal/cli/run_test.go`, `golangci-lint run ./internal/cli/...`, `go
test ./internal/cli/... -v -run TestRun`, and `go test
./internal/cli/...`; confirmed done, no errors. No commit message --
`cmd/java`/`cmd/typescript` still don't compile (this task touched no
`cmd/*` file either).
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T012 -- extend `Run` through step 5 (`Parse`, exit-3/1
mapping)
**What happened:** `Run` now calls `matched.Parse(context.Background(),
input.RawText)`. `errors.Is(err, parser.ErrUnparseable)` -> exit 3,
`"could not parse the input: <err>"` (FR12). Any other non-nil error ->
exit 1. On success, falls through to a `// TODO T013+` marker;
`chain`/`runtimeInfo`/`stdout` all discarded via `_, _, _ = ...` until
T013 uses them. Extended `configurableFakeParser` (`run_test.go`) with a
`parseErr error` field, zero-value `nil` meaning success -- important
since every T011 success-path test now also flows through `Parse`, not
just `DetectLanguage`, and needed to keep passing unmodified. Added
`TestRun_Parse`: `ErrUnparseable`-wrapped error -> exit 3, plain error ->
exit 1, success -> exit 0. While rewriting the test file, hit the exact
empty-`oldText` `edit_file` gotcha from `tooling-learnings.md` a second
time -- an insert landed before `package cli` instead of at the end;
caught and fixed before handing off the gate commands, along with
correcting `configurableFakeParser`'s doc comment (previously claimed
Parse would never be called, which T012 makes false). Vedant ran `go
build ./internal/cli/...`, `gofumpt -l internal/cli/run.go
internal/cli/run_test.go`, `golangci-lint run ./internal/cli/...`, `go
test ./internal/cli/... -v -run TestRun`, and `go test
./internal/cli/...`; confirmed done, no errors. No commit message --
`cmd/java`/`cmd/typescript` still don't compile (this task touched no
`cmd/*` file either).
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T013 -- extend `Run` through step 9 (`getwd`,
`buildGitMetadata`, `buildCodeContexts`, dependency resolver,
`assembleBundle`)
**What happened:** Added package-level vars `getwd`, `buildGitMetadata`,
`buildCodeContexts` wrapping `os.Getwd`/`codecontext.BuildGitMetadata`/
`codecontext.BuildCodeContexts` (tasks.md's `getwd` naming, extended to
the other two since CONVENTIONS.md requires anything shelling out be
tested behind a fake). `getwd()` failure -> exit 1 before any git/
dependency call. On success: `buildGitMetadata` -> `buildCodeContexts` ->
`cfg.Resolvers[matched.Language()]` if registered -> `assembleBundle`, a
new pure function extracted specifically so FR4's field-mapping rules
(`SchemaVersion`, `OS` via `runtime.GOOS`, `Fingerprint` via
`contract.ComputeFingerprint`, pass-through fields, nil-safe
`GitMetadata`/`Dependencies`) are unit-testable without `Run` itself
needing to expose the `Bundle` it builds -- it doesn't surface anywhere
until T014 renders and delivers it. Added `TestRun_Getwd` (failure ->
exit 1, confirms `buildGitMetadata`/`buildCodeContexts` never called) and
`TestAssembleBundle`/`TestAssembleBundle_NilGitMetadataAndDependencies`.

Caught and fixed a real bug before handing off the gate commands: once
the three new package-level vars existed, every already-passing
success-path test (`TestRun_StdinIgnoredLogsAtDebug`, most of
`TestRun_DetectLanguage`, `TestRun_Parse`'s success case) would have
started shelling out to real `git` against this repo's actual working
directory via the real `os.Getwd`/`codecontext.BuildGitMetadata` --
none of those tests had ever needed to touch this layer before T013.
Fixed with a `TestMain` that stubs all three package-wide to safe,
deterministic, zero-I/O defaults; `TestRun_Getwd` overrides further for
its own case and restores TestMain's defaults (not the real `os.Getwd`)
afterward via `t.Cleanup`.

After Vedant's first gate run, `gocyclo` flagged `Run` at complexity 17
(> 15) -- extracted step 4 (candidate re-selection, `DetectLanguage`,
the exit-4 mapping, both Info logs) into a new `resolveLanguage` helper,
same pattern as T007's `TestParseAll`/`TestParseFixedLang` fix; no
behavior change. Vedant ran `go build ./internal/cli/...`, `gofumpt -l
internal/cli/run.go internal/cli/run_test.go`, `golangci-lint run
./internal/cli/...`, `go test ./internal/cli/... -v -run
'TestRun|TestAssembleBundle'`, and `go test ./internal/cli/...`;
confirmed done, no errors (gocyclo included, after the fix). No commit
message -- `cmd/java`/`cmd/typescript` still don't compile (this task
touched no `cmd/*` file either).
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T014 -- extend `Run` through step 12 (render, `deliver`,
clipboard, `bundle delivered` logging) -- feature-complete in
`internal/cli`
**What happened:** Added `clipboardWrite` (wrapping `clipboard.Write`,
injectable like `getwd`/`buildGitMetadata`/`buildCodeContexts`). Added
`finishDelivery(rendered, outputPath string, noClipboard bool, stdout
io.Writer) int`, extracted from the start this time (not after a gocyclo
failure) given T013's pattern: calls `deliver` -> exit 1 on failure,
clipboard never attempted; on success, `clipboardWrite` unless
`noClipboard`, Warn (not fatal) on failure via `clipboardWarnText` (FR16's
two exact message variants -- the longer "install one of the listed
utilities" suffix for `clipboard.ErrNoClipboardUtility`, the shorter
"pass --no-clipboard" suffix for `ErrClipboardWriteFailed` or anything
else); logs `"bundle delivered"` with `destination` (`"stdout"` or
`outputPath`) and `clipboard` (`copied`/`skipped`/`failed`) attributes.
`Run` itself only picks `render.JSON`/`render.Markdown` by
`input.Format` and calls `finishDelivery` -- `internal/cli`'s side of
the feature is now feature-complete, pending T015's exit-code-matrix
consolidation pass.

Added `clipboardWrite`'s default stub (always succeeds) to `TestMain`,
same reasoning as T013's git stubs -- every existing success-path test
now also reaches delivery. Added `withClipboardWrite` test helper
(override + `t.Cleanup` + a `*bool` call-recorder). Added
`TestRun_Delivery` (stdout+copied, `NoClipboard`+skipped,
clipboard-fails+Warn+exit 0) and `TestRun_Delivery_Output` (writes to
`--output` with nothing on stdout and the clipboard still written; a
deliver failure -> exit 1, clipboard never called). For the
deliver-failure case, used an *existing directory* as `--output` rather
than a nonexistent parent -- the latter fails at `validateOutput` (exit
2) long before `Run` would ever reach `deliver`; an existing directory
passes validation (parent exists, not the input file) but fails
`deliver`'s real `os.OpenFile` with EISDIR, commented inline so it isn't
mistaken for a copy-paste error from T009's own test. Vedant ran `go
build ./internal/cli/...`, `gofumpt -l internal/cli/run.go
internal/cli/run_test.go`, `golangci-lint run ./internal/cli/...`, `go
test ./internal/cli/... -v -run 'TestRun|TestAssembleBundle'`, and `go
test ./internal/cli/...`; confirmed done, no errors. No commit message --
`cmd/java`/`cmd/typescript` still don't compile (this task touched no
`cmd/*` file either).
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-09-26
**Task(s):** T015 -- fill in `run_test.go`'s FR21 exit-code coverage
matrix
**What happened:** Added a doc comment immediately above `TestRun_Help`
listing every FR21 exit code and sub-reason, cross-referenced to the
specific test/case that covers it through `Run` (not just a smaller
unit test from an earlier task, e.g. `ParseAll`'s own `parse_test.go`
cases). Auditing the matrix surfaced five FR21 exit-2 sub-reasons with
no existing Run-level case: no input, empty (whitespace-only) input, an
unsupported `--lang` hint via `Run`'s own dispatch (previously only
tested at the `ParseAll` level), an invalid `--output` path (nonexistent
parent), and `--output` equal to the input file argument. Added all five
as `TestRun_ExitCodeMatrix_UsageErrors`. Every other FR21 code/reason
already had Run-level coverage from T010-T014 and just needed the
cross-reference added. Hit the empty-`oldText` `edit_file` gotcha again
while appending the new test function -- it landed before `package cli`
a second time in this task alone, this time compounded by the Filesystem
MCP connector dropping mid-fix; reconnected and corrected before handing
off the gate commands. Vedant ran `go build ./internal/cli/...`,
`gofumpt -l internal/cli/run_test.go`, `golangci-lint run
./internal/cli/...`, `go test ./internal/cli/... -v -run
TestRun_ExitCodeMatrix`, and `go test ./internal/cli/...`; confirmed
done, no errors. No commit message -- `cmd/java`/`cmd/typescript` still
don't compile (this task touched no `cmd/*` file either).
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-10-07
**Task(s):** T016 -- rewrite `cmd/all/main.go`
**What happened:** Rewrote `cmd/all/main.go` entirely: builds
`cli.BinaryConfig` with `Candidates: [tsparser.NewJavaScriptParser(),
tsparser.NewTypeScriptParser()]`, `Resolvers` mapping both
`contract.LanguageJavaScript` and `contract.LanguageTypeScript` to
`tsdeps.ResolveDependencies` (006b -- one resolver shared by both
languages, since npm/yarn dependency resolution doesn't depend on which
of the two produced the trace), five real `Examples` built from the
runtime-derived `progName` (`filepath.Base(os.Args[0])`), calls
`cli.Run`, `os.Exit`s the result. Kept only `stdinIsPiped` from the old
`main.go`; removed the 002a stub entirely (Info `parsed input`, Debug
dump, the old 3-arg `ParseAll` call). Aliased both `typescript`-named
packages (`internal/parser/typescript` and
`internal/dependency/typescript` are both literally `package typescript`)
as `tsparser`/`tsdeps` rather than tasks.md's illustrative
`dependencytypescript` -- flagged to Vedant as a naming choice, not
confirmed against any explicit requirement (import aliases aren't part
of any public contract).

Updated `help_test.go`'s anti-drift case: replaced `placeholderExamples`
with `cmdAllExamples`, the literal real five examples (can't import
`cmd/all`'s `main()`-local `Examples` across package boundaries -- a
`main` package exports nothing -- so this is deliberately kept in sync
by hand, same as `CONVENTIONS.md`'s exit-code table). Added a skip for
the piped `"cat trace.txt | stba"` example in
`TestUsageText_ExamplesRoundTripThroughParseAll`, since it isn't a plain
invocation and breaks `exampleArgs`' "drop the leading program-name
token" assumption.

First genuine end-to-end verification in the whole feature: Vedant ran
`go build ./cmd/all` (binary lands at the repo root, not inside
`cmd/all/` -- corrected after an initial wrong path) and ran it against
two real fixtures (`internal/parser/typescript/testdata/
crash-typeerror-no-cause.txt` auto-detected as javascript,
`ts-native-execution.txt` with `--lang=typescript`). Both produced a
complete, correctly-formatted markdown bundle on stdout (language, OS,
runtime, git metadata, fingerprint, exception chain with own-code/
runtime bucketing) and degraded gracefully on stderr (own-code context
and dependency-resolution Warn lines, since the fixtures' paths are from
a different machine than this checkout) -- no errors, confirming the
full detect -> parse -> code-context -> dependency -> render -> deliver
pipeline end to end for the first time. Vedant also ran `go build
./internal/cli/...`, `gofumpt -l cmd/all/main.go
internal/cli/help_test.go`, `golangci-lint run ./cmd/all/...
./internal/cli/...`, and `go test ./internal/cli/...`; confirmed done,
no errors. No commit message -- `cmd/java`/`cmd/typescript` still don't
compile (old 4-arg `ParseFixedLang` calls, T017/T018), so `go build
./...` repo-wide and `lefthook`'s gate still fail.
**Deviations from plan (if any):** import aliases `tsparser`/`tsdeps`
instead of tasks.md's illustrative `dependencytypescript` (see above --
flagged, not confirmed).
**New open questions:** none

---

**Date:** 2026-10-07
**Task(s):** T017 -- rewrite `cmd/java/main.go`
**What happened:** Rewrote `cmd/java/main.go`: `cli.BinaryConfig{
FixedLang: "java", Candidates: nil, Resolvers: nil, Examples: [...]}`
(two examples: file and piped stdin), calls `cli.Run`, `os.Exit`s the
result. `Candidates`/`Resolvers` both nil by design until 005a/005b
exist -- nil `Candidates` is exactly what makes `selectCandidates`'
FR10 fail-fast fire on every normal invocation. Same stub removal as
T016 (kept only `stdinIsPiped`). Added `cmdJavaExamples` and
`TestUsageText_CmdJavaExamplesRoundTripThroughParseFixedLang` to
`help_test.go` -- every case is *expected* to surface
`languageUnsupportedError`, so this only guards the flags themselves
parsing cleanly, same pattern as T016's `cmdAllExamples` test.

Vedant ran `go build ./cmd/java`, `go build ./internal/cli/...`,
`gofumpt -l cmd/java/main.go internal/cli/help_test.go`, `golangci-lint
run ./cmd/java/... ./internal/cli/...`, `go test ./internal/cli/...`,
and the three manual checks T017 calls for: a file argument and piped
stdin both logged exactly `"Java is not supported yet"` at Error level
and exited 2; `--help` printed the full usage text (including the real
`Examples` and the exit-code table) and exited 0. All three matched
exactly, no errors. No commit message -- `cmd/typescript` still doesn't
compile (old 4-arg `ParseFixedLang` call, T018), so `go build ./...`
repo-wide and `lefthook`'s gate still fail.
**Deviations from plan (if any):** n/a
**New open questions:** none

---

**Date:** 2026-10-09
**Task(s):** T018 -- rewrite `cmd/typescript/main.go`
**What happened:** Rewrote `cmd/typescript/main.go`:
`cli.BinaryConfig{FixedLang: "typescript", Candidates:
[NewJavaScriptParser(), NewTypeScriptParser()], Resolvers: both
languages -> tsdeps.ResolveDependencies, Examples: [...]}`. Both parsers
are registered, not just TypeScript's, because the fixed `"typescript"`
hint narrows to both (a TS-compiled-to-JS trace can legitimately detect
as javascript) and Detect still decides. Four `Examples`, deliberately
without a `--lang` one -- this binary never registers that flag, so such
an example would fail its own parser and mislead anyone reading
`--help`. Added `cmdTypescriptExamples` and
`TestUsageText_CmdTypescriptExamplesRoundTripThroughParseFixedLang` to
`help_test.go`; unlike the Java one, it passes a fake TypeScript
candidate so the fail-fast doesn't fire and the examples actually reach
`validateOutput`/`readTrace`.

First full repo-wide gate since T006 broke the `cmd/*` call sites. Vedant
ran `go build ./...`, `gofumpt -l .`, `golangci-lint run ./...`, `go
test ./...`: all clean (0 lint issues, every package ok, gofumpt printed
nothing). Manual checks matched: a real trace (`ts-native-execution.txt`)
exits 0 with a bundle on stdout; `--lang=typescript` exits 2 with
`parsing flags: unknown flag: --lang`; `--help` shows no `--lang` line
and lists this binary's own examples. Commit is now possible for the
first time since T006 -- see the note below on how to split it.
**Deviations from plan (if any):** n/a
**New open questions:** T006-T018 all sit uncommitted in one working
tree and cannot be split into per-task commits that each build (the
`cmd/*` call sites only compile once all three are rewritten), which
conflicts with the "atomic commit per task" preference. Raised with
Vedant; see his decision below if recorded.

---

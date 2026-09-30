# Tasks: Pipeline wiring

Derived from `plan.md`. Work through in order unless told otherwise.
`[ ]` todo, `[~]` in progress, `[x]` done. Append a `progress.md` entry
after each task.

- [x] **T001** — Verify the two carried-forward risk items from `plan.md`
  before writing any new code: (1) grep `internal/render` and
  `internal/contract` for any comparison against `OSLinux`/`OSDarwin`/
  `OSWindows` (Q11's `Bundle.OS` answer depends on none existing); (2)
  confirm `--lang`, `--format`, `-v` already have non-empty help strings
  in `parse.go`.
  - Depends on: none
  - Acceptance: both findings recorded in `progress.md`; if (1) finds a
    validation site, stop and flag it before continuing to T002.

- [x] **T002** — Add `Output string` and `NoClipboard bool` to
  `internal/cli.Input` (`input.go`), with doc comments matching the
  file's existing style.
  - Depends on: T001
  - Acceptance: `go build ./internal/cli/...` passes; no other file
    changed yet.

- [x] **T003** — Extract flag registration out of `ParseAll` and
  `ParseFixedLang` into a shared `registerFlags(fs *pflag.FlagSet,
  fixedLang string) (langFlag, formatFlag, outputFlag *string,
  noClipboard *bool, verbosity *int)`-shaped helper in `parse.go` (exact
  shape is an implementation choice; goal is one registration path for
  both `ParseAll`/`ParseFixedLang` and `help.go`'s `usageText`, per
  plan.md). Do not change behavior yet -- `ParseAll`/`ParseFixedLang`
  call the new helper and proceed exactly as before.
  - Depends on: T002
  - Acceptance: existing `parse_test.go` passes unmodified.

- [x] **T004** — Add `validateOutput(outputPath, fileArg string) error`
  to `read.go` (FR17, Q4): parent-directory-exists-and-is-a-directory
  check; `os.SameFile` same-as-input-file check (skipped when
  `fileArg == ""`). Add `read_test.go` cases for both failures and the
  pass-through empty-`outputPath` case. Not wired into `ParseAll`/
  `ParseFixedLang` yet.
  - Depends on: T003
  - Acceptance: new tests pass in isolation; nothing else changed.

- [x] **T005** — Add `internal/cli/candidates.go`: `hintLanguages`,
  `hintDisplayName`, `selectCandidates(hint string, registered
  []parser.LanguageParser) ([]parser.LanguageParser, error)`. Add
  `internal/cli/errors.go`: `languageUnsupportedError`. Add
  `candidates_test.go` with two fake `parser.LanguageParser`
  implementations (FR3, FR9, FR10).
  - Depends on: T003
  - Acceptance: `go test ./internal/cli/... -run Candidates` passes;
    `selectCandidates("", registered)` returns `registered` unchanged;
    `selectCandidates("java", <no java parser in registered>)` returns
    the `languageUnsupportedError` with `Error() == "Java is not
    supported yet"`.

- [ ] **T006** — Update `ParseAll`'s signature to
  `ParseAll(args []string, stdin io.Reader, stdinIsPiped bool,
  candidates []parser.LanguageParser) (Input, int, error)`. Wire the new
  order: `Parse -> validateLang -> validateFormat ->
  selectCandidates(fail-fast) -> validateOutput -> readTrace`. Populate
  `Input.Output`/`Input.NoClipboard`. Update every `ParseAll` call site in
  `parse_test.go` to pass a candidates slice (fakes from T005 where the
  test doesn't care about real parsers).
  - Depends on: T004, T005
  - Acceptance: new `parse_test.go` cases: unsupported-hint fail-fast
    exits before `readTrace` is invoked (assert via a stdin fake that
    records read calls); `--format=bogus --lang=java` reports the format
    error, not the language error (plan.md's documented order); all
    existing 002a `ParseAll` test cases still pass with the added
    parameter; a case with `lang="java"` and a nonexistent `--output`
    parent directory reports the Java-not-supported message, not the
    output-path error (plan.md's candidate-fail-fast-before-
    `validateOutput` order); a case with `lang="typescript"` (supported,
    candidates non-empty) and the same nonexistent `--output` parent
    directory instead surfaces the output-path error, confirming
    `validateOutput` is actually reached once candidate fail-fast
    doesn't fire.

- [ ] **T007** — Same change as T006, for `ParseFixedLang` (adds the
  `candidates` parameter; the fixed `lang` is passed straight into
  `selectCandidates` as the hint). `cmd/java`'s eventual candidates slice
  will be empty, which is exactly what should make its fail-fast fire.
  Apply the same order as T006 (`validateFormat` before the
  `selectCandidates` fail-fast; there's no `validateLang` step here
  since `lang` is fixed, not a flag).
  - Depends on: T006
  - Acceptance: a `ParseFixedLang` test case with `lang="java"` and an
    empty candidates slice returns the `languageUnsupportedError` before
    touching stdin; a case with `lang="typescript"` and both fake
    parsers registered succeeds through to `readTrace`; a case with
    `lang="java"`, empty candidates, and an invalid `--format` value
    returns the format error, not `languageUnsupportedError` (mirrors
    T006's `--format=bogus --lang=java` case, but through the
    fixed-lang path `cmd/java` actually uses); a case with `lang="java"`,
    empty candidates, and a nonexistent `--output` parent directory
    returns the Java-not-supported error, not the output-path error
    (same ordering, through `cmd/java`'s actual path); a case with
    `lang="typescript"`, both fake parsers registered, and the same
    nonexistent `--output` parent directory surfaces the output-path
    error instead, confirming the same wiring on `cmd/typescript`'s
    path.

- [ ] **T008** — Add `internal/cli/help.go`: `Example`, `exitCodeDocs`,
  `usageText(programName string, fs *pflag.FlagSet, examples []Example)
  string` (FR19-20). Reuses T003's `registerFlags` to build the `fs` it
  renders `FlagUsages()` from. Add `help_test.go`.
  - Depends on: T003
  - Acceptance: for a `cmd/all`-shaped `fs` (has `--lang`) and a
    `cmd/java`/`cmd/typescript`-shaped `fs` (no `--lang`), `usageText`'s
    output includes/excludes `--lang` accordingly; every `exitCodeDocs`
    code number appears in the output; a placeholder set of examples
    (final real examples land in T016-018) round-trips through flag
    parsing with no error.

- [ ] **T009** — Add `internal/cli/output.go`: `deliver(bundle string,
  outputPath string, stdout io.Writer) error` (FR13, FR15, FR17). Handles
  the trailing-newline rule and silent overwrite; opens the file only at
  call time (never earlier). Add `output_test.go`.
  - Depends on: T002
  - Acceptance: stdout case writes to the injected `io.Writer` with the
    newline rule applied; file case overwrites an existing file; a
    write-failure case (e.g. a directory removed after being statted, or
    a read-only path) returns a non-nil error and touches nothing else.

- [ ] **T010** — Add `internal/cli/run.go`: `DependencyResolver` type,
  `BinaryConfig` struct, and `Run` stubbed through step 3 only (parse,
  `pflag.ErrHelp` -> print `usageText`, return 0; any other parse error ->
  `slog.Error`, return 2; configure `slog` from verbosity; log
  `stdin ignored` if set). No pipeline execution yet -- `Run` returns 0
  immediately after step 3 for now, with a `// TODO T011+` marker.
  - Depends on: T005, T006, T007, T008
  - Acceptance: `run_test.go` cases for the help path (exit 0, usage text
    contains the program name) and a flag-error path (exit 2) pass;
    plus the FR19/Q14 ordering cases: `--lang=cobol -h` still prints
    help and exits 0 (bad flag *value*, not an unknown flag, doesn't
    block `-h`), `--bogus -h` exits 2 on the unknown flag; a case with a
    file argument and piped stdin both present asserts the Debug
    `stdin ignored` line (FR22, carried over from 002a FR5) fires,
    captured via a test-local `slog` handler.

- [ ] **T011** — Extend `Run`: re-run `selectCandidates` on the parsed
  `Input.LangHint`, call `parser.DetectLanguage`, map `ErrNoMatch`/
  `ErrAmbiguous` to exit 4 with the FR11/FR12 messages (including the
  single-line-input tip). Log `language resolved` at Info (attribute
  `language`) always; additionally, when the hint doesn't match the
  detected language, log a second, separate FR9 Info line (`detected
  language differs from --lang hint`, attributes `hint`/`detected`) --
  two log calls in the mismatch case, not one.
  - Depends on: T010
  - Acceptance: `run_test.go` cases using two fake parsers: no match ->
    exit 4, message names both; both match -> exit 4, ambiguous message;
    single-line no-match input -> message includes the tip text,
    multi-line does not; hint differs from the one matching parser's
    `Language()` -> both the `language resolved` Info line AND the
    separate FR9 hint-mismatch Info line fire (capture via a test-local
    `slog` handler; assert on both messages and their attribute sets,
    not just "a log fired"); a plain, non-mismatch successful-detection
    case asserts the `language resolved` Info log fires with a
    `language` attribute and that no FR9 hint-mismatch line fires.

- [ ] **T012** — Extend `Run`: call the resolved parser's `Parse`. Map
  `errors.Is(err, parser.ErrUnparseable)` to exit 3 with the FR12 message;
  any other non-nil error to exit 1.
  - Depends on: T011
  - Acceptance: fake parser returning an `ErrUnparseable`-wrapping error
    -> exit 3; fake parser returning a plain error -> exit 1; fake parser
    returning success -> proceeds (verified by T013 taking over from
    here).

- [ ] **T013** — Extend `Run`: `os.Getwd` (via a package-level `var getwd
  = os.Getwd` for test override) -> exit 1 on failure; call
  `codecontext.BuildGitMetadata`, `codecontext.BuildCodeContexts`, and
  `cfg.Resolvers[language]` if present; assemble `contract.Bundle` per
  FR4.
  - Depends on: T012
  - Acceptance: a test overriding `getwd` to fail -> exit 1, no further
    calls made (assert via fakes that record invocation); a success path
    produces a `Bundle` with `SchemaVersion`, `OS`, `Fingerprint`, and
    `Dependencies` populated as FR4 specifies, using fake git/dependency
    functions (no real `git`/`node`).

- [ ] **T014** — Extend `Run`: render via `render.Markdown`/`render.JSON`
  per `Input.Format`; call `deliver` (T009); on success, call
  `clipboard.Write` unless `NoClipboard`, with the FR16 Warn text and
  non-fatal handling; log `bundle delivered` at Info.
  - Depends on: T009, T013
  - Acceptance: `run_test.go` full success case (exit 0, stdout/file has
    the rendered bundle, fake clipboard writer was called); `--output`
    case (nothing written to the stdout stand-in); a `deliver` failure ->
    exit 1, fake clipboard writer never called; a clipboard-writer
    failure after a successful `deliver` -> exit 0, Warn logged; a
    `NoClipboard: true` case asserts the fake clipboard writer's call
    log stays empty; the `bundle delivered` Info log fires with
    `destination` and `clipboard` attributes, covering at least one
    stdout case and one `--output` case, and at least one each of
    clipboard `copied`/`skipped`/`failed`.

- [ ] **T015** — Fill in `run_test.go`'s exit-code matrix so every
  `spec.md` exit-code acceptance criterion (FR21) has a corresponding
  case in one place, cross-referencing which criterion each covers.
  - Depends on: T014
  - Acceptance: `go test ./internal/cli/...` covers exit 0/1/2/3/4 at
    least once each through `Run` specifically (not only through the
    smaller unit tests from earlier tasks).

- [ ] **T016** — Rewrite `cmd/all/main.go`: build `BinaryConfig` with
  `Candidates: []parser.LanguageParser{typescript.NewJavaScriptParser(),
  typescript.NewTypeScriptParser()}`, `Resolvers` mapping both
  `contract.LanguageJavaScript` and `contract.LanguageTypeScript` to
  `dependencytypescript.ResolveDependencies`, real usage `Examples`, and
  `os.Args[0]`-derived program name; call `cli.Run`; `os.Exit` the
  result. Remove the 002a stub (Info `parsed input`, Debug dump) --
  keep nothing from the old `main` except `stdinIsPiped`.
  - Depends on: T015
  - Acceptance: `go build ./cmd/all` succeeds; a manual/scripted run
    against a real TS/JS trace fixture (e.g. one of 006a's testdata
    files) produces a bundle on stdout and exits 0 (this is the first
    point in the whole feature where a real end-to-end run is possible);
    `help_test.go`'s anti-drift case (T008) is updated to round-trip
    this binary's actual `Examples` through `ParseAll` with no flag
    error, replacing the placeholder set.

- [ ] **T017** — Rewrite `cmd/java/main.go`: `BinaryConfig{FixedLang:
  "java", Candidates: nil, Resolvers: nil, Examples: ...}`. Remove the
  002a stub.
  - Depends on: T015
  - Acceptance: `go build ./cmd/java` succeeds; running it any way (file,
    stdin, `--help`) either prints help (exit 0) or exits 2 with exactly
    `Java is not supported yet` -- never exit 0 from a normal run, never
    a stub log line; `help_test.go`'s anti-drift case is updated to
    round-trip this binary's actual `Examples` through `ParseFixedLang`
    with no flag error.

- [ ] **T018** — Rewrite `cmd/typescript/main.go`: `BinaryConfig{
  FixedLang: "typescript", Candidates: [js, ts parsers], Resolvers: {js:
  resolver, ts: resolver}, Examples: ...}`. Remove the 002a stub.
  - Depends on: T015
  - Acceptance: `go build ./cmd/typescript` succeeds; same end-to-end
    check as T016 against the same fixture; `help_test.go`'s anti-drift
    case is updated to round-trip this binary's actual `Examples`
    through `ParseFixedLang` with no flag error.

- [ ] **T019** — Update `specs/002a-cli-input-handling/spec.md`: mark
  FR13 and its three dependent acceptance criteria (Info summary at
  `-v`; Info summary + dump at `-vv`; the criterion mentioning both) as
  superseded by 002b, with a one-line pointer to this feature. Close the
  "temporary stdout behavior" note referenced in this feature's FR22.
  Also update `specs/002a-cli-input-handling/plan.md`'s "API / contracts"
  section (the `ParseAll`/`ParseFixedLang` signatures) and "Data model"
  section (the `Input` struct) to show the `candidates` parameter and
  the `Output`/`NoClipboard` fields this feature adds -- per this
  project's practice (004, 006b) of keeping a touched dependency's
  plan.md in sync, not just its spec.md.
  - Depends on: T016, T017, T018
  - Acceptance: 002a's `spec.md` no longer asserts behavior the code no
    longer has; 002a's `plan.md` documents `ParseAll`/`ParseFixedLang`
    and `Input` exactly as this feature leaves them, not as 002a
    originally shipped them.

- [ ] **T020** — Update `memory/known-gaps.md`: remove or mark done the
  006a bare-stack-shape row (now covered by FR12's tip). Check the
  corresponding box in `specs/006a-ts-js-parser/spec.md` if one exists
  for this criterion.
  - Depends on: T016
  - Acceptance: no open reference to this gap remains pointing at an
    unimplemented feature.

- [ ] **T021** — Update `CONVENTIONS.md`'s exit-code line: code `0` now
  explicitly includes `--help`; code `2` explicitly includes "an
  unsupported `--lang` value with no matching parser" and "a bad
  `--output` path." Keep it prose-length, consistent with the rest of the
  file.
  - Depends on: T015
  - Acceptance: the line no longer undersells what's actually true of
    exit 0/2, and still matches `internal/cli/help.go`'s `exitCodeDocs`
    in substance (not generated from it -- see plan.md's Risks section).

- [ ] **T022** — Full gate: `gofumpt -l -w .`, `golangci-lint run ./...`,
  `go build ./...`, `go test ./...` (or `lefthook run pre-commit`).
  - Depends on: T001-T021
  - Acceptance: all four pass clean, repo-wide, not just for
    `internal/cli`.

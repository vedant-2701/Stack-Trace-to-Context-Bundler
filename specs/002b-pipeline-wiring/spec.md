# Spec: Pipeline wiring

**Status:** Ready for planning -- every [NEEDS CLARIFICATION] is resolved (see `qa-log.md`, Q1-Q18)
**Folder:** specs/002b-pipeline-wiring

## Overview

002a reads and validates input, then stops with a temporary stub log. 002b
connects everything already built into the tool's real behavior: detect the
language (003b), parse the trace (006a), attach own-code context and git
metadata (004) and dependency versions (006b), render it (007/008), deliver
it to stdout or a file, and copy it to the clipboard (009). It also removes
002a's stub output and adds three user-facing pieces the pipeline needs:
`-o`/`--output`, `--no-clipboard`, and a real `-h`/`--help`.

Java cannot be wired yet -- there is no Java parser (005a) or Java
dependency resolver (005b). The pipeline is built generically so those
land later as one parser plus one resolver; until then any Java request
fails fast with a clear error instead of producing a wrong bundle.

## User stories

- As a developer debugging a Node.js/TypeScript crash, I want to run one
  command on a trace (file or piped) and get a complete bundle on my
  clipboard, ready to paste into an AI chat.
- As a developer scripting the tool, I want only the bundle on stdout, or
  in a file I name with `-o`, so I can pipe or save it.
- As a developer on a headless machine, I want a failed clipboard copy to
  not turn a good bundle into a failed run.
- As a new user, I want `--help` to show every flag, worked examples, and
  what each exit code means.
- As a developer who gives bad input or flags, I want a specific error and
  a stable exit code.

## Functional requirements

### Pipeline

1. After 002a has read and validated input, the pipeline runs in this
   order: choose candidate parsers (FR3) -> `parser.DetectLanguage` ->
   the detected parser's `Parse` -> `codecontext.BuildGitMetadata` ->
   `codecontext.BuildCodeContexts` -> the detected language's dependency
   resolver (if one is registered) -> assemble `contract.Bundle` ->
   render -> deliver. The code-context and dependency steps run
   sequentially, written as independent calls (Q18).
2. `Parse` is only ever called on the parser `DetectLanguage` returned for
   that trace. `Detect` is never bypassed, including when a language hint
   is given: the 006a engine accepts any first line containing `": "`, so
   a non-Node trace parsed without `Detect` would yield a plausible but
   wrong bundle (Q7).
3. Candidate parsers are supplied by each binary's `main.go`, not
   hardcoded in `internal/cli` (Q1). An empty language hint means every
   registered parser. A non-empty hint maps, through a per-binary hint
   table supplied by `main.go`, to a set of languages, and the candidates
   are the registered parsers in that set: `typescript` -> `javascript`
   and `typescript`; `java` -> `java`. `cmd/all` and `cmd/typescript`
   register the JavaScript and TypeScript parsers; `cmd/java` registers
   none.
4. The `contract.Bundle` is assembled as: `SchemaVersion` =
   `contract.SchemaVersion`; `Language` = the detected parser's
   `Language()`; `OS` = `contract.OS(runtime.GOOS)` for every platform,
   passed through verbatim (Q11); `RawInput` and `RawInputTruncated` from
   the read input; `Fingerprint` = `contract.ComputeFingerprint(chain)`;
   `Runtime` and `Chain` from `Parse`; `CodeContexts` from
   `BuildCodeContexts`; `GitMetadata` from `BuildGitMetadata` (nil when no
   repo); `Dependencies` from the resolver registered for the detected
   `Language` (nil when none is registered or the resolver returns nil).
   `javascript` and `typescript` both use the TS/JS resolver; the resolver
   map is supplied by `main.go`. No change to `internal/contract`, no
   schema bump.
5. The working directory for the git and dependency steps is
   `os.Getwd()`. If `Getwd` fails, the run exits 1 (Q17).
6. `--format` selects `render.Markdown` or `render.JSON`.
7. `BuildGitMetadata`, `BuildCodeContexts` and the dependency resolvers
   never return errors -- they degrade internally (nil results, per-frame
   statuses, Warn logs). The pipeline adds no error branch or exit code
   for them (Q12).
8. The pipeline runs on `context.Background()`: no signal handling, no
   global deadline. Ctrl+C uses default SIGINT behavior, and output is
   only written after the whole pipeline succeeds. Standing invariant: if a
   cancellable or deadline context is ever introduced, `ctx.Err()` must be
   checked before any output is written, because the enrichment steps
   degrade silently on cancellation (Q12).

### Language hint, detection, and their errors

9. `--lang=typescript` on `cmd/all`, and `cmd/typescript`'s fixed
   `typescript`, narrow the candidates to the hint's language set (FR3)
   and leave `Detect` to decide, so `Bundle.Language` may be
   `javascript`. When the detected language differs from the hint, log at
   Info: `detected language differs from --lang hint` with `hint` and
   `detected` attributes. Invisible at the default Warn level, visible
   with `-v` (Q7).
10. A hint whose language set has no registered parser is a usage error:
    exit 2, one Error-level message `<Name> is not supported yet` where
    `<Name>` comes from the hint table (`Java` for `java`), so exactly
    `Java is not supported yet`. No feature IDs or internal names.
    This applies to `--lang=java` on `cmd/all` and to every invocation of
    `cmd/java`. It runs after flag parsing and validation and before any
    input is read, and `DetectLanguage` is never called with zero
    candidates. `-h`/`--help` wins over it (Q8, Q14). When a Java parser is
    registered later, this check stops firing with no code change.
11. `parser.ErrNoMatch` -> exit 4, Error-level message `could not detect
    the language of the input: <error text>` (the error text already names
    every checked language). `parser.ErrAmbiguous` -> exit 4, message
    `the input matched more than one language: <error text>`. The two are
    told apart with `errors.Is` (CONVENTIONS.md; 003b).
12. When the exit-4 no-match case's trimmed input is a single line, the
    message also carries a tip as a second sentence: `The input is a single
    line with no stack frames, so there is nothing for this tool to add --
    paste it directly into your LLM instead. If it came from Node.js,
    Error.stackTraceLimit may have been set to 0.` (Q9). Multi-line inputs
    get no tip. `parser.ErrUnparseable` from `Parse` -> exit 3, message
    `could not parse the input: <error text>`. Any other `Parse` error ->
    exit 1.

### Delivery

13. By default the rendered bundle goes to stdout and nothing else does
    (Article II); logs go to stderr. With `-o`/`--output <path>` the
    bundle goes to that file and nothing is written to stdout (Q3).
14. The bundle is also copied to the clipboard by default, byte-for-byte
    via `clipboard.Write`. `--no-clipboard` skips the copy (Q2).
15. Trailing newline (Q10): for stdout and the `--output` file only, one
    `\n` is appended if the rendered text does not already end with one.
    The clipboard copy is the rendered text unchanged, so for JSON it is
    one byte shorter than the stdout/file copy (008 Q5 stays intact for
    the paste payload).
16. Order and failure handling (Q5, Q15): write stdout or the `--output`
    file first. If that fails, exit 1 and leave the clipboard untouched.
    Only after a successful write, attempt the clipboard. A clipboard
    failure is a Warn on stderr, exit 0, no exit code allocated (this
    resolves 009 FR8's deferred question). The Warn text is
    `could not copy the bundle to the clipboard: <error text>` followed by
    `; install one of the listed utilities, or pass --no-clipboard to skip
    copying` for `clipboard.ErrNoClipboardUtility`, or `; pass
    --no-clipboard to skip copying` for `clipboard.ErrClipboardWriteFailed`.
    If a downstream reader closes stdout (`| head -1`), Go's default
    SIGPIPE behavior ends the process before the clipboard step -- accepted.
17. `--output` handling (Q4, Q6). Usage errors, exit 2, checked before any
    input is read or pipeline work is done: the path's parent directory
    does not exist or is not a directory; the path resolves to the same
    file as the input file argument (checked with `os.SameFile`; not
    applicable when input comes from stdin). An existing file at the path
    is overwritten silently; there is no `--force`. The file is opened and
    written only after rendering succeeds, so a pipeline failure never
    touches an existing output file. A runtime write failure (permission
    denied, disk full, ...) is exit 1 with an error naming the path. No
    temp-file-and-rename.
18. `-o`/`--output <path>` and `--no-clipboard` are registered on all three
    binaries (`cmd/all`, `cmd/java`, `cmd/typescript`), each with a help
    string. `cli.Input` gains `Output` and `NoClipboard` fields (Q3, Q17).

### Help

19. `-h`/`--help` prints the usage text to stderr and exits 0 -- not
    through `slog.Error`, not exit 2 (Q13). Stdout stays bundle-only.
    Each binary lists its own flag set (`cmd/all` shows `--lang`;
    `cmd/java` and `cmd/typescript` do not). `pflag`'s left-to-right
    handling is taken as-is: `stba --lang=cobol -h` prints help,
    `stba --bogus -h` fails on the unknown flag (exit 2), `stba -h --bogus`
    prints help, `cmd/java --help` prints help (Q14).
20. Usage text contents, in order: a synopsis line using
    `filepath.Base(os.Args[0])` as the program name; one sentence on input
    (a file argument, or stdin when piped; the file wins if both are
    present); the flag listing (every flag has a help string); usage
    examples; an exit-code table. The exit codes and examples are defined
    once as Go data and the help text is rendered from them, with tests
    that (1) every exit code appears in the help text and (2) every
    example's arguments are accepted by `ParseAll`/`ParseFixedLang`
    (Q14, Q17).

### Exit codes and logging

21. Exit codes, unchanged set from CONVENTIONS.md (no new codes): 0 success
    (including `--help` and a failed clipboard copy after a successful
    write); 1 unexpected/internal error (including `Getwd`, an output write
    failure, and a `Parse` error that is not `ErrUnparseable`); 2 usage
    error (flag errors, no input, empty input, an unsupported language hint,
    an invalid `--output` path, `--output` equal to the input file); 3
    unparseable input; 4 language undetected or ambiguous.
22. 002a's stub logging is removed from all three `main.go` files: the Info
    `parsed input` line and the Debug `Input` dump. The Debug `stdin
    ignored: file argument took precedence` line stays (002a FR5). New Info
    lines: `language resolved` (attribute `language`; plus the hint-mismatch
    line in FR9) and `bundle delivered` (attributes `destination` =
    `stdout` or the `--output` path, and `clipboard` = `copied`, `skipped`
    or `failed`). The existing `dependencies resolved` Info line from 006b
    is unchanged (Q16).
23. Log levels stay as in 002a: Warn by default, `-v` Info, `-vv` Debug.
    Every error path logs once through `slog.Error` before exiting.

## Non-functional requirements

- Article II: stdout carries only the bundle; help, logs and warnings go
  to stderr.
- Article VI: no path may produce a plausible-looking wrong bundle. `Parse`
  is never called without a passing `Detect` (FR2); a failed output write
  never leaves the clipboard overwritten (FR16).
- Article VII/VIII: no new third-party dependency (`pflag` is already
  adopted, ADR 0002). Parsers and resolvers are injected because three
  binaries with different sets already exist, not for hypothetical
  languages.
- Tests never call real `git`, `node`, or clipboard utilities (CONVENTIONS
  rule: anything that shells out is tested behind a fake); the pipeline's
  steps, the clipboard writer and `Getwd` are injectable.
- The three `main.go` files become thin wiring; orchestration lives in
  `internal/cli`.
- Performance: sequential execution. Worst case is bounded only by the
  existing per-call git timeouts (10s per call in `codecontext/runner.go`).
  No overall deadline (FR8).

## Out of scope

- Wiring a Java parser or resolver (005a/005b). When they land, they are
  registered in `main.go` and the FR10 check stops firing.
- Running the code-context and dependency steps in parallel, or
  parallelizing per-frame git calls inside 004 (Q18).
- Signal handling, a global deadline, atomic (temp-file-and-rename) output
  writes, `--force`, and `-o -` as a stdout alias.
- `--pretty` (013) and `--context-lines` (011). When either lands, the
  help text's flag listing updates automatically and its examples need
  reviewing.
- Any change to `internal/contract`, the renderers, the parsers,
  `codecontext`, `dependency/typescript`, or `clipboard`.
- Detection improvements: compiled-TypeScript classification and the
  bare-stack shape stay as documented in `memory/known-gaps.md`, apart from
  FR12's single-line tip.
- Bun/Deno runtimes (`memory/known-gaps.md`).

## Acceptance criteria

- [ ] Given a Node.js trace file with `.ts` frames, when the tool runs on
      `cmd/all`, then stdout holds the Markdown bundle with `Language:
      typescript`, the clipboard holds the same text, and the exit code is 0.
- [ ] Given `--format json`, when the tool runs, then stdout holds the JSON
      bundle plus one trailing newline and the clipboard holds the JSON
      without it.
- [ ] Given a plain-JS trace (no `.ts` frames) and `--lang typescript`, when
      the tool runs, then `Bundle.Language` is `javascript` and an Info line
      (`-v`) reports hint `typescript`, detected `javascript`.
- [ ] Given a trace from another language and no `--lang`, when the tool
      runs, then it exits 4 with the FR11 message and does not call `Parse`.
- [ ] Given a single-line input with no frames (e.g. `TypeError: fetch
      failed`), when the tool runs, then it exits 4 and the message also
      contains the FR12 tip; given a multi-line no-match input, the message
      has no tip.
- [ ] Given a trace matching two registered parsers (fake parsers in
      tests), when the tool runs, then it exits 4 with the ambiguous
      message.
- [ ] Given a `Parse` error wrapping `ErrUnparseable`, when the tool runs,
      then it exits 3; given any other `Parse` error, it exits 1.
- [ ] Given `cmd/all --lang=java`, or any `cmd/java` invocation with valid
      flags, when the tool runs, then it exits 2 with exactly `Java is not
      supported yet`, and no input is read (a blocking stdin is never read).
- [ ] Given `cmd/java --format=bogus`, when the tool runs, then the format
      error is reported (exit 2), not the Java message.
- [ ] Given `-o out.md`, when the tool runs, then the bundle plus a trailing
      newline is in `out.md`, nothing is written to stdout, and the
      clipboard is still written unless `--no-clipboard` is set.
- [ ] Given `--no-clipboard`, when the tool runs, then the clipboard writer
      is never called.
- [ ] Given `-o` whose parent directory does not exist, when the tool runs,
      then it exits 2 before any input is read or pipeline step runs.
- [ ] Given `stba trace.txt -o trace.txt`, when the tool runs, then it exits
      2, `trace.txt` is unchanged, and nothing else is written.
- [ ] Given `-o` pointing at an existing file, when the tool runs, then it
      is overwritten without a prompt; given a pipeline failure (exit 3/4),
      an existing output file is untouched.
- [ ] Given an `-o` write failure at the final write, when the tool runs,
      then it exits 1 with an error naming the path and the clipboard writer
      is never called.
- [ ] Given a failing clipboard writer, when the tool runs after a
      successful write, then a Warn (FR16 text) is logged, the exit code is
      0, and the bundle was already delivered.
- [ ] Given `os.Getwd` failing (injected), when the tool runs, then it
      exits 1.
- [ ] Given `-h` or `--help` on each of the three binaries, when the tool
      runs, then the usage text goes to stderr, stdout is empty, the exit
      code is 0, `cmd/all`'s text lists `--lang` and the others' do not, and
      every flag, every exit code and every example appears.
- [ ] Given each example in the usage data, when its arguments are parsed
      by the matching `ParseAll`/`ParseFixedLang`, then no flag error
      occurs.
- [ ] Given `--lang=cobol -h`, when the tool runs, then help is printed
      (exit 0); given `--bogus -h`, exit 2.
- [ ] Given a normal run at default verbosity, when the tool runs, then
      stderr carries no `parsed input` line and no `Input` dump; at `-v` it
      carries `language resolved` and `bundle delivered`.
- [ ] Given a file argument with piped stdin, when the tool runs at `-vv`,
      then the Debug `stdin ignored` line is still logged.
- [ ] `go build ./...`, `go vet ./...`, `go test ./...` and the repo's lint
      pass; nothing outside the files this feature owns changed behavior.
- [ ] Follow-ups done: 002a's `spec.md` marks FR13 and its affected
      acceptance criteria superseded by 002b and closes its "temporary
      stdout behavior" note; `memory/known-gaps.md`'s 006a bare-stack row is
      removed or marked done and 006a's spec checkbox for it is checked;
      `CONVENTIONS.md`'s exit-code descriptions cover help (0), the Java
      message and invalid `--output` (2).

## Open questions

None. Two items are verified during planning, not open decisions:
- Nothing validates `Bundle.OS` against the three `contract.OS` constants
  (FR4 depends on it) -- checked in the plan's first task.
- 002a's `--lang`, `--format` and `-v` already have help strings (read
  from `parse.go`); the new flags need theirs.

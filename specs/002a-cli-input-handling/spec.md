# Spec: CLI Input Handling

**Status:** Spec'd
**Folder:** specs/002a-cli-input-handling

## Overview

Reads a stack trace from stdin or a file-path argument, parses CLI flags
(`--lang`, `--format`) for the `cmd/all`, `cmd/java`, and `cmd/typescript`
entrypoints, and packages the result into an internal struct for 002b to
consume later. This feature owns raw input acquisition and flag validation
only — no language detection, trace parsing, rendering, or clipboard writing
happens here.

## User stories

- As a developer, I want to pipe a stack trace via stdin so I can chain it
  from another command without saving a file first.
- As a developer, I want to pass a file path directly so I can point the
  tool at a saved trace log.
- As a developer, I want to specify `--lang` on `cmd/all` when
  auto-detection isn't available yet, so I can still exercise other
  pipeline stages during development.
- As a developer, I want `--format` to control whether I eventually get
  JSON or Markdown output, so I can choose based on whether I'm scripting
  or pasting into a chat.
- As a developer, I want clear, distinguishable error messages when my
  invocation is wrong, so I don't have to guess which of several possible
  usage errors occurred.

## Functional requirements

1. `cmd/all` registers a `--lang` flag with values `java` or `typescript`;
   omitted/empty means defer to future auto-detection (003). Any other
   value is a usage error.
2. `cmd/java` and `cmd/typescript` do not register `--lang`; passing it is
   a usage error (unknown flag).
3. All three binaries (`cmd/all`, `cmd/java`, `cmd/typescript`) register a
   `--format` flag with values `json` or `markdown`, defaulting to
   `markdown`. Any other value is a usage error.
4. Trace input is read from a positional file-path argument if given, else
   from stdin.
5. If both a file-path argument and piped stdin data are present, the file
   argument is used and stdin is ignored; this is logged at `Debug` level,
   not treated as an error.
6. If no file-path argument is given and stdin is not piped (i.e. it's an
   interactive terminal), the program exits immediately with a usage error
   — it never blocks waiting for interactive input.
7. If a file-path argument is given but the file cannot be opened (missing,
   permission denied, is a directory, etc.), this is a usage error.
8. If the read input (from either source) is empty or contains only
   whitespace, this is a usage error — treated the same regardless of
   source.
9. Reads are bounded: the underlying reader is wrapped in an
   `io.LimitReader` capped somewhat above the contract's 512KB truncation
   limit (e.g. 1MB) before being fully buffered, so a very large accidental
   input cannot cause unbounded memory growth before truncation applies.
10. After bounded reading, `contract.TruncateRawInput` (from
    `001-data-contract`) is applied to produce the final rune-safe,
    exactly-512KB-capped raw text and its `RawInputTruncated` flag.
11. Every usage-error exit path — including flag validation failures —
    logs a single `Error`-level message via `log/slog` to stderr that
    specifically identifies which condition triggered it (e.g.
    distinguishing "no input provided" from "file not found" from "input
    is empty" from "invalid --lang value"). No generic, unlabeled "usage
    error" message. All such paths exit with code `2`; no new exit codes
    are introduced beyond CONVENTIONS.md's existing table.
12. Flag parsing runs in `flag.ContinueOnError` mode, not the stdlib
    default `ExitOnError` — returned parse errors are logged via
    `slog.Error` (per requirement 11) before the program exits with code
    `2`, keeping message formatting consistent across every usage-error
    path rather than mixing in the flag package's own default output.
13. On a successful parse + read, this feature does **not** produce a real
    bundle — 007 (Markdown renderer), 008 (JSON renderer), and 009
    (clipboard) don't exist yet, and full wiring is 002b's job. Instead it
    logs an `Info`-level one-line summary (bytes read, lang hint, format)
    to stderr, and a separate `Debug`-level full dump of the parsed input
    struct to stderr, gated behind `-v`/`-vv` respectively.

## Non-functional requirements

- No interactive prompts required for core operation (constitution Article
  I) — the TTY-with-no-input case fails fast rather than blocking.
- This feature's "never blocks" guarantee (requirement 6) covers only the
  no-file-arg-and-stdin-not-piped case. It does not mean stdin reads never
  block: a piped source that produces less than the 1MB `LimitReader` cap
  and never closes will still block the read until EOF, same as any
  standard Unix text-filter tool (`cat`, `grep`, etc.) — not a gap, just
  the normal contract of reading from a pipe. No read timeout is applied.
- Stdout is reserved exclusively for the assembled bundle (constitution
  Article II) — this feature writes **nothing** to stdout under any
  circumstance, success or failure.
- Memory use during input reading is bounded regardless of actual input
  size (`LimitReader` before buffering).
- Default log verbosity remains `Warn` (per CONVENTIONS.md) — the
  Info/Debug stub output is opt-in via `-v`/`-vv`, not visible by default.

> **⚠️ Note — temporary stdout behavior:** Requirement 13's stub logging is
> scaffolding, not a permanent design decision. Real stdout bundle output
> is out of scope here and becomes 002b's responsibility once
> detect → parse → render → clipboard is actually wired up. When starting
> or completing 002b, revisit this feature's `main.go` stub output and
> replace/remove it — don't let it linger once the real pipeline exists.
> Tracked here rather than left implicit so it isn't forgotten.

## Out of scope

- Language auto-detection (003)
- Actual trace parsing / cause-chain extraction (005a, 006a)
- Actual bundle rendering to Markdown/JSON (007, 008)
- Clipboard writing (009)
- Repo-level git metadata (004)
- Dependency resolution (005b, 006b)
- Full pipeline wiring / real stdout bundle output (002b)

## Acceptance criteria

- [ ] Given a valid file-path arg pointing to a readable file with
      trace-shaped text, when `cmd/all` runs with no flags, then the raw
      text is read, bounded-read via `LimitReader`, truncated via
      `contract.TruncateRawInput` if needed, format defaults to
      `markdown`, lang hint is empty, and an Info summary + Debug dump are
      available via `-v`/`-vv`, with nothing written to stdout.
- [ ] Given both a file-path arg and piped stdin, when the binary runs,
      then the file's content is used, stdin is ignored, and a
      `Debug`-level log records that stdin was ignored.
- [ ] Given no file-path arg and stdin is not piped (TTY), when the binary
      runs, then it exits immediately with code 2 and a specific "no
      input" `Error`-level message, without blocking.
- [ ] Given a file-path arg pointing to a nonexistent file, when the binary
      runs, then it exits with code 2 and a specific "file not
      found"-style `Error`-level message including the path.
- [ ] Given input (from either source) that is empty or whitespace-only,
      when the binary runs, then it exits with code 2 and a specific
      "input is empty" `Error`-level message.
- [ ] Given `--lang=cobol` on `cmd/all`, when the binary runs, then it
      exits with code 2 and a specific `Error`-level message naming the
      invalid value and the accepted values.
- [ ] Given `--lang` passed to `cmd/java` or `cmd/typescript`, when the
      binary runs, then it exits with code 2 and a specific `Error`-level
      message noting the flag isn't valid for this binary.
- [ ] Given `--format=yaml` on any binary, when the binary runs, then it
      exits with code 2 and a specific `Error`-level message naming the
      invalid value and the accepted values.
- [ ] Given input larger than 512KB, when read from either stdin or a
      file, then the read itself does not buffer unbounded data, and the
      resulting raw text is exactly rune-safe-truncated to 512KB with
      `RawInputTruncated` set to `true`.
- [ ] Given a successful run with `-v`, then exactly one `Info`-level
      one-line summary is logged to stderr and nothing is logged to
      stdout.
- [ ] Given a successful run with `-vv`, then both the Info summary and a
      `Debug`-level full struct dump are logged to stderr, and nothing is
      logged to stdout.
- [ ] Given a successful run with no `-v`/`-vv`, then nothing is logged at
      all (default `Warn` level), consistent with CONVENTIONS.md's "quiet
      by default."

## Open questions

None remaining — all resolved during interrogation (see
`specs/002a-cli-input-handling/progress.md` for the session log once
implementation begins).

# AGENTS.md

This file is the single source of truth for any AI coding agent working in this
repository (Claude, Copilot, Cursor, Gemini/Antigravity, Codex, etc.). Read this
in full before making changes. If a chat instruction conflicts with this file,
the chat instruction wins for that session only — do not silently overwrite
this file to match a one-off instruction.

## Project overview

Stack-Trace → Context Bundle is a CLI tool for developers debugging with AI
assistance. It takes a pasted stack trace (Java or TypeScript/JavaScript, via
stdin or a file argument) and produces a single clipboard-ready bundle
containing: the error's cause chain, the relevant own-code snippets with git
blame for each frame, and resolved dependency versions for vendor frames —
so an AI assistant gets real, current context instead of a bare trace it has
to ask follow-up questions about.

- **Primary language / framework:** Go (stdlib-first; `log/slog` for logging,
  `os/exec` for all subprocess shell-outs to `git`/`mvn`/`gradle`/clipboard
  commands — no embedded native libraries, per constitution Article VII)
- **Package manager:** Go modules (`go mod`)

## Behavior — mandatory, applies to every agent/tool reading this file

- Before agreeing with any claim, proposal, or decision the user states,
  actively look for a reason it could be wrong, expensive, or based on a
  false assumption. State that reason out loud before responding to the
  substance — even if you end up agreeing anyway. Silence on this step is
  not allowed; if you find nothing, say "I checked for a problem here and
  didn't find one," don't just skip straight to agreement.
- Do not open a response by validating the user's idea ("great idea," "that
  makes sense," "you're right") before you've actually evaluated it. If
  validation is warranted, it comes after the check, not before it.
- Treat every factual claim the user makes as unverified until checked —
  against project files, or a web search if it's about something external
  (a library, an API, a technique). If you can't verify it, say plainly "I
  can't verify this, treating it as your claim" rather than proceeding as if
  it's confirmed.
- If the user restates a point you already pushed back on, don't fold just
  because they repeated it. Either give a new, specific reason you're
  changing your position, or hold the position and say so directly.
- Never use hedge phrases ("you might want to consider," "it could be worth
  thinking about") for something you actually believe is a problem. State it
  as a problem.
- If unsure whether something needs pushback, treat that uncertainty as a
  signal to look harder, not as permission to move on.
- If the user is vague or skips a decision, don't fill the gap silently. Ask,
  or mark it `[NEEDS CLARIFICATION]` in the relevant file.
- When multiple valid approaches exist, give real tradeoffs (cost, time,
  complexity, lock-in) instead of picking one and presenting it as obvious.
- Be direct and concise.

## Spec-driven workflow — mandatory

This project follows spec-driven development. Before writing or changing code:

1. Read `memory/constitution.md` — these rules are non-negotiable.
2. Read `specs/INDEX.md` for the full feature list, dependencies, and status
   — don't read every feature's full spec just to get oriented. Also check
   `memory/deferred-acceptance-criteria.md` for any criterion deferred to
   the feature you're about to start; fold it into that feature's own
   acceptance criteria during spec interrogation rather than treating it
   as separate extra work.
3. Find the relevant feature folder under `specs/NNN-feature-name/`.
4. Read `spec.md` (what/why) and `plan.md` (how) for that feature.
5. Work from `tasks.md`. Implement **one task at a time**, in order, unless told
   otherwise.
6. After completing a task, append a short entry to that feature's
   `progress.md`, then update that feature's row in `specs/INDEX.md`
   (status column) to reflect current state.
7. Before starting work on a feature, check its `progress.md` first. If it
   has prior entries, treat this as resuming: summarize where things left
   off and confirm with the user before continuing, don't restart from
   scratch.
8. If work on a feature reveals that `memory/constitution.md` itself needs
   to change, do not edit it silently. Propose the change to the user; if
   they agree, write a record to `memory/decisions/` explaining why, then
   update the constitution and AGENTS.md's stack section if relevant.

Never invent requirements. If something in `spec.md` is ambiguous or missing,
mark it `[NEEDS CLARIFICATION: your question]` and ask instead of guessing.

## Build & test commands

```
# one-time dev tool setup
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install github.com/evilmartians/lefthook@latest
lefthook install

# run (example: universal binary, against a trace file)
go run ./cmd/all path/to/trace.txt

# build a specific variant
go build -o bin/stack-trace-bundler ./cmd/all
go build -o bin/stack-trace-bundler-java ./cmd/java
go build -o bin/stack-trace-bundler-ts ./cmd/typescript

# test
go test ./...

# lint
golangci-lint run ./...

# format
gofumpt -l -w .

# full pre-commit gate, on demand (same checks Lefthook runs automatically)
lefthook run pre-commit
```

## Coding conventions

See `CONVENTIONS.md` for naming, error handling, logging, and testing
patterns, and `COMMIT_CONVENTIONS.md` for commit message format. Do not
duplicate that content here.

## Boundaries — things the agent must NOT do

- Never add a new third-party Go dependency without asking the user first
  and, if agreed, recording why in `memory/decisions/` — per constitution
  Article VII (shell out, don't embed) and Article VIII (simplicity).
- Never hand-edit `internal/contract/testdata/example_java.json` or
  `example_ts.json` — they are generated by
  `internal/contract/types_test.go`'s golden tests
  (`TestGolden_ExampleJava`/`TestGolden_ExampleTS`, run with `-update`).
  Change the struct or the builder functions, regenerate the fixtures,
  don't edit them directly.
- Never add a JSON Schema file or any second copy of the bundle shape
  anywhere in the repo — `internal/contract` is the only source of truth
  (constitution Article IV).
- Never let a package under `internal/parser/<lang>/` import from another
  language's parser package. Cross-language sharing only happens through
  `internal/parser/registry.go` (the `LanguageParser` interface) or
  genuinely language-agnostic shared code (e.g. own-code extraction).
- Never write to stdout except the final assembled bundle. Logs and
  diagnostics go to stderr via `log/slog`, always (constitution Article II).
- Never add environment variables, command-line arguments, memory stats, or
  network state to the bundle contract — out of scope per the constitution,
  not a v1 gap to quietly fill in.
- Never bypass the pre-commit gate (`git commit --no-verify`) when operating
  as an AI agent in this repo; `.claude/settings.json` denies this
  explicitly, but the rule holds regardless of tool.
- Never edit `memory/constitution.md` without following workflow step 8
  above — propose first, record the decision, then update.

## Testing philosophy

Tests are required, not test-first (constitution Article III): implementation
and its tests land in the same change, not tests-before-code. Nothing is
considered done until the pre-commit gate passes — `gofumpt` → `golangci-lint`
→ `go build` → `go test`, run automatically via Lefthook on commit, or
manually via `lefthook run pre-commit`. Test files mirror the `internal/`
package structure (e.g. `internal/parser/java/parser_test.go` next to
`parser.go`), one test file per source file as the default, splitting further
only if a single test file becomes unwieldy.

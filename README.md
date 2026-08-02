# Stack-Trace → Context Bundle

A CLI tool for developers debugging with AI assistance. Paste a stack trace
in, get back a single clipboard-ready bundle containing the error's cause
chain, the relevant own-code snippets with git blame, and resolved dependency
versions for any vendor frames — so the AI you paste it into has real,
current context instead of a bare trace it has to ask follow-up questions
about.

## Status

Early development — no code yet. See `specs/INDEX.md` for feature status;
everything is currently `idea` stage. `memory/constitution.md` and
`AGENTS.md` are locked; implementation hasn't started.

## Stack

- **Language:** Go
- **Package manager:** Go modules (`go mod`)
- Supports Java and TypeScript/JavaScript stack traces in v1, auto-detected
  from the pasted trace. Additional languages are added as independent
  parser modules — see `AGENTS.md` for how.

## Getting started

```
# one-time dev tool setup
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install github.com/evilmartians/lefthook@latest
lefthook install

# run (example: universal binary, against a trace file)
go run ./cmd/all path/to/trace.txt

# build
go build -o bin/stack-trace-bundler ./cmd/all

# test
go test ./...
```

See `AGENTS.md` for the full command list (lint, format, per-variant builds).

## Project structure

```
memory/           project principles (constitution.md), decision records
specs/            feature specs, plans, tasks, progress (spec-driven dev)
cmd/              CLI entrypoints — all/, java/, typescript/ build variants
internal/         cli, contract, codecontext, parser/{java,typescript}, render, clipboard
CONVENTIONS.md    coding style for this project
COMMIT_CONVENTIONS.md   commit message format
```

## Contributing / working on this repo

This project follows spec-driven development — see `AGENTS.md` for the full
workflow if you're picking up a feature or opening a new one. Commits go
through a pre-commit gate (format, lint, build, test) and a commit message
check, both via Lefthook — see `COMMIT_CONVENTIONS.md` for the required
message format.

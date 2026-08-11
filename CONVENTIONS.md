# Coding Conventions

Idiomatic patterns for this project's stack (Go). This file is referenced
from `AGENTS.md`, not duplicated there.

## Naming

- Packages: short, lowercase, no underscores or camelCase (`parser`,
  `contract`, `render`, `clipboard`, `codecontext`) — matches the
  `internal/` layout in AGENTS.md.
- Exported identifiers: `PascalCase`, and exported only when another package
  genuinely needs it. Default to unexported.
- Interfaces: named for the capability, not prefixed with `I` — `LanguageParser`,
  not `ILanguageParser`.
- Files: lowercase, underscore only when it aids clarity (`parser.go`,
  `blame.go`). Test files use the standard `_test.go` suffix, next to the
  file they test.
- Branches: `<type>/<short-description>`, using the same types as
  `COMMIT_CONVENTIONS.md` — e.g. `feat/java-caused-by-chain`.
- Commit messages: see `COMMIT_CONVENTIONS.md`, not restated here.

## Error handling

- Always wrap with context: `fmt.Errorf("reading %s: %w", path, err)` — never
  a bare re-throw.
- No panics for expected failure paths (missing file, unparseable trace,
  `mvn`/`gradle` not on `PATH`). Panics are reserved for genuine
  programmer-error invariants, not runtime conditions.
- `errcheck` (via `golangci-lint`) must pass — no discarded error return
  value without an explicit `_ =` and a comment explaining why it's safe.
- Per constitution Article VI: when something can't be determined, encode
  that in the contract's own fields (`codeContexts[].status`/`note`, an
  unresolved dependency version left explicitly empty with a note) — never
  return a zero value that could be mistaken for a real one.
- CLI exit codes: `0` success · `1` unexpected/internal error · `2` usage
  error (bad flags/args) · `3` input could not be parsed as any known trace
  format · `4` source language ambiguous or undetected.

## Logging

- `log/slog`, structured, **stderr only** — never stdout (constitution
  Article II). Stdout carries the bundle and nothing else.
- Levels: `Debug` (verbose internal steps, opt-in), `Info` (milestones —
  language detected, N dependencies resolved), `Warn` (degraded but
  continuing — file not found, fell back to a note), `Error` (operation
  failed, process will exit non-zero).
- Default level is `Warn` — a CLI tool should be quiet by default. `-v`
  raises to `Info`, `-vv` to `Debug`.
- Never log environment variables, full file contents beyond what's already
  in the bundle, or anything that could be a secret — same reasoning as the
  constitution's ban on environment data in the bundle itself.

## API / response shape

Not applicable in the HTTP sense — this is a CLI, not a service. The nearest
equivalent, the bundle's JSON/Markdown output, is governed entirely by
`internal/contract` per constitution Article IV. Not restated here.

## Testing

- Test files live beside the code they test (`_test.go`), one per source
  file by default, matching `AGENTS.md`'s testing philosophy.
- Parser logic (`internal/parser/*`): table-driven tests — one table entry
  per real-world trace shape/edge case, not one test function per case.
- Renderers (`internal/render/*`): golden-file tests — compare full rendered
  output against a checked-in fixture; a diff is easier to review than a
  pile of field-by-field assertions.
- Anything that shells out (`git`, `mvn`, `gradle`, clipboard commands) must
  be tested behind a small interface with a hand-written fake — `go test
  ./...` must never require those binaries to be installed. Tests that
  genuinely exercise the real subprocess live behind a separate build tag
  and run only in CI, not on every local test run.
- No mocking framework/library — hand-written fakes implementing the same
  interface, consistent with the project's general "fewer dependencies"
  stance (constitution Articles VII/VIII).

## Formatting & linting

- `gofumpt` for formatting (stricter superset of `gofmt`).
- `golangci-lint` for linting — `govet`, `staticcheck`, `errcheck`, `revive`,
  `gocyclo`, `unused`, config in `.golangci.yml` at repo root.
- Enforced pre-commit via Lefthook (`format` → `lint` → `build` → `test`,
  sequential — see `lefthook.yml`), and the identical set re-run in CI so a
  skipped or bypassed local hook can't slip a violation through.
- Never put a file-path label comment (e.g. `// internal/cli/read.go`)
  directly above a `package X` clause, in any file — `revive`'s
  `package-comments` check treats any comment immediately preceding the
  package clause as an attempted package doc comment and fails it unless
  it starts with `"Package X ..."`. Only the one file that carries the
  real `// Package X ...` doc comment (per Go convention, usually the
  file most central to the package, e.g. `input.go` for `internal/cli`)
  should have anything directly above `package X`. Every other file in
  the package starts with a bare `package X` line, no comment, no
  exceptions — including `_test.go` files.

## File / folder layout

```
internal/
├── cli/          orchestration: arg parsing, wiring detect → parse → render → clipboard
├── contract/     canonical Go structs (constitution Article IV) + generated test fixture
├── codecontext/  shared, language-agnostic: file read, snippet windowing, git blame
├── parser/
│   ├── registry.go     LanguageParser interface + detection registry
│   ├── java/
│   └── typescript/
├── render/       markdown.go, json.go
└── clipboard/    OS-appropriate subprocess write
```

- New target language → new package under `internal/parser/<lang>/`
  implementing `LanguageParser`, a new `cmd/<lang>/main.go`, and one line
  added to `cmd/all/main.go`. Nothing else moves.
- Logic that doesn't depend on source language never lives inside a
  `parser/<lang>/` package — it belongs in `codecontext` (constitution
  Article V).

# Commit Message Conventions

This project uses Conventional Commits, enforced by a Lefthook `commit-msg`
hook (plain regex, see `lefthook.yml` — no external linting tool for this,
consistent with the project's "shell out over embed / minimal dependencies"
stance).

## Format

```
<type>(<scope>): <description>
```

`(<scope>)` is optional. `<description>` is required and must be non-empty.

## Allowed types

| Type | Use for |
|------|---------|
| `feat` | a new feature or capability |
| `fix` | a bug fix |
| `chore` | tooling, dependency bumps, repo maintenance — no source behavior change |
| `docs` | documentation only (`AGENTS.md`, `CONVENTIONS.md`, `README.md`, specs) |
| `refactor` | code change that is not a fix or a feature |
| `test` | adding or correcting tests, no production code change |
| `build` | build system, `go.mod`, CI config |
| `ci` | CI pipeline changes specifically |
| `perf` | a performance improvement |

## Examples

```
feat(java-parser): add caused-by chain parsing
fix(codecontext): handle stale file when line count no longer matches trace
docs: update AGENTS.md build commands
chore: bump golangci-lint version
```

## What's enforced vs. not

Enforced by the `commit-msg` hook: the message matches
`<type>(<scope>): <description>` with one of the allowed types.

Not enforced (by choice, to keep contributor friction low — see constitution
Article VIII): scope is not mandatory, and there's no hook checking the type
against what actually changed (e.g. nothing stops a `docs:` commit that also
touches source code). If this becomes a real problem in practice, revisit —
don't add enforcement speculatively.

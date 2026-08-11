
# 0002 — Adopt spf13/pflag for CLI flag parsing

**Status:** Accepted
**Date:** 2026-08-11
**Depends on:** none

## Context

`internal/cli.ParseAll`/`ParseFixedLang` (feature 002a) used the stdlib
`flag` package in `ContinueOnError` mode. Discovered during T007's actual
manual run-through against the real built `cmd/all` binary (not caught by
any unit test up to that point): `flag.Parse` stops looking for flags the
moment it hits the first non-flag argument. `stba /tmp/valid.txt -vv` —
the exact natural way most people type a command, and exactly what T007's
own acceptance criteria exercises — silently dropped `-vv` entirely,
treating it as an ignored extra positional argument rather than a flag.
Every flag (`--lang`, `--format`, `-v`, `-vv`) was affected, not just
verbosity; this was present from T004 onward and went undetected because
every unit test across T004/T005/T006b/T006c happened to append the file
path *after* the flags in the constructed `args` slice, never testing the
reverse order.

This is documented stdlib `flag` behavior, not a bug in project code —
but it fails constitution Article I's "scriptable" expectation and is a
real, likely-to-be-hit usability problem for a CLI meant to be piped and
typed quickly.

## Decision

Add `github.com/spf13/pflag` as a dependency and replace stdlib `flag`
with `pflag.FlagSet` in `ParseAll`/`ParseFixedLang`. `pflag` permutes
arguments by default — flags and the positional trace-file argument can
appear in any order — and correctly honors a `--` terminator, both
confirmed against its documentation before committing to this.

`-v`/`-vv` change from two independently-registered stdlib bool flags to
a single `pflag.CountVarP` shorthand (`-v`). This isn't optional cosmetic
change: `pflag` shorthands are strictly single-dash-single-character —
`-vv` as a literal two-character single-dash flag name has no direct
equivalent in `pflag`'s model the way it did under stdlib `flag`.
`CountVarP` is `pflag`'s idiomatic mechanism for exactly this pattern —
POSIX-style clustering, where `-v` counts 1, `-vv` counts 2, `-vvv`
counts 3, and so on. This still satisfies spec.md's literal `-v`/`-vv`
wording, and requires no change to the already-built `LogLevel` mapping,
which already treats any count `>= 2` as `Debug`. `verbosityFromFlags`
(added in T006b to combine two bools into one count) becomes dead code
and is removed.

## Alternatives considered

- **Keep stdlib `flag`; document that flags must precede the positional
  argument.** Rejected — this is a real footgun for exactly the use case
  this tool exists for (fast, scriptable, piped invocation), and a clean
  fix exists. Documenting around a fixable limitation isn't the standard
  the rest of this feature has held to (see T006b/T006c, both of which
  fixed rather than documented around real gaps).
- **Hand-roll an `argv` permutation pass ourselves**, reordering flags to
  the front before calling stdlib `flag.Parse`. Rejected — this is the
  third time this exact category of fix has come up in this feature
  (previously: the rejected `-v`/`-vv` extraction pre-pass in T006b, and
  the rejected "peek before configuring log level" idea in T006c).
  Correctly reordering `argv` while preserving flag/value pairing
  (`--lang java` two-token form vs. `--lang=java` one-token form) and
  respecting `--` terminators is exactly re-implementing what a real flag
  library already does correctly — the highest-risk option of the three
  considered, not a novel one.
- **A different/heavier Go CLI library** (e.g. `cobra`, `urfave/cli`,
  `kingpin`). `pflag` specifically chosen over these: it's shaped as a
  near-drop-in replacement for the exact stdlib `flag` surface already in
  use (`NewFlagSet`, `*Var` family, `ContinueOnError`, `SetOutput`,
  `Parse`, `Args`/`Arg`), so the change stays contained to `parse.go`
  rather than restructuring the CLI entrypoints around a full framework —
  consistent with constitution Article VIII (simplicity: fix the actual
  problem, don't take on more than it needs). It's also the most widely
  used and audited flag library in the Go ecosystem (used by `kubectl`,
  Docker, Helm, and `cobra` itself internally), a reasonable trust signal
  for this project's first non-stdlib dependency.

## Consequences

- `github.com/spf13/pflag` added to `go.mod`/`go.sum` — the project's
  first third-party Go dependency. `AGENTS.md`'s "Go (stdlib-first; ...)"
  line is not contradicted by this (it says "stdlib-first," not
  "stdlib-only," and this ADR is the recording-why step that same section
  requires before any such addition) — no `AGENTS.md` text change needed.
- `-lang=java` (single-dash form of a multi-character flag name) stops
  working; only `--lang=java` does, going forward. Not a regression
  against anything tested or documented — spec.md's own acceptance
  criteria always show the double-dash form, and no test in
  `parse_test.go` ever exercised the single-dash form. Arguably brings
  the tool in line with conventional GNU long/short flag separation
  rather than stdlib `flag`'s more permissive (and, per this ADR, buggy
  in a different way) equivalence.
- `internal/cli/parse.go`'s `ParseAll`/`ParseFixedLang` need
  re-implementing against `pflag`'s API (tracked as task T007a in
  `specs/002a-cli-input-handling/tasks.md`); `parse_test.go` needs
  re-verification against the new library's actual behavior, plus a new
  test case covering a flag placed *after* the positional argument — the
  exact case that was silently broken and went undetected until now.
- `-h`/`--help` behavior under `pflag` is an open question carried over
  unchanged from the original stdlib-`flag` design (parked, non-blocking,
  since T004's `progress.md` entry) — not resolved or changed by this
  decision either way.

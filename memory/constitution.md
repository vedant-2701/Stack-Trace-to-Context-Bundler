# Project Constitution

Non-negotiable principles for this project. Every spec, plan, and task must be
consistent with this document. Changing this file is a deliberate, rare act —
not something an agent does mid-task to make a spec fit.

## Articles

### I. CLI-first, text in, text out
This tool is a CLI, not a library or a service. It reads a stack trace from
stdin or a file-path argument and produces a clipboard-ready bundle. Every
feature must work headless and scriptable — no interactive prompts required
for core operation.

### II. Stdout is reserved for output; stderr is for everything else
The assembled bundle is the only thing this tool ever writes to stdout. All
logs, warnings, and diagnostics — structured via `log/slog` — go to stderr,
without exception. A stray log line on stdout silently corrupts every
pipeline built on top of this tool; that is treated as a correctness bug,
not a style nit.

### III. Tests are required, not test-first
Implementation and its tests land together — tests are not required to exist
before the code they cover. But nothing reaches `main` untested: the
pre-commit gate (format → lint → build → test, run via Lefthook) must pass
before a commit is accepted. Bypassing the gate (e.g. `git commit
--no-verify`) is not permitted for AI agents operating in this repo.

### IV. One canonical contract, never duplicated
The bundle's shape is defined exactly once, as Go structs in
`internal/contract`. There is no hand-maintained JSON Schema and no second
copy of the shape anywhere else in the repo. A generated, tested fixture
(`internal/contract/testdata/example.json`) is the only derived artifact
permitted, and it is produced from the struct — never hand-written in
parallel.

### V. Shared logic lives once
Any behavior that does not depend on the source language — reading a file,
windowing a snippet around a line, running `git blame` — is implemented
exactly once and used by every language parser. It is never copied into a
per-language package "just this once."

### VI. Never present a guess as fact
When the tool cannot verify something — a file that is missing or stale
relative to the trace, a trace it cannot confidently attribute to a
language, a dependency version it cannot resolve — the output must say so
explicitly. The tool must never guess silently and hand over unverified
information as if it were confirmed.

### VII. Shell out, don't embed
Git, Maven/Gradle, and the OS clipboard are invoked as subprocesses, never
as embedded native libraries or CGO bindings. This keeps the binary small,
keeps cross-compilation simple, and avoids taking on a dependency's release
cycle and platform quirks as our own.

### VIII. Simplicity over speculative capability
Build what the current spec needs, not what a future language or feature
might need. Capabilities like on-demand binary composition or multi-manifest
dependency resolution are deferred until real usage shows they're needed,
not built preemptively because they might matter later.

### IX. Java dependency resolution is offline-first and time-boxed
The only place `mvn`/`gradle` are invoked (Article VII) is resolving
concrete versions for `dependency`-bucket frames in the Java parser. That
call is always offline-only (`-o` / `--offline`) and wrapped in a hard
timeout. Offline mode turns a missing local artifact into an immediate,
catchable error instead of a network hang; the timeout exists separately,
to bound JVM/daemon startup and Gradle's configuration-phase cost, both of
which are real regardless of network or cache state. On timeout or an
offline resolution failure, the tool does not retry online and does not
guess — per Article VI, it records that one dependency's version as
unresolved, with a note, and keeps going. Automatic online fallback is out
of scope for v1 (Article VIII). See
`memory/decisions/0001-offline-first-time-boxed-dependency-version-resolution.md`
for the full reasoning. TypeScript/JavaScript dependency resolution is
unaffected — it parses `package.json` and the lockfile directly and never
shells out to a subprocess.

## Stack governance

The actual stack (language, framework, database, hosting) is recorded once in
AGENTS.md, not duplicated here. The rule this constitution enforces: once
AGENTS.md's stack section is written, it is fixed for the project's lifetime.
Changing it is not a per-feature decision — it requires a record in
memory/decisions/ explaining why, and an update to AGENTS.md.

## Explicitly out of scope

<!-- Things this project will never do, to stop an agent from "helpfully"
     adding them. -->

- **Branching cause chains** (`AggregateError`, Java `Suppressed`) — v1
  supports linear cause chains only. Branching is deferred to v2 and must be
  explicitly detected and marked, never silently mishandled as linear.
- **Literal AST-based snippet extraction** — v1 uses line-window text
  extraction only. Full syntax-tree parsing per language is a v2+ idea, not
  a v1 requirement.
- **Output formats beyond JSON and Markdown** — no XML, no TOON, until a
  real consumer is identified. Not built speculatively.
- **Environment data beyond the OS name** — no environment variables,
  command-line arguments, memory stats, or network status in the bundle.
  This tool's output is pasted into third-party AI chats; that is not a safe
  place to be wrong about "filtered secrets."
- **Multi-manifest / multi-module dependency resolution** — v1 assumes a
  single manifest (`package.json` / `pom.xml` / `build.gradle`) at the repo
  root. Monorepo/workspace-aware resolution is a known future gap, not a
  silent limitation.
- **Compiling a custom binary combination on demand at download time** —
  ship a fixed set of build variants (all-languages, per-language) via CI.
  No build-on-demand infrastructure.

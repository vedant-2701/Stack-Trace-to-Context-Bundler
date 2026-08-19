# Feature Index

One line per feature. This file is the map — read it before starting work on
any single feature so you don't need to load every feature's full spec into
context. Update the Status column as work progresses; update Depends on if a dependency is discovered mid-project.

Status values: `idea` (listed, not discussed) → `specifying` (spec.md in
progress) → `spec'd` (spec.md done, plan.md not started) → `planned` (plan.md
+ tasks.md done) → `in-progress` (implementation underway) → `done`

<!--
When adding a new feature discovered mid-project, append a row here first,
then open a new chat to spec it — don't spec it inline in whatever chat
you're already in, or that chat's context balloons.
-->
| ID | Feature | One-line description | Depends on | Status |
|----|---------|----------------------|------------| ------ |
| 001 | Data contract | Canonical bundle shape — `internal/contract` structs, generated+tested JSON fixture | — | done |
| 002a | CLI input handling | Entrypoints, stdin/file-arg reading, flags (`--lang`, output format) | 001 | done |
| 002b | Pipeline wiring | Wires 003a/003b–009 together into the full detect → parse → render → clipboard flow | 002a, 003a, 003b, 004, 005a, 006a, 007, 008, 009 | idea |
| 003a | Language parser interface | Defines the shared `LanguageParser` interface (method signatures + doc-comment contract), validated by hand-tracing real Java and real TS/JS example traces through it as pseudocode in `plan.md` -- scoped to Java + TS/JS only, generalizing to other languages (Go, Rust, ...) is an explicit non-goal until a real parser for one exists to test against (Article VIII). Does not decide bucketing rules, chain-elision rules, or runtime-detection heuristics -- those stay 005a/006a's own scope, same pattern 001 already used for `Bucket`. | 001 | done |
| 003b | Language auto-detection registry | Tries each registered parser's `Detect()` against the raw trace; defines "no match"/ambiguous behavior. Needs real parsers to test detection heuristics against, not just the interface shape. | 001, 003a, 005a, 006a | idea |
| 004 | Own-code context extraction | Shared: read file at line, snippet window, `git blame -L`, repo-level git metadata (current commit, branch, uncommitted changes), stale/not-found handling — language-agnostic | 001 (+ requires a `001` contract patch: `GitMetadata` → pointer, schemaVersion MAJOR bump) | done |
| 005a | Java parser | `Caused by:` chain parsing, frame bucketing (own/dependency/runtime), runtime + runtime-version detection (JVM) | 001, 003a, 004 | idea |
| 005b | Java dependency resolution | `mvn`/`gradle` shell-out to resolve dependency versions for `dependency`-bucket frames | 001, 005a | idea |
| 006a | TypeScript/JS parser | `.cause` chain parsing (incl. Node's frame-elision, `"... N lines matching cause stack trace ..."`), frame bucketing (own/`node_modules`/`node:internal`), runtime + runtime-version detection (Node.js only for v1 -- Bun/Deno deferred, see `memory/known-gaps.md`) | 001, 003a, 004 | planned |
| 006b | TS/JS dependency resolution | `package.json`+lockfile resolution for `dependency`-bucket frames | 001, 006a | idea |
| 007 | Markdown renderer | Bundle → clipboard-ready Markdown | 001 | idea |
| 008 | JSON renderer | Bundle → raw contract JSON | 001 | idea |
| 009 | Clipboard integration | OS-appropriate clipboard write via subprocess (`pbcopy` on macOS; `wl-copy` → `xclip` fallback chain on Linux, clear stderr error if neither is available; `clip.exe` on Windows) | — | idea |
| 010 | Distribution & release packaging | Build matrix (`all`/`java`/`typescript` × 5 platform targets), how users actually get the binary | 002b; at least one of 005a/006a working end-to-end | idea |
| 011 | Configurable snippet context window | Expose 004's own-code snippet line-count (fixed at ±5/side in 004) as a `--context-lines`-style CLI flag | 004, 002a | idea |
| 012 | Browser trace capture | Launch/attach to a browser and capture an uncaught JS/TS error's stack trace directly, without manual copy-paste; likely needs a new browser-automation dependency and its own Article VII/VIII ADR before that's added -- not a trivial add | 006a | idea |

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
| 002b | Pipeline wiring | Wires 003–009 together into the full detect → parse → render → clipboard flow | 002a, 003, 004, 005a, 006a, 007, 008, 009 | idea |
| 003 | Language auto-detection | Tries each registered parser's `Detect()` against the raw trace; defines "no match"/ambiguous behavior | 001; needs ≥1 real parser (005a or 006a) to test against | idea |
| 004 | Own-code context extraction | Shared: read file at line, snippet window, `git blame -L`, repo-level git metadata (current commit, branch, uncommitted changes), stale/not-found handling — language-agnostic | 001 | idea |
| 005a | Java parser | `Caused by:` chain parsing, frame bucketing (own/dependency/runtime), runtime + runtime-version detection (JVM) | 001, 004 | idea |
| 005b | Java dependency resolution | `mvn`/`gradle` shell-out to resolve dependency versions for `dependency`-bucket frames | 001, 005a | idea |
| 006a | TypeScript/JS parser | `.cause` chain parsing, frame bucketing (own/`node_modules`/`node:internal`), runtime + runtime-version detection (node/bun/deno) | 001, 004 | idea |
| 006b | TS/JS dependency resolution | `package.json`+lockfile resolution for `dependency`-bucket frames | 001, 006a | idea |
| 007 | Markdown renderer | Bundle → clipboard-ready Markdown | 001 | idea |
| 008 | JSON renderer | Bundle → raw contract JSON | 001 | idea |
| 009 | Clipboard integration | OS-appropriate clipboard write via subprocess (`pbcopy` on macOS; `wl-copy` → `xclip` fallback chain on Linux, clear stderr error if neither is available; `clip.exe` on Windows) | — | idea |
| 010 | Distribution & release packaging | Build matrix (`all`/`java`/`typescript` × 5 platform targets), how users actually get the binary | 002b; at least one of 005a/006a working end-to-end | idea |

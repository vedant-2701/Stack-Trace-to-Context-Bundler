# Spec-Driven Development (SDD) — Project Template

> First time opening this template? Run `scripts/init-repo/run.sh` (or
> `.\scripts\init-repo\run.ps1` on Windows) before anything else. It moves
> this file to `docs/SDD-GUIDE.md` and promotes `SAMPLE-README.md` to
> `README.md`, so the repo's front page becomes your actual project instead
> of this guide.

This is a tool-agnostic spec-driven development scaffold. It works the same whether
you drive it with Claude Desktop (filesystem MCP), Claude Code, GitHub Copilot,
Cursor, Antigravity/Gemini CLI, or any other coding agent — because the actual
instructions live in one file (`AGENTS.md`) and every tool either reads it directly
or is pointed at it through a one-line bridge file.

## Folder structure

```
.
├── README.md                     ← this file (moves to docs/SDD-GUIDE.md after init)
├── SAMPLE-README.md              ← becomes the real README.md after init
├── AGENTS.md                     ← SOURCE OF TRUTH: instructions every agent reads
├── CLAUDE.md                     ← bridge file for Claude Code (imports AGENTS.md)
├── GEMINI.md                     ← bridge file for Gemini CLI / Antigravity
├── CONVENTIONS.md                ← stack-specific coding conventions
├── desktop-project-instruction.txt ← paste into Claude Project → Custom Instructions
├── memory/
│   ├── constitution.md           ← non-negotiable project principles (write once)
│   └── decisions/                ← architecture decision records (ADRs)
│       ├── INDEX.md              ← decision map: title, dependencies, status, date
│       └── _template.md          ← template used by scripts/new-decision
├── prompts/                      ← copy-paste prompts for each workflow phase
│   ├── kick-off.md               ← start here for a brand-new project
│   ├── review-setup.md           ← audit files for consistency, run after kickoff
│   ├── review-plan.md            ← audit the plan's substance, run after review-setup
│   ├── new-feature.md            ← start a new feature's spec/plan/tasks
│   └── implement-task.md         ← implement one task from an existing feature
├── scripts/                      ← automation (Bash & PowerShell)
│   ├── init-repo/                ← one-time post-clone bootstrap (see above)
│   ├── new-feature/               ← creates a feature folder + registers it in specs/INDEX.md
│   ├── update-status/             ← updates a feature's status in specs/INDEX.md
│   └── new-decision/              ← creates a new ADR + registers it in memory/decisions/INDEX.md
└── specs/
    ├── INDEX.md                   ← feature map: description, dependencies, status
    ├── _templates/                ← master templates used by scripts/new-feature
    └── 001-example-feature/       ← one numbered folder per feature
        ├── spec.md                ← WHAT and WHY
        ├── plan.md                ← HOW (architecture, stack, data model)
        ├── tasks.md                ← atomic, ordered implementation steps
        └── progress.md             ← running log of what's actually been done
```

Copilot and Cursor read `AGENTS.md` natively — no bridge file needed for them.
Copilot also happens to read `CLAUDE.md`/`GEMINI.md` if present, but that's a bonus,
not a requirement.

## Why numbered feature folders (`001-`, `002-`...) and decision records (`0001-`, `0002-`...)

Each feature/module gets its own folder under `specs/`, scoped and diffable, so
old specs stay a historical record instead of one giant file being overwritten.
Each significant architectural choice gets its own record under
`memory/decisions/` for the same reason — but decisions are append-only:
once accepted, a record is never edited to change its meaning. If a decision
changes, a new record supersedes the old one; the old one stays as history.

`specs/INDEX.md` and `memory/decisions/INDEX.md` exist so an agent (or you)
can get oriented on the whole project or its full decision history in one
cheap read, instead of loading every feature's or every decision's full file.

## The workflow (5 phases)

### Phase 0 — Constitution (once per project)
Use `prompts/kick-off.md` in a new chat. It walks through interrogating the
idea, then writes `memory/constitution.md`, `AGENTS.md`, `CONVENTIONS.md`, and
`specs/INDEX.md` once the project's shape and stack are actually settled —
not guessed from the first message.

### Phase 0.5a — Review setup (once, after kickoff)
Use `prompts/review-setup.md` in a new chat. Audits the files themselves for
contradictions, duplication, dangling references, and leftover scaffolding —
a documentation-hygiene pass, not a judgment on the decisions made.

### Phase 0.5b — Review plan (once, after review-setup)
Use `prompts/review-plan.md` in a new chat. A technical review of the
substance: whether the feature breakdown, dependencies, stack choice, and
constitution actually hold up — separate from whether the files are
internally consistent.

### Phase 1 — Specify
Run `./scripts/new-feature/run.sh <feature-name> [description] [depends-on]`
(or `.\scripts\new-feature\run.ps1` on Windows) to create a new numbered
feature folder under `specs/` with all 4 template files and register it in
`specs/INDEX.md`. Then use `prompts/new-feature.md` in a new chat to fill in
`spec.md` — what you're building and why, zero tech detail. Mark anything
unclear as `[NEEDS CLARIFICATION: ...]`.

### Phase 2 — Clarify
Resolve every `[NEEDS CLARIFICATION]` marker before moving on. This is the step
people skip and regret — an underspecified spec produces an agent that guesses.

### Phase 3 — Plan
`prompts/new-feature.md` continues into this once spec.md is clean: architecture,
stack + exact versions, data model, file/module layout, testing strategy.
Must not contradict the constitution. If this phase surfaces a significant,
hard-to-reverse call (not specific to just this feature), run
`./scripts/new-decision/run.sh` to record it in `memory/decisions/`.

### Phase 4 — Tasks
Also part of `prompts/new-feature.md`'s pass: break the plan into small,
ordered, independently-verifiable tasks in `tasks.md`.

### Phase 5 — Implement
Use `prompts/implement-task.md` in a chat, one task at a time. After each one,
`progress.md` gets a log entry and `specs/INDEX.md` gets its status updated —
do this via `./scripts/update-status/run.sh` (or `.\scripts\update-status\run.ps1`)
if you're updating status outside of an agent chat.

Keep implementation to **one task per turn** — this is the single biggest lever
for keeping an agent's output aligned with the spec over a long session.

## Automation scripts

See [scripts/README.md](scripts/README.md) for full usage and interactive options.

- **`scripts/init-repo/`** — one-time post-clone setup (see top of this file)
- **`scripts/new-feature/`** — creates `specs/NNN-feature-name/` and registers it in `specs/INDEX.md`
- **`scripts/update-status/`** — interactive search/menu to update a feature's status
- **`scripts/new-decision/`** — creates a new ADR under `memory/decisions/` and registers it in `memory/decisions/INDEX.md`

## Adding a new tool later

If you later pick up Cursor, Windsurf, or Copilot, you do nothing extra — they
read `AGENTS.md` directly. If you pick up a tool that wants its own file name,
just add a one-line bridge file like `CLAUDE.md`/`GEMINI.md` that says
"See AGENTS.md — that file is authoritative" plus any tool-specific quirks.
# Spec: TypeScript/JS parser

**Status:** Clarifying
**Folder:** specs/006a-ts-js-parser

## Overview

<!-- 2-4 sentences: what is this feature and why does it matter to the user.
     No implementation details here — that belongs in plan.md. -->

## User stories

- As a <user type>, I want to <do something>, so that <benefit>.

## Functional requirements

1. The system must ...
2. The system must ...

## Non-functional requirements

<!-- performance, security, accessibility, limits, etc. -->

## Out of scope

<!-- explicitly excluded from this feature, to prevent scope creep -->

## Acceptance criteria

- [ ] Given <context>, when <action>, then <expected result>
- [ ] Given <context>, when <action>, then <expected result>

## Open questions

- [RESOLVED] Q1 -- runtime/environment scope: Node.js, Bun, Deno only, and only their CLI/server crash-style output (uncaught-exception dump or a logged `err.stack`) -- captured from a file or paste. No browser-originated traces (Chrome/Firefox/Safari DevTools) for v1. Browser trace capture (including a possible future "launch a browser and capture live" mode) is split out as a new feature: see `specs/INDEX.md` #012.
- [BLOCKED] Q2 -- public API shape / relationship to the `LanguageParser` interface: rather than 006a guessing at an interface that didn't exist yet, `specs/INDEX.md` #003 was reshaped into two features: **#003a** (the `LanguageParser` interface itself -- shape + pseudocode contract, scoped to Java + TS/JS only per Article VIII) and **#003b** (the auto-detection registry, which still needs real parsers to test against, so it depends on 005a/006a same as before). This changes 006a's `Depends on` in `specs/INDEX.md` to `001, 003a, 004`. **006a interrogation is paused here pending 003a being spec'd/planned in a separate chat**, per `AGENTS.md`'s own rule for mid-project feature discovery.

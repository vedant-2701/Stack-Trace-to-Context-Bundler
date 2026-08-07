# Deferred acceptance criteria

Tracks acceptance criteria written in a completed feature's `spec.md` that
describe behavior owned by a *different*, not-yet-built feature. Lives on
disk (not just Claude's cross-session memory) specifically so it survives
an account change or a memory reset -- this file, not memory, is the
source of truth for these items.

**When completing any feature listed below as an owner:** also go back and
check the corresponding box in the source feature's `spec.md`, then remove
that row here (or mark it done, whichever keeps this file smallest).

**When starting any feature listed below as an owner:** check whether it
owes a deferred criterion here, and fold it into that feature's own
acceptance criteria during spec interrogation -- these aren't extra work,
they're criteria the feature already needs to satisfy for its own reason,
just written down in a sibling feature's file first.

| Source feature | Criterion | Owner | Status |
|---|---|---|---|
| 001-data-contract | Java `Caused by:` chain parses into `chain[]` with correct `elidedFrameCount` | 005a | pending |
| 001-data-contract | TS/JS `Error.cause` chain parses into `chain[]` with `elidedFrameCount` omitted/0 | 006a | pending |
| 001-data-contract | File with uncommitted local changes → `codeContexts[].status: "stale"` | 004 | pending |
| 001-data-contract | File not present in checkout → `codeContexts[].status: "not_found"` | 004 | pending |
| 001-data-contract | Package with no locally resolvable version → `dependencies.locked[pkg].version` omitted, `.note` explains why (end-to-end; struct/JSON shape already covered by 001's own tests) | 005b, 006b | pending |

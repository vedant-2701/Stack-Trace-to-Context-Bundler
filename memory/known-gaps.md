# Known gaps

Two kinds of intentionally-not-done-yet, tracked in one file so there's a
single place to check instead of searching through spec.md/plan.md files
across every feature. Lives on disk (not just Claude's cross-session
memory) specifically so it survives an account change or a memory reset
-- this file, not memory, is the source of truth for both tables below.

**Checked at every feature kickoff** (`AGENTS.md` workflow step 2, both
tables): does this feature own a deferred criterion, or does it touch an
area with an accepted limitation worth re-examining now that real usage
exists?

## Deferred acceptance criteria

Acceptance criteria written in a completed feature's `spec.md` that
describe behavior owned by a *different*, not-yet-built feature -- not
rejected, just not this feature's job to satisfy.

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
| 001-data-contract | Package with no locally resolvable version → `dependencies.locked[pkg].version` omitted, `.note` explains why (end-to-end; struct/JSON shape already covered by 001's own tests) | 005b, 006b | pending |

## Accepted v1 limitations

Gaps with no owning feature -- deliberately scoped out of v1 rather than
deferred to a specific future feature. Each row should be traceable back
to the source feature's `plan.md`/spec.md for full reasoning; this table
is an index, not a duplicate copy of that reasoning (Article IV's
one-copy discipline, applied to docs instead of code).

**When starting or touching any feature listed below as source:** check
whether real usage has surfaced a reason to revisit -- if so, don't fix it
silently; propose the change and record why, same as any other
constitution-adjacent decision.

| Source feature | Limitation | Why accepted for v1 | Revisit if |
|---|---|---|---|
| _(none yet)_ | | | |

This is an audit pass — you are reviewing, not building or writing new
project content. Read every one of these files in full, if they exist:
AGENTS.md, memory/constitution.md, CONVENTIONS.md, specs/INDEX.md,
memory/decisions/INDEX.md, README.md, CLAUDE.md, GEMINI.md,
desktop-project-instruction.txt.

Check specifically for:

1. **Contradictions** — does any file state something that conflicts with
   another? (e.g. two files disagreeing on the stack, a convention that
   doesn't match a constitution principle, a boundary in one file that
   another file's instructions would violate.)

2. **Duplication** — is any fact stated in more than one place that could
   drift out of sync later? Flag it even if the two copies currently agree
   — the risk is future edits to one and not the other, not current
   correctness.

3. **Dangling references** — does any file reference a script, path, or
   file that doesn't actually exist in this repo? Check literally against
   the filesystem, don't assume a reference is correct because it reads
   plausibly.

4. **Unresolved markers** — any leftover `[NEEDS CLARIFICATION]` anywhere,
   in any file.

5. **Leftover scaffolding** — does `specs/001-example-feature/` or
   `memory/decisions/0001-example-decision.md` still exist? If so, ask
   whether that's intentional (kept as reference) or should be removed via
   `scripts/init-repo/run.sh` before the first real feature/decision is
   created, since leaving it changes what ID the first real one gets.

6. **Index integrity** — do the columns actually used in `specs/INDEX.md`
   and `memory/decisions/INDEX.md` match what the automation scripts
   (`new-feature`, `update-status`, `new-decision`) actually read and write?
   Does every status listed correspond to files that actually exist for
   that entry?

For anything found: don't fix it silently. List it, explain concretely why
it's a problem (not just "this could be an issue"), and propose a specific
fix. Ask before editing anything.

If, after checking all of the above, nothing is actually wrong, say so
plainly rather than manufacturing a minor issue to seem thorough. A clean
result is a valid result.

Do not start spec'ing any feature in this pass, even if the review surfaces
one you're eager to start.
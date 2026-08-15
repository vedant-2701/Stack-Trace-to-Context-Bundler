This is a technical design review — different from a file-consistency
check. Assume this feature's files are internally consistent; this pass
questions whether the *decisions* inside them actually hold up. Scoped to
a single feature, not the whole project.

Before doing anything else:

1. Read AGENTS.md, memory/constitution.md, and CONVENTIONS.md.

2. Read specs/INDEX.md. Find feature <FEATURE ID OR NAME> and its
   "Depends on" entries.

3. Read specs/<ID>-<name>/spec.md, plan.md, and tasks.md in full. Also read
   the spec.md and plan.md of every feature it depends on — don't rely on
   a summary or on memory of an earlier conversation about them; read the
   files as they stand right now.

Then go through each of these, one at a time, out loud — don't just skim
and conclude it's fine:

1. **Requirement coverage.** For every functional requirement in spec.md:
   does plan.md's design actually address it somewhere? Conversely, does
   plan.md build anything spec.md never asked for — scope creep that
   should either be pulled into spec.md explicitly or cut?

2. **Task breakdown.** For each task in tasks.md: is it actually one
   coherent, independently-verifiable unit of work, or secretly two things
   bundled together, or a fragment too small to test on its own? For each
   task's "Depends on" line: is that ordering real, or assumed without
   checking whether the two tasks actually need to be sequenced?

3. **Dependency fit.** Does this feature's design correctly account for
   what its dependencies *actually* provide, checked against their current
   files — not what was assumed about them when this feature's spec/plan
   were written? If a dependency's shape changed since (a contract field,
   a function signature, a flag) and this feature's plan.md still
   describes the old shape, that's a real bug waiting to surface at
   implementation time, not a documentation nitpick.

4. **Constitution/convention fit.** Does plan.md's design actually follow
   memory/constitution.md's principles and CONVENTIONS.md's idioms in
   practice, not just in the abstract — e.g. does every subprocess call it
   describes have a stated timeout if the constitution requires one; does
   it follow the project's error-handling/logging/naming conventions
   exactly, not approximately?

5. **Claims worth verifying.** Does plan.md rely on any claim about a
   library, tool, command flag, or external behavior that was taken at
   face value rather than actually checked? Verify it now via search or by
   testing directly rather than trusting the reasoning that was written
   down earlier — a plausible-sounding claim isn't a verified one.

For anything high-stakes and still genuinely uncertain after this pass —
not settled by a quick check — say so explicitly and suggest invoking the
llm-council skill before locking it in, rather than resolving it yourself
with a single pass.

For each issue found: state it, explain concretely why it's a problem, and
propose a specific fix or question. Ask before editing any file.

If everything genuinely holds up, say so plainly — don't manufacture a
concern to seem thorough.

Do not start implementing any task during this pass, even if the review
surfaces one you're eager to fix.

Feature: <ID/NAME FROM INDEX.md>

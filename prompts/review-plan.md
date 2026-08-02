This is a technical design review — different from a file-consistency check.
Assume the files themselves are internally consistent; this pass questions whether the *decisions*
inside them actually hold up.

Read `memory/constitution.md`, `AGENTS.md`, `CONVENTIONS.md`, and
`specs/INDEX.md` in full. Then go through each of these, one at a time, out
loud — don't just skim and conclude it's fine:

1. **Feature breakdown.** For each feature in `specs/INDEX.md`: is it
   actually one coherent feature, or is it secretly two that got merged, or
   a fragment of something that should be one feature? Is anything from the
   original idea missing entirely — not listed anywhere?

2. **Dependencies.** For each "Depends on" entry: is that dependency real,
   or was it assumed without checking whether the two features actually
   need to be sequenced? Also check the reverse — any feature that should
   list a dependency but doesn't.

3. **Stack fit.** Does the stack recorded in `AGENTS.md` actually fit what
   this project needs (scale, real-time requirements, team's stated
   experience, hosting constraints) — or was it picked out of familiarity
   without checking it against the actual requirements discussed at
   kickoff? If a claim about a library/framework's capability was taken at
   face value, verify it now via search rather than trusting the earlier
   conversation.

4. **Constitution soundness.** Do the principles in `memory/constitution.md`
   actually serve this specific project, or do they read like generic
   boilerplate that wasn't really stress-tested against this idea?

5. **Conventions fit.** Does `CONVENTIONS.md` reflect real idiomatic
   practice for the chosen stack, or generic advice that could apply to any
   language?

For anything high-stakes and still genuinely uncertain after this pass —
not settled by a quick check — say so explicitly and suggest invoking the
llm-council skill before locking it in, rather than resolving it yourself
with a single pass.

For each issue found: state it, explain concretely why it's a problem, and
propose a specific fix or question. Ask before editing any file.

If everything genuinely holds up, say so plainly — don't manufacture a
concern to seem thorough.

Do not start spec'ing or building any feature during this pass, even if the review surfaces one you're eager to start.
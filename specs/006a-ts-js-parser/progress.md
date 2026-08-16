# Progress Log: TypeScript/JS parser

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-08-16 (continued)
**Task(s):** Follow-up on Q2 -- resolving how far to push interface-first vs. parser-first sequencing.
**What happened:** Discussed the interface-first vs. parser-first tradeoff directly (parser-first risks expensive rework of real, already-tested code; pure interface-first with no concrete examples risks a clean-looking abstraction that doesn't actually fit either language). Landed on a middle path: split the reshaped 003 into **003a** (interface only -- method signatures + doc-comment contract, required to be validated by hand-tracing real Java and real TS/JS example traces through it as pseudocode in `plan.md`, not designed in the abstract; does not decide bucketing/elision/runtime-detection rules, those stay 005a/006a's job) and **003b** (the auto-detection registry, which keeps its original real-parsers-needed-to-test dependency on 005a/006a). `specs/INDEX.md` updated accordingly: 002b's deps now list 003a/003b instead of 003; 005a/006a now depend on `001, 003a, 004`; 006a's `spec.md` Q2 note updated to reference 003a specifically.
**Deviations from plan (if any):** None beyond what's already logged in the prior entry -- this refines that decision rather than reversing it.
**New open questions:** None new for 006a. Interrogation remains paused, now specifically pending **003a** (not 003) being spec'd/planned in a separate chat. Vedant is building 003a next; resume 006a's interrogation from Q2 once 003a exists.

---

**Date:** 2026-08-16
**Task(s):** Pre-kickoff (dependency check + start of spec interrogation; spec.md not yet fully written)
**What happened:** Confirmed 001 and 004 (006a's stated dependencies) are `done`. Checked `memory/known-gaps.md`: 006a owes the deferred criterion "TS/JS `Error.cause` chain parses into `chain[]` with `elidedFrameCount` omitted/0" from 001 -- to be folded into 006a's own acceptance criteria once interrogation resumes, not tracked separately. Flagged that `ComputeFingerprint` (`internal/contract/fingerprint.go`) already hard-codes "Frames[0] is the originating/throw-site frame" in a code comment, but this was never verified against a real parser or pinned as a requirement in 001's spec/plan -- must be explicitly confirmed as part of 006a's own spec (does 006a's own frame ordering actually put the throw-site frame at index 0?), not assumed. Asked Q1 (runtime/environment scope) -- resolved: Node.js/Bun/Deno CLI-or-server crash output only, no browser traces for v1; browser capture split out as new feature 012 in `specs/INDEX.md`. Asked Q2 (public API shape, given 003's `LanguageParser` interface doesn't exist yet) -- this led to a bigger decision: 003 is being reshaped to define that interface (shape + pseudocode/algorithm contract) as its own feature, built BEFORE 005a/006a instead of after (previously 003 depended on ≥1 real parser existing first). `specs/INDEX.md` updated: 003's description and `Depends on` changed (now depends on 001 only), 005a and 006a both now depend on 003, 006a's status moved to `specifying`.
**Deviations from plan (if any):** This reverses 003's previously-recorded dependency direction ("needs ≥1 real parser (005a or 006a) to test against"). Flagged to Vedant as a real Article VIII (no speculative capability) tension: designing an interface generalized enough for hypothetical future languages (Go, Rust were mentioned) before any real parser exists to validate it is a genuine premature-abstraction risk, not just a philosophical one -- Go/Rust don't share Java/JS's thrown-exception-with-automatic-stack-capture model at all. Recommended, and reflected in `specs/INDEX.md`'s new row text: scope the reshaped 003 to generalize cleanly across Java + TS/JS only (the two committed v1 languages); treat further-language generalization as an explicit non-goal until a real Go/Rust (etc.) parser feature exists to test against. Final scoping call for 003 itself is deferred to the new chat that specs it.
**New open questions:** None new for 006a itself. Interrogation is paused (Q1 resolved, Q2 blocked) -- resume once 003 (Language parser interface & auto-detection) is spec'd/planned in a separate chat. `spec.md`, `plan.md`, and `tasks.md` here still hold only template placeholders beyond the Open Questions section.

---

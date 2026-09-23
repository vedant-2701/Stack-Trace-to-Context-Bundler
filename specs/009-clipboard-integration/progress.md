# Progress Log: Clipboard integration

Append an entry each time a task is completed or a significant decision is made.
This is what lets you (or an agent) resume the feature in a new session without
losing context.

---

**Date:** 2026-09-23
**Task(s):** Post-spec audit revisions (no implementation task started)
**What happened:** An audit pass, done before any implementation, surfaced two issues in the already-written spec.md/plan.md/tasks.md, both now fixed with the user's direction: (1) tasks.md's T002 said `Run`'s FR7 timeout applies "regardless of the caller's `ctx`" -- ambiguous wording that could be read as ignoring `ctx` entirely, contradicting FR7's own text ("on the caller-supplied ctx") and the `execGitRunner.Run` precedent it claims to mirror (`context.WithTimeout(ctx, gitTimeout)`, deriving from the passed-in `ctx`). Fixed by rewording T002 and plan.md's File layout to explicitly derive the timeout via `context.WithTimeout(ctx, clipboardTimeout)` from the caller's `ctx`, and introducing `clipboardTimeout` as a named package-level `const` (mirroring `gitTimeout`) so the 5s value lives in one declaration, not scattered literals -- this does NOT reopen the already-rejected runtime-configurability question (the Non-functional requirements still say fixed, unexported Go constant, no CLI flag/env var/config file). (2) spec.md's FR5 and its Acceptance criteria never covered the "wl-copy not found, xclip found+succeeds" permutation, nor the "one tool found+failed, the other simply absent" `ErrClipboardWriteFailed` cases -- both real, valid FR5 outcomes with a different error-message shape (only actually-invoked tools get named) than the "both found+failed" case. Rewrote FR5 to define "unavailable" once (not-found and found-but-failed are the same fallback trigger) and consolidated the Acceptance criteria's 4 near-duplicate/incomplete Linux bullets into 3 complete, non-redundant ones. tasks.md's T006 now lists all 7 real permutations as distinct `TestWrite` table entries (not-found and found-but-failed are different `fakeCmdRunner` setups even when the downstream behavior matches), up from 4.
**Deviations from plan (if any):** N/A -- spec.md/plan.md/tasks.md revised, no code written, per the user's explicit instruction not to implement in this pass.
**New open questions:** None.

---

**Date:** 2026-09-23
**Task(s):** Spec interrogation complete; plan.md + tasks.md written
**What happened:** Interrogated the user one question at a time (all via tappable options) to resolve every [NEEDS CLARIFICATION] item: (1) public API is `func Write(ctx context.Context, text string) error` in package `clipboard`, matching `codecontext`'s ctx-threading convention; (2) Linux detection special-cases WSL (bridges to `clip.exe` exclusively) since `wl-copy`/`xclip` don't work in a bare WSL shell -- this project's real dev environment; (3) the Linux fallback from `wl-copy` to `xclip` is a runtime-failure fallback, not presence-only; (4) subprocess timeout is a hardcoded 5s constant, NOT configurable via `.env` -- pushed back on a `.env`-based config request (verified via `go.mod` that no `.env`/config-file mechanism exists anywhere in this codebase, only `pflag`; also conflicts with feature 011's established precedent of exposing a tunable via a dedicated CLI flag feature, not a config file) and the user agreed to hardcode it instead; (5) two sentinel errors (`ErrNoClipboardUtility`, `ErrClipboardWriteFailed`) so a future 002b can distinguish exit codes, mirroring `parser.ErrNoMatch`/`ErrAmbiguous`; (6) testing is fake-only for 009 -- confirmed via directory listing that this repo has no CI at all (no `.github/`), so build-tag-gated real-subprocess tests would currently run nowhere; revisit once CI exists. Also verified during interrogation, not just assumed: `render.JSON`/`render.Markdown` both already return plain `string`, confirming 009 genuinely has no dependency on `internal/contract` (INDEX.md's "Depends on: —" is correct, not just trusted). Wrote `spec.md` (Approved), `plan.md`, and `tasks.md` (T001-T009) reflecting all of the above.
**Deviations from plan (if any):** N/A -- this session only specs/plans, per the user's explicit instruction not to start implementation in this pass.
**New open questions:** None remaining in spec.md. plan.md flags two unproven-but-accepted risks for whoever picks up T003/T006: `isWSL()`'s heuristic hasn't been smoke-tested against this machine's real WSL2 environment yet (T003's acceptance line requires doing so before trusting it), and `xclip -selection clipboard`'s exact invocation shape is asserted from common practice, not a real invocation (no CI to catch a real-argument bug).

---

**Date:** 2026-09-23
**Task(s):** Kickoff — spec interrogation started
**What happened:** specs/009-clipboard-integration/ created from specs/_templates/. Checked specs/INDEX.md: 009 has no dependencies. Checked memory/known-gaps.md: no deferred acceptance criteria owed to 009, no accepted v1 limitations touch it. No prior progress.md existed -- this is a fresh kickoff, not a resume.
**Deviations from plan (if any):** N/A
**New open questions:** See spec.md's Open questions section.

---

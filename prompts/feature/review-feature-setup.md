This is an audit pass — you are reviewing, not building or writing new
feature content. Scoped to a single feature's own files, not the whole
project.

Before doing anything else:

1. Read AGENTS.md, memory/constitution.md, and CONVENTIONS.md.

2. Read specs/INDEX.md. Find feature <FEATURE ID OR NAME> and note its
   current status and "Depends on" entries.

3. Read specs/<ID>-<name>/spec.md, plan.md, tasks.md, and progress.md in
   full — all four, even if the feature looks early-stage. Also read
   spec.md/plan.md for every feature listed in its "Depends on" column,
   since several of the checks below need to compare against what those
   dependencies actually say now, not what this feature's docs assumed
   about them when they were written.

Check specifically for:

1. **Contradictions** — does this feature's spec.md, plan.md, or tasks.md
   state something that conflicts with another of its own files (e.g.
   plan.md's architecture doesn't match a functional requirement in
   spec.md; tasks.md orders work in a sequence plan.md's design doesn't
   support)? Also check against the shared governance files — does
   anything here quietly contradict a principle in memory/constitution.md
   or an idiom in CONVENTIONS.md?

2. **Duplication** — is any concrete fact (a default value, a timeout, a
   field name, a status enum's exact values) stated in more than one of
   this feature's files? Flag it even if the copies currently agree — the
   risk is one getting edited later and not the other. Pay particular
   attention to values that also appear in a dependency's own spec.md/
   plan.md (e.g. a contract shape this feature relies on) — are they still
   consistent with what the dependency's files say today, not just what
   was true when this feature's docs were written?

3. **Dangling references** — does spec.md, plan.md, or tasks.md reference
   a file, package, function, script, or another feature ID that doesn't
   actually exist in this repo, or doesn't exist yet in the form assumed?
   Check literally against the filesystem, don't assume a reference is
   correct because it reads plausibly.

4. **Unresolved markers** — any leftover `[NEEDS CLARIFICATION]` in any of
   this feature's four files.

5. **Task/acceptance-criteria integrity** — does every acceptance
   criterion in spec.md map to at least one task in tasks.md that would
   actually verify it? Is there a task in tasks.md marked `[x]` with no
   corresponding entry in progress.md, or a progress.md entry describing
   work with no matching completed task? Do tasks.md's `[ ]`/`[~]`/`[x]`
   markers match what progress.md's log actually says happened?

6. **Dependency-state integrity** — for each feature listed in "Depends
   on": does specs/INDEX.md say it's actually at the status this feature's
   plan.md assumes? If this feature's design relies on a specific shape of
   a dependency (a struct field, a function signature, a CLI flag), does
   that dependency's current files still define it that way — not the
   shape assumed when this feature's spec/plan were written, which may
   have drifted if the dependency changed since?

For anything found: don't fix it silently. List it, explain concretely why
it's a problem (not just "this could be an issue"), and propose a specific
fix. Ask before editing anything.

If, after checking all of the above, nothing is actually wrong, say so
plainly rather than manufacturing a minor issue to seem thorough. A clean
result is a valid result.

Do not start implementing any task in this pass, even if the review
surfaces one you're eager to fix.

Feature: <ID/NAME FROM INDEX.md>

Before doing anything else:

1. Read AGENTS.md, memory/constitution.md, and CONVENTIONS.md.

2. Read specs/INDEX.md. Find feature <FEATURE ID OR NAME> and check its
   dependencies — if a dependency isn't marked "done," tell me before
   proceeding.

3. Check specs/<ID>-<name>/progress.md if that folder already exists. If it
   has entries, this is a resume — summarize where things left off and
   confirm with me before continuing. If the folder doesn't exist yet,
   create it using `./scripts/new-feature/run.sh <name>` (or `.\scripts\new-feature\run.ps1` on Windows)
   or from `specs/_templates/`.

4. Interrogate me about this feature specifically, one question at a time
   with a suggested answer, until spec.md can be written without guesses.
   Mark anything unresolved as [NEEDS CLARIFICATION] and don't move past it.

5. Write spec.md. Once every [NEEDS CLARIFICATION] is resolved, write
   plan.md — consistent with the constitution and conventions, not
   contradicting either. Then write tasks.md, broken into small, ordered,
   independently verifiable steps.

6. Update this feature's row in specs/INDEX.md (status, and dependency
   info if it changed).

7. Stop here — do not start implementation in this same pass unless I
   explicitly ask you to.

Feature: <ID/NAME FROM INDEX.md>
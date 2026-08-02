Before doing anything else:

1. Read AGENTS.md, memory/constitution.md, and CONVENTIONS.md.

2. Read specs/INDEX.md. Find feature <FEATURE ID OR NAME> and verify its status is "in-progress" (or update it to "in-progress" using ./scripts/update-status/run.sh). Check its dependencies — if any dependency isn't marked "done," flag it to me before proceeding.

3. Read specs/<ID>-<name>/plan.md, tasks.md, and progress.md.

4. Identify the next uncompleted task in tasks.md (marked `[ ]`). Summarize what task needs to be implemented and ask for confirmation before writing any code.

5. Implement **only** that single task. Show the proposed diff or code changes before applying them.

6. Once verified, mark the task as completed (`[x]`) in tasks.md, append a log entry to progress.md detailing what changed/decisions made, and update specs/INDEX.md if the feature is fully completed.

Feature: <ID/NAME FROM INDEX.md>
Task: <TASK ID E.G. T001 (OPTIONAL, DEFAULTS TO NEXT TODO TASK)>

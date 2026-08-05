Before doing anything else:

Throughout every step below: if you don't have a tool that executes commands directly on the user's machine, say so plainly, give the exact command(s) to run, and wait for the user to paste back the actual output before proceeding. Never assume, guess, or claim a result you haven't actually seen run.

1. Read AGENTS.md, memory/constitution.md, and CONVENTIONS.md.

2. Read specs/INDEX.md. Find feature <FEATURE ID OR NAME> and verify its status is "in-progress" (or update it to "in-progress" using ./scripts/update-status/run.sh). Check its dependencies — if any dependency isn't marked "done," flag it to me before proceeding.

3. Read specs/<ID>-<name>/spec.md, plan.md, tasks.md, and progress.md. spec.md holds the acceptance criteria and out-of-scope boundaries that tasks.md doesn't fully restate — don't skip it.

4. Identify the next uncompleted task in tasks.md (marked `[ ]`). Summarize what task needs to be implemented and ask for confirmation before writing any code.

5. Implement **only** that single task. Show the proposed diff or code changes before applying them.

6. Verify the task actually satisfies its stated acceptance criteria: run `go build`, `go test ./...`, and `golangci-lint run` for the affected package, and fix any failures before proceeding. Also run `gofumpt -l` directly on the changed files and confirm it reports nothing — don't rely solely on the pre-commit hook for this. Automated gates in this repo have had bugs before, so if a gate's behavior doesn't match what its config claims (e.g. a commit goes through despite unformatted code), flag it explicitly rather than assuming it's working.

7. Do not generate or offer a commit message, git commands, or PR description yet. Wait for the user to explicitly confirm the code is working correctly — whether that's from output you ran yourself or output they pasted back. Only once they confirm, ask whether they'd like the commit message, the exact commands to run, and/or a PR description.

8. Once the user has confirmed and any commit has happened (or been explicitly declined), mark the task as completed (`[x]`) in tasks.md, append a log entry to progress.md detailing what changed/decisions made, and update specs/INDEX.md if the feature is fully completed.

9. If providing a commit message, follow COMMIT_CONVENTIONS.md's `<type>(<scope>): <description>` format — enforced by a commit-msg hook, so a message that doesn't match will be rejected.

Feature: <ID/NAME FROM INDEX.md>
Task: <TASK ID E.G. T001 (OPTIONAL, DEFAULTS TO NEXT TODO TASK)>

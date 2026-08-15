package codecontext

import (
	"context"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// BuildGitMetadata detects whether workDir is inside a git working tree
// and, if so, returns the bundle-level repo state (spec.md requirements
// 3, 4, 8). It never returns an error: a failure to find or read git
// metadata is a valid, representable outcome (nil) that the caller
// doesn't need to branch on separately -- mirrors
// contract.TruncateRawInput's "outcome as a value, not an error" shape.
// This is the production entry point; it wraps the real gitRunner.
func BuildGitMetadata(ctx context.Context, workDir string) *contract.GitMetadata {
	return buildGitMetadata(ctx, workDir, execGitRunner{})
}

// buildGitMetadata is the runner-injectable implementation. Exercised
// directly by this package's table-driven tests via fakeGitRunner, so
// go test ./internal/codecontext/... never needs a real git binary.
func buildGitMetadata(ctx context.Context, workDir string, runner gitRunner) *contract.GitMetadata {
	if !isInsideGitWorkTree(ctx, workDir, runner) {
		return nil
	}

	currentCommit, err := runner.Run(ctx, workDir, "rev-parse", "HEAD")
	if err != nil {
		// A repo was just confirmed to exist, but HEAD can't be
		// resolved (e.g. a brand-new repo with zero commits), or this
		// specific call failed/timed out independently of the
		// detection call above. Not explicitly named in spec.md
		// requirement 8 (which only covers the detection call's
		// timeout), but a GitMetadata with some fields populated and
		// others silently zero-valued would violate requirement 4's
		// "all three always populated" invariant and Article VI's
		// "never guess silently" -- so this collapses to the same nil
		// outcome as "no repo found" rather than a partial struct.
		return nil
	}

	branch, err := runner.Run(ctx, workDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil
	}

	// No pathspec: reports the whole working tree's status, not just
	// workDir's subdirectory -- correct regardless of which
	// subdirectory of the repo workDir happens to be, per git's own
	// upward discovery (same reasoning plan.md's design review already
	// relies on for per-file commands, requirement 9).
	statusOut, err := runner.Run(ctx, workDir, "status", "--porcelain")
	if err != nil {
		return nil
	}

	return &contract.GitMetadata{
		CurrentCommit:      strings.TrimSpace(currentCommit),
		Branch:             strings.TrimSpace(branch),
		UncommittedChanges: strings.TrimSpace(statusOut) != "",
	}
}

// isInsideGitWorkTree runs `git rev-parse --is-inside-work-tree` to
// answer spec.md requirement 3. Deliberately the *only* detection call
// -- `--show-toplevel` was dropped per this task's design-review note in
// plan.md: nothing in this feature consumes a repo-root path
// (contract.GitMetadata has no field for one, and per-file git commands
// rely on git's own upward .git discovery per spec.md requirement 9,
// not a passed-in root), so a second call would only spend half the
// detection timeout budget for no consumer.
func isInsideGitWorkTree(ctx context.Context, workDir string, runner gitRunner) bool {
	out, err := runner.Run(ctx, workDir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		// Covers both "not a git repository" and a detection timeout
		// (context.DeadlineExceeded) identically, per spec.md
		// requirement 8's "rev-parse timeout -> treated as no repo
		// found" rule -- both resolve to false here, no separate
		// branch needed.
		return false
	}
	return strings.TrimSpace(out) == "true"
}

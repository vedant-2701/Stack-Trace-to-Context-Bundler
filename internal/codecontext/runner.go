// Package codecontext builds language-agnostic own-code context for a
// bundle: windowed source snippets, git blame, and repo-level git
// metadata for own-bucket frames. See
// specs/004-own-code-context-extraction for the full design. Everything
// here is a pure function or a function taking a small injected
// interface for subprocess calls (CONVENTIONS.md's testing section).
package codecontext

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// gitTimeout is the hard per-call timeout for every git subprocess this
// package invokes (rev-parse, status, blame). Applied independently per
// call (spec.md requirement 8): one slow call must never block another
// -- a hanging blame on one file can't stall another file's staleness
// check or the initial repo detection.
const gitTimeout = 10 * time.Second

// gitRunner abstracts a single `git <args...>` subprocess invocation, so
// every caller in this package can be tested without a real git binary
// on PATH (CONVENTIONS.md's testing section) via a hand-written fake
// (runner_fake_test.go), not a mocking framework.
type gitRunner interface {
	// Run executes `git <args...>` with cwd set to dir. The 10-second
	// hard timeout (spec.md requirement 8) is applied internally by the
	// implementation -- callers pass a normal ctx (request-scoped
	// cancellation, or context.Background()) and do not need to set up
	// their own per-call deadline. Returns stdout on success. A non-nil
	// error covers both a real git error and a timeout; callers
	// distinguish the two via errors.Is(err, context.DeadlineExceeded),
	// since spec.md requirement 8 calls for different handling per call
	// type (rev-parse timeout -> treated as "no repo"; status timeout
	// -> treated as "stale"; blame timeout -> treated as a blame
	// failure).
	Run(ctx context.Context, dir string, args ...string) (stdout string, err error)
}

// execGitRunner is the production gitRunner, shelling out to the real
// git binary via os/exec (constitution Article VII: shell out, don't
// embed -- no CGO bindings, no embedded native library).
type execGitRunner struct{}

var _ gitRunner = execGitRunner{}

// Run implements gitRunner.
func (execGitRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(callCtx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if callCtx.Err() != nil {
			// The timeout firing takes priority over whatever exec
			// itself reports -- a killed process often surfaces as a
			// generic "signal: killed" rather than a deadline error,
			// but the real, caller-relevant cause is the timeout.
			return "", fmt.Errorf("git %v: %w", args, callCtx.Err())
		}
		return "", fmt.Errorf("git %v: %w (stderr: %s)", args, err, stderr.String())
	}

	return stdout.String(), nil
}

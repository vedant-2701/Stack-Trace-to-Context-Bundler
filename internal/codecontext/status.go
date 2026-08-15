package codecontext

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// gitStatus is the parsed result of `git status --porcelain <file>`,
// used internally to decide ok/stale (spec.md requirement 5) before
// context.go (T007) constructs the actual contract.CodeContext.
type gitStatus int

const (
	gitStatusClean     gitStatus = iota
	gitStatusModified            // tracked, uncommitted changes
	gitStatusUntracked           // never committed
	gitStatusUnknown             // status check itself failed/timed out -- spec.md requirement 8's cautious "stale" path
)

// checkFileStatus runs `git status --porcelain <file>` for one
// own-bucket frame's file (spec.md requirement 5), with cwd set to the
// file's own directory so the command resolves via git's upward .git
// discovery without needing a separately-tracked repo root -- same
// mechanism plan.md's design review already relies on for
// BuildGitMetadata (T003). Returns the parsed gitStatus plus a note
// explaining it; the note is empty for gitStatusClean, since context.go
// only needs an explanation for the non-clean outcomes.
//
// Any failure calling git -- a real error or a timeout alike -- resolves
// to gitStatusUnknown, spec.md requirement 8's cautious default: an
// unverified file is presented as possibly-stale, never confidently
// clean. This also transparently covers spec.md requirement 9's "file
// outside the repo found from the working directory" edge case (that
// status command fails with "not a git repository") without any extra
// branching -- both are simply "couldn't verify," just with different
// note wording.
func checkFileStatus(ctx context.Context, filePath string, runner gitRunner) (gitStatus, string) {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	out, err := runner.Run(ctx, dir, "status", "--porcelain", base)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return gitStatusUnknown, "git status could not be verified within the timeout"
		}
		return gitStatusUnknown, fmt.Sprintf("git status failed: %s", err)
	}

	// A single-file pathspec produces at most one line of porcelain
	// output; take the first defensively rather than assuming that
	// holds.
	firstLine := out
	if idx := strings.IndexByte(out, '\n'); idx != -1 {
		firstLine = out[:idx]
	}
	trimmed := strings.TrimSpace(firstLine)

	if trimmed == "" {
		return gitStatusClean, ""
	}
	if strings.HasPrefix(trimmed, "??") {
		return gitStatusUntracked, "file is untracked (never committed)"
	}
	return gitStatusModified, "file has uncommitted local changes"
}

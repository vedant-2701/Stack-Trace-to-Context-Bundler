package typescript

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// nodeVersionTimeout is the hard timeout for the "node --version"
// shell-out (spec.md FR15): 2s, much shorter than
// internal/codecontext/runner.go's 10s git timeout, because
// "node --version" is near-instant on a working install -- a longer
// timeout would only delay failure on a broken/hung node, unlike git
// operations which can legitimately be slow.
const nodeVersionTimeout = 2 * time.Second

// nodeVersionRunner abstracts a single "node --version" subprocess
// invocation, mirroring internal/codecontext/runner.go's gitRunner --
// same reason: this package's local-environment fallback (spec.md FR15)
// must be testable without a real node binary on PATH (CONVENTIONS.md's
// testing section), via a hand-written fake (runtime_test.go), not a
// mocking framework.
type nodeVersionRunner interface {
	// Run executes "node --version". The 2-second hard timeout
	// (nodeVersionTimeout) is applied internally by the production
	// implementation -- callers pass a normal ctx and don't set up
	// their own deadline. Returns stdout on success. A non-nil error
	// covers both a real failure (node not on PATH) and a timeout.
	Run(ctx context.Context) (stdout string, err error)
}

// execNodeVersionRunner is the production nodeVersionRunner, shelling
// out to the real node binary via os/exec (constitution Article VII:
// shell out, don't embed).
type execNodeVersionRunner struct{}

var _ nodeVersionRunner = execNodeVersionRunner{}

// Run implements nodeVersionRunner.
func (execNodeVersionRunner) Run(ctx context.Context) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, nodeVersionTimeout)
	defer cancel()

	cmd := exec.CommandContext(callCtx, "node", "--version")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if callCtx.Err() != nil {
			// The timeout firing takes priority over whatever exec
			// itself reports, same reasoning as
			// internal/codecontext/runner.go's execGitRunner.Run.
			return "", fmt.Errorf("node --version: %w", callCtx.Err())
		}
		return "", fmt.Errorf("node --version: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}

// localNodeVersionPattern matches "node --version"'s stdout: a leading
// "v", dotted version numbers, then optional trailing whitespace
// (the real output is "vX.Y.Z\n").
var localNodeVersionPattern = regexp.MustCompile(`^v(\d+\.\d+\.\d+)\s*$`)

// localNodeVersion implements spec.md FR15/16: shells out to the local
// environment via runner, bounded by nodeVersionTimeout. On success,
// returns the parsed version, VersionSourceLocalEnvironment, and a note
// explaining the version was inferred locally and may not reflect drift
// (e.g. an active nvm/fnm version, or a container) even on the same
// physical machine. On any failure -- node not on PATH, a timeout, or
// output that doesn't parse as a version -- returns VersionSourceUnknown
// with no version and no note. VersionSourceUnknown is a real,
// meaningful value per its own doc comment in internal/contract/types.go,
// not an omission.
//
// This returns the raw Version/VersionSource/Note trio, not a full
// contract.Runtime -- assembling the complete Runtime (including Name,
// which is always "node" per FR13 regardless of this function's result)
// is composition-layer work, T010's job.
func localNodeVersion(ctx context.Context, runner nodeVersionRunner) (version string, source contract.VersionSource, note string) {
	stdout, err := runner.Run(ctx)
	if err != nil {
		return "", contract.VersionSourceUnknown, ""
	}

	m := localNodeVersionPattern.FindStringSubmatch(stdout)
	if m == nil {
		return "", contract.VersionSourceUnknown, ""
	}

	return m[1], contract.VersionSourceLocalEnvironment,
		"version inferred from local `node --version`; may not reflect the environment that actually produced this trace (e.g. an active nvm/fnm version, or a container), even on the same machine"
}

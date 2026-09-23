package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// clipboardTimeout is the hard per-call timeout for every clipboard
// subprocess this package invokes (pbcopy, wl-copy, xclip, clip.exe).
// Applied independently per attempt (spec.md FR7): one candidate tool
// timing out must never block a fallback attempt at a different tool.
// Fixed, unexported constant for v1 -- not configurable (spec.md's
// Non-functional requirements).
const clipboardTimeout = 5 * time.Second

// cmdRunner abstracts a single clipboard-tool subprocess invocation, so
// every caller in this package can be tested without any of
// pbcopy/wl-copy/xclip/clip.exe actually being installed
// (CONVENTIONS.md's testing section) via a hand-written fake
// (clipboard_test.go), not a mocking framework.
type cmdRunner interface {
	// LookPath reports whether name is resolvable on PATH. Used to tell
	// "not found" (spec.md FR8's ErrNoClipboardUtility) apart from
	// "found but its invocation failed" (ErrClipboardWriteFailed).
	LookPath(name string) bool

	// Run executes name with args, writing stdin to the subprocess's
	// standard input byte-for-byte (spec.md FR1: no added/stripped
	// newline, no re-encoding). The 5-second hard timeout (spec.md FR7)
	// is applied internally by the implementation, derived from the
	// caller-supplied ctx -- callers pass a normal ctx (request-scoped
	// cancellation, or context.Background()) and do not need to set up
	// their own per-call deadline. A non-nil error covers both a real
	// subprocess error (non-zero exit) and a timeout; callers fold both
	// into the same "this attempt failed, try the next candidate"
	// handling (spec.md FR5/FR6) rather than distinguishing them.
	Run(ctx context.Context, name string, args []string, stdin string) error
}

// execCmdRunner is the production cmdRunner, shelling out to the real
// clipboard tool via os/exec (constitution Article VII: shell out,
// don't embed -- no CGO bindings, no embedded native library).
type execCmdRunner struct{}

var _ cmdRunner = execCmdRunner{}

// LookPath implements cmdRunner.
func (execCmdRunner) LookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Run implements cmdRunner.
func (execCmdRunner) Run(ctx context.Context, name string, args []string, stdin string) error {
	callCtx, cancel := context.WithTimeout(ctx, clipboardTimeout)
	defer cancel()

	cmd := exec.CommandContext(callCtx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if callCtx.Err() != nil {
			// The timeout firing takes priority over whatever exec
			// itself reports -- a killed process often surfaces as a
			// generic "signal: killed" rather than a deadline error,
			// but the real, caller-relevant cause is the timeout.
			return fmt.Errorf("%s %v: %w", name, args, callCtx.Err())
		}
		return fmt.Errorf("%s %v: %w (stderr: %s)", name, args, err, stderr.String())
	}

	return nil
}

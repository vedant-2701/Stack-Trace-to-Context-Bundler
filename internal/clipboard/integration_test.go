//go:build integration

package clipboard

// This file exercises the real os/exec execCmdRunner (T002) against real
// binaries -- clip.exe bridged into this WSL environment, and (for
// TestIntegration_ExecCmdRunner_RealTimeout) the real `sleep` coreutil to
// prove the actual 5s clipboardTimeout constant is enforced end-to-end
// through a real hung subprocess, not just the fake's simulated version
// in clipboard_test.go's TestWrite_Timeout. Calls execCmdRunner directly
// rather than through Write/isWSL, so these tests exercise T002/T007's
// actual subprocess invocation regardless of whether isWSL()'s heuristic
// (T003) still correctly detects this environment. This machine only has
// clip.exe available for clipboard tools specifically (no pbcopy/wl-copy/
// xclip -- spec.md's Out of scope), so those remain untested here.
// Excluded from the default `go test ./...` (lefthook, and any default
// CI job) by the build tag above, per CONVENTIONS.md's testing section --
// run explicitly via:
//
//	go test -tags integration ./internal/clipboard/...
//
// TestIntegration_ExecCmdRunner_RealTimeout deliberately takes ~5
// real seconds to run (it waits out the real clipboardTimeout) -- that
// cost is exactly why this lives behind the integration tag rather than
// in the default fast unit-test suite.

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIntegration_RealClipExe_WSL(t *testing.T) {
	runner := execCmdRunner{}

	if !runner.LookPath("clip.exe") {
		t.Fatal("clip.exe not found on PATH -- expected on this WSL dev machine")
	}

	err := runner.Run(context.Background(), "clip.exe", nil, "stack-trace-bundler integration test\n")
	if err != nil {
		t.Fatalf("execCmdRunner.Run(clip.exe) error = %v, want nil", err)
	}
}

// TestIntegration_ExecCmdRunner_RealTimeout proves clipboardTimeout (5s,
// runner.go) is actually enforced against a real, genuinely hung
// subprocess -- not just the fake's simulated ctx.Done() in
// clipboard_test.go's TestWrite_Timeout, which never exercises
// execCmdRunner's own context.WithTimeout derivation at all. `sleep 10`
// is used as the hung process: any real coreutils `sleep` on this WSL
// machine qualifies, it isn't a clipboard tool -- execCmdRunner.Run has
// no clipboard-specific behavior, so this is a faithful test of the
// timeout mechanism itself.
func TestIntegration_ExecCmdRunner_RealTimeout(t *testing.T) {
	runner := execCmdRunner{}

	if !runner.LookPath("sleep") {
		t.Fatal("sleep not found on PATH -- expected on this WSL dev machine")
	}

	start := time.Now()
	err := runner.Run(context.Background(), "sleep", []string{"10"}, "")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run(sleep 10) error = nil, want a timeout error -- clipboardTimeout (5s) should have fired before sleep's own 10s completed")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run(sleep 10) error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}

	// clipboardTimeout is a fixed 5s; generous slack on both sides
	// absorbs real process spawn/kill/scheduling overhead on this
	// machine without masking a real regression -- well under sleep's
	// own 10s (which would mean the timeout never fired at all) and
	// well over near-instant (which would mean ctx was already dead
	// before this call even started).
	if elapsed < 4*time.Second || elapsed > 8*time.Second {
		t.Errorf("Run(sleep 10) took %v, want ~%v (clipboardTimeout), not sleep's own duration", elapsed, clipboardTimeout)
	}
}

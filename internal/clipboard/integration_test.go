//go:build integration

package clipboard

// This file exercises the real os/exec execCmdRunner (T002) against the
// real clip.exe binary bridged into this WSL environment, validating
// that clip.exe genuinely accepts a plain-text write from this
// package -- not just the hand-written fake's idea of it (plan.md's
// Testing strategy). Calls execCmdRunner directly rather than through
// Write/isWSL, so this test exercises T002+T007's actual subprocess
// invocation regardless of whether isWSL()'s heuristic (T003) still
// correctly detects this environment. This machine only has clip.exe
// available (no pbcopy/wl-copy/xclip -- spec.md's Out of scope), so
// those are not exercised here. Excluded from the default `go test
// ./...` (lefthook, and any default CI job) by the build tag above, per
// CONVENTIONS.md's testing section -- run explicitly via:
//
//	go test -tags integration ./internal/clipboard/...

import (
	"context"
	"testing"
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

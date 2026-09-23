// Package clipboard writes an already-rendered bundle string to the OS
// clipboard via an OS-appropriate subprocess (constitution Article VII:
// shell out, don't embed). See specs/009-clipboard-integration for the
// full design.
package clipboard

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

// tool names a single clipboard-tool candidate: an executable name and
// the arguments it needs for a plain copy from stdin (nil for
// pbcopy/wl-copy/clip.exe; ["-selection", "clipboard"] for xclip).
type tool struct {
	name string
	args []string
}

// tryOne attempts a single candidate tool. It is tryChain with a
// single-element list (plan.md's Architecture) -- FR6's "single
// candidate" case is naturally the same code path as FR5's chain, not a
// separate implementation.
func tryOne(ctx context.Context, name string, args []string, text string, runner cmdRunner) error {
	return tryChain(ctx, []tool{{name: name, args: args}}, text, runner)
}

// tryChain attempts each candidate tool in order via runner.LookPath/
// runner.Run: a tool not found on PATH is skipped; a tool found but
// whose invocation fails (non-zero exit or the FR7 timeout) counts as a
// failed attempt and falls through to the next candidate (spec.md FR5).
// Returns nil on the first successful invocation. If no candidate was
// found on PATH at all, returns an error wrapping ErrNoClipboardUtility
// and naming every tool that was checked. If at least one candidate was
// found but every found candidate's invocation failed, returns an error
// wrapping ErrClipboardWriteFailed naming only the tool(s) that were
// actually found and invoked, and how each failed -- a tool that was
// never found is never named as a "failure" (spec.md FR5/FR6).
func tryChain(ctx context.Context, tools []tool, text string, runner cmdRunner) error {
	var attempted, failures []string

	for _, t := range tools {
		if !runner.LookPath(t.name) {
			continue
		}
		attempted = append(attempted, t.name)

		if err := runner.Run(ctx, t.name, t.args, text); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", t.name, err))
			continue
		}
		return nil
	}

	if len(attempted) == 0 {
		names := make([]string, len(tools))
		for i, t := range tools {
			names[i] = t.name
		}
		return fmt.Errorf("%w (checked: %s)", ErrNoClipboardUtility, strings.Join(names, ", "))
	}

	return fmt.Errorf("%w (%s)", ErrClipboardWriteFailed, strings.Join(failures, "; "))
}

// write is the injectable core behind Write, taking goos/wsl as plain
// values rather than an injected detector interface (plan.md's
// Alternatives considered) so every OS/WSL/fallback branch is directly
// testable regardless of which OS actually runs `go test`.
//
// The linux branch is a stub for now (unconditionally returns
// ErrNoClipboardUtility, ignoring wsl) -- the WSL-exclusive clip.exe
// branch and the non-WSL wl-copy/xclip fallback chain are completed in
// T007 and T006 respectively.
func write(ctx context.Context, text, goos string, wsl bool, runner cmdRunner) error {
	switch goos {
	case "darwin":
		return tryOne(ctx, "pbcopy", nil, text, runner)
	case "windows":
		return tryOne(ctx, "clip.exe", nil, text, runner)
	case "linux":
		_ = wsl // handled once T006/T007 land the real linux branch
		return ErrNoClipboardUtility
	default:
		// Unrecognized GOOS: no attempt made (spec.md FR6).
		return ErrNoClipboardUtility
	}
}

// Write sends text to the OS clipboard byte-for-byte via an OS-
// appropriate subprocess (constitution Article VII). See spec.md for
// full OS/WSL/fallback/timeout behavior.
func Write(ctx context.Context, text string) error {
	goos := runtime.GOOS

	var wsl bool
	if goos == "linux" {
		wsl = isWSL()
	}

	return write(ctx, text, goos, wsl, execCmdRunner{})
}

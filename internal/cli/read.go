package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// boundedReadCapBytes bounds how much readTrace buffers from either
// source before contract.TruncateRawInput's 512KB rune-safe cap applies,
// per spec.md requirement 9 -- deliberately larger than that cap so an
// ordinary large trace isn't cut mid-read before truncation gets a clean
// chance to run. Provisional pending real-world calibration, same spirit
// as ADR 0001's provisional 30s timeout (see plan.md's Risks section).
const boundedReadCapBytes = 1024 * 1024 // 1MB

// readTrace selects the trace source -- a positional file argument wins
// over piped stdin, per spec requirement 5 -- reads it bounded via
// io.LimitReader, rejects empty/whitespace-only content, and applies
// contract.TruncateRawInput for the final rune-safe 512KB cap.
//
// readTrace is a pure function: it never logs and never calls os.Exit.
// stdinIgnored signals the both-present case so callers can surface it
// (via Input.StdinIgnored) for main.go to log at Debug level after slog
// is configured -- logging decisions live one layer up, per plan.md's
// Architecture section.
func readTrace(fileArg string, stdin io.Reader, stdinIsPiped bool) (raw string, truncated bool, stdinIgnored bool, err error) {
	var source io.Reader

	switch {
	case fileArg != "":
		f, openErr := os.Open(fileArg)
		if openErr != nil {
			return "", false, false, fmt.Errorf("reading trace file %q: %w", fileArg, openErr)
		}
		defer func() {
			// Read-only file; nothing actionable if Close fails here.
			_ = f.Close()
		}()

		source = f
		stdinIgnored = stdinIsPiped

	case stdinIsPiped:
		source = stdin

	default:
		// Neither a file argument nor piped stdin -- per spec
		// requirement 6, this must fail immediately, never block
		// waiting on an interactive terminal.
		return "", false, false, fmt.Errorf("no input: stdin not piped and no file argument given")
	}

	buf, readErr := io.ReadAll(io.LimitReader(source, boundedReadCapBytes))
	if readErr != nil {
		if fileArg != "" {
			return "", false, false, fmt.Errorf("reading trace file %q: %w", fileArg, readErr)
		}
		return "", false, false, fmt.Errorf("reading trace from stdin: %w", readErr)
	}

	s := string(buf)
	if strings.TrimSpace(s) == "" {
		return "", false, stdinIgnored, fmt.Errorf("input is empty after reading")
	}

	out, wasTruncated := contract.TruncateRawInput(s)
	return out, wasTruncated, stdinIgnored, nil
}

// validateOutput checks the --output path (FR17, Q4) before any input is
// read or pipeline work begins: the path's parent directory must exist
// and be a directory, and the path must not resolve to the same file as
// fileArg. The same-file check is skipped when fileArg is "" (input is
// coming from stdin, so there's no input file to collide with) and when
// either path doesn't exist yet -- os.SameFile can only compare files
// that already exist, and a not-yet-existing --output path can't be the
// same file as anything. An existing file at outputPath that isn't
// fileArg is always fine here; per FR17 it's silently overwritten later,
// not validateOutput's concern.
//
// An empty outputPath (no --output given) always returns nil immediately
// -- matches Input.Output's own "empty means no --output" semantics, and
// keeps this a no-op pass-through for the two binaries/invocations that
// never set it.
//
// Like readTrace, this is a pure function: no logging, no os.Exit. It is
// not wired into ParseAll/ParseFixedLang yet -- that's T006/T007.
func validateOutput(outputPath, fileArg string) error {
	if outputPath == "" {
		return nil
	}

	parent := filepath.Dir(outputPath)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return fmt.Errorf("--output path %q: parent directory %q does not exist or is not accessible: %w", outputPath, parent, err)
	}
	if !parentInfo.IsDir() {
		return fmt.Errorf("--output path %q: parent path %q is not a directory", outputPath, parent)
	}

	if fileArg != "" {
		outInfo, outErr := os.Stat(outputPath)
		inInfo, inErr := os.Stat(fileArg)
		if outErr == nil && inErr == nil && os.SameFile(outInfo, inInfo) {
			return fmt.Errorf("--output path %q is the same file as the input file %q", outputPath, fileArg)
		}
	}

	return nil
}

package cli

import (
	"fmt"
	"io"
	"os"
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
// stdinIgnored signals the both-present case so the caller (ParseAll /
// ParseFixedLang) can log it at Debug level -- logging decisions live one
// layer up, per plan.md's Architecture section.
func readTrace(fileArg string, stdin io.Reader, stdinIsPiped bool) (raw string, truncated bool, stdinIgnored bool, err error) {
	var source io.Reader

	switch {
	case fileArg != "":
		f, openErr := os.Open(fileArg)
		if openErr != nil {
			return "", false, false, fmt.Errorf("reading trace file %s: %w", fileArg, openErr)
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
			return "", false, false, fmt.Errorf("reading trace file %s: %w", fileArg, readErr)
		}
		return "", false, false, fmt.Errorf("reading trace from stdin: %w", readErr)
	}

	if strings.TrimSpace(string(buf)) == "" {
		return "", false, stdinIgnored, fmt.Errorf("input is empty after reading")
	}

	out, wasTruncated := contract.TruncateRawInput(string(buf))
	return out, wasTruncated, stdinIgnored, nil
}

package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// deliver writes bundle to outputPath if it's non-empty, else to stdout
// (FR13). Per FR15, a single trailing newline is appended if bundle
// doesn't already end with one -- applied identically to both the
// stdout and file cases, so a bundle's on-disk and on-stdout content
// never differ except in destination.
//
// When outputPath is non-empty, it is opened only here, at call time
// (os.O_WRONLY|os.O_CREATE|os.O_TRUNC) -- never earlier, and never by
// validateOutput (T004), which only checks the path looks usable
// without ever opening it. An existing file at outputPath is silently
// overwritten (FR17, Q4) -- validateOutput already rejected the one
// case that isn't a plain overwrite (outputPath resolving to the same
// file as the input argument).
//
// deliver is called only after rendering succeeds (Run, T014) -- a
// deliver failure is the caller's exit-1 case (FR16), with the
// clipboard write attempted only when deliver returns nil.
func deliver(bundle, outputPath string, stdout io.Writer) error {
	out := bundle
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	if outputPath == "" {
		if _, err := io.WriteString(stdout, out); err != nil {
			return fmt.Errorf("writing bundle to stdout: %w", err)
		}
		return nil
	}

	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("opening --output path %q: %w", outputPath, err)
	}

	if _, err := io.WriteString(f, out); err != nil {
		f.Close() //nolint:errcheck // the write error is what's reported; a Close error here would be secondary
		return fmt.Errorf("writing bundle to %q: %w", outputPath, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", outputPath, err)
	}

	return nil
}

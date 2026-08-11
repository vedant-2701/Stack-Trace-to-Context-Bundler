// Command stack-trace-bundler-java is the Java-only stack-trace-bundler
// CLI entrypoint -- unlike cmd/all, it never registers --lang and always
// fixes the language to "java". For now (002a), it only reads and
// validates CLI input -- see specs/002a-cli-input-handling/spec.md. The
// real detect -> parse -> render -> clipboard pipeline, and real stdout
// bundle output, land in 002b; until then this only logs a stub
// summary/dump to stderr per spec requirement 13, and never writes to
// stdout.
package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/vedant-2701/stack-trace-bundler/internal/cli"
)

func main() {
	input, verbosity, err := cli.ParseFixedLang(os.Args[1:], os.Stdin, stdinIsPiped(), "java")
	if err != nil {
		// slog's default (unconfigured) level is Info, which still
		// shows Error -- no need to configure the handler before this
		// call, unlike the Debug-level calls below.
		slog.Error(err.Error())
		os.Exit(2)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: cli.LogLevel(verbosity),
	})))

	// Must come after the slog.SetDefault above, not before -- see
	// Input.StdinIgnored's doc comment and plan.md's Architecture
	// section (T006c) for why logging this during cli.ParseFixedLang
	// itself, before the level was known, silently dropped the message
	// under every verbosity setting.
	if input.StdinIgnored {
		slog.Debug("stdin ignored: file argument took precedence")
	}

	slog.Info("parsed input",
		"bytes", len(input.RawText),
		"lang", input.LangHint,
		"format", input.Format,
	)

	if verbosity >= 2 {
		dump, err := json.MarshalIndent(input, "", "  ")
		if err != nil {
			// Marshaling our own known-shape struct should never fail.
			// This is a Debug-only diagnostic dump, not a functional
			// path, so log and continue rather than treating it as
			// fatal.
			slog.Debug("marshaling input dump failed", "error", err)
		} else {
			slog.Debug("input dump", "input", string(dump))
		}
	}
}

// stdinIsPiped reports whether stdin is piped (a file or another
// process) rather than an interactive terminal. Computed once here via
// os.Stdin.Stat() and passed into cli.ParseFixedLang as a plain bool --
// keeps internal/cli's core logic pure and testable with fake readers,
// no real TTY needed in tests, per plan.md's Architecture section.
func stdinIsPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		// Can't determine TTY state -- fail safe toward "not piped" so
		// cli.ParseFixedLang's no-input fast-fail (spec requirement 6)
		// kicks in, rather than risking a block on a read that might
		// never complete.
		return false
	}
	return stat.Mode()&os.ModeCharDevice == 0
}

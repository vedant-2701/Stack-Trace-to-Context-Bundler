// Command stack-trace-bundler-java is the Java-only stack-trace-bundler
// CLI entrypoint -- unlike cmd/all, it never registers --lang and always
// fixes the language to "java". Java support itself doesn't exist yet
// (005a/005b): this binary's Candidates stays nil, which is exactly what
// makes cli.Run's candidate-selection step fail fast on every normal
// invocation with "Java is not supported yet" (FR10) -- -h/--help still
// works. All real work lives in cli.Run; this file only wires this
// binary's BinaryConfig and hands off. See
// specs/002b-pipeline-wiring/spec.md.
package main

import (
	"os"
	"path/filepath"

	"github.com/vedant-2701/stack-trace-bundler/internal/cli"
)

func main() {
	progName := filepath.Base(os.Args[0])

	cfg := cli.BinaryConfig{
		ProgramName: progName,
		FixedLang:   "java",
		// Candidates and Resolvers both stay nil until 005a/005b exist --
		// nil Candidates is exactly what makes selectCandidates' FR10
		// fail-fast fire on every normal invocation (see package doc
		// comment above); nil Resolvers has nothing to look up for a
		// language that never successfully detects.
		Examples: []cli.Example{
			{
				Command:     progName + " trace.txt",
				Description: "bundle a Java stack trace from a file",
			},
			{
				Command:     "cat trace.txt | " + progName,
				Description: "bundle a Java stack trace piped on stdin",
			},
		},
	}

	os.Exit(cli.Run(os.Args[1:], os.Stdin, stdinIsPiped(), cfg, os.Stdout, os.Stderr))
}

// stdinIsPiped reports whether stdin is piped (a file or another
// process) rather than an interactive terminal. Computed once here via
// os.Stdin.Stat() and passed into cli.Run as a plain bool -- keeps
// internal/cli's core logic pure and testable with fake readers, no real
// TTY needed in tests, per plan.md's Architecture section.
func stdinIsPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		// Can't determine TTY state -- fail safe toward "not piped" so
		// readTrace's no-input fast-fail kicks in, rather than risking a
		// block on a read that might never complete.
		return false
	}
	return stat.Mode()&os.ModeCharDevice == 0
}

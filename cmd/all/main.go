// Command stack-trace-bundler-all is the stack-trace-bundler CLI
// entrypoint that auto-detects JavaScript vs. TypeScript Node.js traces
// (Java support lands with 005a/005b; until then --lang=java and plain
// auto-detection both fail fast via cli.Run's candidate-selection step).
// All real work -- detect, parse, code context, dependency resolution,
// render, deliver, clipboard -- lives in cli.Run; this file only wires
// this binary's BinaryConfig and hands off. See
// specs/002b-pipeline-wiring/spec.md.
package main

import (
	"os"
	"path/filepath"

	"github.com/vedant-2701/stack-trace-bundler/internal/cli"
	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	tsdeps "github.com/vedant-2701/stack-trace-bundler/internal/dependency/typescript"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
	tsparser "github.com/vedant-2701/stack-trace-bundler/internal/parser/typescript"
)

func main() {
	progName := filepath.Base(os.Args[0])

	cfg := cli.BinaryConfig{
		ProgramName: progName,
		// FixedLang stays "" -- this binary registers --lang and lets
		// cli.Run/ParseAll auto-detect by default (FR3); --lang=typescript
		// narrows candidates without bypassing Detect (Q10), and
		// --lang=java fails fast via candidate selection finding no java
		// parser in Candidates below (FR10), with no cmd/java-specific
		// code needed here at all.
		Candidates: []parser.LanguageParser{
			tsparser.NewJavaScriptParser(),
			tsparser.NewTypeScriptParser(),
		},
		// Both languages share one dependency resolver (006b) -- npm/yarn
		// dependency resolution doesn't differ between hand-written JS and
		// TypeScript-compiled-to-JS traces, only parsing does.
		Resolvers: map[contract.Language]cli.DependencyResolver{
			contract.LanguageJavaScript: tsdeps.ResolveDependencies,
			contract.LanguageTypeScript: tsdeps.ResolveDependencies,
		},
		Examples: []cli.Example{
			{
				Command:     progName + " trace.txt",
				Description: "bundle a trace from a file, auto-detecting the language",
			},
			{
				Command:     "cat trace.txt | " + progName,
				Description: "bundle a trace piped on stdin",
			},
			{
				Command:     progName + " --lang=typescript trace.txt",
				Description: "hint the language instead of relying on auto-detection",
			},
			{
				Command:     progName + " --format=json -o bundle.json trace.txt",
				Description: "write a JSON bundle to a file instead of stdout",
			},
			{
				Command:     progName + " --no-clipboard trace.txt",
				Description: "skip copying the bundle to the clipboard",
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

// Command stack-trace-bundler-typescript is the TypeScript/JavaScript-only
// stack-trace-bundler CLI entrypoint -- unlike cmd/all, it never
// registers --lang and always fixes the language to "typescript"
// (which narrows candidates to both the JavaScript and TypeScript
// parsers; Detect still decides between them, FR3/Q10). All real work
// -- detect, parse, code context, dependency resolution, render,
// deliver, clipboard -- lives in cli.Run; this file only wires this
// binary's BinaryConfig and hands off. See
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
		FixedLang:   "typescript",
		// Same two parsers and shared resolver as cmd/all -- the fixed
		// "typescript" hint narrows candidates to both of them (a
		// TypeScript-compiled-to-JS trace can legitimately be detected as
		// javascript), so both must be registered here too.
		Candidates: []parser.LanguageParser{
			tsparser.NewJavaScriptParser(),
			tsparser.NewTypeScriptParser(),
		},
		Resolvers: map[contract.Language]cli.DependencyResolver{
			contract.LanguageJavaScript: tsdeps.ResolveDependencies,
			contract.LanguageTypeScript: tsdeps.ResolveDependencies,
		},
		// No --lang example: this binary never registers that flag, so
		// an example using it would fail the help_test.go round-trip
		// check (and mislead anyone reading --help).
		Examples: []cli.Example{
			{
				Command:     progName + " trace.txt",
				Description: "bundle a TypeScript/JavaScript trace from a file",
			},
			{
				Command:     "cat trace.txt | " + progName,
				Description: "bundle a trace piped on stdin",
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

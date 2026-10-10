package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// Example is one usage example shown in --help output (FR19-20, Q14).
// Command is the full invocation as a user would type it (including the
// program name, e.g. "stba --format=json trace.txt"); Description is a
// short line explaining what it does.
type Example struct {
	Command     string
	Description string
}

// exitCodeDocs is the Go-side source of truth for the exit-code table
// in --help output (FR19-20, Q14). usageText renders every entry here;
// help_test.go checks that every code number appears in the rendered
// output. CONVENTIONS.md carries its own prose version of this same
// table -- keeping the two in sync is a manual step (this feature's
// last acceptance criterion, T021), not an automated one; nothing here
// generates or checks against CONVENTIONS.md (plan.md's Risks section).
var exitCodeDocs = []struct {
	Code    int
	Meaning string
}{
	{0, "success (including --help)"},
	{1, "unexpected/internal error"},
	{2, "usage error (bad flags, no input, unsupported language, bad --output path)"},
	{3, "input could not be parsed"},
	{4, "source language ambiguous or undetected"},
}

// usageText renders the full --help output for one binary: a synopsis
// naming programName, the flag listing from fs.FlagUsages(), usage
// examples, and the exit-code table from exitCodeDocs (FR19-20).
//
// fs must already have its flags registered (via registerFlags) but not
// yet parsed -- it is not built inside usageText itself, so the caller
// (Run, T010) passes in the exact same *pflag.FlagSet it registered for
// parsing. This is deliberate: FlagUsages() can never drift from what's
// actually accepted, because it's rendering the same FlagSet a second
// invocation would parse against, not a second one built independently
// for display purposes only.
//
// examples may be nil or empty -- the Examples section is omitted
// entirely in that case, rather than printed as an empty heading.
func usageText(programName string, fs *pflag.FlagSet, examples []Example) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s -- bundle a stack trace and surrounding code context for an LLM\n\n", programName)
	fmt.Fprintf(&b, "Usage:\n  %s [flags] [file]\n\n", programName)
	fmt.Fprint(&b, "Reads a stack trace from the given file, or from stdin if no file is\ngiven and stdin is piped.\n\n")

	fmt.Fprint(&b, "Flags:\n")
	fmt.Fprint(&b, fs.FlagUsages())
	b.WriteString("\n")

	if len(examples) > 0 {
		fmt.Fprint(&b, "Examples:\n")
		for _, ex := range examples {
			fmt.Fprintf(&b, "  %s\n", ex.Command)
			if ex.Description != "" {
				fmt.Fprintf(&b, "      %s\n", ex.Description)
			}
		}
		b.WriteString("\n")
	}

	fmt.Fprint(&b, "Exit codes:\n")
	for _, d := range exitCodeDocs {
		fmt.Fprintf(&b, "  %d  %s\n", d.Code, d.Meaning)
	}

	return b.String()
}

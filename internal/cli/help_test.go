package cli

import (
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

func TestUsageText_LangFlagPresenceMatchesFixedLang(t *testing.T) {
	tests := []struct {
		name      string
		fixedLang string
		wantLang  bool
	}{
		{name: "cmd/all-shaped (fixedLang empty) includes --lang", fixedLang: "", wantLang: true},
		{name: "cmd/java-shaped (fixedLang set) excludes --lang", fixedLang: "java", wantLang: false},
		{name: "cmd/typescript-shaped (fixedLang set) excludes --lang", fixedLang: "typescript", wantLang: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("stack-trace-bundler", pflag.ContinueOnError)
			registerFlags(fs, tc.fixedLang)

			text := usageText("stba", fs, nil)

			got := strings.Contains(text, "--lang")
			if got != tc.wantLang {
				t.Errorf("usageText output contains --lang = %v, want %v", got, tc.wantLang)
			}
		})
	}
}

func TestUsageText_ExitCodesAllPresent(t *testing.T) {
	fs := pflag.NewFlagSet("stack-trace-bundler", pflag.ContinueOnError)
	registerFlags(fs, "")

	text := usageText("stba", fs, nil)

	for _, d := range exitCodeDocs {
		if !strings.Contains(text, strconv.Itoa(d.Code)) {
			t.Errorf("usageText output missing exit code %d", d.Code)
		}
	}
}

// cmdAllExamples mirrors cmd/all/main.go's real Examples exactly (T016)
// -- "stba" standing in for the runtime-derived program name, matching
// this file's other tests. What this test guards is the round-trip
// mechanism itself (an Example.Command's flags parse cleanly through
// ParseAll), not these specific commands -- the anti-drift check from
// Q14: if cmd/all/main.go's Examples ever drift out of sync with what
// registerFlags actually accepts, this is what catches it.
var cmdAllExamples = []Example{
	{Command: "stba trace.txt", Description: "bundle a trace from a file, auto-detecting the language"},
	{Command: "cat trace.txt | stba", Description: "bundle a trace piped on stdin"},
	{Command: "stba --lang=typescript trace.txt", Description: "hint the language instead of relying on auto-detection"},
	{Command: "stba --format=json -o bundle.json trace.txt", Description: "write a JSON bundle to a file instead of stdout"},
	{Command: "stba --no-clipboard trace.txt", Description: "skip copying the bundle to the clipboard"},
}

func TestUsageText_ExamplesRoundTripThroughParseAll(t *testing.T) {
	for _, ex := range cmdAllExamples {
		t.Run(ex.Command, func(t *testing.T) {
			fields := strings.Fields(ex.Command)
			if len(fields) == 0 {
				t.Fatalf("example command %q has no fields", ex.Command)
			}
			if fields[0] != "stba" {
				// Not a plain "<program> [flags] [file]" invocation (the
				// piped "cat trace.txt | stba" example) -- exampleArgs'
				// "drop the leading program-name token" rule doesn't apply
				// to a shell pipeline, and this example has no flags to
				// round-trip in the first place.
				t.Skip("not a plain invocation, nothing to round-trip")
			}

			args := exampleArgs(t, ex.Command)

			// The fixture file named in these examples doesn't exist on
			// disk, so this always returns a "reading trace file" error
			// from readTrace -- that's expected and fine, it's proof the
			// flags parsed cleanly and readTrace was actually reached. A
			// "parsing flags" or "invalid --" error is what this test
			// actually guards against: an Example whose flags don't
			// round-trip through the FlagSet registerFlags builds. (The
			// --lang=typescript example passes nil candidates here, so it
			// actually surfaces a languageUnsupportedError instead of
			// reaching readTrace -- harmless for this test, since that
			// error message is neither of the two substrings checked
			// below; it's still proof --lang=typescript itself parsed.)
			_, _, err := ParseAll(args, strings.NewReader(""), false, nil)
			if err != nil && (strings.Contains(err.Error(), "parsing flags") || strings.Contains(err.Error(), "invalid --")) {
				t.Errorf("example %q failed to parse: %v", ex.Command, err)
			}
		})
	}
}

// exampleArgs splits an Example.Command like "stba --format=json
// trace.txt" into the args ParseAll/ParseFixedLang expect, dropping the
// leading program-name token.
func exampleArgs(t *testing.T, command string) []string {
	t.Helper()

	fields := strings.Fields(command)
	if len(fields) == 0 {
		t.Fatalf("example command %q has no fields", command)
	}

	return fields[1:]
}

// cmdJavaExamples mirrors cmd/java/main.go's real Examples exactly
// (T017). Every one of these is expected to surface
// languageUnsupportedError ("Java is not supported yet") when run
// through ParseFixedLang below -- cmd/java's Candidates is nil until
// 005a/005b exist, by design (FR10) -- so this test only guards the
// flags themselves parsing cleanly, same as cmdAllExamples' round-trip
// test.
var cmdJavaExamples = []Example{
	{Command: "stba trace.txt", Description: "bundle a Java stack trace from a file"},
	{Command: "cat trace.txt | stba", Description: "bundle a Java stack trace piped on stdin"},
}

func TestUsageText_CmdJavaExamplesRoundTripThroughParseFixedLang(t *testing.T) {
	for _, ex := range cmdJavaExamples {
		t.Run(ex.Command, func(t *testing.T) {
			fields := strings.Fields(ex.Command)
			if len(fields) == 0 {
				t.Fatalf("example command %q has no fields", ex.Command)
			}
			if fields[0] != "stba" {
				t.Skip("not a plain invocation, nothing to round-trip")
			}

			args := exampleArgs(t, ex.Command)

			// Expected to fail with languageUnsupportedError ("Java is not
			// supported yet") every time -- cmd/java has no java parser
			// registered, by design, until 005a/005b exist. A "parsing
			// flags" or "invalid --" error is what this test actually
			// guards against.
			_, _, err := ParseFixedLang(args, strings.NewReader(""), false, "java", nil)
			if err != nil && (strings.Contains(err.Error(), "parsing flags") || strings.Contains(err.Error(), "invalid --")) {
				t.Errorf("example %q failed to parse: %v", ex.Command, err)
			}
		})
	}
}

// cmdTypescriptExamples mirrors cmd/typescript/main.go's real Examples
// exactly (T018) -- notably with no --lang example, since that binary
// never registers the flag.
var cmdTypescriptExamples = []Example{
	{Command: "stba trace.txt", Description: "bundle a TypeScript/JavaScript trace from a file"},
	{Command: "cat trace.txt | stba", Description: "bundle a trace piped on stdin"},
	{Command: "stba --format=json -o bundle.json trace.txt", Description: "write a JSON bundle to a file instead of stdout"},
	{Command: "stba --no-clipboard trace.txt", Description: "skip copying the bundle to the clipboard"},
}

func TestUsageText_CmdTypescriptExamplesRoundTripThroughParseFixedLang(t *testing.T) {
	for _, ex := range cmdTypescriptExamples {
		t.Run(ex.Command, func(t *testing.T) {
			fields := strings.Fields(ex.Command)
			if len(fields) == 0 {
				t.Fatalf("example command %q has no fields", ex.Command)
			}
			if fields[0] != "stba" {
				t.Skip("not a plain invocation, nothing to round-trip")
			}

			args := exampleArgs(t, ex.Command)

			// A registered typescript fake candidate lets ParseFixedLang's
			// fail-fast pass, so these actually reach validateOutput and
			// readTrace (which then fails on the nonexistent fixture file --
			// expected, and not one of the two substrings checked below).
			candidates := []parser.LanguageParser{fakeParser{lang: contract.LanguageTypeScript}}
			_, _, err := ParseFixedLang(args, strings.NewReader(""), false, "typescript", candidates)
			if err != nil && (strings.Contains(err.Error(), "parsing flags") || strings.Contains(err.Error(), "invalid --")) {
				t.Errorf("example %q failed to parse: %v", ex.Command, err)
			}
		})
	}
}

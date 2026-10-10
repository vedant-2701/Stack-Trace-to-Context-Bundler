package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		want          string
		wantErrSubstr string
	}{
		{name: "empty defaults to markdown", in: "", want: "markdown"},
		{name: "explicit markdown", in: "markdown", want: "markdown"},
		{name: "explicit json", in: "json", want: "json"},
		{name: "invalid value", in: "yaml", wantErrSubstr: `invalid --format value "yaml"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateFormat(tc.in)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("err = %q, want it to contain %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("got = %q, want %q", got, tc.want)
			}
		})
	}
}

// buildTestTraceArgs assembles the args slice for one TestParseAll/
// TestParseFixedLang case: optionally writes a temp fixture file and
// appends its path to args (before or after, per fileBeforeFlags), and
// optionally appends an --output flag pointing at a nonexistent parent
// directory. Factored out of both tests' t.Run bodies to keep gocyclo's
// per-function complexity down -- this and assertParseResult are the
// only two things either loop body does now besides calling the
// function under test.
func buildTestTraceArgs(t *testing.T, args []string, fileContent *string, fileBeforeFlags, outputNonexistentParent bool) []string {
	t.Helper()

	out := append([]string{}, args...)
	if fileContent != nil {
		dir := t.TempDir()
		path := filepath.Join(dir, "trace.txt")
		if err := os.WriteFile(path, []byte(*fileContent), 0o600); err != nil {
			t.Fatalf("writing test fixture file: %v", err)
		}
		if fileBeforeFlags {
			out = append([]string{path}, args...)
		} else {
			out = append(out, path)
		}
	}
	if outputNonexistentParent {
		out = append(out, "--output="+filepath.Join(t.TempDir(), "does-not-exist", "out.md"))
	}

	return out
}

// assertParseResult is the shared assertion body for TestParseAll and
// TestParseFixedLang: on a non-empty wantErrSubstr, checks the error
// message and that verbosity is 0; otherwise checks LangHint, Format (if
// wantFormat is non-empty), verbosity, and StdinIgnored against got.
func assertParseResult(t *testing.T, got Input, verbosity int, err error, wantErrSubstr, wantLangHint, wantFormat string, wantVerbosity int, wantStdinIgnored bool) {
	t.Helper()

	if wantErrSubstr != "" {
		if err == nil {
			t.Fatalf("err = nil, want error containing %q", wantErrSubstr)
		}
		if !strings.Contains(err.Error(), wantErrSubstr) {
			t.Errorf("err = %q, want it to contain %q", err.Error(), wantErrSubstr)
		}
		if verbosity != 0 {
			t.Errorf("verbosity = %d, want 0 on any error return", verbosity)
		}
		return
	}

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.LangHint != wantLangHint {
		t.Errorf("LangHint = %q, want %q", got.LangHint, wantLangHint)
	}
	if wantFormat != "" && got.Format != wantFormat {
		t.Errorf("Format = %q, want %q", got.Format, wantFormat)
	}
	if verbosity != wantVerbosity {
		t.Errorf("verbosity = %d, want %d", verbosity, wantVerbosity)
	}
	if got.StdinIgnored != wantStdinIgnored {
		t.Errorf("StdinIgnored = %v, want %v", got.StdinIgnored, wantStdinIgnored)
	}
}

func TestParseAll(t *testing.T) {
	tests := []struct {
		name string

		args []string
		// fileContent, if non-nil, creates a temp file with this content.
		fileContent *string
		// fileBeforeFlags controls append order when fileContent is set:
		// false (default) appends the file path after tc.args (flags
		// first, the historical/always-worked ordering); true puts the
		// file path first and tc.args after it (T007a: the ordering that
		// was silently broken under stdlib flag.Parse until the switch
		// to pflag).
		fileBeforeFlags bool
		stdinContent    string
		stdinIsPiped    bool
		// candidates is passed straight through to ParseAll -- nil is fine
		// for any case whose lang hint is "" (selectCandidates("", nil)
		// returns nil unchanged, no error) or that fails before
		// selectCandidates ever runs.
		candidates []parser.LanguageParser
		// outputNonexistentParent appends --output=<a per-subtest
		// t.TempDir()>/does-not-exist/out.md to args, for the T006
		// candidate-fail-fast-vs-validateOutput ordering cases.
		outputNonexistentParent bool

		wantErrSubstr    string
		wantLangHint     string
		wantFormat       string
		wantVerbosity    int
		wantStdinIgnored bool
	}{
		{
			name:         "valid stdin, defaults",
			stdinContent: "java.lang.NullPointerException\n",
			stdinIsPiped: true,
			wantLangHint: "",
			wantFormat:   "markdown",
		},
		{
			name:         "valid file, explicit lang and format",
			args:         []string{"--lang=java", "--format=json"},
			fileContent:  strPtr("java.lang.NullPointerException\n"),
			candidates:   []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint: "java",
			wantFormat:   "json",
		},
		{
			name:             "both present: file wins, StdinIgnored propagates",
			fileContent:      strPtr("from file\n"),
			stdinContent:     "from stdin\n",
			stdinIsPiped:     true,
			wantStdinIgnored: true,
		},
		{
			name:          "invalid lang",
			args:          []string{"--lang=cobol"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantErrSubstr: `invalid --lang value "cobol"`,
		},
		{
			name:          "invalid format",
			args:          []string{"--format=yaml"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantErrSubstr: `invalid --format value "yaml"`,
		},
		{
			name:          "no input surfaces readTrace's error",
			stdinIsPiped:  false,
			wantErrSubstr: "no input",
		},
		{
			name:          "combined failure: invalid flag wins over no-input",
			args:          []string{"--lang=cobol"},
			stdinIsPiped:  false,
			wantErrSubstr: `invalid --lang value "cobol"`,
		},
		{
			name:          "unknown flag rejected by flag.Parse",
			args:          []string{"--nope=1"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantErrSubstr: "parsing flags",
		},
		{
			name:          "-v sets verbosity 1",
			args:          []string{"-v"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantVerbosity: 1,
		},
		{
			name:          "-vv sets verbosity 2 (pflag POSIX clustering)",
			args:          []string{"-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantVerbosity: 2,
		},
		{
			name:          "-v and -vv as separate tokens: counts accumulate (1+2=3), still Debug via LogLevel",
			args:          []string{"-v", "-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantVerbosity: 3,
		},
		{
			name:          "-v after --lang: order independent",
			args:          []string{"--lang=java", "-v"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			candidates:    []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:  "java",
			wantVerbosity: 1,
		},
		{
			name:          "-v before --lang: order independent",
			args:          []string{"-v", "--lang=java"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			candidates:    []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:  "java",
			wantVerbosity: 1,
		},
		{
			name:          "positional arg named -v after -- terminator is not read as the flag",
			args:          []string{"--", "-v"},
			stdinIsPiped:  false,
			wantErrSubstr: `reading trace file "-v"`,
		},
		{
			name:            "T007a: flags AFTER the positional file argument (silently broken under stdlib flag)",
			fileContent:     strPtr("java.lang.NullPointerException\n"),
			args:            []string{"--lang=java", "-vv"},
			fileBeforeFlags: true,
			candidates:      []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:    "java",
			wantVerbosity:   2,
		},
		{
			name:          "--format=bogus --lang=java: format error wins over language error",
			args:          []string{"--format=bogus", "--lang=java"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantErrSubstr: `invalid --format value "bogus"`,
		},
		{
			name:                    "java hint fails fast before --output is validated (no java parser registered)",
			args:                    []string{"--lang=java"},
			stdinContent:            "trace\n",
			stdinIsPiped:            true,
			outputNonexistentParent: true,
			wantErrSubstr:           "Java is not supported yet",
		},
		{
			name:                    "typescript hint (supported) reaches --output validation once candidate fail-fast doesn't fire",
			args:                    []string{"--lang=typescript"},
			stdinContent:            "trace\n",
			stdinIsPiped:            true,
			candidates:              []parser.LanguageParser{fakeParser{lang: contract.LanguageJavaScript}, fakeParser{lang: contract.LanguageTypeScript}},
			outputNonexistentParent: true,
			wantErrSubstr:           "does not exist or is not accessible",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := buildTestTraceArgs(t, tc.args, tc.fileContent, tc.fileBeforeFlags, tc.outputNonexistentParent)

			stdin := strings.NewReader(tc.stdinContent)

			got, verbosity, err := ParseAll(args, stdin, tc.stdinIsPiped, tc.candidates)

			assertParseResult(t, got, verbosity, err, tc.wantErrSubstr, tc.wantLangHint, tc.wantFormat, tc.wantVerbosity, tc.wantStdinIgnored)
		})
	}
}

// recordingReader wraps an io.Reader and records whether Read was ever
// called on it, so a test can assert that a fail-fast path never reached
// readTrace.
type recordingReader struct {
	r          io.Reader
	readCalled bool
}

func (r *recordingReader) Read(p []byte) (int, error) {
	r.readCalled = true
	return r.r.Read(p)
}

func TestParseAll_UnsupportedHintFailsFastBeforeReadTrace(t *testing.T) {
	stdin := &recordingReader{r: strings.NewReader("trace\n")}

	_, verbosity, err := ParseAll([]string{"--lang=java"}, stdin, true, nil)

	if err == nil {
		t.Fatal("err = nil, want a languageUnsupportedError")
	}
	if !strings.Contains(err.Error(), "Java is not supported yet") {
		t.Errorf("err = %q, want it to contain %q", err.Error(), "Java is not supported yet")
	}
	if verbosity != 0 {
		t.Errorf("verbosity = %d, want 0 on any error return", verbosity)
	}
	if stdin.readCalled {
		t.Error("stdin.Read was called -- readTrace ran despite the unsupported-hint fail-fast, want it never reached")
	}
}

func TestParseFixedLang(t *testing.T) {
	tests := []struct {
		name string

		lang            string
		args            []string
		fileContent     *string
		fileBeforeFlags bool
		stdinContent    string
		stdinIsPiped    bool
		// candidates is passed straight through to ParseFixedLang -- see
		// TestParseAll's candidates field doc comment for when nil is fine.
		candidates []parser.LanguageParser
		// outputNonexistentParent appends --output=<a per-subtest
		// t.TempDir()>/does-not-exist/out.md to args, for the T007
		// candidate-fail-fast-vs-validateOutput ordering cases.
		outputNonexistentParent bool

		wantErrSubstr    string
		wantLangHint     string
		wantFormat       string
		wantVerbosity    int
		wantStdinIgnored bool
	}{
		{
			name:         "java, defaults",
			lang:         "java",
			stdinContent: "java.lang.NullPointerException\n",
			stdinIsPiped: true,
			candidates:   []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint: "java",
			wantFormat:   "markdown",
		},
		{
			name:         "typescript, explicit format",
			lang:         "typescript",
			args:         []string{"--format=json"},
			fileContent:  strPtr("TypeError: x is not a function\n"),
			candidates:   []parser.LanguageParser{fakeParser{lang: contract.LanguageTypeScript}},
			wantLangHint: "typescript",
			wantFormat:   "json",
		},
		{
			name:             "both present: file wins, StdinIgnored propagates",
			lang:             "java",
			fileContent:      strPtr("from file\n"),
			stdinContent:     "from stdin\n",
			stdinIsPiped:     true,
			candidates:       []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:     "java",
			wantStdinIgnored: true,
		},
		{
			// Also covers T007's acceptance criterion: with lang="java" and
			// no java parser in candidates (nil here), an invalid --format
			// value still reports the format error, not
			// languageUnsupportedError -- validateFormat runs before
			// selectCandidates, so it never gets there.
			name:          "invalid format",
			lang:          "java",
			args:          []string{"--format=yaml"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantErrSubstr: `invalid --format value "yaml"`,
		},
		{
			name:          "--lang rejected: not registered on this FlagSet",
			lang:          "java",
			args:          []string{"--lang=java"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantErrSubstr: "parsing flags",
		},
		{
			name:          "no input surfaces readTrace's error",
			lang:          "typescript",
			stdinIsPiped:  false,
			candidates:    []parser.LanguageParser{fakeParser{lang: contract.LanguageTypeScript}},
			wantErrSubstr: "no input",
		},
		{
			name:          "-v sets verbosity 1",
			lang:          "java",
			args:          []string{"-v"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			candidates:    []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:  "java",
			wantVerbosity: 1,
		},
		{
			name:          "-vv sets verbosity 2 (pflag POSIX clustering)",
			lang:          "java",
			args:          []string{"-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			candidates:    []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:  "java",
			wantVerbosity: 2,
		},
		{
			name:          "-v and -vv as separate tokens: counts accumulate (1+2=3)",
			lang:          "java",
			args:          []string{"-v", "-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			candidates:    []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:  "java",
			wantVerbosity: 3,
		},
		{
			name:          "-v after --format: order independent",
			lang:          "java",
			args:          []string{"--format=json", "-v"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			candidates:    []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:  "java",
			wantFormat:    "json",
			wantVerbosity: 1,
		},
		{
			name:            "T007a: flags AFTER the positional file argument (silently broken under stdlib flag)",
			lang:            "java",
			fileContent:     strPtr("java.lang.NullPointerException\n"),
			args:            []string{"--format=json", "-vv"},
			fileBeforeFlags: true,
			candidates:      []parser.LanguageParser{fakeParser{lang: contract.LanguageJava}},
			wantLangHint:    "java",
			wantFormat:      "json",
			wantVerbosity:   2,
		},
		{
			name:                    "java hint fails fast before --output is validated (no java parser registered)",
			lang:                    "java",
			stdinContent:            "trace\n",
			stdinIsPiped:            true,
			outputNonexistentParent: true,
			wantErrSubstr:           "Java is not supported yet",
		},
		{
			name:                    "typescript hint (supported) reaches --output validation once candidate fail-fast doesn't fire",
			lang:                    "typescript",
			stdinContent:            "trace\n",
			stdinIsPiped:            true,
			candidates:              []parser.LanguageParser{fakeParser{lang: contract.LanguageJavaScript}, fakeParser{lang: contract.LanguageTypeScript}},
			outputNonexistentParent: true,
			wantErrSubstr:           "does not exist or is not accessible",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := buildTestTraceArgs(t, tc.args, tc.fileContent, tc.fileBeforeFlags, tc.outputNonexistentParent)

			stdin := strings.NewReader(tc.stdinContent)

			got, verbosity, err := ParseFixedLang(args, stdin, tc.stdinIsPiped, tc.lang, tc.candidates)

			assertParseResult(t, got, verbosity, err, tc.wantErrSubstr, tc.wantLangHint, tc.wantFormat, tc.wantVerbosity, tc.wantStdinIgnored)
		})
	}
}

func TestParseFixedLang_UnsupportedHintFailsFastBeforeReadTrace(t *testing.T) {
	stdin := &recordingReader{r: strings.NewReader("trace\n")}

	_, verbosity, err := ParseFixedLang([]string{}, stdin, true, "java", nil)

	if err == nil {
		t.Fatal("err = nil, want a languageUnsupportedError")
	}
	if !strings.Contains(err.Error(), "Java is not supported yet") {
		t.Errorf("err = %q, want it to contain %q", err.Error(), "Java is not supported yet")
	}
	if verbosity != 0 {
		t.Errorf("verbosity = %d, want 0 on any error return", verbosity)
	}
	if stdin.readCalled {
		t.Error("stdin.Read was called -- readTrace ran despite the unsupported-hint fail-fast, want it never reached")
	}
}

func TestParseFixedLang_InvalidLangPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("ParseFixedLang did not panic on an invalid lang argument")
		}
	}()

	_, _, _ = ParseFixedLang(nil, strings.NewReader("trace\n"), true, "cobol", nil)
}

func TestValidateLang(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		want          string
		wantErrSubstr string
	}{
		{name: "empty defers to auto-detection", in: "", want: ""},
		{name: "java", in: "java", want: "java"},
		{name: "typescript", in: "typescript", want: "typescript"},
		{name: "invalid value", in: "cobol", wantErrSubstr: `invalid --lang value "cobol"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateLang(tc.in)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("err = %q, want it to contain %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("got = %q, want %q", got, tc.want)
			}
		})
	}
}

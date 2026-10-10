package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

// configurableFakeParser is a parser.LanguageParser whose Detect result
// and Parse error are set per test case, unlike candidates_test.go's
// fakeParser (Detect always false, Parse always errors) -- needed here
// to exercise DetectLanguage's match/no-match/ambiguous outcomes (T011)
// and Parse's success/ErrUnparseable/plain-error outcomes (T012). A zero
// value (detect: false, parseErr: nil) means "never matches"; Parse
// itself never checks detect, so a test giving a non-matching parser a
// parseErr would simply never have it read -- DetectLanguage only
// returns parsers whose Detect returned true.
type configurableFakeParser struct {
	lang     contract.Language
	detect   bool
	parseErr error
}

func (f configurableFakeParser) Language() contract.Language { return f.lang }

func (f configurableFakeParser) Detect(_ string) bool { return f.detect }

func (f configurableFakeParser) Parse(_ context.Context, _ string) ([]contract.ExceptionNode, contract.Runtime, error) {
	return nil, contract.Runtime{}, f.parseErr
}

// TestMain stubs getwd/buildGitMetadata/buildCodeContexts package-wide
// with safe, deterministic, zero-I/O defaults for every test in this
// file. None of this package's tests exercise real git/filesystem
// behavior through them -- that's codecontext's own test suite's job
// (CONVENTIONS.md: anything that shells out is tested behind a fake).
// Without this, every success-path test that reaches Run's step 6+
// (T013) would call the real os.Getwd/git against this repo's actual
// working directory. A test that needs non-default behavior (a
// failure, or asserting a call was made, like TestRun_Getwd) overrides
// further per-test, with t.Cleanup restoring exactly these defaults
// (captured as "the current value" at that point, not the real
// os.Getwd/codecontext functions) once it's done.
func TestMain(m *testing.M) {
	getwd = func() (string, error) { return "/fake/workdir", nil }
	buildGitMetadata = func(_ context.Context, _ string) *contract.GitMetadata { return nil }
	buildCodeContexts = func(_ context.Context, _ []contract.ExceptionNode, _ contract.Language, _ *contract.GitMetadata) []contract.CodeContext {
		return []contract.CodeContext{}
	}
	clipboardWrite = func(_ context.Context, _ string) error { return nil }

	os.Exit(m.Run())
}

// Exit-code coverage matrix (FR21, filled in by T015) -- every exit
// code and sub-reason spec.md's FR21 lists, with which test below
// covers it through Run specifically, not only via a smaller unit test
// from an earlier task (e.g. ParseAll/ParseFixedLang's own parse_test.go
// cases, or validateOutput's/selectCandidates' own direct unit tests):
//
//	0  success                            TestRun_DetectLanguage (exit-0 cases), TestRun_Parse ("successful Parse"), TestRun_Delivery
//	0  --help                             TestRun_Help
//	0  failed clipboard copy after success TestRun_Delivery ("clipboard write fails after a successful deliver")
//	1  os.Getwd failure                   TestRun_Getwd
//	1  output write failure               TestRun_Delivery_Output ("deliver failure")
//	1  Parse error, not ErrUnparseable    TestRun_Parse ("plain Parse error")
//	2  flag errors                        TestRun_FlagError
//	2  no input                           TestRun_ExitCodeMatrix_UsageErrors ("no input")
//	2  empty input                        TestRun_ExitCodeMatrix_UsageErrors ("empty input")
//	2  unsupported language hint          TestRun_ExitCodeMatrix_UsageErrors ("unsupported language hint")
//	2  invalid --output path              TestRun_ExitCodeMatrix_UsageErrors ("invalid --output path")
//	2  --output equal to the input file   TestRun_ExitCodeMatrix_UsageErrors ("--output equal to")
//	3  unparseable input (ErrUnparseable) TestRun_Parse ("ErrUnparseable-wrapping Parse error")
//	4  language undetected (ErrNoMatch)   TestRun_DetectLanguage ("no match" cases)
//	4  language ambiguous (ErrAmbiguous)  TestRun_DetectLanguage ("both match")
func TestRun_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "-h alone", args: []string{"-h"}},
		{name: "--help alone", args: []string{"--help"}},
		{
			// FR19/Q14 ordering: a bad flag VALUE (not an unknown flag)
			// doesn't block -h -- pflag's own ErrHelp fires as soon as it
			// reaches -h in the arg list, before validateLang ever runs,
			// since --lang=cobol is a perfectly well-formed string flag
			// at the pflag layer; only our own post-Parse validation
			// would reject "cobol", and Parse never gets that far.
			name: "--lang=cobol -h: bad flag value doesn't block -h",
			args: []string{"--lang=cobol", "-h"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			cfg := BinaryConfig{ProgramName: "stba"}

			got := Run(tc.args, strings.NewReader(""), false, cfg, &stdout, &stderr)

			if got != 0 {
				t.Errorf("exit code = %d, want 0", got)
			}
			if !strings.Contains(stderr.String(), "stba") {
				t.Errorf("stderr = %q, want it to contain the program name %q", stderr.String(), "stba")
			}
			if stdout.String() != "" {
				t.Errorf("stdout = %q, want empty -- help goes to stderr only", stdout.String())
			}
		})
	}
}

func TestRun_FlagError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"--nope=1"}},
		{
			// FR19/Q14 ordering, the other direction from TestRun_Help's
			// --lang=cobol case: an unknown flag encountered BEFORE -h
			// stops pflag's left-to-right Parse there -- ErrHelp is never
			// reached, so this is exit 2, not exit 0.
			name: "unknown flag before -h: -h never reached",
			args: []string{"--bogus", "-h"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			cfg := BinaryConfig{ProgramName: "stba"}

			got := Run(tc.args, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

			if got != 2 {
				t.Errorf("exit code = %d, want 2", got)
			}
		})
	}
}

func TestRun_StdinIgnoredLogsAtDebug(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.txt")
	if err := os.WriteFile(path, []byte("from file\n"), 0o600); err != nil {
		t.Fatalf("writing test fixture file: %v", err)
	}

	var stdout, stderr strings.Builder
	cfg := BinaryConfig{
		ProgramName: "stba",
		// A single always-matching, never-erroring-on-Parse candidate:
		// hint is "" here (no --lang), so this needs to be non-empty and
		// unambiguous purely to get Run past DetectLanguage and Parse
		// without a real language or parse result mattering to this
		// test.
		Candidates: []parser.LanguageParser{
			configurableFakeParser{lang: contract.LanguageJavaScript, detect: true},
		},
	}

	// -vv reaches Debug level -- the "stdin ignored" line is Debug, so
	// it's filtered out below that (see LogLevel's doc comment).
	got := Run([]string{"-vv", path}, strings.NewReader("from stdin\n"), true, cfg, &stdout, &stderr)

	if got != 0 {
		t.Fatalf("exit code = %d, want 0 -- stderr: %s", got, stderr.String())
	}
	if !strings.Contains(stderr.String(), "stdin ignored") {
		t.Errorf("stderr = %q, want it to contain the Debug stdin-ignored line", stderr.String())
	}
}

func TestRun_DetectLanguage(t *testing.T) {
	jsNoMatch := configurableFakeParser{lang: contract.LanguageJavaScript, detect: false}
	tsNoMatch := configurableFakeParser{lang: contract.LanguageTypeScript, detect: false}
	jsMatch := configurableFakeParser{lang: contract.LanguageJavaScript, detect: true}
	tsMatch := configurableFakeParser{lang: contract.LanguageTypeScript, detect: true}

	tests := []struct {
		name string

		args         []string
		stdinContent string
		candidates   []parser.LanguageParser

		wantExit           int
		wantStderrContains []string
		wantStderrMissing  []string
	}{
		{
			name:         "no match, multi-line input: exit 4, message names both checked languages, no tip",
			stdinContent: "line one\nline two\n",
			candidates:   []parser.LanguageParser{jsNoMatch, tsNoMatch},
			wantExit:     4,
			wantStderrContains: []string{
				"could not detect the language of the input",
				"javascript", "typescript",
			},
			wantStderrMissing: []string{"paste it directly into your LLM"},
		},
		{
			name:         "no match, single-line input: exit 4, message includes the FR12 tip",
			stdinContent: "TypeError: fetch failed",
			candidates:   []parser.LanguageParser{jsNoMatch, tsNoMatch},
			wantExit:     4,
			wantStderrContains: []string{
				"could not detect the language of the input",
				"paste it directly into your LLM instead",
				"Error.stackTraceLimit may have been set to 0",
			},
		},
		{
			name:         "both match: exit 4, ambiguous message, no tip regardless of line count",
			stdinContent: "single line, but matches two parsers",
			candidates:   []parser.LanguageParser{jsMatch, tsMatch},
			wantExit:     4,
			wantStderrContains: []string{
				"the input matched more than one language",
				"javascript", "typescript",
			},
			wantStderrMissing: []string{"paste it directly into your LLM"},
		},
		{
			name:         "hint differs from the one matching parser: both Info lines fire",
			args:         []string{"--lang=typescript", "-v"},
			stdinContent: "matches only the js parser",
			candidates:   []parser.LanguageParser{jsMatch},
			wantExit:     0,
			wantStderrContains: []string{
				"language resolved", "language=javascript",
				"detected language differs from --lang hint",
				"hint=typescript", "detected=javascript",
			},
		},
		{
			name:         "hint matches the detected language: only language resolved fires",
			args:         []string{"--lang=typescript", "-v"},
			stdinContent: "matches only the ts parser",
			candidates:   []parser.LanguageParser{tsMatch},
			wantExit:     0,
			wantStderrContains: []string{
				"language resolved", "language=typescript",
			},
			wantStderrMissing: []string{"detected language differs from --lang hint"},
		},
		{
			name:         "no hint at all: successful detection, no mismatch line",
			args:         []string{"-v"},
			stdinContent: "matches only the js parser",
			candidates:   []parser.LanguageParser{jsMatch},
			wantExit:     0,
			wantStderrContains: []string{
				"language resolved", "language=javascript",
			},
			wantStderrMissing: []string{"detected language differs from --lang hint"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			cfg := BinaryConfig{ProgramName: "stba", Candidates: tc.candidates}

			got := Run(tc.args, strings.NewReader(tc.stdinContent), true, cfg, &stdout, &stderr)

			if got != tc.wantExit {
				t.Errorf("exit code = %d, want %d -- stderr: %s", got, tc.wantExit, stderr.String())
			}
			for _, want := range tc.wantStderrContains {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr missing %q -- stderr: %s", want, stderr.String())
				}
			}
			for _, notWant := range tc.wantStderrMissing {
				if strings.Contains(stderr.String(), notWant) {
					t.Errorf("stderr unexpectedly contains %q -- stderr: %s", notWant, stderr.String())
				}
			}
		})
	}
}

func TestRun_Parse(t *testing.T) {
	tests := []struct {
		name     string
		parseErr error
		wantExit int
	}{
		{
			name:     "ErrUnparseable-wrapping Parse error: exit 3",
			parseErr: fmt.Errorf("malformed frame on line 2: %w", parser.ErrUnparseable),
			wantExit: 3,
		},
		{
			name:     "plain Parse error: exit 1",
			parseErr: errors.New("boom"),
			wantExit: 1,
		},
		{
			name:     "successful Parse: proceeds (exit 0, T013 takes over from here)",
			parseErr: nil,
			wantExit: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			cfg := BinaryConfig{
				ProgramName: "stba",
				Candidates: []parser.LanguageParser{
					configurableFakeParser{lang: contract.LanguageJavaScript, detect: true, parseErr: tc.parseErr},
				},
			}

			got := Run(nil, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

			if got != tc.wantExit {
				t.Errorf("exit code = %d, want %d -- stderr: %s", got, tc.wantExit, stderr.String())
			}
		})
	}
}

func TestRun_Getwd(t *testing.T) {
	origGetwd := getwd
	origBuildGitMetadata := buildGitMetadata
	origBuildCodeContexts := buildCodeContexts
	t.Cleanup(func() {
		getwd = origGetwd
		buildGitMetadata = origBuildGitMetadata
		buildCodeContexts = origBuildCodeContexts
	})

	var gitMetaCalled, codeContextsCalled bool
	getwd = func() (string, error) { return "", errors.New("getwd failed") }
	buildGitMetadata = func(_ context.Context, _ string) *contract.GitMetadata {
		gitMetaCalled = true
		return nil
	}
	buildCodeContexts = func(_ context.Context, _ []contract.ExceptionNode, _ contract.Language, _ *contract.GitMetadata) []contract.CodeContext {
		codeContextsCalled = true
		return nil
	}

	var stdout, stderr strings.Builder
	cfg := BinaryConfig{
		ProgramName: "stba",
		Candidates: []parser.LanguageParser{
			configurableFakeParser{lang: contract.LanguageJavaScript, detect: true},
		},
	}

	got := Run(nil, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

	if got != 1 {
		t.Errorf("exit code = %d, want 1 -- stderr: %s", got, stderr.String())
	}
	if gitMetaCalled {
		t.Error("buildGitMetadata was called despite getwd failing -- want it never reached")
	}
	if codeContextsCalled {
		t.Error("buildCodeContexts was called despite getwd failing -- want it never reached")
	}
}

func TestAssembleBundle(t *testing.T) {
	chain := []contract.ExceptionNode{
		{
			ClassName: "TypeError",
			Message:   "x is not a function",
			Frames: []contract.Frame{
				{Index: 0, FilePath: "/repo/app.js", MethodName: "main", LineNumber: 10, Bucket: contract.BucketOwn},
			},
		},
	}
	rt := contract.Runtime{Name: "node", VersionSource: contract.VersionSourceUnknown}
	codeContexts := []contract.CodeContext{
		{
			FrameRef: contract.FrameRef{ChainIndex: 0, FrameIndex: 0},
			FilePath: "/repo/app.js",
			Language: contract.LanguageJavaScript,
			Status:   contract.StatusOK,
		},
	}
	gitMeta := &contract.GitMetadata{CurrentCommit: "abc123", Branch: "main"}
	deps := &contract.Dependencies{
		ManifestFile: contract.ManifestFilePackageJSON,
		Direct:       map[string]string{"left-pad": "^1.0.0"},
	}

	input := Input{
		RawText:           "TypeError: x is not a function\n    at main (/repo/app.js:10:1)",
		RawInputTruncated: false,
	}

	got := assembleBundle(input, contract.LanguageJavaScript, chain, rt, codeContexts, gitMeta, deps)

	if got.SchemaVersion != contract.SchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", got.SchemaVersion, contract.SchemaVersion)
	}
	if got.Language != contract.LanguageJavaScript {
		t.Errorf("Language = %q, want %q", got.Language, contract.LanguageJavaScript)
	}
	if got.OS != contract.OS(runtime.GOOS) {
		t.Errorf("OS = %q, want %q", got.OS, contract.OS(runtime.GOOS))
	}
	if got.RawInput != input.RawText {
		t.Errorf("RawInput = %q, want %q", got.RawInput, input.RawText)
	}
	if got.RawInputTruncated != input.RawInputTruncated {
		t.Errorf("RawInputTruncated = %v, want %v", got.RawInputTruncated, input.RawInputTruncated)
	}
	wantFingerprint := contract.ComputeFingerprint(chain)
	if got.Fingerprint != wantFingerprint {
		t.Errorf("Fingerprint = %q, want %q", got.Fingerprint, wantFingerprint)
	}
	if got.Runtime != rt {
		t.Errorf("Runtime = %+v, want %+v", got.Runtime, rt)
	}
	if len(got.Chain) != 1 {
		t.Fatalf("len(Chain) = %d, want 1", len(got.Chain))
	}
	if len(got.CodeContexts) != 1 {
		t.Fatalf("len(CodeContexts) = %d, want 1", len(got.CodeContexts))
	}
	if got.GitMetadata != gitMeta {
		t.Error("GitMetadata pointer not passed through unchanged")
	}
	if got.Dependencies != deps {
		t.Error("Dependencies pointer not passed through unchanged")
	}
}

func TestAssembleBundle_NilGitMetadataAndDependencies(t *testing.T) {
	chain := []contract.ExceptionNode{{ClassName: "Error", Message: "boom"}}
	rt := contract.Runtime{Name: "node", VersionSource: contract.VersionSourceUnknown}

	got := assembleBundle(Input{}, contract.LanguageJavaScript, chain, rt, nil, nil, nil)

	if got.GitMetadata != nil {
		t.Errorf("GitMetadata = %+v, want nil", got.GitMetadata)
	}
	if got.Dependencies != nil {
		t.Errorf("Dependencies = %+v, want nil", got.Dependencies)
	}
}

// withClipboardWrite overrides the package-level clipboardWrite var for
// the duration of one test, restoring TestMain's default via t.Cleanup,
// and returns a pointer to a bool the caller can check afterward to see
// whether it was called.
func withClipboardWrite(t *testing.T, err error) *bool {
	t.Helper()

	orig := clipboardWrite
	t.Cleanup(func() { clipboardWrite = orig })

	called := new(bool)
	clipboardWrite = func(_ context.Context, _ string) error {
		*called = true
		return err
	}
	return called
}

func TestRun_Delivery(t *testing.T) {
	tests := []struct {
		name string

		noClipboard  bool
		clipboardErr error

		wantExit            int
		wantClipboardCalled bool
		wantStderrContains  []string
	}{
		{
			name:                "stdout delivery, clipboard copied",
			wantExit:            0,
			wantClipboardCalled: true,
			wantStderrContains:  []string{"bundle delivered", "destination=stdout", "clipboard=copied"},
		},
		{
			name:                "NoClipboard: true -- clipboard writer never called",
			noClipboard:         true,
			wantExit:            0,
			wantClipboardCalled: false,
			wantStderrContains:  []string{"bundle delivered", "destination=stdout", "clipboard=skipped"},
		},
		{
			name:                "clipboard write fails after a successful deliver: exit 0, Warn logged",
			clipboardErr:        errors.New("boom"),
			wantExit:            0,
			wantClipboardCalled: true,
			wantStderrContains: []string{
				"could not copy the bundle to the clipboard",
				"pass --no-clipboard to skip copying",
				"bundle delivered", "clipboard=failed",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clipboardCalled := withClipboardWrite(t, tc.clipboardErr)

			var stdout, stderr strings.Builder
			cfg := BinaryConfig{
				ProgramName: "stba",
				Candidates: []parser.LanguageParser{
					configurableFakeParser{lang: contract.LanguageJavaScript, detect: true},
				},
			}

			args := []string{"-v"}
			if tc.noClipboard {
				args = append(args, "--no-clipboard")
			}

			got := Run(args, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

			if got != tc.wantExit {
				t.Errorf("exit code = %d, want %d -- stderr: %s", got, tc.wantExit, stderr.String())
			}
			if *clipboardCalled != tc.wantClipboardCalled {
				t.Errorf("clipboard called = %v, want %v", *clipboardCalled, tc.wantClipboardCalled)
			}
			if stdout.Len() == 0 {
				t.Error("stdout is empty, want the rendered bundle")
			}
			for _, want := range tc.wantStderrContains {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr missing %q -- stderr: %s", want, stderr.String())
				}
			}
		})
	}
}

func TestRun_Delivery_Output(t *testing.T) {
	t.Run("writes to --output, nothing to stdout, clipboard still written", func(t *testing.T) {
		clipboardCalled := withClipboardWrite(t, nil)

		dir := t.TempDir()
		outPath := filepath.Join(dir, "out.md")

		var stdout, stderr strings.Builder
		cfg := BinaryConfig{
			ProgramName: "stba",
			Candidates: []parser.LanguageParser{
				configurableFakeParser{lang: contract.LanguageJavaScript, detect: true},
			},
		}

		got := Run([]string{"-v", "--output=" + outPath}, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

		if got != 0 {
			t.Fatalf("exit code = %d, want 0 -- stderr: %s", got, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout = %q, want empty -- bundle should go to --output only", stdout.String())
		}
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("reading --output file: %v", err)
		}
		if len(content) == 0 {
			t.Error("--output file is empty, want the rendered bundle")
		}
		if !*clipboardCalled {
			t.Error("clipboard writer was not called, want it called (NoClipboard not set)")
		}
		if !strings.Contains(stderr.String(), "destination="+outPath) {
			t.Errorf("stderr missing destination=%s -- stderr: %s", outPath, stderr.String())
		}
	})

	t.Run("deliver failure (--output path is an existing directory): exit 1, clipboard never called", func(t *testing.T) {
		// validateOutput only rejects a nonexistent/non-directory PARENT
		// and the same-file-as-input case -- outPath itself being an
		// existing directory passes validation (its parent exists, and
		// it's not the input file, since input is piped stdin here), so
		// this reaches deliver's real os.OpenFile call, which fails
		// opening a directory for writing (EISDIR). This is deliberately
		// NOT a nonexistent-parent path -- that fails at validateOutput
		// (exit 2) long before Run would ever reach deliver.
		clipboardCalled := withClipboardWrite(t, nil)

		dir := t.TempDir()
		outPath := filepath.Join(dir, "a-directory")
		if err := os.Mkdir(outPath, 0o755); err != nil {
			t.Fatalf("creating directory fixture: %v", err)
		}

		var stdout, stderr strings.Builder
		cfg := BinaryConfig{
			ProgramName: "stba",
			Candidates: []parser.LanguageParser{
				configurableFakeParser{lang: contract.LanguageJavaScript, detect: true},
			},
		}

		got := Run([]string{"--output=" + outPath}, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

		if got != 1 {
			t.Errorf("exit code = %d, want 1 -- stderr: %s", got, stderr.String())
		}
		if *clipboardCalled {
			t.Error("clipboard writer was called despite deliver failing -- want it never reached")
		}
	})
}

// TestRun_ExitCodeMatrix_UsageErrors fills in the FR21 exit-2
// sub-reasons not otherwise exercised through Run specifically by the
// other TestRun_* tests above (see the matrix comment above
// TestRun_Help): no input, empty input, an unsupported language hint
// (via Run's own full dispatch, not just ParseAll's parse_test.go
// cases), an invalid --output path, and --output equal to the input
// file.
func TestRun_ExitCodeMatrix_UsageErrors(t *testing.T) {
	cfg := func() BinaryConfig { return BinaryConfig{ProgramName: "stba"} }

	t.Run("no input: neither a file argument nor piped stdin", func(t *testing.T) {
		var stdout, stderr strings.Builder

		got := Run(nil, strings.NewReader(""), false, cfg(), &stdout, &stderr)

		if got != 2 {
			t.Errorf("exit code = %d, want 2 -- stderr: %s", got, stderr.String())
		}
	})

	t.Run("empty input: piped stdin is whitespace-only", func(t *testing.T) {
		var stdout, stderr strings.Builder

		got := Run(nil, strings.NewReader("   \n\t\n"), true, cfg(), &stdout, &stderr)

		if got != 2 {
			t.Errorf("exit code = %d, want 2 -- stderr: %s", got, stderr.String())
		}
	})

	t.Run("unsupported language hint via Run (no java parser registered)", func(t *testing.T) {
		var stdout, stderr strings.Builder

		// cfg() leaves Candidates nil -- ParseAll's own fail-fast (inside
		// its selectCandidates call, before readTrace) is what actually
		// fires here, surfacing as a plain parse error to Run's
		// err != nil branch; resolveLanguage's defensive re-check is
		// never reached on this path.
		got := Run([]string{"--lang=java"}, strings.NewReader("trace\n"), true, cfg(), &stdout, &stderr)

		if got != 2 {
			t.Errorf("exit code = %d, want 2 -- stderr: %s", got, stderr.String())
		}
	})

	t.Run("invalid --output path: nonexistent parent directory", func(t *testing.T) {
		dir := t.TempDir()
		outPath := filepath.Join(dir, "does-not-exist", "out.md")

		var stdout, stderr strings.Builder

		got := Run([]string{"--output=" + outPath}, strings.NewReader("trace\n"), true, cfg(), &stdout, &stderr)

		if got != 2 {
			t.Errorf("exit code = %d, want 2 -- stderr: %s", got, stderr.String())
		}
	})

	t.Run("--output equal to the input file argument", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "trace.txt")
		if err := os.WriteFile(path, []byte("trace\n"), 0o600); err != nil {
			t.Fatalf("writing test fixture file: %v", err)
		}

		var stdout, stderr strings.Builder

		got := Run([]string{"--output=" + path, path}, strings.NewReader(""), false, cfg(), &stdout, &stderr)

		if got != 2 {
			t.Errorf("exit code = %d, want 2 -- stderr: %s", got, stderr.String())
		}
	})
}

// TestRun_ClipboardMatchesStdoutExceptTrailingNewline pins spec.md FR15 at
// the Run level: the clipboard gets the rendered text unchanged, and
// stdout gets the same text plus one trailing newline only when it
// doesn't already end with one. For JSON that means the clipboard copy is
// one byte shorter than stdout (008 Q5 stays intact for the paste
// payload).
func TestRun_ClipboardMatchesStdoutExceptTrailingNewline(t *testing.T) {
	for _, format := range []string{"markdown", "json"} {
		t.Run(format, func(t *testing.T) {
			var clip string
			orig := clipboardWrite
			t.Cleanup(func() { clipboardWrite = orig })
			clipboardWrite = func(_ context.Context, text string) error {
				clip = text
				return nil
			}

			var stdout, stderr strings.Builder
			cfg := BinaryConfig{
				ProgramName: "stba",
				Candidates: []parser.LanguageParser{
					configurableFakeParser{lang: contract.LanguageJavaScript, detect: true},
				},
			}

			got := Run([]string{"--format=" + format}, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

			if got != 0 {
				t.Fatalf("exit code = %d, want 0 -- stderr: %s", got, stderr.String())
			}
			if clip == "" {
				t.Fatal("clipboard writer was not called, or was called with empty text")
			}

			want := clip
			if !strings.HasSuffix(want, "\n") {
				want += "\n"
			}
			if stdout.String() != want {
				t.Errorf("stdout = %q, want the clipboard text plus at most one trailing newline (%q)", stdout.String(), want)
			}
			if format == "json" && strings.HasSuffix(clip, "\n") {
				t.Error("JSON clipboard text ends with a newline, want it byte-for-byte without one (008 Q5)")
			}
		})
	}
}

// TestRun_FailureLeavesFilesUntouched covers two spec.md acceptance
// criteria that held only by construction until now: a pipeline failure
// never touches an existing --output file (FR17: the file is opened only
// after rendering succeeds), and --output equal to the input file is
// rejected before anything is written.
func TestRun_FailureLeavesFilesUntouched(t *testing.T) {
	t.Run("pipeline failure (exit 4) leaves an existing --output file untouched", func(t *testing.T) {
		dir := t.TempDir()
		outPath := filepath.Join(dir, "out.md")
		if err := os.WriteFile(outPath, []byte("old\n"), 0o600); err != nil {
			t.Fatalf("writing pre-existing output fixture: %v", err)
		}

		var stdout, stderr strings.Builder
		cfg := BinaryConfig{
			ProgramName: "stba",
			Candidates: []parser.LanguageParser{
				configurableFakeParser{lang: contract.LanguageJavaScript, detect: false},
			},
		}

		got := Run([]string{"--output=" + outPath}, strings.NewReader("trace\n"), true, cfg, &stdout, &stderr)

		if got != 4 {
			t.Fatalf("exit code = %d, want 4 -- stderr: %s", got, stderr.String())
		}
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("reading output file: %v", err)
		}
		if string(content) != "old\n" {
			t.Errorf("output file = %q, want it untouched (%q)", string(content), "old\n")
		}
	})

	t.Run("--output equal to the input file leaves the input unchanged", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "trace.txt")
		if err := os.WriteFile(path, []byte("trace\n"), 0o600); err != nil {
			t.Fatalf("writing test fixture file: %v", err)
		}

		var stdout, stderr strings.Builder
		cfg := BinaryConfig{ProgramName: "stba"}

		got := Run([]string{"--output=" + path, path}, strings.NewReader(""), false, cfg, &stdout, &stderr)

		if got != 2 {
			t.Fatalf("exit code = %d, want 2 -- stderr: %s", got, stderr.String())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading input file: %v", err)
		}
		if string(content) != "trace\n" {
			t.Errorf("input file = %q, want it unchanged (%q)", string(content), "trace\n")
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
	})
}

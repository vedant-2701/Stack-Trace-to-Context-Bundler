package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestParseAll(t *testing.T) {
	tests := []struct {
		name string

		args []string
		// fileContent, if non-nil, creates a temp file with this content
		// and appends its path as the positional arg.
		fileContent  *string
		stdinContent string
		stdinIsPiped bool

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
			name:          "-vv sets verbosity 2",
			args:          []string{"-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantVerbosity: 2,
		},
		{
			name:          "-v and -vv together: max wins, not sum",
			args:          []string{"-v", "-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantVerbosity: 2,
		},
		{
			name:          "-v after --lang: order independent",
			args:          []string{"--lang=java", "-v"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantLangHint:  "java",
			wantVerbosity: 1,
		},
		{
			name:          "-v before --lang: order independent",
			args:          []string{"-v", "--lang=java"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantLangHint:  "java",
			wantVerbosity: 1,
		},
		{
			name:          "positional arg named -v after -- terminator is not read as the flag",
			args:          []string{"--", "-v"},
			stdinIsPiped:  false,
			wantErrSubstr: "reading trace file -v",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{}, tc.args...)
			if tc.fileContent != nil {
				dir := t.TempDir()
				path := filepath.Join(dir, "trace.txt")
				if err := os.WriteFile(path, []byte(*tc.fileContent), 0o600); err != nil {
					t.Fatalf("writing test fixture file: %v", err)
				}
				args = append(args, path)
			}

			stdin := strings.NewReader(tc.stdinContent)

			got, verbosity, err := ParseAll(args, stdin, tc.stdinIsPiped)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("err = %q, want it to contain %q", err.Error(), tc.wantErrSubstr)
				}
				if verbosity != 0 {
					t.Errorf("verbosity = %d, want 0 on any error return", verbosity)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got.LangHint != tc.wantLangHint {
				t.Errorf("LangHint = %q, want %q", got.LangHint, tc.wantLangHint)
			}
			if tc.wantFormat != "" && got.Format != tc.wantFormat {
				t.Errorf("Format = %q, want %q", got.Format, tc.wantFormat)
			}
			if verbosity != tc.wantVerbosity {
				t.Errorf("verbosity = %d, want %d", verbosity, tc.wantVerbosity)
			}
			if got.StdinIgnored != tc.wantStdinIgnored {
				t.Errorf("StdinIgnored = %v, want %v", got.StdinIgnored, tc.wantStdinIgnored)
			}
		})
	}
}

func TestParseFixedLang(t *testing.T) {
	tests := []struct {
		name string

		lang         string
		args         []string
		fileContent  *string
		stdinContent string
		stdinIsPiped bool

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
			wantLangHint: "java",
			wantFormat:   "markdown",
		},
		{
			name:         "typescript, explicit format",
			lang:         "typescript",
			args:         []string{"--format=json"},
			fileContent:  strPtr("TypeError: x is not a function\n"),
			wantLangHint: "typescript",
			wantFormat:   "json",
		},
		{
			name:             "both present: file wins, StdinIgnored propagates",
			lang:             "java",
			fileContent:      strPtr("from file\n"),
			stdinContent:     "from stdin\n",
			stdinIsPiped:     true,
			wantLangHint:     "java",
			wantStdinIgnored: true,
		},
		{
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
			wantErrSubstr: "no input",
		},
		{
			name:          "-v sets verbosity 1",
			lang:          "java",
			args:          []string{"-v"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantLangHint:  "java",
			wantVerbosity: 1,
		},
		{
			name:          "-vv sets verbosity 2",
			lang:          "java",
			args:          []string{"-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantLangHint:  "java",
			wantVerbosity: 2,
		},
		{
			name:          "-v and -vv together: max wins, not sum",
			lang:          "java",
			args:          []string{"-v", "-vv"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantLangHint:  "java",
			wantVerbosity: 2,
		},
		{
			name:          "-v after --format: order independent",
			lang:          "java",
			args:          []string{"--format=json", "-v"},
			stdinContent:  "trace\n",
			stdinIsPiped:  true,
			wantLangHint:  "java",
			wantFormat:    "json",
			wantVerbosity: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{}, tc.args...)
			if tc.fileContent != nil {
				dir := t.TempDir()
				path := filepath.Join(dir, "trace.txt")
				if err := os.WriteFile(path, []byte(*tc.fileContent), 0o600); err != nil {
					t.Fatalf("writing test fixture file: %v", err)
				}
				args = append(args, path)
			}

			stdin := strings.NewReader(tc.stdinContent)

			got, verbosity, err := ParseFixedLang(args, stdin, tc.stdinIsPiped, tc.lang)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("err = %q, want it to contain %q", err.Error(), tc.wantErrSubstr)
				}
				if verbosity != 0 {
					t.Errorf("verbosity = %d, want 0 on any error return", verbosity)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got.LangHint != tc.wantLangHint {
				t.Errorf("LangHint = %q, want %q", got.LangHint, tc.wantLangHint)
			}
			if tc.wantFormat != "" && got.Format != tc.wantFormat {
				t.Errorf("Format = %q, want %q", got.Format, tc.wantFormat)
			}
			if verbosity != tc.wantVerbosity {
				t.Errorf("verbosity = %d, want %d", verbosity, tc.wantVerbosity)
			}
			if got.StdinIgnored != tc.wantStdinIgnored {
				t.Errorf("StdinIgnored = %v, want %v", got.StdinIgnored, tc.wantStdinIgnored)
			}
		})
	}
}

func TestParseFixedLang_InvalidLangPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("ParseFixedLang did not panic on an invalid lang argument")
		}
	}()

	_, _, _ = ParseFixedLang(nil, strings.NewReader("trace\n"), true, "cobol")
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

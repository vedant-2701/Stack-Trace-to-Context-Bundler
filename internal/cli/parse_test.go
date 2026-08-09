package cli

import (
	"bytes"
	"log/slog"
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

		wantErrSubstr string
		wantLangHint  string
		wantFormat    string
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

			got, err := ParseAll(args, stdin, tc.stdinIsPiped)

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
			if got.LangHint != tc.wantLangHint {
				t.Errorf("LangHint = %q, want %q", got.LangHint, tc.wantLangHint)
			}
			if got.Format != tc.wantFormat {
				t.Errorf("Format = %q, want %q", got.Format, tc.wantFormat)
			}
		})
	}
}

func TestParseAll_StdinIgnoredLogging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.txt")
	if err := os.WriteFile(path, []byte("from file\n"), 0o600); err != nil {
		t.Fatalf("writing test fixture file: %v", err)
	}

	t.Run("logs Debug when stdin is ignored", func(t *testing.T) {
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
		defer slog.SetDefault(prev)

		_, err := ParseAll([]string{path}, strings.NewReader("from stdin\n"), true)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if !strings.Contains(buf.String(), "stdin ignored") {
			t.Errorf("log output = %q, want it to contain %q", buf.String(), "stdin ignored")
		}
	})

	t.Run("does not log when stdin is not piped", func(t *testing.T) {
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
		defer slog.SetDefault(prev)

		_, err := ParseAll([]string{path}, strings.NewReader(""), false)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if strings.Contains(buf.String(), "stdin ignored") {
			t.Errorf("log output = %q, want no \"stdin ignored\" message", buf.String())
		}
	})
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

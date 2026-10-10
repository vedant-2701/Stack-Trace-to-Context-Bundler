package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestReadTrace(t *testing.T) {
	tests := []struct {
		name string

		// fileContent, if non-nil, creates a temp file with this
		// content and passes its path as fileArg.
		fileContent *string
		// useNonexistentFile, if true, passes a path that is
		// guaranteed not to exist as fileArg (fileContent must be nil).
		useNonexistentFile bool

		stdinContent string
		stdinIsPiped bool

		wantErrSubstr    string // "" means no error expected
		wantRaw          string // "" means don't check exact content
		wantRawLen       int    // 0 means don't check exact length
		wantTruncated    bool
		wantStdinIgnored bool
	}{
		{
			name:        "file only",
			fileContent: strPtr("java.lang.NullPointerException\n\tat com.example.Foo.bar(Foo.java:10)\n"),
			wantRaw:     "java.lang.NullPointerException\n\tat com.example.Foo.bar(Foo.java:10)\n",
		},
		{
			name:         "stdin only",
			stdinContent: "TypeError: x is not a function\n    at Object.<anonymous> (/app/index.js:5:1)\n",
			stdinIsPiped: true,
			wantRaw:      "TypeError: x is not a function\n    at Object.<anonymous> (/app/index.js:5:1)\n",
		},
		{
			name:             "both present: file wins, stdin ignored",
			fileContent:      strPtr("from file\n"),
			stdinContent:     "from stdin\n",
			stdinIsPiped:     true,
			wantRaw:          "from file\n",
			wantStdinIgnored: true,
		},
		{
			name:          "neither present, TTY: immediate error, no block",
			stdinIsPiped:  false,
			wantErrSubstr: "no input",
		},
		{
			name:               "file not found",
			useNonexistentFile: true,
			wantErrSubstr:      "reading trace file",
		},
		{
			name:          "file empty",
			fileContent:   strPtr(""),
			wantErrSubstr: "input is empty",
		},
		{
			name:          "stdin whitespace only",
			stdinContent:  "   \n\t\n  ",
			stdinIsPiped:  true,
			wantErrSubstr: "input is empty",
		},
		{
			name:          "over 512KB from stdin: truncated, no error",
			stdinContent:  strings.Repeat("a", 600*1024),
			stdinIsPiped:  true,
			wantTruncated: true,
			wantRawLen:    512 * 1024,
		},
		{
			name:          "over 1MB bounded-read cap from stdin: still capped to 512KB, no error",
			stdinContent:  strings.Repeat("b", 2*1024*1024),
			stdinIsPiped:  true,
			wantTruncated: true,
			wantRawLen:    512 * 1024,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fileArg := ""
			switch {
			case tc.fileContent != nil:
				dir := t.TempDir()
				path := filepath.Join(dir, "trace.txt")
				if err := os.WriteFile(path, []byte(*tc.fileContent), 0o600); err != nil {
					t.Fatalf("writing test fixture file: %v", err)
				}
				fileArg = path
			case tc.useNonexistentFile:
				fileArg = filepath.Join(t.TempDir(), "does-not-exist.txt")
			}

			stdin := strings.NewReader(tc.stdinContent)

			raw, truncated, stdinIgnored, err := readTrace(fileArg, stdin, tc.stdinIsPiped)

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
			if tc.wantRaw != "" && raw != tc.wantRaw {
				t.Errorf("raw = %q, want %q", raw, tc.wantRaw)
			}
			if tc.wantRawLen != 0 && len(raw) != tc.wantRawLen {
				t.Errorf("len(raw) = %d, want %d", len(raw), tc.wantRawLen)
			}
			if truncated != tc.wantTruncated {
				t.Errorf("truncated = %v, want %v", truncated, tc.wantTruncated)
			}
			if stdinIgnored != tc.wantStdinIgnored {
				t.Errorf("stdinIgnored = %v, want %v", stdinIgnored, tc.wantStdinIgnored)
			}
		})
	}
}

func TestValidateOutput(t *testing.T) {
	tests := []struct {
		name string

		// Fixture toggles: paths are built fresh per-case from t.TempDir(),
		// selected via these booleans rather than literal strings.
		outputIsSameAsInput bool // outputPath == the fileArg fixture
		outputParentMissing bool // outputPath's parent dir doesn't exist
		outputParentIsFile  bool // outputPath's parent is an existing file
		noOutputPath        bool // outputPath = "" (pass-through case)
		noFileArg           bool // fileArg = "" (stdin case)
		outputAlreadyExists bool // pre-create outputPath as an unrelated file

		wantErrSubstr string // "" means no error expected
	}{
		{
			name:         "empty outputPath is always valid (pass-through)",
			noOutputPath: true,
		},
		{
			name:         "empty outputPath valid even with no fileArg",
			noOutputPath: true,
			noFileArg:    true,
		},
		{
			name:                "parent directory does not exist",
			outputParentMissing: true,
			wantErrSubstr:       "does not exist or is not accessible",
		},
		{
			name:               "parent path exists but is not a directory",
			outputParentIsFile: true,
			wantErrSubstr:      "is not a directory",
		},
		{
			name:                "output path is the same file as the input file",
			outputIsSameAsInput: true,
			wantErrSubstr:       "is the same file as the input file",
		},
		{
			name:      "same-file check skipped when input is stdin (no fileArg)",
			noFileArg: true,
		},
		{
			name:                "existing output file, different from input: valid (overwrite happens later)",
			outputAlreadyExists: true,
		},
		{
			name: "new output file, valid parent, different from input: valid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			fileArg := ""
			if !tc.noFileArg {
				fileArg = filepath.Join(dir, "trace.txt")
				if err := os.WriteFile(fileArg, []byte("trace\n"), 0o600); err != nil {
					t.Fatalf("writing input fixture: %v", err)
				}
			}

			outputPath := ""
			switch {
			case tc.noOutputPath:
				outputPath = ""
			case tc.outputIsSameAsInput:
				outputPath = fileArg
			case tc.outputParentMissing:
				outputPath = filepath.Join(dir, "does-not-exist", "out.md")
			case tc.outputParentIsFile:
				notADir := filepath.Join(dir, "not-a-dir")
				if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
					t.Fatalf("writing parent-as-file fixture: %v", err)
				}
				outputPath = filepath.Join(notADir, "out.md")
			case tc.outputAlreadyExists:
				outputPath = filepath.Join(dir, "existing-out.md")
				if err := os.WriteFile(outputPath, []byte("old\n"), 0o600); err != nil {
					t.Fatalf("writing pre-existing output fixture: %v", err)
				}
			default:
				outputPath = filepath.Join(dir, "out.md")
			}

			err := validateOutput(outputPath, fileArg)

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
		})
	}
}

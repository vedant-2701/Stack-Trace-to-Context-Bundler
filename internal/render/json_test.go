package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// jsonTestBundle is a small, hand-built literal for T001's tests -- not
// one of fixtures_test.go's shared builders. T004+ switches to those
// shared fixtures for the golden-file layer.
func jsonTestBundle() contract.Bundle {
	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		RawInput:          "TypeError: something failed",
		RawInputTruncated: false,
		Fingerprint:       "abc123",
		Runtime: contract.Runtime{
			Name:          "node",
			Version:       "20.11.0",
			VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "TypeError",
				Message:   "processing List<String> & Map<string, number> failed",
				Frames: []contract.Frame{
					{
						Index:      0,
						FilePath:   "/home/dev/project/src/index.ts",
						MethodName: "main",
						LineNumber: 42,
						Bucket:     contract.BucketOwn,
					},
				},
			},
		},
	}
}

func TestJSON_Valid(t *testing.T) {
	got := JSON(jsonTestBundle())
	if !json.Valid([]byte(got)) {
		t.Fatalf("JSON() produced invalid JSON: %s", got)
	}
}

func TestJSON_Compact(t *testing.T) {
	got := JSON(jsonTestBundle())
	if strings.Contains(got, "\n") {
		t.Errorf("JSON() output contains a newline, want compact single-line output: %s", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("JSON() output contains a double-space run, want compact output with no added whitespace: %s", got)
	}
}

func TestJSON_HTMLCharsLiteral(t *testing.T) {
	got := JSON(jsonTestBundle())

	for _, want := range []string{"<String>", "&"} {
		if !strings.Contains(got, want) {
			t.Errorf("JSON() output missing literal %q, want HTML-escaping disabled: %s", want, got)
		}
	}
	for _, notWant := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if strings.Contains(got, notWant) {
			t.Errorf("JSON() output contains %q, want HTML-escaping disabled (SetEscapeHTML(false)): %s", notWant, got)
		}
	}
}

// TestJSON_NoTrailingNewline asserts spec.md's requirement 4 against
// this task's hand-built literal. T004 adds a second case using
// tsBasicBundle once it's in scope, completing plan.md's "run against
// at least tsBasicBundle."
func TestJSON_NoTrailingNewline(t *testing.T) {
	got := JSON(jsonTestBundle())
	if strings.HasSuffix(got, "\n") {
		t.Errorf("JSON() output ends in a newline, want no trailing terminator: %q", got)
	}
}

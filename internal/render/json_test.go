package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

// TestJSON_NoTrailingNewline asserts spec.md's requirement 4: the
// returned string never ends in a '\n' byte. Runs against T001's
// hand-built literal and, now that it's in scope, tsBasicBundle --
// completing plan.md's "run against at least tsBasicBundle."
func TestJSON_NoTrailingNewline(t *testing.T) {
	tests := []struct {
		name   string
		bundle func(t *testing.T) contract.Bundle
	}{
		{"hand-built literal", func(_ *testing.T) contract.Bundle { return jsonTestBundle() }},
		{"tsBasicBundle", tsBasicBundle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSON(tt.bundle(t))
			if strings.HasSuffix(got, "\n") {
				t.Errorf("JSON() output ends in a newline, want no trailing terminator: %q", got)
			}
		})
	}
}

// roundTripTestBundle is a hand-built literal for T002's round-trip test
// -- not the shared fixtures_test.go builders yet (those land in T004+).
// Exercises nested slices (a Chain of 2 ExceptionNodes, each with its own
// Frames), a non-nil GitMetadata, and a non-nil Dependencies with 2+
// Direct/Locked entries -- including one note-only Locked entry (no
// Version) and one version+note entry, the same omitempty combinations
// dependencyStatesBundle exercises later.
func roundTripTestBundle() contract.Bundle {
	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		RawInput:          "TypeError: cannot read property 'foo' of undefined",
		RawInputTruncated: false,
		Fingerprint:       "def456",
		Runtime: contract.Runtime{
			Name:          "node",
			Version:       "20.11.0",
			VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "TypeError",
				Message:   "cannot read property 'foo' of undefined",
				Frames: []contract.Frame{
					{
						Index:      0,
						FilePath:   "/home/dev/project/src/index.ts",
						MethodName: "main",
						LineNumber: 42,
						Bucket:     contract.BucketOwn,
					},
					{
						Index:       1,
						FilePath:    "/home/dev/project/node_modules/chalk/index.js",
						MethodName:  "format",
						LineNumber:  17,
						Bucket:      contract.BucketDependency,
						PackageName: "chalk",
					},
				},
			},
			{
				ClassName:        "Error",
				Message:          "underlying cause",
				ElidedFrameCount: 3,
				Frames: []contract.Frame{
					{
						Index:      0,
						FilePath:   "/home/dev/project/src/db.ts",
						MethodName: "connect",
						LineNumber: 8,
						Bucket:     contract.BucketOwn,
					},
				},
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "a1b2c3d4e5f6",
			Branch:             "main",
			UncommittedChanges: true,
		},
		Dependencies: &contract.Dependencies{
			ManifestFile: contract.ManifestFilePackageJSON,
			Direct: map[string]string{
				"chalk":    "^5.3.0",
				"left-pad": "^1.3.0",
			},
			Locked: map[string]contract.LockedDependency{
				"chalk": {
					Version: "5.3.0",
					Note:    "resolved via top-level lookup, not tied to exact frame path",
				},
				"left-pad": {
					Note: "no local npm cache on this checkout -- expected on a fresh clone",
				},
			},
		},
	}
}

func TestJSON_RoundTrip(t *testing.T) {
	want := roundTripTestBundle()

	var got contract.Bundle
	if err := json.Unmarshal([]byte(JSON(want)), &got); err != nil {
		t.Fatalf("json.Unmarshal(JSON(want)) failed: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("round-trip mismatch:\nwant: %+v\ngot:  %+v", want, got)
	}
}

// --- T004: golden fixture tests ---
//
// Run `go test ./internal/render/... -run TestJSON -update` after a
// deliberate change to JSON()'s output shape, to regenerate the
// affected golden file(s). Hand-review the diff before committing --
// this flag is how a fixture is ever produced or updated, never by
// hand-editing a .golden.json file directly. Reuses markdown_test.go's
// package-level updateGolden flag rather than redeclaring it.
func TestJSON(t *testing.T) {
	tests := []struct {
		name       string
		bundle     func(t *testing.T) contract.Bundle
		goldenPath string
	}{
		{"ts_basic", tsBasicBundle, filepath.Join("testdata", "golden_json", "ts_basic.golden.json")},
		{"no_git_metadata", noGitMetadataBundle, filepath.Join("testdata", "golden_json", "no_git_metadata.golden.json")},
		{"no_dependencies", noDependenciesBundle, filepath.Join("testdata", "golden_json", "no_dependencies.golden.json")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSON(tt.bundle(t))
			assertGoldenJSON(t, got, tt.goldenPath)
		})
	}
}

// assertGoldenJSON compares got byte-for-byte against the fixture at
// path. With -update, it (re)writes the fixture from got instead of
// comparing -- mirrors assertGoldenMarkdown in markdown_test.go.
func assertGoldenJSON(t *testing.T, got, path string) {
	t.Helper()

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v -- run with -update to generate it", path, err)
	}

	if got != string(want) {
		t.Errorf("%s is out of date with JSON()'s current output -- run:\n  go test ./internal/render/... -run TestJSON -update\nto regenerate it, then review the diff", path)
	}
}

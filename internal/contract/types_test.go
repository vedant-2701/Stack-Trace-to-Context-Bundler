package contract

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSchemaVersion(t *testing.T) {
	if SchemaVersion != "1.0.0" {
		t.Errorf("SchemaVersion = %q, want %q", SchemaVersion, "1.0.0")
	}
}

// --- T004: JSON marshal/unmarshal correctness ---
//
// Two different kinds of tests are needed here, not one. A round-trip
// test (marshal, then unmarshal back into a fresh struct, then compare)
// cannot by itself detect a missing or wrongly-applied `omitempty`: an
// omitted zero value and a present zero value unmarshal back to the same
// Go zero value either way, so a struct-to-struct comparison can't tell
// them apart. Catching that requires inspecting the actual marshaled
// JSON bytes for key presence/absence, not just re-unmarshaling them. So
// this file has both: round-trip tests below for general marshal/
// unmarshal correctness (wrong tag name, wrong type, broken nesting), and
// separate raw-JSON key-presence tests further down for the specific
// omitempty behaviors tasks.md/spec.md call out by name.

func roundTrip[T any](t *testing.T, original T) T {
	t.Helper()

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got T
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	return got
}

func TestRoundTrip_Runtime(t *testing.T) {
	original := Runtime{
		Name:          "node",
		Version:       "20.11.0",
		VersionSource: VersionSourceTrace,
	}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_ExceptionNode(t *testing.T) {
	original := ExceptionNode{
		ClassName:        "app.MyError",
		Message:          "boom",
		ElidedFrameCount: 3,
		Frames: []Frame{
			{Index: 0, FilePath: "/src/app.go", MethodName: "DoThing", LineNumber: 10, Bucket: BucketOwn},
		},
	}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_Frame(t *testing.T) {
	original := Frame{
		Index:        2,
		FilePath:     "/src/app.ts",
		ClassName:    "Handler",
		MethodName:   "handle",
		LineNumber:   42,
		ColumnNumber: 7,
		Bucket:       BucketDependency,
		PackageName:  "lodash",
	}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_FrameRef(t *testing.T) {
	original := FrameRef{ChainIndex: 1, FrameIndex: 2}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_Snippet(t *testing.T) {
	original := Snippet{StartLine: 8, EndLine: 12, TargetLine: 10, Code: "func DoThing() {}"}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_CodeContext(t *testing.T) {
	original := CodeContext{
		FrameRef: FrameRef{ChainIndex: 0, FrameIndex: 0},
		FilePath: "/src/app.go",
		Language: LanguageJava,
		Status:   StatusOK,
		Snippet:  Snippet{StartLine: 8, EndLine: 12, TargetLine: 10, Code: "..."},
		Blame: []BlameEntry{
			{StartLine: 8, EndLine: 12, CommitHash: "abc123", Author: "vedant", CommitDate: "2026-08-06T00:00:00Z", Summary: "fix bug"},
		},
	}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_BlameEntry(t *testing.T) {
	original := BlameEntry{
		StartLine:  1,
		EndLine:    5,
		CommitHash: "deadbeef",
		Author:     "vedant",
		CommitDate: "2026-08-06T00:00:00Z",
		Summary:    "initial commit",
	}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_GitMetadata(t *testing.T) {
	original := GitMetadata{CurrentCommit: "abc123", Branch: "main", UncommittedChanges: true}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_Dependencies(t *testing.T) {
	original := Dependencies{
		ManifestFile: ManifestFilePackageJSON,
		Direct:       map[string]string{"lodash": "^4.17.0"},
		Locked: map[string]LockedDependency{
			"lodash": {Version: "4.17.21"},
		},
	}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_LockedDependency(t *testing.T) {
	original := LockedDependency{Version: "1.2.3"}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRoundTrip_Bundle(t *testing.T) {
	original := Bundle{
		SchemaVersion:     SchemaVersion,
		Language:          LanguageJava,
		OS:                OSLinux,
		RawInput:          "java.lang.NullPointerException",
		RawInputTruncated: false,
		Fingerprint:       "abc123def4567890",
		Runtime: Runtime{
			Name:          "jvm",
			VersionSource: VersionSourceUnknown,
		},
		Chain: []ExceptionNode{
			{
				ClassName: "app.MyError",
				Message:   "boom",
				Frames: []Frame{
					{Index: 0, FilePath: "/src/App.java", ClassName: "App", MethodName: "main", LineNumber: 10, Bucket: BucketOwn},
				},
			},
		},
		CodeContexts: []CodeContext{
			{
				FrameRef: FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "/src/App.java",
				Language: LanguageJava,
				Status:   StatusOK,
				Snippet:  Snippet{StartLine: 8, EndLine: 12, TargetLine: 10, Code: "..."},
			},
		},
		GitMetadata: GitMetadata{CurrentCommit: "abc123", Branch: "main"},
		Dependencies: Dependencies{
			ManifestFile: ManifestFilePomXML,
			Direct:       map[string]string{},
			Locked:       map[string]LockedDependency{},
		},
	}
	if got := roundTrip(t, original); !reflect.DeepEqual(original, got) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, original)
	}
}

// --- T004: targeted omitempty key-presence assertions ---

func jsonKeys(t *testing.T, v any) map[string]any {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal into map: %v", err)
	}

	return m
}

func TestFrame_ColumnNumber_OmittedForJava(t *testing.T) {
	f := Frame{
		Index: 0, FilePath: "/src/App.java", MethodName: "main",
		LineNumber: 10, Bucket: BucketOwn, // ColumnNumber left unset: Java never has one
	}

	m := jsonKeys(t, f)
	if _, present := m["columnNumber"]; present {
		t.Errorf(`"columnNumber" key present in JSON for a Java frame with no column number; want it omitted entirely`)
	}
}

func TestFrame_ColumnNumber_PresentForJSTS(t *testing.T) {
	f := Frame{
		Index: 0, FilePath: "/src/app.ts", MethodName: "handle",
		LineNumber: 10, ColumnNumber: 7, Bucket: BucketOwn,
	}

	m := jsonKeys(t, f)
	col, present := m["columnNumber"]
	if !present {
		t.Fatalf(`"columnNumber" key absent in JSON for a JS/TS frame with a real column number; want it present`)
	}
	if col != float64(7) { // JSON numbers decode as float64 into map[string]any
		t.Errorf("columnNumber = %v, want 7", col)
	}
}

func TestLockedDependency_Unresolved_OmitsVersionIncludesNote(t *testing.T) {
	d := LockedDependency{Note: "no local mvn cache on this checkout"}

	m := jsonKeys(t, d)
	if _, present := m["version"]; present {
		t.Errorf(`"version" key present for an unresolved LockedDependency; want it omitted`)
	}
	if _, present := m["note"]; !present {
		t.Errorf(`"note" key absent for an unresolved LockedDependency; want it present`)
	}
}

func TestLockedDependency_Resolved_IncludesVersionOmitsNote(t *testing.T) {
	d := LockedDependency{Version: "4.17.21"}

	m := jsonKeys(t, d)
	if _, present := m["version"]; !present {
		t.Errorf(`"version" key absent for a resolved LockedDependency; want it present`)
	}
	if _, present := m["note"]; present {
		t.Errorf(`"note" key present for a resolved LockedDependency; want it omitted`)
	}
}

func TestBundle_RawInputTruncated_AlwaysPresent(t *testing.T) {
	for _, want := range []bool{true, false} {
		b := minimalBundle()
		b.RawInputTruncated = want

		m := jsonKeys(t, b)
		got, present := m["rawInputTruncated"]
		if !present {
			t.Fatalf("rawInputTruncated = %v: key absent from JSON; want it always present, even when false", want)
		}
		if got != want {
			t.Errorf("rawInputTruncated = %v, want %v", got, want)
		}
	}
}

// minimalBundle is the smallest Bundle that marshals meaningfully for a
// single-field assertion -- used where the test only cares about one
// field's presence, not the full nested shape.
func minimalBundle() Bundle {
	return Bundle{
		SchemaVersion: SchemaVersion,
		Language:      LanguageJava,
		OS:            OSLinux,
		Runtime:       Runtime{VersionSource: VersionSourceUnknown},
		Dependencies: Dependencies{
			Direct: map[string]string{},
			Locked: map[string]LockedDependency{},
		},
	}
}

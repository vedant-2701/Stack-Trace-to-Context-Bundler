package contract

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
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
			ManifestFile: ManifestFilePomXML,
			Direct:       map[string]string{},
			Locked:       map[string]LockedDependency{},
		},
	}
}

// --- T005: golden fixture tests ---
//
// exampleJavaBundle and exampleTSBundle each build one fully-populated,
// REALISTIC example Bundle -- one per language, not one bundle mixing
// both languages' field shapes. A single invocation only ever runs one
// language's parser and produces one Bundle (factory-style dispatch by
// runtime/language); a chimera fixture wouldn't correspond to any real
// code path and could mislead 007/008 implementers reading it as a
// reference (see progress.md's T005 scoping entry for the full
// reasoning). Each function is the single source used both to generate
// its corresponding testdata/example_*.json fixture and to verify it
// stays byte-for-byte in sync with the struct (Article IV: the fixture
// is generated from the struct, never hand-written in parallel).
//
// Run `go test ./internal/contract/... -run TestGolden -update` after a
// deliberate change to either builder, to regenerate both fixtures.

var updateGolden = flag.Bool("update", false, "regenerate golden testdata fixtures")

func exampleJavaBundle() Bundle {
	chain := []ExceptionNode{
		{
			ClassName: "java.lang.RuntimeException",
			Message:   "Failed to process request",
			Frames: []Frame{
				{Index: 0, FilePath: "/repo/src/main/java/com/example/Handler.java", ClassName: "com.example.Handler", MethodName: "handle", LineNumber: 42, Bucket: BucketOwn},
				{Index: 1, FilePath: "/repo/.m2/repository/com/fasterxml/jackson/core/jackson-databind/2.16.0/ObjectMapper.java", MethodName: "readValue", LineNumber: 3418, Bucket: BucketDependency, PackageName: "com.fasterxml.jackson.core:jackson-databind"},
				{Index: 2, FilePath: "/usr/lib/jvm/java-17-openjdk/lib/src/java.base/java/lang/Thread.java", MethodName: "run", LineNumber: 833, Bucket: BucketRuntime},
			},
		},
		{
			ClassName:        "java.sql.SQLException",
			Message:          "Connection refused",
			ElidedFrameCount: 3,
			Frames: []Frame{
				{Index: 0, FilePath: "/repo/src/main/java/com/example/Repository.java", ClassName: "com.example.Repository", MethodName: "query", LineNumber: 88, Bucket: BucketOwn},
			},
		},
	}

	rawInput := `java.lang.RuntimeException: Failed to process request
	at com.example.Handler.handle(Handler.java:42)
	at com.fasterxml.jackson.databind.ObjectMapper.readValue(ObjectMapper.java:3418)
	at java.lang.Thread.run(Thread.java:833)
Caused by: java.sql.SQLException: Connection refused
	at com.example.Repository.query(Repository.java:88)
	... 3 more`

	return Bundle{
		SchemaVersion:     SchemaVersion,
		Language:          LanguageJava,
		OS:                OSLinux,
		RawInput:          rawInput,
		RawInputTruncated: false,
		Fingerprint:       ComputeFingerprint(chain),
		Runtime: Runtime{
			Name:          "jvm",
			Version:       "17.0.9",
			VersionSource: VersionSourceLocalEnvironment,
			Note:          "inferred from local `java -version`; may not match the environment that produced this trace",
		},
		Chain: chain,
		CodeContexts: []CodeContext{
			{
				FrameRef: FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "/repo/src/main/java/com/example/Handler.java",
				Language: LanguageJava,
				Status:   StatusOK,
				Snippet:  Snippet{StartLine: 40, EndLine: 44, TargetLine: 42, Code: "    public Response handle(Request req) {\n        var payload = req.body();\n        return repository.query(payload.id());\n    }\n"},
				Blame: []BlameEntry{
					{StartLine: 40, EndLine: 44, CommitHash: "0123456789abcdef0123456789abcdef01234567", Author: "vedant", CommitDate: "2026-07-28T09:15:00Z", Summary: "handle request payload validation"},
				},
			},
			{
				FrameRef: FrameRef{ChainIndex: 1, FrameIndex: 0},
				FilePath: "/repo/src/main/java/com/example/Repository.java",
				Language: LanguageJava,
				Status:   StatusOK,
				Snippet:  Snippet{StartLine: 86, EndLine: 90, TargetLine: 88, Code: "    public Response query(String id) {\n        var conn = pool.getConnection();\n        return conn.execute(id);\n    }\n"},
				Blame: []BlameEntry{
					{StartLine: 86, EndLine: 90, CommitHash: "fedcba9876543210fedcba9876543210fedcba98", Author: "vedant", CommitDate: "2026-07-30T14:02:00Z", Summary: "add connection pooling"},
				},
			},
		},
		GitMetadata: GitMetadata{
			CurrentCommit:      "fedcba9876543210fedcba9876543210fedcba98",
			Branch:             "main",
			UncommittedChanges: false,
		},
		Dependencies: Dependencies{
			ManifestFile: ManifestFilePomXML,
			Direct:       map[string]string{"com.fasterxml.jackson.core:jackson-databind": "2.16.0", "org.postgresql:postgresql": "42.7.1"},
			Locked:       map[string]LockedDependency{"com.fasterxml.jackson.core:jackson-databind": {Version: "2.16.0"}, "org.postgresql:postgresql": {Note: "no local mvn cache on this checkout (Article IX, decision 0001) -- expected on a fresh clone that hasn't been built locally yet, not a bug"}},
		},
	}
}

func exampleTSBundle() Bundle {
	chain := []ExceptionNode{
		{
			ClassName: "TypeError",
			Message:   "Cannot read properties of undefined (reading 'id')",
			Frames: []Frame{
				{Index: 0, FilePath: "/repo/src/handler.ts", MethodName: "handleRequest", LineNumber: 27, ColumnNumber: 14, Bucket: BucketOwn},
				{Index: 1, FilePath: "/repo/node_modules/lodash/lodash.js", MethodName: "get", LineNumber: 11812, ColumnNumber: 3, Bucket: BucketDependency, PackageName: "lodash"},
				{Index: 2, FilePath: "node:internal/process/task_queues", MethodName: "processTicksAndRejections", LineNumber: 95, ColumnNumber: 5, Bucket: BucketRuntime},
			},
		},
		{
			ClassName: "Error",
			Message:   "ECONNREFUSED",
			Frames: []Frame{
				{Index: 0, FilePath: "/repo/src/service.ts", MethodName: "queryDatabase", LineNumber: 63, ColumnNumber: 9, Bucket: BucketOwn},
			},
		},
	}

	rawInput := `TypeError: Cannot read properties of undefined (reading 'id')
    at handleRequest (/repo/src/handler.ts:27:14)
    at Object.get (/repo/node_modules/lodash/lodash.js:11812:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
Caused by: Error: ECONNREFUSED
    at queryDatabase (/repo/src/service.ts:63:9)
Node.js v20.11.0`

	return Bundle{
		SchemaVersion:     SchemaVersion,
		Language:          LanguageTypeScript,
		OS:                OSDarwin,
		RawInput:          rawInput,
		RawInputTruncated: false,
		Fingerprint:       ComputeFingerprint(chain),
		Runtime: Runtime{
			Name:          "node",
			Version:       "20.11.0",
			VersionSource: VersionSourceTrace,
		},
		Chain: chain,
		CodeContexts: []CodeContext{
			{
				FrameRef: FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "/repo/src/handler.ts",
				Language: LanguageTypeScript,
				Status:   StatusOK,
				Snippet:  Snippet{StartLine: 25, EndLine: 29, TargetLine: 27, Code: "export function handleRequest(req: Request) {\n  const payload = req.body;\n  return service.queryDatabase(payload.id);\n}\n"},
				Blame: []BlameEntry{
					{StartLine: 25, EndLine: 29, CommitHash: "89abcdef0123456789abcdef0123456789abcdef", Author: "vedant", CommitDate: "2026-07-29T11:40:00Z", Summary: "validate request payload before dispatch"},
				},
			},
			{
				FrameRef: FrameRef{ChainIndex: 1, FrameIndex: 0},
				FilePath: "/repo/src/service.ts",
				Language: LanguageTypeScript,
				Status:   StatusOK,
				Snippet:  Snippet{StartLine: 61, EndLine: 65, TargetLine: 63, Code: "export async function queryDatabase(id: string) {\n  const conn = await pool.connect();\n  return conn.query(id);\n}\n"},
				Blame: []BlameEntry{
					{StartLine: 61, EndLine: 65, CommitHash: "76543210fedcba9876543210fedcba9876543210", Author: "vedant", CommitDate: "2026-08-01T16:20:00Z", Summary: "add connection pooling for query path"},
				},
			},
		},
		GitMetadata: GitMetadata{
			CurrentCommit:      "76543210fedcba9876543210fedcba9876543210",
			Branch:             "feature/query-fix",
			UncommittedChanges: true,
		},
		Dependencies: Dependencies{
			ManifestFile: ManifestFilePackageJSON,
			Direct:       map[string]string{"lodash": "^4.17.21", "express": "^4.18.2"},
			Locked: map[string]LockedDependency{
				"lodash":  {Version: "4.17.21"},
				"express": {Note: "no local npm cache on this checkout (Article IX, decision 0001) -- expected on a fresh clone that hasn't been built locally yet, not a bug"},
			},
		},
	}
}

func TestGolden_ExampleJava(t *testing.T) {
	assertGolden(t, exampleJavaBundle(), filepath.Join("testdata", "example_java.json"))
}

func TestGolden_ExampleTS(t *testing.T) {
	assertGolden(t, exampleTSBundle(), filepath.Join("testdata", "example_ts.json"))
}

// assertGolden re-marshals bundle and compares it byte-for-byte against
// the fixture at path. With -update, it (re)writes the fixture from
// bundle instead of comparing -- the only way either fixture is ever
// produced, per Article IV.
func assertGolden(t *testing.T, bundle Bundle, path string) {
	t.Helper()

	got, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	got = append(got, '\n')

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v -- run with -update to generate it", path, err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("%s is out of date with the struct it's generated from -- run:\n  go test ./internal/contract/... -run TestGolden -update\nto regenerate it, then review the diff", path)
	}
}

package typescript

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

// TestFixturesLoad is a T001 placeholder: it only confirms every fixture
// referenced in testdata/README.md is present on disk and non-empty. It
// does not parse or assert on trace content -- that begins in T002+ once
// engine.go's frame-line matcher exists. Table entries mirror
// testdata/README.md's two tables (real captures, then synthetic) so the
// two stay in sync; a fixture added to one without the other should fail
// review, not just this test.
func TestFixturesLoad(t *testing.T) {
	fixtures := []string{
		// Real captures
		"crash-with-cause.txt",
		"crash-typeerror-no-cause.txt",
		"logged-object-with-cause.txt",
		"bare-stack.txt",
		"crash-3level-cause.txt",
		"crash-async-rejection-preamble.txt",
		"logged-object-fetch-cause.txt",
		"bare-stack-fetch-cause.txt",
		"nested-deps-flat.txt",
		"zero-stack-trace-limit.txt",
		"esm-runtime-error.txt",
		"import-outside-module.txt",
		"ts-native-execution.txt",
		"ts-compiled-then-run.txt",
		"ts-tsx-transformer-path.txt",
		"esbuild-minified-bundle.txt",
		"scoped-package-swc-false-caused-by.txt",
		"aggregate-error-uncaught.txt",
		"assert-multiline-diff.txt",
		// Synthetic (no real capture exists for these shapes)
		"scoped-package-babel.txt",
		"deep-nested-node-modules.txt",
		"browser-trace-false-positive.txt",
		"truncated-mid-frame.txt",
		"cutoff-cause-chain.txt",
		"unparseable-input.txt",
	}

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("testdata", name)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}
			if len(content) == 0 {
				t.Fatalf("%s is empty", path)
			}
		})
	}
}

// mustReadFixture is typescript_test.go's own read-fixture helper --
// engine_test.go's mustParseChain isn't reused here since these tests
// need the raw content itself (for Detect()/Parse()), not an
// already-parsed chain.
func mustReadFixture(t *testing.T, fixture string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return string(content)
}

// TestDetectSplit exercises T010/spec.md FR6: javascriptParser and
// typescriptParser's Detect() methods must never both match, and never
// both reject, a given real trace -- exactly one claims it.
func TestDetectSplit(t *testing.T) {
	tests := []struct {
		fixture        string
		wantJS, wantTS bool
	}{
		// All .js/node: paths, no .ts anywhere -- plain JavaScript.
		{"crash-with-cause.txt", true, false},
		// Node's native type-stripping execution: real .ts frame paths,
		// no transformer frame at all.
		{"ts-native-execution.txt", false, true},
		// tsx-wrapped execution: real .ts frame paths PLUS a transformer
		// frame under node_modules -- must still classify TypeScript, not
		// get confused by the extra dependency frame.
		{"ts-tsx-transformer-path.txt", false, true},
		// Compiled-then-node-run TypeScript: only .js paths post-
		// compilation -- correctly indistinguishable from hand-written JS
		// (spec.md FR6's own note), classified JavaScript, not a bug.
		{"ts-compiled-then-run.txt", true, false},
		// Neither: detectNodeTrace itself already returns false (no
		// Node-specific signal) -- browser trace, out of scope entirely.
		{"browser-trace-false-positive.txt", false, false},
	}

	js := javascriptParser{}
	ts := typescriptParser{}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			content := mustReadFixture(t, tt.fixture)
			gotJS := js.Detect(content)
			gotTS := ts.Detect(content)
			if gotJS != tt.wantJS {
				t.Errorf("javascriptParser.Detect(%s) = %v, want %v", tt.fixture, gotJS, tt.wantJS)
			}
			if gotTS != tt.wantTS {
				t.Errorf("typescriptParser.Detect(%s) = %v, want %v", tt.fixture, gotTS, tt.wantTS)
			}
			if gotJS && gotTS {
				t.Errorf("javascriptParser and typescriptParser both matched %s -- FR6 requires exactly one", tt.fixture)
			}
		})
	}
}

// TestLanguage confirms each value's Language() is the static constant
// FR5 requires, independent of any trace content.
func TestLanguage(t *testing.T) {
	if got := (javascriptParser{}).Language(); got != contract.LanguageJavaScript {
		t.Errorf("javascriptParser.Language() = %q, want %q", got, contract.LanguageJavaScript)
	}
	if got := (typescriptParser{}).Language(); got != contract.LanguageTypeScript {
		t.Errorf("typescriptParser.Language() = %q, want %q", got, contract.LanguageTypeScript)
	}
}

// TestParseFullComposition exercises T010's full composition end to end
// against testdata/nested-deps-flat.txt: a single real fixture that
// happens to carry all three buckets (dependency, own, runtime) plus a
// trace-sourced version line, in one shot -- chosen specifically so this
// test covers bucketing (T008) and trace-sourced Runtime (FR14)
// together, not because a single node needing all three is typical.
func TestParseFullComposition(t *testing.T) {
	content := mustReadFixture(t, "nested-deps-flat.txt")

	chain, runtime, err := NewJavaScriptParser().Parse(context.Background(), content)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	if runtime.Name != "node" {
		t.Errorf("runtime.Name = %q, want %q", runtime.Name, "node")
	}
	if runtime.Version != "24.18.0" || runtime.VersionSource != contract.VersionSourceTrace {
		t.Errorf("runtime = {Version: %q, VersionSource: %q}, want {24.18.0, trace}", runtime.Version, runtime.VersionSource)
	}
	if runtime.Note != "" {
		t.Errorf("runtime.Note = %q, want empty for VersionSourceTrace", runtime.Note)
	}

	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1", len(chain))
	}
	frames := chain[0].Frames
	if len(frames) != 10 {
		t.Fatalf("len(frames) = %d, want 10", len(frames))
	}

	// Frames[0], Frames[1]: node_modules/statuses -- BucketDependency,
	// PackageName "statuses".
	for i := range 2 {
		if frames[i].Bucket != contract.BucketDependency || frames[i].PackageName != "statuses" {
			t.Errorf("frames[%d] = {Bucket: %q, PackageName: %q}, want {dependency, statuses}", i, frames[i].Bucket, frames[i].PackageName)
		}
	}
	// Frames[2]: pg-fails.js, outside node_modules -- BucketOwn.
	if frames[2].Bucket != contract.BucketOwn {
		t.Errorf("frames[2].Bucket = %q, want own", frames[2].Bucket)
	}
	// Frames[3..9]: node: internal -- BucketRuntime.
	for i := 3; i < 10; i++ {
		if frames[i].Bucket != contract.BucketRuntime {
			t.Errorf("frames[%d].Bucket = %q, want runtime", i, frames[i].Bucket)
		}
	}
}

// TestParseEngineLocalFallback exercises parseEngine directly (T010's
// runner-injectable core, mirroring internal/codecontext/context.go's
// buildCodeContexts/BuildCodeContexts split) against a shape (c) fixture
// with no trace-sourced version line, so it must fall through to the
// local-environment path -- using fakeNodeVersionRunner (already defined
// in runtime_test.go for T005) rather than letting a real "node
// --version" subprocess run in this test, which would be both slow and
// non-deterministic across machines/CI.
func TestParseEngineLocalFallback(t *testing.T) {
	content := mustReadFixture(t, "bare-stack.txt")

	t.Run("local node --version succeeds", func(t *testing.T) {
		fake := &fakeNodeVersionRunner{fn: func(context.Context) (string, error) {
			return "v24.18.0\n", nil
		}}
		_, runtime, err := parseEngine(context.Background(), content, fake)
		if err != nil {
			t.Fatalf("parseEngine() error = %v, want nil", err)
		}
		if runtime.VersionSource != contract.VersionSourceLocalEnvironment || runtime.Version != "24.18.0" {
			t.Errorf("runtime = {Version: %q, VersionSource: %q}, want {24.18.0, local-environment}", runtime.Version, runtime.VersionSource)
		}
		if runtime.Note == "" {
			t.Error("runtime.Note is empty, want a non-empty note explaining local inference")
		}
	})

	t.Run("local node --version fails", func(t *testing.T) {
		fake := &fakeNodeVersionRunner{fn: func(context.Context) (string, error) {
			return "", errors.New("exec: \"node\": executable file not found in $PATH")
		}}
		_, runtime, err := parseEngine(context.Background(), content, fake)
		if err != nil {
			t.Fatalf("parseEngine() error = %v, want nil", err)
		}
		if runtime.VersionSource != contract.VersionSourceUnknown || runtime.Version != "" {
			t.Errorf("runtime = {Version: %q, VersionSource: %q}, want {\"\", unknown}", runtime.Version, runtime.VersionSource)
		}
	})
}

// TestParseUnparseable exercises T009/FR20 through the full Parse()
// composition (not just parseTrace directly, as engine_test.go's
// TestParseTraceUnparseable already does): confirms the binary-outcome
// contract registry.go's LanguageParser.Parse doc comment requires --
// nil chain, zero-value Runtime (Name == ""), error wrapping
// parser.ErrUnparseable.
func TestParseUnparseable(t *testing.T) {
	content := mustReadFixture(t, "unparseable-input.txt")

	chain, runtime, err := NewJavaScriptParser().Parse(context.Background(), content)
	if chain != nil {
		t.Errorf("chain = %+v, want nil", chain)
	}
	if runtime != (contract.Runtime{}) {
		t.Errorf("runtime = %+v, want zero-value Runtime{}", runtime)
	}
	if !errors.Is(err, parser.ErrUnparseable) {
		t.Fatalf("err = %v, want it to wrap parser.ErrUnparseable", err)
	}
}

// TestParseChainStructureIdenticalAcrossShapes exercises T011/spec.md
// AC1+AC2 together: shape (a) (crash-with-cause.txt, a true crash with
// a trailing version line) and shape (b) (logged-object-with-cause.txt,
// the same exception content logged via console.error(err), no
// preamble, no version line) must parse to the IDENTICAL chain
// structure -- only Runtime.VersionSource differs. Uses parseEngine
// directly with a fixed fake runner (not the exported Parse(), which
// T010's TestParseFullComposition already proves is a thin wrapper
// around parseEngine) so the comparison is fully deterministic.
func TestParseChainStructureIdenticalAcrossShapes(t *testing.T) {
	crashContent := mustReadFixture(t, "crash-with-cause.txt")
	loggedContent := mustReadFixture(t, "logged-object-with-cause.txt")

	fakeRunner := &fakeNodeVersionRunner{fn: func(context.Context) (string, error) {
		return "v24.18.0\n", nil
	}}

	crashChain, crashRuntime, err := parseEngine(context.Background(), crashContent, fakeRunner)
	if err != nil {
		t.Fatalf("parseEngine(crash-with-cause.txt) error = %v", err)
	}
	loggedChain, loggedRuntime, err := parseEngine(context.Background(), loggedContent, fakeRunner)
	if err != nil {
		t.Fatalf("parseEngine(logged-object-with-cause.txt) error = %v", err)
	}

	if !reflect.DeepEqual(crashChain, loggedChain) {
		t.Errorf("chains differ:\ncrash-with-cause.txt: %+v\nlogged-object-with-cause.txt: %+v", crashChain, loggedChain)
	}

	// AC1: shape (a) has a trailing "Node.js vX.Y.Z" line -> trace-sourced,
	// the fake runner is never even consulted for this one.
	if crashRuntime.VersionSource != contract.VersionSourceTrace || crashRuntime.Version != "24.18.0" {
		t.Errorf("crash runtime = %+v, want {Version: 24.18.0, VersionSource: trace}", crashRuntime)
	}
	// AC2: shape (b) has no trailing version line -> falls back to the
	// local environment (VersionSourceUnknown's fallback path is already
	// covered generically by TestParseEngineLocalFallback -- the failure
	// branch inside parseEngine doesn't care which fixture triggered it).
	if loggedRuntime.VersionSource != contract.VersionSourceLocalEnvironment {
		t.Errorf("logged runtime.VersionSource = %q, want local-environment", loggedRuntime.VersionSource)
	}
}

// TestParseEngineNormalizesFileURI exercises T011/spec.md AC12 through
// the FULL pipeline (parseChain + bucketing), not just assignBucket in
// isolation against a literal string as bucket_test.go's "file:// URI is
// normalized before bucketing" case already does -- this is the one
// real fixture (esm-runtime-error.txt) that carries an actual "file://"
// frame path all the way from raw text through parseEngine's
// composition, and it was never exercised as a whole fixture by any
// earlier task's tests (only by TestParseFrameLine's single-line case
// and TestDetectNodeTrace/TestExtractTraceVersion, neither of which
// reaches bucketing).
func TestParseEngineNormalizesFileURI(t *testing.T) {
	content := mustReadFixture(t, "esm-runtime-error.txt")
	fakeRunner := &fakeNodeVersionRunner{fn: func(context.Context) (string, error) {
		return "v24.18.0\n", nil
	}}

	chain, _, err := parseEngine(context.Background(), content, fakeRunner)
	if err != nil {
		t.Fatalf("parseEngine error = %v", err)
	}
	if len(chain) != 1 || len(chain[0].Frames) == 0 {
		t.Fatalf("unexpected chain shape: %+v", chain)
	}

	got := chain[0].Frames[0].FilePath
	want := "/home/vedant/stack-trace-bundler/errors-test/pg-fails.js"
	if got != want {
		t.Errorf("Frames[0].FilePath = %q, want %q -- file:// must be normalized", got, want)
	}
	if chain[0].Frames[0].Bucket != contract.BucketOwn {
		t.Errorf("Frames[0].Bucket = %q, want own", chain[0].Frames[0].Bucket)
	}
}

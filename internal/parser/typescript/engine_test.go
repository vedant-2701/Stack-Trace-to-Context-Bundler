package typescript

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

func TestParseFrameLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantOK    bool
		wantFrame contract.Frame
	}{
		{
			name:   "plain frame with function name",
			line:   "at Foo (bar.js:1:2)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "bar.js", LineNumber: 1, ColumnNumber: 2,
				MethodName: "Foo",
			},
		},
		{
			name:   "bare frame, no function name",
			line:   "at bar.js:1:2",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "bar.js", LineNumber: 1, ColumnNumber: 2,
			},
		},
		{
			name:   "non-matching line is not an error",
			line:   "this is not an error header just random text",
			wantOK: false,
		},
		{
			name:   "system-error property line is not a frame (FR7)",
			line:   "  errno: -111,",
			wantOK: false,
		},
		{
			name:   "top-level property line is not a frame (FR7)",
			line:   "  code: 'GenericFailure'",
			wantOK: false,
		},
		{
			name:   "6-space-indented nested-cause frame",
			line:   "      at level2 (/home/vedant/script.js:5:10)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "/home/vedant/script.js", LineNumber: 5, ColumnNumber: 10,
				MethodName: "level2",
			},
		},
		{
			name:   "bare frame with trailing brace before [cause]:",
			line:   "    at node:internal/main/run_main_module:33:47 {",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:internal/main/run_main_module", LineNumber: 33, ColumnNumber: 47,
			},
		},
		{
			name:   "described frame with trailing brace before [cause]:",
			line:   "      at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:net", LineNumber: 1706, ColumnNumber: 16,
				ClassName: "TCPConnectWrap", MethodName: "afterConnect [as oncomplete]",
			},
		},
		{
			name:   "async bare frame",
			line:   "    at async node:internal/modules/esm/loader:643:26",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:internal/modules/esm/loader", LineNumber: 643, ColumnNumber: 26,
			},
		},
		{
			name:   "async described frame",
			line:   "    at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:internal/modules/run_main", LineNumber: 101, ColumnNumber: 5,
				MethodName: "asyncRunEntryPointWithESMLoader",
			},
		},
		{
			name:   "dot-qualified description splits into ClassName/MethodName",
			line:   "    at Object.<anonymous> (/home/vedant/script.js:7:15)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "/home/vedant/script.js", LineNumber: 7, ColumnNumber: 15,
				ClassName: "Object", MethodName: "<anonymous>",
			},
		},
		{
			name:   "file:// URI frame path is preserved, not yet normalized",
			line:   "    at main (file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:4:9)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath:   "file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js",
				LineNumber: 4, ColumnNumber: 9,
				MethodName: "main",
			},
		},
		{
			name:   "blank line is not a frame",
			line:   "",
			wantOK: false,
		},
		{
			name:   "closing brace alone is not a frame",
			line:   "}",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrame, gotOK := parseFrameLine(tt.line)
			if gotOK != tt.wantOK {
				t.Fatalf("parseFrameLine(%q) ok = %v, want %v", tt.line, gotOK, tt.wantOK)
			}
			if !gotOK {
				return
			}
			if gotFrame != tt.wantFrame {
				t.Errorf("parseFrameLine(%q) = %+v, want %+v", tt.line, gotFrame, tt.wantFrame)
			}
		})
	}
}

// TestDetectNodeTrace exercises spec.md FR4's Detect() heuristic against
// every fixture in testdata/. wantDetect is true for every real and
// synthetic fixture except bare-stack-fetch-cause.txt, which is
// genuinely undetectable by design (see spec.md FR4's "known residual
// gap" note and memory/known-gaps.md) -- and false for the
// browser-trace synthetic fixture and a hardcoded plain-text case.
func TestDetectNodeTrace(t *testing.T) {
	tests := []struct {
		fixture    string
		wantDetect bool
	}{
		{"crash-with-cause.txt", true},
		{"crash-typeerror-no-cause.txt", true},
		{"logged-object-with-cause.txt", true},
		{"bare-stack.txt", true},
		{"crash-3level-cause.txt", true},
		{"crash-async-rejection-preamble.txt", true},
		{"logged-object-fetch-cause.txt", true},
		{"bare-stack-fetch-cause.txt", false}, // the confirmed exception -- see doc comment above
		{"nested-deps-flat.txt", true},
		{"zero-stack-trace-limit.txt", true}, // zero frames, but preamble + version line present
		{"esm-runtime-error.txt", true},
		{"import-outside-module.txt", true},
		{"ts-native-execution.txt", true},
		{"ts-compiled-then-run.txt", true},
		{"ts-tsx-transformer-path.txt", true},
		{"esbuild-minified-bundle.txt", true},
		{"scoped-package-swc-false-caused-by.txt", true},
		{"aggregate-error-uncaught.txt", true},
		{"assert-multiline-diff.txt", true},
		{"scoped-package-babel.txt", true},
		{"deep-nested-node-modules.txt", true},
		{"browser-trace-false-positive.txt", false},
		{"truncated-mid-frame.txt", true},
		{"cutoff-cause-chain.txt", true},
		{"unparseable-input.txt", true}, // Detect() matches loosely by design (FR20) -- Parse() fails this one, not Detect()
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("testdata", tt.fixture))
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}
			if got := detectNodeTrace(string(content)); got != tt.wantDetect {
				t.Errorf("detectNodeTrace(%s) = %v, want %v", tt.fixture, got, tt.wantDetect)
			}
		})
	}

	t.Run("plain non-trace text", func(t *testing.T) {
		text := "just some random text\nwith multiple lines\nno error at all here"
		if got := detectNodeTrace(text); got {
			t.Errorf("detectNodeTrace(plain text) = %v, want false", got)
		}
	})
}

// TestExtractTraceVersion exercises spec.md FR14: version extracted +
// found only for shape (a) (true crash) fixtures carrying the trailing
// Node.js vX.Y.Z line, distinguishing both preamble variants from
// shape (b)/(c), which never legitimately carry one.
func TestExtractTraceVersion(t *testing.T) {
	tests := []struct {
		fixture     string
		wantVersion string
		wantOK      bool
	}{
		{"crash-with-cause.txt", "24.18.0", true},               // sync-throw variant
		{"crash-async-rejection-preamble.txt", "24.18.0", true}, // async-rejection variant
		{"zero-stack-trace-limit.txt", "24.18.0", true},         // zero frames, still shape (a)
		{"import-outside-module.txt", "24.18.0", true},          // multi-caret "^^^^^^" preamble
		{"logged-object-with-cause.txt", "", false},             // shape (b): no preamble, no version line
		{"bare-stack.txt", "", false},                           // shape (c): no preamble, no version line
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("testdata", tt.fixture))
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}
			gotVersion, gotOK := extractTraceVersion(string(content))
			if gotOK != tt.wantOK {
				t.Fatalf("extractTraceVersion(%s) ok = %v, want %v", tt.fixture, gotOK, tt.wantOK)
			}
			if gotVersion != tt.wantVersion {
				t.Errorf("extractTraceVersion(%s) version = %q, want %q", tt.fixture, gotVersion, tt.wantVersion)
			}
		})
	}
}

// TestParseChain exercises T006's [cause]: bracket-chain parsing
// (spec.md FR7/FR8) against every real fixture listed in T006's own
// acceptance criteria in tasks.md. Fixtures requiring T009's later
// tolerance logic (truncated/cut-off/zero-frame/unparseable inputs) are
// deliberately not exercised here -- see engine.go's parseNodeAndCause
// doc comment.
//
// Split into one top-level Test func per fixture group (plus a shared
// mustParseChain helper) rather than one Test func with many t.Run
// subtests -- gocyclo scores all nested closures against the enclosing
// function, and a single TestParseChain covering all six fixture groups
// tripped golangci-lint's complexity threshold.

// mustParseChain reads fixture from testdata/ and parses it, failing
// the test immediately if either step doesn't succeed. Shared by every
// TestParseChain* function below to avoid repeating the read+parse+ok
// boilerplate six times.
func mustParseChain(t *testing.T, fixture string) []contract.ExceptionNode {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	chain, ok := parseChain(string(content))
	if !ok {
		t.Fatalf("parseChain(%s) ok = false, want true", fixture)
	}
	return chain
}

// checkTwoNodeCrashChain asserts the shared 2-node shape that both
// crash-with-cause.txt and logged-object-with-cause.txt must parse to
// (spec.md FR7/FR8/FR21).
func checkTwoNodeCrashChain(t *testing.T, fixture string) {
	t.Helper()
	chain := mustParseChain(t, fixture)
	if len(chain) != 2 {
		t.Fatalf("len(chain) = %d, want 2", len(chain))
	}

	outer, inner := chain[0], chain[1]

	if outer.ClassName != "Error" || outer.Message != "outer failure" {
		t.Errorf("outer = %q: %q, want Error: outer failure", outer.ClassName, outer.Message)
	}
	if outer.ElidedFrameCount != 5 {
		t.Errorf("outer.ElidedFrameCount = %d, want 5", outer.ElidedFrameCount)
	}
	if len(outer.Frames) != 3 {
		t.Fatalf("len(outer.Frames) = %d, want 3", len(outer.Frames))
	}
	// FR21 reverification: Frames[0] must be the frame nearest this
	// node's own origin -- confirmed against the raw fixture's first
	// "at" line for outer, not just a count.
	wantFrame0 := contract.Frame{
		FilePath: "/home/vedant/script.js", LineNumber: 2, ColumnNumber: 15,
		ClassName: "Object", MethodName: "<anonymous>",
	}
	if outer.Frames[0] != wantFrame0 {
		t.Errorf("outer.Frames[0] = %+v, want %+v", outer.Frames[0], wantFrame0)
	}

	if inner.ClassName != "Error" || inner.Message != "inner failure" {
		t.Errorf("inner = %q: %q, want Error: inner failure", inner.ClassName, inner.Message)
	}
	if inner.ElidedFrameCount != 0 {
		t.Errorf("inner.ElidedFrameCount = %d, want 0", inner.ElidedFrameCount)
	}
	if len(inner.Frames) != 8 {
		t.Fatalf("len(inner.Frames) = %d, want 8", len(inner.Frames))
	}
	wantInnerFrame0 := contract.Frame{
		FilePath: "/home/vedant/script.js", LineNumber: 1, ColumnNumber: 15,
		ClassName: "Object", MethodName: "<anonymous>",
	}
	if inner.Frames[0] != wantInnerFrame0 {
		t.Errorf("inner.Frames[0] = %+v, want %+v", inner.Frames[0], wantInnerFrame0)
	}
}

func TestParseChainCrashAndLoggedObjectSameChain(t *testing.T) {
	for _, fixture := range []string{"crash-with-cause.txt", "logged-object-with-cause.txt"} {
		t.Run(fixture, func(t *testing.T) {
			checkTwoNodeCrashChain(t, fixture)
		})
	}
}

func TestParseChainThreeLevelNested(t *testing.T) {
	chain := mustParseChain(t, "crash-3level-cause.txt")
	if len(chain) != 3 {
		t.Fatalf("len(chain) = %d, want 3", len(chain))
	}

	wantElided := []int{5, 6, 0}
	wantFrameCount := []int{3, 3, 10}
	wantMessage := []string{"outer failure", "middle failure", "innermost failure"}
	wantFrame0MethodName := []string{"<anonymous>", "level2", "level1"}
	wantFrame0Line := []int{7, 5, 2}

	for i, node := range chain {
		if node.Message != wantMessage[i] {
			t.Errorf("chain[%d].Message = %q, want %q", i, node.Message, wantMessage[i])
		}
		if node.ElidedFrameCount != wantElided[i] {
			t.Errorf("chain[%d].ElidedFrameCount = %d, want %d", i, node.ElidedFrameCount, wantElided[i])
		}
		if len(node.Frames) != wantFrameCount[i] {
			t.Fatalf("len(chain[%d].Frames) = %d, want %d", i, len(node.Frames), wantFrameCount[i])
		}
		if node.Frames[0].MethodName != wantFrame0MethodName[i] {
			t.Errorf("chain[%d].Frames[0].MethodName = %q, want %q", i, node.Frames[0].MethodName, wantFrame0MethodName[i])
		}
		if node.Frames[0].LineNumber != wantFrame0Line[i] {
			t.Errorf("chain[%d].Frames[0].LineNumber = %d, want %d", i, node.Frames[0].LineNumber, wantFrame0Line[i])
		}
	}
}

func TestParseChainFetchCauseZeroFrameBracket(t *testing.T) {
	chain := mustParseChain(t, "logged-object-fetch-cause.txt")
	if len(chain) != 2 {
		t.Fatalf("len(chain) = %d, want 2", len(chain))
	}
	outer, inner := chain[0], chain[1]
	if outer.ClassName != "TypeError" || outer.Message != "fetch failed" {
		t.Errorf("outer = %q: %q, want TypeError: fetch failed", outer.ClassName, outer.Message)
	}
	if len(outer.Frames) != 0 {
		t.Errorf("len(outer.Frames) = %d, want 0 (bracket zero-frame header)", len(outer.Frames))
	}
	if inner.ClassName != "Error" || inner.Message != "connect ECONNREFUSED 127.0.0.1:59999" {
		t.Errorf("inner = %q: %q, want Error: connect ECONNREFUSED 127.0.0.1:59999", inner.ClassName, inner.Message)
	}
	if len(inner.Frames) != 1 {
		t.Fatalf("len(inner.Frames) = %d, want 1", len(inner.Frames))
	}
}

func TestParseChainScopedPackageSWCCausedByNotBoundary(t *testing.T) {
	chain := mustParseChain(t, "scoped-package-swc-false-caused-by.txt")
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1 -- 'Caused by:' text must not open a second node", len(chain))
	}
	node := chain[0]
	wantMessage := "Failed to deserialize buffer as swc::config::Options\n" +
		"JSON: {\"jsc\":{\"target\":\"invalid-target\"}}\n" +
		"\n" +
		"Caused by:\n" +
		"    Unknown ES version: invalid-target at line 1 column 35"
	if node.Message != wantMessage {
		t.Errorf("Message = %q, want %q", node.Message, wantMessage)
	}
	if len(node.Frames) != 10 {
		t.Errorf("len(Frames) = %d, want 10", len(node.Frames))
	}
}

func TestParseChainAggregateErrorDropped(t *testing.T) {
	chain := mustParseChain(t, "aggregate-error-uncaught.txt")
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1 -- [errors] must not be attempted as a chain", len(chain))
	}
	node := chain[0]
	if node.ClassName != "AggregateError" || node.Message != "All promises were rejected" {
		t.Errorf("node = %q: %q, want AggregateError: All promises were rejected", node.ClassName, node.Message)
	}
	if len(node.Frames) != 0 {
		t.Errorf("len(Frames) = %d, want 0", len(node.Frames))
	}
}

// TestParseChainBareStack exercises T007/spec.md FR9: shape (c), a bare
// `.stack` string logged via console.error(err.stack), has no [cause]:
// or brace-body structure at all -- its header line's collectMessageAndFrames
// call never finds an opensBody frame line, so parseNodeAndCause returns
// a single-node chain without ever entering the body-processing loop
// that looks for [cause]:/[errors]:/a closing "}". This requires no new
// production code: shape (c) falls out of T006's parseChain as the
// "header not followed by a body" case, confirmed here rather than
// assumed (see T006's progress.md entry, which flagged this as likely
// but not yet checked against the fixture).
func TestParseChainBareStack(t *testing.T) {
	chain := mustParseChain(t, "bare-stack.txt")
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1 -- bare .stack must never produce a cause node", len(chain))
	}

	node := chain[0]
	if node.ClassName != "Error" || node.Message != "outer failure" {
		t.Errorf("node = %q: %q, want Error: outer failure", node.ClassName, node.Message)
	}
	if node.ElidedFrameCount != 0 {
		t.Errorf("node.ElidedFrameCount = %d, want 0", node.ElidedFrameCount)
	}
	if len(node.Frames) != 8 {
		t.Fatalf("len(node.Frames) = %d, want 8", len(node.Frames))
	}
	// FR21: Frames[0] is the frame nearest this node's own origin,
	// matching the fixture's first "at" line.
	wantFrame0 := contract.Frame{
		FilePath: "/home/vedant/script.js", LineNumber: 2, ColumnNumber: 15,
		ClassName: "Object", MethodName: "<anonymous>",
	}
	if node.Frames[0] != wantFrame0 {
		t.Errorf("node.Frames[0] = %+v, want %+v", node.Frames[0], wantFrame0)
	}
	// Last frame has no file:// or node_modules signal, just a bare
	// node: internal path with no described function name -- confirms
	// the bare-form frame-line branch is exercised too, not just the
	// described form.
	wantLastFrame := contract.Frame{
		FilePath: "node:internal/main/run_main_module", LineNumber: 33, ColumnNumber: 47,
	}
	wantLastFrame.Index = 7
	gotLastFrame := node.Frames[7]
	gotLastFrame.Index = 7
	if gotLastFrame != wantLastFrame {
		t.Errorf("node.Frames[7] = %+v, want %+v", gotLastFrame, wantLastFrame)
	}
}

// TestParseChainTruncatedMidFrame exercises T009/spec.md FR17: a
// trailing incomplete frame line ("at Module._load (node:internal/
// modules/cjs/lo", cut off mid-location) must be dropped, not counted
// as a frame and not swallowed into Message -- the 4 complete frames
// before it are still a successful parse.
func TestParseChainTruncatedMidFrame(t *testing.T) {
	chain := mustParseChain(t, "truncated-mid-frame.txt")
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1", len(chain))
	}
	node := chain[0]
	if node.ClassName != "Error" || node.Message != "outer failure" {
		t.Errorf("node = %q: %q, want Error: outer failure -- the truncated line must not leak into Message", node.ClassName, node.Message)
	}
	if len(node.Frames) != 4 {
		t.Fatalf("len(node.Frames) = %d, want 4", len(node.Frames))
	}
}

// TestParseChainCutoffCauseChain exercises T009/spec.md FR18: a
// [cause]: node that opens but is itself cut off mid-frame
// (testdata/cutoff-cause-chain.txt's inner "Error: inner fail" node has
// only a truncated "at Object.<anonymous> (/home/vedant/script.js:1:"
// line, no closing brace) must be dropped ENTIRELY -- not kept with the
// truncated line corrupting its Message, which is what the pre-T009
// code did. Only the outer node survives.
func TestParseChainCutoffCauseChain(t *testing.T) {
	chain := mustParseChain(t, "cutoff-cause-chain.txt")
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1 -- the incomplete inner cause must be dropped entirely, not kept corrupted", len(chain))
	}
	node := chain[0]
	if node.ClassName != "Error" || node.Message != "outer failure" {
		t.Errorf("node = %q: %q, want Error: outer failure", node.ClassName, node.Message)
	}
	if len(node.Frames) != 3 {
		t.Fatalf("len(node.Frames) = %d, want 3", len(node.Frames))
	}
}

// TestParseChainZeroFrameLegit exercises T009/spec.md FR19 against the
// canonical Error.stackTraceLimit = 0 fixture: a message-only error
// with zero frames is a valid, complete degraded result (distinct from
// FR17's truncated case) -- Frames must be a non-nil empty slice, not
// nil, so it marshals to JSON "[]" rather than "null"
// (contract.ExceptionNode.Frames carries no omitempty tag).
func TestParseChainZeroFrameLegit(t *testing.T) {
	chain := mustParseChain(t, "zero-stack-trace-limit.txt")
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1", len(chain))
	}
	node := chain[0]
	if node.ClassName != "Error" || node.Message != "Zero stack frames requested" {
		t.Errorf("node = %q: %q, want Error: Zero stack frames requested", node.ClassName, node.Message)
	}
	if node.Frames == nil {
		t.Error("node.Frames is nil, want a non-nil empty slice (must marshal to JSON [], not null)")
	}
	if len(node.Frames) != 0 {
		t.Errorf("len(node.Frames) = %d, want 0", len(node.Frames))
	}
}

// TestParseTraceUnparseable exercises T009/spec.md FR20: genuinely
// unparseable input (no valid exception header at all) must surface as
// an error wrapping parser.ErrUnparseable via errors.Is, not just a
// bare ok==false the caller has to remember to check -- this is
// parseTrace's whole purpose (see its doc comment for why it's a
// separate function from parseChain).
func TestParseTraceUnparseable(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "unparseable-input.txt"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	chain, err := parseTrace(string(content))
	if chain != nil {
		t.Errorf("parseTrace returned non-nil chain %+v on unparseable input, want nil", chain)
	}
	if !errors.Is(err, parser.ErrUnparseable) {
		t.Fatalf("parseTrace error = %v, want it to wrap parser.ErrUnparseable", err)
	}
}

// TestParseTraceSuccess is parseTrace's happy-path counterpart to
// TestParseTraceUnparseable -- confirms it also correctly passes
// through a valid parse with a nil error, not just that it detects the
// failure case.
func TestParseTraceSuccess(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "bare-stack.txt"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	chain, err := parseTrace(string(content))
	if err != nil {
		t.Fatalf("parseTrace(bare-stack.txt) error = %v, want nil", err)
	}
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1", len(chain))
	}
}

func TestParseChainAssertMultilineDiff(t *testing.T) {
	chain := mustParseChain(t, "assert-multiline-diff.txt")
	if len(chain) != 1 {
		t.Fatalf("len(chain) = %d, want 1", len(chain))
	}
	node := chain[0]
	if node.ClassName != "AssertionError [ERR_ASSERTION]" {
		t.Errorf("ClassName = %q, want %q", node.ClassName, "AssertionError [ERR_ASSERTION]")
	}
	wantMessage := "Expected \"actual\" to be reference-equal to \"expected\":\n" +
		"+ actual - expected\n" +
		"\n" +
		"  {\n" +
		"+   age: 30,\n" +
		"-   age: 31,\n" +
		"    name: 'Alice'\n" +
		"  }\n"
	if node.Message != wantMessage {
		t.Errorf("Message = %q, want %q", node.Message, wantMessage)
	}
	if len(node.Frames) != 8 {
		t.Errorf("len(Frames) = %d, want 8 -- diff lines must not be counted as frames", len(node.Frames))
	}
}

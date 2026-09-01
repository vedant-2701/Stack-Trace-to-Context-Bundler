// Package typescript implements the 006a LanguageParser for TypeScript
// and JavaScript Node.js stack traces. Two exported constructors,
// NewJavaScriptParser and NewTypeScriptParser, return the (unexported)
// javascriptParser/typescriptParser values -- both share one unexported
// internal parse engine (engine.go/bucket.go/runtime.go) -- see
// plan.md's Architecture/approach section.
package typescript

import (
	"context"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

// hasTSExtensionFrame implements spec.md FR6: a trace is classified
// TypeScript if ANY frame's FilePath ends in ".ts" or ".tsx" --
// confirmed real via testdata/ts-native-execution.txt (Node's native
// type-stripping execution, no transformer frame) and
// testdata/ts-tsx-transformer-path.txt (tsx-wrapped, WITH a transformer
// frame under node_modules) -- both carry real ".ts" frame paths, so
// one check covers both real TypeScript-execution shapes. Compiled-
// then-node-run TypeScript (testdata/ts-compiled-then-run.txt) has only
// ".js" paths post-compilation and is correctly classified JavaScript
// by this same check finding nothing -- not a bug, a genuine limit of
// what the trace can reveal (spec.md FR6's own note).
//
// Reuses parseFrameLine directly (T002) rather than a separate regex,
// so this can never disagree with what engine.go itself considers a
// frame line -- checked before any bucketing/normalization runs, so a
// "file://"-prefixed ".ts" path (were that ever to occur; no real
// fixture does) would still be caught, since normalizeFileURI only
// strips the scheme prefix and never touches the file extension.
func hasTSExtensionFrame(rawTrace string) bool {
	for _, line := range strings.Split(rawTrace, "\n") {
		frame, ok := parseFrameLine(line)
		if !ok {
			continue
		}
		if strings.HasSuffix(frame.FilePath, ".ts") || strings.HasSuffix(frame.FilePath, ".tsx") {
			return true
		}
	}
	return false
}

// parseEngine is the runner-injectable shared implementation behind both
// javascriptParser.Parse and typescriptParser.Parse -- language only
// affects Detect(), not parsing itself, per plan.md's Architecture
// section. Composes T002-T009's chain parsing (parseTrace) with T008's
// bucketing and T005's runtime/version resolution (plan.md's pipeline
// steps 4-5), mirroring internal/codecontext/context.go's
// buildCodeContexts/BuildCodeContexts split: this function takes the
// nodeVersionRunner as an explicit parameter so it can be exercised
// directly with a fake in typescript_test.go, without a real "node
// --version" subprocess call in the test path; the two Parse() methods
// below are the production entry points that hardcode
// execNodeVersionRunner{}.
//
// Per registry.go's LanguageParser.Parse contract, the outcome is
// strictly binary: chain parses successfully into non-nil chain + a
// fully-populated Runtime with a nil error, or a nil chain + zero-value
// Runtime with a non-nil error -- never a partial result. The
// zero-value contract.Runtime{} on error (Name == "") is deliberate,
// not an oversight: FR13's "Runtime.Name is always node" applies to
// successful parses, not the error case, where there is no Runtime to
// describe at all.
func parseEngine(ctx context.Context, rawTrace string, runner nodeVersionRunner) ([]contract.ExceptionNode, contract.Runtime, error) {
	chain, err := parseTrace(rawTrace)
	if err != nil {
		return nil, contract.Runtime{}, err
	}

	// Bucketing runs per-frame across the whole chain (every node, not
	// just the outermost), after all blocks are parsed -- plan.md's
	// pipeline step 4: bucketing rules don't depend on chain position.
	for i := range chain {
		for j := range chain[i].Frames {
			chain[i].Frames[j] = assignBucket(chain[i].Frames[j])
		}
	}

	runtime := contract.Runtime{Name: "node"} // FR13: always "node" for v1 (Bun/Deno deferred)

	if version, ok := extractTraceVersion(rawTrace); ok {
		// FR14: shape (a) only -- a trailing "Node.js vX.Y.Z" line was
		// actually present in the trace text itself, the most trustworthy
		// source. No Note: Runtime.Note's own doc comment says it's
		// "omitted when VersionSource is VersionSourceTrace" -- there's
		// nothing uncertain to explain here.
		runtime.Version = version
		runtime.VersionSource = contract.VersionSourceTrace
	} else {
		// FR15/16: shape (b)/(c), or shape (a) without a parseable version
		// line -- fall back to the local environment. localNodeVersion
		// (T005) already returns the complete Version/VersionSource/Note
		// trio for both its success and failure outcomes (including
		// VersionSourceUnknown with no Note, per its own doc comment), so
		// no further branching is needed here.
		version, source, note := localNodeVersion(ctx, runner)
		runtime.Version = version
		runtime.VersionSource = source
		runtime.Note = note
	}

	return chain, runtime, nil
}

// javascriptParser implements parser.LanguageParser for plain JavaScript
// Node.js traces (spec.md FR5).
type javascriptParser struct{}

var _ parser.LanguageParser = javascriptParser{}

// NewJavaScriptParser returns the javascriptParser LanguageParser value.
func NewJavaScriptParser() parser.LanguageParser { return javascriptParser{} }

// Language implements parser.LanguageParser. Static per value (FR5) --
// see registry.go's own doc comment on why this is provisional pending
// 003b, not a decision this feature is reopening.
func (javascriptParser) Language() contract.Language { return contract.LanguageJavaScript }

// Detect implements parser.LanguageParser (FR4/FR6): looks like a Node
// trace AND has no TypeScript-extension frame. The !hasTSExtensionFrame
// half is what guarantees javascriptParser and typescriptParser never
// both match the same real trace (FR6's own note, exercised directly in
// typescript_test.go).
func (javascriptParser) Detect(rawTrace string) bool {
	return detectNodeTrace(rawTrace) && !hasTSExtensionFrame(rawTrace)
}

// Parse implements parser.LanguageParser: the production entry point,
// hardcoding the real execNodeVersionRunner{} (constitution Article
// VII: shell out, don't embed). See parseEngine's doc comment for why
// the runner-injectable core is a separate function.
func (javascriptParser) Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error) {
	return parseEngine(ctx, rawTrace, execNodeVersionRunner{})
}

// typescriptParser implements parser.LanguageParser for TypeScript
// Node.js traces (spec.md FR5) -- see javascriptParser's doc comments
// above for the shared reasoning; this type differs only in Language()
// and Detect()'s hasTSExtensionFrame polarity.
type typescriptParser struct{}

var _ parser.LanguageParser = typescriptParser{}

// NewTypeScriptParser returns the typescriptParser LanguageParser value.
func NewTypeScriptParser() parser.LanguageParser { return typescriptParser{} }

// Language implements parser.LanguageParser.
func (typescriptParser) Language() contract.Language { return contract.LanguageTypeScript }

// Detect implements parser.LanguageParser (FR4/FR6).
func (typescriptParser) Detect(rawTrace string) bool {
	return detectNodeTrace(rawTrace) && hasTSExtensionFrame(rawTrace)
}

// Parse implements parser.LanguageParser -- identical body to
// javascriptParser.Parse; language only affects Detect(), not parsing
// (plan.md's Architecture section).
func (typescriptParser) Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error) {
	return parseEngine(ctx, rawTrace, execNodeVersionRunner{})
}

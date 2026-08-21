package typescript

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// frameLinePattern matches a single V8 stack-frame line, once leading
// whitespace has already been stripped by the caller (parseFrameLine).
// Indentation is not part of this pattern because it varies with
// cause-chain nesting depth -- 2 extra spaces per [cause] level,
// confirmed via testdata/crash-3level-cause.txt (spec.md FR4) -- so
// anchoring to a fixed column count would reject real, valid frames.
//
// Accepts:
//   - the described form: "at <description> (<file>:<line>:<col>)"
//   - the bare form (no function name): "at <file>:<line>:<col>"
//   - an optional "async " modifier V8 inserts for awaited frames,
//     between "at " and either form above (confirmed real, see
//     testdata/esm-runtime-error.txt and
//     testdata/aggregate-error-uncaught.txt)
//   - an optional trailing "{" when the line is the last frame before a
//     [cause]: block (spec.md FR4), in either form
var frameLinePattern = regexp.MustCompile(
	`^at\s+(?:async\s+)?(?:(.+?)\s+\(([^()]+)\)|(\S+))(?:\s*\{)?$`,
)

// locationPattern splits a "<file>:<line>:<col>" location string. The
// leading capture is greedy so a file path that itself contains colons
// (e.g. Node's "node:internal/modules/cjs/loader", or a "file://" URI)
// is preserved intact -- only the trailing two colon-delimited numeric
// segments are taken as line/column.
var locationPattern = regexp.MustCompile(`^(.+):(\d+):(\d+)$`)

// parseFrameLine parses a single raw trace line into a contract.Frame.
// Only FilePath, ClassName, MethodName, LineNumber, and ColumnNumber are
// set. Frame.Index, Frame.Bucket, and Frame.PackageName are left at
// their zero values -- Index is assigned once a block's full frame list
// is assembled (engine.go's block-parsing pass), Bucket/PackageName by
// bucket.go (T008). FilePath is returned exactly as it appears in the
// trace, including any "file://" prefix -- URI normalization (spec.md
// FR12) is a separate later step, not this function's job.
//
// Returns ok == false, never an error, for any line that isn't a frame
// line: non-matching text, a "key: value," property line (spec.md FR7),
// a blank line, or a closing brace. This is an expected, routine outcome
// during block parsing (T006), not a failure condition.
func parseFrameLine(line string) (frame contract.Frame, ok bool) {
	trimmed := strings.TrimLeft(line, " ")

	m := frameLinePattern.FindStringSubmatch(trimmed)
	if m == nil {
		return contract.Frame{}, false
	}

	desc, described, bare := m[1], m[2], m[3]
	location := described
	if location == "" {
		location = bare
	}

	loc := locationPattern.FindStringSubmatch(location)
	if loc == nil {
		return contract.Frame{}, false
	}

	lineNum, err := strconv.Atoi(loc[2])
	if err != nil {
		return contract.Frame{}, false
	}
	colNum, err := strconv.Atoi(loc[3])
	if err != nil {
		return contract.Frame{}, false
	}

	frame = contract.Frame{
		FilePath:     loc[1],
		LineNumber:   lineNum,
		ColumnNumber: colNum,
	}

	if desc != "" {
		frame.ClassName, frame.MethodName = splitDescription(desc)
	}

	return frame, true
}

// versionLinePattern matches Node's trailing "Node.js vX.Y.Z" line,
// present on shape (a) (true uncaught crash) only per spec.md FR2/14.
var versionLinePattern = regexp.MustCompile(`(?m)^Node\.js v\d+\.\d+\.\d+$`)

// crashPreamblePattern matches the caret marker line that ends both
// crash-preamble variants (sync-throw and unhandled-promise-rejection,
// spec.md FR2) -- some amount of leading whitespace (variable, since V8
// positions the caret under the offending source column), then a single
// "^", then end of line. Detecting only the caret line, not the full
// two-variant preamble text, is enough for Detect()'s presence check;
// T004 does the full preamble parse (distinguishing which variant, for
// version extraction).
var crashPreamblePattern = regexp.MustCompile(`(?m)^\s*\^\s*$`)

// detectNodeTrace implements spec.md FR4's Detect() heuristic, shared by
// both javascriptParser and typescriptParser (they differ only in
// Language() and the extension check in Detect(), per FR5/FR6 -- see
// T010). Returns true only if condition (i) is met -- at least one V8
// frame-line match, OR the crash preamble is present, OR a trailing
// version line is present (the OR relaxation was added during T003; see
// spec.md FR4 for why: a real zero-frame fixture,
// testdata/zero-stack-trace-limit.txt, still carries the preamble and
// version line and would otherwise be incorrectly rejected) -- AND
// condition (ii): at least one Node-specific signal (a node: internal
// frame, a node_modules path segment, the version line, or the
// preamble).
//
// A bare .stack line with zero frames and none of the four signals
// (testdata/bare-stack-fetch-cause.txt) correctly returns false here --
// see spec.md FR4's "known residual gap" note. That's not an oversight;
// no heuristic can distinguish that input from an arbitrary one-line
// string in any language without reopening the false-positive risk
// condition (ii) exists to prevent.
func detectNodeTrace(rawTrace string) bool {
	hasFrameLine := false
	hasNodeInternalFrame := false
	hasNodeModulesFrame := false

	for _, line := range strings.Split(rawTrace, "\n") {
		frame, ok := parseFrameLine(line)
		if !ok {
			continue
		}
		hasFrameLine = true
		if strings.HasPrefix(frame.FilePath, "node:") {
			hasNodeInternalFrame = true
		}
		if strings.Contains(frame.FilePath, "node_modules") {
			hasNodeModulesFrame = true
		}
	}

	hasPreamble := crashPreamblePattern.MatchString(rawTrace)
	hasVersionLine := versionLinePattern.MatchString(rawTrace)

	if !hasFrameLine && !hasPreamble && !hasVersionLine {
		return false
	}

	return hasNodeInternalFrame || hasNodeModulesFrame || hasVersionLine || hasPreamble
}

// splitDescription splits a V8 frame description on its first "." into
// (ClassName, MethodName): "Object.<anonymous>" -> ("Object",
// "<anonymous>"), "Module.executeUserEntryPoint [as runMain]" ->
// ("Module", "executeUserEntryPoint [as runMain]"), "level1" (no dot)
// -> ("", "level1"). Not specified in spec.md/plan.md -- confirmed with
// Vedant during T002 as the working default. Revisit if a real fixture
// surfaces a description this doesn't split correctly (e.g. a
// meaningful second dot).
func splitDescription(desc string) (className, methodName string) {
	if idx := strings.Index(desc, "."); idx != -1 {
		return desc[:idx], desc[idx+1:]
	}
	return "", desc
}

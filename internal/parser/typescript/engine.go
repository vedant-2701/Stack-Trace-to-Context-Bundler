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

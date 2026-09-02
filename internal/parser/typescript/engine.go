package typescript

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
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
//   - the bare form (no function name): "at <file>:<line>:<col>" -- the
//     bare-location group excludes "("/")" (so a genuinely truncated
//     line that opened a described form's "(" but never got its
//     closing ")" still correctly fails to match at all, rather than
//     silently being reinterpreted as a valid bare frame -- this is
//     what T009's truncation tolerance, e.g.
//     testdata/truncated-mid-frame.txt, depends on) but is otherwise
//     permissive of any character INCLUDING spaces, so a bare file path
//     containing a space (real on macOS/Windows, and not structurally
//     impossible on Linux either) is still recognized, not silently
//     dropped as "not a frame line." The group is non-greedy (`+?`),
//     not greedy: a greedy version would swallow a trailing " {" (the
//     last-frame-before-[cause] case below) into the location itself,
//     since "{" isn't excluded by the character class either --
//     non-greedy correctly stops at the shortest match that still lets
//     the optional trailing-brace group below match, exactly
//     reproducing \S+'s original stop-before-any-whitespace behavior
//     for that case while additionally permitting a space that ISN'T
//     immediately followed by " {" (confirmed by hand-tracing both
//     testdata/nested-deps-flat.txt's plain bare frames and
//     testdata/crash-with-cause.txt's bare-frame-with-trailing-brace
//     case against this exact pattern).
//   - an optional "async " modifier V8 inserts for awaited frames,
//     between "at " and either form above (confirmed real, see
//     testdata/esm-runtime-error.txt and
//     testdata/aggregate-error-uncaught.txt)
//   - an optional trailing "{" when the line is the last frame before a
//     [cause]: block (spec.md FR4), in either form
var frameLinePattern = regexp.MustCompile(
	`^at\s+(?:async\s+)?(?:(.+?)\s+\(([^()]+)\)|([^()]+?))(?:\s*\{)?$`,
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
// Capturing group 1 is the version number, used by extractTraceVersion
// (T004); MatchString callers (detectNodeTrace) that only need presence
// are unaffected by the capture group.
var versionLinePattern = regexp.MustCompile(`(?m)^Node\.js v(\d+\.\d+\.\d+)$`)

// crashPreamblePattern matches the caret marker line that ends both
// crash-preamble variants (sync-throw and unhandled-promise-rejection,
// spec.md FR2) -- some amount of leading whitespace (variable, since V8
// positions the caret under the offending source column), then one or
// more "^" characters, then end of line. Multiple carets is real, not a
// defensive guess: testdata/import-outside-module.txt emits "^^^^^^",
// not a single "^" (confirmed during T004; the original T003 version of
// this pattern only matched a single caret and would have silently
// missed this real case -- it happened not to affect any T003 test
// outcome only because that fixture also has a node: internal frame
// satisfying detectNodeTrace's other OR-branch, but the gap was real).
// Detecting only the caret line, not the full preamble text, is enough
// for Detect()'s presence check; T004's crashPreambleBlockPattern below
// does the fuller three-line structural match needed for version
// extraction, where a bare caret line alone would be too loose a signal
// to attribute VersionSourceTrace to.
var crashPreamblePattern = regexp.MustCompile(`(?m)^\s*\^+\s*$`)

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
		if _, isDependency := splitAfterLastNodeModules(frame.FilePath); isDependency {
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

// looksLikeFrameLine reports whether line's trimmed form starts with
// V8's frame-line marker "at ", the same prefix frameLinePattern itself
// anchors on. Used to distinguish a genuinely truncated frame line
// (spec.md FR17/FR18) from ordinary multi-line message text -- no real
// message line in any captured fixture starts with this literal prefix
// (confirmed against every multi-line-message fixture:
// testdata/scoped-package-swc-false-caused-by.txt,
// testdata/assert-multiline-diff.txt), but this remains a heuristic: a
// message that coincidentally started with the word "at" would be
// misclassified. Accepted risk, same category as every other heuristic
// in this grammar.
func looksLikeFrameLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " "), "at ")
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

// crashPreambleBlockPattern matches the full three-line crash-preamble
// shape shared by both spec.md FR2 variants: a marker line ending in
// ":<N>" (the source file path for the synchronous-throw variant, or
// the literal "node:internal/process/promises:<N>" for the
// unhandled-promise-rejection variant), a non-blank content line (the
// offending source line, or Node's own internal
// triggerUncaughtException(...) line), then a caret-marker line. Both
// variants share this exact three-line structure -- confirmed against
// testdata/crash-with-cause.txt (sync) and
// testdata/crash-async-rejection-preamble.txt (async) -- so one pattern
// covers both rather than two near-duplicate ones (Article IV).
//
// Deliberately stricter than detectNodeTrace's crashPreamblePattern
// (caret-line only): gating trace-sourced version extraction (FR14) on
// the fuller three-line structure is more precise than gating it on a
// bare caret line, which in principle could appear for unrelated
// reasons. A false negative here (preamble present but not matched)
// only costs falling back to T005's local shell-out, not a wrong
// answer -- so this can afford to be conservative.
var crashPreambleBlockPattern = regexp.MustCompile(
	`(?m)^.+:\d+\n.*\S.*\n\s*\^+\s*$`,
)

// extractTraceVersion returns the version parsed from a trailing
// "Node.js vX.Y.Z" line, and whether one was found. Per spec.md FR14,
// this only applies to shape (a) (true uncaught crash): the crash
// preamble must be present first. A logged-object dump (shape b) or
// bare .stack log (shape c) never legitimately carries a trailing
// version line, so a coincidental version-looking line without the
// preamble present is not attributed to VersionSourceTrace -- that
// attribution belongs to T010's composition step, this function only
// supplies the raw extraction.
//
// If more than one version-line match exists in the text (not expected
// in a real trace, but not structurally impossible), the last one is
// used, matching Node's own convention of appending it at the very end
// of stdout.
func extractTraceVersion(rawTrace string) (version string, ok bool) {
	if !crashPreambleBlockPattern.MatchString(rawTrace) {
		return "", false
	}

	matches := versionLinePattern.FindAllStringSubmatch(rawTrace, -1)
	if len(matches) == 0 {
		return "", false
	}

	last := matches[len(matches)-1]
	return last[1], true
}

// elisionLinePattern matches Node's frame-elision line (spec.md FR8),
// confirmed real via testdata/crash-with-cause.txt and every other
// real multi-node fixture: "... N lines matching cause stack trace
// ...", indented to match the surrounding frame list (indentation isn't
// part of this pattern for the same reason frameLinePattern ignores it
// -- it grows with [cause] nesting depth).
var elisionLinePattern = regexp.MustCompile(
	`^\s*\.\.\. (\d+) lines matching cause stack trace \.\.\.\s*$`,
)

// parseElisionLine reports whether line is Node's frame-elision marker,
// returning the elided count if so.
func parseElisionLine(line string) (count int, ok bool) {
	m := elisionLinePattern.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// leadingSpaces returns the count of leading space characters in line.
// Node's util.inspect indents consistently with spaces, never tabs, in
// every real fixture captured for this feature.
func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

// lineOpensBody reports whether line's trimmed form ends with "{",
// meaning a brace body ([cause]:/[errors]:/dropped-property content,
// closed by a matching "}") follows this line.
func lineOpensBody(line string) bool {
	return strings.HasSuffix(strings.TrimRight(line, " "), "{")
}

// splitHeaderLine parses one exception node's own header line (already
// advanced past any "[cause]: " prefix by the caller -- see
// blockParser.parseNodeAndCause) into its ClassName and the first line
// of its message, and reports whether it used the zero-frame bracket
// form -- "[ClassName: message]", which util.inspect uses specifically
// when the exception has no "at" frames to render inline (confirmed
// real via testdata/logged-object-fetch-cause.txt's outer node,
// testdata/zero-stack-trace-limit.txt, and
// testdata/aggregate-error-uncaught.txt) -- or the plain form,
// "ClassName: message", used whenever at least one frame follows. Also
// reports whether this line's own trailing "{" opens a brace body --
// only meaningful for the bracket form; the plain form's body-opening
// brace, if any, is always on its LAST FRAME line instead (checked
// separately via lineOpensBody, since collectMessageAndFrames finds
// that line, not this function).
//
// The split point used is the FIRST ": " on the line, not the first
// "[": ClassName can itself carry a bracketed error-code suffix (e.g.
// "AssertionError [ERR_ASSERTION]", confirmed real via
// testdata/assert-multiline-diff.txt) that must stay part of ClassName,
// not be mistaken for the bracket form's own delimiters.
func splitHeaderLine(line string) (className, message string, isBracketForm, opensBody bool, ok bool) {
	trimmed := strings.TrimRight(line, " ")
	opensBody = strings.HasSuffix(trimmed, "{")
	if opensBody {
		trimmed = strings.TrimRight(strings.TrimSuffix(trimmed, "{"), " ")
	}

	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		inner := trimmed[1 : len(trimmed)-1]
		idx := strings.Index(inner, ": ")
		if idx == -1 {
			return "", "", false, false, false
		}
		return inner[:idx], inner[idx+2:], true, opensBody, true
	}

	idx := strings.Index(trimmed, ": ")
	if idx == -1 {
		return "", "", false, false, false
	}
	return trimmed[:idx], trimmed[idx+2:], false, opensBody, true
}

// blockParser holds the lines of a trace body (after any crash preamble
// has already been skipped, see skipPreamble) and a cursor, so nested
// [cause]: blocks can be parsed via plain recursive calls
// (blockParser.parseNodeAndCause) without threading index state through
// every return value.
type blockParser struct {
	lines []string
	pos   int
}

func (p *blockParser) peek() (string, bool) {
	if p.pos >= len(p.lines) {
		return "", false
	}
	return p.lines[p.pos], true
}

func (p *blockParser) next() (string, bool) {
	line, ok := p.peek()
	if ok {
		p.pos++
	}
	return line, ok
}

// skipPreamble strips everything up through and including the crash
// preamble's caret line (spec.md FR2), reusing
// crashPreambleBlockPattern rather than duplicating its structure
// (Article IV). Searches rather than anchors to text start, so any
// noise before the preamble (e.g. testdata/import-outside-module.txt's
// leading "(node:N) Warning: ..." lines) is naturally skipped too
// without special-casing. For shape (b)/(c), which have no preamble at
// all, crashPreambleBlockPattern doesn't match and rawTrace is returned
// unchanged -- the header is already the first line for those shapes.
func skipPreamble(rawTrace string) string {
	loc := crashPreambleBlockPattern.FindStringIndex(rawTrace)
	if loc == nil {
		return rawTrace
	}
	return strings.TrimLeft(rawTrace[loc[1]:], "\n")
}

// collectMessageAndFrames consumes lines starting at the parser's
// current position, for a node whose header used the plain (non-bracket)
// form: message-continuation lines (anything before the first real
// frame or elision line -- multi-line messages are real, confirmed via
// testdata/scoped-package-swc-false-caused-by.txt's embedded JSON blob
// and false "Caused by:" text, and testdata/assert-multiline-diff.txt's
// diff body, neither of which starts with "at " so neither is ever
// mistaken for a frame), then frame/elision lines in order (frame order
// is preserved exactly as encountered, satisfying spec.md FR21's
// Frames[0]-is-nearest-origin requirement by construction). Stops when
// either a consumed frame line's trailing "{" opens a body (opensBody
// returns true), or -- once at least one frame has been collected -- a
// line matches neither pattern (frame zone ended without a body; that
// line is left unconsumed for the caller/an enclosing recursive call to
// see, e.g. an enclosing block's own closing "}").
func (p *blockParser) collectMessageAndFrames() (messageLines []string, frames []contract.Frame, elided int, opensBody, truncated bool) {
	inFrameZone := false
	frames = []contract.Frame{}
	for {
		line, ok := p.peek()
		if !ok {
			return messageLines, frames, elided, false, false
		}

		if n, isElision := parseElisionLine(line); isElision {
			elided += n
			p.next()
			inFrameZone = true
			continue
		}

		if frame, isFrame := parseFrameLine(line); isFrame {
			frame.Index = len(frames)
			frames = append(frames, frame)
			opens := lineOpensBody(line)
			p.next()
			inFrameZone = true
			if opens {
				return messageLines, frames, elided, true, false
			}
			continue
		}

		if looksLikeFrameLine(line) {
			// spec.md FR17: this line was clearly attempting to be a frame
			// line ("at ...") but failed frameLinePattern -- truncated
			// mid-frame, not message text. Drop it, warn, and stop:
			// whatever's already been collected (possibly zero frames,
			// e.g. a [cause] node cut off before its first frame -- see
			// parseNodeAndCause's FR18 handling of this truncated signal)
			// is the correct partial result. Nothing meaningful can follow
			// a truncated frame line within this same block.
			slog.Warn("dropping truncated frame line", "line", line)
			p.next()
			return messageLines, frames, elided, false, true
		}

		if !inFrameZone {
			messageLines = append(messageLines, line)
			p.next()
			continue
		}

		return messageLines, frames, elided, false, false
	}
}

// skipErrorsArray drops an AggregateError's "[errors]: [...]" array
// entirely, per spec.md's Out of scope section (branching chains are
// not supported; the array's nested errors are dropped, not partially
// parsed -- confirmed real via testdata/aggregate-error-uncaught.txt).
// arrayIndent is the leading-space count of the "[errors]: [" line
// itself; the matching close is the next line whose trimmed content is
// exactly "]" at that same indent -- confirmed real: util.inspect
// closes the array at the same indent it opened, same pattern as a
// node's own "{"/"}" body (see parseNodeAndCause's headerIndent
// matching). Assumes a multi-line array (the only real shape captured);
// a single-line "[errors]: []" isn't handled specially, since no real
// fixture exercises it.
func (p *blockParser) skipErrorsArray(arrayIndent int) {
	for {
		line, ok := p.peek()
		if !ok {
			return
		}
		p.next()
		if strings.TrimLeft(line, " ") == "]" && leadingSpaces(line) == arrayIndent {
			return
		}
	}
}

// parseNodeAndCause parses one exception node starting at the parser's
// current position, which must already be positioned exactly at that
// node's header line (for a nested cause, the caller strips the
// "[cause]: " prefix from that line first -- see the body loop below).
// Consumes everything belonging to it: header, message, frames, and --
// if a body follows -- dropped property lines, a dropped [errors] array
// (spec.md's Out-of-scope AggregateError handling), and/or exactly one
// nested [cause] node (Bundle.Chain is strictly linear per 001's
// contract, so there is never more than one [cause] per block).
//
// Returns this node plus every node found by following its cause chain
// (this node first) -- spec.md FR7. Returns ok == false only when the
// current line isn't a valid header at all (no ClassName/message split
// point found) -- parseTrace (below) wraps that into
// parser.ErrUnparseable; this function stays a raw primitive,
// consistent with parseFrameLine/detectNodeTrace/
// extractTraceVersion's established pattern. The third return value,
// truncated, is T009's addition (spec.md FR17/FR18): reports whether
// THIS node's own header/frame collection was cut off mid-parse --
// never whether a descendant cause node was truncated and dropped,
// which is already fully resolved (see the [cause]-handling branch
// below) before this function returns.
func (p *blockParser) parseNodeAndCause() ([]contract.ExceptionNode, bool, bool) {
	headerLine, ok := p.next()
	if !ok {
		return nil, false, false
	}

	headerIndent := leadingSpaces(headerLine)
	className, firstMessageLine, isBracketForm, headerOpensBody, ok := splitHeaderLine(strings.TrimLeft(headerLine, " "))
	if !ok {
		return nil, false, false
	}

	messageLines := []string{firstMessageLine}
	frames := []contract.Frame{}
	elided := 0
	opensBody := headerOpensBody
	truncated := false

	// The bracket form always means zero frames (that's why util.inspect
	// uses it) and a fully self-contained single-line message -- no
	// further message/frame collection needed or correct to attempt.
	if !isBracketForm {
		var moreMessage []string
		moreMessage, frames, elided, opensBody, truncated = p.collectMessageAndFrames()
		messageLines = append(messageLines, moreMessage...)
	}

	node := contract.ExceptionNode{
		ClassName:        className,
		Message:          strings.Join(messageLines, "\n"),
		ElidedFrameCount: elided,
		Frames:           frames,
	}

	if len(node.Frames) == 0 && !truncated {
		// spec.md FR19: a message-only error with zero frames (e.g.
		// Error.stackTraceLimit = 0, or an outer node in a logged-object
		// dump that util.inspect renders in the zero-frame bracket form,
		// confirmed real via testdata/logged-object-fetch-cause.txt's
		// outer node) is a valid, complete degraded result, not a
		// failure -- but still worth surfacing via slog.Warn since a
		// caller/AI assistant consuming the bundle has no own-code
		// frames to work with for this node. Gated on !truncated so this
		// doesn't double-warn on top of collectMessageAndFrames' own
		// FR17 warning for the same underlying cause.
		slog.Warn("exception node has zero frames", "className", className)
	}

	chain := []contract.ExceptionNode{node}

	if !opensBody {
		return chain, true, truncated
	}

	for {
		line, ok := p.peek()
		if !ok {
			// Body opened ("... {") but ran out before a closing "}" was
			// ever found, and no [cause]: was found either (that case is
			// handled below, inside the [cause]-handling branch). Nothing
			// was ever started that needs dropping, so returning what's
			// already been built (just this node) is already the correct
			// result -- still worth a warning for visibility.
			slog.Warn("exception body truncated before closing brace", "className", className)
			break
		}

		lineIndent := leadingSpaces(line)
		trimmedLine := strings.TrimLeft(line, " ")

		if trimmedLine == "}" && lineIndent == headerIndent {
			p.next()
			break
		}

		if strings.HasPrefix(trimmedLine, "[cause]: ") {
			// Strip the "[cause]: " marker in place, preserving the
			// line's own leading whitespace, so the recursive call parses
			// the remainder as an ordinary header line.
			p.lines[p.pos] = line[:lineIndent] + strings.TrimPrefix(trimmedLine, "[cause]: ")
			causeChain, causeOK, causeTruncated := p.parseNodeAndCause()
			if !causeOK {
				break
			}
			if causeTruncated {
				// spec.md FR18: the cause chain opened ([cause]: appeared)
				// but never resolved -- trace was cut off mid-cause. Keep
				// this node (already in chain), drop the incomplete cause
				// entirely rather than keeping it with corrupted/partial
				// content (confirmed real via
				// testdata/cutoff-cause-chain.txt, where the cause's own
				// truncated frame line would otherwise be swallowed into
				// its Message), and stop: input is exhausted, nothing more
				// to find.
				slog.Warn("dropping incomplete cause node -- trace cut off mid-cause",
					"outerClassName", className)
				break
			}
			chain = append(chain, causeChain...)
			continue
		}

		if strings.HasPrefix(trimmedLine, "[errors]: [") {
			p.next()
			p.skipErrorsArray(lineIndent)
			slog.Warn("dropping [errors] array -- AggregateError branching not supported, degraded single-node parse",
				"className", className)
			continue
		}

		// Anything else at this level is a dropped property line
		// (spec.md FR7 -- e.g. errno/code/syscall on a system error, or
		// a top-level code: 'GenericFailure'). Not surfaced on
		// contract.ExceptionNode, which has no field for it.
		slog.Warn("dropping unrecognized property line in exception body", "line", line)
		p.next()
	}

	return chain, true, truncated
}

// parseChain is T006's entry point: parses the [cause]: bracket-chain
// structure (shapes a/b, spec.md FR7) from a raw, unmodified trace --
// including any crash preamble, which this function strips itself via
// skipPreamble -- into a linear chain of contract.ExceptionNode, plus
// ElidedFrameCount per node (FR8). Returns ok == false only when no
// valid exception header can be found at all. Discards
// parseNodeAndCause's own truncated bool -- nothing at this level
// needs it; the FR17/FR18 tolerance behavior it drives is already fully
// resolved (warned and, where required, dropped) inside
// parseNodeAndCause/collectMessageAndFrames before this function ever
// sees the result.
func parseChain(rawTrace string) ([]contract.ExceptionNode, bool) {
	body := skipPreamble(rawTrace)
	p := &blockParser{lines: strings.Split(body, "\n")}
	chain, ok, _ := p.parseNodeAndCause()
	return chain, ok
}

// parseTrace is T009's addition on top of parseChain: converts
// parseChain's binary ok bool into an actual parser.ErrUnparseable
// -wrapped error (spec.md FR20), for T010's Parse() composition to
// return directly. Kept as a separate function rather than changing
// parseChain's own signature, since engine_test.go's existing
// TestParseChain* table tests all assert against the ok-bool form and
// there's no reason to disturb that; parseChain also stays consistent
// with parseFrameLine/detectNodeTrace/extractTraceVersion's established
// raw-primitive-returns-ok-bool pattern.
func parseTrace(rawTrace string) ([]contract.ExceptionNode, error) {
	chain, ok := parseChain(rawTrace)
	if !ok {
		return nil, fmt.Errorf("%w: no valid exception header found", parser.ErrUnparseable)
	}
	return chain, nil
}

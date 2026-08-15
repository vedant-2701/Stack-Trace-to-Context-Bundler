package codecontext

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// DefaultContextLines is the number of source lines requested before and
// after a frame's target line when building its snippet window (spec.md
// requirement 1: 5 before, the target line, 5 after -- 11 lines total,
// before any clamping at file boundaries). Exported so feature 011
// (specs/INDEX.md) can reference this feature's default instead of
// duplicating the number when it exposes the window size as a
// --context-lines flag.
const DefaultContextLines = 5

// errEmptyFile is returned by buildSnippet when the target file exists
// and is readable but has zero lines. Kept as its own sentinel rather
// than reusing fs.ErrNotExist or a generic wrap: this is a genuinely
// different situation from "not found" or "unreadable" (spec.md
// requirement 2's two named cases), and a note claiming the file wasn't
// found would misdescribe a file that's actually right there, just
// empty. Still surfaces through contract.StatusNotFound, the same
// reused-status approach requirement 2 already uses for
// permission-denied -- a 4th CodeContextStatus value was already
// considered and rejected once in this feature's plan.md for that case,
// for the same reason (not worth another MAJOR-bump contract change for
// a distinction the note field can carry).
var errEmptyFile = errors.New("file has no lines")

// buildSnippet extracts a fixed-size text window of source lines around
// targetLine in the file at path (spec.md requirement 1), clamped to the
// file's actual first/last line when the window would run past either
// boundary. Reads only the needed line range, not the whole file into
// memory (non-functional requirement): lines before the window are
// scanned and discarded one at a time, and scanning stops as soon as the
// window's end line is reached, so a huge file with an early target line
// is never read past what's actually needed.
//
// A non-nil error means the file could not be extracted at all --
// missing, unreadable, or present-but-empty (spec.md requirement 2's two
// named cases, plus errEmptyFile above; all three reuse
// contract.StatusNotFound). This function doesn't decide status/note
// wording itself -- callers distinguish these via errors.Is(err,
// fs.ErrNotExist) / errors.Is(err, errEmptyFile) to build requirement 2's
// distinguishing note text, the same "caller decides status/note" split
// blame.go (T006) also uses.
func buildSnippet(path string, targetLine, contextLines int) (contract.Snippet, error) {
	f, err := os.Open(path)
	if err != nil {
		return contract.Snippet{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	start := targetLine - contextLines
	if start < 1 {
		start = 1
	}
	end := targetLine + contextLines

	var lines []string
	lastLine := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxScannerLineBytes)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		lastLine = lineNum
		if lineNum < start {
			continue
		}
		if lineNum > end {
			break
		}
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return contract.Snippet{}, fmt.Errorf("read %s: %w", path, err)
	}

	if lastLine == 0 {
		return contract.Snippet{}, fmt.Errorf("%s: %w", path, errEmptyFile)
	}

	actualEnd := end
	if lastLine < end {
		actualEnd = lastLine
	}
	// start can also exceed the file's actual length entirely (e.g. a
	// target line referencing a file that's since shrunk) -- clamp so
	// StartLine never exceeds EndLine in that degenerate case.
	actualStart := start
	if actualStart > actualEnd {
		actualStart = actualEnd
	}

	code := strings.Join(lines, "\n")
	if len(lines) > 0 {
		code += "\n"
	}

	return contract.Snippet{
		StartLine:  actualStart,
		EndLine:    actualEnd,
		TargetLine: targetLine,
		Code:       code,
	}, nil
}

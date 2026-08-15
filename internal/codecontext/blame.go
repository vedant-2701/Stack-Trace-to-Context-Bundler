package codecontext

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// buildBlame runs `git blame --porcelain -L start,end <file>` for a
// tracked, clean own-bucket frame's file (spec.md requirement 6), over
// the already-clamped snippet range (T004's StartLine/EndLine, not the
// raw pre-clamp window), with cwd set to the file's own directory --
// same "no separately-tracked repo root" approach checkFileStatus (T005)
// and BuildGitMetadata (T003) both use.
//
// A non-nil error means blame could not be produced at all: the git
// command failed or timed out, or its porcelain output couldn't be
// parsed (spec.md requirements 7, 8). This function doesn't decide
// status/note wording -- callers (context.go, T007) do, the same
// "caller decides" split snippet.go and status.go both use.
func buildBlame(ctx context.Context, filePath string, startLine, endLine int, runner gitRunner) ([]contract.BlameEntry, error) {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	rangeArg := fmt.Sprintf("%d,%d", startLine, endLine)

	out, err := runner.Run(ctx, dir, "blame", "--porcelain", "-L", rangeArg, base)
	if err != nil {
		return nil, fmt.Errorf("git blame %s: %w", base, err)
	}

	entries, err := parseBlamePorcelain(out)
	if err != nil {
		return nil, fmt.Errorf("parse git blame output for %s: %w", base, err)
	}
	return entries, nil
}

// commitMeta is the subset of a commit's porcelain metadata this feature
// needs, cached by commit hash across a single buildBlame call.
type commitMeta struct {
	author     string
	commitDate string
	summary    string
}

// parseBlamePorcelain parses `git blame --porcelain`'s stdout into
// contract.BlameEntry, one entry per contiguous same-commit group
// exactly as git blame -L itself groups its output -- this parser
// leans entirely on git's own grouping (the header line's optional 4th
// field, present only on a group's first line) rather than re-deriving
// group boundaries by comparing consecutive commit hashes itself.
//
// Per porcelain's format, a commit's full metadata block (author,
// author-time, author-tz, summary, etc.) is only printed the first time
// that commit hash appears anywhere in this output; a later,
// non-contiguous group attributed to the same commit gets only the
// header line (still with its own 4th group-size field) followed
// straight by "filename" (no metadata repeated). This function caches
// each commit's metadata by hash the first time it's seen and reuses it
// for later groups that don't repeat it.
func parseBlamePorcelain(out string) ([]contract.BlameEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), maxScannerLineBytes)
	cache := map[string]commitMeta{}
	var entries []contract.BlameEntry

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 || !isHexSHA(fields[0]) {
			return nil, fmt.Errorf("unexpected line in git blame output, wanted a header: %q", line)
		}
		sha := fields[0]
		finalLine, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("parse final line number in %q: %w", line, err)
		}

		if len(fields) == 3 {
			// Continuation of the currently-open group: no metadata
			// repeated, the very next line is this line's file
			// content -- discard it and move on to the next header.
			if !scanner.Scan() {
				return nil, fmt.Errorf("git blame output ended mid-group after %q", line)
			}
			continue
		}

		groupSize, err := strconv.Atoi(fields[3])
		if err != nil {
			return nil, fmt.Errorf("parse group size in %q: %w", line, err)
		}

		cm, err := readGroupMetadata(scanner, sha, line, cache)
		if err != nil {
			return nil, err
		}

		entries = append(entries, contract.BlameEntry{
			StartLine:  finalLine,
			EndLine:    finalLine + groupSize - 1,
			CommitHash: sha,
			Author:     cm.author,
			CommitDate: cm.commitDate,
			Summary:    cm.summary,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan git blame output: %w", err)
	}

	return entries, nil
}

// readGroupMetadata consumes one blame group's metadata block -- the
// lines between a group-starting header and its terminating
// tab-prefixed content line -- and returns the resolved commitMeta for
// sha: either freshly parsed from this block, or reused from cache if
// this block was empty because porcelain already showed sha's metadata
// earlier in this same output (the non-contiguous-reappearance case).
// line is only used to give parse errors a useful location. Split out of
// parseBlamePorcelain to keep that function's branching within gocyclo's
// limit, and because this block is a genuinely separable unit: "resolve
// one group's commit metadata" from "walk the header stream and emit
// entries."
func readGroupMetadata(scanner *bufio.Scanner, sha, line string, cache map[string]commitMeta) (commitMeta, error) {
	meta := map[string]string{}
	sawContentLine := false
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "\t") {
			sawContentLine = true
			break // file content line, discard
		}
		key, val, _ := strings.Cut(l, " ")
		meta[key] = val
	}
	if !sawContentLine {
		// EOF (or a scan error) before this group's metadata block was
		// terminated by its content line -- malformed/truncated porcelain
		// output. execGitRunner already fails the whole call on a
		// non-zero git exit, so this should only happen if git exits 0
		// but writes output that doesn't match its own documented format;
		// fail loudly rather than silently emitting a BlameEntry from a
		// possibly-incomplete metadata block (Article VI).
		if err := scanner.Err(); err != nil {
			return commitMeta{}, fmt.Errorf("read git blame metadata for %q: %w", line, err)
		}
		return commitMeta{}, fmt.Errorf("git blame output ended before %q's metadata block was terminated by a content line", line)
	}

	if author, ok := meta["author"]; ok {
		commitDate, err := formatCommitDate(meta["author-time"], meta["author-tz"])
		if err != nil {
			return commitMeta{}, fmt.Errorf("commit %s: %w", sha, err)
		}
		cache[sha] = commitMeta{
			author:     author,
			commitDate: commitDate,
			summary:    meta["summary"],
		}
	}

	cm, ok := cache[sha]
	if !ok {
		// A repeat occurrence of a commit whose first occurrence was
		// never cached -- shouldn't happen with well-formed porcelain
		// output (git always shows full metadata the first time a commit
		// appears in this output), but fail loudly rather than silently
		// emitting an empty author/summary (Article VI).
		return commitMeta{}, fmt.Errorf("commit %s referenced before its metadata was seen in this output", sha)
	}
	return cm, nil
}

// isHexSHA reports whether s looks like a full 40-character git commit
// hash, distinguishing a porcelain header line from a metadata line
// (author, summary, filename, ...) at the start of each parse loop
// iteration.
func isHexSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// formatCommitDate combines porcelain's author-time (unix epoch seconds,
// as a decimal string) and author-tz (e.g. "+0530", "-0500") into an
// ISO 8601 timestamp, computed once here rather than left as a raw epoch
// for renderers to each format independently (constitution Article V).
func formatCommitDate(authorTime, authorTZ string) (string, error) {
	sec, err := strconv.ParseInt(authorTime, 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse author-time %q: %w", authorTime, err)
	}
	offsetSeconds, err := parseGitTZOffset(authorTZ)
	if err != nil {
		return "", err
	}
	loc := time.FixedZone(authorTZ, offsetSeconds)
	return time.Unix(sec, 0).In(loc).Format(time.RFC3339), nil
}

// parseGitTZOffset parses a git porcelain author-tz value like "+0530"
// or "-0500" into a signed offset in seconds east of UTC.
func parseGitTZOffset(tz string) (int, error) {
	if len(tz) != 5 || (tz[0] != '+' && tz[0] != '-') {
		return 0, fmt.Errorf("unexpected author-tz format: %q", tz)
	}
	hours, err := strconv.Atoi(tz[1:3])
	if err != nil {
		return 0, fmt.Errorf("parse author-tz %q: %w", tz, err)
	}
	minutes, err := strconv.Atoi(tz[3:5])
	if err != nil {
		return 0, fmt.Errorf("parse author-tz %q: %w", tz, err)
	}
	offset := hours*3600 + minutes*60
	if tz[0] == '-' {
		offset = -offset
	}
	return offset, nil
}

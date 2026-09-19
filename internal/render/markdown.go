// Package render turns a contract.Bundle into the clipboard-ready output
// formats consumers of this tool paste into third-party AI chats.
package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// Markdown renders the bundle as a single, self-contained Markdown document.
func Markdown(_ contract.Bundle) string {
	return ""
}

// renderPreamble returns the fixed, one-line blockquote that orients a
// reader who has no other context for this document (spec.md req. 1).
func renderPreamble() string {
	return "> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.\n"
}

// renderMetadata renders the compact metadata bullet list that follows
// the preamble: Language, OS, Runtime, Git (omitted when b.GitMetadata is
// nil), Fingerprint, in that fixed order (spec.md req. 2, 20, 21).
func renderMetadata(b contract.Bundle) string {
	lines := []string{
		fmt.Sprintf("- Language: %s", b.Language),
		fmt.Sprintf("- OS: %s", b.OS),
		"- " + renderRuntime(b.Runtime),
	}
	if b.GitMetadata != nil {
		lines = append(lines, "- "+renderGit(b.GitMetadata))
	}
	lines = append(lines, fmt.Sprintf("- Fingerprint: %s", b.Fingerprint))
	return strings.Join(lines, "\n") + "\n"
}

// renderRuntime renders the single "Runtime: ..." metadata line
// (spec.md req. 20). When VersionSource is VersionSourceTrace, the
// version is printed with no caveat. Otherwise (LocalEnvironment or
// Unknown), Runtime.Note (when present) is appended as an escaped
// parenthetical rather than the literal enum value ever being printed;
// when both Version and Note are absent, "(version unknown)" is used.
func renderRuntime(r contract.Runtime) string {
	if r.VersionSource == contract.VersionSourceTrace {
		return fmt.Sprintf("Runtime: %s %s", r.Name, r.Version)
	}
	if r.Version == "" && r.Note == "" {
		return fmt.Sprintf("Runtime: %s (version unknown)", r.Name)
	}
	s := "Runtime: " + r.Name
	if r.Version != "" {
		s += " " + r.Version
	}
	if r.Note != "" {
		s += " (" + escapeMarkdown(r.Note) + ")"
	}
	return s
}

// renderGit renders the single "Git: ..." metadata line (spec.md
// req. 21). g must be non-nil; callers are responsible for omitting this
// line entirely when Bundle.GitMetadata is nil.
func renderGit(g *contract.GitMetadata) string {
	status := "clean"
	if g.UncommittedChanges {
		status = "uncommitted changes"
	}
	commit := g.CurrentCommit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	return fmt.Sprintf("Git: %s @ %s (%s)", escapeMarkdown(g.Branch), commit, status)
}

// renderSnippet renders s as a fenced code block tagged with lang, each
// line prefixed with its real source line number (StartLine + offset)
// and the line matching TargetLine marked with a leading → (spec.md
// req. 11). s.Code is never escaped -- the fence protects it (req. 19).
//
// internal/codecontext.buildSnippet always appends exactly one trailing
// "\n" to Code beyond its real lines, regardless of whether the window's
// last source line is itself blank, so that trailing newline is trimmed
// before splitting rather than naively splitting and rendering every
// element -- otherwise a bogus blank EndLine+1 line would be appended to
// every rendered snippet.
func renderSnippet(s contract.Snippet, lang contract.Language) string {
	lines := strings.Split(strings.TrimSuffix(s.Code, "\n"), "\n")
	numWidth := len(strconv.Itoa(s.EndLine))

	rendered := make([]string, len(lines))
	for i, line := range lines {
		lineNum := s.StartLine + i
		marker := "  "
		if lineNum == s.TargetLine {
			marker = "→ "
		}
		rendered[i] = fmt.Sprintf("%s%*d | %s", marker, numWidth, lineNum, line)
	}

	return fmt.Sprintf("```%s\n%s\n```\n", lang, strings.Join(rendered, "\n"))
}

// renderBlameTable renders entries as a Markdown table with columns
// Lines | Commit | Author | Date | Summary, one row per entry (spec.md
// req. 12) -- never one row per line, matching how `git blame -L` itself
// groups contiguous ranges under one last-touching commit. Commit is the
// short (first 7 characters) hash. Date is CommitDate's date-only
// portion (YYYY-MM-DD): ISO 8601 is fixed-width up to that point, so a
// straight substring is safe regardless of what follows it. Author and
// Summary are developer-arbitrary text and go through escapeMarkdown --
// this is also what keeps a `|` in Summary from corrupting the table
// structure, since `|` is itself in the escaped character set.
func renderBlameTable(entries []contract.BlameEntry) string {
	var b strings.Builder
	b.WriteString("| Lines | Commit | Author | Date | Summary |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")

	for _, e := range entries {
		lines := strconv.Itoa(e.StartLine)
		if e.EndLine != e.StartLine {
			lines += "-" + strconv.Itoa(e.EndLine)
		}
		commit := e.CommitHash
		if len(commit) > 7 {
			commit = commit[:7]
		}
		date := e.CommitDate
		if len(date) > 10 {
			date = date[:10]
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			lines, commit, escapeMarkdown(e.Author), date, escapeMarkdown(e.Summary))
	}

	return b.String()
}

// renderFrame renders one Frame as a single line
// "at [ClassName.]MethodName (FilePath:LineNumber[:ColumnNumber]) —
// <bucket-suffix>" (spec.md req. 8), followed by that frame's own-code
// context (req. 9) when it's an own-bucket frame with a non-nil cc --
// cc is nil for every other bucket. FilePath is rendered verbatim, never
// escaped -- it's a normalized path, not developer-arbitrary prose
// (req. 19's escape list doesn't include it). The dependency suffix
// shows PackageName identity only, never a version (req. 13) -- that
// belongs solely to the Dependencies section.
func renderFrame(f contract.Frame, cc *contract.CodeContext) string {
	name := f.MethodName
	if f.ClassName != "" {
		name = f.ClassName + "." + name
	}

	location := fmt.Sprintf("%s:%d", f.FilePath, f.LineNumber)
	if f.ColumnNumber != 0 {
		location += fmt.Sprintf(":%d", f.ColumnNumber)
	}

	var suffix string
	switch f.Bucket {
	case contract.BucketOwn:
		suffix = "own"
	case contract.BucketDependency:
		suffix = "dependency: " + f.PackageName
	case contract.BucketRuntime:
		suffix = "runtime"
	}

	line := fmt.Sprintf("at %s (%s) — %s\n", name, location, suffix)
	if f.Bucket == contract.BucketOwn && cc != nil {
		line += renderCodeContext(*cc)
	}
	return line
}

// renderCodeContext renders one own-bucket frame's code context (spec.md
// req. 10-12). When Status is not_found or stale, only a flagged line
// using Note (escaped) is rendered -- no snippet or blame table. When
// Status is ok, the snippet always renders; a non-empty Blame renders as
// a table below it, otherwise a flagged Note line takes the table's
// place (e.g. no git repo found, or `git blame` itself failed/timed out).
func renderCodeContext(cc contract.CodeContext) string {
	if cc.Status == contract.StatusNotFound || cc.Status == contract.StatusStale {
		return fmt.Sprintf("⚠ %s\n", escapeMarkdown(cc.Note))
	}

	var b strings.Builder
	b.WriteString(renderSnippet(cc.Snippet, cc.Language))
	if len(cc.Blame) > 0 {
		b.WriteString(renderBlameTable(cc.Blame))
	} else {
		fmt.Fprintf(&b, "⚠ %s\n", escapeMarkdown(cc.Note))
	}
	return b.String()
}

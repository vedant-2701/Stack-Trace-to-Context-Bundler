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

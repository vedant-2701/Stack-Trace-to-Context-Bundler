package render

import "strings"

// escapeCharSet is the curated subset of CommonMark's ASCII-punctuation
// escape set that affects this document's Markdown constructs: headings,
// emphasis, links, code spans, tables, and blockquotes. See plan.md's
// "Escaping character set" section for why this subset (not the full
// CommonMark punctuation set) was chosen, and why '<' is included
// alongside '>'.
const escapeCharSet = "\\`*_{}[]()#+-.!|><"

// escapeMarkdown escapes Markdown special characters in s so that
// developer-arbitrary text (an exception message, a commit summary, a
// branch name, etc.) cannot be misread as Markdown structure when
// interpolated into prose, heading, or blockquote context. It must never
// be called on content that is already inside a fenced code block — the
// fence itself is what protects that content (spec.md requirement 19).
func escapeMarkdown(s string) string {
	// A single forward scan over s, escaping each rune in place as it's
	// written, is inherently safe regardless of escapeCharSet's order:
	// an inserted backslash is never itself re-scanned, so there is no
	// double-escaping risk to guard against here.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if strings.ContainsRune(escapeCharSet, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

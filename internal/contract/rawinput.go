package contract

import "unicode/utf8"

// RawInputCapBytes is the maximum size of Bundle.RawInput, per spec.md
// requirement 4: 512 KB using the standard binary convention (512*1024
// bytes), not decimal. The cap exists to bound total bundle size for
// clipboard/LLM-context use (spec.md non-functional requirements) -- a
// product constraint, not a technical limit, not expected to affect any
// realistic stack trace even at pathological recursion depth.
//
// Exported (was unexported until 007-markdown-renderer needed it) so
// renderers can format a truncation note from the real value instead of
// hardcoding "512 KB" as a literal string -- same reasoning as
// CommitDate being pre-formatted once here rather than independently by
// each renderer (constitution Article V). See
// specs/001-data-contract/progress.md for the change record.
const RawInputCapBytes = 512 * 1024

// TruncateRawInput enforces RawInputCapBytes on s. If s already fits, it
// is returned unchanged with truncated=false. Otherwise s is cut to at
// most RawInputCapBytes bytes and truncated=true.
//
// The cut point is UTF-8-rune-safe: if the exact byte cap falls in the
// middle of a multi-byte rune, TruncateRawInput backs up (at most 3
// bytes -- UTF-8 encodes a rune in up to 4 bytes, so a continuation byte
// is at most 3 bytes past its rune's lead byte) to the last complete
// rune boundary at or before the cap, rather than emit a string ending
// in an invalid trailing byte sequence. The result can therefore be up
// to 3 bytes under RawInputCapBytes in that case -- an accepted trade-off:
// rawInput is parse-fallback only, not the primary payload (chain[] is),
// so exactness to the byte isn't load-bearing, while a corrupted
// trailing character would be a real, if minor, misrepresentation of
// what was actually pasted. Splitting the remainder into a second field
// instead of discarding it was considered and rejected (see
// progress.md's T006 scoping entry): rawInput's cap exists specifically
// to bound total bundle size, so keeping the remainder anywhere would
// defeat that purpose.
func TruncateRawInput(s string) (out string, truncated bool) {
	if len(s) <= RawInputCapBytes {
		return s, false
	}

	cut := RawInputCapBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}

	return s[:cut], true
}

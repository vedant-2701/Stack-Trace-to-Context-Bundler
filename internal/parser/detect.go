package parser

import (
	"fmt"
	"strings"
)

// DetectLanguage calls Detect(rawTrace) on every element of candidates, in
// order, and returns the single LanguageParser that claims it.
//
// candidates must be non-empty -- an empty slice means the calling binary
// registered zero parsers, which is a build-time wiring bug (e.g. cmd/all's
// main.go forgot to list any), never something real trace input can
// trigger. DetectLanguage panics in that case rather than returning
// ErrNoMatch, per CONVENTIONS.md's guidance that panics are for genuine
// programmer-error invariants, not runtime conditions.
//
// Returns:
//   - exactly one candidate matched: that LanguageParser, nil error.
//   - zero candidates matched: nil, an error wrapping ErrNoMatch, naming
//     every checked candidate's Language() value.
//   - two or more candidates matched: nil, an error wrapping ErrAmbiguous,
//     naming every matched candidate's Language() value.
//
// DetectLanguage performs no I/O and takes no context.Context: every
// Detect() call it makes is already required (003a's LanguageParser
// interface) to be fast, in-memory, and side-effect-free, so there is
// nothing here that could block or need cancellation.
func DetectLanguage(rawTrace string, candidates []LanguageParser) (LanguageParser, error) {
	if len(candidates) == 0 {
		panic("parser.DetectLanguage: candidates is empty -- caller wired zero parsers")
	}

	var matched []LanguageParser
	for _, c := range candidates {
		if c.Detect(rawTrace) {
			matched = append(matched, c)
		}
	}

	switch len(matched) {
	case 0:
		names := make([]string, len(candidates))
		for i, c := range candidates {
			names[i] = string(c.Language())
		}
		return nil, fmt.Errorf("checked %s: %w", strings.Join(names, ", "), ErrNoMatch)
	case 1:
		return matched[0], nil
	default:
		names := make([]string, len(matched))
		for i, m := range matched {
			names[i] = string(m.Language())
		}
		return nil, fmt.Errorf("%s: %w", strings.Join(names, ", "), ErrAmbiguous)
	}
}

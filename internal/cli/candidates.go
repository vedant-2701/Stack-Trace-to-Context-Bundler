package cli

import (
	"fmt"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

// hintLanguages maps a --lang hint value (as validated by validateLang,
// or the fixed lang ParseFixedLang is always called with) to the set of
// contract.Language values that hint narrows candidates to (FR3).
// "typescript" maps to both javascript and typescript -- 006a's parser
// can't always tell TypeScript-compiled-to-JS apart from hand-written JS
// (see parser.LanguageParser.Language's doc comment and
// memory/known-gaps.md), so --lang=typescript must not exclude the
// javascript candidate; Detect still decides between the two (FR9).
//
// The only non-empty hint values ever passed to selectCandidates are
// "java" and "typescript" -- validateLang rejects anything else for
// cmd/all's --lang flag, and ParseFixedLang panics on any lang other
// than these two -- so both are covered here.
var hintLanguages = map[string][]contract.Language{
	"typescript": {contract.LanguageJavaScript, contract.LanguageTypeScript},
	"java":       {contract.LanguageJava},
}

// hintDisplayName maps a --lang hint value to the human-readable name
// used in FR10's "<Name> is not supported yet" error message. Only
// "java" is populated today -- it's the only hint whose language set can
// currently match zero registered parsers, until 005a's Java parser
// exists.
var hintDisplayName = map[string]string{
	"java": "Java",
}

// selectCandidates narrows registered to the parsers matching hint, per
// FR3. An empty hint ("" -- the "defer to auto-detection" case, see
// validateLang's doc comment) returns registered unchanged: every
// registered parser is a candidate, and parser.DetectLanguage decides.
//
// When hint is non-empty and its language set (hintLanguages) matches
// zero parsers in registered -- the FR10 case, currently only "java"
// until 005a exists -- selectCandidates returns a
// *languageUnsupportedError instead of an empty slice. This is
// deliberate, not incidental: parser.DetectLanguage panics if ever
// called with an empty candidates slice (see its own doc comment), so
// the caller (ParseAll/ParseFixedLang, T006/T007) must fail fast here,
// before candidates ever reaches DetectLanguage.
//
// A non-empty hint with no entry in hintLanguages is a caller bug -- an
// invalid hint should already have been rejected by validateLang
// (cmd/all's --lang) or is impossible for ParseFixedLang's
// caller-supplied lang (panics on anything but "java"/"typescript") --
// so this panics rather than returning an error, same reasoning as
// ParseFixedLang's own invalid-lang panic (CONVENTIONS.md: panics are
// for programmer-error invariants, not runtime conditions).
func selectCandidates(hint string, registered []parser.LanguageParser) ([]parser.LanguageParser, error) {
	if hint == "" {
		return registered, nil
	}

	langs, ok := hintLanguages[hint]
	if !ok {
		panic(fmt.Sprintf("cli.selectCandidates: no hintLanguages entry for hint %q -- this is a caller bug, an invalid hint should never reach here", hint))
	}

	var candidates []parser.LanguageParser
	for _, p := range registered {
		for _, l := range langs {
			if p.Language() == l {
				candidates = append(candidates, p)
				break
			}
		}
	}

	if len(candidates) == 0 {
		name, ok := hintDisplayName[hint]
		if !ok {
			name = hint
		}
		return nil, &languageUnsupportedError{name: name}
	}

	return candidates, nil
}

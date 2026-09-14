package parser

import "errors"

// ErrUnparseable is wrapped by a LanguageParser implementation's Parse
// method when rawTrace matched that language's general shape (Detect
// would return true) but could not actually be converted into a valid
// exception chain. Distinguishes this expected failure mode (mapped by
// the caller to CLI exit code 3) from an unexpected internal error
// (exit code 1). See registry.go's Parse doc comment.
var ErrUnparseable = errors.New("trace matched this language's shape but could not be parsed into a valid exception chain")

// ErrNoMatch is wrapped by DetectLanguage when no candidate LanguageParser's
// Detect() returned true for the given raw trace. Distinguishes "the trace
// doesn't look like any registered language" from ErrAmbiguous ("it looks
// like more than one") -- callers (002b) map both to CLI exit code 4
// (CONVENTIONS.md) but log a different, specific message for each via
// errors.Is. The wrapping error's message names every checked candidate's
// Language() value (comma-joined, prefixed "checked "), mirroring
// ErrAmbiguous's message shape below -- so a caller logging this error
// doesn't need to re-derive which parsers were even in play.
var ErrNoMatch = errors.New("no registered parser matched this trace")

// ErrAmbiguous is wrapped by DetectLanguage when two or more candidate
// LanguageParsers' Detect() both returned true for the given raw trace.
// The wrapping error's message names every matched candidate's Language()
// value, so a caller doesn't need to re-derive which languages collided.
var ErrAmbiguous = errors.New("trace matched more than one registered language")

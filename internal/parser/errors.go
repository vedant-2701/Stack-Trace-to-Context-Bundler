package parser

import "errors"

// ErrUnparseable is wrapped by a LanguageParser implementation's Parse
// method when rawTrace matched that language's general shape (Detect
// would return true) but could not actually be converted into a valid
// exception chain. Distinguishes this expected failure mode (mapped by
// the caller to CLI exit code 3) from an unexpected internal error
// (exit code 1). See registry.go's Parse doc comment.
var ErrUnparseable = errors.New("trace matched this language's shape but could not be parsed into a valid exception chain")

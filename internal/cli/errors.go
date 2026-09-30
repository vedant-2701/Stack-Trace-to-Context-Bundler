package cli

import "fmt"

// languageUnsupportedError is returned by selectCandidates when a --lang
// hint's language set matches zero registered parsers (FR10) --
// currently only "java", until 005a's Java parser exists. Its Error()
// text is exactly "<Name> is not supported yet", with no feature IDs or
// internal names (FR10), so it's safe to log directly via slog.Error
// with no further wrapping.
//
// A distinct type, not a plain fmt.Errorf string, so a caller that needs
// to check for this specific condition can use errors.As rather than
// matching on message text -- same sentinel-style reasoning as
// parser.ErrUnparseable/ErrNoMatch/ErrAmbiguous, adapted to a struct
// here since the message carries a per-hint display name rather than
// being one fixed package-level value.
type languageUnsupportedError struct {
	name string
}

func (e *languageUnsupportedError) Error() string {
	return fmt.Sprintf("%s is not supported yet", e.name)
}

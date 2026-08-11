// Package cli handles command-line input for the stack-trace-bundler
// binaries: reading a stack trace from a file argument or stdin, and
// validating the --lang and --format flags. It owns raw input
// acquisition and flag validation only; language detection, trace
// parsing, rendering, and clipboard writing happen elsewhere (see
// specs/002a-cli-input-handling/spec.md for scope boundaries).
package cli

// Input holds everything read and parsed from the command line before
// any language detection, trace parsing, or rendering happens (002b's
// job once it exists).
//
// Tagged for JSON so spec requirement 13's Debug-level dump can use
// json.MarshalIndent for readability, rather than %+v. No field is
// omitempty: applying contract's own cross-cutting rule (omitempty for
// "not applicable", never for "empty-but-meaningful") correctly, every
// field here is always meaningful even at its zero value -- there is no
// "not applicable" case among them. This is a deviation from plan.md's
// original untagged code sample; see progress.md for when/why.
type Input struct {
	// RawText is the rune-safe stack trace text read from the file
	// argument or stdin, capped at 512KB via contract.TruncateRawInput.
	// Never legitimately empty on a successfully-built Input -- empty
	// or whitespace-only input is a usage error handled before Input is
	// constructed.
	RawText string `json:"rawText"`

	// RawInputTruncated is true if RawText was cut short by the 512KB
	// cap. Always present -- false is a confirmed value here, not an
	// unset placeholder.
	RawInputTruncated bool `json:"rawInputTruncated"`

	// LangHint is the source language hint: "java", "typescript", or ""
	// (empty). An empty LangHint is not a missing/unset value -- it is
	// the explicit signal to defer language detection to 003's
	// auto-detection, per constitution Article VI (never present a
	// guess as fact). ParseFixedLang always sets this to a fixed,
	// non-empty value; only ParseAll can leave it empty.
	LangHint string `json:"langHint"`

	// Format is the requested output format: "json" or "markdown".
	// Always present -- validateFormat rejects any other value and
	// defaults it to "markdown" when omitted from the command line.
	Format string `json:"format"`

	// StdinIgnored is true if a file-path argument was given while stdin
	// was also piped -- the file wins (spec requirement 5) and stdin is
	// ignored. main.go logs this at Debug level itself, after configuring
	// slog's default handler from the returned verbosity -- ParseAll and
	// ParseFixedLang deliberately do not log it internally, since that
	// call happens before the level is known and Go's default
	// (unconfigured) slog handler sits at Info, silently and permanently
	// dropping any Debug call made before configuration. See plan.md's
	// Architecture section for the full trace of this (T006c).
	StdinIgnored bool `json:"stdinIgnored"`
}

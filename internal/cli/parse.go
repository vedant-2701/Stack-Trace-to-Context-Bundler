package cli

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
)

// validateFormat checks the --format flag value. An empty string (the
// unset/omitted case) defaults to "markdown", matching Input.Format's
// documented default. Any value other than "", "json", or "markdown" is
// a usage error naming the bad value and the accepted set, per spec
// requirement 3 / 11.
func validateFormat(v string) (string, error) {
	switch v {
	case "", "markdown":
		return "markdown", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("invalid --format value %q: accepted values are \"json\" or \"markdown\"", v)
	}
}

// validateLang checks the --lang flag value, used only by ParseAll
// (cmd/all) -- cmd/java and cmd/typescript never register --lang at all,
// so this is never called on those paths. An empty string is a valid,
// meaningful value: it is the explicit "defer to 003 auto-detection"
// signal (spec requirement 1, constitution Article VI), returned
// unchanged rather than defaulted to anything. Any value other than "",
// "java", or "typescript" is a usage error naming the bad value and the
// accepted set, per spec requirement 1 / 11.
func validateLang(v string) (string, error) {
	switch v {
	case "", "java", "typescript":
		return v, nil
	default:
		return "", fmt.Errorf("invalid --lang value %q: accepted values are \"java\" or \"typescript\" (or omit to defer to auto-detection)", v)
	}
}

// ParseAll registers --lang and --format for cmd/all, validates both
// before touching any I/O, then delegates to readTrace. Validation order
// is fixed and deliberate (flag.Parse -> validateLang -> validateFormat
// -> readTrace): flag/value validation is cheap and does no I/O, so it
// fails fast ahead of anything that opens a file or reads stdin. This
// matters when an invocation triggers more than one failure condition at
// once (e.g. --lang=cobol with no available input) -- the flag error is
// what's returned, deterministically, per plan.md.
//
// The FlagSet runs in flag.ContinueOnError mode with output discarded
// (io.Discard): per spec requirement 12, every usage-error path goes
// through a single slog.Error call in main.go for consistent formatting,
// rather than mixing in the flag package's own default stderr output.
// Side effect: -h/--help (flag.ErrHelp) produces no usage listing, only
// a generic "help requested" error -- spec.md doesn't cover -h/--help at
// all, so this is a deliberate minimal-scope choice, not an oversight;
// revisit if -h support becomes a real requirement.
//
// ParseAll never logs anything itself except the stdin-ignored Debug
// line -- error logging and os.Exit are main.go's job, per plan.md's
// Architecture section.
func ParseAll(args []string, stdin io.Reader, stdinIsPiped bool) (Input, error) {
	fs := flag.NewFlagSet("stack-trace-bundler", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var langFlag, formatFlag string
	fs.StringVar(&langFlag, "lang", "", `source language: "java" or "typescript" (omit to defer to auto-detection)`)
	fs.StringVar(&formatFlag, "format", "markdown", `output format: "json" or "markdown"`)

	if err := fs.Parse(args); err != nil {
		return Input{}, fmt.Errorf("parsing flags: %w", err)
	}

	lang, err := validateLang(langFlag)
	if err != nil {
		return Input{}, err
	}

	format, err := validateFormat(formatFlag)
	if err != nil {
		return Input{}, err
	}

	// Extra positional args beyond the first are silently ignored, same
	// as fs.Arg(0) semantics -- not covered by spec.md, minimal-scope
	// choice consistent with the -h/--help note above.
	fileArg := fs.Arg(0)

	raw, truncated, stdinIgnored, err := readTrace(fileArg, stdin, stdinIsPiped)
	if err != nil {
		return Input{}, err
	}

	if stdinIgnored {
		slog.Debug("stdin ignored: file argument took precedence", "file", fileArg)
	}

	return Input{
		RawText:           raw,
		RawInputTruncated: truncated,
		LangHint:          lang,
		Format:            format,
	}, nil
}

// ParseFixedLang registers only --format for cmd/java and cmd/typescript
// -- --lang is never registered on this FlagSet, so passing it produces
// the flag package's own "flag provided but not defined" error (spec
// requirement 2), with no special-case handling needed here.
//
// lang must be "java" or "typescript" and is only ever supplied
// internally, as a hardcoded literal in cmd/java/main.go and
// cmd/typescript/main.go -- never from user input. An invalid lang here
// is therefore a genuine programmer-error invariant, not a runtime
// condition, so this panics rather than returning an error, per
// CONVENTIONS.md's error-handling section. validateLang is deliberately
// not reused here: it treats "" as a valid "defer to auto-detection"
// value for ParseAll's user-facing --lang flag, which is the wrong
// semantics for a caller-fixed language.
//
// Otherwise mirrors ParseAll: fixed validation order (flag.Parse ->
// validateFormat -> readTrace), output discarded via io.Discard for the
// same single-formatting-channel reason (see ParseAll's doc comment for
// the -h/--help caveat), stdin-ignored logged at Debug, no logging or
// os.Exit beyond that -- those stay in main.go.
func ParseFixedLang(args []string, stdin io.Reader, stdinIsPiped bool, lang string) (Input, error) {
	if lang != "java" && lang != "typescript" {
		panic(fmt.Sprintf("cli.ParseFixedLang: invalid lang %q, want \"java\" or \"typescript\" -- this is a caller bug, not a runtime condition", lang))
	}

	fs := flag.NewFlagSet("stack-trace-bundler", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var formatFlag string
	fs.StringVar(&formatFlag, "format", "markdown", `output format: "json" or "markdown"`)

	if err := fs.Parse(args); err != nil {
		return Input{}, fmt.Errorf("parsing flags: %w", err)
	}

	format, err := validateFormat(formatFlag)
	if err != nil {
		return Input{}, err
	}

	fileArg := fs.Arg(0)

	raw, truncated, stdinIgnored, err := readTrace(fileArg, stdin, stdinIsPiped)
	if err != nil {
		return Input{}, err
	}

	if stdinIgnored {
		slog.Debug("stdin ignored: file argument took precedence", "file", fileArg)
	}

	return Input{
		RawText:           raw,
		RawInputTruncated: truncated,
		LangHint:          lang,
		Format:            format,
	}, nil
}

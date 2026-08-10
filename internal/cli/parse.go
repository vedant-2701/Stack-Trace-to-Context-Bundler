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

// ParseAll registers --lang, --format, -v, and -vv for cmd/all,
// validates before touching any I/O, then delegates to readTrace.
// Validation order is fixed and deliberate (flag.Parse -> validateLang
// -> validateFormat -> readTrace): flag/value validation is cheap and
// does no I/O, so it fails fast ahead of anything that opens a file or
// reads stdin. This matters when an invocation triggers more than one
// failure condition at once (e.g. --lang=cobol with no available input)
// -- the flag error is what's returned, deterministically, per plan.md.
//
// -v/-vv are registered on this same FlagSet rather than parsed
// separately in main.go -- a single Parse call avoids an ordering bug a
// two-pass approach would have (e.g. `-v --lang=java` vs
// `--lang=java -v` behaving differently), and inherits the stdlib flag
// package's correct `--` terminator handling and single/double-dash
// equivalence for free. See plan.md's Architecture / approach and
// Alternatives considered for the full rationale, including the
// rejected pre-pass approach.
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
// The returned int is a verbosity count for main.go to pass into
// LogLevel: 0 (neither flag), 1 (-v), or 2 (-vv, or both together --
// max, not sum). Always 0 on any error return -- see plan.md's
// API/contracts section for why a flag.Parse failure specifically can't
// reliably report a real value.
//
// ParseAll never logs anything itself except the stdin-ignored Debug
// line -- error logging and os.Exit are main.go's job, per plan.md's
// Architecture section.
func ParseAll(args []string, stdin io.Reader, stdinIsPiped bool) (Input, int, error) {
	fs := flag.NewFlagSet("stack-trace-bundler", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var langFlag, formatFlag string
	var vFlag, vvFlag bool
	fs.StringVar(&langFlag, "lang", "", `source language: "java" or "typescript" (omit to defer to auto-detection)`)
	fs.StringVar(&formatFlag, "format", "markdown", `output format: "json" or "markdown"`)
	fs.BoolVar(&vFlag, "v", false, "increase log verbosity to Info")
	fs.BoolVar(&vvFlag, "vv", false, "increase log verbosity to Debug")

	if err := fs.Parse(args); err != nil {
		return Input{}, 0, fmt.Errorf("parsing flags: %w", err)
	}

	verbosity := verbosityFromFlags(vFlag, vvFlag)

	lang, err := validateLang(langFlag)
	if err != nil {
		return Input{}, 0, err
	}

	format, err := validateFormat(formatFlag)
	if err != nil {
		return Input{}, 0, err
	}

	// Extra positional args beyond the first are silently ignored, same
	// as fs.Arg(0) semantics -- not covered by spec.md, minimal-scope
	// choice consistent with the -h/--help note above.
	fileArg := fs.Arg(0)

	raw, truncated, stdinIgnored, err := readTrace(fileArg, stdin, stdinIsPiped)
	if err != nil {
		return Input{}, 0, err
	}

	if stdinIgnored {
		slog.Debug("stdin ignored: file argument took precedence", "file", fileArg)
	}

	return Input{
		RawText:           raw,
		RawInputTruncated: truncated,
		LangHint:          lang,
		Format:            format,
	}, verbosity, nil
}

// ParseFixedLang registers only --format, -v, and -vv for cmd/java and
// cmd/typescript -- --lang is never registered on this FlagSet, so
// passing it produces the flag package's own "flag provided but not
// defined" error (spec requirement 2), with no special-case handling
// needed here.
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
// Otherwise mirrors ParseAll: -v/-vv registered on the same FlagSet
// (same single-parse ordering rationale, see ParseAll's doc comment),
// fixed validation order (flag.Parse -> validateFormat -> readTrace),
// output discarded via io.Discard for the same single-formatting-channel
// reason (see ParseAll's doc comment for the -h/--help caveat), the
// returned int is a verbosity count with the same 0/1/2 semantics and
// the same always-0-on-error rule as ParseAll, stdin-ignored logged at
// Debug, no logging or os.Exit beyond that -- those stay in main.go.
func ParseFixedLang(args []string, stdin io.Reader, stdinIsPiped bool, lang string) (Input, int, error) {
	if lang != "java" && lang != "typescript" {
		panic(fmt.Sprintf("cli.ParseFixedLang: invalid lang %q, want \"java\" or \"typescript\" -- this is a caller bug, not a runtime condition", lang))
	}

	fs := flag.NewFlagSet("stack-trace-bundler", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var formatFlag string
	var vFlag, vvFlag bool
	fs.StringVar(&formatFlag, "format", "markdown", `output format: "json" or "markdown"`)
	fs.BoolVar(&vFlag, "v", false, "increase log verbosity to Info")
	fs.BoolVar(&vvFlag, "vv", false, "increase log verbosity to Debug")

	if err := fs.Parse(args); err != nil {
		return Input{}, 0, fmt.Errorf("parsing flags: %w", err)
	}

	verbosity := verbosityFromFlags(vFlag, vvFlag)

	format, err := validateFormat(formatFlag)
	if err != nil {
		return Input{}, 0, err
	}

	fileArg := fs.Arg(0)

	raw, truncated, stdinIgnored, err := readTrace(fileArg, stdin, stdinIsPiped)
	if err != nil {
		return Input{}, 0, err
	}

	if stdinIgnored {
		slog.Debug("stdin ignored: file argument took precedence", "file", fileArg)
	}

	return Input{
		RawText:           raw,
		RawInputTruncated: truncated,
		LangHint:          lang,
		Format:            format,
	}, verbosity, nil
}

package cli

import (
	"fmt"
	"io"

	"github.com/spf13/pflag"
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

// ParseAll registers --lang, --format, and -v for cmd/all, validates
// before touching any I/O, then delegates to readTrace. Validation order
// is fixed and deliberate (Parse -> validateLang -> validateFormat ->
// readTrace): flag/value validation is cheap and does no I/O, so it
// fails fast ahead of anything that opens a file or reads stdin. This
// matters when an invocation triggers more than one failure condition at
// once (e.g. --lang=cobol with no available input) -- the flag error is
// what's returned, deterministically, per plan.md.
//
// Uses pflag.FlagSet, not stdlib flag.FlagSet, as of task T007a (see
// memory/decisions/0002-adopt-pflag-for-flag-parsing.md) -- stdlib
// flag.Parse stops recognizing flags once it hits the first positional
// argument, so `stba trace.txt -vv` silently dropped -vv entirely.
// pflag permutes flags and the positional argument in any order and
// correctly honors a `--` terminator.
//
// -v is registered via CountVarP, pflag's POSIX-clustering counting
// shorthand (-v counts 1, -vv counts 2, -vvv counts 3, ...), not two
// separate bool flags -- pflag shorthands are strictly
// single-dash-single-character, so a literal two-character `-vv` token
// has no direct equivalent in pflag's model the way it did under stdlib
// flag. This still satisfies spec.md's `-v`/`-vv` wording and needs no
// change to LogLevel, whose existing default case already treats any
// count >= 2 as Debug.
//
// The FlagSet runs in pflag.ContinueOnError mode with output discarded
// (io.Discard): per spec requirement 12, every usage-error path goes
// through a single slog.Error call in main.go for consistent formatting,
// rather than mixing in pflag's own default stderr output. Side effect:
// -h/--help produces no usage listing, only whatever error pflag returns
// for it -- spec.md doesn't cover -h/--help at all, so this is a
// deliberate minimal-scope choice, not an oversight; revisit if -h
// support becomes a real requirement (this is an open question carried
// over unchanged from the original stdlib-flag design, not newly
// introduced by the pflag switch).
//
// The returned int is a verbosity count for main.go to pass into
// LogLevel: 0 (no -v), 1 (-v), 2 or more (-vv, -vvv, ...). Always 0 on
// any error return -- see plan.md's API/contracts section for why a
// Parse failure specifically can't reliably report a real value.
//
// ParseAll never calls slog itself, not even for the stdin-ignored case:
// it surfaces that via Input.StdinIgnored instead, for main.go to log
// after configuring slog's level from the returned verbosity (see
// Input.StdinIgnored's doc comment and plan.md's Architecture section
// for why logging it here, before the level is known, doesn't work --
// T006c). Error logging and os.Exit are also main.go's job.
func ParseAll(args []string, stdin io.Reader, stdinIsPiped bool) (Input, int, error) {
	fs := pflag.NewFlagSet("stack-trace-bundler", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var langFlag, formatFlag string
	var verbosityCount int
	fs.StringVar(&langFlag, "lang", "", `source language: "java" or "typescript" (omit to defer to auto-detection)`)
	fs.StringVar(&formatFlag, "format", "markdown", `output format: "json" or "markdown"`)
	fs.CountVarP(&verbosityCount, "verbose", "v", "increase log verbosity (-v for Info, -vv for Debug)")

	if err := fs.Parse(args); err != nil {
		return Input{}, 0, fmt.Errorf("parsing flags: %w", err)
	}

	verbosity := verbosityCount

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

	return Input{
		RawText:           raw,
		RawInputTruncated: truncated,
		LangHint:          lang,
		Format:            format,
		StdinIgnored:      stdinIgnored,
	}, verbosity, nil
}

// ParseFixedLang registers only --format and -v for cmd/java and
// cmd/typescript -- --lang is never registered on this FlagSet, so
// passing it produces pflag's own "unknown flag" error (spec requirement
// 2), with no special-case handling needed here.
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
// Otherwise mirrors ParseAll: pflag.FlagSet with -v as a CountVarP
// shorthand (same rationale, see ParseAll's doc comment), fixed
// validation order (Parse -> validateFormat -> readTrace), output
// discarded via io.Discard for the same single-formatting-channel reason
// (see ParseAll's doc comment for the -h/--help caveat), the returned
// int is a verbosity count with the same semantics and the same
// always-0-on-error rule as ParseAll. Like ParseAll, it never calls slog
// itself -- Input.StdinIgnored carries the stdin-ignored signal for
// main.go to log post-configuration; see ParseAll's doc comment for why
// (T006c).
func ParseFixedLang(args []string, stdin io.Reader, stdinIsPiped bool, lang string) (Input, int, error) {
	if lang != "java" && lang != "typescript" {
		panic(fmt.Sprintf("cli.ParseFixedLang: invalid lang %q, want \"java\" or \"typescript\" -- this is a caller bug, not a runtime condition", lang))
	}

	fs := pflag.NewFlagSet("stack-trace-bundler", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var formatFlag string
	var verbosityCount int
	fs.StringVar(&formatFlag, "format", "markdown", `output format: "json" or "markdown"`)
	fs.CountVarP(&verbosityCount, "verbose", "v", "increase log verbosity (-v for Info, -vv for Debug)")

	if err := fs.Parse(args); err != nil {
		return Input{}, 0, fmt.Errorf("parsing flags: %w", err)
	}

	verbosity := verbosityCount

	format, err := validateFormat(formatFlag)
	if err != nil {
		return Input{}, 0, err
	}

	fileArg := fs.Arg(0)

	raw, truncated, stdinIgnored, err := readTrace(fileArg, stdin, stdinIsPiped)
	if err != nil {
		return Input{}, 0, err
	}

	return Input{
		RawText:           raw,
		RawInputTruncated: truncated,
		LangHint:          lang,
		Format:            format,
		StdinIgnored:      stdinIgnored,
	}, verbosity, nil
}

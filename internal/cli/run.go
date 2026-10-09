package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/pflag"

	"github.com/vedant-2701/stack-trace-bundler/internal/clipboard"
	"github.com/vedant-2701/stack-trace-bundler/internal/codecontext"
	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
	"github.com/vedant-2701/stack-trace-bundler/internal/render"
)

// getwd, buildGitMetadata, buildCodeContexts, and clipboardWrite are
// package-level vars wrapping os.Getwd/codecontext.BuildGitMetadata/
// codecontext.BuildCodeContexts/clipboard.Write -- CONVENTIONS.md's rule
// that anything shelling out (git, the OS clipboard utility) must be
// tested behind a fake, and Getwd can fail in ways worth testing too.
// run_test.go overrides these per test (with t.Cleanup restoring the
// current value) -- production code never reassigns them.
var (
	getwd             = os.Getwd
	buildGitMetadata  = codecontext.BuildGitMetadata
	buildCodeContexts = codecontext.BuildCodeContexts
	clipboardWrite    = clipboard.Write
)

// DependencyResolver resolves a Bundle's dependency list for one
// exception chain (FR4) -- e.g. typescript.ResolveDependencies (006b).
// ctx is context.Background() throughout Run (FR8, Q12); workDir comes
// from os.Getwd() (step 6, T013). Returns nil when there's nothing to
// report, matching contract.Bundle.Dependencies' own optional semantics
// -- never a zero-value struct standing in for "none."
type DependencyResolver func(ctx context.Context, workDir string, chain []contract.ExceptionNode) *contract.Dependencies

// BinaryConfig is what each of cmd/all, cmd/java, and cmd/typescript's
// main.go supplies to Run -- everything that differs between the three
// binaries, so Run itself has no cmd/*-specific branching.
type BinaryConfig struct {
	// ProgramName is filepath.Base(os.Args[0]), computed in main.go --
	// not read from os.Args here, to keep internal/cli free of a direct
	// os.Args reference (plan.md's Help text section). Used in
	// usageText's synopsis line.
	ProgramName string

	// FixedLang is "" for cmd/all (Run calls ParseAll, which registers
	// --lang) or "java"/"typescript" for cmd/java/cmd/typescript (Run
	// calls ParseFixedLang with this as the fixed lang, which never
	// registers --lang).
	FixedLang string

	// Candidates is this binary's full set of registered
	// parser.LanguageParsers -- cmd/java's is nil until 005a exists,
	// which is exactly what makes its selectCandidates fail-fast fire
	// on every invocation.
	Candidates []parser.LanguageParser

	// Resolvers maps a contract.Language to the DependencyResolver Run
	// calls for it (step 9, T013) -- nil (or missing an entry) for a
	// language with no resolver (FR4); cmd/java's Resolvers is nil
	// throughout.
	Resolvers map[contract.Language]DependencyResolver

	// Examples are this binary's --help usage examples (FR19-20, Q14).
	Examples []Example
}

// singleLineTip is FR12's exact second-sentence tip, appended to the
// ErrNoMatch message (never the ErrAmbiguous one) when the trimmed input
// is a single line -- i.e. has no stack frames at all, so there is
// nothing this tool's enrichment can add over pasting the line directly.
const singleLineTip = "The input is a single line with no stack frames, so there is nothing for this tool to add -- paste it directly into your LLM instead. If it came from Node.js, Error.stackTraceLimit may have been set to 0."

// isSingleLineInput reports whether raw, once trimmed of leading/
// trailing whitespace, contains no newline -- FR12's trigger condition
// for singleLineTip.
func isSingleLineInput(raw string) bool {
	return !strings.Contains(strings.TrimSpace(raw), "\n")
}

// assembleBundle builds the contract.Bundle for a successful parse, per
// FR4 -- a pure field-mapping function, deliberately separated from Run
// so FR4's assembly rules (which field comes from where) can be unit
// tested directly (TestAssembleBundle) without needing Run itself to
// expose the Bundle it builds; Run doesn't surface it anywhere until
// T014 renders and delivers it.
func assembleBundle(input Input, language contract.Language, chain []contract.ExceptionNode, rt contract.Runtime, codeContexts []contract.CodeContext, gitMeta *contract.GitMetadata, deps *contract.Dependencies) contract.Bundle {
	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          language,
		OS:                contract.OS(runtime.GOOS),
		RawInput:          input.RawText,
		RawInputTruncated: input.RawInputTruncated,
		Fingerprint:       contract.ComputeFingerprint(chain),
		Runtime:           rt,
		Chain:             chain,
		CodeContexts:      codeContexts,
		GitMetadata:       gitMeta,
		Dependencies:      deps,
	}
}

// resolveLanguage runs step 4 (re-run candidate selection, then
// parser.DetectLanguage) and, on a match, logs "language resolved" plus
// the FR9 hint-mismatch line. Returns the matched parser and exitCode 0
// on success; on failure, matched is nil and exitCode is the code Run
// should return immediately -- the error has already been logged via
// slog.Error here, so Run only needs to propagate the code. Extracted
// from Run solely to keep Run's own cyclomatic complexity under
// golangci-lint's gocyclo threshold; no behavior change from when this
// was inline.
func resolveLanguage(input Input, cfg BinaryConfig) (matched parser.LanguageParser, exitCode int) {
	// Re-run candidate selection rather than threading the narrowed slice
	// ParseAll/ParseFixedLang already computed and discarded (plan.md's
	// Candidate filtering section, Article VIII) -- a few slice
	// comparisons, no I/O. Should be unreachable: ParseAll/ParseFixedLang
	// already ran this exact check (same hint, same candidates) before
	// Run ever got this far; handled defensively rather than assumed
	// impossible.
	filtered, err := selectCandidates(input.LangHint, cfg.Candidates)
	if err != nil {
		slog.Error(err.Error())
		return nil, 2
	}

	matched, err = parser.DetectLanguage(input.RawText, filtered)
	switch {
	case errors.Is(err, parser.ErrNoMatch):
		msg := fmt.Sprintf("could not detect the language of the input: %s", err)
		if isSingleLineInput(input.RawText) {
			msg += ". " + singleLineTip
		}
		slog.Error(msg)
		return nil, 4
	case errors.Is(err, parser.ErrAmbiguous):
		slog.Error(fmt.Sprintf("the input matched more than one language: %s", err))
		return nil, 4
	case err != nil:
		// DetectLanguage's contract (see its doc comment) only ever wraps
		// ErrNoMatch or ErrAmbiguous -- this branch should be unreachable,
		// handled defensively rather than assumed impossible, same as the
		// selectCandidates re-check above.
		slog.Error(err.Error())
		return nil, 1
	}

	detectedLang := string(matched.Language())
	slog.Info("language resolved", "language", detectedLang)

	// FR9: a separate, second Info line in the mismatch case -- not
	// folded into the line above -- only when a hint was actually given
	// (input.LangHint != "") and it differs from what Detect actually
	// matched.
	if input.LangHint != "" && detectedLang != input.LangHint {
		slog.Info("detected language differs from --lang hint", "hint", input.LangHint, "detected", detectedLang)
	}

	return matched, 0
}

// finishDelivery runs steps 10-12: deliver the already-rendered bundle
// to stdout or outputPath (T009's deliver), then -- only after a
// successful delivery, per FR16's ordering -- write it to the clipboard
// unless noClipboard, and log "bundle delivered" (FR22) with its
// destination/clipboard attributes. Returns the exit code Run should
// return immediately; any error has already been logged via slog here.
// Extracted from Run to keep its own cyclomatic complexity under
// golangci-lint's gocyclo threshold, same reasoning as resolveLanguage.
func finishDelivery(rendered, outputPath string, noClipboard bool, stdout io.Writer) int {
	if err := deliver(rendered, outputPath, stdout); err != nil {
		slog.Error(err.Error())
		return 1
	}

	destination := "stdout"
	if outputPath != "" {
		destination = outputPath
	}

	clipboardStatus := "skipped"
	if !noClipboard {
		if err := clipboardWrite(context.Background(), rendered); err != nil {
			clipboardStatus = "failed"
			slog.Warn(clipboardWarnText(err))
		} else {
			clipboardStatus = "copied"
		}
	}

	slog.Info("bundle delivered", "destination", destination, "clipboard", clipboardStatus)

	return 0
}

// clipboardWarnText builds FR16's exact Warn text: a fixed prefix naming
// the underlying error, plus a suffix that depends on which of
// clipboard's two sentinel errors it wraps -- the longer
// install-something suggestion for ErrNoClipboardUtility (nothing was
// even found to try), the shorter --no-clipboard-only suggestion for
// ErrClipboardWriteFailed or anything else (a tool WAS found, so
// "install one" doesn't apply).
func clipboardWarnText(err error) string {
	msg := fmt.Sprintf("could not copy the bundle to the clipboard: %s", err)
	if errors.Is(err, clipboard.ErrNoClipboardUtility) {
		return msg + "; install one of the listed utilities, or pass --no-clipboard to skip copying"
	}
	return msg + "; pass --no-clipboard to skip copying"
}

// Run is the single entry point all three binaries call from main --
// everything past reading os.Args/os.Stdin and calling os.Exit on the
// result lives here, not in any cmd/*/main.go (plan.md's Architecture
// section). stdout and stderr are injected explicitly, not os.Stdout/
// os.Stderr directly, so run_test.go can assert on Run's output without
// swapping the real process-wide variables (plan.md's API/contracts
// section); slog's handler is configured against the injected stderr
// too, not os.Stderr, for the same reason -- each main.go passes the
// real os.Stdout/os.Stderr when calling Run.
//
// Currently implemented in full, through step 12 (delivery): parse;
// pflag.ErrHelp -> print usageText, exit 0; any other parse error ->
// slog.Error, exit 2; configure slog from verbosity; log stdin ignored
// if set; re-run candidate selection and DetectLanguage, mapping
// ErrNoMatch/ErrAmbiguous to exit 4 with the FR11/FR12 messages, and
// logging language resolved plus the FR9 hint-mismatch line; call the
// resolved parser's Parse, mapping ErrUnparseable to exit 3 and any
// other error to exit 1; os.Getwd (exit 1 on failure, before any git/
// dependency call is made) -> BuildGitMetadata -> BuildCodeContexts ->
// the detected language's DependencyResolver if registered -> assemble
// the Bundle; render via render.Markdown/render.JSON per Input.Format;
// deliver to stdout or Input.Output (exit 1 on failure, clipboard never
// attempted); clipboard.Write unless Input.NoClipboard, with a Warn (not
// fatal) on failure; log bundle delivered.
func Run(args []string, stdin io.Reader, stdinIsPiped bool, cfg BinaryConfig, stdout, stderr io.Writer) int {
	var (
		input     Input
		verbosity int
		err       error
	)

	if cfg.FixedLang == "" {
		input, verbosity, err = ParseAll(args, stdin, stdinIsPiped, cfg.Candidates)
	} else {
		input, verbosity, err = ParseFixedLang(args, stdin, stdinIsPiped, cfg.FixedLang, cfg.Candidates)
	}

	if errors.Is(err, pflag.ErrHelp) {
		fs := pflag.NewFlagSet(cfg.ProgramName, pflag.ContinueOnError)
		registerFlags(fs, cfg.FixedLang)
		_, _ = io.WriteString(stderr, usageText(cfg.ProgramName, fs, cfg.Examples))
		return 0
	}

	// Configured against the injected stderr (not os.Stderr) for both
	// the error and success paths below, per the doc comment above --
	// verbosity is always 0 on any error (ParseAll/ParseFixedLang's own
	// contract), and LogLevel(0) is Warn, which Error always clears
	// regardless.
	slog.SetDefault(slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: LogLevel(verbosity),
	})))

	if err != nil {
		slog.Error(err.Error())
		return 2
	}

	// Must come after the slog.SetDefault above, not before -- see
	// Input.StdinIgnored's doc comment (T006c): logging this during
	// ParseAll/ParseFixedLang itself, before the level was known,
	// silently dropped the message under every verbosity setting.
	if input.StdinIgnored {
		slog.Debug("stdin ignored: file argument took precedence")
	}

	matched, exitCode := resolveLanguage(input, cfg)
	if exitCode != 0 {
		return exitCode
	}

	chain, runtimeInfo, err := matched.Parse(context.Background(), input.RawText)
	switch {
	case errors.Is(err, parser.ErrUnparseable):
		slog.Error(fmt.Sprintf("could not parse the input: %s", err))
		return 3
	case err != nil:
		slog.Error(err.Error())
		return 1
	}

	// FR5: the working directory for the git and dependency steps, exit 1
	// before any of those calls if it can't be determined (Q17) -- nothing
	// past this point has a meaningful fallback for "which directory."
	workDir, err := getwd()
	if err != nil {
		slog.Error(err.Error())
		return 1
	}

	ctx := context.Background()
	gitMeta := buildGitMetadata(ctx, workDir)
	codeContexts := buildCodeContexts(ctx, chain, matched.Language(), gitMeta)

	// FR4: nil (and so omitted from the JSON, per contract.Bundle.
	// Dependencies' own doc comment) when no resolver is registered for
	// the detected language -- cfg.Resolvers is nil throughout for
	// cmd/java, and intentionally has no entry at all for languages no
	// binary resolves dependencies for.
	var deps *contract.Dependencies
	if resolver, ok := cfg.Resolvers[matched.Language()]; ok && resolver != nil {
		deps = resolver(ctx, workDir, chain)
	}

	bundle := assembleBundle(input, matched.Language(), chain, runtimeInfo, codeContexts, gitMeta, deps)

	var rendered string
	if input.Format == "json" {
		rendered = render.JSON(bundle)
	} else {
		rendered = render.Markdown(bundle)
	}

	return finishDelivery(rendered, input.Output, input.NoClipboard, stdout)
}

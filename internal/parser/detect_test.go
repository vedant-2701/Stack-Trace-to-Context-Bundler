package parser_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser/typescript"
)

// tsNativeExecutionTrace mirrors
// internal/parser/typescript/testdata/ts-native-execution.txt: Node's
// native TypeScript type-stripping execution, no transformer frame, one
// real ".ts"-suffixed frame path -- the real trace that only
// typescriptParser.Detect() returns true for (javascriptParser's
// !hasTSExtensionFrame half excludes it).
const tsNativeExecutionTrace = `/home/vedant/stack-trace-bundler/errors-test/app.ts:2
  throw new Error("TS execution test error");
  ^

Error: TS execution test error
    at level3 (/home/vedant/stack-trace-bundler/errors-test/app.ts:2:9)
    at level2 (/home/vedant/stack-trace-bundler/errors-test/app.ts:4:21)
    at level1 (/home/vedant/stack-trace-bundler/errors-test/app.ts:5:21)
    at Object.<anonymous> (/home/vedant/stack-trace-bundler/errors-test/app.ts:6:1)
    at Module._compile (node:internal/modules/cjs/loader:1871:14)
    at Object..js (node:internal/modules/cjs/loader:2002:10)
    at Module.load (node:internal/modules/cjs/loader:1594:32)
    at Module._load (node:internal/modules/cjs/loader:1396:12)
    at wrapModuleLoad (node:internal/modules/cjs/loader:255:19)
    at Module.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:154:5)

Node.js v24.18.0
`

// bareStackFetchCauseTrace mirrors
// internal/parser/typescript/testdata/bare-stack-fetch-cause.txt: the
// confirmed real case where both javascriptParser.Detect() and
// typescriptParser.Detect() return false (spec.md's second acceptance
// criterion).
const bareStackFetchCauseTrace = "TypeError: fetch failed\n"

func TestDetectLanguage_RealSingleMatch(t *testing.T) {
	candidates := []parser.LanguageParser{
		typescript.NewJavaScriptParser(),
		typescript.NewTypeScriptParser(),
	}

	got, err := parser.DetectLanguage(tsNativeExecutionTrace, candidates)
	if err != nil {
		t.Fatalf("DetectLanguage returned error, want nil: %v", err)
	}
	if got.Language() != contract.LanguageTypeScript {
		t.Fatalf("DetectLanguage returned Language() = %q, want %q", got.Language(), contract.LanguageTypeScript)
	}
}

func TestDetectLanguage_RealNoMatch(t *testing.T) {
	candidates := []parser.LanguageParser{
		typescript.NewJavaScriptParser(),
		typescript.NewTypeScriptParser(),
	}

	got, err := parser.DetectLanguage(bareStackFetchCauseTrace, candidates)
	if got != nil {
		t.Fatalf("DetectLanguage returned non-nil parser, want nil: %v", got)
	}
	if !errors.Is(err, parser.ErrNoMatch) {
		t.Fatalf("errors.Is(err, parser.ErrNoMatch) = false, want true (err: %v)", err)
	}
	for _, lang := range []string{string(contract.LanguageJavaScript), string(contract.LanguageTypeScript)} {
		if !strings.Contains(err.Error(), lang) {
			t.Errorf("error message %q does not name checked candidate %q", err.Error(), lang)
		}
	}
}

// fakeLanguageParser is a hand-written fake LanguageParser (no mocking
// framework, per CONVENTIONS.md), used only to exercise DetectLanguage's
// ambiguous (2+ match) branch. No real two-language combination produces
// this today -- javascriptParser/typescriptParser are constructed to
// never both match the same real trace (plan.md's Out of scope).
type fakeLanguageParser struct {
	lang    contract.Language
	matches bool
}

func (f fakeLanguageParser) Language() contract.Language { return f.lang }

func (f fakeLanguageParser) Detect(_ string) bool { return f.matches }

func (f fakeLanguageParser) Parse(_ context.Context, _ string) ([]contract.ExceptionNode, contract.Runtime, error) {
	return nil, contract.Runtime{}, nil
}

func TestDetectLanguage_Ambiguous(t *testing.T) {
	candidates := []parser.LanguageParser{
		fakeLanguageParser{lang: "fake-a", matches: true},
		fakeLanguageParser{lang: "fake-b", matches: true},
	}

	got, err := parser.DetectLanguage("irrelevant trace content", candidates)
	if got != nil {
		t.Fatalf("DetectLanguage returned non-nil parser, want nil: %v", got)
	}
	if !errors.Is(err, parser.ErrAmbiguous) {
		t.Fatalf("errors.Is(err, parser.ErrAmbiguous) = false, want true (err: %v)", err)
	}
	for _, lang := range []string{"fake-a", "fake-b"} {
		if !strings.Contains(err.Error(), lang) {
			t.Errorf("error message %q does not name matched candidate %q", err.Error(), lang)
		}
	}
}

func TestDetectLanguage_EmptyCandidatesPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("DetectLanguage(rawTrace, nil) did not panic, want panic")
		}
	}()

	_, _ = parser.DetectLanguage("irrelevant trace content", nil)
}

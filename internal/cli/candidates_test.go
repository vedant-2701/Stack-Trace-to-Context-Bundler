package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
	"github.com/vedant-2701/stack-trace-bundler/internal/parser"
)

// fakeParser is a minimal parser.LanguageParser for testing
// selectCandidates in isolation -- Detect and Parse are never exercised
// by these tests, only Language().
type fakeParser struct {
	lang contract.Language
}

func (f fakeParser) Language() contract.Language { return f.lang }

func (f fakeParser) Detect(_ string) bool { return false }

func (f fakeParser) Parse(_ context.Context, _ string) ([]contract.ExceptionNode, contract.Runtime, error) {
	return nil, contract.Runtime{}, errors.New("fakeParser.Parse should not be called by selectCandidates tests")
}

func TestSelectCandidates(t *testing.T) {
	jsParser := fakeParser{lang: contract.LanguageJavaScript}
	tsParser := fakeParser{lang: contract.LanguageTypeScript}
	registered := []parser.LanguageParser{jsParser, tsParser}

	t.Run("empty hint returns registered unchanged", func(t *testing.T) {
		got, err := selectCandidates("", registered)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != len(registered) {
			t.Fatalf("len(got) = %d, want %d", len(got), len(registered))
		}
		for i := range registered {
			if got[i] != registered[i] {
				t.Errorf("got[%d] = %v, want %v", i, got[i], registered[i])
			}
		}
	})

	t.Run("typescript hint narrows to both js and ts candidates", func(t *testing.T) {
		got, err := selectCandidates("typescript", registered)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}
	})

	t.Run("java hint with no java parser registered: languageUnsupportedError", func(t *testing.T) {
		got, err := selectCandidates("java", registered)
		if got != nil {
			t.Errorf("got = %v, want nil", got)
		}
		if err == nil {
			t.Fatalf("err = nil, want a languageUnsupportedError")
		}
		var unsupported *languageUnsupportedError
		if !errors.As(err, &unsupported) {
			t.Fatalf("err = %v (%T), want *languageUnsupportedError", err, err)
		}
		if err.Error() != "Java is not supported yet" {
			t.Errorf("err.Error() = %q, want %q", err.Error(), "Java is not supported yet")
		}
	})

	t.Run("java hint with a java parser registered: succeeds", func(t *testing.T) {
		javaParser := fakeParser{lang: contract.LanguageJava}
		got, err := selectCandidates("java", []parser.LanguageParser{jsParser, tsParser, javaParser})
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(got) != 1 || got[0] != javaParser {
			t.Fatalf("got = %v, want [javaParser]", got)
		}
	})
}

func TestSelectCandidates_InvalidHintPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("selectCandidates did not panic on an unrecognized hint")
		}
	}()

	_, _ = selectCandidates("cobol", nil)
}

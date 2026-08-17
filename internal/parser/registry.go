// Package parser defines the LanguageParser interface every
// language-specific parser (005a Java, 006a TS/JS) implements, and the
// registry that auto-detects which implementation applies to a given raw
// trace (003b). This file (registry.go) currently holds only the
// interface -- the registration/detection logic is 003b's scope.
package parser

import (
	"context"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// LanguageParser is implemented once per source language (005a for Java,
// 006a for TypeScript/JavaScript). No package under internal/parser/<lang>/
// may import another language's parser package -- cross-language sharing
// happens only through this interface or genuinely language-agnostic code
// (internal/codecontext).
type LanguageParser interface {
	// Language identifies which contract.Language this implementation
	// produces. Provisional for the TS/JS implementation: TypeScript
	// compiles to .js before running, which may make JavaScript and
	// TypeScript indistinguishable from parsed trace content alone in the
	// common case -- see memory/known-gaps.md's entry owned by 006a. If
	// that turns out to be a real per-trace distinction rather than a
	// per-implementation constant, this method's shape will need to
	// change (e.g. moving into Parse's return) as part of 006a.
	Language() contract.Language

	// Detect reports whether rawTrace looks like this implementation's
	// language, as a fast, in-memory, side-effect-free check -- no I/O, no
	// subprocess calls, no ctx. rawTrace is guaranteed non-empty and
	// already bounded to internal/cli's 512KB cap
	// (contract.TruncateRawInput) -- implementations do not need to
	// defensively handle an empty string. The future auto-detection
	// registry (003b) may call Detect once per registered parser on every
	// invocation where the language isn't hinted via --lang, so cost here
	// is not free to ignore.
	Detect(rawTrace string) bool

	// Parse converts rawTrace into a linear exception chain and the
	// detected runtime. rawTrace is guaranteed non-empty and already
	// bounded to internal/cli's 512KB cap (contract.TruncateRawInput) --
	// implementations do not need to defensively handle an empty string.
	//
	// Contract:
	//   - Frames within each ExceptionNode are ordered exactly as they
	//     appear in the raw trace: Frames[0] is the frame where that
	//     node's exception was thrown. contract.ComputeFingerprint depends
	//     on this ordering.
	//   - Every Frame.Bucket in the returned chain is fully assigned --
	//     never partially bucketed. Neither ExceptionNode nor Frame has a
	//     field to represent a degraded bucketing state, so there is no
	//     partial-success case to fall back on.
	//   - Outcome is binary: either a complete, valid chain and runtime
	//     with a nil error, or a nil chain and zero-value runtime with a
	//     non-nil error. Never a partial result.
	//   - When rawTrace matched this language's general shape (Detect
	//     would return true) but could not actually be converted into a
	//     valid chain, the returned error wraps ErrUnparseable (via %w).
	//     Any other non-nil error is an unexpected internal error, not an
	//     expected parse failure -- callers distinguish the two via
	//     errors.Is(err, parser.ErrUnparseable).
	//   - ctx allows caller cancellation. This interface does not require
	//     or promise a caller-set deadline. An implementation that shells
	//     out internally (e.g. inferring Runtime.Version via
	//     contract.VersionSourceLocalEnvironment) is responsible for its
	//     own bounded timeout derived from ctx, matching
	//     internal/codecontext/runner.go's gitTimeout pattern.
	Parse(ctx context.Context, rawTrace string) ([]contract.ExceptionNode, contract.Runtime, error)
}

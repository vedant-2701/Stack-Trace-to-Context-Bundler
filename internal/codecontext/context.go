package codecontext

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// BuildCodeContexts builds a contract.CodeContext for every own-bucket
// frame across chain (spec.md requirements 1-2, 5-7, 9-10), wiring
// together buildSnippet (T004), checkFileStatus (T005), and buildBlame
// (T006). gitMeta is whatever BuildGitMetadata already produced for this
// bundle -- hasRepo collapses to gitMeta != nil, per plan.md's
// Architecture section; there is no separate repoRoot parameter here,
// since per-file git commands resolve via each frame's own file
// directory (see gitmeta.go/status.go/blame.go's shared comment on this).
// This is the production entry point; it wraps the real gitRunner.
func BuildCodeContexts(ctx context.Context, chain []contract.ExceptionNode, language contract.Language, gitMeta *contract.GitMetadata) []contract.CodeContext {
	return buildCodeContexts(ctx, chain, language, gitMeta, execGitRunner{})
}

// buildCodeContexts is the runner-injectable implementation. Exercised
// directly by this package's table-driven tests via fakeGitRunner.
func buildCodeContexts(ctx context.Context, chain []contract.ExceptionNode, language contract.Language, gitMeta *contract.GitMetadata, runner gitRunner) []contract.CodeContext {
	hasRepo := gitMeta != nil

	// Non-nil even when empty: contract.Bundle.CodeContexts has no
	// omitempty, and types.go's own header comment says "never null" --
	// a nil Go slice would marshal to JSON null, not [], so a chain with
	// zero own-bucket frames still needs a real (empty) slice here.
	contexts := make([]contract.CodeContext, 0)

	for chainIdx, node := range chain {
		for frameIdx, frame := range node.Frames {
			if frame.Bucket != contract.BucketOwn {
				continue
			}
			ref := contract.FrameRef{ChainIndex: chainIdx, FrameIndex: frameIdx}
			contexts = append(contexts, buildOneCodeContext(ctx, ref, frame, language, hasRepo, runner))
		}
	}

	return contexts
}

// buildOneCodeContext builds a single own-bucket frame's CodeContext,
// short-circuiting at the first stage that can't proceed: an
// unreadable/missing file skips git entirely (nothing to blame); no
// repo skips status/blame (requirement 3); a non-clean status skips
// blame (requirement 5); only a clean, tracked file with a repo present
// reaches blame (requirement 6).
//
// Every degraded-but-continuing outcome here (not_found, no repo, stale,
// blame failure) gets a slog.Warn at the point it's decided, carrying the
// same reasoning as the note that ends up on cc -- per plan.md's
// Architecture section ("Logging"), so the diagnostic is visible at the
// default log level without -v/-vv. Centralized here rather than spread
// across gitmeta.go/status.go/blame.go individually: cc.Note is what
// those functions' return values eventually become, and this is the one
// place that's true for every path, so logging here covers all of them
// without duplicating a log line per helper.
func buildOneCodeContext(ctx context.Context, ref contract.FrameRef, frame contract.Frame, language contract.Language, hasRepo bool, runner gitRunner) contract.CodeContext {
	cc := contract.CodeContext{
		FrameRef: ref,
		FilePath: frame.FilePath,
		Language: language,
	}

	snippet, err := buildSnippet(frame.FilePath, frame.LineNumber, DefaultContextLines)
	if err != nil {
		cc.Status = contract.StatusNotFound
		cc.Note = notFoundNote(err)
		warnDegraded(ref, frame.FilePath, cc.Status, cc.Note)
		return cc
	}
	cc.Snippet = snippet

	if !hasRepo {
		cc.Status = contract.StatusOK
		cc.Note = "no git repository found"
		warnDegraded(ref, frame.FilePath, cc.Status, cc.Note)
		return cc
	}

	status, note := checkFileStatus(ctx, frame.FilePath, runner)
	if status != gitStatusClean {
		cc.Status = contract.StatusStale
		cc.Note = note
		warnDegraded(ref, frame.FilePath, cc.Status, cc.Note)
		return cc
	}

	blame, err := buildBlame(ctx, frame.FilePath, snippet.StartLine, snippet.EndLine, runner)
	if err != nil {
		// Requirement 7: a blame failure is not fatal, and does not
		// demote status away from "ok" -- the file itself is still
		// tracked and clean, only the blame lookup failed.
		cc.Status = contract.StatusOK
		cc.Note = blameFailureNote(err)
		warnDegraded(ref, frame.FilePath, cc.Status, cc.Note)
		return cc
	}

	cc.Status = contract.StatusOK
	cc.Blame = blame
	return cc
}

// warnDegraded logs one degraded-but-continuing CodeContext outcome at
// Warn level (package-level slog default, stderr-only per constitution
// Article II; CONVENTIONS.md's Logging section: Warn is the default
// level, so this is visible without -v/-vv).
func warnDegraded(ref contract.FrameRef, filePath string, status contract.CodeContextStatus, note string) {
	slog.Warn("own-code context degraded",
		"file", filePath,
		"chainIndex", ref.ChainIndex,
		"frameIndex", ref.FrameIndex,
		"status", status,
		"reason", note,
	)
}

// notFoundNote distinguishes "file doesn't exist", "file exists but
// couldn't be read", and "file exists but is empty" (spec.md requirement
// 2's two named cases, plus errEmptyFile) -- all three map to
// contract.StatusNotFound, but the note must state the actual reason.
func notFoundNote(err error) string {
	if errors.Is(err, fs.ErrNotExist) {
		return "file not found in current checkout -- trace may be from a different commit"
	}
	if errors.Is(err, errEmptyFile) {
		return "file exists but is empty (no lines) -- trace may be from a different commit or the file was truncated"
	}
	return fmt.Sprintf("file exists but could not be read: %s", err)
}

// blameFailureNote distinguishes a blame timeout from any other blame
// failure (spec.md requirements 7, 8), same "different note, same
// outcome" pattern checkFileStatus already uses for its own errors.
func blameFailureNote(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "git blame could not be completed within the timeout"
	}
	return fmt.Sprintf("git blame failed: %s", err)
}

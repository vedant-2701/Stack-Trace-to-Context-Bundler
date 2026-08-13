package codecontext

import (
	"context"
	"errors"
	"testing"
)

// fakeGitRunner is the hand-written test double shared by every
// *_test.go file in this package (CONVENTIONS.md: no mocking
// framework). fn is invoked for every Run call; each test supplies
// whatever behavior it needs for that case -- canned stdout, a specific
// error, or a simulated timeout (return context.DeadlineExceeded, or an
// error wrapping it, so errors.Is matches it the same way it would
// match execGitRunner's real timeout).
type fakeGitRunner struct {
	fn func(ctx context.Context, dir string, args ...string) (string, error)
}

var _ gitRunner = (*fakeGitRunner)(nil)

// Run implements gitRunner.
func (f *fakeGitRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	return f.fn(ctx, dir, args...)
}

func TestFakeGitRunner_ForwardsArgsAndReturnsResult(t *testing.T) {
	var gotDir string
	var gotArgs []string

	fake := &fakeGitRunner{
		fn: func(_ context.Context, dir string, args ...string) (string, error) {
			gotDir = dir
			gotArgs = args
			return "output", nil
		},
	}

	out, err := fake.Run(context.Background(), "/repo", "status", "--porcelain")
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if out != "output" {
		t.Errorf("Run stdout = %q, want %q", out, "output")
	}
	if gotDir != "/repo" {
		t.Errorf("dir passed to fn = %q, want %q", gotDir, "/repo")
	}
	if len(gotArgs) != 2 || gotArgs[0] != "status" || gotArgs[1] != "--porcelain" {
		t.Errorf("args passed to fn = %v, want [status --porcelain]", gotArgs)
	}
}

func TestFakeGitRunner_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeGitRunner{
		fn: func(context.Context, string, ...string) (string, error) {
			return "", wantErr
		},
	}

	_, err := fake.Run(context.Background(), "/repo", "blame")
	if !errors.Is(err, wantErr) {
		t.Errorf("Run error = %v, want %v", err, wantErr)
	}
}

func TestFakeGitRunner_PropagatesDeadlineExceeded(t *testing.T) {
	fake := &fakeGitRunner{
		fn: func(context.Context, string, ...string) (string, error) {
			return "", context.DeadlineExceeded
		},
	}

	_, err := fake.Run(context.Background(), "/repo", "rev-parse", "--is-inside-work-tree")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run error = %v, want context.DeadlineExceeded", err)
	}
}

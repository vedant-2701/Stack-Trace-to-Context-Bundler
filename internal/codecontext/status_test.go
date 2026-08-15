package codecontext

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestFileStatus_Clean(t *testing.T) {
	fake := &fakeGitRunner{fn: func(_ context.Context, dir string, args ...string) (string, error) {
		if dir != "/repo/internal/codecontext" {
			t.Errorf("dir = %q, want %q", dir, "/repo/internal/codecontext")
		}
		wantArgs := []string{"status", "--porcelain", "gitmeta.go"}
		if !equalArgs(args, wantArgs) {
			t.Errorf("args = %v, want %v", args, wantArgs)
		}
		return "", nil
	}}

	status, note := checkFileStatus(context.Background(), "/repo/internal/codecontext/gitmeta.go", fake)
	if status != gitStatusClean {
		t.Errorf("status = %v, want gitStatusClean", status)
	}
	if note != "" {
		t.Errorf("note = %q, want empty", note)
	}
}

func TestFileStatus_Modified(t *testing.T) {
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return " M gitmeta.go\n", nil
	}}

	status, note := checkFileStatus(context.Background(), "/repo/internal/codecontext/gitmeta.go", fake)
	if status != gitStatusModified {
		t.Errorf("status = %v, want gitStatusModified", status)
	}
	if !strings.Contains(note, "uncommitted") {
		t.Errorf("note = %q, want it to mention uncommitted changes", note)
	}
}

func TestFileStatus_Untracked(t *testing.T) {
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return "?? gitmeta.go\n", nil
	}}

	status, note := checkFileStatus(context.Background(), "/repo/internal/codecontext/gitmeta.go", fake)
	if status != gitStatusUntracked {
		t.Errorf("status = %v, want gitStatusUntracked", status)
	}
	if !strings.Contains(note, "untracked") {
		t.Errorf("note = %q, want it to mention untracked, distinguishing it from the modified case", note)
	}
}

func TestFileStatus_Timeout(t *testing.T) {
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return "", context.DeadlineExceeded
	}}

	status, note := checkFileStatus(context.Background(), "/repo/internal/codecontext/gitmeta.go", fake)
	if status != gitStatusUnknown {
		t.Errorf("status = %v, want gitStatusUnknown", status)
	}
	if !strings.Contains(note, "timeout") {
		t.Errorf("note = %q, want it to mention the timeout", note)
	}
}

func TestFileStatus_GenericFailure(t *testing.T) {
	// Not in T005's minimum acceptance list, but covers spec.md
	// requirement 9's "file outside the repo found from the working
	// directory" edge case: git status fails with a real error, not a
	// timeout. Same cautious gitStatusUnknown outcome, distinct note
	// wording from the timeout case.
	wantErr := errors.New("fatal: not a git repository (or any of the parent directories): .git")
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return "", wantErr
	}}

	status, note := checkFileStatus(context.Background(), "/repo/internal/codecontext/gitmeta.go", fake)
	if status != gitStatusUnknown {
		t.Errorf("status = %v, want gitStatusUnknown", status)
	}
	if strings.Contains(note, "timeout") {
		t.Errorf("note = %q, should not claim a timeout for a non-timeout failure", note)
	}
	if note == "" {
		t.Error("note = empty, want the underlying error reflected")
	}
}

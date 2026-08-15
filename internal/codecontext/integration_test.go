//go:build integration

package codecontext

// This file exercises the real os/exec gitRunner (execGitRunner) against
// a throwaway git repo, validating that this package's porcelain parsing
// actually matches real git's output -- not just the hand-written fake's
// idea of it (plan.md's Testing strategy). Requires a real `git` binary
// on PATH. Excluded from the default `go test ./...` (lefthook, and any
// default CI job) by the build tag above, per CONVENTIONS.md's testing
// section -- run explicitly via:
//
//	go test -tags integration ./internal/codecontext/...

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// runGit shells out to the real git binary for test setup (init, add,
// commit) -- separate from execGitRunner, which is what's actually
// under test here.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
}

func TestIntegration_RealGit_CleanTrackedFile(t *testing.T) {
	repoDir := t.TempDir()
	initRepo(t, repoDir)

	filePath := filepath.Join(repoDir, "Handler.java")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(t, repoDir, "add", "Handler.java")
	runGit(t, repoDir, "commit", "-q", "-m", "add Handler.java")
	head := strings.TrimSpace(runGit(t, repoDir, "rev-parse", "HEAD"))

	ctx := context.Background()

	gitMeta := BuildGitMetadata(ctx, repoDir)
	if gitMeta == nil {
		t.Fatal("BuildGitMetadata returned nil, want a populated GitMetadata")
	}
	if gitMeta.CurrentCommit != head {
		t.Errorf("CurrentCommit = %q, want %q", gitMeta.CurrentCommit, head)
	}
	if gitMeta.Branch == "" {
		t.Error("Branch is empty, want the current branch name")
	}
	if gitMeta.UncommittedChanges {
		t.Error("UncommittedChanges = true, want false right after a clean commit")
	}

	chain := []contract.ExceptionNode{
		{Frames: []contract.Frame{
			{FilePath: filePath, LineNumber: 3, Bucket: contract.BucketOwn},
		}},
	}
	contexts := BuildCodeContexts(ctx, chain, contract.LanguageJava, gitMeta)
	if len(contexts) != 1 {
		t.Fatalf("got %d contexts, want 1", len(contexts))
	}
	cc := contexts[0]
	if cc.Status != contract.StatusOK {
		t.Errorf("Status = %v, want StatusOK; note: %q", cc.Status, cc.Note)
	}
	if len(cc.Blame) == 0 {
		t.Fatal("Blame is empty, want at least one entry for a freshly-committed file")
	}
	if cc.Blame[0].CommitHash != head {
		t.Errorf("Blame[0].CommitHash = %q, want %q", cc.Blame[0].CommitHash, head)
	}
	if cc.Blame[0].Author != "Test" {
		t.Errorf("Blame[0].Author = %q, want %q", cc.Blame[0].Author, "Test")
	}
}

func TestIntegration_RealGit_UncommittedChanges(t *testing.T) {
	repoDir := t.TempDir()
	initRepo(t, repoDir)

	filePath := filepath.Join(repoDir, "Handler.java")
	if err := os.WriteFile(filePath, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(t, repoDir, "add", "Handler.java")
	runGit(t, repoDir, "commit", "-q", "-m", "add Handler.java")

	// Modify without committing.
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3 (uncommitted)\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctx := context.Background()
	gitMeta := BuildGitMetadata(ctx, repoDir)
	if gitMeta == nil {
		t.Fatal("BuildGitMetadata returned nil, want a populated GitMetadata")
	}
	if !gitMeta.UncommittedChanges {
		t.Error("UncommittedChanges = false, want true after an uncommitted edit")
	}

	chain := []contract.ExceptionNode{
		{Frames: []contract.Frame{
			{FilePath: filePath, LineNumber: 2, Bucket: contract.BucketOwn},
		}},
	}
	contexts := BuildCodeContexts(ctx, chain, contract.LanguageJava, gitMeta)
	cc := contexts[0]
	if cc.Status != contract.StatusStale {
		t.Errorf("Status = %v, want StatusStale; note: %q", cc.Status, cc.Note)
	}
	if len(cc.Blame) != 0 {
		t.Errorf("Blame = %+v, want none for a stale file", cc.Blame)
	}
}

func TestIntegration_RealGit_NoRepo(t *testing.T) {
	dir := t.TempDir() // deliberately not a git repo

	gitMeta := BuildGitMetadata(context.Background(), dir)
	if gitMeta != nil {
		t.Errorf("BuildGitMetadata = %+v, want nil outside any git repo", gitMeta)
	}
}

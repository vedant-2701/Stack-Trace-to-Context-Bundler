package codecontext

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// assertGitMetadata compares two *contract.GitMetadata, treating nil
// specially since dereferencing a nil *contract.GitMetadata for a plain
// == comparison would panic.
func assertGitMetadata(t *testing.T, got, want *contract.GitMetadata) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Fatalf("buildGitMetadata() = %+v, want %+v", got, want)
	}
	if got == nil {
		return
	}
	if *got != *want {
		t.Errorf("buildGitMetadata() = %+v, want %+v", *got, *want)
	}
}

func TestBuildGitMetadata_RepoFound_CleanTree(t *testing.T) {
	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --is-inside-work-tree":
			return "true\n", nil
		case "rev-parse HEAD":
			return "abc123def456\n", nil
		case "rev-parse --abbrev-ref HEAD":
			return "main\n", nil
		case "status --porcelain":
			return "", nil
		}
		t.Fatalf("unexpected git call: git %s", strings.Join(args, " "))
		return "", nil
	}}

	got := buildGitMetadata(context.Background(), "/repo", fake)
	assertGitMetadata(t, got, &contract.GitMetadata{
		CurrentCommit:      "abc123def456",
		Branch:             "main",
		UncommittedChanges: false,
	})
}

func TestBuildGitMetadata_RepoFound_DirtyTree(t *testing.T) {
	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --is-inside-work-tree":
			return "true\n", nil
		case "rev-parse HEAD":
			return "abc123def456\n", nil
		case "rev-parse --abbrev-ref HEAD":
			return "main\n", nil
		case "status --porcelain":
			return " M internal/codecontext/gitmeta.go\n", nil
		}
		t.Fatalf("unexpected git call: git %s", strings.Join(args, " "))
		return "", nil
	}}

	got := buildGitMetadata(context.Background(), "/repo", fake)
	assertGitMetadata(t, got, &contract.GitMetadata{
		CurrentCommit:      "abc123def456",
		Branch:             "main",
		UncommittedChanges: true,
	})
}

func TestBuildGitMetadata_DetachedHead(t *testing.T) {
	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --is-inside-work-tree":
			return "true\n", nil
		case "rev-parse HEAD":
			return "deadbeef1234\n", nil
		case "rev-parse --abbrev-ref HEAD":
			// git literally prints "HEAD" in a detached-HEAD state --
			// spec.md requirement 4: no special sentinel, use as-is.
			return "HEAD\n", nil
		case "status --porcelain":
			return "", nil
		}
		t.Fatalf("unexpected git call: git %s", strings.Join(args, " "))
		return "", nil
	}}

	got := buildGitMetadata(context.Background(), "/repo", fake)
	assertGitMetadata(t, got, &contract.GitMetadata{
		CurrentCommit:      "deadbeef1234",
		Branch:             "HEAD",
		UncommittedChanges: false,
	})
}

func TestBuildGitMetadata_NoRepoFound(t *testing.T) {
	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		if strings.Join(args, " ") == "rev-parse --is-inside-work-tree" {
			return "", errors.New("fatal: not a git repository (or any of the parent directories): .git")
		}
		t.Fatalf("unexpected git call beyond detection: git %s", strings.Join(args, " "))
		return "", nil
	}}

	got := buildGitMetadata(context.Background(), "/not-a-repo", fake)
	assertGitMetadata(t, got, nil)
}

func TestBuildGitMetadata_DetectionTimeout(t *testing.T) {
	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		if strings.Join(args, " ") == "rev-parse --is-inside-work-tree" {
			return "", context.DeadlineExceeded
		}
		t.Fatalf("unexpected git call beyond detection: git %s", strings.Join(args, " "))
		return "", nil
	}}

	got := buildGitMetadata(context.Background(), "/repo", fake)
	assertGitMetadata(t, got, nil)
}

func TestBuildGitMetadata_RepoFoundButHEADUnresolvable(t *testing.T) {
	// A repo with zero commits: detection succeeds, but `rev-parse HEAD`
	// fails. Documents the T003 decision recorded in gitmeta.go: this
	// collapses to nil rather than a partially-populated GitMetadata.
	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --is-inside-work-tree":
			return "true\n", nil
		case "rev-parse HEAD":
			return "", errors.New("fatal: ambiguous argument 'HEAD': unknown revision or path not in the working tree")
		}
		t.Fatalf("unexpected git call: git %s", strings.Join(args, " "))
		return "", nil
	}}

	got := buildGitMetadata(context.Background(), "/repo", fake)
	assertGitMetadata(t, got, nil)
}

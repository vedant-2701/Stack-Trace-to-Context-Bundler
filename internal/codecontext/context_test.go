package codecontext

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

func writeSourceFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// singleOwnFrameChain builds a one-node, one-frame chain with that frame
// marked own-bucket, for tests that only need one CodeContext.
func singleOwnFrameChain(path string, lineNumber int) []contract.ExceptionNode {
	return []contract.ExceptionNode{
		{Frames: []contract.Frame{
			{FilePath: path, LineNumber: lineNumber, Bucket: contract.BucketOwn},
		}},
	}
}

func TestBuildCodeContexts_CleanTracked(t *testing.T) {
	path := writeSourceFile(t, t.TempDir(), "Handler.java", strings.Repeat("line\n", 3))

	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		switch args[0] {
		case "status":
			return "", nil // clean
		case "blame":
			return porcelain(
				shaA+" 1 1 3",
				"author Jane Doe",
				"author-mail <jane@example.com>",
				"author-time 0",
				"author-tz +0000",
				"committer Jane Doe",
				"committer-mail <jane@example.com>",
				"committer-time 0",
				"committer-tz +0000",
				"summary initial commit",
				"filename Handler.java",
				"\tline",
				shaA+" 2 2",
				"\tline",
				shaA+" 3 3",
				"\tline",
			), nil
		}
		t.Fatalf("unexpected git call: %v", args)
		return "", nil
	}}

	got := buildCodeContexts(context.Background(), singleOwnFrameChain(path, 2), contract.LanguageJava,
		&contract.GitMetadata{CurrentCommit: "x", Branch: "main"}, fake)

	if len(got) != 1 {
		t.Fatalf("got %d contexts, want 1", len(got))
	}
	cc := got[0]
	if cc.Status != contract.StatusOK {
		t.Errorf("Status = %v, want StatusOK", cc.Status)
	}
	if len(cc.Blame) != 1 || cc.Blame[0].CommitHash != shaA {
		t.Errorf("Blame = %+v, want one entry for %s", cc.Blame, shaA)
	}
	if cc.FrameRef != (contract.FrameRef{ChainIndex: 0, FrameIndex: 0}) {
		t.Errorf("FrameRef = %+v, want {0 0}", cc.FrameRef)
	}
	if cc.Language != contract.LanguageJava {
		t.Errorf("Language = %v, want LanguageJava", cc.Language)
	}
}

func TestBuildCodeContexts_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.java")

	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		t.Fatal("git should never be called for a file that couldn't even be read")
		return "", nil
	}}

	got := buildCodeContexts(context.Background(), singleOwnFrameChain(path, 10), contract.LanguageJava,
		&contract.GitMetadata{}, fake)

	if len(got) != 1 {
		t.Fatalf("got %d contexts, want 1", len(got))
	}
	cc := got[0]
	if cc.Status != contract.StatusNotFound {
		t.Errorf("Status = %v, want StatusNotFound", cc.Status)
	}
	if !strings.Contains(cc.Note, "not found") {
		t.Errorf("Note = %q, want it to mention the file wasn't found", cc.Note)
	}
	if len(cc.Blame) != 0 {
		t.Errorf("Blame = %+v, want none", cc.Blame)
	}
}

func TestBuildCodeContexts_FileUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: file permissions don't block reads, can't exercise this case")
	}

	path := writeSourceFile(t, t.TempDir(), "Handler.java", "line1\nline2\n")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		t.Fatal("git should never be called for a file that couldn't even be read")
		return "", nil
	}}

	got := buildCodeContexts(context.Background(), singleOwnFrameChain(path, 1), contract.LanguageJava,
		&contract.GitMetadata{}, fake)

	cc := got[0]
	if cc.Status != contract.StatusNotFound {
		t.Errorf("Status = %v, want StatusNotFound", cc.Status)
	}
	if strings.Contains(cc.Note, "not found in current checkout") {
		t.Errorf("Note = %q, should state the real reason (unreadable), not claim the file is absent", cc.Note)
	}
}

func TestBuildCodeContexts_StaleModified(t *testing.T) {
	path := writeSourceFile(t, t.TempDir(), "Handler.java", "line1\nline2\nline3\n")

	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return " M Handler.java\n", nil
	}}

	got := buildCodeContexts(context.Background(), singleOwnFrameChain(path, 2), contract.LanguageJava,
		&contract.GitMetadata{}, fake)

	cc := got[0]
	if cc.Status != contract.StatusStale {
		t.Errorf("Status = %v, want StatusStale", cc.Status)
	}
	if !strings.Contains(cc.Note, "uncommitted") {
		t.Errorf("Note = %q, want it to mention uncommitted changes", cc.Note)
	}
	if len(cc.Blame) != 0 {
		t.Errorf("Blame = %+v, want none for a stale file", cc.Blame)
	}
}

func TestBuildCodeContexts_StaleUntracked(t *testing.T) {
	path := writeSourceFile(t, t.TempDir(), "Handler.java", "line1\nline2\nline3\n")

	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return "?? Handler.java\n", nil
	}}

	got := buildCodeContexts(context.Background(), singleOwnFrameChain(path, 2), contract.LanguageJava,
		&contract.GitMetadata{}, fake)

	cc := got[0]
	if cc.Status != contract.StatusStale {
		t.Errorf("Status = %v, want StatusStale", cc.Status)
	}
	if !strings.Contains(cc.Note, "untracked") {
		t.Errorf("Note = %q, want it to mention untracked, distinguishing it from the modified case", cc.Note)
	}
}

func TestBuildCodeContexts_NoRepo(t *testing.T) {
	path := writeSourceFile(t, t.TempDir(), "Handler.java", "line1\nline2\nline3\n")

	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		t.Fatal("git should never be called when hasRepo is false")
		return "", nil
	}}

	got := buildCodeContexts(context.Background(), singleOwnFrameChain(path, 2), contract.LanguageJava,
		nil /* no repo */, fake)

	cc := got[0]
	if cc.Status != contract.StatusOK {
		t.Errorf("Status = %v, want StatusOK", cc.Status)
	}
	if !strings.Contains(cc.Note, "no git repository") {
		t.Errorf("Note = %q, want it to explain no repo was found", cc.Note)
	}
	if len(cc.Blame) != 0 {
		t.Errorf("Blame = %+v, want none with no repo", cc.Blame)
	}
}

func TestBuildCodeContexts_BlameFails(t *testing.T) {
	path := writeSourceFile(t, t.TempDir(), "Handler.java", "line1\nline2\nline3\n")

	fake := &fakeGitRunner{fn: func(_ context.Context, _ string, args ...string) (string, error) {
		switch args[0] {
		case "status":
			return "", nil // clean
		case "blame":
			return "", errors.New("git: command not found")
		}
		t.Fatalf("unexpected git call: %v", args)
		return "", nil
	}}

	got := buildCodeContexts(context.Background(), singleOwnFrameChain(path, 2), contract.LanguageJava,
		&contract.GitMetadata{}, fake)

	cc := got[0]
	// Requirement 7: status stays "ok" even though blame failed.
	if cc.Status != contract.StatusOK {
		t.Errorf("Status = %v, want StatusOK (blame failure isn't fatal)", cc.Status)
	}
	if len(cc.Blame) != 0 {
		t.Errorf("Blame = %+v, want none", cc.Blame)
	}
	if !strings.Contains(cc.Note, "blame") {
		t.Errorf("Note = %q, want it to explain the blame failure", cc.Note)
	}
}

func TestBuildCodeContexts_NoOwnFrames(t *testing.T) {
	chain := []contract.ExceptionNode{
		{Frames: []contract.Frame{
			{FilePath: "/lib/dep.jar", LineNumber: 1, Bucket: contract.BucketDependency},
			{FilePath: "/runtime/Thread.java", LineNumber: 1, Bucket: contract.BucketRuntime},
		}},
	}
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		t.Fatal("git should never be called when there are no own-bucket frames")
		return "", nil
	}}

	got := buildCodeContexts(context.Background(), chain, contract.LanguageJava, &contract.GitMetadata{}, fake)

	if got == nil {
		t.Error("buildCodeContexts returned nil, want a non-nil empty slice (contract.Bundle.CodeContexts has no omitempty)")
	}
	if len(got) != 0 {
		t.Errorf("got %d contexts, want 0", len(got))
	}
}

func TestBuildCodeContexts_FrameRefIndexing(t *testing.T) {
	dir := t.TempDir()
	pathA := writeSourceFile(t, dir, "A.java", "a\n")
	pathB := writeSourceFile(t, dir, "B.java", "b\n")

	chain := []contract.ExceptionNode{
		{Frames: []contract.Frame{
			{FilePath: "/lib/dep.jar", LineNumber: 1, Bucket: contract.BucketDependency}, // skipped
			{FilePath: pathA, LineNumber: 1, Bucket: contract.BucketOwn},                 // {0, 1}
		}},
		{Frames: []contract.Frame{
			{FilePath: "/runtime/Thread.java", LineNumber: 1, Bucket: contract.BucketRuntime}, // skipped
			{FilePath: pathB, LineNumber: 1, Bucket: contract.BucketOwn},                      // {1, 1}
		}},
	}
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		t.Fatal("git should never be called when hasRepo is false")
		return "", nil
	}}

	got := buildCodeContexts(context.Background(), chain, contract.LanguageJava, nil, fake)

	wantRefs := []contract.FrameRef{
		{ChainIndex: 0, FrameIndex: 1},
		{ChainIndex: 1, FrameIndex: 1},
	}
	if len(got) != len(wantRefs) {
		t.Fatalf("got %d contexts, want %d", len(got), len(wantRefs))
	}
	for i, want := range wantRefs {
		if got[i].FrameRef != want {
			t.Errorf("contexts[%d].FrameRef = %+v, want %+v", i, got[i].FrameRef, want)
		}
	}
}

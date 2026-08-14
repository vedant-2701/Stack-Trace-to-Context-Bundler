package codecontext

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

const (
	shaA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	shaB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// porcelain joins fixture lines with "\n" and appends a trailing
// newline, matching how git actually terminates its output.
func porcelain(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func assertBlameEntries(t *testing.T, got, want []contract.BlameEntry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestBlame_SingleCommitWindow(t *testing.T) {
	out := porcelain(
		shaA+" 10 10 5",
		"author Jane Doe",
		"author-mail <jane@example.com>",
		"author-time 0",
		"author-tz +0000",
		"committer Jane Doe",
		"committer-mail <jane@example.com>",
		"committer-time 0",
		"committer-tz +0000",
		"summary Fix bug in handler",
		"filename app.go",
		"\tline 10 content",
		shaA+" 11 11",
		"\tline 11 content",
		shaA+" 12 12",
		"\tline 12 content",
		shaA+" 13 13",
		"\tline 13 content",
		shaA+" 14 14",
		"\tline 14 content",
	)
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return out, nil
	}}

	got, err := buildBlame(context.Background(), "/repo/app.go", 10, 14, fake)
	if err != nil {
		t.Fatalf("buildBlame: unexpected error: %v", err)
	}
	assertBlameEntries(t, got, []contract.BlameEntry{
		{
			StartLine:  10,
			EndLine:    14,
			CommitHash: shaA,
			Author:     "Jane Doe",
			CommitDate: "1970-01-01T00:00:00Z",
			Summary:    "Fix bug in handler",
		},
	})
}

func TestBlame_MultiCommitWindow(t *testing.T) {
	out := porcelain(
		shaA+" 20 20 2",
		"author Alice",
		"author-mail <alice@example.com>",
		"author-time 100",
		"author-tz +0000",
		"committer Alice",
		"committer-mail <alice@example.com>",
		"committer-time 100",
		"committer-tz +0000",
		"summary A change",
		"filename app.go",
		"\tline 20 content",
		shaA+" 21 21",
		"\tline 21 content",
		shaB+" 22 22 3",
		"author Bob",
		"author-mail <bob@example.com>",
		"author-time 200",
		"author-tz +0000",
		"committer Bob",
		"committer-mail <bob@example.com>",
		"committer-time 200",
		"committer-tz +0000",
		"summary B change",
		"filename app.go",
		"\tline 22 content",
		shaB+" 23 23",
		"\tline 23 content",
		shaB+" 24 24",
		"\tline 24 content",
	)
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return out, nil
	}}

	got, err := buildBlame(context.Background(), "/repo/app.go", 20, 24, fake)
	if err != nil {
		t.Fatalf("buildBlame: unexpected error: %v", err)
	}
	// Split at the actual boundary (20-21 vs 22-24), not one entry per
	// line -- exercises git's own grouping via the header's group-size
	// field, not a hand-rolled "compare to previous sha" approach.
	assertBlameEntries(t, got, []contract.BlameEntry{
		{
			StartLine:  20,
			EndLine:    21,
			CommitHash: shaA,
			Author:     "Alice",
			CommitDate: "1970-01-01T00:01:40Z",
			Summary:    "A change",
		},
		{
			StartLine:  22,
			EndLine:    24,
			CommitHash: shaB,
			Author:     "Bob",
			CommitDate: "1970-01-01T00:03:20Z",
			Summary:    "B change",
		},
	})
}

func TestBlame_CommandFailure(t *testing.T) {
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return "", errors.New("git: command not found")
	}}

	_, err := buildBlame(context.Background(), "/repo/app.go", 10, 14, fake)
	if err == nil {
		t.Fatal("buildBlame: expected an error, got nil")
	}
}

func TestBlame_Timeout(t *testing.T) {
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return "", context.DeadlineExceeded
	}}

	_, err := buildBlame(context.Background(), "/repo/app.go", 10, 14, fake)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("buildBlame error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
}

func TestBlame_NonContiguousSameCommitReappearance(t *testing.T) {
	// shaA covers lines 1-3, shaB covers lines 4-5, then shaA reappears
	// at line 6 -- non-contiguous with its first group. Porcelain omits
	// shaA's author/author-time/author-tz/summary block on this second
	// occurrence (already shown once); the parser must reuse the cached
	// metadata from the first occurrence rather than treating this
	// group as metadata-less.
	out := porcelain(
		shaA+" 1 1 3",
		"author Carol",
		"author-mail <carol@example.com>",
		"author-time 300",
		"author-tz +0000",
		"committer Carol",
		"committer-mail <carol@example.com>",
		"committer-time 300",
		"committer-tz +0000",
		"summary first change",
		"filename app.go",
		"\tline 1 content",
		shaA+" 2 2",
		"\tline 2 content",
		shaA+" 3 3",
		"\tline 3 content",
		shaB+" 4 4 2",
		"author Dave",
		"author-mail <dave@example.com>",
		"author-time 400",
		"author-tz +0000",
		"committer Dave",
		"committer-mail <dave@example.com>",
		"committer-time 400",
		"committer-tz +0000",
		"summary second change",
		"filename app.go",
		"\tline 4 content",
		shaB+" 5 5",
		"\tline 5 content",
		shaA+" 6 6 1",
		"filename app.go", // no author/author-time/etc. -- already shown for shaA above
		"\tline 6 content",
	)
	fake := &fakeGitRunner{fn: func(context.Context, string, ...string) (string, error) {
		return out, nil
	}}

	got, err := buildBlame(context.Background(), "/repo/app.go", 1, 6, fake)
	if err != nil {
		t.Fatalf("buildBlame: unexpected error: %v", err)
	}
	assertBlameEntries(t, got, []contract.BlameEntry{
		{
			StartLine:  1,
			EndLine:    3,
			CommitHash: shaA,
			Author:     "Carol",
			CommitDate: "1970-01-01T00:05:00Z",
			Summary:    "first change",
		},
		{
			StartLine:  4,
			EndLine:    5,
			CommitHash: shaB,
			Author:     "Dave",
			CommitDate: "1970-01-01T00:06:40Z",
			Summary:    "second change",
		},
		{
			StartLine:  6,
			EndLine:    6,
			CommitHash: shaA,
			Author:     "Carol",
			CommitDate: "1970-01-01T00:05:00Z",
			Summary:    "first change",
		},
	})
}

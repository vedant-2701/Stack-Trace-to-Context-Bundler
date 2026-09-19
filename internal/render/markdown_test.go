package render

import (
	"strings"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

func TestRenderPreamble(t *testing.T) {
	want := "> This is a stack-trace-bundler bundle: an exception chain with own-code snippets, git blame, and resolved dependency versions, packaged for pasting into an AI chat.\n"
	if got := renderPreamble(); got != want {
		t.Errorf("renderPreamble() = %q, want %q", got, want)
	}
}

func TestRenderRuntime(t *testing.T) {
	tests := []struct {
		name string
		r    contract.Runtime
		want string
	}{
		{
			name: "trace",
			r:    contract.Runtime{Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace},
			want: "Runtime: node 20.11.0",
		},
		{
			name: "local-environment with note",
			r: contract.Runtime{
				Name: "node", Version: "20.11.0",
				VersionSource: contract.VersionSourceLocalEnvironment,
				Note:          "inferred from local `node -v`",
			},
			want: "Runtime: node 20.11.0 (inferred from local \\`node \\-v\\`)",
		},
		{
			name: "local-environment without note",
			r: contract.Runtime{
				Name: "node", Version: "20.11.0",
				VersionSource: contract.VersionSourceLocalEnvironment,
			},
			want: "Runtime: node 20.11.0",
		},
		{
			name: "unknown with note",
			r: contract.Runtime{
				Name:          "jvm",
				VersionSource: contract.VersionSourceUnknown,
				Note:          "no version info in printStackTrace output",
			},
			want: "Runtime: jvm (no version info in printStackTrace output)",
		},
		{
			name: "unknown without note",
			r: contract.Runtime{
				Name:          "jvm",
				VersionSource: contract.VersionSourceUnknown,
			},
			want: "Runtime: jvm (version unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderRuntime(tt.r)
			if got != tt.want {
				t.Errorf("renderRuntime(%+v) = %q, want %q", tt.r, got, tt.want)
			}
		})
	}
}

func TestRenderGit(t *testing.T) {
	tests := []struct {
		name string
		g    *contract.GitMetadata
		want string
	}{
		{
			name: "clean",
			g: &contract.GitMetadata{
				CurrentCommit: "abcdef1234567890", Branch: "main", UncommittedChanges: false,
			},
			want: "Git: main @ abcdef1 (clean)",
		},
		{
			name: "uncommitted changes",
			g: &contract.GitMetadata{
				CurrentCommit: "abcdef1234567890", Branch: "main", UncommittedChanges: true,
			},
			want: "Git: main @ abcdef1 (uncommitted changes)",
		},
		{
			name: "branch with markdown special characters",
			g: &contract.GitMetadata{
				CurrentCommit: "abcdef1234567890", Branch: "feature/fix_bug#123", UncommittedChanges: false,
			},
			want: `Git: feature/fix\_bug\#123 @ abcdef1 (clean)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderGit(tt.g)
			if got != tt.want {
				t.Errorf("renderGit(%+v) = %q, want %q", tt.g, got, tt.want)
			}
		})
	}
}

func TestRenderMetadata(t *testing.T) {
	t.Run("full metadata present", func(t *testing.T) {
		b := contract.Bundle{
			Language: contract.LanguageTypeScript,
			OS:       contract.OSLinux,
			Runtime: contract.Runtime{
				Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
			},
			GitMetadata: &contract.GitMetadata{
				CurrentCommit: "abcdef1234567890", Branch: "main", UncommittedChanges: false,
			},
			Fingerprint: "deadbeefcafe",
		}
		want := "- Language: typescript\n" +
			"- OS: linux\n" +
			"- Runtime: node 20.11.0\n" +
			"- Git: main @ abcdef1 (clean)\n" +
			"- Fingerprint: deadbeefcafe\n"
		if got := renderMetadata(b); got != want {
			t.Errorf("renderMetadata(%+v) = %q, want %q", b, got, want)
		}
	})

	t.Run("nil GitMetadata omits Git line", func(t *testing.T) {
		b := contract.Bundle{
			Language: contract.LanguageJava,
			OS:       contract.OSDarwin,
			Runtime: contract.Runtime{
				Name: "jvm", VersionSource: contract.VersionSourceUnknown,
			},
			GitMetadata: nil,
			Fingerprint: "cafebabe1234",
		}
		want := "- Language: java\n" +
			"- OS: darwin\n" +
			"- Runtime: jvm (version unknown)\n" +
			"- Fingerprint: cafebabe1234\n"
		if got := renderMetadata(b); got != want {
			t.Errorf("renderMetadata(%+v) = %q, want %q", b, got, want)
		}
	})
}

// buildSnippetCode joins lines with "\n" and appends exactly one further
// trailing "\n", matching internal/codecontext.buildSnippet's own
// construction exactly -- never a hand-typed string missing that
// trailing newline, since that's the real off-by-one risk renderSnippet
// has to guard against.
func buildSnippetCode(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestRenderSnippet(t *testing.T) {
	t.Run("exact line count and target marker", func(t *testing.T) {
		s := contract.Snippet{
			StartLine:  10,
			EndLine:    12,
			TargetLine: 11,
			Code:       buildSnippetCode("func foo() {", "    bar()", "}"),
		}
		want := "```typescript\n" +
			"  10 | func foo() {\n" +
			"→ 11 |     bar()\n" +
			"  12 | }\n" +
			"```\n"
		if got := renderSnippet(s, contract.LanguageTypeScript); got != want {
			t.Errorf("renderSnippet(%+v) = %q, want %q", s, got, want)
		}
	})

	t.Run("mixed digit-width line numbers stay aligned", func(t *testing.T) {
		s := contract.Snippet{
			StartLine:  8,
			EndLine:    11,
			TargetLine: 10,
			Code:       buildSnippetCode("a", "b", "c", "d"),
		}
		want := "```typescript\n" +
			"   8 | a\n" +
			"   9 | b\n" +
			"→ 10 | c\n" +
			"  11 | d\n" +
			"```\n"
		if got := renderSnippet(s, contract.LanguageTypeScript); got != want {
			t.Errorf("renderSnippet(%+v) = %q, want %q", s, got, want)
		}
	})

	t.Run("java fence tag", func(t *testing.T) {
		s := contract.Snippet{
			StartLine:  1,
			EndLine:    1,
			TargetLine: 1,
			Code:       buildSnippetCode("System.out.println(\"hi\");"),
		}
		got := renderSnippet(s, contract.LanguageJava)
		if !strings.HasPrefix(got, "```java\n") {
			t.Errorf("renderSnippet(%+v) with LanguageJava = %q, want prefix %q", s, got, "```java\n")
		}
	})
}

func TestRenderBlameTable(t *testing.T) {
	t.Run("single entry", func(t *testing.T) {
		entries := []contract.BlameEntry{
			{
				StartLine: 10, EndLine: 15,
				CommitHash: "abcdef1234567890", Author: "Alice",
				CommitDate: "2024-01-02T03:04:05Z", Summary: "Fix bug",
			},
		}
		want := "| Lines | Commit | Author | Date | Summary |\n" +
			"| --- | --- | --- | --- | --- |\n" +
			"| 10-15 | abcdef1 | Alice | 2024-01-02 | Fix bug |\n"
		if got := renderBlameTable(entries); got != want {
			t.Errorf("renderBlameTable(%+v) = %q, want %q", entries, got, want)
		}
	})

	t.Run("multi-entry, two different commits", func(t *testing.T) {
		entries := []contract.BlameEntry{
			{
				StartLine: 1, EndLine: 3,
				CommitHash: "aaaaaaa1111111", Author: "Alice",
				CommitDate: "2023-11-20T10:00:00Z", Summary: "Initial commit",
			},
			{
				StartLine: 4, EndLine: 4,
				CommitHash: "bbbbbbb2222222", Author: "Bob",
				CommitDate: "2024-02-15T08:30:00Z", Summary: "Add validation",
			},
		}
		want := "| Lines | Commit | Author | Date | Summary |\n" +
			"| --- | --- | --- | --- | --- |\n" +
			"| 1-3 | aaaaaaa | Alice | 2023-11-20 | Initial commit |\n" +
			"| 4 | bbbbbbb | Bob | 2024-02-15 | Add validation |\n"
		if got := renderBlameTable(entries); got != want {
			t.Errorf("renderBlameTable(%+v) = %q, want %q", entries, got, want)
		}
	})

	t.Run("pipe in summary does not corrupt table", func(t *testing.T) {
		entries := []contract.BlameEntry{
			{
				StartLine: 5, EndLine: 5,
				CommitHash: "ccccccc3333333", Author: "Carol",
				CommitDate: "2024-03-01T00:00:00Z", Summary: "Fix a | b bug",
			},
		}
		got := renderBlameTable(entries)
		rows := strings.Split(strings.TrimRight(got, "\n"), "\n")
		if len(rows) != 3 {
			t.Fatalf("renderBlameTable(%+v) produced %d rows, want 3 (header + separator + 1 data row): %q", entries, len(rows), got)
		}
		wantDataRow := "| 5 | ccccccc | Carol | 2024-03-01 | Fix a \\| b bug |"
		if rows[2] != wantDataRow {
			t.Errorf("data row = %q, want %q", rows[2], wantDataRow)
		}
	})
}

func TestRenderCodeContext(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		cc := contract.CodeContext{
			Status: contract.StatusNotFound,
			Note:   "file not found in current checkout",
		}
		want := "⚠ file not found in current checkout\n"
		if got := renderCodeContext(cc); got != want {
			t.Errorf("renderCodeContext(%+v) = %q, want %q", cc, got, want)
		}
	})

	t.Run("stale", func(t *testing.T) {
		cc := contract.CodeContext{
			Status: contract.StatusStale,
			Note:   "file has uncommitted local changes",
		}
		want := "⚠ file has uncommitted local changes\n"
		if got := renderCodeContext(cc); got != want {
			t.Errorf("renderCodeContext(%+v) = %q, want %q", cc, got, want)
		}
	})

	t.Run("ok with blame", func(t *testing.T) {
		cc := contract.CodeContext{
			Status:   contract.StatusOK,
			Language: contract.LanguageTypeScript,
			Snippet: contract.Snippet{
				StartLine: 1, EndLine: 1, TargetLine: 1,
				Code: buildSnippetCode("foo();"),
			},
			Blame: []contract.BlameEntry{
				{StartLine: 1, EndLine: 1, CommitHash: "abcdef1234567890", Author: "Alice", CommitDate: "2024-01-02T00:00:00Z", Summary: "Add foo"},
			},
		}
		want := renderSnippet(cc.Snippet, cc.Language) + renderBlameTable(cc.Blame)
		if got := renderCodeContext(cc); got != want {
			t.Errorf("renderCodeContext(%+v) = %q, want %q", cc, got, want)
		}
	})

	t.Run("ok with empty blame falls back to note", func(t *testing.T) {
		cc := contract.CodeContext{
			Status:   contract.StatusOK,
			Language: contract.LanguageTypeScript,
			Snippet: contract.Snippet{
				StartLine: 1, EndLine: 1, TargetLine: 1,
				Code: buildSnippetCode("foo();"),
			},
			Blame: nil,
			Note:  "no git repo found",
		}
		want := renderSnippet(cc.Snippet, cc.Language) + "⚠ no git repo found\n"
		if got := renderCodeContext(cc); got != want {
			t.Errorf("renderCodeContext(%+v) = %q, want %q", cc, got, want)
		}
	})

	t.Run("ok status with java language passes through to renderSnippet", func(t *testing.T) {
		cc := contract.CodeContext{
			Status:   contract.StatusOK,
			Language: contract.LanguageJava,
			Snippet: contract.Snippet{
				StartLine: 1, EndLine: 1, TargetLine: 1,
				Code: buildSnippetCode("System.out.println(\"hi\");"),
			},
			Blame: []contract.BlameEntry{
				{StartLine: 1, EndLine: 1, CommitHash: "abcdef1234567890", Author: "Alice", CommitDate: "2024-01-02T00:00:00Z", Summary: "Add main"},
			},
		}
		got := renderCodeContext(cc)
		if !strings.HasPrefix(got, "```java\n") {
			t.Errorf("renderCodeContext(%+v) = %q, want prefix %q", cc, got, "```java\n")
		}
	})
}

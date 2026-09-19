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

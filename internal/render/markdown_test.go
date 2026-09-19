package render

import (
	"fmt"
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

func TestRenderFrame(t *testing.T) {
	t.Run("own bucket with code context", func(t *testing.T) {
		f := contract.Frame{
			FilePath: "/repo/src/foo.ts", ClassName: "Foo", MethodName: "bar",
			LineNumber: 42, Bucket: contract.BucketOwn,
		}
		cc := &contract.CodeContext{
			Status: contract.StatusNotFound,
			Note:   "file not found",
		}
		want := "at Foo.bar (/repo/src/foo.ts:42) — own\n⚠ file not found\n"
		if got := renderFrame(f, cc); got != want {
			t.Errorf("renderFrame(%+v, %+v) = %q, want %q", f, cc, got, want)
		}
	})

	t.Run("own bucket with nil code context appends nothing", func(t *testing.T) {
		f := contract.Frame{
			FilePath: "/repo/src/foo.ts", MethodName: "bar",
			LineNumber: 42, Bucket: contract.BucketOwn,
		}
		want := "at bar (/repo/src/foo.ts:42) — own\n"
		if got := renderFrame(f, nil); got != want {
			t.Errorf("renderFrame(%+v, nil) = %q, want %q", f, got, want)
		}
	})

	t.Run("dependency bucket shows package name only", func(t *testing.T) {
		f := contract.Frame{
			FilePath: "/repo/node_modules/lodash/index.js", MethodName: "map",
			LineNumber: 10, Bucket: contract.BucketDependency, PackageName: "lodash",
		}
		want := "at map (/repo/node_modules/lodash/index.js:10) — dependency: lodash\n"
		if got := renderFrame(f, nil); got != want {
			t.Errorf("renderFrame(%+v, nil) = %q, want %q", f, got, want)
		}
	})

	t.Run("runtime bucket", func(t *testing.T) {
		f := contract.Frame{
			FilePath: "node:internal/process", MethodName: "processTicksAndRejections",
			LineNumber: 95, Bucket: contract.BucketRuntime,
		}
		want := "at processTicksAndRejections (node:internal/process:95) — runtime\n"
		if got := renderFrame(f, nil); got != want {
			t.Errorf("renderFrame(%+v, nil) = %q, want %q", f, got, want)
		}
	})

	t.Run("class name absent", func(t *testing.T) {
		f := contract.Frame{
			FilePath: "/repo/src/foo.ts", MethodName: "bareFunc",
			LineNumber: 5, Bucket: contract.BucketOwn,
		}
		want := "at bareFunc (/repo/src/foo.ts:5) — own\n"
		if got := renderFrame(f, nil); got != want {
			t.Errorf("renderFrame(%+v, nil) = %q, want %q", f, got, want)
		}
	})

	t.Run("column number present", func(t *testing.T) {
		f := contract.Frame{
			FilePath: "/repo/src/foo.ts", MethodName: "bar",
			LineNumber: 5, ColumnNumber: 12, Bucket: contract.BucketOwn,
		}
		want := "at bar (/repo/src/foo.ts:5:12) — own\n"
		if got := renderFrame(f, nil); got != want {
			t.Errorf("renderFrame(%+v, nil) = %q, want %q", f, got, want)
		}
	})

	t.Run("column number absent (java)", func(t *testing.T) {
		f := contract.Frame{
			FilePath: "/repo/src/Foo.java", ClassName: "Foo", MethodName: "bar",
			LineNumber: 5, Bucket: contract.BucketOwn,
		}
		want := "at Foo.bar (/repo/src/Foo.java:5) — own\n"
		if got := renderFrame(f, nil); got != want {
			t.Errorf("renderFrame(%+v, nil) = %q, want %q", f, got, want)
		}
	})
}

func TestRenderChain(t *testing.T) {
	t.Run("single node, no transition, no elided line", func(t *testing.T) {
		chain := []contract.ExceptionNode{
			{
				ClassName: "NullPointerException", Message: "boom",
				Frames: []contract.Frame{
					{FilePath: "/repo/src/foo.ts", MethodName: "bar", LineNumber: 1, Bucket: contract.BucketRuntime},
				},
			},
		}
		want := "### NullPointerException\n" +
			"> boom\n" +
			"at bar (/repo/src/foo.ts:1) — runtime\n"
		if got := renderChain(chain, nil); got != want {
			t.Errorf("renderChain(%+v, nil) = %q, want %q", chain, got, want)
		}
	})

	t.Run("two-node chain has Caused by transition between, not after last", func(t *testing.T) {
		chain := []contract.ExceptionNode{
			{
				ClassName: "RuntimeException", Message: "outer",
				Frames: []contract.Frame{
					{FilePath: "/repo/src/a.ts", MethodName: "a", LineNumber: 1, Bucket: contract.BucketRuntime},
				},
			},
			{
				ClassName: "IOException", Message: "inner",
				Frames: []contract.Frame{
					{FilePath: "/repo/src/b.ts", MethodName: "b", LineNumber: 2, Bucket: contract.BucketRuntime},
				},
			},
		}
		want := "### RuntimeException\n" +
			"> outer\n" +
			"at a (/repo/src/a.ts:1) — runtime\n" +
			"\nCaused by ↓\n\n" +
			"### IOException\n" +
			"> inner\n" +
			"at b (/repo/src/b.ts:2) — runtime\n"
		if got := renderChain(chain, nil); got != want {
			t.Errorf("renderChain(%+v, nil) = %q, want %q", chain, got, want)
		}
	})

	t.Run("elided frame count renders language-neutral line", func(t *testing.T) {
		chain := []contract.ExceptionNode{
			{
				ClassName: "Error", Message: "msg", ElidedFrameCount: 3,
				Frames: []contract.Frame{
					{FilePath: "/repo/src/a.ts", MethodName: "a", LineNumber: 1, Bucket: contract.BucketRuntime},
				},
			},
		}
		want := "### Error\n" +
			"> msg\n" +
			"at a (/repo/src/a.ts:1) — runtime\n" +
			"... 3 more frames (shared with enclosing exception)\n"
		if got := renderChain(chain, nil); got != want {
			t.Errorf("renderChain(%+v, nil) = %q, want %q", chain, got, want)
		}
	})

	t.Run("multiline message with embedded blank line stays one blockquote", func(t *testing.T) {
		chain := []contract.ExceptionNode{
			{
				ClassName: "AssertionError",
				Message:   "Expected values to be strictly equal:\n\nfoo !== bar",
				Frames:    nil,
			},
		}
		want := "### AssertionError\n" +
			"> Expected values to be strictly equal:\n" +
			">\n" +
			"> foo \\!== bar\n"
		if got := renderChain(chain, nil); got != want {
			t.Errorf("renderChain(%+v, nil) = %q, want %q", chain, got, want)
		}
	})

	t.Run("CodeContext lookup resolves by slice position, not Frame.Index field", func(t *testing.T) {
		chain := []contract.ExceptionNode{
			{
				ClassName: "Error", Message: "msg",
				Frames: []contract.Frame{
					// Index field deliberately wrong (99) -- the lookup must
					// still find this frame's CodeContext by its real slice
					// position (0), not by this field's value.
					{Index: 99, FilePath: "/repo/src/a.ts", MethodName: "a", LineNumber: 1, Bucket: contract.BucketOwn},
				},
			},
		}
		codeContexts := []contract.CodeContext{
			{
				FrameRef: contract.FrameRef{ChainIndex: 0, FrameIndex: 0},
				Status:   contract.StatusNotFound,
				Note:     "file not found",
			},
		}
		want := "### Error\n" +
			"> msg\n" +
			"at a (/repo/src/a.ts:1) — own\n" +
			"⚠ file not found\n"
		if got := renderChain(chain, codeContexts); got != want {
			t.Errorf("renderChain(%+v, %+v) = %q, want %q", chain, codeContexts, got, want)
		}
	})
}

func TestRenderDependencies(t *testing.T) {
	t.Run("nil Dependencies renders nothing", func(t *testing.T) {
		if got := renderDependencies(nil); got != "" {
			t.Errorf("renderDependencies(nil) = %q, want %q", got, "")
		}
	})

	t.Run("exact-match package", func(t *testing.T) {
		d := &contract.Dependencies{
			ManifestFile: contract.ManifestFilePackageJSON,
			Direct:       map[string]string{"react": "^18.2.0"},
			Locked: map[string]contract.LockedDependency{
				"react": {Version: "18.2.5"},
			},
		}
		want := "## Dependencies\n" +
			"- react — declared ^18.2.0, resolved 18.2.5\n"
		if got := renderDependencies(d); got != want {
			t.Errorf("renderDependencies(%+v) = %q, want %q", d, got, want)
		}
	})

	t.Run("fallback-match package with note, no Direct entry", func(t *testing.T) {
		d := &contract.Dependencies{
			ManifestFile: contract.ManifestFilePackageJSON,
			Direct:       map[string]string{},
			Locked: map[string]contract.LockedDependency{
				"lodash": {
					Version: "4.17.21",
					Note:    "resolved via top-level lookup, not tied to this exact frame path",
				},
			},
		}
		want := "## Dependencies\n" +
			"- lodash — resolved 4.17.21 (resolved via top\\-level lookup, not tied to this exact frame path)\n"
		if got := renderDependencies(d); got != want {
			t.Errorf("renderDependencies(%+v) = %q, want %q", d, got, want)
		}
	})

	t.Run("fully-unresolved package", func(t *testing.T) {
		d := &contract.Dependencies{
			ManifestFile: contract.ManifestFilePomXML,
			Direct:       map[string]string{},
			Locked: map[string]contract.LockedDependency{
				"com.example:leftpad": {
					Note: "no local mvn/gradle cache on this checkout",
				},
			},
		}
		want := "## Dependencies\n" +
			"- com.example:leftpad — resolved unresolved (no local mvn/gradle cache on this checkout)\n"
		if got := renderDependencies(d); got != want {
			t.Errorf("renderDependencies(%+v) = %q, want %q", d, got, want)
		}
	})

	t.Run("multi-entry map renders identically and alphabetically every time", func(t *testing.T) {
		d := &contract.Dependencies{
			ManifestFile: contract.ManifestFilePackageJSON,
			Direct: map[string]string{
				"react": "^18.2.0",
				"zod":   "^3.22.0",
			},
			Locked: map[string]contract.LockedDependency{
				"react":  {Version: "18.2.5"},
				"lodash": {Version: "4.17.21"},
				"zod":    {Version: "3.22.4"},
			},
		}
		want := "## Dependencies\n" +
			"- lodash — resolved 4.17.21\n" +
			"- react — declared ^18.2.0, resolved 18.2.5\n" +
			"- zod — declared ^3.22.0, resolved 3.22.4\n"
		for i := 0; i < 5; i++ {
			if got := renderDependencies(d); got != want {
				t.Fatalf("renderDependencies(%+v) iteration %d = %q, want %q", d, i, got, want)
			}
		}
	})
}

func TestRenderRawInput(t *testing.T) {
	t.Run("plain input uses minimum 3-backtick fence", func(t *testing.T) {
		want := "<details><summary>Raw input</summary>\n\n" +
			"```\n" +
			"Error: boom\n    at foo (/repo/src/a.ts:1:1)\n" +
			"```\n" +
			"</details>\n"
		if got := renderRawInput("Error: boom\n    at foo (/repo/src/a.ts:1:1)", false); got != want {
			t.Errorf("renderRawInput(...) = %q, want %q", got, want)
		}
	})

	t.Run("input containing a 3-backtick run gets a 4-backtick fence", func(t *testing.T) {
		raw := "some text with ``` a fenced block inside it"
		want := "<details><summary>Raw input</summary>\n\n" +
			"````\n" +
			raw + "\n" +
			"````\n" +
			"</details>\n"
		if got := renderRawInput(raw, false); got != want {
			t.Errorf("renderRawInput(%q, false) = %q, want %q", raw, got, want)
		}
	})

	t.Run("truncated true includes note with correct KB figure", func(t *testing.T) {
		got := renderRawInput("some input", true)
		wantNote := fmt.Sprintf("⚠ input truncated at the %d KB cap\n\n", contract.RawInputCapBytes/1024)
		if !strings.Contains(got, wantNote) {
			t.Errorf("renderRawInput(..., true) = %q, want it to contain %q", got, wantNote)
		}
	})

	t.Run("truncated false has no note", func(t *testing.T) {
		got := renderRawInput("some input", false)
		if strings.Contains(got, "truncated") {
			t.Errorf("renderRawInput(..., false) = %q, want no truncation note", got)
		}
	})
}

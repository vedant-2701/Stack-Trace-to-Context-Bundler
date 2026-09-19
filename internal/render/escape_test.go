package render

import "testing"

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// One case per character in escapeCharSet, in isolation.
		{"backslash", `a\b`, `a\\b`},
		{"backtick", "a`b", "a\\`b"},
		{"asterisk", "a*b", `a\*b`},
		{"underscore", "a_b", `a\_b`},
		{"open brace", "a{b", `a\{b`},
		{"close brace", "a}b", `a\}b`},
		{"open bracket", "a[b", `a\[b`},
		{"close bracket", "a]b", `a\]b`},
		{"open paren", "a(b", `a\(b`},
		{"close paren", "a)b", `a\)b`},
		{"hash", "a#b", `a\#b`},
		{"plus", "a+b", `a\+b`},
		{"minus", "a-b", `a\-b`},
		{"period", "a.b", `a\.b`},
		{"bang", "a!b", `a\!b`},
		{"pipe", "a|b", `a\|b`},
		{"greater than", "a>b", `a\>b`},
		{"less than", "a<b", `a\<b`},

		// No-op: plain text with no special characters is unchanged.
		{"no-op plain text", "just some ordinary prose, nothing special", "just some ordinary prose, nothing special"},

		// Combined: several special characters together in one string,
		// including a repeated character.
		{"combined", "a_b*c|d#e_f", `a\_b\*c\|d\#e\_f`},

		// Generic-type case: '<' and '>' together, the real-world shape
		// spec.md requirement 19 calls out (e.g. Java/TS generic types
		// landing in an exception Message).
		{"generic type List<String>", "List<String>", `List\<String\>`},
		{"generic type Map<string, number>", "Map<string, number>", `Map\<string, number\>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

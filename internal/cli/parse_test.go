package cli

import (
	"strings"
	"testing"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		want          string
		wantErrSubstr string
	}{
		{name: "empty defaults to markdown", in: "", want: "markdown"},
		{name: "explicit markdown", in: "markdown", want: "markdown"},
		{name: "explicit json", in: "json", want: "json"},
		{name: "invalid value", in: "yaml", wantErrSubstr: `invalid --format value "yaml"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateFormat(tc.in)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("err = %q, want it to contain %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("got = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateLang(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		want          string
		wantErrSubstr string
	}{
		{name: "empty defers to auto-detection", in: "", want: ""},
		{name: "java", in: "java", want: "java"},
		{name: "typescript", in: "typescript", want: "typescript"},
		{name: "invalid value", in: "cobol", wantErrSubstr: `invalid --lang value "cobol"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateLang(tc.in)

			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("err = nil, want error containing %q", tc.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("err = %q, want it to contain %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("got = %q, want %q", got, tc.want)
			}
		})
	}
}

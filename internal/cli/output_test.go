package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeliver_Stdout(t *testing.T) {
	tests := []struct {
		name   string
		bundle string
		want   string
	}{
		{name: "no trailing newline: one is appended", bundle: "hello", want: "hello\n"},
		{name: "already ends with a newline: not doubled", bundle: "hello\n", want: "hello\n"},
		{name: "multiple trailing newlines: only checked, not trimmed", bundle: "hello\n\n", want: "hello\n\n"},
		{name: "empty bundle: still gets a trailing newline", bundle: "", want: "\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout strings.Builder

			if err := deliver(tc.bundle, "", &stdout); err != nil {
				t.Fatalf("deliver() err = %v, want nil", err)
			}
			if stdout.String() != tc.want {
				t.Errorf("stdout = %q, want %q", stdout.String(), tc.want)
			}
		})
	}
}

func TestDeliver_File(t *testing.T) {
	t.Run("writes a new file with the newline rule applied", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.md")

		if err := deliver("hello", path, nil); err != nil {
			t.Fatalf("deliver() err = %v, want nil", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading written file: %v", err)
		}
		if string(got) != "hello\n" {
			t.Errorf("file content = %q, want %q", string(got), "hello\n")
		}
	})

	t.Run("silently overwrites an existing file, fully truncated not prefixed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.md")
		if err := os.WriteFile(path, []byte("old content, much longer than the new one\n"), 0o600); err != nil {
			t.Fatalf("writing pre-existing file: %v", err)
		}

		if err := deliver("new", path, nil); err != nil {
			t.Fatalf("deliver() err = %v, want nil", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading written file: %v", err)
		}
		if string(got) != "new\n" {
			t.Errorf("file content = %q, want %q", string(got), "new\n")
		}
	})

	t.Run("open failure (nonexistent parent directory) returns a non-nil error and leaves nothing behind", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "does-not-exist", "out.md")

		if err := deliver("hello", path, nil); err == nil {
			t.Fatal("deliver() err = nil, want a non-nil error")
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("stat(%q) = %v, want a not-exist error -- deliver should not have created anything", path, err)
		}
	})
}

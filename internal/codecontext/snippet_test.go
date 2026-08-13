package codecontext

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// writeLines creates a temp file with n lines, each just its own 1-based
// line number as text (e.g. "1", "2", ...), so assertions can check
// expected content directly off the line numbers involved.
func writeLines(t *testing.T, dir, name string, n int) string {
	t.Helper()
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString(strconv.Itoa(i))
		b.WriteString("\n")
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestSnippet_NormalWindow(t *testing.T) {
	path := writeLines(t, t.TempDir(), "app.go", 30)

	snip, err := buildSnippet(path, 15, DefaultContextLines)
	if err != nil {
		t.Fatalf("buildSnippet: unexpected error: %v", err)
	}
	if snip.StartLine != 10 || snip.EndLine != 20 || snip.TargetLine != 15 {
		t.Errorf("got {Start:%d End:%d Target:%d}, want {10 20 15}", snip.StartLine, snip.EndLine, snip.TargetLine)
	}
	wantCode := "10\n11\n12\n13\n14\n15\n16\n17\n18\n19\n20\n"
	if snip.Code != wantCode {
		t.Errorf("Code = %q, want %q", snip.Code, wantCode)
	}
}

func TestSnippet_TargetLineOne(t *testing.T) {
	path := writeLines(t, t.TempDir(), "app.go", 30)

	snip, err := buildSnippet(path, 1, DefaultContextLines)
	if err != nil {
		t.Fatalf("buildSnippet: unexpected error: %v", err)
	}
	// Clamped at the start -- can't go below line 1, so the window
	// isn't symmetric here.
	if snip.StartLine != 1 || snip.EndLine != 6 || snip.TargetLine != 1 {
		t.Errorf("got {Start:%d End:%d Target:%d}, want {1 6 1}", snip.StartLine, snip.EndLine, snip.TargetLine)
	}
}

func TestSnippet_TargetLineAtEOF(t *testing.T) {
	path := writeLines(t, t.TempDir(), "app.go", 30)

	snip, err := buildSnippet(path, 30, DefaultContextLines)
	if err != nil {
		t.Fatalf("buildSnippet: unexpected error: %v", err)
	}
	// Clamped at the end -- file only has 30 lines.
	if snip.StartLine != 25 || snip.EndLine != 30 || snip.TargetLine != 30 {
		t.Errorf("got {Start:%d End:%d Target:%d}, want {25 30 30}", snip.StartLine, snip.EndLine, snip.TargetLine)
	}
}

func TestSnippet_FileShorterThanWindow(t *testing.T) {
	path := writeLines(t, t.TempDir(), "app.go", 4) // fewer than the 11-line window

	snip, err := buildSnippet(path, 2, DefaultContextLines)
	if err != nil {
		t.Fatalf("buildSnippet: unexpected error: %v", err)
	}
	if snip.StartLine != 1 || snip.EndLine != 4 || snip.TargetLine != 2 {
		t.Errorf("got {Start:%d End:%d Target:%d}, want {1 4 2}", snip.StartLine, snip.EndLine, snip.TargetLine)
	}
	wantCode := "1\n2\n3\n4\n"
	if snip.Code != wantCode {
		t.Errorf("Code = %q, want %q", snip.Code, wantCode)
	}
}

func TestSnippet_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.go")

	_, err := buildSnippet(path, 10, DefaultContextLines)
	if err == nil {
		t.Fatal("buildSnippet: expected an error for a missing file, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want errors.Is(err, fs.ErrNotExist)", err)
	}
}

func TestSnippet_PermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: file permissions don't block reads, can't exercise this case")
	}

	path := writeLines(t, t.TempDir(), "unreadable.go", 10)

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	_, err := buildSnippet(path, 5, DefaultContextLines)
	if err == nil {
		t.Fatal("buildSnippet: expected an error for an unreadable file, got nil")
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v: matched fs.ErrNotExist, want a distinct permission error (file exists, just unreadable)", err)
	}
}

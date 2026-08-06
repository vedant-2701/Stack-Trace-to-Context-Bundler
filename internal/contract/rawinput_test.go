package contract

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateRawInput_OneByteUnderCap(t *testing.T) {
	s := strings.Repeat("a", rawInputCapBytes-1)

	out, truncated := TruncateRawInput(s)
	if truncated {
		t.Errorf("truncated = true, want false for input one byte under the cap")
	}
	if out != s {
		t.Errorf("out was modified for input one byte under the cap")
	}
}

func TestTruncateRawInput_ExactlyAtCap(t *testing.T) {
	s := strings.Repeat("a", rawInputCapBytes)

	out, truncated := TruncateRawInput(s)
	if truncated {
		t.Errorf("truncated = true, want false for input exactly at the cap")
	}
	if out != s {
		t.Errorf("out was modified for input exactly at the cap")
	}
}

func TestTruncateRawInput_OneByteOverCap_ASCII(t *testing.T) {
	s := strings.Repeat("a", rawInputCapBytes+1)

	out, truncated := TruncateRawInput(s)
	if !truncated {
		t.Fatalf("truncated = false, want true for input one byte over the cap")
	}
	if len(out) != rawInputCapBytes {
		t.Errorf("len(out) = %d, want exactly %d for ASCII input one byte over the cap", len(out), rawInputCapBytes)
	}
	if out != s[:rawInputCapBytes] {
		t.Errorf("out doesn't match the expected prefix")
	}
}

// TestTruncateRawInput_CapFallsMidRune covers the case tasks.md's T006
// scoping amendment exists for: the exact byte cap landing inside a
// multi-byte rune. The cut must back up to the last valid rune boundary
// rather than emit invalid UTF-8.
func TestTruncateRawInput_CapFallsMidRune(t *testing.T) {
	prefix := strings.Repeat("a", rawInputCapBytes-1)
	s := prefix + "\u00e9" // "é", 2 bytes (0xC3 0xA9) -- its second byte would land exactly at index rawInputCapBytes

	out, truncated := TruncateRawInput(s)
	if !truncated {
		t.Fatalf("truncated = false, want true")
	}
	if !utf8.ValidString(out) {
		t.Errorf("out is not valid UTF-8: %q", out)
	}
	if out != prefix {
		t.Errorf("out = %q, want the ASCII prefix with the split rune dropped entirely, not partially included", out)
	}
	if len(out) >= rawInputCapBytes {
		t.Errorf("len(out) = %d, want strictly under %d (the split rune must be fully excluded)", len(out), rawInputCapBytes)
	}
}

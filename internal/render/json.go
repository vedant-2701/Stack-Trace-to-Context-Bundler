package render

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// JSON renders b as a compact, HTML-unescaped JSON string of the raw
// contract.Bundle shape, with no trailing newline (spec.md reqs. 1-4).
// No per-field logic: the entire shape and every omission rule already
// live in contract.Bundle's own struct tags (Article IV).
func JSON(b contract.Bundle) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(b); err != nil {
		// contract.Bundle's field types (string/int/bool/slice/map/
		// pointer-to-struct only) can never make JSON serialization
		// fail (spec.md req. 1) -- a non-nil error here means that
		// invariant itself broke, a genuine programmer-error condition
		// (CONVENTIONS.md's panic carve-out), not a runtime condition
		// to swallow (qa-log.md Q7).
		panic(fmt.Sprintf("render.JSON: contract.Bundle failed to encode, which should be impossible: %v", err))
	}

	// Encoder.Encode always appends its own trailing '\n'; trim exactly
	// that one byte (spec.md req. 4).
	return string(bytes.TrimSuffix(buf.Bytes(), []byte("\n")))
}

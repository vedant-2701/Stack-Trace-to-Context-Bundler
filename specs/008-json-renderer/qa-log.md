# Interrogation Log: JSON renderer (008)

Question-and-answer record from spec interrogation, kept separately from
`spec.md` so a long chat session doesn't lose this context. Not a
replacement for `progress.md` (which logs task completion) or `spec.md`
(which states the resolved requirements) -- this is the raw negotiation
trail behind them.

---

**Q1 — Does `render.JSON` return an error?**
Considered: no-error `func JSON(b contract.Bundle) string` (matches
`Markdown`'s precedent) vs. `(string, error)` defensively.
**Answer:** No-error signature. `contract.Bundle` has no field type that
can make `json.Marshal` fail (no chans/funcs/complex/cycles). Confirmed
after a direct "confirm" check against that reasoning.

---

**Q2 — Indentation: pretty-printed or compact?**
Considered: `json.MarshalIndent(b, "", "  ")` (matches 001's own golden
fixtures in `internal/contract/testdata/*.json`) vs. compact
`json.Marshal(b)`.
**Answer:** Compact. Rationale (user's): both Markdown and JSON output
are pasted into an LLM chat by different developers (not "JSON is for
scripting, Markdown is for chat" -- that framing was wrong), so
indentation whitespace is a real, avoidable token cost. 001's
`MarshalIndent`-based fixtures are that feature's own test-only
convention, not a contract requirement 008 must match, and are NOT
treated as the source of truth for 008's output shape.

---

**Q3 — Disable HTML-escaping (`<`, `>`, `&` → `\u003c` etc.)?**
**Answer:** Yes, disable it (`json.Encoder` + `SetEscapeHTML(false)` --
plain `json.Marshal` can't do this). Same token-efficiency rationale as
Q2, plus it avoids ugly `\u003c` noise in messages/notes containing
generics (`List<String>`, `Map<string, number>` -- a real shape per
006a's own fixtures).

---

**Q4 — API shape for compact vs. indented output; is 008 building both?**
Considered: Option A, two named functions (`JSON` / `JSONIndented`) vs.
Option B, one function with a `pretty bool` param.
**Answer:** Option A's *naming pattern* (named functions over a bool
param) is the right call if an indented variant is ever built -- but
008 does NOT build it now. Only `JSON(b contract.Bundle) string`
(compact, HTML-escaping disabled) ships in this feature. A future
`--pretty` CLI flag is tracked as new feature **013** in
`specs/INDEX.md` (depends on 008, 002a), not folded into
`known-gaps.md` and not implemented as part of 008 -- keeps 008 scoped
to what it actually ships today, and doesn't reopen 002a (already
`Implemented`) to add a flag for a capability 008 doesn't expose yet.

---

**Q5 — Trailing newline after the final `}`?**
Clarified the actual question first: an embedded newline *inside* a JSON
string value is always encoded as the `\n` escape sequence by
`encoding/json` regardless of anything decided here -- that's required
by the JSON spec, not a choice. The open question was only the
terminator after the closing `}` itself.
**Answer:** No terminator -- output ends immediately at `}`, like a
minified JS/JSON file. Same token-efficiency rationale as Q2/Q3.
Implementation note (not a spec-level decision): `json.Encoder.Encode`
(needed for `SetEscapeHTML(false)`, per Q3) always appends its own
trailing `\n` -- `json.go` must trim that one byte off before returning,
since the no-terminator output isn't reachable via `Encoder.Encode`'s
default behavior alone.

---

**Q6 — Testing strategy.**
`render.JSON` is a thin pass-through -- the omitempty/nil-pointer/map-
key-sort semantics are `contract` package guarantees, already
exhaustively tested by 001's `types_test.go`. 008 only needs to prove
its own logic: compact output, disabled HTML-escaping, no trailing
byte, and no data corruption.
**Answer:** Agreed as proposed:
- Golden-file tests reusing existing `internal/render/fixtures_test.go`
  builders (same package, no duplication): `tsBasicBundle`,
  `noGitMetadataBundle`, `noDependenciesBundle`,
  `markdownSpecialCharsBundle` (`<`/`>` unescaped),
  `dependencyStatesBundle` (map-key sort + version/note combos),
  `runtimeVersionStatesBundle` (`VersionSourceUnknown`).
- One NEW fixture for the `&` case (nothing existing covers it, and Q3's
  HTML-escaping decision needs it verified) -- in a NEW file,
  `internal/render/json_fixtures_test.go`, not by editing 007's already-
  done `fixtures_test.go`.
- Golden files under `internal/render/testdata/golden_json/*.golden.json`.
  Harness reuses the existing `var updateGolden` from `markdown_test.go`
  (compile-conflict issue flagged earlier) rather than redeclaring it.
- Two extra non-golden unit tests: (a) output never ends in `\n`,
  (b) round-trip `json.Unmarshal` back into `contract.Bundle` +
  `reflect.DeepEqual` against the input.
- `assertGoldenJSON` is a NEW, separate helper (near-duplicate of
  `assertGoldenMarkdown`) rather than refactoring both into one shared
  helper -- matches the existing precedent that 001's `assertGolden` and
  007's `assertGoldenMarkdown` are already independent near-duplicates,
  and avoids touching a done feature's file for a marginal DRY win.

---

**Q7 — `JSON`'s own signature has no error return (Q1) -- but
`Encoder.Encode` itself still returns one internally. What happens to
that value?**
Considered: silently discard via `_ = Encode(b)` with a justifying
comment (the discard pattern `CONVENTIONS.md`'s errcheck rule permits)
vs. panic on non-nil.
**Answer:** Panic. `Encode` can only fail for unsupported field types
(chans/funcs/complex numbers) or cycles -- Q1 already confirmed
`contract.Bundle` contains none of those, so a non-nil error here would
mean that invariant broke (e.g. a future field change violating Article
IV), not a reachable runtime condition. `CONVENTIONS.md` reserves panics
for exactly this: "genuine programmer-error invariants, not runtime
conditions." Discarding it instead was rejected -- Article VI treats
handing back unverified/malformed output as confirmed as worse than a
loud failure, and a silent `_ =` here would do exactly that if the
invariant were ever violated.

---

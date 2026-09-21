# Plan: JSON renderer

Derived from `spec.md`. Consistent with `memory/constitution.md` (Articles
II, IV, VI, VIII) and `CONVENTIONS.md`.

## Architecture / approach

One pure function, `render.JSON(b contract.Bundle) string`, implemented
via a `json.Encoder` writing into a `bytes.Buffer`, with
`SetEscapeHTML(false)` set before calling `Encode(b)` (spec.md req. 3).
`Encoder.Encode` always appends its own trailing `\n`; the function trims
exactly that one byte off the buffer's contents before returning as a
string (spec.md req. 4) — e.g. `bytes.TrimSuffix(buf.Bytes(), []byte("\n"))`
or an equivalent length-check-and-slice. Plain `json.Marshal`/
`json.MarshalIndent` are not used anywhere in this feature: neither
offers a way to disable HTML-escaping, which req. 3 requires.

`Encode` itself returns an `error` regardless of req. 1's no-error API
for `JSON` -- that return value is not discarded silently. `JSON` panics
if `Encode` ever returns non-nil: req. 1 already established that a
well-typed `contract.Bundle` (no chans/funcs/complex/cycles) cannot
produce one, so a non-nil error here means that invariant itself broke
(e.g. a future field violating Article IV) -- a genuine programmer-error
condition per `CONVENTIONS.md`'s panic carve-out, not a runtime condition
to swallow. Silently discarding via `_ = Encode(b)` was rejected: per
Article VI, handing back malformed/truncated output as if it were a
valid serialization is worse than a loud failure (qa-log.md Q7).

No per-field helpers exist, unlike `Markdown` (007), which composes many
field-by-field rendering helpers because its output is a human-curated
document layout. `JSON` has no per-field logic at all — the entire shape
and every omission rule already live in `contract.Bundle`'s own struct
tags (Article IV). This is the one place in `internal/render` where "one
canonical contract, never duplicated" is most directly visible: this
feature adds zero shape logic of its own, only a formatting-knob
decision (compact / unescaped / no-terminator) applied uniformly to
whatever `contract.Bundle` value it's given.

## Stack & versions

Standard library only: `encoding/json`, `bytes`. No new dependency
(Article VIII) — nothing here needs more than what `encoding/json`'s
`Encoder` already exposes.

## Data model

Consumes `contract.Bundle` as-is; defines no new exported or unexported
types.

## File / module layout

```
internal/render/
├── json.go                 JSON() -- the only production code this
│                            feature adds
├── json_test.go             golden-file tests (table-driven, reusing
│                            the existing markdown_test.go's
│                            `updateGolden` flag) + TestJSON_NoTrailing
│                            Newline + TestJSON_RoundTrip
├── json_fixtures_test.go    one new builder: ampersandBundle -- the
│                            only new fixture this feature needs; every
│                            other fixture is reused from
│                            fixtures_test.go (same package `render`,
│                            written for 007, no changes needed there)
└── testdata/
    └── golden_json/
        ├── ts_basic.golden.json
        ├── no_git_metadata.golden.json
        ├── no_dependencies.golden.json
        ├── markdown_special_chars.golden.json  -- reuses 007's fixture
        │                                          name; covers <, >
        ├── ampersand.golden.json               -- the one case no
        │                                          reused fixture covers
        ├── dependency_states.golden.json
        └── runtime_version_states.golden.json
```

`markdown.go`/`markdown_test.go`/`fixtures_test.go` (007) are read-only
inputs to this feature — no changes made to any of them.

## API / contracts

```go
package render

// JSON renders b as a compact, HTML-unescaped JSON string of the raw
// contract.Bundle shape, with no trailing newline.
func JSON(b contract.Bundle) string
```

No other exported surface.

## Testing strategy

- **Golden-file tests** (`json_test.go`), one per fixture in the table
  above — table-driven, per `CONVENTIONS.md`. Reuses the SAME
  `var updateGolden` flag already declared in `markdown_test.go` (same
  package `render`) — redeclaring it would be a duplicate package-level
  identifier and a compile error, not merely a runtime conflict.
  `-update` regenerates fixtures the same way `TestMarkdown`'s harness
  does.
- `assertGoldenJSON(t, got, path)` — a new, separate helper mirroring
  `assertGoldenMarkdown` exactly (byte-for-byte comparison / `-update`
  rewrite). Kept separate rather than refactored into one shared helper
  with `assertGoldenMarkdown`, to avoid touching 007's already-done
  `markdown_test.go` for a marginal DRY win (qa-log.md Q6) — matches the
  existing precedent that 001's `assertGolden` and 007's
  `assertGoldenMarkdown` are themselves already independent
  near-duplicates, not shared.
- **`TestJSON_NoTrailingNewline`**: asserts
  `!strings.HasSuffix(JSON(b), "\n")`, run against at least
  `tsBasicBundle`.
- **`TestJSON_RoundTrip`**: `json.Unmarshal([]byte(JSON(b)), &got)` then
  `reflect.DeepEqual(b, got)` — run against `tsBasicBundle` (baseline)
  and `dependencyStatesBundle` (map coverage: multiple `Direct`/`Locked`
  entries, mixed resolved/unresolved/fallback states). This is the
  cheapest real proof that compacting and unescaping never drop or
  mutate data — a byte-diff golden failure would also catch this, but
  less legibly than a direct equality assertion.
- No fakes/mocks needed — `JSON` is pure, no subprocess or filesystem
  interaction, same reasoning as `Markdown`.

## Risks & open decisions

- Unlike 007's plan.md (which flags "a future contract field could go
  unrendered" as a risk needing a `known-gaps.md` entry when 005a/005b
  land), that risk does NOT apply here: this feature has zero per-field
  logic, so any future field added to `contract.Bundle` appears in
  `JSON`'s output automatically, with no code change needed. Worth
  stating explicitly so a future reader doesn't assume 008 needs the
  same revisit-on-new-field discipline 007 does.
- Open, deferred, not blocking: whether feature 013 (`--pretty` flag)
  ends up adding a second exported function (`JSONIndented`, matching
  qa-log.md Q4's naming-pattern preference) or a parameter to `JSON` —
  left to that feature's own spec interrogation, not decided now
  (Article VIII).

## Alternatives considered

- **`json.MarshalIndent` (pretty) as the default** — rejected: token-cost
  reasoning (qa-log.md Q2), even though it matches 001's own
  `types_test.go` golden-fixture convention; that convention was
  explicitly ruled not binding on this feature's output shape.
- **Plain `json.Marshal`** — rejected: offers no way to disable
  HTML-escaping, which spec.md req. 3 requires.
- **A shared `assertGoldenFile` helper across 007 and 008** — considered,
  rejected for now: touches a done feature's test file for a marginal
  DRY benefit (qa-log.md Q6).
- **Returning `(string, error)`** — rejected: no real failure mode exists
  for a well-typed `contract.Bundle` (qa-log.md Q1).

# Spec: JSON renderer

**Status:** Planned — plan.md and tasks.md written; ready for implementation.
**Folder:** specs/008-json-renderer
**Depends on:** 001-data-contract (done)

## Overview

Renders a `contract.Bundle` into a compact JSON string of the raw
canonical contract shape — the second of two renderer formats (007
Markdown is the other); both read the same canonical `contract.Bundle`
and neither defines or modifies its shape (constitution Article IV).

Unlike Markdown, which curates a human-readable document layout, JSON
performs no reformatting or reinterpretation of the data at all — it is
the contract's own shape, serialized as-is. Both output formats are
pasted directly into an LLM chat by developers who simply prefer one
prompt shape over the other (not "JSON is for scripting, Markdown is for
chat" — both are chat-paste formats); because of that, this renderer is
deliberately optimized for minimal token footprint: compact (no
indentation), no unnecessary character-escaping, and no trailing
terminator byte.

Real, end-to-end bundles for v1 are TS/JS-shaped only, same as 007 — 005a
(Java parser) and 005b (Java dependency resolution) are both `idea`
status. This feature's logic has no field-level branching at all (see
Functional requirements), so language-shape coverage is not a real
concern the way it was for 007.

## User stories

- As a developer who prefers pasting structured JSON rather than
  Markdown into an AI chat, I want the bundle serialized compactly, so I
  don't spend context tokens on indentation whitespace I don't need.
- As a developer whose stack trace messages or notes contain generic-type
  syntax (`List<String>`, `Map<string, number>`) or other `<`/`>`/`&`
  characters, I want them to appear literally in the JSON output, not
  HTML-escaped as `\u003c` etc., so the output stays both legible and
  token-efficient.
- As a developer scripting against the bundle output (e.g. piping through
  `jq` or a custom tool), I want the raw, unmodified contract shape with
  nothing renderer-specific added or removed, so I can process it without
  special-casing this feature's output.

## Functional requirements

1. Public API: `func JSON(b contract.Bundle) string` in
   `internal/render/json.go` (package `render`, per `CONVENTIONS.md`'s
   file/folder layout). No error return: `contract.Bundle`'s field types
   (string/int/bool/slice/map/pointer-to-struct only) can never make JSON
   serialization fail — no channels, funcs, complex numbers, or cycles
   exist anywhere in the shape.
2. Output is compact: no added whitespace or indentation between JSON
   tokens (`json.Marshal`'s default form, never `json.MarshalIndent`).
3. HTML-escaping is disabled: `<`, `>`, and `&` appear literally in
   output strings, never as `\u003c`/`\u003e`/`\u0026`. This requires
   `json.Encoder.SetEscapeHTML(false)` — plain `json.Marshal` offers no
   way to disable this.
4. Output has no trailing newline or any other terminator after the
   final `}` — it ends immediately at the closing brace, matching
   minified-JSON convention. (`json.Encoder.Encode` appends its own
   trailing `\n`; requirement 4 means trimming exactly that one byte
   before returning.)
5. The full `contract.Bundle` is serialized as-is: no field filtering,
   transformation, or reformatting of any value. Every `omitempty` /
   pointer-nil-omission / enum-value behavior is exactly what the
   `contract` package's own struct tags already produce — this feature
   adds no shape logic of its own (Article IV).
6. Map fields (`Dependencies.Direct`, `Dependencies.Locked`) serialize
   with deterministic, alphabetically-sorted keys. This is
   `encoding/json`'s own built-in map-marshaling behavior, not logic this
   feature implements — stated here only because 008's output
   determinism (needed for golden-file tests) depends on it, the same
   property 007 had to implement manually via an explicit sort.

## Non-functional requirements

- Pure function, no I/O, no subprocess calls — same purity constraint as
  `Markdown` (007) and `Detect()` (003a).
- Output must remain valid JSON for any `contract.Bundle` value
  satisfying the `contract` package's own invariants, including
  pathological developer-arbitrary string content (embedded newlines,
  quotes, backslashes, `<`/`>`/`&`). Standard JSON string-escaping
  (quotes, backslashes, control characters) still applies via
  `encoding/json` — only HTML-escaping is disabled (requirement 3).
- Language-agnostic: no branch in this feature's own logic varies by
  `Bundle.Language` — the entire bundle serializes identically regardless
  of source language.
- Deliberately minimized relative to output size: no added whitespace, no
  added terminator — output is pasted directly into constrained LLM
  context the same way 007's Markdown output is (see Overview).

## Out of scope

- Pipeline wiring / writing the returned string to stdout, a file, or the
  clipboard — 002b (`idea`) and 009 (`idea`), same carve-out as 007.
- A `--pretty` / indented output mode — tracked separately as feature
  **013** in `specs/INDEX.md` (depends on 008, 002a; `idea`), not built
  as part of this feature.
- Any transformation, filtering, or reinterpretation of
  `contract.Bundle`'s fields — this feature is a pure format-shape
  decision (compact / unescaped / no-terminator), never a data decision.
- Re-verifying the `contract` package's own omitempty / pointer-nil-
  omission / enum-value semantics — already exhaustively tested by 001's
  `types_test.go`. This feature's own tests target only its own added
  behavior (compactness, escaping, terminator, round-trip fidelity).
- Testing against real Java-parser output shapes — same carve-out as
  007: reused fixtures only need to be shape-valid `contract.Bundle`
  values, not stand-ins for 005a's parser correctness.

## Acceptance criteria

- [ ] Given a realistic two-node bundle (`tsBasicBundle`), when rendered,
      then the output is valid, compact JSON with no added whitespace
      between tokens, matching a checked-in golden fixture exactly.
- [ ] Given a bundle with `GitMetadata == nil`, and, separately, one with
      `Dependencies == nil`, when rendered, then the corresponding JSON
      key (`gitMetadata` / `dependencies`) is entirely absent from the
      output — not `null`, not an empty object — matching a checked-in
      golden fixture for each.
- [ ] Given a `Message`/`Note` containing `<` and `>` together, and,
      separately, one containing `&`, when rendered, then those
      characters appear literally in the output, never as
      `\u003c`/`\u003e`/`\u0026`.
- [ ] Given `Bundle.Dependencies` with multiple `Locked`/`Direct` entries,
      when rendered repeatedly, then the map keys appear in stable,
      alphabetically-sorted order every time.
- [ ] Given `Runtime.VersionSource == VersionSourceUnknown` with no
      `Version`/`Note`, when rendered, then the output matches the golden
      fixture with `version`/`note` keys both absent and `versionSource`
      present as `"unknown"`.
- [ ] Given any bundle, when rendered, then the returned string never
      ends in a `\n` byte.
- [ ] Given any bundle, when the rendered output is `json.Unmarshal`ed
      back into a fresh `contract.Bundle`, then the result is
      `reflect.DeepEqual` to the original input bundle.

## Open questions

None — all resolved during interrogation. Full session log in
`specs/008-json-renderer/qa-log.md`.

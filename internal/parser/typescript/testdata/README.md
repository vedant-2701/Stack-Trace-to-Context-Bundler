# testdata fixtures — index

Every fixture below is either a **real** capture (copied verbatim from
`specs/006a-ts-js-parser/testdata/{fetch-refused,round2-altprint-tsxpath,
full-machine-reverify}/ERRORS.md`, or from the four original root-level
`node-*.txt` files, or from this conversation's live 3-level `.cause`
capture) or explicitly marked **synthetic** (hand-written, no real
capture exists for that shape — see `spec.md`'s "Behavior & patterns"
note on synthetic fixtures). FR numbers refer to `spec.md`'s Functional
requirements; AC bullets refer to its Acceptance criteria list.

## Real captures

| Fixture | Source | Exercises |
|---|---|---|
| `crash-with-cause.txt` | `node-uncaught-crash-with-cause.txt` | FR2(a) sync-throw preamble, FR7 `[cause]:` parsing, FR8 elision, FR14 version-from-trace, FR21 frame order — AC1 |
| `crash-typeerror-no-cause.txt` | `node-uncaught-crash-typeerror-no-cause.txt` | FR2(a) sync-throw preamble, no-cause case, FR14 version-from-trace |
| `logged-object-with-cause.txt` | `node-console-error-object-with-cause.txt` | FR2(b) logged-object shape, FR7, FR8, FR15 local-environment version fallback — AC2 |
| `bare-stack.txt` | `node-console-error-bare-stack.txt` | FR2(c) bare `.stack` shape, FR9 single-node no-cause — AC3 |
| `crash-3level-cause.txt` | live capture, this conversation | FR4 indentation growth confirmed at 3+ nesting levels (+2 spaces/level), FR7, FR8, FR21 — resolves the previously-unverified claim in `CAPTURE-FINDINGS.md` §A1 |
| `crash-async-rejection-preamble.txt` | `full-machine-reverify` #1 shape A | FR2(a) unhandled-promise-rejection preamble variant (distinct from sync-throw) |
| `logged-object-fetch-cause.txt` | `fetch-refused` #2 shape B | FR7 trailing property lines (`errno`, `code`, `syscall`, `address`, `port`) after frame list, FR10 `BucketRuntime` for `node:net` |
| `bare-stack-fetch-cause.txt` | `fetch-refused` #2 shape C | FR9, FR19 zero-frame message-only `.stack` (confirmed real, not just synthetic — `.stack` alone never surfaces `.cause`) |
| `nested-deps-flat.txt` | `full-machine-reverify` #2 | FR10 `PackageName` from segment after last `node_modules/` (`"statuses"`) |
| `zero-stack-trace-limit.txt` | `full-machine-reverify` #4 shape A | FR19 zero-frame crash (`[Error: ...]` bracket form) |
| `esm-runtime-error.txt` | `full-machine-reverify` #5 | FR12 `file://` URI normalization, FR2(a) sync-throw preamble with `file://` path |
| `import-outside-module.txt` | `full-machine-reverify` #6 | FR17 tolerating a non-frame warning preamble before the real error |
| `ts-native-execution.txt` | `full-machine-reverify` #7, direct `node app.ts` block | AC: native TS execution (type-stripping, no transformer frame) → `LanguageTypeScript` |
| `ts-compiled-then-run.txt` | `full-machine-reverify` #7, `tsc`-then-`node` block | AC: compiled TS (all `.js` paths) → `LanguageJavaScript` |
| `ts-tsx-transformer-path.txt` | `round2-altprint-tsxpath` #3 | FR6 `.ts` frames present despite an intervening `tsx` transformer frame path → `LanguageTypeScript` |
| `esbuild-minified-bundle.txt` | `full-machine-reverify` #8 | FR11 bundled/minified single-file output defaults to `BucketOwn` (no `node_modules` segment) |
| `scoped-package-swc-false-caused-by.txt` | `full-machine-reverify` #9 | FR7 plain-text `"Caused by:"` (Rust binding convention) MUST NOT be treated as a `[cause]:` boundary; FR10 scoped package `@swc/core` |
| `aggregate-error-uncaught.txt` | `full-machine-reverify` #10 shape A | Out-of-scope handling: `[errors]:` body → single terminal `ExceptionNode`, `[errors]` dropped, `slog.Warn`, not `ErrUnparseable` |
| `assert-multiline-diff.txt` | `full-machine-reverify` #12 | AC: multi-line diff message (indented brace-like lines) not mistaken for frame lines |

## Synthetic (no real capture exists for this shape)

| Fixture | Exercises |
|---|---|
| `scoped-package-babel.txt` | FR10 scoped package `PackageName` = `"@babel/core"` (two segments after last `node_modules/`) |
| `deep-nested-node-modules.txt` | FR10 `PackageName` = segment after the *last* `node_modules/` occurrence specifically (`"statuses"`, not `"express"` or `"finalhandler"`) — accepted-synthetic per `known-gaps.md` §D |
| `browser-trace-false-positive.txt` | FR4(ii): `Detect()` returns `false` for V8-grammar trace lacking any Node-specific signal — AC bullet |
| `truncated-mid-frame.txt` | FR17: trailing incomplete frame line dropped, `slog.Warn`, still a successful parse — AC bullet |
| `cutoff-cause-chain.txt` | FR18: `[cause]:` opens but never resolves — outer node kept, incomplete cause dropped, `slog.Warn` — AC bullet |
| `unparseable-input.txt` | FR20: `Detect()` matches loosely (has a `node:` frame line) but no coherent header/frame → `Parse()` returns `ErrUnparseable` — AC bullet |

## Not included (confirmed dead ends, per `CAPTURE-FINDINGS.md` §C)

`console.trace`, `JSON.stringify(err)`, and `console.dir(err)` alternate
print-method captures from `round2-altprint-tsxpath/ERRORS.md` are
excluded: `Detect()` never needs to distinguish them from shapes
(a)/(b)/(c) as already covered, and `known-gaps.md`'s 006a→003b row
confirms non-`Error` thrown values (`full-machine-reverify` #11) are out
of this parser's `Detect()` surface entirely.

# Known gaps

Two kinds of intentionally-not-done-yet, tracked in one file so there's a
single place to check instead of searching through spec.md/plan.md files
across every feature. Lives on disk (not just Claude's cross-session
memory) specifically so it survives an account change or a memory reset
-- this file, not memory, is the source of truth for both tables below.

**Checked at every feature kickoff** (`AGENTS.md` workflow step 2, both
tables): does this feature own a deferred criterion, or does it touch an
area with an accepted limitation worth re-examining now that real usage
exists?

## Deferred acceptance criteria

Acceptance criteria written in a completed feature's `spec.md` that
describe behavior owned by a *different*, not-yet-built feature -- not
rejected, just not this feature's job to satisfy.

**When completing any feature listed below as an owner:** also go back and
check the corresponding box in the source feature's `spec.md`, then remove
that row here (or mark it done, whichever keeps this file smallest).

**When starting any feature listed below as an owner:** check whether it
owes a deferred criterion here, and fold it into that feature's own
acceptance criteria during spec interrogation -- these aren't extra work,
they're criteria the feature already needs to satisfy for its own reason,
just written down in a sibling feature's file first.

| Source feature | Criterion | Owner | Status |
|---|---|---|---|
| 001-data-contract | Java `Caused by:` chain parses into `chain[]` with correct `elidedFrameCount` | 005a | pending |
| 001-data-contract | TS/JS `Error.cause` chain parses into `chain[]` with `elidedFrameCount` reflecting Node's real `util.inspect` cause-chain elision (confirmed via real Node 24.18.0 output during 006a interrogation: a run of shared trailing stack lines collapses into a literal `"... N lines matching cause stack trace ..."` line -- this is NOT omitted/0 as originally assumed; that assumption was wrong) | 006a | pending |
| 001-data-contract | Package with no locally resolvable version → `dependencies.locked[pkg].version` omitted, `.note` explains why (end-to-end; struct/JSON shape already covered by 001's own tests) | 005b, 006b | pending |
| 006a-ts-js-parser | Non-`Error` thrown values (`throw "string"`, `throw {...}`) produce no `at ...` frame lines at all, so FR4's `Detect()` signal requirement (>=1 frame-line match) means no `LanguageParser` -- including this one -- will ever claim this input. Common real bug pattern, but the resulting "no parser matched this input" outcome is 003b's to define, not 006a's | 003b | pending |
| 006a-ts-js-parser | A bare `.stack`-only log line with zero frames and none of FR4's four Node-specific signals (confirmed real: `testdata/bare-stack-fetch-cause.txt`, literally `"TypeError: fetch failed"`, nothing else) is genuinely undetectable by any parser -- `Detect()` correctly returns `false`. When 002b/003b build the "no parser matched" user-facing path, consider a more helpful message for this specific shape than a generic failure: suggest pasting the trace directly to an LLM (there's no file/frame info here for this tool's git-blame/snippet/dependency value-add to act on regardless of parser), and mention `Error.stackTraceLimit` being set to `0` as a likely cause if the error's own `.message` pattern looks like a native-binding/fetch-style failure. Not required, just a UX improvement worth remembering since the underlying cause was diagnosed here | 003b, 002b | pending |

## Accepted v1 limitations

Gaps with no owning feature -- deliberately scoped out of v1 rather than
deferred to a specific future feature. Each row should be traceable back
to the source feature's `plan.md`/spec.md for full reasoning; this table
is an index, not a duplicate copy of that reasoning (Article IV's
one-copy discipline, applied to docs instead of code).

**When starting or touching any feature listed below as source:** check
whether real usage has surfaced a reason to revisit -- if so, don't fix it
silently; propose the change and record why, same as any other
constitution-adjacent decision.

| Source feature | Limitation | Why accepted for v1 | Revisit if |
|---|---|---|---|
| 006a-ts-js-parser | Runtime detection recognizes Node.js only for v1; Bun and Deno traces are out of scope (may be misclassified or rejected by `Detect()`) | Deno shares V8 with Node, so is comparatively low-risk to add later. Bun uses JavaScriptCore, not V8 (confirmed via web search during 006a interrogation) -- its crash-dump/console formatting is NOT confirmed to match Node's `util.inspect`-based format, so adding it needs its own real-captured fixtures, not an assumption it matches Node's | Real Bun and/or Deno stack-trace fixtures are captured and their format is confirmed (or found to meaningfully differ from Node's); user demand emerges |
| 006a-ts-js-parser (applies to any future `VersionSourceLocalEnvironment` use, e.g. 005a's JVM detection too) | Runtime-version detection assumes the trace was generated on the same machine currently running `stack-trace-bundler` -- no support for a trace pasted from a different machine, CI run, or container than the one running this binary | Cross-machine version correlation needs its own explicit design (the tool has no way to know the origin machine's version without being told). The narrower residual case -- an active `nvm`/`fnm` version, or a container, that doesn't match what actually produced the trace even on nominally "the same machine" -- is already covered by the existing `Runtime.Note` mechanism (001): the parser's `Note` text says the version is inferred from the local environment, so the user can verify/correct it themselves if it looks wrong. No new mechanism needed for that narrower case | A real need for cross-machine/CI-log analysis emerges |
| 006a-ts-js-parser | Bundled/minified single-file output (e.g. webpack/esbuild `dist/bundle.js` combining own and vendored code, with no `node_modules` path segment left to detect) defaults every frame to `BucketOwn` -- no source-map-aware bucketing to correctly attribute frames back to their original vendored-vs-own source (spec.md FR11) | Source-map parsing/resolution (reading `.map` files, reversing a bundled position back to an original file/line) is a substantial separate capability with no real fixture-driven design done for it yet. Confirmed real via `testdata/esbuild-minified-bundle.txt` that the misclassification risk is real but bounded: it defaults to the more common/safer case (own code), not silently dropped or miscategorized as a dependency | Real usage surfaces frequent bundled-output frames being misattributed as own code when they're actually vendored, or a future feature explicitly adds source-map support |
| 006a-ts-js-parser | `splitAfterLastNodeModules` (bucketing's `node_modules`-segment detection) normalizes backslashes to forward slashes before splitting, specifically to recognize Windows-style `C:\...\node_modules\pkg\...` paths -- added post-review (a code-review tool caught the gap), but unverified against any real Windows-generated trace: every real capture for this feature is from Linux/WSL | `contract.OS` explicitly includes `OSWindows` (this tool is meant to run there, not just parse traces mentioning it), so this is a real in-scope case, not a hypothetical one -- but the fix is a low-risk, narrow separator normalization (not new parsing logic), so it's included now rather than deferred behind a future feature, just flagged as unverified rather than silently presented as fully proven | A real Windows-generated Node.js trace fixture is captured and confirmed to actually use backslash-separated `node_modules` paths as assumed |

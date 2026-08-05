# Spec: Data contract

**Status:** Planned (spec.md + plan.md + tasks.md done; implementation not started)
**Folder:** specs/001-data-contract

## Overview

The canonical shape every renderer (007 Markdown, 008 JSON) and every
language parser (005a Java, 006a TS/JS) reads from and writes to. This
feature defines that shape as Go structs in `internal/contract`, plus a
generated, tested JSON fixture — the single source of truth for what a
"bundle" is, so no two parts of the tool can silently disagree about it.

## User stories

- As a developer pasting a stack trace into `stack-trace-bundler`, I want
  the resulting bundle to have one predictable shape regardless of source
  language, so that I can build reliable expectations around it.
- As a contributor implementing a parser (005a/006a) or renderer (007/008),
  I want one canonical struct to read from and write to, so I never have to
  guess a field's shape or duplicate its definition.
- As an LLM consuming a bundle as context, I want unresolved, unknown, or
  not-applicable data to be marked explicitly and unambiguously — never a
  silent `null` or a zero value standing in for "not applicable" — so I
  don't treat missing information as if it were confirmed.

## Functional requirements

1. The system must define a canonical Go struct (in `internal/contract`)
   covering the full bundle shape: `language`, `os`, `rawInput` +
   `rawInputTruncated`, `fingerprint`, `schemaVersion`, `runtime`, `chain`,
   `codeContexts`, `gitMetadata`, `dependencies`.
2. `language` must be one of `"javascript"`, `"typescript"`, `"java"`.
3. `os` must be populated from Go's `runtime.GOOS` and use Go's own values
   — `"linux"`, `"darwin"`, `"windows"` — not another ecosystem's
   platform-naming convention (e.g. not JS's `"win32"`).
4. `rawInput` must store the original pasted trace verbatim, capped at
   512 KB. When the cap is hit, `rawInputTruncated` must be `true`.
   `rawInputTruncated` must always be present as an explicit boolean
   (not `omitempty`) — a real, always-relevant status, not a
   not-applicable case.
5. `fingerprint` must be computed by hashing, for every exception node in
   `chain` (not only the outermost): the file+method/class identity
   (excluding line numbers) of every `own`-bucket frame in that node, plus
   the single frame where that node's exception originates, regardless of
   that frame's bucket. Frames beyond that one originating frame in a
   `dependency`/`runtime` bucket must be excluded from the hash, so a
   library version bump alone does not change the fingerprint of an
   unchanged bug.
6. `schemaVersion` must start at `"1.0.0"` (semver) and follow: MAJOR for
   any breaking change (field renamed/removed/type changed, or an existing
   enum/status value's meaning changed); MINOR for additive-only changes
   (a new optional field); no bump for implementation changes that don't
   touch the shape.
7. `runtime` must be a nested object (`name`, `version`, `versionSource`,
   `note`) — not flat top-level strings. `versionSource` must be one of
   `"trace"`, `"local-environment"`, `"unknown"`. `"trace"` must only be
   used when the version was actually present in the parsed trace text
   (confirmed case: Node's own uncaught-exception crash output, which
   appends a trailing `Node.js vX.Y.Z` line — this does not fire when the
   exception is caught and logged by user code). Java's `printStackTrace()`
   never includes JVM version, so Java is always `"local-environment"` or
   `"unknown"`. `version` must be omitted entirely (not an empty string)
   when unknown.
8. `chain` must support a linear sequence of exception nodes, outermost to
   root cause (branching chains are out of scope, see below). Each node
   must carry `className`, `message`, `elidedFrameCount` (Java's
   `"... N more"`; omitted/0 for JS/TS, which doesn't elide shared frames
   the same way), and `frames[]`.
9. Each frame must carry: `index`; `filePath` (a normalized absolute
   filesystem path, never a URI — even where the source ecosystem natively
   emits one, e.g. Deno's `file://` URLs, since `codeContexts` needs a real
   path for git blame and snippet extraction); optional `className`
   (absent for e.g. a bare JS function not attached to a class);
   `methodName`; `lineNumber`; optional `columnNumber` (JS/TS only, omitted
   entirely for Java — Java traces never carry column info, so there is no
   real zero value here); `bucket` (`"own"|"dependency"|"runtime"`, fixed
   here even though the classification logic that assigns it lives in
   005a/006a); and optional `packageName` (set only when
   `bucket:"dependency"`; for Java, `group:artifact` format, since artifact
   name alone doesn't disambiguate libraries sharing a name under
   different groups; must match the key format used in
   `dependencies.direct`/`dependencies.locked`).
10. `codeContexts` must exist only for `own`-bucket frames, and each entry
    must carry `frameRef` (`chainIndex`, `frameIndex`), `filePath`,
    `language`, `status`, optional `note`, `snippet` (`startLine`,
    `endLine`, `targetLine`, `code`), and `blame[]` (present only when
    `status:"ok"` — nothing to blame for a file that doesn't exist or
    wasn't extracted). `blame` lives here rather than on `gitMetadata`,
    since `git blame` is inherently file-and-line scoped, not repo-level —
    with more than one `own`-bucket frame in a trace, a single
    undifferentiated `gitMetadata.blame[]` array would have no way to say
    which entry belongs to which file. Each `blame` entry must carry
    `startLine`/`endLine` (the range it covers within that
    `codeContext`'s snippet window — matching how `git blame -L` actually
    groups contiguous same-commit lines, not one entry per line),
    `commitHash`, `author` (from `git blame --porcelain`'s `author`
    field), `commitDate` (pre-formatted ISO 8601, derived once at parse
    time from porcelain's `author-time` unix epoch + `author-tz` — not
    stored as a raw epoch, so both renderers read the same value instead
    of each formatting it independently), and `summary` (first line of
    the commit message).
11. `codeContexts[].status` must be `"not_found"` when the referenced file
    doesn't exist in the current checkout; `"stale"` when that specific
    file has uncommitted local changes (a per-file check, distinct from
    the repo-wide `gitMetadata.uncommittedChanges` flag); `"ok"` otherwise.
12. `gitMetadata` must carry `currentCommit`, `branch`, and
    `uncommittedChanges` (repo-wide boolean) — genuinely global facts
    about repo state, distinct from the per-file `blame` data on
    `codeContexts`.
13. `dependencies` must carry `manifestFile`
    (`"package.json"|"pom.xml"|"build.gradle"`, single manifest per
    bundle — no monorepo/workspace support in v1) and `direct` (declared
    version range per package, as written directly in the manifest, e.g.
    `"^18.2.0"` — always available from parsing the manifest text alone,
    no external resolution step, so no unresolved state is possible
    here). `locked` must map each package to an object, not a bare
    string — `{version, note}` — because resolution requires actually
    invoking mvn/gradle/npm and querying a local cache, which can fail
    per-package rather than all-or-nothing, unlike `direct`. `version`
    must be omitted entirely (not an empty string) when unresolved, per
    the omission rule below; `note` must be present whenever `version` is
    absent, explaining why (e.g. "no local mvn/gradle cache on this
    checkout" — expected on a fresh clone that hasn't been built locally
    yet, not a bug — Article IX, decision 0001). `locked`'s shape must
    stay generic to whichever lockfile/resolution mechanism produced the
    value — which lockfile formats (npm/yarn/pnpm) actually get parsed is
    006b's scope, not this feature's.
14. Any field that doesn't apply to a given language or case must be
    omitted from the JSON entirely (Go `omitempty`) — never `null`, never
    a zero value standing in for "not applicable." Zero/false values are
    reserved for cases where they're a real, meaningful answer (e.g.
    `elidedFrameCount: 0`, or `rawInputTruncated: false`, which is
    intentionally NOT `omitempty` since it's a real, always-relevant
    status rather than a not-applicable case).
15. The contract must be defined exactly once, as Go struct(s) in
    `internal/contract` — no second hand-maintained copy of the shape
    anywhere else in the repo (Article IV). The only permitted derived
    artifact is a generated, tested JSON fixture produced from the struct,
    never hand-written in parallel.

## Non-functional requirements

- The shape must read legibly as plain prose, not only as JSON, since it
  is consumed by both a JSON renderer (008) and a Markdown renderer (007)
  — field names and structures should not be JSON-only idioms that would
  look awkward rendered as Markdown.
- The 512 KB `rawInput` cap is a product constraint (keeping bundles
  clipboard/LLM-context-appropriate), not a technical limit — it exists to
  catch "wrong thing pasted" (e.g. an entire log file), not to constrain
  any realistic stack trace, even at pathological recursion depth.

## Out of scope

- Branching/non-linear chains (`AggregateError`/`Suppressed`) — v1 stays
  linear-only.
- AST-based snippet extraction — v1 stays line-window text extraction.
- Output formats beyond JSON/Markdown (e.g. TOON, XML) — no consumer
  identified for either yet.
- `environment` fields beyond `os` (env vars, CLI args, memory stats,
  network status) — cut for leak risk (bundles get pasted into third-party
  AI chats) and no identified need.
- Monorepo/multi-manifest dependency resolution (`dependencies` supporting
  more than one manifest per bundle).
- `codeContexts.status:"stale"` firing on line-number drift near/past
  current file length alone, with a clean working tree — deferred; v1
  covers only the uncommitted-local-changes case.
- Which lockfile formats (npm/yarn/pnpm) get parsed — that's 006b's scope.
- A hand-maintained Markdown mirror of the contract shape — 007 generates
  Markdown from the struct; nothing hand-written in parallel (Article IV).

## Acceptance criteria

- [ ] Given a Java stack trace with a `Caused by:` chain, when parsed into
      the contract, then `chain` contains one node per exception with the
      correct `elidedFrameCount` for any `"... N more"` line.
- [ ] Given a TS/JS stack trace with an `Error.cause` chain, when parsed,
      then `chain` contains one node per cause, with `elidedFrameCount`
      omitted or `0`.
- [ ] Given a bundle built for a file that has uncommitted local changes,
      when `codeContexts` is built for a frame in that file, then `status`
      is `"stale"`.
- [ ] Given a bundle built for a file not present in the current checkout,
      when `codeContexts` is built for a frame referencing it, then
      `status` is `"not_found"`.
- [ ] Given a pasted trace exceeding 512 KB, when the bundle is built,
      then `rawInput` is truncated to the cap and `rawInputTruncated` is
      `true`.
- [ ] Given a pasted trace under 512 KB, when the bundle is built, then
      `rawInputTruncated` is present and `false` (not omitted).
- [ ] Given two bundles for the same bug that differ only by a dependency
      version bump, when their fingerprints are compared, then they match.
- [ ] Given two bundles for genuinely different bugs, when their
      fingerprints are compared, then they differ.
- [ ] Given the generated JSON fixture, when compared against the Go
      struct's marshaled output, then they match exactly (fixture is
      produced from the struct, never hand-written).
- [ ] Given a package with no locally resolvable version (no local
      mvn/gradle/npm cache for it), when `dependencies.locked` is built
      for that package, then its `version` is omitted and `note` explains
      why.

## Open questions

None — all resolved during interrogation. Full field-by-field reasoning is
recorded in `specs/001-data-contract/plan.md`.

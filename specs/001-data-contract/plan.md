# Plan: Data contract

Derived from `spec.md`. Must be consistent with `memory/constitution.md`.

## Architecture / approach

All types live in a single package, `internal/contract`, as the one
canonical definition (Article IV) -- no separate schema file, no
hand-maintained mirror anywhere else in the repo.

- `types.go` -- every struct in the Data model above, as Go structs with
  `json` tags. `omitempty` applied per the cross-cutting omission rule
  (not-applicable fields omitted), deliberately withheld on
  `rawInputTruncated` (always-relevant, real boolean).
- `fingerprint.go` -- `ComputeFingerprint` lives here rather than in a
  parser or the pipeline-wiring feature (002b), because it's a pure
  function of already-defined contract types (`[]ExceptionNode` in,
  `string` out) and its exact scope (which frames count) is itself a
  contract-level guarantee that needs to be pinned down and tested
  alongside the types it operates on.
- `rawinput.go` -- the 512 KB truncation helper, same reasoning: a pure
  function of contract-level constraints, not parser-specific logic.
- `testdata/example_java.json` / `testdata/example_ts.json` -- generated
  from two fully-populated example `Bundle`s (one Java-shaped, one
  TS/JS-shaped -- a single invocation only ever produces one language's
  bundle, so one fixture mixing both would misrepresent any real code
  path), never hand-written (Article IV).

005a/006a (parsers) and 004 (own-code context extraction) populate these
structs; they do not define or modify the shape. 007/008 (renderers) read
them; they do not define the shape either.

## Stack & versions

No new external dependencies for this feature specifically:

- `encoding/json` (stdlib) for marshaling -- no third-party JSON library
  needed at this layer.
- `crypto/sha256` (stdlib) for the fingerprint hash.

Consistent with the project's general minimal-dependency posture; parsers
and dependency-resolution features (005a/005b/006a/006b) will have their
own stack needs, out of scope here.

## Data model

> Working checkpoint, populated incrementally as fields are resolved during
> spec interrogation, so decisions aren't lost mid-session. Superseded by the
> actual Go struct in `internal/contract` once implemented. Supersedes
> `data-contract.md`, which is deleted once this section is complete.

### Resolved

```jsonc
{
  // CROSS-CUTTING RULE, applies to every field below: if a field doesn't
  // apply to a given language/case, it is omitted from the JSON entirely
  // (Go `omitempty`) -- never `null`, never a zero value standing in for
  // "not applicable". Zero values are reserved for cases where zero is a
  // real, meaningful answer. Ambiguity here is exactly what Article VI
  // ("never guess silently") is meant to prevent, just at the
  // serialization layer instead of the data-collection layer.

  // Starts at "1.0.0" (full semver, not the draft's "1.0" -- leaves a
  // patch slot even if unused now). Bump triggers:
  //   MAJOR: any breaking change -- a field renamed, removed, or its type
  //     changed; an existing enum/status value's meaning changed.
  //     (e.g. runtime going from two flat strings to a nested object,
  //     earlier in this doc, would have been a MAJOR-class change.)
  //   MINOR: additive only -- a new optional field added.
  //     (e.g. columnNumber's omission rule, or filePath's normalization
  //     rule, would be MINOR-class additions.)
  //   No bump: implementation changes that don't touch the shape at all.
  "schemaVersion": "string, semver",

  // Enum is closed for v1 -- matches the two parser features in INDEX.md
  // (005a Java, 006a TS/JS). No other languages in scope.
  "language": "javascript" | "typescript" | "java",

  // Populated via Go's runtime.GOOS, so it MUST use Go's own constant
  // values, not JS's process.platform convention. Go's values are always
  // lowercase: linux, darwin, windows (verified against Go docs -- NOT
  // 'win32', which is what the original draft had).
  "os": "linux" | "darwin" | "windows",

  // Verbatim pasted trace -- parse fallback only, not the primary payload
  // (chain[] is that). Capped at 512 KB: generous enough that no real
  // stack trace, even pathological recursion depth, gets truncated,
  // while still catching "wrong thing pasted" (e.g. an entire log file
  // pasted by mistake).
  "rawInput": "string",
  // Always present, true or false -- NOT omitempty, unlike most fields in
  // this doc. The cross-cutting omission rule (above) is for "not
  // applicable" cases; this is a real, always-relevant status (was the
  // cap hit or not), so an explicit false is the correct signal here,
  // not an omission that could be misread as "wasn't checked."
  "rawInputTruncated": "boolean",

  // Computed over EVERY exception node in the chain (not just the
  // outermost) -- two exceptions sharing an outer wrapper but differing
  // in an inner cause are different bugs and must not collide. For each
  // node: hashes the file+method/class identity (excluding line numbers,
  // which are too volatile) of every own-bucket frame in that node, plus
  // the single frame where that node's exception actually originates,
  // regardless of which bucket that originating frame belongs to.
  // Deliberately excludes dependency/runtime frames beyond that one
  // originating frame, so a library version bump doesn't change the
  // fingerprint of a bug that hasn't actually changed (Article IX).
  // Algorithm: SHA-256, truncated to first 16 hex chars -- an
  // implementation detail, not a functional requirement, so no
  // ambiguity worth raising as a question.
  "fingerprint": "string, hex",

  "runtime": {
    // NOT 'java': the source language is already captured separately by
    // the top-level `language` field ('java'/'javascript'/'typescript');
    // `runtime` is the execution engine, which is orthogonal (TS can run
    // on node, bun, or deno alike).
    "name": "string, e.g. 'node' | 'bun' | 'deno' | 'jvm'",
    // Omitted entirely (omitempty) if unknown, per Article VI and the
    // cross-cutting omission rule above -- not an empty string.
    "version": "string, e.g. '20.11.0'",
    // 'trace' applies narrowly: confirmed only for Node's own
    // uncaught-exception crash output, which appends a trailing
    // "Node.js vX.Y.Z" line. Does NOT fire if the exception is caught
    // and logged by user code -- the common case. Java's
    // printStackTrace() never includes JVM version, so Java is always
    // 'local-environment' or 'unknown'.
    "versionSource": "enum: 'trace' | 'local-environment' | 'unknown'",
    // Explains a non-'trace' versionSource, e.g. "inferred from local
    // `node -v`; may not match the environment that produced this trace"
    "note": "string, optional"
  },

  "chain": [
    {
      "className": "string, e.g. 'java.lang.RuntimeException' or 'TypeError'",
      "message": "string",
      // Java's "... N more" line, which elides frames this node shares
      // with its enclosing exception. 0/omit for JS/TS -- V8 doesn't
      // elide shared frames the same way.
      "elidedFrameCount": "number",
      "frames": [
        {
          "index": "number",
          // Normalized absolute filesystem path, not a URI -- even though
          // Deno traces natively use `file://` URLs, codeContexts needs a
          // real path for git blame and snippet extraction, so the URL
          // form would just get parsed back into a path everywhere it's
          // consumed. Named filePath, not fileUrl, so the name matches
          // what it actually holds.
          "filePath": "string",
          // absent for e.g. a bare JS function not attached to a class
          "className": "string, optional",
          "methodName": "string",
          "lineNumber": "number",
          // Omitted entirely for Java (omitempty) -- Java stack traces
          // never carry column info, so there is no real zero value here,
          // per the cross-cutting omission rule above.
          "columnNumber": "number, JS/TS only",
          // Values fixed here; the logic that decides which bucket a
          // frame falls into belongs to 005a/006a, not this feature.
          "bucket": "own" | "dependency" | "runtime",
          // Set only when bucket:"dependency". Same field, both
          // languages -- not split per-language, since it's one string
          // whose convention differs by ecosystem, same pattern as
          // elidedFrameCount/columnNumber above. For Java: `group:artifact`
          // (Maven/Gradle coordinate), since artifact name alone doesn't
          // disambiguate libraries that share a name under different
          // groups, and this value must match whatever key format
          // dependencies.direct/.locked use for Java entries.
          "packageName": "string, optional"
        }
      ]
    }
  ],

  "codeContexts": [
    {
      "frameRef": { "chainIndex": "number", "frameIndex": "number" },
      // Same normalized absolute path as frames[].filePath above.
      "filePath": "string",
      "language": "typescript" | "javascript" | "java",
      // "not_found": file doesn't exist in the current checkout.
      // "stale": that specific file has uncommitted local changes (a
      //   per-file check, not the repo-wide gitMetadata.uncommittedChanges
      //   flag) -- the snippet may not reflect what actually ran.
      // "ok": neither of the above.
      "status": "ok" | "not_found" | "stale",
      // e.g. "file not found in current checkout -- trace may be from a
      // different commit"
      "note": "string, optional",
      "snippet": { "startLine": "number", "endLine": "number", "targetLine": "number", "code": "string" },
      // Moved here from gitMetadata: git blame is inherently file+line
      // scoped, not repo-level, so it belongs with the filePath/snippet
      // this codeContext already has, not in a single undifferentiated
      // array with no way to tell which file an entry belongs to.
      // Matches INDEX.md's own feature 004 description, which already
      // lists "git blame -L" separately from "repo-level git metadata".
      // Present only when status:"ok" -- nothing to blame for a file
      // that doesn't exist or wasn't extracted.
      "blame": [
        {
          // The range this entry covers WITHIN this codeContext's
          // snippet window -- matches how `git blame -L` actually groups
          // contiguous same-commit lines, not one entry per line.
          "startLine": "number",
          "endLine": "number",
          "commitHash": "string, full 40-char SHA-1",
          // from `git blame --porcelain`'s "author" field
          "author": "string",
          // Derived once at parse time from porcelain's author-time (unix
          // epoch) + author-tz. Stored pre-formatted, not as raw epoch,
          // so both renderers (007 Markdown, 008 JSON) read the same
          // value instead of each formatting it independently
          // (Article V).
          "commitDate": "string, ISO 8601",
          // first line of the commit message
          "summary": "string"
        }
      ]
    }
  ],

  // Optional at the bundle level as of schemaVersion "2.0.0" (bumped
  // from "1.0.0" by 004-own-code-context-extraction): Bundle.GitMetadata
  // is *GitMetadata (json:"gitMetadata,omitempty"), omitted entirely
  // (not null) when no git repository is found. When present, all three
  // fields below are always populated.
  "gitMetadata": {
    "currentCommit": "string",
    "branch": "string",
    // Repo-wide flag. Distinct from codeContexts[].status:"stale", which
    // is a per-file check -- this one says nothing about which files.
    "uncommittedChanges": "boolean"
  },

  "dependencies": {
    // Single manifest per bundle, no monorepo/workspace support in v1 --
    // an accepted v1 simplification (data-contract.md), not a hidden gap.
    // No concrete multi-manifest need identified yet, so building array
    // support now would be exactly the speculative generality Article
    // VIII says not to do.
    "manifestFile": "package.json" | "pom.xml" | "build.gradle",
    // Declared version range as written in the manifest (e.g. "^18.2.0"),
    // NOT the resolved version -- that's `locked`, below. Always
    // available from parsing the manifest text alone, no external
    // resolution step, so no unresolved state is possible here (unlike
    // `locked`).
    "direct": { "packageName": "string, version range as declared" },
    // Value is an object, not a bare string, because resolution requires
    // actually invoking mvn/gradle/npm and querying a local cache, which
    // can fail per-package rather than all-or-nothing -- same pattern as
    // runtime.versionSource/.note and codeContexts.status/.note. Shape
    // stays generic to whichever lockfile/resolution mechanism produced
    // the value (npm's package-lock.json, yarn.lock, pnpm-lock.yaml,
    // Maven/Gradle's resolved graph) -- which lockfile formats 006b
    // actually parses is that feature's scope, not this one's.
    "locked": {
      "packageName": {
        // Omitted entirely (omitempty) when unresolved, per the
        // cross-cutting omission rule -- not an empty string.
        "version": "string, resolved version, optional",
        // Present only when version is absent. E.g. "no local mvn/gradle
        // cache on this checkout (Article IX, decision 0001) -- expected
        // on a fresh clone that hasn't been built locally yet, not a
        // bug."
        "note": "string, optional"
      }
    }
  }
}
```

### Deferred -- explicitly out of v1

- `codeContexts.status:"stale"` firing on line-number drift near/past current
  file length, even with a clean working tree (option (b) from the stale
  discussion). Uncommitted-changes detection (option (a)) covers v1.
- `dependencies` supporting multiple manifests for monorepos/workspaces.

All Data model items resolved -- nothing left open.

## File / module layout

```
internal/contract/
  types.go            # all struct definitions from the Data model above
  types_test.go        # JSON round-trip tests, omitempty behavior tests
  fingerprint.go        # ComputeFingerprint(chain []ExceptionNode) string
  fingerprint_test.go
  rawinput.go           # TruncateRawInput(s string) (string, bool)
  rawinput_test.go
  testdata/
    example_java.json    # generated fixture, produced FROM the struct (realistic Java-shaped bundle)
    example_ts.json      # generated fixture, produced FROM the struct (realistic TS/JS-shaped bundle)
```

## API / contracts (if applicable)

Public Go API surface of the package (no HTTP/RPC surface -- this is a
library consumed in-process by other features):

- `contract.Bundle` -- top-level struct, the full shape from the Data
  model above.
- `contract.SchemaVersion` -- exported constant, `"1.0.0"`.
- `contract.ComputeFingerprint(chain []contract.ExceptionNode) string`
- `contract.TruncateRawInput(s string) (out string, truncated bool)`

## Testing strategy

- **Marshal/unmarshal round-trips**, per struct, asserting `omitempty`
  behavior specifically: a Java frame with no `columnNumber` must not have
  that key in the output JSON at all; a JS/TS frame with it set must have
  it present. `rawInputTruncated` must always be present, `true` or
  `false`, never absent.
- **Fingerprint table tests**, covering spec.md's acceptance criteria
  directly: same bug + dependency version bump -> same fingerprint;
  genuinely different bugs -> different fingerprints; a chain with an
  identical outer wrapper but a different inner cause -> different
  fingerprints (proves every node is hashed, not just the outermost).
- **Golden-file tests** -- re-marshal each example `Bundle` (one
  realistically Java-shaped, one realistically TS/JS-shaped -- a single
  invocation only ever runs one language's parser, so one bundle can't
  realistically mix both shapes) and compare byte-for-byte against
  `testdata/example_java.json` / `testdata/example_ts.json`, so neither
  fixture can silently drift from the struct it's supposed to be
  generated from.
- **Truncation boundary tests** -- exactly at 512 KB, one byte under, one
  byte over.

## Risks & open decisions

- The whole shape is still unverified against real, messy, copy-pasted
  stack traces (truncated traces, minified/obfuscated JS, unusual
  `Caused by` formatting) -- `data-contract.md`'s original caveat, and
  still true of this spec. Expect this to surface a `schemaVersion` MINOR
  or MAJOR bump once 005a/006a are actually implemented against real
  input; that's an accepted v1 risk, not a blocker to writing the struct.
- The 512 KB `rawInput` cap is a reasoned estimate (real stack traces,
  even at pathological recursion depth, land well under it), not
  empirically validated against a corpus of real traces yet.
- Deferred, not resolved: `codeContexts.status:"stale"` firing on
  line-number drift alone (see spec.md Out of scope), and
  monorepo/multi-manifest `dependencies` support.
- On a fresh checkout with no local mvn/gradle cache, expect most or all
  `dependencies.locked` entries to have `version` omitted and `note` set
  (Article IX, decision 0001) -- this is now a first-class, tested part of
  the contract (see spec.md acceptance criteria), not just a documented
  risk, so it shouldn't surprise anyone reading a bundle from a
  fresh-clone environment.

## Alternatives considered

- **Base contract + per-language inheriting subtypes**, to keep
  language-specific fields (like Java's `group:artifact` `packageName`
  convention) off languages that don't need them. Rejected: not an actual
  ISP violation (both languages use the same field, same type, just a
  different string convention -- nothing forced on a client that doesn't
  use it), and Go has no real inheritance, so this becomes struct
  embedding + an interface, forcing both renderers (007/008) to
  type-switch before reading a bundle. A single struct with a
  consistently-applied `omitempty` rule solves the same ambiguity problem
  with no new types.
- **`commitDate` as raw Unix epoch**, left for each renderer to format.
  Rejected: would duplicate date-formatting logic in both 007 and 008,
  against Article V ("shared logic lives once"); a pre-formatted ISO 8601
  string computed once at parse time is correct at the source instead.
- **A bare `runtimeVersionInferred` bool** instead of a `versionSource`
  enum. Rejected: collapses three real provenance states (found in trace /
  inferred locally / unknown) into two, and separates the confidence flag
  from the value it modifies under a different key name -- exactly the
  cross-field ambiguity Article VI is meant to prevent.
- **`fileUrl` (a real URI)** instead of `filePath` (normalized absolute
  path). Rejected: `codeContexts` needs a real filesystem path for git
  blame and snippet extraction regardless, so a URI would just get parsed
  back into a path everywhere it's consumed.

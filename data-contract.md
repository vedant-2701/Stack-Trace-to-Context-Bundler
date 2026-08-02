> **STATUS: DRAFT — NOT FINAL. NOT A SPEC.**
> This is a rough capture of what was discussed and agreed in the project
> kickoff chat. It has not been re-verified, has no spec.md/plan.md/tasks.md
> yet, and nothing here should be implemented as-is. When feature 001 is
> actually planned, re-evaluate every field against real-world traces before
> writing `internal/contract/types.go` or `spec.md`. This file should
> probably be deleted once that happens.

# Data contract — draft reference

The shape every renderer (Markdown, JSON) and every language parser (Java,
TypeScript/JS) produces/consumes. Discussed over several rounds; this is the
last corrected version from that discussion, not a verified final design.

## JSON shape (draft)

```jsonc
{
  "schemaVersion": "1.0",
  "language": "javascript" | "typescript" | "java",
  "runtime": "node" | "bun" | "deno" | "jvm",
  "runtimeVersion": "string",
  "os": "linux" | "darwin" | "win32",
  "fingerprint": "string",              // hash of chain[].frames minus line numbers — dedup, no storage needed
  "rawInput": "string",                 // original pasted trace, verbatim — parse fallback

  "chain": [                            // outermost/reported → root cause. v1 = LINEAR ONLY, see deferred list.
    {
      "className": "string",            // e.g. "java.lang.RuntimeException" or "TypeError"
      "message": "string",
      "elidedFrameCount": 0,            // Java's "... N more"; 0/omit for JS
      "frames": [
        {
          "index": 0,
          "fileUrl": "string",
          "className": "string?",
          "methodName": "string",
          "lineNumber": 0,
          "columnNumber": 0,            // JS/TS only
          "bucket": "own" | "dependency" | "runtime",
          "packageName": "string?"      // set only when bucket:"dependency" — key into dependencies.*
        }
      ]
    }
  ],

  "codeContexts": [                     // only for bucket:"own" frames
    {
      "frameRef": { "chainIndex": 0, "frameIndex": 0 },  // unambiguous link back to the frame
      "fileUrl": "string",
      "language": "typescript" | "javascript" | "java",
      "status": "ok" | "not_found" | "stale",
      "note": "string?",                // e.g. "file not found in current checkout — trace may be from a different commit"
      "snippet": { "startLine": 0, "endLine": 0, "targetLine": 0, "code": "string" }
    }
  ],

  "gitMetadata": {
    "currentCommit": "string",
    "branch": "string",
    "uncommittedChanges": true,
    "blame": [                          // ranges, not one row per line — matches how `git blame -L` actually groups
      { "fileUrl": "string", "startLine": 0, "endLine": 0, "author": "string", "commitHash": "string", "commitDate": "string", "summary": "string" }
    ]
  },

  "dependencies": {                     // NEEDS RE-EVALUATION: v1 assumes a single manifest at repo root;
    "manifestFile": "package.json" | "pom.xml" | "build.gradle",   // multi-module/workspace resolution not designed
    "direct": { "packageName": "versionString" },
    "locked": { "packageName": "resolvedVersionString" }
  }
}
```

## Markdown shape (draft)

```
# Stack Trace Context Bundle

Runtime: <language + version>              e.g. Java 21 / Node v20.11.0
Error:      <ExceptionType>: <message>     <- outermost / reported error
Root cause: <ExceptionType>: <message>     <- deepest cause (omitted if not chained)
Chain: N level(s)

## [1] <file>:<line> — <function()>   [own code | dependency | runtime]
  <snippet + blame, if own code>

## [2] ...

## Dependency Versions
  <package> <resolved version>

## Notes
  <N runtime/stdlib frames omitted per run>
```

## Deferred — explicitly out of v1, revisit in a future version

- **Branching chains** — `AggregateError` / `Suppressed`, non-linear
  "multiple failing sites" instead of one root cause. v1 stays linear-only.
- **Literal AST-based snippet extraction** — v1 stays line-window text
  extraction (`snippet.startLine`/`endLine`/`code`), not real syntax-tree
  parsing.
- **Output formats beyond JSON/MD** — TOON, XML. No consumer identified for
  either yet; revisit if one shows up.
- **`environment` beyond `os`** — no env vars, command-line args, memory
  stats, or network status. Cut for leak-risk (output gets pasted into
  third-party AI chats) and because it wasn't in the original mechanism.

## Other open items, not yet re-verified

- `dependencies.manifestFile` assumes one manifest — doesn't hold for a
  monorepo/workspace. Accepted as a v1 simplification, not a hidden gap.
- Whole shape needs checking against real, messy, copy-pasted stack traces
  (truncated traces, obfuscated/minified JS, unusual `Caused by` formatting)
  before it's trusted as final.
- Java entries in `dependencies.direct`/`dependencies.locked` come from an
  offline-only `mvn`/`gradle` call (constitution Article IX, decision 0001).
  On a fresh checkout that hasn't been built locally yet, most or all of
  these will come back as an unresolved version + note rather than a real
  value, since nothing's in the local cache. This is expected, not a bug —
  but it means the bundle is meaningfully less complete on a brand-new
  clone than on one the user has already built at least once.

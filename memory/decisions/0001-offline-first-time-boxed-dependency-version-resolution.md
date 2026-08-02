
# 0001 — Offline-first, time-boxed dependency version resolution

**Status:** Accepted
**Date:** 2026-08-02
**Depends on:** none

## Context

Article VII commits to shelling out to `mvn`/`gradle` rather than embedding a
JVM or a resolver library. That's the only place these two tools get
invoked in the whole project: resolving concrete versions for
`dependency`-bucket frames in the Java parser (feature 005b). TypeScript/JS
dependency resolution (006b) parses `package.json` and the lockfile
directly — it never shells out to `npm`/`yarn`/`pnpm` — so it's out of scope
for this decision.

Both tools can, in the worst case, sit waiting on a remote repository — a
slow mirror, a corporate proxy, a network outage — for far longer than is
acceptable in a CLI whose entire value proposition is producing a
paste-ready bundle quickly. The constitution didn't originally say what
bounds that call, which is a gap given Article VI's promise to never guess
and never go silent about something it couldn't determine.

## Decision

`mvn`/`gradle` are invoked in offline mode only (`-o` / `--offline`),
wrapped in a hard timeout regardless.

Verified directly against both tools' own documentation and real-world bug
reports, not assumed: both fail immediately with a clear, catchable error
when a required artifact isn't in the local cache — offline mode does not
hang waiting on the network, it fails fast. That converts the primary hang
risk into an ordinary, expected error path rather than something we need a
timeout to catch.

A timeout still applies on top of that, because offline mode does not
remove JVM/daemon startup cost, and for Gradle specifically does not remove
full-project configuration-phase evaluation (which scales with project
size, independent of network or cache state) — both are real latency
sources unrelated to the network.

On timeout, or on an offline-mode resolution failure — including a known
Gradle limitation where dynamic/range versions (e.g. `1.+`) can fail to
resolve offline even when a similar version is cached, because the
version-listing metadata itself wasn't cached (gradle/gradle#10934) — the
tool does not retry online and does not guess. It marks that dependency's
version with an explicit unresolved note (Article VI) and continues
producing the rest of the bundle.

## Alternatives considered

- **Online resolution with a generous timeout.** Rejected for v1: reopens
  the exact open-ended-hang risk this decision exists to close, and a
  timeout generous enough to tolerate a slow-but-working network is still
  far too slow for a paste-and-go CLI.
- **No timeout at all, rely on offline mode alone.** Rejected: offline mode
  bounds network latency to zero but does not bound JVM/daemon-startup or
  Gradle configuration-phase latency, both confirmed to be real and
  independent of caching.
- **Embed a dependency-resolution library instead of shelling out.**
  Out of scope for this decision — Article VII already settled
  shell-out-vs-embed for unrelated reasons (binary size, cross-compilation,
  not inheriting a dependency's release cycle), and an embedded resolver
  would face the same local-cache-vs-network tradeoff regardless.

## Consequences

- A fresh, never-locally-built checkout will show more "unresolved" version
  notes than a warm one, since nothing's cached yet. This is expected and
  honest per Article VI, not a bug — but it does mean the tool is
  meaningfully less complete on a brand-new clone until the user has built
  it at least once locally. Worth knowing going in.
- No online fallback exists in v1. If real usage shows offline-only
  resolution is too incomplete to be useful, an opt-in online-resolution
  flag can be added later (Article VIII) — deliberately not built
  speculatively now.
- The exact timeout duration is not fixed by this record. It should be set
  empirically — time a cold `mvn -o dependency:tree` / `gradle --offline
  dependencies` invocation — when 005b is actually planned, not guessed
  here.

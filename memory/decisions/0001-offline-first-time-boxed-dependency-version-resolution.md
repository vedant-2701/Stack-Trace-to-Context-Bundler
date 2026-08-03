
# 0001 — Offline-first, time-boxed dependency version resolution

**Status:** Accepted (timeout value provisional — see Round 1 addendum below)
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
hang waiting on the network for ordinary dependency resolution, it fails
fast. That converts the primary hang risk into an ordinary, expected error
path rather than something we need a timeout to catch — for that specific
case.

Offline mode does not cover every network path, though, and the original
draft of this decision overstated that it did. The Gradle wrapper
downloads its declared distribution *before* Gradle itself starts, which
is outside `--offline`'s reach entirely — confirmed both from Gradle's own
docs ("downloading it beforehand if necessary") and empirically in Round 1
below (scenario G2). On a machine without that distribution already
cached, the hard timeout is the *only* thing standing between this call
and a real network wait — not the offline flag. This is why the timeout
stays mandatory regardless of offline mode, not merely as a backstop for
JVM/configuration-phase cost.

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

## Round 1 addendum (Linux only) — see `0001-benchmark-protocol.md`

One full pass of the benchmark protocol was run on Linux
(Ubuntu 24.04.4, OpenJDK 21, Maven 3.8.7, Gradle 8.10.2). Full results and
methodology live in `memory/decisions/0001-benchmark-protocol.md`; summary:

**Confirmed:** on Linux, with a genuinely absent network path, both tools
fail in well under 3 seconds across every scenario tested (missing
plugin, missing dependency, missing wrapper distribution, BOM-managed
versions, dynamic/range versions). Worst clean measurement across the
whole pass: 13.1s (cold Gradle, first-run plugin resolution). Scenario G2
confirmed the wrapper-bootstrap gap described above is real, not just
theoretical: without a network block, the wrapper attempts a live DNS
lookup to `services.gradle.org` before `--offline` is ever parsed.

**Provisional timeout: 30s.** Chosen with meaningful margin above the one
clean "things are legitimately starting up" number available (13.1s),
pending Round 2. Not a final value — do not treat as calibrated until
Round 2 closes the gaps below.

**Still open, not yet measured:**
- *Silent-network condition.* Round 1's network-blocked scenarios
  (`unshare -n`, an unreachable proxy) both produce a fast, clean
  rejection — the best case for a hung connection. They don't yet cover a
  firewall that DROPs packets or a dead-but-configured DNS server, which
  causes TCP retransmission/resolver timeouts on the order of tens of
  seconds to minutes. This is the actual case the timeout exists to bound
  and it hasn't been tested yet.
- *Module-count scaling.* The synthetic 10- and 30-module Gradle runs
  intended to show whether configuration-phase cost grows with project
  size weren't cleanly captured (noise from an unrelated toolchain
  quirk). Only the 1-module number is clean. This is the largest
  remaining risk to whatever number gets locked in, since a real project
  is likely larger than the 1-module case.
- *Non-Linux platforms.* Windows and macOS are untested. Windows JVM
  cold-start is known to run slower under real-time antivirus scanning.
- *Corrupted-cache handling (G6).* Whether Gradle attempts a network
  repair on a corrupted cached artifact under `--offline` is still
  unconfirmed — the Round 1 attempt was contaminated by the same
  toolchain quirk.

Revisit this addendum, and Article IX's provisional 30s, once Round 2
closes these gaps — before feature 005b ships, not necessarily before it
starts.

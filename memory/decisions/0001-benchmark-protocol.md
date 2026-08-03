
# Supporting material for decision 0001 — Java build-tool offline resolution & timeout benchmark protocol

This is not itself an ADR (not listed in `memory/decisions/INDEX.md`). It's the
test protocol referenced by
`0001-offline-first-time-boxed-dependency-version-resolution.md`, to be run
for real once feature 005b (Java dependency resolution) is actually being
built — not before. The one preliminary pass already run (via an external
sandbox, not this protocol) confirmed the core assumption directionally but
left real gaps; this protocol exists to close those gaps with real evidence
rather than another single-sample run.

**Goal:** get real timing + failure-mode data across scenarios that matter
for setting the hard timeout in constitution Article IX. Not a goal: landing
on a final number from this pass alone — calibrate that against the actual
project's real Java codebase during 005b.

## Safety rules for whoever runs this — non-negotiable

- **Every command must be wrapped in an external kill switch.** `timeout
  <N>s <command>` on Linux/macOS; on Windows, a PowerShell job with
  `Wait-Job -Timeout`. Use 60s for offline-mode scenarios (should never
  legitimately need more), 180s for the online/network-blocked baseline
  scenarios (M6/G7 below), which exist specifically to probe how long an
  uncontained hang could run. Without this, a genuine hang freezes the test
  session, not just the tool under test.
- Time every run with `time` (bash) or `Measure-Command` (PowerShell), not
  a stopwatch.
- **Run each scenario 3 times minimum.** Report min/median/max. A single
  sample (as in the earlier preliminary pass) can't distinguish signal from
  noise.
- Record per run: exact command, exit code, last ~20 lines of
  stdout+stderr, `mvn -version` / `gradle --version` / `java -version` /
  `uname -a` (or `systeminfo` on Windows).
- Reset state precisely between scenarios per that scenario's setup step —
  don't let one scenario's warm cache leak into the next.

## Environment matrix

Run on whatever's available now. Explicitly report which OS was skipped —
don't silently assume Linux numbers generalize.

| OS | Priority |
|----|----------|
| Linux (dev machine / WSL) | required |
| macOS | if available |
| Windows (native, not WSL) | if available — JVM cold-start is known to run slower under Defender real-time scanning of new process launches; explicitly note whether Defender/AV was on or off during the run |

## Test project(s)

Don't use a trivial single-module hello-world for the Gradle scenarios —
configuration-phase cost scales with subproject/plugin count, so a trivial
project understates real cost. If no real multi-module project is on hand,
generate synthetic ones at three sizes: 1 module, 10 modules, 30 modules,
each with 2–3 common plugins applied (e.g. `java`, `checkstyle`, a shadow/fat-jar
plugin). Run the cold-cache Gradle scenario (G1) at all three sizes to get a
scaling curve, not just one data point.

## Maven scenarios

- **M1 — never-run machine.** `rm -rf ~/.m2/repository`, then
  `timeout 60 mvn -o dependency:tree`. Expect a plugin-prefix failure
  ("No plugin found for prefix 'dependency'"). This confirms that failure
  mode is about the *dependency plugin itself* not being cached, distinct
  from the *target dependency* not being cached — don't conflate the two
  when 005b handles error messages.
- **M2 — tool warm, target cold.** Maven previously used successfully for
  some *other* project (so the dependency plugin is cached), but this
  project's specific dependency versions aren't. Isolates "target
  uncached" from "tool uncached."
- **M3 — fully warm.** One prior successful online `mvn dependency:tree`
  run, then repeat offline. Baseline success case.
- **M4 — multi-module reactor, warm.** Does Maven's per-module cost scale
  the way Gradle's configuration phase does, or is it negligible?
- **M5 — BOM/property-managed version.** A dependency whose version comes
  from a parent POM or BOM, not a literal string in this module's POM.
  Does offline resolution behave differently than a directly-pinned
  version?
- **M6 — baseline-only: online mode, network blackholed.** No `-o`. Point
  at an unreachable proxy (`-DproxyHost=10.255.255.1 -DproxyPort=1`) or
  block egress at the OS/firewall level. Wrap in `timeout 180`. Purpose:
  document the actual worst case offline-first is protecting against —
  how long it takes to fail, and how.

## Gradle scenarios

- **G1 — wrapper present, dependency cache cleared, offline.**
  `rm -rf ~/.gradle/caches/`, then `timeout 60 ./gradlew --offline
  :app:dependencies`. Repeat ≥3× per project size from the test-project
  section above.
- **G2 — wrapper distribution ALSO absent, network fully blocked.**
  `rm -rf ~/.gradle/wrapper/dists`, block egress at the OS/firewall level
  (not just via the `--offline` flag — the point is to test what happens
  when the flag *can't* help), then `timeout 60 ./gradlew --offline
  :app:dependencies`. **This is the untested gap from the preliminary
  pass.** The wrapper script downloads its declared Gradle distribution
  before Gradle itself starts, which is outside `--offline`'s reach.
  Expected one of two outcomes — record which: a fast, clear failure
  (good), or a hang bounded only by the external `timeout` (confirms the
  hard timeout, not offline mode, is what actually protects this path).
- **G3 — fully warm.** Daemon already up, caches warm. Baseline.
- **G4 — daemon stopped, cache warm.** `./gradlew --stop` immediately
  before running. Isolates daemon-startup cost alone from
  configuration-phase cost (G1/G3 conflate the two).
- **G5 — dynamic/range version.** A dependency declared as `1.+` or
  `[1.0,2.0)`, offline, with everything else warm except that range's
  version-listing metadata. Reproduces the known Gradle limitation
  (gradle/gradle#10934) where offline resolution can fail even with a
  similar version cached. Record the actual error text.
- **G6 — corrupted cached artifact.** `truncate -s 100 <a cached jar
  under ~/.gradle/caches/modules-2>`, then run offline. Does Gradle detect
  corruption and fail fast, or attempt a network repair — which would
  silently defeat offline mode?
- **G7 — baseline-only: online mode, network blackholed.** Same purpose
  as M6, for Gradle. `timeout 180`.

## Deliberately out of scope for this pass

JAVA_HOME/JVM-version misconfiguration, Gradle-plugin-portal-specific
failure modes beyond G5/G6, and a full Defender-on-vs-off A/B (only worth
it if Windows is actually accessible for this pass). Pick these up during
005b if real usage shows they matter — don't burn this pass on them
speculatively (Article VIII).

## Reporting format

One row per scenario, matching the shape already used once:

| Scenario | Real time (min / median / max of 3) | Exit code | Outcome | Did it touch the network? (yes/no — state how you verified this, e.g. "egress hard-blocked at OS level," not just "flag says offline") |
|---|---|---|---|---|

Plus, for G2 and M6/G7 specifically: a one-line note on whether the
external `timeout` was what actually stopped the command, or whether it
exited cleanly on its own before the timeout fired.

---

## Round 1 results (Linux only) — run 2026-08-03

**Platform:** Ubuntu 24.04.4 LTS, kernel 6.12.8+, x86_64.
**Versions:** OpenJDK 21.0.10/21.0.11, Apache Maven 3.8.7, Gradle 8.10.2.
No Windows or macOS coverage this round.

### Confirmed

- With a genuinely absent network path (empty network namespace or an
  actively-rejecting proxy), both tools fail in well under 3 seconds
  across every scenario tested — missing plugin (M1), missing target
  dependency (M2), missing wrapper distribution (G2), BOM-managed
  versions (M5), dynamic/range versions (G5). Worst clean measurement
  across the whole pass: 13.1s (cold Gradle G1, first-run plugin
  resolution).
- **G2 confirmed the wrapper-bootstrap gap is real, not theoretical.**
  With the wrapper distribution absent and no network block, Gradle
  attempts a live DNS lookup to `services.gradle.org` before `--offline`
  is ever parsed (`UnknownHostException` once blocked). This is outside
  `--offline`'s reach entirely — only an external timeout protects it.
- Maven's offline failures are consistently fast (1–2.5s) once offline is
  decided; M1 (plugin uncached) vs M2 (target dependency uncached) are
  clearly distinguishable failure modes, as expected.
- Warm-cache cases for both tools stay well under 3s for the project
  sizes actually measured cleanly.

### Not yet confirmed — open going into Round 2

- **Silent-network condition, not yet tested.** Both network-blocked
  scenarios used `unshare -n` or an unreachable proxy, both of which fail
  fast (empty route / connection refused) — the *best* case for a hung
  connection. Neither covers a firewall that DROPs packets or a
  configured-but-dead DNS server, which is the actual slow/silent failure
  mode a hard timeout exists to bound (TCP retransmission and DNS
  resolver timeouts run tens of seconds to minutes, not milliseconds).
  Round 2 needs an `iptables ... -j DROP` (or DNS-blackhole) variant of
  G2/M6/G7.
- **Module-count scaling not cleanly captured.** The synthetic 10- and
  30-module Gradle runs meant to show whether configuration-phase cost
  grows with project size were not cleanly isolated (see toolchain note
  below); only the 1-module G1 number is clean. This is the largest
  remaining risk to any timeout value, since real Java projects are
  likely larger than the 1-module case.
- **G3, G4, G5, G6 were contaminated by an unrelated toolchain-detection
  quirk** on the packaged OpenJDK, honestly flagged in the raw results.
  Core offline-resolution behavior was still visible in G3–G5, but G6
  (does Gradle attempt a network repair on a corrupted cached artifact
  under `--offline`?) was not fully isolated and remains genuinely
  unanswered.
- **No non-Linux data.** Windows JVM cold-start is known to run slower
  under real-time AV scanning; macOS untested entirely.

### Provisional timeout

**30 seconds**, set with margin above the one clean "things are
legitimately starting up" number available (13.1s cold Gradle). Recorded
as provisional in constitution Article IX and decision 0001's Round 1
addendum — not to be treated as calibrated until Round 2 closes the gaps
above.

### Round 2 checklist

1. Repeat G2/M6/G7 with packets DROPped (not REJECTed) at the firewall,
   or with a dead-but-configured DNS server, to measure the actual silent
   failure mode.
2. Get clean 10-module and 30-module Gradle numbers, isolated from the
   toolchain quirk that contaminated this round.
3. Isolate G6 (corrupted-cache handling) cleanly.
4. At least one non-Linux data point, Windows preferred given the AV
   concern.
